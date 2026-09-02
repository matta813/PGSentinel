package api

import (
	"sort"
	"strings"

	"github.com/matta813/pgsentinel/internal/models"
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
