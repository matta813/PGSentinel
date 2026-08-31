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
	a.mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		write(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	a.mux.HandleFunc("GET /ready", a.ready)
	a.mux.HandleFunc("GET /metrics", a.prometheusMetrics)
	a.mux.HandleFunc("GET /api/v1/version", func(w http.ResponseWriter, _ *http.Request) { write(w, http.StatusOK, buildinfo.Current()) })
	a.mux.HandleFunc("POST /api/v1/auth/login", a.login)
	a.mux.HandleFunc("GET /api/v1/auth/session", a.session)
	a.mux.HandleFunc("POST /api/v1/auth/logout", a.logout)
	a.mux.HandleFunc("PUT /api/v1/auth/password", a.changePassword)
	a.mux.HandleFunc("GET /api/v1/users", a.listUsers)
	a.mux.HandleFunc("POST /api/v1/users", a.createUser)
	a.mux.HandleFunc("PUT /api/v1/users/{id}/role", a.updateUserRole)
	a.mux.HandleFunc("GET /api/v1/servers", a.listServers)
	a.mux.HandleFunc("POST /api/v1/servers", a.createServer)
	a.mux.HandleFunc("GET /api/v1/servers/{id}", a.getServer)
	a.mux.HandleFunc("PUT /api/v1/servers/{id}", a.updateServer)
	a.mux.HandleFunc("DELETE /api/v1/servers/{id}", a.deleteServer)
	a.mux.HandleFunc("POST /api/v1/servers/{id}/test", a.testServer)
	a.mux.HandleFunc("GET /api/v1/servers/{id}/metric-history", a.metricHistory)
	a.mux.HandleFunc("GET /api/v1/servers/{id}/freshness", a.serverFreshness)
	a.mux.HandleFunc("GET /api/v1/problems", a.listProblems)
	a.mux.HandleFunc("PUT /api/v1/problems/{id}/status", a.updateProblemStatus)
	a.mux.HandleFunc("GET /api/v1/incidents", a.listIncidents)
	a.mux.HandleFunc("GET /api/v1/incidents/{id}", a.getIncident)
	a.mux.HandleFunc("GET /api/v1/overview", a.overview)
	a.mux.HandleFunc("POST /api/v1/notifications/test", a.testNotification)
	a.mux.HandleFunc("GET /api/v1/notifications", a.listNotificationDestinations)
	a.mux.HandleFunc("POST /api/v1/notifications", a.createNotificationDestination)
	a.mux.HandleFunc("PUT /api/v1/notifications/{id}", a.updateNotificationDestination)
	a.mux.HandleFunc("DELETE /api/v1/notifications/{id}", a.deleteNotificationDestination)
	a.mux.HandleFunc("GET /api/v1/notification-routes", a.listNotificationRoutes)
	a.mux.HandleFunc("POST /api/v1/notification-routes", a.createNotificationRoute)
	a.mux.HandleFunc("PUT /api/v1/notification-routes/{id}", a.updateNotificationRoute)
	a.mux.HandleFunc("DELETE /api/v1/notification-routes/{id}", a.deleteNotificationRoute)
	a.mux.HandleFunc("GET /api/v1/notification-deliveries", a.listNotificationDeliveries)
	a.mux.HandleFunc("GET /api/v1/maintenance-windows", a.listMaintenanceWindows)
	a.mux.HandleFunc("POST /api/v1/maintenance-windows", a.createMaintenanceWindow)
	a.mux.HandleFunc("DELETE /api/v1/maintenance-windows/{id}", a.deleteMaintenanceWindow)
	a.mux.HandleFunc("GET /api/v1/suppressions", a.listSuppressions)
	a.mux.HandleFunc("POST /api/v1/suppressions", a.createSuppression)
	a.mux.HandleFunc("DELETE /api/v1/suppressions/{id}", a.deleteSuppression)
	a.mux.HandleFunc("GET /api/v1/threshold-overrides", a.listThresholdOverrides)
	a.mux.HandleFunc("POST /api/v1/threshold-overrides", a.createThresholdOverride)
	a.mux.HandleFunc("DELETE /api/v1/threshold-overrides/{id}", a.deleteThresholdOverride)
	a.mux.HandleFunc("GET /api/v1/audit-events", a.listAuditEvents)
	a.mux.HandleFunc("GET /api/v1/change-events", a.listChangeEvents)
	a.mux.HandleFunc("POST /api/v1/deployments", a.createDeploymentEvent)
	a.mux.HandleFunc("DELETE /api/v1/deployments/{id}", a.deleteDeploymentEvent)
	a.mux.HandleFunc("GET /api/v1/diagnostic-bundle", a.diagnosticBundle)
	a.mux.HandleFunc("GET /api/v1/servers/{id}/{resource}", a.serverResource)
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
