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
	"routex/stats"
	"routex/subscription"
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
	subManager   *subscription.Manager
	stats        *stats.Stats
}

// New creates a new App instance
func New() *App {
	a := &App{
		config: constant.DefaultAppConfig,
		stats:  stats.New(),
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

// Stats returns the stats collector
func (a *App) Stats() *stats.Stats {
	return a.stats
}

// GetStats returns a point-in-time snapshot of all statistics
func (a *App) GetStats() stats.Snapshot {
	snap := a.stats.TakeSnapshot()
	if a.recordsCache != nil {
		snap.CacheDomains = a.recordsCache.DomainCount()
		snap.CacheAddresses = a.recordsCache.AddressCount()
	}

	snap.Groups = make([]stats.GroupSnapshot, len(a.groups))
	for i, g := range a.groups {
		model := g.Model()
		gs := stats.GroupSnapshot{
			ID:             model.ID.String(),
			Name:           model.Name,
			Color:          model.Color,
			Interface:      model.Interface,
			Enabled:        g.Enabled(),
			RuleCount:      len(model.Rules),
			MatchedDomains: g.matchedDomains.Load(),
		}
		for _, rule := range model.Rules {
			if rule.IsEnabled() {
				gs.ActiveRules++
			}
		}
		if g.Enabled() {
			if v4, err := g.ListIPv4Subnets(); err == nil {
				gs.IPv4Entries = len(v4)
			}
			if v6, err := g.ListIPv6Subnets(); err == nil {
				gs.IPv6Entries = len(v6)
			}
		}
		snap.Groups[i] = gs
	}
	return snap
}

// TestDomain checks which rules match the given domain
func (a *App) TestDomain(domain string) models.TestResult {
	result := models.TestResult{
		Domain:  domain,
		Matches: []models.TestMatch{},
	}

	if a.recordsCache != nil {
		aliases := a.recordsCache.GetAliases(domain)
		if len(aliases) > 1 {
			result.Aliases = aliases[1:]
		}

		addresses := a.recordsCache.GetAddresses(domain)
		for _, addr := range addresses {
			result.CachedIPs = append(result.CachedIPs, addr.Address.String())
		}
	}

	names := []string{domain}
	if len(result.Aliases) > 0 {
		names = append(names, result.Aliases...)
	}

	for _, group := range a.groups {
		model := group.Model()
		for _, rule := range model.Rules {
			if !rule.IsEnabled() {
				continue
			}
			for _, name := range names {
				if rule.IsMatch(name) {
					result.Matches = append(result.Matches, models.TestMatch{
						GroupID:     model.ID.String(),
						GroupName:   model.Name,
						GroupColor:  model.Color,
						Interface:   model.Interface,
						RuleID:      rule.ID.String(),
						RuleName:    rule.Name,
						RuleType:    rule.Type,
						RulePattern: rule.Rule,
					})
					break
				}
			}
		}
	}

	return result
}

// SubscriptionManager returns the subscription manager
func (a *App) SubscriptionManager() *subscription.Manager {
	return a.subManager
}

// subscriptionGroupAccessor adapts App to subscription.GroupAccessor
type subscriptionGroupAccessor struct {
	app *App
}

func (s *subscriptionGroupAccessor) Groups() []subscription.SubscribableGroup {
	groups := make([]subscription.SubscribableGroup, len(s.app.groups))
	for i, g := range s.app.groups {
		groups[i] = g
	}
	return groups
}
