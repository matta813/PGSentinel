package api

import (
	"net/http"
	"strings"
)

type loginRequest struct {
	Password string `json:"password"`
}

func (a *API) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/v1/") || r.URL.Path == "/api/v1/version" || r.URL.Path == "/api/v1/auth/login" {
			next.ServeHTTP(w, r)
			return
		}
		if a.auth == nil || !a.auth.Valid(r) {
			failure(w, http.StatusUnauthorized, "Authentication required", nil)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (a *API) login(w http.ResponseWriter, r *http.Request) {
	if a.auth == nil {
		failure(w, http.StatusServiceUnavailable, "Authentication is not configured", nil)
		return
	}
	if !a.auth.AllowAttempt(r.RemoteAddr) {
		failure(w, http.StatusTooManyRequests, "Too many login attempts; try again later", nil)
		return
	}
	var request loginRequest
	if !decode(w, r, &request) {
		return
	}
	if !a.auth.CheckPassword(request.Password) {
		failure(w, http.StatusUnauthorized, "Invalid credentials", nil)
		return
	}
	a.auth.ResetAttempts(r.RemoteAddr)
	if err := a.auth.Start(w); err != nil {
		a.log.Error("create session", "error", err)
		failure(w, http.StatusInternalServerError, "Unable to create session", nil)
		return
	}
	write(w, http.StatusOK, map[string]bool{"authenticated": true})
}

func (a *API) session(w http.ResponseWriter, _ *http.Request) {
	write(w, http.StatusOK, map[string]bool{"authenticated": true})
}

func (a *API) logout(w http.ResponseWriter, r *http.Request) {
	a.auth.End(w, r)
	write(w, http.StatusOK, map[string]bool{"authenticated": false})
}
