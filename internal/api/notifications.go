package api

import (
	"database/sql"
	"github.com/google/uuid"
	"github.com/matta813/pgsentinel/internal/models"
	"github.com/matta813/pgsentinel/internal/notifications"
	"net/http"
	"sort"
	"strconv"
	"strings"
)

type notificationRouteRequest struct {
	Name            string   `json:"name"`
	Enabled         bool     `json:"enabled"`
	Priority        int      `json:"priority"`
	Severities      []string `json:"severities"`
	Categories      []string `json:"categories"`
	ServerIDs       []string `json:"serverIds"`
	ServerTags      []string `json:"serverTags"`
	Transitions     []string `json:"transitions"`
	DestinationIDs  []string `json:"destinationIds"`
	CooldownSeconds int      `json:"cooldownSeconds"`
}

func normalizeRouteValues(values []string, fold bool) []string {
	seen := map[string]string{}
	for _, raw := range values {
		value := strings.TrimSpace(raw)
		if fold {
			value = strings.ToLower(value)
		}
		if value != "" {
			seen[value] = value
		}
	}
	out := make([]string, 0, len(seen))
	for _, v := range seen {
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}
func (v notificationRouteRequest) route(id string) models.NotificationRoute {
	return models.NotificationRoute{ID: id, Name: strings.TrimSpace(v.Name), Enabled: v.Enabled, Priority: v.Priority, Severities: normalizeRouteValues(v.Severities, false), Categories: normalizeRouteValues(v.Categories, true), ServerIDs: normalizeRouteValues(v.ServerIDs, false), ServerTags: normalizeRouteValues(v.ServerTags, true), Transitions: normalizeRouteValues(v.Transitions, false), DestinationIDs: normalizeRouteValues(v.DestinationIDs, false), CooldownSeconds: v.CooldownSeconds}
}
func validateNotificationRoute(v models.NotificationRoute) string {
	if v.Name == "" || len(v.Name) > 100 {
		return "Name is required and must not exceed 100 characters"
	}
	if v.Priority < 0 || v.Priority > 1000 {
		return "Priority must be between 0 and 1000"
	}
	if v.CooldownSeconds < 0 || v.CooldownSeconds > 86400 {
		return "Cooldown must be between 0 and 86400 seconds"
	}
	if len(v.DestinationIDs) == 0 || len(v.DestinationIDs) > 20 {
		return "Select between 1 and 20 destinations"
	}
	if len(v.Severities) > 5 || len(v.Categories) > 50 || len(v.ServerIDs) > 50 || len(v.ServerTags) > 50 || len(v.Transitions) > 4 {
		return "Route filters exceed their allowed size"
	}
	allowedSeverity := map[string]bool{"INFO": true, "LOW": true, "MEDIUM": true, "HIGH": true, "CRITICAL": true}
	for _, value := range v.Severities {
		if !allowedSeverity[value] {
			return "Invalid severity: " + value
		}
	}
	allowedTransition := map[string]bool{"new": true, "severity_increased": true, "reopened": true, "resolved": true}
	for _, value := range v.Transitions {
		if !allowedTransition[value] {
			return "Invalid lifecycle transition: " + value
		}
	}
	for _, value := range append(append([]string{}, v.Categories...), v.ServerTags...) {
		if len(value) > 64 {
			return "Categories and tags must not exceed 64 characters"
		}
	}
	for _, id := range append(append([]string{}, v.ServerIDs...), v.DestinationIDs...) {
		if !validID(id) {
			return "Server and destination IDs must be UUIDs"
		}
	}
	return ""
}

type notificationRequest struct {
	Name       string `json:"name"`
	Enabled    bool   `json:"enabled"`
	Provider   string `json:"provider"`
	ServerURL  string `json:"serverUrl"`
	Topic      string `json:"topic"`
	Token      string `json:"token"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	WebhookURL string `json:"webhookUrl"`
}

func (v notificationRequest) destination(id string) models.NotificationDestination {
	return models.NotificationDestination{ID: id, Provider: v.Provider, Name: strings.TrimSpace(v.Name), Enabled: v.Enabled, Config: map[string]string{
		"serverUrl": v.ServerURL, "topic": v.Topic, "token": v.Token, "username": v.Username, "password": v.Password, "webhookUrl": v.WebhookURL,
	}}
}

func validateNotificationDestination(v notificationRequest, policy notifications.TargetPolicy) string {
	if strings.TrimSpace(v.Name) == "" {
		return "Name is required"
	}
	switch v.Provider {
	case "ntfy":
		if strings.TrimSpace(v.ServerURL) == "" || strings.TrimSpace(v.Topic) == "" {
			return "Server URL and topic are required for ntfy"
		}
		if err := policy.ValidateURL(v.ServerURL); err != nil {
			return "Invalid server URL: " + err.Error()
		}
	case "webhook":
		if strings.TrimSpace(v.WebhookURL) == "" {
			return "Webhook URL is required"
		}
		if err := policy.ValidateURL(v.WebhookURL); err != nil {
			return "Invalid webhook URL: " + err.Error()
		}
	default:
		return "Unsupported notification provider"
	}
	return ""
}

func (a *API) listNotificationDestinations(w http.ResponseWriter, r *http.Request) {
	v, err := a.store.ListNotificationDestinations(r.Context())
	if err != nil {
		failure(w, 500, "Unable to load notification destinations", err)
		return
	}
	write(w, 200, v)
}

func (a *API) createNotificationDestination(w http.ResponseWriter, r *http.Request) {
	var request notificationRequest
	if !decode(w, r, &request) {
		return
	}
	if message := validateNotificationDestination(request, a.notificationPolicy); message != "" {
		failure(w, 422, message, nil)
		return
	}
	v := request.destination(uuid.NewString())
	if err := a.store.CreateNotificationDestination(r.Context(), &v); err != nil {
		failure(w, 409, "Unable to save notification destination", err)
		return
	}
	write(w, 201, v)
}

func (a *API) updateNotificationDestination(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validID(id) {
		failure(w, 400, "Invalid notification destination ID", nil)
		return
	}
	var request notificationRequest
	if !decode(w, r, &request) {
		return
	}
	if message := validateNotificationDestination(request, a.notificationPolicy); message != "" {
		failure(w, 422, message, nil)
		return
	}
	v := request.destination(id)
	if err := a.store.UpdateNotificationDestination(r.Context(), &v); err == sql.ErrNoRows {
		failure(w, 404, "Notification destination not found", nil)
		return
	} else if err != nil {
		failure(w, 409, "Unable to update notification destination", err)
		return
	}
	updated, _ := a.store.GetNotificationDestination(r.Context(), id, false)
	write(w, 200, updated)
}

func (a *API) deleteNotificationDestination(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validID(id) {
		failure(w, 400, "Invalid notification destination ID", nil)
		return
	}
	if err := a.store.DeleteNotificationDestination(r.Context(), id); err == sql.ErrNoRows {
		failure(w, 404, "Notification destination not found", nil)
		return
	} else if err != nil {
		failure(w, 500, "Unable to delete notification destination", err)
		return
	}
	w.WriteHeader(204)
}

func (a *API) testNotification(w http.ResponseWriter, r *http.Request) {
	var v notificationRequest
	if !decode(w, r, &v) {
		return
	}
	msg := notifications.Message{Title: "pgsentinel test notification", Body: "Notification delivery is configured correctly.", Severity: "INFO"}
	var p notifications.Provider
	switch v.Provider {
	case "ntfy":
		n := notifications.NewNtfy(v.ServerURL, v.Topic, a.notificationPolicy)
		n.Token = v.Token
		n.Username = v.Username
		n.Password = v.Password
		p = n
	case "webhook":
		hook, err := notifications.NewWebhook(v.WebhookURL, nil, a.notificationPolicy)
		if err != nil {
			failure(w, 422, "Invalid webhook configuration", err)
			return
		}
		p = hook
	default:
		failure(w, 422, "Unsupported notification provider", nil)
		return
	}
	if err := p.Send(r.Context(), msg); err != nil {
		failure(w, 502, "Test notification failed", err)
		return
	}
	write(w, 200, map[string]bool{"ok": true})
}

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
func (a *API) listNotificationDeliveries(w http.ResponseWriter, r *http.Request) {
	limit := 50
	offset := 0
	var err error
	if raw := r.URL.Query().Get("limit"); raw != "" {
		limit, err = strconv.Atoi(raw)
		if err != nil {
			failure(w, 422, "Limit must be an integer", nil)
			return
		}
	}
	if raw := r.URL.Query().Get("offset"); raw != "" {
		offset, err = strconv.Atoi(raw)
		if err != nil {
			failure(w, 422, "Offset must be an integer", nil)
			return
		}
	}
	history, err := a.store.ListNotificationDeliveryHistory(r.Context(), limit, offset)
	if err != nil {
		failure(w, 422, "Invalid delivery history request", err)
		return
	}
	write(w, 200, history)
}
