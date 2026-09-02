package api

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/matta813/pgsentinel/internal/models"
)

func (a *API) registerChangeEventRoutes() {
	a.mux.HandleFunc("GET /api/v1/change-events", a.listChangeEvents)
	a.mux.HandleFunc("POST /api/v1/deployments", a.createDeploymentEvent)
	a.mux.HandleFunc("DELETE /api/v1/deployments/{id}", a.deleteDeploymentEvent)
}

func (a *API) listChangeEvents(w http.ResponseWriter, r *http.Request) {
	serverID := r.URL.Query().Get("serverId")
	if !validID(serverID) {
		failure(w, http.StatusBadRequest, "Valid serverId is required", nil)
		return
	}
	if _, err := a.store.GetServer(r.Context(), serverID, false); err == sql.ErrNoRows {
		failure(w, http.StatusNotFound, "Server not found", nil)
		return
	} else if err != nil {
		failure(w, http.StatusInternalServerError, "Unable to load server", err)
		return
	}
	limit := 100
	if raw := r.URL.Query().Get("limit"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 || value > 200 {
			failure(w, 422, "Limit must be between 1 and 200", nil)
			return
		}
		limit = value
	}
	items, err := a.store.ListChangeEvents(r.Context(), serverID, time.Time{}, time.Time{}, limit)
	if err != nil {
		failure(w, 500, "Unable to load change history", err)
		return
	}
	write(w, 200, items)
}

func (a *API) createDeploymentEvent(w http.ResponseWriter, r *http.Request) {
	var request struct {
		ServerID   string    `json:"serverId"`
		Summary    string    `json:"summary"`
		OccurredAt time.Time `json:"occurredAt"`
	}
	if !decode(w, r, &request) {
		return
	}
	request.Summary = strings.TrimSpace(request.Summary)
	if !validID(request.ServerID) || request.Summary == "" || len(request.Summary) > 300 || request.OccurredAt.IsZero() || request.OccurredAt.After(time.Now().Add(5*time.Minute)) || request.OccurredAt.Before(time.Now().AddDate(-1, 0, 0)) {
		failure(w, 422, "Deployment requires a server, summary, and occurrence within the last year", nil)
		return
	}
	if _, err := a.store.GetServer(r.Context(), request.ServerID, false); err == sql.ErrNoRows {
		failure(w, 404, "Server not found", nil)
		return
	} else if err != nil {
		failure(w, 500, "Unable to record deployment", err)
		return
	}
	event := models.ChangeEvent{ServerID: request.ServerID, Kind: "deployment", Summary: request.Summary, OccurredAt: request.OccurredAt}
	if err := a.store.RecordChangeEvent(r.Context(), &event); err != nil {
		failure(w, 500, "Unable to record deployment", err)
		return
	}
	a.audit(r, "", "deployment.recorded", "change_event", event.ID, "A deployment marker was recorded for query correlation.")
	write(w, http.StatusCreated, event)
}

func (a *API) deleteDeploymentEvent(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validID(id) {
		failure(w, 400, "Invalid change event ID", nil)
		return
	}
	if err := a.store.DeleteChangeEvent(r.Context(), id); err == sql.ErrNoRows {
		failure(w, 404, "Deployment marker not found", nil)
		return
	} else if err != nil {
		failure(w, 500, "Unable to delete deployment marker", err)
		return
	}
	a.audit(r, "", "deployment.deleted", "change_event", id, "A deployment marker was deleted.")
	w.WriteHeader(http.StatusNoContent)
}
