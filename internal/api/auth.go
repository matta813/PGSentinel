package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/matta813/pgsentinel/internal/auth"
)

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type changePasswordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

func (a *API) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/v1/") || r.URL.Path == "/api/v1/version" || r.URL.Path == "/api/v1/auth/login" {
			next.ServeHTTP(w, r)
			return
		}
		if a.auth == nil {
			failure(w, http.StatusUnauthorized, "Authentication required", nil)
			return
		}
		session, valid := a.auth.Session(r)
		if !valid {
			failure(w, http.StatusUnauthorized, "Authentication required", nil)
			return
		}
		if session.MustChangePassword && r.URL.Path != "/api/v1/auth/session" && r.URL.Path != "/api/v1/auth/password" && r.URL.Path != "/api/v1/auth/logout" {
			failure(w, http.StatusForbidden, "Password change required", nil)
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
	if !a.auth.AllowAttempt(r) {
		failure(w, http.StatusTooManyRequests, "Too many login attempts; try again later", nil)
		return
	}
	var request loginRequest
	if !decode(w, r, &request) {
		return
	}
	user, err := a.auth.Authenticate(r.Context(), request.Username, request.Password)
	if err != nil {
		failure(w, http.StatusUnauthorized, "Invalid credentials", nil)
		return
	}
	a.auth.ResetAttempts(r)
	if err := a.auth.Start(w, user); err != nil {
		a.log.Error("create session", "error", err)
		failure(w, http.StatusInternalServerError, "Unable to create session", nil)
		return
	}
	write(w, http.StatusOK, map[string]any{"authenticated": true, "username": user.Username, "mustChangePassword": user.MustChangePassword})
}

func (a *API) session(w http.ResponseWriter, r *http.Request) {
	session, valid := a.auth.Session(r)
	if !valid {
		failure(w, http.StatusUnauthorized, "Authentication required", nil)
		return
	}
	write(w, http.StatusOK, map[string]any{"authenticated": true, "username": session.Username, "mustChangePassword": session.MustChangePassword})
}

func (a *API) logout(w http.ResponseWriter, r *http.Request) {
	a.auth.End(w, r)
	write(w, http.StatusOK, map[string]bool{"authenticated": false})
}

func (a *API) changePassword(w http.ResponseWriter, r *http.Request) {
	var request changePasswordRequest
	if !decode(w, r, &request) {
		return
	}
	if err := a.auth.ChangePassword(r.Context(), r, request.CurrentPassword, request.NewPassword); err != nil {
		switch {
		case errors.Is(err, auth.ErrWeakPassword), errors.Is(err, auth.ErrPasswordReuse):
			failure(w, http.StatusUnprocessableEntity, err.Error(), nil)
		case errors.Is(err, auth.ErrInvalidCredentials):
			failure(w, http.StatusUnprocessableEntity, "Current password is incorrect", nil)
		default:
			a.log.Error("change administrator password", "error", err)
			failure(w, http.StatusInternalServerError, "Unable to change password", nil)
		}
		return
	}
	session, _ := a.auth.Session(r)
	write(w, http.StatusOK, map[string]any{"authenticated": true, "username": session.Username, "mustChangePassword": false})
}
