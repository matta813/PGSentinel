package api

import (
	"database/sql"
	"net/http"

	"github.com/google/uuid"
	"github.com/matta813/pgsentinel/internal/models"
)

func (a *API) listNotificationRoutes(w http.ResponseWriter, r *http.Request) {
	routes, err := a.store.ListNotificationRoutes(r.Context())
	if err != nil {
		failure(w, 500, "Unable to load notification routes", err)
		return
	}
	write(w, 200, routes)
}
func (a *API) createNotificationRoute(w http.ResponseWriter, r *http.Request) {
	var request notificationRouteRequest
	if !decode(w, r, &request) {
		return
	}
	route := request.route(uuid.NewString())
	if message := validateNotificationRoute(route); message != "" {
		failure(w, 422, message, nil)
		return
	}
	if err := a.validateRouteReferences(r, &route); err != "" {
		failure(w, 422, err, nil)
		return
	}
	if err := a.store.CreateNotificationRoute(r.Context(), &route); err != nil {
		failure(w, 409, "Unable to save notification route", err)
		return
	}
	a.audit(r, "", "notification_route.created", "notification_route", route.ID, "A notification routing rule was added.")
	write(w, 201, route)
}
func (a *API) updateNotificationRoute(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validID(id) {
		failure(w, 400, "Invalid notification route ID", nil)
		return
	}
	var request notificationRouteRequest
	if !decode(w, r, &request) {
		return
	}
	route := request.route(id)
	if message := validateNotificationRoute(route); message != "" {
		failure(w, 422, message, nil)
		return
	}
	if message := a.validateRouteReferences(r, &route); message != "" {
		failure(w, 422, message, nil)
		return
	}
	if err := a.store.UpdateNotificationRoute(r.Context(), &route); err == sql.ErrNoRows {
		failure(w, 404, "Notification route not found", nil)
		return
	} else if err != nil {
		failure(w, 409, "Unable to update notification route", err)
		return
	}
	a.audit(r, "", "notification_route.updated", "notification_route", id, "A notification routing rule was edited.")
	routes, _ := a.store.ListNotificationRoutes(r.Context())
	for _, saved := range routes {
		if saved.ID == id {
			write(w, 200, saved)
			return
		}
	}
	write(w, 200, route)
}
func (a *API) deleteNotificationRoute(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validID(id) {
		failure(w, 400, "Invalid notification route ID", nil)
		return
	}
	if err := a.store.DeleteNotificationRoute(r.Context(), id); err == sql.ErrNoRows {
		failure(w, 404, "Notification route not found", nil)
		return
	} else if err != nil {
		failure(w, 500, "Unable to delete notification route", err)
		return
	}
	a.audit(r, "", "notification_route.deleted", "notification_route", id, "A notification routing rule was removed.")
	w.WriteHeader(204)
}
func (a *API) validateRouteReferences(r *http.Request, route *models.NotificationRoute) string {
	for _, id := range route.DestinationIDs {
		destination, err := a.store.GetNotificationDestination(r.Context(), id, false)
		if err != nil {
			return "A selected destination does not exist"
		}
		if !destination.Enabled {
			return "Disabled destinations cannot be assigned to a route"
		}
	}
	for _, id := range route.ServerIDs {
		if _, err := a.store.GetServer(r.Context(), id, false); err != nil {
			return "A selected server does not exist"
		}
	}
	return ""
}
