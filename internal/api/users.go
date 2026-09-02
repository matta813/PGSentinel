package api

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"

	"github.com/matta813/pgsentinel/internal/auth"
)

func (a *API) registerUserRoutes() {
	a.mux.HandleFunc("GET /api/v1/users", a.listUsers)
	a.mux.HandleFunc("POST /api/v1/users", a.createUser)
	a.mux.HandleFunc("PUT /api/v1/users/{id}/role", a.updateUserRole)
}

type createUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Role     string `json:"role"`
}
type updateRoleRequest struct {
	Role string `json:"role"`
}

func (a *API) listUsers(w http.ResponseWriter, r *http.Request) {
	users, err := a.store.ListUsers(r.Context())
	if err != nil {
		failure(w, 500, "Unable to list users", nil)
		return
	}
	write(w, http.StatusOK, users)
}

func (a *API) createUser(w http.ResponseWriter, r *http.Request) {
	var request createUserRequest
	if !decode(w, r, &request) {
		return
	}
	request.Username = strings.TrimSpace(request.Username)
	if request.Username == "" || len(request.Username) > 100 || !auth.ValidRole(request.Role) || len(request.Password) < auth.MinimumPasswordLength {
		failure(w, http.StatusUnprocessableEntity, "Username, role, or initial password is invalid", nil)
		return
	}
	count, err := a.store.CountUsers(r.Context())
	if err != nil {
		failure(w, http.StatusInternalServerError, "Unable to create user", nil)
		return
	}
	if count >= 100 {
		failure(w, http.StatusUnprocessableEntity, "Local user limit reached", nil)
		return
	}
	user, err := a.auth.CreateUser(r.Context(), request.Username, request.Password, request.Role)
	if err != nil {
		failure(w, http.StatusConflict, "Unable to create user", nil)
		return
	}
	a.audit(r, "", "user.created", "user", user.ID, "A user account was created with role "+user.Role+".")
	write(w, http.StatusCreated, user)
}

func (a *API) updateUserRole(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validID(id) {
		failure(w, 400, "Invalid user ID", nil)
		return
	}
	var request updateRoleRequest
	if !decode(w, r, &request) {
		return
	}
	if !auth.ValidRole(request.Role) {
		failure(w, 422, "Invalid role", nil)
		return
	}
	session, _ := a.auth.Session(r)
	if session.UserID == id {
		failure(w, 422, "The active account role cannot be changed", nil)
		return
	}
	if err := a.store.UpdateUserRole(r.Context(), id, request.Role); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			failure(w, 404, "User not found", nil)
		} else {
			failure(w, 500, "Unable to update role", nil)
		}
		return
	}
	a.auth.InvalidateUser(id)
	a.audit(r, "", "user.role.changed", "user", id, "A user role was changed to "+request.Role+" and active sessions were invalidated.")
	write(w, http.StatusOK, map[string]string{"id": id, "role": request.Role})
}
