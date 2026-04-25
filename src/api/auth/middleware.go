package auth

import (
	"errors"
	"net/http"
	"strings"

	"routex/api/utils"
	"routex/app"
)

const authHeaderPrefix = "Bearer "

func Middleware(app app.Main) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := strings.TrimSpace(r.Header.Get("Authorization"))
			if !strings.HasPrefix(token, authHeaderPrefix) {
				utils.WriteError(w, http.StatusUnauthorized, "Yetkisiz erişim")
				return
			}
			token = strings.TrimPrefix(token, authHeaderPrefix)
			if token == "" {
				utils.WriteError(w, http.StatusUnauthorized, "Yetkisiz erişim")
				return
			}

			login, passwordHash, err := parseTokenSubject(token)
			if err != nil {
				utils.WriteError(w, http.StatusUnauthorized, "Yetkisiz erişim")
				return
			}
			if err := verifyToken(token, login, passwordHash); err != nil {
				utils.WriteError(w, http.StatusUnauthorized, "Yetkisiz erişim")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func parseTokenSubject(token string) (string, string, error) {
	claims, err := parseJWTWithoutVerification(token)
	if err != nil {
		return "", "", err
	}
	if claims.Sub == "" {
		return "", "", errors.New("boş konu alanı")
	}
	passwordHash, err := loadPasswordHash(claims.Sub)
	if err != nil {
		return "", "", err
	}
	return claims.Sub, passwordHash, nil
}
