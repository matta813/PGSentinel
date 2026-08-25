package api

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"net/http"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/matta813/pgsentinel/internal/analyzer"
	"github.com/matta813/pgsentinel/internal/models"
)

const operatorControlLimit = 200

func (a *API) listMaintenanceWindows(w http.ResponseWriter, r *http.Request) {
	items, err := a.store.ListMaintenanceWindows(r.Context(), operatorControlLimit, time.Now().UTC())
	if err != nil {
		failure(w, 500, "Unable to load maintenance windows", err)
		return
	}
	write(w, 200, items)
}
func (a *API) createMaintenanceWindow(w http.ResponseWriter, r *http.Request) {
	var item models.MaintenanceWindow
	if !decode(w, r, &item) {
		return
	}
	item.ID = uuid.NewString()
	item.Description = strings.TrimSpace(item.Description)
	item.ServerTag = strings.ToLower(strings.TrimSpace(item.ServerTag))
	item.Category = strings.TrimSpace(item.Category)
	item.RuleID = strings.TrimSpace(item.RuleID)
	now := time.Now().UTC()
	if item.Description == "" || len(item.Description) > 500 {
		failure(w, 422, "Description is required and must not exceed 500 characters", nil)
		return
	}
	if item.ServerID == "" && item.ServerTag == "" && item.Category == "" && item.RuleID == "" {
		failure(w, 422, "At least one maintenance scope is required", nil)
		return
	}
	if !validControlScope(item.ServerTag, item.Category, item.RuleID) {
		failure(w, 422, "Invalid maintenance scope", nil)
		return
	}
	if item.RuleID != "" && !knownFindingRule(item.RuleID) {
		failure(w, 422, "Unknown finding rule", nil)
		return
	}
	if !item.EndsAt.After(item.StartsAt) || !item.EndsAt.After(now) || item.EndsAt.Sub(item.StartsAt) > 30*24*time.Hour || item.StartsAt.After(now.Add(365*24*time.Hour)) {
		failure(w, 422, "Maintenance window must end in the future, last at most 30 days, and start within one year", nil)
		return
	}
	if item.ServerID != "" && !a.serverExists(r, item.ServerID) {
		failure(w, 422, "Unknown server", nil)
		return
	}
	if !a.controlCapacity(r, "maintenance_windows") {
		failure(w, 409, "Maintenance window limit reached", nil)
		return
	}
	if err := a.store.CreateMaintenanceWindow(r.Context(), &item); err != nil {
		failure(w, 409, "Unable to create maintenance window", err)
		return
	}
	item.State = temporalStateAPI(item.StartsAt, item.EndsAt, now)
	write(w, 201, item)
}
func (a *API) deleteMaintenanceWindow(w http.ResponseWriter, r *http.Request) {
	deleteOperatorControl(w, r, a.store.DeleteMaintenanceWindow, "Maintenance window")
}

func (a *API) listSuppressions(w http.ResponseWriter, r *http.Request) {
	items, err := a.store.ListSuppressions(r.Context(), operatorControlLimit, time.Now().UTC())
	if err != nil {
		failure(w, 500, "Unable to load suppressions", err)
		return
	}
	write(w, 200, items)
}
func (a *API) createSuppression(w http.ResponseWriter, r *http.Request) {
	var item models.FindingSuppression
	if !decode(w, r, &item) {
		return
	}
	item.ID = uuid.NewString()
	item.Reason = strings.TrimSpace(item.Reason)
	item.RuleID = strings.TrimSpace(item.RuleID)
	item.ServerTag = strings.ToLower(strings.TrimSpace(item.ServerTag))
	now := time.Now().UTC()
	if item.Reason == "" || len(item.Reason) > 500 {
		failure(w, 422, "Reason is required and must not exceed 500 characters", nil)
		return
	}
	if item.FindingID == "" && item.RuleID == "" {
		failure(w, 422, "A finding or rule is required", nil)
		return
	}
	if item.FindingID != "" && item.RuleID != "" {
		failure(w, 422, "Choose either a finding or a rule", nil)
		return
	}
	if item.RuleID != "" && item.ServerID == "" && item.ServerTag == "" {
		failure(w, 422, "Rule suppressions require a server or tag scope", nil)
		return
	}
	if item.FindingID != "" && !validFindingID(item.FindingID) {
		failure(w, 422, "Invalid finding ID", nil)
		return
	}
	if item.FindingID != "" {
		var findingServerID string
		if err := a.store.DB.QueryRowContext(r.Context(), `SELECT server_id FROM findings WHERE id=?`, item.FindingID).Scan(&findingServerID); err != nil {
			failure(w, 422, "Unknown finding", nil)
			return
		}
		if item.ServerID != "" && item.ServerID != findingServerID {
			failure(w, 422, "Finding does not belong to the scoped server", nil)
			return
		}
	}
	if !validControlScope(item.ServerTag, "", item.RuleID) {
		failure(w, 422, "Invalid suppression scope", nil)
		return
	}
	if item.RuleID != "" && !knownFindingRule(item.RuleID) {
		failure(w, 422, "Unknown finding rule", nil)
		return
	}
	if !item.ExpiresAt.After(now) || item.ExpiresAt.After(now.Add(30*24*time.Hour)) {
		failure(w, 422, "Suppression expiry must be within the next 30 days", nil)
		return
	}
	if item.ServerID != "" && !a.serverExists(r, item.ServerID) {
		failure(w, 422, "Unknown server", nil)
		return
	}
	if !a.controlCapacity(r, "finding_suppressions") {
		failure(w, 409, "Suppression limit reached", nil)
		return
	}
	if err := a.store.CreateSuppression(r.Context(), &item); err != nil {
		failure(w, 409, "Unable to create suppression", err)
		return
	}
	item.State = "active"
	write(w, 201, item)
}
func (a *API) deleteSuppression(w http.ResponseWriter, r *http.Request) {
	deleteOperatorControl(w, r, a.store.DeleteSuppression, "Suppression")
}

