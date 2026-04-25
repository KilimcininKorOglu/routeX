package auth

import (
	"net/http"

	"routex/api/utils"
	"routex/app"
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
		if !app.Config().HTTPWeb.Auth.Enabled {
			utils.WriteError(w, http.StatusNotFound, "Kimlik doğrulama devre dışı")
			return
		}
		
		req, err := utils.ReadJson[LoginRequest](r)
		if err != nil {
			utils.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		if req.Login == "" || req.Password == "" {
			utils.WriteError(w, http.StatusBadRequest, "kimlik bilgileri eksik")
			return
		}

		token, err := Authenticate(req.Login, req.Password)
		if err != nil {
			utils.WriteError(w, http.StatusForbidden, "Geçersiz kimlik bilgileri")
			return
		}
		utils.WriteJson(w, http.StatusOK, LoginResponse{Token: token})
	}
}
