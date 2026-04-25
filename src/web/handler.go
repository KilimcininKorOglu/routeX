package web

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"routex/api/auth"
	"routex/app"
	"routex/config"
	"routex/models"
	"routex/utils/intID"
	"routex/web/templates/components"
	"routex/web/templates/pages"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

const sessionCookieName = "mt_session"

type Handler struct {
	app app.Main
}

func NewHandler(a app.Main) *Handler {
	return &Handler{app: a}
}

func (h *Handler) LoginPage(w http.ResponseWriter, r *http.Request) {
	if !h.app.Config().HTTPWeb.Auth.Enabled {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	pages.Login("").Render(r.Context(), w)
}

func (h *Handler) LoginSubmit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		pages.Login("Geçersiz form verisi").Render(r.Context(), w)
		return
	}

	login := r.FormValue("login")
	password := r.FormValue("password")

	if login == "" || password == "" {
		pages.Login("Lütfen kullanıcı adı ve şifre girin").Render(r.Context(), w)
		return
	}

	token, err := auth.Authenticate(login, password)
	if err != nil {
		pages.Login("Geçersiz kimlik bilgileri").Render(r.Context(), w)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(24 * time.Hour / time.Second),
	})

	w.Header().Set("HX-Redirect", "/")
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	pages.Dashboard().Render(r.Context(), w)
}

func (h *Handler) Settings(w http.ResponseWriter, r *http.Request) {
	pages.Settings(h.app.Config()).Render(r.Context(), w)
}

func (h *Handler) getInterfaces() []string {
	ifaces, err := h.app.ListInterfaces()
	if err != nil {
		return []string{"blackhole"}
	}
	result := make([]string, 0, len(ifaces)+1)
	result = append(result, "blackhole")
	for _, iface := range ifaces {
		result = append(result, iface.Name)
	}
	return result
}

func (h *Handler) getGroupModels() []*models.Group {
	appGroups := h.app.Groups()
	result := make([]*models.Group, len(appGroups))
	for i, g := range appGroups {
		result[i] = g.Model()
	}
	return result
}

func (h *Handler) findGroupIndex(groupID string) (int, error) {
	id, err := intID.ParseID(groupID)
	if err != nil {
		return -1, fmt.Errorf("geçersiz grup kimliği")
	}
	for i, g := range h.app.Groups() {
		if g.Model().ID == id {
			return i, nil
		}
	}
	return -1, fmt.Errorf("grup bulunamadı")
}

func (h *Handler) findRuleIndex(groupIdx int, ruleID string) (int, error) {
	id, err := intID.ParseID(ruleID)
	if err != nil {
		return -1, fmt.Errorf("geçersiz kural kimliği")
	}
	for i, rule := range h.app.Groups()[groupIdx].Model().Rules {
		if rule.ID == id {
			return i, nil
		}
	}
	return -1, fmt.Errorf("kural bulunamadı")
}

func (h *Handler) HtmxGetGroups(w http.ResponseWriter, r *http.Request) {
	groups := h.getGroupModels()
	ifaces := h.getInterfaces()
	pages.GroupsList(groups, ifaces).Render(r.Context(), w)
}

func (h *Handler) HtmxCreateGroup(w http.ResponseWriter, r *http.Request) {
	group := &models.Group{
		ID:        intID.RandomID(),
		Name:      "New Group",
		Color:     "#ffffff",
		Interface: "blackhole",
		Enable:    true,
	}
	if err := h.app.AddGroup(group); err != nil {
		http.Error(w, "Sunucu hatası", http.StatusInternalServerError)
		return
	}
	ifaces := h.getInterfaces()
	components.GroupPanel(group, ifaces).Render(r.Context(), w)
}