func (a *API) listThresholdOverrides(w http.ResponseWriter, r *http.Request) {
	items, err := a.store.ListThresholdOverrides(r.Context(), operatorControlLimit)
	if err != nil {
		failure(w, 500, "Unable to load threshold overrides", err)
		return
	}
	write(w, 200, map[string]any{"items": items, "specs": analyzer.ThresholdSpecs()})
}
func (a *API) createThresholdOverride(w http.ResponseWriter, r *http.Request) {
	var item models.ThresholdOverride
	if !decode(w, r, &item) {
		return
	}
	item.ID = uuid.NewString()
	item.RuleID = strings.TrimSpace(item.RuleID)
	item.ScopeType = strings.TrimSpace(item.ScopeType)
	item.ScopeValue = strings.TrimSpace(item.ScopeValue)
	item.Reason = strings.TrimSpace(item.Reason)
	spec, ok := analyzer.ThresholdSpecs()[item.RuleID]
	if !ok {
		failure(w, 422, "Unsupported threshold rule", nil)
		return
	}
	if math.IsNaN(item.Value) || math.IsInf(item.Value, 0) || item.Value < spec.Min || item.Value > spec.Max {
		failure(w, 422, "Threshold value is outside the safe range", nil)
		return
	}
	if item.Reason == "" || len(item.Reason) > 500 {
		failure(w, 422, "Reason is required and must not exceed 500 characters", nil)
		return
	}
	switch item.ScopeType {
	case "global":
		if item.ScopeValue != "" {
			failure(w, 422, "Global scope must not have a value", nil)
			return
		}
	case "server":
		if !validID(item.ScopeValue) || !a.serverExists(r, item.ScopeValue) {
			failure(w, 422, "Unknown server scope", nil)
			return
		}
	case "tag":
		item.ScopeValue = strings.ToLower(item.ScopeValue)
		if !validControlToken(item.ScopeValue) {
			failure(w, 422, "Invalid tag scope", nil)
			return
		}
	default:
		failure(w, 422, "Scope must be global, server, or tag", nil)
		return
	}
	if !a.controlCapacity(r, "threshold_overrides") {
		failure(w, 409, "Threshold override limit reached", nil)
		return
	}
	if err := a.store.CreateThresholdOverride(r.Context(), &item); err != nil {
		failure(w, 409, "A threshold override already exists for this scope", err)
		return
	}
	write(w, 201, item)
}
func (a *API) deleteThresholdOverride(w http.ResponseWriter, r *http.Request) {
	deleteOperatorControl(w, r, a.store.DeleteThresholdOverride, "Threshold override")
}

func deleteOperatorControl(w http.ResponseWriter, r *http.Request, remove func(context.Context, string) error, label string) {
	id := r.PathValue("id")
	if !validID(id) {
		failure(w, 400, "Invalid "+strings.ToLower(label)+" ID", nil)
		return
	}
	if err := remove(r.Context(), id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			failure(w, 404, label+" not found", nil)
		} else {
			failure(w, 500, "Unable to delete "+strings.ToLower(label), err)
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) serverExists(r *http.Request, id string) bool {
	if !validID(id) {
		return false
	}
	_, err := a.store.GetServer(r.Context(), id, false)
	return err == nil
}
func (a *API) controlCapacity(r *http.Request, table string) bool {
	allowed := map[string]bool{"maintenance_windows": true, "finding_suppressions": true, "threshold_overrides": true}
	if !allowed[table] {
		return false
	}
	var count int
	return a.store.DB.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM "+table).Scan(&count) == nil && count < operatorControlLimit
}
func validControlScope(tag, category, rule string) bool {
	return (tag == "" || validControlToken(tag)) && (category == "" || len(category) <= 100) && (rule == "" || validControlToken(rule))
}
func validControlToken(value string) bool {
	if value == "" || len(value) > 100 {
		return false
	}
	for _, char := range value {
		if !unicode.IsLetter(char) && !unicode.IsDigit(char) && char != '-' && char != '_' && char != '.' {
			return false
		}
	}
	return true
}
func knownFindingRule(value string) bool {
	return map[string]bool{"wal-receiver-disconnected": true, "wal-receiver-state": true, "standby-state": true, "standby-replay-lag": true, "inactive-slot-wal": true, "requested-checkpoints": true, "checkpoint-frequency": true, "connection-utilization": true, "idle-in-transaction": true, "long-transaction": true, "blocking-queries": true, "deadlocks": true, "rollback-ratio": true, "cache-hit": true, "dead-tuples": true, "vacuum-behind": true, "large-seq-scans": true, "stale-analyze": true, "query-impact": true, "pgss-unavailable": true, "io-timing-disabled": true, "unused-index": true, "duplicate-index": true, "query-regression": true}[value]
}
func temporalStateAPI(start, end, now time.Time) string {
	if now.Before(start) {
		return "upcoming"
	}
	if now.Before(end) {
		return "active"
	}
	return "expired"
}
