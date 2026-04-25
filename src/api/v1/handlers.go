package v1

import (
	"fmt"
	"net/http"
	"strconv"

	"routex/api/utils"
	"routex/api/v1/types"
	"routex/app"
	"routex/i18n"
	"routex/models"
	"routex/subscription"
	"routex/utils/intID"

	"github.com/rs/zerolog/log"
)

// Handler provides a set of methods for handling API requests.
type Handler struct {
	app app.Main
}

// NewHandler creates a new handler for API v1.
func NewHandler(a app.Main) *Handler {
	return &Handler{app: a}
}

// NetfilterDHook
//
//	@Summary		netfilter.d event hook
//	@Description	Emits a netfilter.d event hook
//	@Tags			hooks
//	@Accept			json
//	@Produce		json
//	@Param			json	body		types.NetfilterDHookReq	true	"Request body"
//	@Success		200
//	@Failure		400		{object}	types.ErrorRes
//	@Failure		500		{object}	types.ErrorRes
//	@Router			/api/v1/system/hooks/netfilterd [post]
func (h *Handler) NetfilterDHook(w http.ResponseWriter, r *http.Request) {
	req, err := utils.ReadJson[types.NetfilterDHookReq](r)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	log.Debug().
		Str("type", req.Type).
		Str("table", req.Table).
		Msg("received netfilter.d event")
	err = h.app.ForceCommitIPTables()
	if err != nil {
		log.Error().Err(err).Msg("error fixing iptables after netfilter.d")
	}
}

// ListInterfaces
//
//	@Summary		Get list of interfaces
//	@Description	Returns the list of interfaces
//	@Tags			config
//	@Produce		json
//	@Success		200		{object}	types.InterfacesRes
//	@Failure		500		{object}	types.ErrorRes
//	@Router			/api/v1/system/interfaces [get]
func (h *Handler) ListInterfaces(w http.ResponseWriter, r *http.Request) {
	loc := i18n.FromContext(r.Context())
	interfaces, err := h.app.ListInterfaces()
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, fmt.Sprintf("%s: %v", loc.T("error.list_interfaces_failed"), err))
		return
	}
	res := make([]types.InterfaceRes, len(interfaces)+1)
	res[0] = types.InterfaceRes{ID: "blackhole"}
	for i, iface := range interfaces {
		res[i+1] = types.InterfaceRes{ID: iface.Name}
	}
	utils.WriteJson(w, http.StatusOK, types.InterfacesRes{Interfaces: res})
}

// SaveConfig
//
//	@Summary		Save configuration
//	@Description	Saves the current configuration to persistent storage
//	@Tags			config
//	@Produce		json
//	@Success		200
//	@Failure		500		{object}	types.ErrorRes
//	@Router			/api/v1/system/config/save [post]
func (h *Handler) SaveConfig(w http.ResponseWriter, r *http.Request) {
	loc := i18n.FromContext(r.Context())
	if err := h.app.SaveConfig(); err != nil {
		utils.WriteError(w, http.StatusInternalServerError, fmt.Sprintf("%s: %v", loc.T("error.save_config_failed"), err))
	}
}

// GetGroups
//
//	@Summary		Get list of groups
//	@Description	Returns the list of groups
//	@Tags			groups
//	@Produce		json
//	@Param			with_rules	query		bool	false	"Return groups with their rules"
//	@Success		200			{object}	types.GroupsRes
//	@Failure		500			{object}	types.ErrorRes
//	@Router			/api/v1/groups [get]
func (h *Handler) GetGroups(w http.ResponseWriter, r *http.Request) {
	withRules := r.URL.Query().Get("with_rules") == "true"
	appGroups := h.app.Groups()
	modelGroups := make([]*models.Group, len(appGroups))
	for i, g := range appGroups {
		modelGroups[i] = g.Model()
	}
	utils.WriteJson(w, http.StatusOK, RespFromGroups(modelGroups, withRules))
}

