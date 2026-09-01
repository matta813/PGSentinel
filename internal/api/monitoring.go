package api

import (
	"database/sql"
	"github.com/matta813/pgsentinel/internal/analyzer"
	"github.com/matta813/pgsentinel/internal/models"
	"github.com/matta813/pgsentinel/internal/storage"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var historicalMetrics = map[string]bool{
	"connections.active": true, "connections.total": true, "connections.utilization": true,
	"connections.waiting": true, "server.uptime_seconds": true,
}

var monitoredResources = []string{"connections", "locks", "wait-events", "database-statistics", "queries", "tables", "indexes", "vacuum", "replication", "wal", "configuration"}

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
	filter := storage.FindingFilter{
		Status: r.URL.Query().Get("status"), ServerID: r.URL.Query().Get("serverId"),
		Database: strings.TrimSpace(r.URL.Query().Get("database")), Severity: strings.ToUpper(r.URL.Query().Get("severity")), Category: strings.TrimSpace(r.URL.Query().Get("category")), Search: strings.TrimSpace(r.URL.Query().Get("search")),
	}
	if filter.Status == "all" {
		filter.Status = ""
	}
	if filter.Status != "" && filter.Status != "active" && filter.Status != "acknowledged" && filter.Status != "resolved" {
		failure(w, 422, "Unsupported problem status", nil)
		return
	}
	if filter.Severity != "" {
		valid := map[string]bool{"CRITICAL": true, "HIGH": true, "MEDIUM": true, "LOW": true, "INFO": true}
		if !valid[filter.Severity] {
			failure(w, 422, "Unsupported severity", nil)
			return
		}
	}
	if len(filter.Search) > 200 {
		failure(w, 422, "Search text is too long", nil)
		return
	}
	if len(filter.Database) > 100 {
		failure(w, 422, "Database name is too long", nil)
		return
	}
	items, err := a.store.FilterFindings(r.Context(), filter)
	if err != nil {
		failure(w, 500, "Unable to load problems", err)
		return
	}
	if err := a.store.ApplyOperatorControls(r.Context(), items, time.Now().UTC()); err != nil {
		failure(w, 500, "Unable to apply operator controls", err)
		return
	}
	a.attachFindingQuality(r, items)
	write(w, 200, items)
}
func (a *API) updateProblemStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validFindingID(id) {
		failure(w, 400, "Invalid problem ID", nil)
		return
	}
	var request struct {
		Status string `json:"status"`
	}
	if !decode(w, r, &request) {
		return
	}
	if request.Status != "active" && request.Status != "acknowledged" {
		failure(w, 422, "Status must be active or acknowledged", nil)
		return
	}
	if err := a.store.SetFindingStatus(r.Context(), id, request.Status); err == sql.ErrNoRows {
		failure(w, 404, "Active problem not found", nil)
		return
	} else if err != nil {
		failure(w, 500, "Unable to update problem", err)
		return
	}
	if request.Status == "acknowledged" {
		a.audit(r, "", "finding.acknowledged", "finding", id, "A finding was acknowledged; its evidence and health impact remain available.")
	} else {
		a.audit(r, "", "finding.reopened", "finding", id, "An acknowledged finding was returned to active triage.")
	}
	w.WriteHeader(http.StatusNoContent)
}
func (a *API) overview(w http.ResponseWriter, r *http.Request) {
	servers, err := a.store.ListServers(r.Context())
	if err != nil {
		failure(w, 500, "Unable to load overview", err)
		return
	}
	findings, err := a.store.ListFindings(r.Context(), "open", "")
	if err != nil {
		failure(w, 500, "Unable to load overview", err)
		return
	}
	if err := a.store.ApplyOperatorControls(r.Context(), findings, time.Now().UTC()); err != nil {
		failure(w, 500, "Unable to apply operator controls", err)
		return
	}
	a.attachFindingQuality(r, findings)
	counts := map[models.Severity]int{}
	for _, f := range findings {
		counts[f.Severity]++
	}
	score := analyzer.HealthScore(findings)
	for _, server := range servers {
		switch server.Status {
		case "unreachable", "error":
			if score.Overall > 50 {
				score.Overall = 50
			}
		case "degraded", "unknown":
			if score.Overall > 75 {
				score.Overall = 75
			}
		}
	}
	freshness, _ := a.store.ListAllCollectionResources(r.Context(), time.Now())
	write(w, 200, map[string]any{"servers": servers, "problems": findings, "counts": counts, "score": score, "freshness": freshness})
}
func (a *API) serverFreshness(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validID(id) {
		failure(w, 400, "Invalid server ID", nil)
		return
	}
	if _, err := a.store.GetServer(r.Context(), id, false); err == sql.ErrNoRows {
		failure(w, 404, "Server not found", nil)
		return
	} else if err != nil {
		failure(w, 500, "Unable to load server", err)
		return
	}
	items, err := a.store.ListCollectionResources(r.Context(), id, time.Now())
	if err != nil {
		failure(w, 500, "Unable to load collection freshness", err)
		return
	}
	byResource := make(map[string]models.CollectionResourceStatus, len(items))
	for _, item := range items {
		byResource[item.Resource] = item
	}
	result := make([]models.CollectionResourceStatus, 0, len(monitoredResources))
	for _, resource := range monitoredResources {
		if item, ok := byResource[resource]; ok {
			result = append(result, item)
		} else {
			result = append(result, models.CollectionResourceStatus{ServerID: id, Resource: resource, State: "unavailable", ErrorSummary: "No collection attempt has completed yet."})
		}
	}
	write(w, 200, result)
}

