package routex

import (
	"errors"
	"fmt"
	"net"
	"slices"
	"sync/atomic"

	"routex/app"
	"routex/constant"
	"routex/models"
	"routex/utils/dnsMITMProxy"
	"routex/utils/netfilterTools"
	"routex/utils/recordsCache"

	"github.com/rs/zerolog/log"
)

var (
	ErrAlreadyRunning           = errors.New("already running")
	ErrGroupIDConflict          = errors.New("group ID conflict")
	ErrRuleIDConflict           = errors.New("rule ID conflict")
	ErrConfigUnsupportedVersion = errors.New("unsupported configuration version")
)

// App is the main application core structure
type App struct {
	enabled atomic.Bool

	config models.AppConfig

	dnsMITM      *dnsMITMProxy.DNSMITMProxy
	nfHelper     *netfilterTools.Helper
	recordsCache *recordsCache.Records
	groups       []*Group
	dnsOverrider *netfilterTools.PortRemap
}

// New creates a new App instance
func New() *App {
	a := &App{
		config: constant.DefaultAppConfig,
	}
	if err := a.LoadConfig(); err != nil {
		log.Error().Err(err).Msg("failed to load config file")
	}
	return a
}

// Config returns the configuration
func (a *App) Config() models.AppConfig {
	return a.config
}

// Groups returns the list of groups
func (a *App) Groups() []app.Group {
	groups := make([]app.Group, len(a.groups))
	for i, g := range a.groups {
		groups[i] = g
	}
	return groups
}

// ClearGroups disables all groups and clears the list
func (a *App) ClearGroups() {
	for _, g := range a.groups {
		_ = g.Disable()
	}
	a.groups = a.groups[:0]
}

// AddGroup adds a new group
func (a *App) AddGroup(groupModel *models.Group) error {
	for _, group := range a.groups {
		if groupModel.ID == group.ID {
			return ErrGroupIDConflict
		}
	}
	// Check rule.ID uniqueness within the group.
	dup := make(map[[4]byte]struct{})
	for _, rule := range groupModel.Rules {
		if _, exists := dup[rule.ID]; exists {
			return ErrRuleIDConflict
		}
		dup[rule.ID] = struct{}{}
	}

	grp, err := NewGroup(groupModel, a)
	if err != nil {
		return fmt.Errorf("failed to create group: %w", err)
	}
	a.groups = append(a.groups, grp)

	log.Info().
		Str("id", grp.ID.String()).
		Str("name", grp.Name).
		Msg("added group")

	// If the application is already running, enable the group and perform synchronization
	if a.enabled.Load() {
		if err = grp.Enable(); err != nil {
			return fmt.Errorf("failed to enable group: %w", err)
		}
		if err = grp.Sync(); err != nil {
			return fmt.Errorf("failed to sync group: %w", err)
		}
	}
	return nil
}

// RemoveGroupByIndex removes a group by index
func (a *App) RemoveGroupByIndex(idx int) {
	a.groups = append(a.groups[:idx], a.groups[idx+1:]...)
}

// SwapGroups swaps two groups by index
func (a *App) SwapGroups(i, j int) {
	a.groups[i], a.groups[j] = a.groups[j], a.groups[i]
}

// ListInterfaces returns a list of network interfaces matching the specified criteria
func (a *App) ListInterfaces() ([]net.Interface, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to get network interfaces: %w", err)
	}

	if a.config.ShowAllInterfaces {
		return interfaces, nil
	}

	var filteredInterfaces []net.Interface
	for _, iface := range interfaces {
		if iface.Flags&net.FlagPointToPoint == 0 || slices.Contains(constant.IgnoredInterfaces, iface.Name) {
			continue
		}
		filteredInterfaces = append(filteredInterfaces, iface)
	}
	return filteredInterfaces, nil
}

// DnsOverrider returns the dnsOverrider
func (a *App) DnsOverrider() *netfilterTools.PortRemap {
	return a.dnsOverrider
}
