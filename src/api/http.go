package api

import (
	"fmt"
	"net"
	"net/http"
	"strings"

	"routex/api/auth"
	v1 "routex/api/v1"
	"routex/app"
	"routex/web"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog/log"
)

func SetupHTTP(a app.Main, errChan chan error) (*http.Server, error) {
	if !a.Config().HTTPWeb.Enabled {
		log.Info().Msg("HTTP WebUI disabled by configuration")
		return nil, nil
	}

	addr := fmt.Sprintf("%s:%d", a.Config().HTTPWeb.Host.Address, a.Config().HTTPWeb.Host.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("HTTP dinleme hatası %s: %v", addr, err)
	}

	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestSize(1 << 20)) // 1 MB
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			next.ServeHTTP(w, r)
		})
	})

	loginLimiter := auth.LoginRateLimitMiddleware()

	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.HasPrefix(r.URL.Path, "/api/") {
				next.ServeHTTP(w, r)
				return
			}
			if r.URL.Path == "/api/v1/auth" && r.Method == http.MethodPost {
				loginLimiter(next).ServeHTTP(w, r)
				return
			}
			if !a.Config().HTTPWeb.Auth.Enabled || r.URL.Path == "/api/v1/auth" {
				next.ServeHTTP(w, r)
				return
			}
			auth.Middleware(a)(next).ServeHTTP(w, r)
		})
	})

	r.Mount("/api/v1", v1.NewRouter(a))

	r.Handle("/static/*", web.StaticFS())

	h := web.NewHandler(a)

	r.Get("/login", h.LoginPage)
	r.With(loginLimiter).Post("/login", h.LoginSubmit)
	r.Get("/logout", h.Logout)

	r.Group(func(r chi.Router) {
		r.Use(h.SessionAuthMiddleware)
		r.Get("/", h.Dashboard)
		r.Get("/settings", h.Settings)

		r.Get("/htmx/groups", h.HtmxGetGroups)
		r.Post("/htmx/groups", h.HtmxCreateGroup)
		r.Put("/htmx/groups/{groupID}", h.HtmxUpdateGroup)
		r.Delete("/htmx/groups/{groupID}", h.HtmxDeleteGroup)
		r.Post("/htmx/groups/{groupID}/rules", h.HtmxAddRuleForm)
		r.Post("/htmx/groups/{groupID}/rules/create", h.HtmxCreateRule)
		r.Put("/htmx/groups/{groupID}/rules/{ruleID}", h.HtmxUpdateRule)
		r.Delete("/htmx/groups/{groupID}/rules/{ruleID}", h.HtmxDeleteRule)
		r.Post("/htmx/config/save", h.HtmxSaveConfig)
		r.Get("/htmx/groups/search", h.HtmxSearchGroups)
		r.Get("/htmx/config/import-form", h.HtmxImportForm)
		r.Post("/htmx/config/import", h.HtmxImportConfig)
		r.Get("/config/export", h.ExportConfig)
		r.Post("/htmx/groups/{groupID}/move/{direction}", h.HtmxMoveGroup)
		r.Post("/htmx/groups/{groupID}/rules/{ruleID}/move/{direction}", h.HtmxMoveRule)
	})

	srv := &http.Server{Handler: r}

	log.Info().Msgf("Starting HTTP server on %s", addr)
	go func() {
		if e := srv.Serve(listener); e != nil && e != http.ErrServerClosed {
			errChan <- fmt.Errorf("HTTP sunucu hatası: %v", e)
		}
		listener.Close()
	}()
	return srv, nil
}
