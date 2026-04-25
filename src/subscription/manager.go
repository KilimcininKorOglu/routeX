package subscription

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"time"

	"routex/models"
	"routex/utils/intID"

	"github.com/rs/zerolog/log"
)

const defaultIntervalMinutes = 1440 // 24 hours

type GroupAccessor interface {
	Groups() []SubscribableGroup
}

type SubscribableGroup interface {
	Model() *models.Group
	Enabled() bool
	Disable() error
	Enable() error
	Sync() error
}

type Manager struct {
	groups   GroupAccessor
	client   *http.Client
	stateDir string
}

func NewManager(groups GroupAccessor, stateDir string) *Manager {
	return &Manager{
		groups:   groups,
		client:   newHTTPClient(),
		stateDir: stateDir,
	}
}

func (m *Manager) Start(ctx context.Context) {
	go func() {
		// Defer first refresh to let networking stabilize
		select {
		case <-time.After(5 * time.Second):
		case <-ctx.Done():
			return
		}

		m.RefreshAll()

		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				m.checkAndRefresh()
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (m *Manager) RefreshAll() {
	for _, group := range m.groups.Groups() {
		model := group.Model()
		if !model.IsSubscription() {
			continue
		}
		if err := m.RefreshGroup(group); err != nil {
			log.Warn().
				Err(err).
				Str("group", model.Name).
				Msg("subscription refresh failed")
		}
	}
}

func (m *Manager) checkAndRefresh() {
	for _, group := range m.groups.Groups() {
		model := group.Model()
		if !model.IsSubscription() {
			continue
		}

		interval := time.Duration(model.SubscriptionInterval) * time.Minute
		if interval <= 0 {
			interval = time.Duration(defaultIntervalMinutes) * time.Minute
		}

		meta, err := loadMetadata(m.stateDir, model.ID)
		if err != nil || meta.LastUpdated.IsZero() || time.Since(meta.LastUpdated) >= interval {
			if err := m.RefreshGroup(group); err != nil {
				log.Warn().
					Err(err).
					Str("group", model.Name).
					Msg("scheduled subscription refresh failed")
			}
		}
	}
}

func (m *Manager) RefreshGroup(group SubscribableGroup) error {
	model := group.Model()
	if !model.IsSubscription() {
		return nil
	}

	meta, err := loadMetadata(m.stateDir, model.ID)
	if err != nil {
		meta = &Metadata{}
	}

	req, err := newRequest(model.SubscriptionURL, meta)
	if err != nil {
		m.recordError(model.ID, meta, fmt.Sprintf("invalid request: %v", err))
		return fmt.Errorf("invalid subscription URL: %w", err)
	}

	resp, err := m.client.Do(req)
	if err != nil {
		m.recordError(model.ID, meta, fmt.Sprintf("fetch failed: %v", err))
		return fmt.Errorf("failed to fetch subscription: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusNotModified:
		meta.LastUpdated = time.Now()
		meta.LastError = ""
		_ = saveMetadata(m.stateDir, model.ID, meta)
		log.Debug().
			Str("group", model.Name).
			Msg("subscription not modified (304)")
		return nil

	case http.StatusOK:
		body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBody))
		if err != nil {
			m.recordError(model.ID, meta, fmt.Sprintf("read body failed: %v", err))
			return fmt.Errorf("failed to read subscription body: %w", err)
		}

		domains, err := ParseList(bytes.NewReader(body))
		if err != nil {
			m.recordError(model.ID, meta, fmt.Sprintf("parse failed: %v", err))
			return fmt.Errorf("failed to parse subscription list: %w", err)
		}

		if err := saveCachedList(m.stateDir, model.ID, body); err != nil {
			log.Warn().Err(err).Str("group", model.Name).Msg("failed to cache subscription list")
		}

		rules := domainsToRules(domains)
		if err := m.applyRules(group, rules); err != nil {
			m.recordError(model.ID, meta, fmt.Sprintf("apply failed: %v", err))
			return fmt.Errorf("failed to apply subscription rules: %w", err)
		}

		meta.LastUpdated = time.Now()
		meta.RuleCount = len(rules)
		meta.LastError = ""
		if etag := resp.Header.Get("ETag"); etag != "" {
			meta.ETag = etag
		}
		if lm := resp.Header.Get("Last-Modified"); lm != "" {
			meta.LastModified = lm
		}
		_ = saveMetadata(m.stateDir, model.ID, meta)

		log.Info().
			Str("group", model.Name).
			Int("rules", len(rules)).
			Msg("subscription updated")
		return nil

	default:
		msg := fmt.Sprintf("HTTP %d", resp.StatusCode)
		m.recordError(model.ID, meta, msg)
		return fmt.Errorf("subscription fetch returned %s", msg)
	}
}

func (m *Manager) LoadCachedRules(groupID intID.ID) ([]*models.Rule, error) {
	body, err := loadCachedList(m.stateDir, groupID)
	if err != nil {
		return nil, err
	}
	if body == nil {
		return nil, nil
	}
	domains, err := ParseList(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	return domainsToRules(domains), nil
}

func (m *Manager) GetMetadata(groupID intID.ID) (*Metadata, error) {
	return loadMetadata(m.stateDir, groupID)
}

func (m *Manager) RemoveCachedFiles(groupID intID.ID) {
	removeCachedFiles(m.stateDir, groupID)
}

func (m *Manager) applyRules(group SubscribableGroup, rules []*models.Rule) error {
	wasEnabled := group.Enabled()
	if wasEnabled {
		if err := group.Disable(); err != nil {
			return fmt.Errorf("failed to disable group for rule swap: %w", err)
		}
	}

	group.Model().Rules = rules

	if wasEnabled {
		if err := group.Enable(); err != nil {
			return fmt.Errorf("failed to re-enable group after rule swap: %w", err)
		}
		if err := group.Sync(); err != nil {
			return fmt.Errorf("failed to sync group after rule swap: %w", err)
		}
	}
	return nil
}

func (m *Manager) recordError(groupID intID.ID, meta *Metadata, msg string) {
	meta.LastError = msg
	meta.LastErrorAt = time.Now()
	_ = saveMetadata(m.stateDir, groupID, meta)
}

func domainsToRules(domains []string) []*models.Rule {
	rules := make([]*models.Rule, len(domains))
	for i, domain := range domains {
		rules[i] = &models.Rule{
			ID:     deterministicID(domain),
			Name:   domain,
			Type:   models.RuleTypeNamespace,
			Rule:   domain,
			Enable: true,
		}
	}
	return rules
}

func deterministicID(domain string) intID.ID {
	h := sha256.Sum256([]byte(domain))
	var id intID.ID
	copy(id[:], h[:4])
	return id
}
