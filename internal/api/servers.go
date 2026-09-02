package api

import (
	"database/sql"
	"github.com/google/uuid"
	"github.com/matta813/pgsentinel/internal/models"
	pg "github.com/matta813/pgsentinel/internal/postgres"
	"net/http"
	"sort"
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
	if tag := strings.TrimSpace(r.URL.Query().Get("tag")); tag != "" {
		filtered := make([]models.Server, 0, len(v))
		for _, server := range v {
			if containsTag(server.Tags, tag) {
				filtered = append(filtered, server)
			}
		}
		v = filtered
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
	v.Tags = normalizeTags(v.Tags)
	if v.Name == "" || v.Host == "" || v.User == "" || v.Password == "" {
		failure(w, 422, "Name, host, user and password are required", nil)
		return
	}
	if v.Port == 0 {
		v.Port = 5432
	}
	if v.Port < 1 || v.Port > 65535 {
		failure(w, 422, "Port must be between 1 and 65535", nil)
		return
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
	a.audit(r, "", "server.created", "server", v.ID, "A PostgreSQL target was added.")
	v.Password = ""
	write(w, 201, v)
}
func normalizeTags(tags []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(tags))
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		key := strings.ToLower(tag)
		if tag == "" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, tag)
	}
	sort.Slice(out, func(i, j int) bool { return strings.ToLower(out[i]) < strings.ToLower(out[j]) })
	return out
}

func containsTag(tags []string, want string) bool {
	for _, tag := range tags {
		if strings.EqualFold(tag, want) {
			return true
		}
	}
	return false
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
	v.Tags = normalizeTags(v.Tags)
	if v.Name == "" || v.Host == "" || v.User == "" {
		failure(w, 422, "Name, host and user are required", nil)
		return
	}
	if v.Port == 0 {
		v.Port = 5432
	}
	if v.Port < 1 || v.Port > 65535 {
		failure(w, 422, "Port must be between 1 and 65535", nil)
		return
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
	credentialsRotated := v.Password != ""
	if err := a.store.UpdateServer(r.Context(), &v); err == sql.ErrNoRows {
		failure(w, 404, "Server not found", nil)
		return
	} else if err != nil {
		failure(w, 409, "Unable to update PostgreSQL server", err)
		return
	}
	a.audit(r, "", "server.updated", "server", id, "A PostgreSQL target was edited.")
	if credentialsRotated {
		a.audit(r, "", "server.credentials_rotated", "server", id, "PostgreSQL target credentials were rotated.")
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
	if err := a.store.DeleteServer(r.Context(), id); err == sql.ErrNoRows {
		failure(w, 404, "Server not found", nil)
		return
	} else if err != nil {
		failure(w, 500, "Unable to delete server", err)
		return
	}
	a.audit(r, "", "server.deleted", "server", id, "A PostgreSQL target was removed.")
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
