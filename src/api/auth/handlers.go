package auth

import (
	"net/http"

	"routex/api/utils"
	"routex/app"
	"routex/i18n"
)

type LoginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

type StatusResponse struct {
	Enabled bool `json:"enabled"`
}

func StatusHandler(app app.Main) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		utils.WriteJson(w, http.StatusOK, StatusResponse{Enabled: app.Config().HTTPWeb.Auth.Enabled})
	}
}

func LoginHandler(app app.Main) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		loc := i18n.FromContext(r.Context())
		if !app.Config().HTTPWeb.Auth.Enabled {
			utils.WriteError(w, http.StatusNotFound, loc.T("error.auth_disabled"))
			return
		}
		
		req, err := utils.ReadJson[LoginRequest](r)
		if err != nil {
			utils.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		if req.Login == "" || req.Password == "" {
			utils.WriteError(w, http.StatusBadRequest, loc.T("error.credentials_missing"))
			return
		}

		token, err := Authenticate(req.Login, req.Password)
		if err != nil {
			utils.WriteError(w, http.StatusForbidden, loc.T("error.login_invalid"))
			return
		}
		utils.WriteJson(w, http.StatusOK, LoginResponse{Token: token})
	}
}
