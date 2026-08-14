package api

import (
	"github.com/matta813/pgsentinel/internal/notifications"
	"net/http"
)

type notificationRequest struct {
	Provider   string `json:"provider"`
	ServerURL  string `json:"serverUrl"`
	Topic      string `json:"topic"`
	Token      string `json:"token"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	WebhookURL string `json:"webhookUrl"`
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
