package models

import (
	"routex/utils/intID"
)

type Group struct {
	ID        intID.ID
	Name      string
	Color     string
	Interface string
	Enable    bool
	Rules     []*Rule

	SubscriptionURL      string
	SubscriptionInterval uint // minutes; 0 means default (1440 = 24h)
}

func (g *Group) IsSubscription() bool {
	return g.SubscriptionURL != ""
}
