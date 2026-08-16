package api

import (
	"database/sql"
	"github.com/matta813/pgsentinel/internal/analyzer"
	"github.com/matta813/pgsentinel/internal/models"
	"net/http"
	"strconv"
	"time"
)

var historicalMetrics = map[string]bool{
	"connections.active": true, "connections.total": true, "connections.utilization": true,
	"connections.waiting": true, "server.uptime_seconds": true,
}

func (a *API) metricHistory(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validID(id) {
		failure(w, 400, "Invalid server ID", nil)
		return
	}
	name := r.URL.Query().Get("name")
	if !historicalMetrics[name] {
		failure(w, 422, "Unsupported metric name", nil)
		return
	}
	limit := 200
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 1000 {
			failure(w, 422, "Limit must be between 1 and 1000", nil)
			return
		}
		limit = parsed
	}
	var from time.Time
	if raw := r.URL.Query().Get("from"); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			failure(w, 422, "From must be an RFC3339 timestamp", nil)
			return
		}
		from = parsed
	}
	metrics, err := a.store.ListMetrics(r.Context(), id, name, from, limit)
	if err != nil {
		failure(w, 500, "Unable to load metric history", err)
		return
	}
	write(w, 200, metrics)
}

func (a *API) listProblems(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	serverID := r.URL.Query().Get("serverId")
	items, err := a.store.ListFindings(r.Context(), status, serverID)
	if err != nil {
		failure(w, 500, "Unable to load problems", err)
		return
	}
	write(w, 200, items)
}
func (a *API) overview(w http.ResponseWriter, r *http.Request) {
	servers, err := a.store.ListServers(r.Context())
	if err != nil {
		failure(w, 500, "Unable to load overview", err)
		return
	}
	findings, err := a.store.ListFindings(r.Context(), "active", "")
	if err != nil {
		failure(w, 500, "Unable to load overview", err)
		return
	}
	counts := map[models.Severity]int{}
	for _, f := range findings {
		counts[f.Severity]++
	}
	write(w, 200, map[string]any{"servers": servers, "problems": findings, "counts": counts, "score": analyzer.HealthScore(findings)})
}
func (a *API) serverResource(w http.ResponseWriter, r *http.Request) {
	id, resource := r.PathValue("id"), r.PathValue("resource")
	allowed := map[string]string{"metrics": "core", "databases": "core", "connections": "core", "queries": "queries", "tables": "tables", "indexes": "indexes", "locks": "locks", "configuration": "configuration", "vacuum": "tables"}
	kind, ok := allowed[resource]
	if !ok {
		failure(w, 404, "Resource not found", nil)
		return
	}
	var value any
	switch kind {
	case "core":
		value = &models.Snapshot{}
	case "queries":
		value = &[]models.QueryStat{}
	case "tables":
		value = &[]models.TableStat{}
	case "indexes":
		value = &[]models.IndexStat{}
	case "locks":
		value = &[]models.LockInfo{}
	case "configuration":
		value = &map[string]string{}
	}
	if err := a.store.LatestSnapshot(r.Context(), id, kind, value); err == sql.ErrNoRows {
		write(w, 200, value)
		return
	} else if err != nil {
		failure(w, 500, "Unable to load monitoring data", err)
		return
	}
	write(w, 200, value)
}
