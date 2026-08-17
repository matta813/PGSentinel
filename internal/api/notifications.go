package api

import (
	"database/sql"
	"github.com/google/uuid"
	"github.com/matta813/pgsentinel/internal/models"
	"github.com/matta813/pgsentinel/internal/notifications"
	"net/http"
	"strings"
)

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