// PutGroups
//
//	@Summary		Update list of groups
//	@Description	Updates the list of groups
//	@Tags			groups
//	@Accept			json
//	@Produce		json
//	@Param			save	query		bool			false	"Save changes to the configuration file"
//	@Param			json	body		types.GroupsReq	true	"Request body"
//	@Success		200			{object}	types.GroupsRes
//	@Failure		400			{object}	types.ErrorRes
//	@Failure		500			{object}	types.ErrorRes
//	@Router			/api/v1/groups [put]
func (h *Handler) PutGroups(w http.ResponseWriter, r *http.Request) {
	req, err := utils.ReadJson[types.GroupsReq](r)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Groups == nil {
		loc := i18n.FromContext(r.Context())
		utils.WriteError(w, http.StatusBadRequest, loc.T("error.group_not_in_request"))
		return
	}
	for _, g := range h.app.Groups() {
		_ = g.Disable()
	}
	newGroups := make([]*models.Group, len(*req.Groups))
	for i, gReq := range *req.Groups {
		var existing *models.Group
		for _, g := range h.app.Groups() {
			if gReq.ID != nil && g.Model().ID == *gReq.ID {
				existing = g.Model()
				break
			}
		}
		newGroups[i], err = GroupFromReq(gReq, existing)
		if err != nil {
			utils.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	h.app.ClearGroups()
	for _, grp := range newGroups {
		if err := h.app.AddGroup(grp); err != nil {
			utils.WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	utils.WriteJson(w, http.StatusOK, RespFromGroups(newGroups, true))
	if r.URL.Query().Get("save") == "true" {
		if err := h.app.SaveConfig(); err != nil {
			log.Error().Err(err).Msg("failed to save config file")
		}
	}
}

// CreateGroup
//
//	@Summary		Create a group
//	@Description	Creates a group
//	@Tags			groups
//	@Accept			json
//	@Produce		json
//	@Param			save	query		bool			false	"Save changes to the configuration file"
//	@Param			json	body		types.GroupReq	true	"Request body"
//	@Success		200			{object}	types.GroupRes
//	@Failure		400			{object}	types.ErrorRes
//	@Failure		500			{object}	types.ErrorRes
//	@Router			/api/v1/groups [post]
func (h *Handler) CreateGroup(w http.ResponseWriter, r *http.Request) {
	req, err := utils.ReadJson[types.GroupReq](r)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	group, err := GroupFromReq(req, nil)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.app.AddGroup(group); err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	utils.WriteJson(w, http.StatusOK, RespFromGroup(group, true))
	if r.URL.Query().Get("save") == "true" {
		if err := h.app.SaveConfig(); err != nil {
			log.Error().Err(err).Msg("failed to save config file")
		}
	}
}

// GetGroup
//
//	@Summary		Get a group
//	@Description	Returns the requested group
//	@Tags			groups
//	@Produce		json
//	@Param			groupID		path		string	true	"Group ID"
//	@Param			with_rules	query		bool	false	"Return the group with its rules"
//	@Success		200			{object}	types.GroupRes
//	@Failure		404			{object}	types.ErrorRes
//	@Failure		500			{object}	types.ErrorRes
//	@Router			/api/v1/groups/{groupID} [get]
func (h *Handler) GetGroup(w http.ResponseWriter, r *http.Request) {
	groupIdx, _ := strconv.Atoi(r.Header.Get("groupIdx"))
	withRules := r.URL.Query().Get("with_rules") == "true"
	group := h.app.Groups()[groupIdx].Model()
	utils.WriteJson(w, http.StatusOK, RespFromGroup(group, withRules))
}

// PutGroup
//
//	@Summary		Update a group
//	@Description	Updates the requested group
//	@Tags			groups
//	@Accept			json
//	@Produce		json
//	@Param			groupID	path		string			true	"Group ID"
//	@Param			save	query		bool			false	"Save changes to the configuration file"
//	@Param			json	body		types.GroupReq	true	"Request body"
//	@Success		200			{object}	types.GroupRes
//	@Failure		400			{object}	types.ErrorRes
//	@Failure		404			{object}	types.ErrorRes
//	@Failure		500			{object}	types.ErrorRes
//	@Router			/api/v1/groups/{groupID} [put]
func (h *Handler) PutGroup(w http.ResponseWriter, r *http.Request) {
	req, err := utils.ReadJson[types.GroupReq](r)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	groupIdx, _ := strconv.Atoi(r.Header.Get("groupIdx"))
	groupWrapper := h.app.Groups()[groupIdx]

	loc := i18n.FromContext(r.Context())
	enabled := groupWrapper.Enabled()
	if enabled {
		if err := groupWrapper.Disable(); err != nil {
			utils.WriteError(w, http.StatusInternalServerError, fmt.Sprintf("%s: %v", loc.T("error.group_disable_failed"), err))
			return
		}
	}

	updatedGroup, err := GroupFromReq(req, groupWrapper.Model())
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	if enabled {
		if err := groupWrapper.Enable(); err != nil {
			utils.WriteError(w, http.StatusInternalServerError, fmt.Sprintf("%s: %v", loc.T("error.group_enable_failed"), err))
			return
		}
		if err := groupWrapper.Sync(); err != nil {
			utils.WriteError(w, http.StatusInternalServerError, fmt.Sprintf("%s: %v", loc.T("error.group_sync_failed"), err))
			return
		}
	}
	utils.WriteJson(w, http.StatusOK, RespFromGroup(updatedGroup, true))
	if r.URL.Query().Get("save") == "true" {
		if err := h.app.SaveConfig(); err != nil {
			log.Error().Err(err).Msg("failed to save config file")
		}
	}
}

// DeleteGroup
//
//	@Summary		Delete a group
//	@Description	Deletes the requested group
//	@Tags			groups
//	@Produce		json
//	@Param			groupID	path		string	true	"Group ID"
//	@Param			save	query		bool	false	"Save changes to the configuration file"
//	@Success		200
//	@Failure		404		{object}	types.ErrorRes
//	@Failure		500		{object}	types.ErrorRes
//	@Router			/api/v1/groups/{groupID} [delete]
func (h *Handler) DeleteGroup(w http.ResponseWriter, r *http.Request) {
	groupIdx, _ := strconv.Atoi(r.Header.Get("groupIdx"))
	groupWrapper := h.app.Groups()[groupIdx]
	model := groupWrapper.Model()
	if model.IsSubscription() {
		if subMgr := h.app.SubscriptionManager(); subMgr != nil {
			subMgr.RemoveCachedFiles(model.ID)
		}
	}
	if groupWrapper.Enabled() {
		if err := groupWrapper.Disable(); err != nil {
			loc := i18n.FromContext(r.Context())
			utils.WriteError(w, http.StatusInternalServerError, fmt.Sprintf("%s: %v", loc.T("error.group_disable_failed"), err))
			return
		}
	}
	h.app.RemoveGroupByIndex(groupIdx)
	if r.URL.Query().Get("save") == "true" {
		if err := h.app.SaveConfig(); err != nil {
			log.Error().Err(err).Msg("failed to save config file")
		}
	}
}

// GetRules
//
//	@Summary		Get list of rules
//	@Description	Returns the list of rules
//	@Tags			rules
//	@Produce		json
//	@Param			groupID	path		string	true	"Group ID"
//	@Success		200			{object}	types.RulesRes
//	@Failure		404			{object}	types.ErrorRes
//	@Failure		500			{object}	types.ErrorRes
//	@Router			/api/v1/groups/{groupID}/rules [get]
func (h *Handler) GetRules(w http.ResponseWriter, r *http.Request) {
	groupIdx, _ := strconv.Atoi(r.Header.Get("groupIdx"))
	rules := h.app.Groups()[groupIdx].Model().Rules
	utils.WriteJson(w, http.StatusOK, RespFromRules(rules))
}

// PutRules
//
//	@Summary		Update list of rules
//	@Description	Updates the list of rules
//	@Tags			rules
//	@Accept			json
//	@Produce		json
//	@Param			groupID	path		string			true	"Group ID"
//	@Param			save	query		bool			false	"Save changes to the configuration file"
//	@Param			json	body		types.RulesReq	true	"Request body"
//	@Success		200			{object}	types.RulesRes
//	@Failure		400			{object}	types.ErrorRes
//	@Failure		404			{object}	types.ErrorRes
//	@Failure		500			{object}	types.ErrorRes
//	@Router			/api/v1/groups/{groupID}/rules [put]
func (h *Handler) PutRules(w http.ResponseWriter, r *http.Request) {
	req, err := utils.ReadJson[types.RulesReq](r)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	loc := i18n.FromContext(r.Context())
	if req.Rules == nil {
		utils.WriteError(w, http.StatusBadRequest, loc.T("error.rule_not_in_request"))
		return
	}
	groupIdx, _ := strconv.Atoi(r.Header.Get("groupIdx"))
	groupWrapper := h.app.Groups()[groupIdx]
	enabled := groupWrapper.Enabled()

	newRules := make([]*models.Rule, len(*req.Rules))
	for i, rr := range *req.Rules {
		id := intID.RandomID()
		if rr.ID != nil {
			found := false
			for _, oldRule := range groupWrapper.Model().Rules {
				if oldRule.ID == *rr.ID {
					id = *rr.ID
					found = true
					break
				}
			}
			if !found {
				utils.WriteError(w, http.StatusNotFound, loc.T("error.rule_not_found"))
				return
			}
		}
		newRules[i] = &models.Rule{
			ID:     id,
			Name:   rr.Name,
			Type:   rr.Type,
			Rule:   rr.Rule,
			Enable: rr.Enable,
		}
	}
	groupWrapper.Model().Rules = newRules
	if enabled {
		if err := groupWrapper.Sync(); err != nil {
			utils.WriteError(w, http.StatusInternalServerError, fmt.Sprintf("%s: %v", loc.T("error.group_sync_failed"), err))
			return
		}
	}
	utils.WriteJson(w, http.StatusOK, RespFromRules(newRules))
	if r.URL.Query().Get("save") == "true" {
		if err := h.app.SaveConfig(); err != nil {
			log.Error().Err(err).Msg("failed to save config file")
		}
	}
}

// CreateRule
//
//	@Summary		Create a rule
//	@Description	Creates a rule
//	@Tags			rules
//	@Accept			json
//	@Produce		json
//	@Param			groupID	path		string			true	"Group ID"
//	@Param			save	query		bool			false	"Save changes to the configuration file"
//	@Param			json	body		types.RuleReq	true	"Request body"
//	@Success		200			{object}	types.RuleRes
//	@Failure		400			{object}	types.ErrorRes
//	@Failure		404			{object}	types.ErrorRes
//	@Failure		500			{object}	types.ErrorRes
//	@Router			/api/v1/groups/{groupID}/rules [post]
func (h *Handler) CreateRule(w http.ResponseWriter, r *http.Request) {
	req, err := utils.ReadJson[types.RuleReq](r)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	groupIdx, _ := strconv.Atoi(r.Header.Get("groupIdx"))
	groupWrapper := h.app.Groups()[groupIdx]
	enabled := groupWrapper.Enabled()

	rule, err := RuleFromReq(req, groupWrapper.Model().Rules)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	groupWrapper.Model().Rules = append(groupWrapper.Model().Rules, rule)
	if enabled {
		if err := groupWrapper.Sync(); err != nil {
			loc := i18n.FromContext(r.Context())
			utils.WriteError(w, http.StatusInternalServerError, fmt.Sprintf("%s: %v", loc.T("error.group_sync_failed"), err))
			return
		}
	}
	utils.WriteJson(w, http.StatusOK, RespFromRule(rule))
	if r.URL.Query().Get("save") == "true" {
		if err := h.app.SaveConfig(); err != nil {
			log.Error().Err(err).Msg("failed to save config file")
		}
	}
}

// GetRule
//
//	@Summary		Get a rule
//	@Description	Returns the requested rule
//	@Tags			rules
//	@Produce		json
//	@Param			groupID	path		string	true	"Group ID"
//	@Param			ruleID	path		string	true	"Rule ID"
//	@Success		200			{object}	types.RuleRes
//	@Failure		404			{object}	types.ErrorRes
//	@Failure		500			{object}	types.ErrorRes
//	@Router			/api/v1/groups/{groupID}/rules/{ruleID} [get]
func (h *Handler) GetRule(w http.ResponseWriter, r *http.Request) {
	groupIdx, _ := strconv.Atoi(r.Header.Get("groupIdx"))
	ruleIdx, _ := strconv.Atoi(r.Header.Get("ruleIdx"))
	rule := h.app.Groups()[groupIdx].Model().Rules[ruleIdx]
	utils.WriteJson(w, http.StatusOK, RespFromRule(rule))
}

// PutRule
//
//	@Summary		Update a rule
//	@Description	Updates the requested rule
//	@Tags			rules
//	@Accept			json
//	@Produce		json
//	@Param			groupID	path		string			true	"Group ID"
//	@Param			ruleID	path		string			true	"Rule ID"
//	@Param			save	query		bool			false	"Save changes to the configuration file"
//	@Param			json	body		types.RuleReq	true	"Request body"
//	@Success		200			{object}	types.RuleRes
//	@Failure		400			{object}	types.ErrorRes
//	@Failure		404			{object}	types.ErrorRes
//	@Failure		500			{object}	types.ErrorRes
//	@Router			/api/v1/groups/{groupID}/rules/{ruleID} [put]
func (h *Handler) PutRule(w http.ResponseWriter, r *http.Request) {
	req, err := utils.ReadJson[types.RuleReq](r)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	groupIdx, _ := strconv.Atoi(r.Header.Get("groupIdx"))
	groupWrapper := h.app.Groups()[groupIdx]
	enabled := groupWrapper.Enabled()

	ruleIdx, _ := strconv.Atoi(r.Header.Get("ruleIdx"))
	rule := groupWrapper.Model().Rules[ruleIdx]
	rule.Name = req.Name
	rule.Type = req.Type
	rule.Rule = req.Rule
	rule.Enable = req.Enable

	if enabled {
		if err := groupWrapper.Sync(); err != nil {
			loc := i18n.FromContext(r.Context())
			utils.WriteError(w, http.StatusInternalServerError, fmt.Sprintf("%s: %v", loc.T("error.group_sync_failed"), err))
			return
		}
	}
	utils.WriteJson(w, http.StatusOK, RespFromRule(rule))
	if r.URL.Query().Get("save") == "true" {
		if err := h.app.SaveConfig(); err != nil {
			log.Error().Err(err).Msg("failed to save config file")
		}
	}
}

// DeleteRule
//
//	@Summary		Delete a rule
//	@Description	Deletes the requested rule
//	@Tags			rules
//	@Produce		json
//	@Param			groupID	path		string	true	"Group ID"
//	@Param			ruleID	path		string	true	"Rule ID"
//	@Param			save	query		bool	false	"Save changes to the configuration file"
//	@Success		200
//	@Failure		404			{object}	types.ErrorRes
//	@Failure		500			{object}	types.ErrorRes
//	@Router			/api/v1/groups/{groupID}/rules/{ruleID} [delete]
func (h *Handler) DeleteRule(w http.ResponseWriter, r *http.Request) {
	groupIdx, _ := strconv.Atoi(r.Header.Get("groupIdx"))
	groupWrapper := h.app.Groups()[groupIdx]
	enabled := groupWrapper.Enabled()

	ruleIdx, _ := strconv.Atoi(r.Header.Get("ruleIdx"))
	groupWrapper.Model().Rules = append(groupWrapper.Model().Rules[:ruleIdx], groupWrapper.Model().Rules[ruleIdx+1:]...)
	if enabled {
		if err := groupWrapper.Sync(); err != nil {
			loc := i18n.FromContext(r.Context())
			utils.WriteError(w, http.StatusInternalServerError, fmt.Sprintf("%s: %v", loc.T("error.group_sync_failed"), err))
			return
		}
	}
	if r.URL.Query().Get("save") == "true" {
		if err := h.app.SaveConfig(); err != nil {
			log.Error().Err(err).Msg("failed to save config file")
		}
	}
}

// RefreshSubscription triggers a manual refresh for a subscription group
func (h *Handler) RefreshSubscription(w http.ResponseWriter, r *http.Request) {
	loc := i18n.FromContext(r.Context())
	groupIdx, _ := strconv.Atoi(r.Header.Get("groupIdx"))
	groupWrapper := h.app.Groups()[groupIdx]
	model := groupWrapper.Model()

	if !model.IsSubscription() {
		utils.WriteError(w, http.StatusBadRequest, loc.T("error.subscription_not_found"))
		return
	}

	subMgr := h.app.SubscriptionManager()
	if subMgr == nil {
		utils.WriteError(w, http.StatusInternalServerError, loc.T("error.api_internal"))
		return
	}

	if err := subMgr.RefreshGroup(groupWrapper); err != nil {
		utils.WriteError(w, http.StatusInternalServerError, fmt.Sprintf("%s: %v", loc.T("error.subscription_fetch_failed"), err))
		return
	}

	meta, _ := subMgr.GetMetadata(model.ID)
	status := subscriptionStatusFromMeta(meta)
	utils.WriteJson(w, http.StatusOK, status)
}

// GetSubscriptionStatus returns subscription status for a group
func (h *Handler) GetSubscriptionStatus(w http.ResponseWriter, r *http.Request) {
	loc := i18n.FromContext(r.Context())
	groupIdx, _ := strconv.Atoi(r.Header.Get("groupIdx"))
	model := h.app.Groups()[groupIdx].Model()

	if !model.IsSubscription() {
		utils.WriteError(w, http.StatusBadRequest, loc.T("error.subscription_not_found"))
		return
	}

	subMgr := h.app.SubscriptionManager()
	if subMgr == nil {
		utils.WriteError(w, http.StatusInternalServerError, loc.T("error.api_internal"))
		return
	}

	meta, err := subMgr.GetMetadata(model.ID)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	status := subscriptionStatusFromMeta(meta)
	utils.WriteJson(w, http.StatusOK, status)
}

// GetStats returns application statistics
func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
	snap := h.app.GetStats()
	utils.WriteJson(w, http.StatusOK, snap)
}

// GetTestDomain tests a domain against all rules
func (h *Handler) GetTestDomain(w http.ResponseWriter, r *http.Request) {
	domain := r.URL.Query().Get("domain")
	if domain == "" {
		loc := i18n.FromContext(r.Context())
		utils.WriteError(w, http.StatusBadRequest, loc.T("test.invalid_domain"))
		return
	}
	result := h.app.TestDomain(domain)
	utils.WriteJson(w, http.StatusOK, result)
}

func subscriptionStatusFromMeta(meta *subscription.Metadata) types.SubscriptionStatus {
	var lastUpdated string
	if !meta.LastUpdated.IsZero() {
		lastUpdated = meta.LastUpdated.Format("2006-01-02T15:04:05Z")
	}
	return types.SubscriptionStatus{
		LastUpdated: lastUpdated,
		RuleCount:   meta.RuleCount,
		LastError:   meta.LastError,
	}
}
