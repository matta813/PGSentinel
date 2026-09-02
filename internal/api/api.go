package api

import (
	"encoding/hex"
	"encoding/json"
	stdErrors "errors"
	"github.com/google/uuid"
	"github.com/matta813/pgsentinel/internal/auth"
	"github.com/matta813/pgsentinel/internal/buildinfo"
	"github.com/matta813/pgsentinel/internal/notifications"
	"github.com/matta813/pgsentinel/internal/storage"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path"
	"strings"
)

type API struct {
	store              *storage.Store
	log                *slog.Logger
	mux                *http.ServeMux
	auth               *auth.Manager
	notificationPolicy notifications.TargetPolicy
}

type Options struct {
	Auth               *auth.Manager
	NotificationPolicy notifications.TargetPolicy
}

func New(store *storage.Store, log *slog.Logger, options ...Options) *API {
	var opts Options
	if len(options) > 0 {
		opts = options[0]
	}
	a := &API{store: store, log: log, mux: http.NewServeMux(), auth: opts.Auth, notificationPolicy: opts.NotificationPolicy}
	a.routes()
	return a
}
func (a *API) Handler() http.Handler { return security(a.authenticate(a.mux)) }
func (a *API) routes() {
	a.registerSystemRoutes()
	a.registerAuthRoutes()
	a.registerUserRoutes()
	a.registerServerRoutes()
	a.registerMonitoringRoutes()
	a.registerIncidentRoutes()
	a.registerNotificationRoutes()
	a.registerOperatorControlRoutes()
	a.registerRuleProfileRoutes()
	a.registerAuditRoutes()
	a.registerChangeEventRoutes()
	a.registerDiagnosticRoutes()
}

func (a *API) registerSystemRoutes() {
	a.mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		write(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	a.mux.HandleFunc("GET /ready", a.ready)
	a.mux.HandleFunc("GET /metrics", a.prometheusMetrics)
	a.mux.HandleFunc("GET /api/v1/version", func(w http.ResponseWriter, _ *http.Request) { write(w, http.StatusOK, buildinfo.Current()) })
}

func (a *API) ServeFrontend(directory string) {
	frontend := os.DirFS(directory)
	files := http.FileServerFS(frontend)
	index, indexErr := fs.ReadFile(frontend, "index.html")
	a.mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			failure(w, http.StatusNotFound, "API endpoint not found", nil)
			return
		}
		name := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
		if name != "" && fs.ValidPath(name) {
			if info, err := fs.Stat(frontend, name); err == nil && !info.IsDir() {
				files.ServeHTTP(w, r)
				return
			}
		}

		if indexErr == nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write(index)
			return
		}
		http.NotFound(w, r)
	})
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
	const maxJSONBody = 64 << 10
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBody)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		var tooLarge *http.MaxBytesError
		if stdErrors.As(err, &tooLarge) {
			failure(w, http.StatusRequestEntityTooLarge, "Request body too large", nil)
			return false
		}
		failure(w, 400, "Invalid request", err)
		return false
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		failure(w, 400, "Request must contain exactly one JSON value", err)
		return false
	}
	return true
}
func validID(v string) bool { _, err := uuid.Parse(v); return err == nil }
func validFindingID(v string) bool {
	if len(v) != 24 {
		return false
	}
	_, err := hex.DecodeString(v)
	return err == nil
}
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
