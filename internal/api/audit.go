package api

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/matta813/pgsentinel/internal/models"
	"github.com/matta813/pgsentinel/internal/storage"
)

func (a *API) registerAuditRoutes() {
	a.mux.HandleFunc("GET /api/v1/audit-events", a.listAuditEvents)
}

func (a *API) audit(r *http.Request, actor, action, resourceType, resourceID, summary string) {
	if actor == "" && a.auth != nil {
		if session, ok := a.auth.Session(r); ok {
			actor = session.Username
		}
	}
	if actor == "" {
		actor = "system"
	}
	event := models.AuditEvent{Actor: actor, Action: action, ResourceType: resourceType, ResourceID: resourceID, Summary: summary}
	if err := a.store.RecordAuditEvent(r.Context(), &event); err != nil {
		a.log.Error("record audit event", "action", action, "resource_type", resourceType, "resource_id", resourceID, "error", err)
	}
}

func (a *API) listAuditEvents(w http.ResponseWriter, r *http.Request) {
	filter := storage.AuditFilter{
		Actor: strings.TrimSpace(r.URL.Query().Get("actor")), Action: strings.TrimSpace(r.URL.Query().Get("action")),
		ResourceType: strings.TrimSpace(r.URL.Query().Get("resourceType")), Search: strings.TrimSpace(r.URL.Query().Get("search")), Limit: 50,
	}
	if len(filter.Actor) > 100 || len(filter.Action) > 100 || len(filter.ResourceType) > 100 || len(filter.Search) > 200 {
		failure(w, http.StatusUnprocessableEntity, "Audit filters are too long", nil)
		return
	}
	for raw, destination := range map[string]*time.Time{"from": &filter.From, "to": &filter.To} {
		if value := r.URL.Query().Get(raw); value != "" {
			parsed, err := time.Parse(time.RFC3339, value)
			if err != nil {
				failure(w, http.StatusUnprocessableEntity, raw+" must be an RFC3339 timestamp", nil)
				return
			}
			*destination = parsed
		}
	}
	if !filter.From.IsZero() && !filter.To.IsZero() && filter.From.After(filter.To) {
		failure(w, http.StatusUnprocessableEntity, "from must not be after to", nil)
		return
	}
	for _, page := range []struct {
		raw         string
		destination *int
		minimum     int
		maximum     int
	}{{r.URL.Query().Get("limit"), &filter.Limit, 1, 100}, {r.URL.Query().Get("offset"), &filter.Offset, 0, 10000}} {
		if page.raw == "" {
			continue
		}
		value, err := strconv.Atoi(page.raw)
		if err != nil || value < page.minimum || value > page.maximum {
			failure(w, http.StatusUnprocessableEntity, "Invalid audit pagination", nil)
			return
		}
		*page.destination = value
	}
	items, err := a.store.ListAuditEvents(r.Context(), filter)
	if err != nil {
		failure(w, http.StatusInternalServerError, "Unable to load audit events", err)
		return
	}
	write(w, http.StatusOK, items)
}
