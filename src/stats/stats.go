package stats

import (
	"fmt"
	"sync/atomic"
	"time"
)

type Stats struct {
	StartedAt time.Time

	QueriesA     atomic.Uint64
	QueriesAAAA  atomic.Uint64
	QueriesPTR   atomic.Uint64
	QueriesOther atomic.Uint64

	Responses        atomic.Uint64
	FakePTRResponses atomic.Uint64
	DroppedAAAA      atomic.Uint64
	MatchedRoutes    atomic.Uint64
}

func New() *Stats {
	return &Stats{
		StartedAt: time.Now(),
	}
}

func (s *Stats) TotalQueries() uint64 {
	return s.QueriesA.Load() + s.QueriesAAAA.Load() +
		s.QueriesPTR.Load() + s.QueriesOther.Load()
}

type Snapshot struct {
	Uptime           string          `json:"uptime"`
	UptimeSeconds    int64           `json:"uptimeSeconds"`
	TotalQueries     uint64          `json:"totalQueries"`
	QueriesA         uint64          `json:"queriesA"`
	QueriesAAAA      uint64          `json:"queriesAAAA"`
	QueriesPTR       uint64          `json:"queriesPTR"`
	QueriesOther     uint64          `json:"queriesOther"`
	TotalResponses   uint64          `json:"totalResponses"`
	FakePTRResponses uint64          `json:"fakePtrResponses"`
	DroppedAAAA      uint64          `json:"droppedAAAA"`
	MatchedRoutes    uint64          `json:"matchedRoutes"`
	CacheDomains     int             `json:"cacheDomains"`
	CacheAddresses   int             `json:"cacheAddresses"`
	Groups           []GroupSnapshot `json:"groups"`
}

type GroupSnapshot struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Color          string `json:"color"`
	Interface      string `json:"interface"`
	Enabled        bool   `json:"enabled"`
	RuleCount      int    `json:"ruleCount"`
	ActiveRules    int    `json:"activeRules"`
	MatchedDomains uint64 `json:"matchedDomains"`
	IPv4Entries    int    `json:"ipv4Entries"`
	IPv6Entries    int    `json:"ipv6Entries"`
}

func (s *Stats) TakeSnapshot() Snapshot {
	uptime := time.Since(s.StartedAt)
	return Snapshot{
		Uptime:           formatDuration(uptime),
		UptimeSeconds:    int64(uptime.Seconds()),
		TotalQueries:     s.TotalQueries(),
		QueriesA:         s.QueriesA.Load(),
		QueriesAAAA:      s.QueriesAAAA.Load(),
		QueriesPTR:       s.QueriesPTR.Load(),
		QueriesOther:     s.QueriesOther.Load(),
		TotalResponses:   s.Responses.Load(),
		FakePTRResponses: s.FakePTRResponses.Load(),
		DroppedAAAA:      s.DroppedAAAA.Load(),
		MatchedRoutes:    s.MatchedRoutes.Load(),
	}
}

func formatDuration(d time.Duration) string {
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}
