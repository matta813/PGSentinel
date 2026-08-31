package api

import (
	"database/sql"
	"github.com/google/uuid"
	"github.com/matta813/pgsentinel/internal/analyzer"
	"github.com/matta813/pgsentinel/internal/models"
	"math"
	"net/http"
	"strings"
	"time"
)

type applyProfileRequest struct {
	ScopeType  string `json:"scopeType"`
	ScopeValue string `json:"scopeValue"`
	Reason     string `json:"reason"`
	Replace    bool   `json:"replace"`
	Preview    bool   `json:"preview"`
}

func (a *API) listRuleProfiles(w http.ResponseWriter, r *http.Request) {
	v, e := a.store.ListRuleProfiles(r.Context())
	if e != nil {
		failure(w, 500, "Unable to load rule profiles", e)
		return
	}
	write(w, 200, map[string]any{"items": v, "specs": analyzer.ThresholdSpecs()})
}
func (a *API) createRuleProfile(w http.ResponseWriter, r *http.Request) {
	var p models.RuleProfile
	if !decode(w, r, &p) {
		return
	}
	p.ID = uuid.NewString()
	p.Name = strings.TrimSpace(p.Name)
	p.Description = strings.TrimSpace(p.Description)
	if p.Name == "" || len(p.Name) > 100 || len(p.Description) > 500 || len(p.Entries) < 1 || len(p.Entries) > 50 || !validProfileEntries(p.Entries) {
		failure(w, 422, "Invalid rule profile", nil)
		return
	}
	if e := a.store.SaveRuleProfile(r.Context(), &p); e != nil {
		failure(w, 409, "Unable to save rule profile", e)
		return
	}
	a.audit(r, "", "rule_profile.created", "rule_profile", p.ID, "A validated analyzer rule profile was created.")
	write(w, 201, p)
}
func validProfileEntries(v []models.RuleProfileEntry) bool {
	seen := map[string]bool{}
	specs := analyzer.ThresholdSpecs()
	for _, x := range v {
		s, ok := specs[x.RuleID]
		if !ok || seen[x.RuleID] || math.IsNaN(x.Value) || math.IsInf(x.Value, 0) || x.Value < s.Min || x.Value > s.Max {
			return false
		}
		seen[x.RuleID] = true
	}
	return true
}
func (a *API) deleteRuleProfile(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validID(id) {
		failure(w, 400, "Invalid rule profile ID", nil)
		return
	}
	if e := a.store.DeleteRuleProfile(r.Context(), id); e == sql.ErrNoRows {
		failure(w, 404, "Rule profile not found", nil)
		return
	} else if e != nil {
		failure(w, 500, "Unable to delete rule profile", e)
		return
	}
	a.audit(r, "", "rule_profile.deleted", "rule_profile", id, "A rule profile was deleted.")
	w.WriteHeader(204)
}
func (a *API) applyRuleProfile(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var q applyProfileRequest
	if !validID(id) || !decode(w, r, &q) {
		return
	}
	q.ScopeType = strings.TrimSpace(q.ScopeType)
	q.ScopeValue = strings.TrimSpace(q.ScopeValue)
	q.Reason = strings.TrimSpace(q.Reason)
	if q.Reason == "" || len(q.Reason) > 500 || !a.validThresholdScope(r, q.ScopeType, q.ScopeValue) {
		failure(w, 422, "Invalid profile application scope or reason", nil)
		return
	}
	p, e := a.store.GetRuleProfile(r.Context(), id)
	if e == sql.ErrNoRows {
		failure(w, 404, "Rule profile not found", nil)
		return
	} else if e != nil {
		failure(w, 500, "Unable to load rule profile", e)
		return
	}
	conflicts := []string{}
	for _, x := range p.Entries {
		var n int
		_ = a.store.DB.QueryRowContext(r.Context(), `SELECT COUNT(*) FROM threshold_overrides WHERE rule_id=? AND scope_type=? AND scope_value=?`, x.RuleID, q.ScopeType, q.ScopeValue).Scan(&n)
		if n > 0 {
			conflicts = append(conflicts, x.RuleID)
		}
	}
	if q.Preview {
		write(w, 200, map[string]any{"profile": p, "scopeType": q.ScopeType, "scopeValue": q.ScopeValue, "conflicts": conflicts, "willApply": len(p.Entries)})
		return
	}
	if len(conflicts) > 0 && !q.Replace {
		failure(w, 409, "Threshold conflicts require replace=true", nil)
		return
	}
	tx, e := a.store.DB.BeginTx(r.Context(), nil)
	if e != nil {
		failure(w, 500, "Unable to apply rule profile", e)
		return
	}
	defer tx.Rollback()
	for _, x := range p.Entries {
		if q.Replace {
			_, e = tx.ExecContext(r.Context(), `DELETE FROM threshold_overrides WHERE rule_id=? AND scope_type=? AND scope_value=?`, x.RuleID, q.ScopeType, q.ScopeValue)
			if e != nil {
				break
			}
		}
		now := time.Now().UTC().Format(time.RFC3339Nano)
		_, e = tx.ExecContext(r.Context(), `INSERT INTO threshold_overrides(id,rule_id,scope_type,scope_value,value,reason,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?)`, uuid.NewString(), x.RuleID, q.ScopeType, q.ScopeValue, x.Value, q.Reason, now, now)
		if e != nil {
			break
		}
	}
	if e == nil {
		e = tx.Commit()
	}
	if e != nil {
		failure(w, 409, "Unable to apply rule profile", e)
		return
	}
	a.audit(r, "", "rule_profile.applied", "rule_profile", id, "A validated rule profile was applied to an analyzer scope.")
	write(w, 200, map[string]any{"applied": len(p.Entries)})
}
func (a *API) validThresholdScope(r *http.Request, t, v string) bool {
	switch t {
	case "global":
		return v == ""
	case "server":
		return validID(v) && a.serverExists(r, v)
	case "tag":
		return validControlToken(strings.ToLower(v))
	}
	return false
}
