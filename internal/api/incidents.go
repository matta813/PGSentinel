package api

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/matta813/pgsentinel/internal/storage"
)

func (a *API) listIncidents(w http.ResponseWriter, r *http.Request) {
	filter := storage.IncidentFilter{Status: r.URL.Query().Get("status"), ServerID: r.URL.Query().Get("serverId"), Limit: 50}
	if filter.Status == "all" {
		filter.Status = ""
	}
	if filter.Status != "" && filter.Status != "active" && filter.Status != "resolved" {
		failure(w, http.StatusUnprocessableEntity, "Unsupported incident status", nil)
		return
	}
	if filter.ServerID != "" && !validID(filter.ServerID) {
		failure(w, http.StatusBadRequest, "Invalid server ID", nil)
		return
	}
	if raw := r.URL.Query().Get("limit"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 || value > 100 {
			failure(w, http.StatusUnprocessableEntity, "Limit must be between 1 and 100", nil)
			return
		}
		filter.Limit = value
	}
	if raw := r.URL.Query().Get("offset"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 0 || value > 10000 {
			failure(w, http.StatusUnprocessableEntity, "Offset must be between 0 and 10000", nil)
			return
		}
		filter.Offset = value
	}
	items, err := a.store.ListIncidents(r.Context(), filter)
	if err != nil {
		failure(w, http.StatusInternalServerError, "Unable to load incidents", err)
		return
	}
	write(w, http.StatusOK, items)
}

func (a *API) getIncident(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validFindingID(id) {
		failure(w, http.StatusBadRequest, "Invalid incident ID", nil)
		return
	}
	incident, err := a.store.GetIncident(r.Context(), id)
	if err == sql.ErrNoRows {
		failure(w, http.StatusNotFound, "Incident not found", nil)
		return
	}
	if err != nil {
		failure(w, http.StatusInternalServerError, "Unable to load incident", err)
		return
	}
	write(w, http.StatusOK, incident)
}
