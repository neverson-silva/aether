package api

import (
	"net/http"
)

const authCookieName = "aether_token"

func (s *Server) setAuthCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     authCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   s.core.Cfg.CookieSecure,
		MaxAge:   86400,
	})
}

func (s *Server) clearAuthCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     authCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   s.core.Cfg.CookieSecure,
		MaxAge:   -1,
	})
}

func (s *Server) authCookieValue(r *http.Request) string {
	c, err := r.Cookie(authCookieName)
	if err != nil {
		return ""
	}
	return c.Value
}
