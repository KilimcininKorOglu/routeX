package routex

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"runtime/debug"
	"strconv"
	"time"

	"routex/api"
	"routex/utils/dnsMITMProxy"
	"routex/utils/iptables"
	"routex/utils/netfilterTools"
	"routex/utils/recordsCache"

	"github.com/rs/zerolog"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netlink/nl"
)

// Start launches the application (core)
func (a *App) Start(ctx context.Context) (err error) {
	if !a.enabled.CompareAndSwap(false, true) {
		return ErrAlreadyRunning
	}
	defer a.enabled.Store(false)

	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "panic: %v\n%s\n", r, debug.Stack())
			err = errors.New(fmt.Sprintf("panik: %v", r))
		}
	}()

	a.setupLogging()

	a.dnsMITM = dnsMITMProxy.NewDNSMITMProxy(
		net.JoinHostPort(a.config.DNSProxy.Upstream.Address, strconv.Itoa(int(a.config.DNSProxy.Upstream.Port))),
		a.config.DNSProxy.MaxIdleConns,
		a.config.DNSProxy.MaxConcurrent,
		a.config.DNSProxy.Timeout,
	)
	a.dnsMITM.RequestHook = a.dnsRequestHook
	a.dnsMITM.ResponseHook = a.dnsResponseHook
	defer func() {
		if a.dnsMITM != nil {
			_ = a.dnsMITM.Close()
		}
	}()

	a.recordsCache = recordsCache.New()
	a.recordsCache.StartCleanup(ctx, 30*time.Second)

	nfh, err := netfilterTools.New(a.config.Netfilter.IPTables.ChainPrefix, a.config.Netfilter.IPSet.TablePrefix, a.config.Netfilter.DisableIPv4, a.config.Netfilter.DisableIPv6, a.config.Netfilter.StartMarkTableIndex)
	if err != nil {
		return fmt.Errorf("netfilter yardımcısı başlatılamadı: %w", err)
	}
	a.nfHelper = nfh

	for _, ipt := range []*iptables.IPTables{a.nfHelper.IPTables4, a.nfHelper.IPTables6} {
		if ipt == nil {
			continue
		}
		ipt.RegisterChainPatch("filter", "FORWARD")
		ipt.RegisterChainPatch("mangle", "PREROUTING")
		ipt.RegisterChainPatch("nat", "PREROUTING")
		ipt.RegisterChainPatch("nat", "POSTROUTING")
	}

	if err := a.nfHelper.CleanIPTables(); err != nil {
		return fmt.Errorf("iptables temizlenemedi: %w", err)
	}

	newCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	errChan := make(chan error)

	httpServer, err := api.SetupHTTP(a, errChan)
	if err != nil {
		return fmt.Errorf("HTTP sunucu kurulumu başarısız: %w", err)
	}
	defer httpServer.Close()

	unixServer, err := api.SetupUnixSocket(a, errChan)
	if err != nil {
		return fmt.Errorf("UNIX soket kurulumu başarısız: %w", err)
	}
	defer unixServer.Close()

	a.startDNSListeners(newCtx, errChan)

	var interfaceAddrs []netlink.Addr
	for _, linkName := range a.config.Link {
		link, err := netlink.LinkByName(linkName)
		if err != nil {
			return fmt.Errorf("%s bağlantısı bulunamadı: %w", linkName, err)
		}
		linkAddrList, err := netlink.AddrList(link, nl.FAMILY_ALL)
		if err != nil {
			return fmt.Errorf("%s arayüzünün adresleri listelenemedi: %w", linkName, err)
		}
		interfaceAddrs = append(interfaceAddrs, linkAddrList...)
	}

	if !a.config.DNSProxy.DisableRemap53 {
		a.dnsOverrider = a.nfHelper.PortRemap("DNSOR", 53, a.config.DNSProxy.Host.Port, interfaceAddrs)
		if err := a.dnsOverrider.Enable(); err != nil {
			return fmt.Errorf("DNS geçersiz kılınamadı: %v", err)
		}
		defer func() {
			_ = a.dnsOverrider.Disable()
		}()
	}

	for _, group := range a.groups {
		if err := group.Enable(); err != nil {
			return fmt.Errorf("grup etkinleştirilemedi: %w", err)
		}
		if err := group.Sync(); err != nil {
			return fmt.Errorf("grup senkronize edilemedi: %w", err)
		}
	}
	defer func() {
		for _, group := range a.groups {
			_ = group.Disable()
		}
	}()

	linkUpdateChannel, linkUpdateDone, err := subscribeLinkUpdates()
	if err != nil {
		return err
	}
	defer close(linkUpdateDone)

	for {
		select {
		case event := <-linkUpdateChannel:
			a.handleLink(event)
		case err := <-errChan:
			return err
		case <-ctx.Done():
			return nil
		}
	}
}

func (a *App) ForceCommitIPTables() error {
	if a.nfHelper == nil {
		return nil
	}

	if a.nfHelper.IPTables4 != nil {
		err := a.nfHelper.IPTables4.Commit()
		if err != nil {
			return fmt.Errorf("iptables kuralları uygulanamadı: %w", err)
		}
	}

	if a.nfHelper.IPTables6 != nil {
		err := a.nfHelper.IPTables6.Commit()
		if err != nil {
			return fmt.Errorf("iptables kuralları uygulanamadı: %w", err)
		}
	}

	return nil
}

func (a *App) setupLogging() {
	switch a.config.LogLevel {
	case "trace":
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "info":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	case "fatal":
		zerolog.SetGlobalLevel(zerolog.FatalLevel)
	case "panic":
		zerolog.SetGlobalLevel(zerolog.PanicLevel)
	case "nolevel":
		zerolog.SetGlobalLevel(zerolog.NoLevel)
	case "disabled":
		zerolog.SetGlobalLevel(zerolog.Disabled)
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
}

func (a *App) getInterfaceAddresses() ([]netlink.Addr, error) {
	var addrList []netlink.Addr
	for _, linkName := range a.config.Link {
		link, err := netlink.LinkByName(linkName)
		if err != nil {
			return nil, fmt.Errorf("%s bağlantısı bulunamadı: %w", linkName, err)
		}
		linkAddrList, err := netlink.AddrList(link, nl.FAMILY_ALL)
		if err != nil {
			return nil, fmt.Errorf("%s arayüzünün adresleri listelenemedi: %w", linkName, err)
		}
		addrList = append(addrList, linkAddrList...)
	}
	return addrList, nil
}
