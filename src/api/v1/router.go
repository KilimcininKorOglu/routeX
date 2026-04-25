package v1

import (
	"net/http"
	"strconv"

	"routex/api/auth"
	"routex/api/utils"
	"routex/app"
	"routex/i18n"
	"routex/utils/intID"

	"github.com/go-chi/chi/v5"
)

// NewRouter assembles the API v1 routes
func NewRouter(a app.Main) chi.Router {
	h := NewHandler(a)
	r := chi.NewRouter()
	r.Get("/auth", auth.StatusHandler(a))
	r.Post("/auth", auth.LoginHandler(a))
	r.Route("/groups", func(r chi.Router) {
		r.Get("/", h.GetGroups)
		r.Put("/", h.PutGroups)
		r.Post("/", h.CreateGroup)
		r.Route("/{groupID}", func(r chi.Router) {
			r.Use(func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					loc := i18n.FromContext(r.Context())
					groupID := chi.URLParam(r, "groupID")
					id, err := intID.ParseID(groupID)
					if err != nil {
						utils.WriteError(w, http.StatusBadRequest, loc.T("error.invalid_group_id"))
						return
					}
					for i, group := range h.app.Groups() {
						if group.Model().ID == id {
							r.Header.Set("groupIdx", strconv.Itoa(i))
							next.ServeHTTP(w, r)
							return
						}
					}
					utils.WriteError(w, http.StatusNotFound, loc.T("error.group_not_found"))
				})
			})
			r.Get("/", h.GetGroup)
			r.Put("/", h.PutGroup)
			r.Delete("/", h.DeleteGroup)
			r.Route("/subscription", func(r chi.Router) {
				r.Post("/refresh", h.RefreshSubscription)
				r.Get("/status", h.GetSubscriptionStatus)
			})
			r.Route("/rules", func(r chi.Router) {
				r.Get("/", h.GetRules)
				r.Put("/", h.PutRules)
				r.Post("/", h.CreateRule)
				r.Route("/{ruleID}", func(r chi.Router) {
					r.Use(func(next http.Handler) http.Handler {
						return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
							loc := i18n.FromContext(r.Context())
							ruleID := chi.URLParam(r, "ruleID")
							id, err := intID.ParseID(ruleID)
							if err != nil {
								utils.WriteError(w, http.StatusBadRequest, loc.T("error.invalid_rule_id"))
								return
							}
							groupIdx, _ := strconv.Atoi(r.Header.Get("groupIdx"))
							for idx, rule := range h.app.Groups()[groupIdx].Model().Rules {
								if rule.ID == id {
									r.Header.Set("ruleIdx", strconv.Itoa(idx))
									next.ServeHTTP(w, r)
									return
								}
							}
							utils.WriteError(w, http.StatusNotFound, loc.T("error.rule_not_found"))
						})
					})
					r.Get("/", h.GetRule)
					r.Put("/", h.PutRule)
					r.Delete("/", h.DeleteRule)
				})
			})
		})
	})
	r.Get("/stats", h.GetStats)
	r.Route("/system", func(r chi.Router) {
		r.Get("/interfaces", h.ListInterfaces)
		r.Route("/config", func(r chi.Router) {
			r.Post("/save", h.SaveConfig)
		})
		r.Route("/hooks", func(r chi.Router) {
			r.Post("/netfilterd", h.NetfilterDHook)
		})
	})
	return r
}
