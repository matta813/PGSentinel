package api

import (
	"database/sql"
	"github.com/google/uuid"
	"github.com/matta813/pgsentinel/internal/models"
	pg "github.com/matta813/pgsentinel/internal/postgres"
	"net/http"
	"strings"
)

func (a *API) ready(w http.ResponseWriter, r *http.Request) {
	if err := a.store.DB.PingContext(r.Context()); err != nil {
		failure(w, 503, "Storage is not ready", err)
		return
	}
	write(w, 200, map[string]string{"status": "ready"})
}
func (a *API) listServers(w http.ResponseWriter, r *http.Request) {
	v, err := a.store.ListServers(r.Context())
	if err != nil {
		failure(w, 500, "Unable to load servers", err)
		return
	}
	write(w, 200, v)
}
func (a *API) createServer(w http.ResponseWriter, r *http.Request) {
	var v models.Server
	if !decode(w, r, &v) {
		return
	}
	v.Name = strings.TrimSpace(v.Name)
	v.Host = strings.TrimSpace(v.Host)
	v.User = strings.TrimSpace(v.User)
	if v.Name == "" || v.Host == "" || v.User == "" || v.Password == "" {
		failure(w, 422, "Name, host, user and password are required", nil)
		return
	}
	if v.Port == 0 {
		v.Port = 5432
	}
	if v.SSLMode == "" {
		v.SSLMode = "prefer"
	}
	switch v.SSLMode {
	case "disable", "allow", "prefer", "require", "verify-ca", "verify-full":
	default:
		failure(w, 422, "Unsupported SSL mode", nil)
		return
	}
	v.ID = uuid.NewString()
	if err := a.store.CreateServer(r.Context(), &v); err != nil {
		failure(w, 409, "Unable to save PostgreSQL server", err)
		return
	}
	v.Password = ""
	write(w, 201, v)
}
func (a *API) updateServer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validID(id) {
		failure(w, 400, "Invalid server ID", nil)
		return
	}
	var v models.Server
	if !decode(w, r, &v) {
		return
	}
	v.ID = id
	v.Name = strings.TrimSpace(v.Name)
	v.Host = strings.TrimSpace(v.Host)
	v.User = strings.TrimSpace(v.User)
	if v.Name == "" || v.Host == "" || v.User == "" {
		failure(w, 422, "Name, host and user are required", nil)
		return
	}
	if v.Port == 0 {
		v.Port = 5432
	}
	if v.SSLMode == "" {
		v.SSLMode = "prefer"
	}
	switch v.SSLMode {
	case "disable", "allow", "prefer", "require", "verify-ca", "verify-full":
	default:
		failure(w, 422, "Unsupported SSL mode", nil)
		return
	}
	if err := a.store.UpdateServer(r.Context(), &v); err == sql.ErrNoRows {
		failure(w, 404, "Server not found", nil)
		return
	} else if err != nil {
		failure(w, 409, "Unable to update PostgreSQL server", err)
		return
	}
	updated, err := a.store.GetServer(r.Context(), id, false)
	if err != nil {
		failure(w, 500, "Unable to load updated server", err)
		return
	}
	write(w, 200, updated)
}
func (a *API) getServer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validID(id) {
		failure(w, 400, "Invalid server ID", nil)
		return
	}
	v, err := a.store.GetServer(r.Context(), id, false)
	if err == sql.ErrNoRows {
		failure(w, 404, "Server not found", nil)
		return
	}
	if err != nil {
		failure(w, 500, "Unable to load server", err)
		return
	}
	write(w, 200, v)
}
func (a *API) deleteServer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validID(id) {
		failure(w, 400, "Invalid server ID", nil)
		return
	}
	if err := a.store.DeleteServer(r.Context(), id); err != nil {
		failure(w, 500, "Unable to delete server", err)
		return
	}
	w.WriteHeader(204)
}
func (a *API) testServer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	v, err := a.store.GetServer(r.Context(), id, true)
	if err != nil {
		failure(w, 404, "Server not found", nil)
		return
	}
	client, err := pg.Connect(r.Context(), v)
	if err != nil {
		_ = a.store.UpdateServerStatus(r.Context(), id, "unreachable", v.Version, err.Error(), false)
		failure(w, 502, "Unable to connect to PostgreSQL", err)
		return
	}
	defer client.Close()
	version, err := client.Version(r.Context())
	if err != nil {
		failure(w, 502, "Connected, but unable to read PostgreSQL metadata", err)
		return
	}
	_ = a.store.UpdateServerStatus(r.Context(), id, "healthy", version, "", true)
	write(w, 200, map[string]any{"ok": true, "version": version})
}