func (h *Handler) HtmxUpdateGroup(w http.ResponseWriter, r *http.Request) {
	groupID := chi.URLParam(r, "groupID")
	idx, err := h.findGroupIndex(groupID)
	if err != nil {
		http.Error(w, "Bulunamadı", http.StatusNotFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Geçersiz istek", http.StatusBadRequest)
		return
	}

	ifaceValue := r.FormValue("interface")
	if err := models.ValidateInterfaceName(ifaceValue); err != nil {
		http.Error(w, "Geçersiz arayüz adı", http.StatusBadRequest)
		return
	}

	groupWrapper := h.app.Groups()[idx]
	group := groupWrapper.Model()
	group.Name = r.FormValue("name")
	group.Interface = ifaceValue
	group.Color = r.FormValue("color")
	group.Enable = r.FormValue("enable") == "on"

	if groupWrapper.Enabled() {
		if err := groupWrapper.Sync(); err != nil {
			log.Error().Err(err).Msg("failed to sync group after update")
		}
	}

	ifaces := h.getInterfaces()
	components.GroupPanel(group, ifaces).Render(r.Context(), w)
}

func (h *Handler) HtmxDeleteGroup(w http.ResponseWriter, r *http.Request) {
	groupID := chi.URLParam(r, "groupID")
	idx, err := h.findGroupIndex(groupID)
	if err != nil {
		http.Error(w, "Bulunamadı", http.StatusNotFound)
		return
	}

	groupWrapper := h.app.Groups()[idx]
	if groupWrapper.Enabled() {
		_ = groupWrapper.Disable()
	}
	h.app.RemoveGroupByIndex(idx)
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) HtmxAddRuleForm(w http.ResponseWriter, r *http.Request) {
	groupID := chi.URLParam(r, "groupID")
	components.NewRuleRow(groupID).Render(r.Context(), w)
}

func (h *Handler) HtmxCreateRule(w http.ResponseWriter, r *http.Request) {
	groupID := chi.URLParam(r, "groupID")
	idx, err := h.findGroupIndex(groupID)
	if err != nil {
		http.Error(w, "Bulunamadı", http.StatusNotFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Geçersiz istek", http.StatusBadRequest)
		return
	}

	groupWrapper := h.app.Groups()[idx]
	rule := &models.Rule{
		ID:     intID.RandomID(),
		Name:   r.FormValue("name"),
		Type:   r.FormValue("type"),
		Rule:   r.FormValue("rule"),
		Enable: true,
	}
	groupWrapper.Model().Rules = append(groupWrapper.Model().Rules, rule)

	if groupWrapper.Enabled() {
		if err := groupWrapper.Sync(); err != nil {
			log.Error().Err(err).Msg("failed to sync group after rule create")
		}
	}

	components.RuleRow(groupID, rule).Render(r.Context(), w)
}

func (h *Handler) HtmxUpdateRule(w http.ResponseWriter, r *http.Request) {
	groupID := chi.URLParam(r, "groupID")
	ruleID := chi.URLParam(r, "ruleID")

	groupIdx, err := h.findGroupIndex(groupID)
	if err != nil {
		http.Error(w, "Bulunamadı", http.StatusNotFound)
		return
	}

	ruleIdx, err := h.findRuleIndex(groupIdx, ruleID)
	if err != nil {
		http.Error(w, "Bulunamadı", http.StatusNotFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Geçersiz istek", http.StatusBadRequest)
		return
	}

	groupWrapper := h.app.Groups()[groupIdx]
	rule := groupWrapper.Model().Rules[ruleIdx]
	rule.Name = r.FormValue("name")
	rule.Type = r.FormValue("type")
	rule.Rule = r.FormValue("rule")
	rule.Enable = r.FormValue("enable") == "on"

	if groupWrapper.Enabled() {
		if err := groupWrapper.Sync(); err != nil {
			log.Error().Err(err).Msg("failed to sync group after rule update")
		}
	}

	components.RuleRow(groupID, rule).Render(r.Context(), w)
}

func (h *Handler) HtmxDeleteRule(w http.ResponseWriter, r *http.Request) {
	groupID := chi.URLParam(r, "groupID")
	ruleID := chi.URLParam(r, "ruleID")

	groupIdx, err := h.findGroupIndex(groupID)
	if err != nil {
		http.Error(w, "Bulunamadı", http.StatusNotFound)
		return
	}

	ruleIdx, err := h.findRuleIndex(groupIdx, ruleID)
	if err != nil {
		http.Error(w, "Bulunamadı", http.StatusNotFound)
		return
	}

	groupWrapper := h.app.Groups()[groupIdx]
	groupWrapper.Model().Rules = append(
		groupWrapper.Model().Rules[:ruleIdx],
		groupWrapper.Model().Rules[ruleIdx+1:]...,
	)

	if groupWrapper.Enabled() {
		if err := groupWrapper.Sync(); err != nil {
			log.Error().Err(err).Msg("failed to sync group after rule delete")
		}
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) HtmxSaveConfig(w http.ResponseWriter, r *http.Request) {
	if err := h.app.SaveConfig(); err != nil {
		log.Error().Err(err).Msg("config save failed")
		http.Error(w, "Kaydetme başarısız", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(`<span style="color:var(--green);font-size:0.875rem" hx-swap-oob="true" id="save-status">Kaydedildi</span>`))
}

func (h *Handler) HtmxMoveGroup(w http.ResponseWriter, r *http.Request) {
	groupID := chi.URLParam(r, "groupID")
	direction := chi.URLParam(r, "direction")

	idx, err := h.findGroupIndex(groupID)
	if err != nil {
		http.Error(w, "Bulunamadı", http.StatusNotFound)
		return
	}

	groupCount := len(h.app.Groups())
	switch direction {
	case "up":
		if idx > 0 {
			h.app.SwapGroups(idx, idx-1)
		}
	case "down":
		if idx < groupCount-1 {
			h.app.SwapGroups(idx, idx+1)
		}
	}

	groups := h.getGroupModels()
	ifaces := h.getInterfaces()
	pages.GroupsList(groups, ifaces).Render(r.Context(), w)
}

func (h *Handler) HtmxMoveRule(w http.ResponseWriter, r *http.Request) {
	groupID := chi.URLParam(r, "groupID")
	ruleID := chi.URLParam(r, "ruleID")
	direction := chi.URLParam(r, "direction")

	groupIdx, err := h.findGroupIndex(groupID)
	if err != nil {
		http.Error(w, "Bulunamadı", http.StatusNotFound)
		return
	}

	ruleIdx, err := h.findRuleIndex(groupIdx, ruleID)
	if err != nil {
		http.Error(w, "Bulunamadı", http.StatusNotFound)
		return
	}

	rules := h.app.Groups()[groupIdx].Model().Rules
	switch direction {
	case "up":
		if ruleIdx > 0 {
			rules[ruleIdx], rules[ruleIdx-1] = rules[ruleIdx-1], rules[ruleIdx]
		}
	case "down":
		if ruleIdx < len(rules)-1 {
			rules[ruleIdx], rules[ruleIdx+1] = rules[ruleIdx+1], rules[ruleIdx]
		}
	}

	ifaces := h.getInterfaces()
	group := h.app.Groups()[groupIdx].Model()
	components.GroupPanel(group, ifaces).Render(r.Context(), w)
}

func (h *Handler) HtmxSearchGroups(w http.ResponseWriter, r *http.Request) {
	query := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	allGroups := h.getGroupModels()
	ifaces := h.getInterfaces()

	if query == "" {
		pages.GroupsList(allGroups, ifaces).Render(r.Context(), w)
		return
	}

	var filtered []*models.Group
	for _, group := range allGroups {
		if strings.Contains(strings.ToLower(group.Name), query) ||
			strings.Contains(strings.ToLower(group.Interface), query) {
			filtered = append(filtered, group)
			continue
		}
		var matchedRules []*models.Rule
		for _, rule := range group.Rules {
			if strings.Contains(strings.ToLower(rule.Rule), query) ||
				strings.Contains(strings.ToLower(rule.Name), query) ||
				strings.Contains(strings.ToLower(rule.Type), query) {
				matchedRules = append(matchedRules, rule)
			}
		}
		if len(matchedRules) > 0 {
			filteredGroup := &models.Group{
				ID:        group.ID,
				Name:      group.Name,
				Color:     group.Color,
				Interface: group.Interface,
				Enable:    group.Enable,
				Rules:     matchedRules,
			}
			filtered = append(filtered, filteredGroup)
		}
	}

	pages.GroupsList(filtered, ifaces).Render(r.Context(), w)
}

func (h *Handler) ExportConfig(w http.ResponseWriter, r *http.Request) {
	cfg := h.app.ExportConfig()
	data, err := yaml.Marshal(cfg)
	if err != nil {
		log.Error().Err(err).Msg("config export failed")
		http.Error(w, "Dışa aktarma başarısız", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/x-yaml")
	w.Header().Set("Content-Disposition", "attachment; filename=routex-config.yaml")
	w.Write(data)
}

func (h *Handler) HtmxImportForm(w http.ResponseWriter, r *http.Request) {
	components.ImportConfigForm().Render(r.Context(), w)
}

func (h *Handler) HtmxImportConfig(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "Dosya çok büyük veya geçersiz form", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("config")
	if err != nil {
		http.Error(w, "Dosya yüklenmedi", http.StatusBadRequest)
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Dosya okunamadı", http.StatusInternalServerError)
		return
	}

	var cfg config.Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		if err := json.Unmarshal(data, &cfg); err != nil {
			http.Error(w, "Geçersiz yapılandırma dosya formatı (YAML veya JSON bekleniyor)", http.StatusBadRequest)
			return
		}
	}

	if err := h.app.ImportConfig(cfg); err != nil {
		log.Error().Err(err).Msg("config import failed")
		http.Error(w, "İçe aktarma başarısız", http.StatusInternalServerError)
		return
	}

	groups := h.getGroupModels()
	ifaces := h.getInterfaces()
	pages.GroupsList(groups, ifaces).Render(r.Context(), w)
}

func (h *Handler) SessionAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !h.app.Config().HTTPWeb.Auth.Enabled {
			next.ServeHTTP(w, r)
			return
		}

		cookie, err := r.Cookie(sessionCookieName)
		if err != nil || cookie.Value == "" {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		if err := auth.VerifyTokenString(cookie.Value); err != nil {
			http.SetCookie(w, &http.Cookie{
				Name:     sessionCookieName,
				Value:    "",
				Path:     "/",
				HttpOnly: true,
				SameSite: http.SameSiteStrictMode,
				MaxAge:   -1,
			})
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		next.ServeHTTP(w, r)
	})
}
