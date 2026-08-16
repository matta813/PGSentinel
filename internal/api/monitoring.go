package api

import (
	"database/sql"
	"github.com/matta813/pgsentinel/internal/analyzer"
	"github.com/matta813/pgsentinel/internal/models"
	"net/http"
)

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
func (a *API) updateProblemStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validID(id) {
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
