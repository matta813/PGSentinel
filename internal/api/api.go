package api

import (
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"gitlab.scruzzi.com/root/postgresqlui/internal/storage"
	"log/slog"
	"net/http"
	"strings"
)

type API struct {
	store *storage.Store
	log   *slog.Logger
	mux   *http.ServeMux
}

func New(store *storage.Store, log *slog.Logger) *API {
	a := &API{store: store, log: log, mux: http.NewServeMux()}
	a.routes()
	return a
}
func (a *API) Handler() http.Handler { return security(a.mux) }
func (a *API) routes() {
	a.mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		write(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	a.mux.HandleFunc("GET /ready", a.ready)
	a.mux.HandleFunc("GET /api/v1/servers", a.listServers)
	a.mux.HandleFunc("POST /api/v1/servers", a.createServer)
	a.mux.HandleFunc("GET /api/v1/servers/{id}", a.getServer)
	a.mux.HandleFunc("DELETE /api/v1/servers/{id}", a.deleteServer)
	a.mux.HandleFunc("POST /api/v1/servers/{id}/test", a.testServer)
	a.mux.HandleFunc("GET /api/v1/problems", a.listProblems)
	a.mux.HandleFunc("GET /api/v1/overview", a.overview)
	a.mux.HandleFunc("POST /api/v1/notifications/test", a.testNotification)
	a.mux.HandleFunc("GET /api/v1/servers/{id}/{resource}", a.serverResource)
}
func write(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func failure(w http.ResponseWriter, status int, msg string, err error) {
	detail := ""
	if err != nil {
		detail = err.Error()
	}
	write(w, status, map[string]string{"error": msg, "detail": detail})
}
func decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		failure(w, 400, "Invalid request", err)
		return false
	}
	return true
}
func validID(v string) bool { _, err := uuid.Parse(v); return err == nil }
func security(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self' 'unsafe-inline'; connect-src 'self'; img-src 'self' data:")
		if strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Set("Cache-Control", "no-store")
		}
		next.ServeHTTP(w, r)
	})
}

var errNotFound = errors.New("not found")
