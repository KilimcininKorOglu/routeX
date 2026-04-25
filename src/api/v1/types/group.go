package types

import (
	"routex/utils/intID"
)

type GroupsReq struct {
	Groups *[]GroupReq `json:"groups"`
}

type GroupsRes struct {
	Groups *[]GroupRes `json:"groups,omitempty"`
}

type GroupReq struct {
	ID        *intID.ID `json:"id" example:"0a1b2c3d" swaggertype:"string"`
	Name      string    `json:"name" example:"Routing"`
	Color     string    `json:"color" example:"#ffffff"`
	Interface string    `json:"interface" example:"nwg0"`
	Enable    *bool     `json:"enable" example:"true" TODO:"Make required after 1.0.0"`
	RulesReq

	SubscriptionURL      *string `json:"subscriptionUrl,omitempty"`
	SubscriptionInterval *uint   `json:"subscriptionInterval,omitempty"`
}

type GroupRes struct {
	ID        intID.ID `json:"id" example:"0a1b2c3d" swaggertype:"string"`
	Name      string   `json:"name" example:"Routing"`
	Color     string   `json:"color" example:"#ffffff"`
	Interface string   `json:"interface" example:"nwg0"`
	Enable    bool     `json:"enable" example:"true"`
	RulesRes

	SubscriptionURL      string              `json:"subscriptionUrl,omitempty"`
	SubscriptionInterval uint                `json:"subscriptionInterval,omitempty"`
	SubscriptionStatus   *SubscriptionStatus `json:"subscriptionStatus,omitempty"`
}

type SubscriptionStatus struct {
	LastUpdated string `json:"lastUpdated,omitempty"`
	RuleCount   int    `json:"ruleCount"`
	LastError   string `json:"lastError,omitempty"`
}