func (a *API) attachFindingQuality(r *http.Request, findings []models.Finding) {
	cache := map[string][]models.CollectionResourceStatus{}
	for index := range findings {
		serverID := findings[index].ServerID
		items, ok := cache[serverID]
		if !ok {
			items, _ = a.store.ListCollectionResources(r.Context(), serverID, time.Now())
			cache[serverID] = items
		}
		quality := qualityForFinding(findings[index], items)
		if quality == nil {
			continue
		}
		findings[index].EvidenceQuality = quality
		if quality.State != "fresh" {
			switch findings[index].Confidence {
			case models.ConfidenceHigh:
				findings[index].Confidence = models.ConfidenceMedium
			case models.ConfidenceMedium:
				findings[index].Confidence = models.ConfidenceLow
			}
		}
	}
}

func qualityForFinding(finding models.Finding, items []models.CollectionResourceStatus) *models.CollectionResourceStatus {
	resource := map[string]string{
		"connection-utilization": "connections", "idle-in-transaction": "connections", "long-transaction": "connections",
		"blocking-queries": "locks", "deadlocks": "database-statistics", "rollback-ratio": "database-statistics", "cache-hit": "database-statistics",
		"wait-lock-pressure": "wait-events", "wait-class-concentration": "wait-events",
		"dead-tuples": "vacuum", "vacuum-behind": "vacuum", "large-seq-scans": "tables", "stale-analyze": "tables",
		"query-impact": "queries", "query-regression": "queries", "pgss-unavailable": "queries", "io-timing-disabled": "configuration",
		"unused-index": "indexes", "duplicate-index": "indexes",
	}[finding.RuleID]
	if resource == "" {
		resource = map[string]string{"Replication": "replication", "WAL": "wal"}[finding.Category]
	}
	for i := range items {
		if items[i].Resource == resource {
			item := items[i]
			return &item
		}
	}
	return nil
}

func (a *API) serverResource(w http.ResponseWriter, r *http.Request) {
	id, resource := r.PathValue("id"), r.PathValue("resource")
	allowed := map[string]string{"metrics": "core", "databases": "core", "connections": "core", "queries": "queries", "tables": "tables", "indexes": "indexes", "locks": "locks", "wait-events": "wait-events", "configuration": "configuration", "vacuum": "tables", "replication": "replication", "wal": "wal"}
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
	case "wait-events":
		value = &[]models.WaitEventSample{}
	case "configuration":
		value = &map[string]string{}
	case "replication":
		value = &models.ReplicationStats{}
	case "wal":
		value = &models.WALStats{}
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
