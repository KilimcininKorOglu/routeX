package auth

import (
	"errors"
	"net/http"
	"strings"

	"routex/api/utils"
	"routex/app"
	"routex/i18n"
)

const authHeaderPrefix = "Bearer "

func Middleware(app app.Main) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			loc := i18n.FromContext(r.Context())
			token := strings.TrimSpace(r.Header.Get("Authorization"))
			if !strings.HasPrefix(token, authHeaderPrefix) {
				utils.WriteError(w, http.StatusUnauthorized, loc.T("error.unauthorized"))
				return
			}
			token = strings.TrimPrefix(token, authHeaderPrefix)
			if token == "" {
				utils.WriteError(w, http.StatusUnauthorized, loc.T("error.unauthorized"))
				return
			}

			login, passwordHash, err := parseTokenSubject(token)
			if err != nil {
				utils.WriteError(w, http.StatusUnauthorized, loc.T("error.unauthorized"))
				return
			}
			if err := verifyToken(token, login, passwordHash); err != nil {
				utils.WriteError(w, http.StatusUnauthorized, loc.T("error.unauthorized"))
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
		return "", "", errors.New("empty subject field")
	}
	passwordHash, err := loadPasswordHash(claims.Sub)
	if err != nil {
		return "", "", err
	}
	return claims.Sub, passwordHash, nil
}
