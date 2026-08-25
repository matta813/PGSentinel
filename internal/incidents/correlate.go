package incidents

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/matta813/pgsentinel/internal/models"
)

const correlationWindow = 15 * time.Minute

var relatedCategories = map[string]map[string]string{
	"connections":  {"locks": "connection pressure and lock waits are operationally related", "transactions": "connection and transaction pressure overlap"},
	"locks":        {"queries": "blocked work and query pressure are operationally related", "transactions": "lock waits and transaction lifetime are operationally related"},
	"replication":  {"wal": "replication progress and WAL pressure share the PostgreSQL write path"},
	"transactions": {"vacuum": "long transactions can overlap vacuum cleanup pressure"},
	"performance":  {"queries": "query workload and observed performance pressure overlap"},
}

var sameCategorySignals = map[string]bool{
	"connections": true, "locks": true, "replication": true, "transactions": true, "vacuum": true, "wal": true,
}

func Correlate(findings []models.Finding) []models.Incident {
	if len(findings) < 2 {
		return nil
	}
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].StartedAt.Equal(findings[j].StartedAt) {
			return findings[i].ID < findings[j].ID
		}
		return findings[i].StartedAt.Before(findings[j].StartedAt)
	})
	parent := make([]int, len(findings))
	earliest := make([]time.Time, len(findings))
	latest := make([]time.Time, len(findings))
	reasons := make(map[int][]string)
	for i := range parent {
		parent[i] = i
		earliest[i], latest[i] = findings[i].StartedAt, findings[i].StartedAt
	}
	var root func(int) int
	root = func(i int) int {
		if parent[i] != i {
			parent[i] = root(parent[i])
		}
		return parent[i]
	}
	union := func(a, b int, reason string) {
		ra, rb := root(a), root(b)
		if ra != rb {
			combinedStart, combinedEnd := earliest[ra], latest[ra]
			if earliest[rb].Before(combinedStart) {
				combinedStart = earliest[rb]
			}
			if latest[rb].After(combinedEnd) {
				combinedEnd = latest[rb]
			}
			if combinedEnd.Sub(combinedStart) > correlationWindow {
				return
			}
			parent[rb] = ra
			earliest[ra], latest[ra] = combinedStart, combinedEnd
			reasons[ra] = append(reasons[ra], reasons[rb]...)
			delete(reasons, rb)
		}
		reasons[root(a)] = append(reasons[root(a)], reason)
	}
	for i := range findings {
		for j := i + 1; j < len(findings); j++ {
			if findings[j].StartedAt.Sub(findings[i].StartedAt) > correlationWindow {
				break
			}
			if findings[i].ServerID != findings[j].ServerID {
				continue
			}
			if reason := relationship(findings[i], findings[j]); reason != "" {
				union(i, j, reason)
			}
		}
	}
	groups := make(map[int][]models.Finding)
	for i, finding := range findings {
		groups[root(i)] = append(groups[root(i)], finding)
	}
	out := make([]models.Incident, 0)
	for groupRoot, group := range groups {
		if len(group) < 2 {
			continue
		}
		out = append(out, buildIncident(group, uniqueSorted(reasons[groupRoot])))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].StartedAt.After(out[j].StartedAt) })
	return out
}

func relationship(a, b models.Finding) string {
	if a.Resource != "" && a.Resource == b.Resource {
		return fmt.Sprintf("both findings reference resource %s", a.Resource)
	}
	left, right := strings.ToLower(a.Category), strings.ToLower(b.Category)
	if left == right && sameCategorySignals[left] {
		return fmt.Sprintf("both findings concern the PostgreSQL %s subsystem", a.Category)
	}
	if reason := relatedCategories[left][right]; reason != "" {
		return reason
	}
	if reason := relatedCategories[right][left]; reason != "" {
		return reason
	}
	return ""
}

func buildIncident(findings []models.Finding, rationale []string) models.Incident {
	started, updated := findings[0].StartedAt, findings[0].UpdatedAt
	status := "resolved"
	severity := models.SeverityInfo
	var resolvedAt *time.Time
	for _, finding := range findings {
		if finding.StartedAt.Before(started) {
			started = finding.StartedAt
		}
		if finding.UpdatedAt.After(updated) {
			updated = finding.UpdatedAt
		}
		if finding.Status != "resolved" {
			status = "active"
		}
		if severityRank(finding.Severity) > severityRank(severity) {
			severity = finding.Severity
		}
		if finding.ResolvedAt != nil && (resolvedAt == nil || finding.ResolvedAt.After(*resolvedAt)) {
			at := *finding.ResolvedAt
			resolvedAt = &at
		}
	}
	digest := sha256.Sum256([]byte(findings[0].ServerID + "\x00" + findings[0].ID))
	incident := models.Incident{
		ID: hex.EncodeToString(digest[:12]), ServerID: findings[0].ServerID,
		Title:     "Overlapping PostgreSQL operational findings",
		Summary:   fmt.Sprintf("%d findings occurred during the same 15-minute period and match an explicit PostgreSQL operational relationship. They may be related; this grouping does not establish causation.", len(findings)),
		Rationale: rationale, Severity: severity, Status: status, StartedAt: started, UpdatedAt: updated, Findings: findings,
	}
	if status == "resolved" {
		incident.ResolvedAt = resolvedAt
	}
	return incident
}

func uniqueSorted(values []string) []string {
	seen := make(map[string]bool)
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value != "" && !seen[value] {
			seen[value] = true
			out = append(out, value)
		}
	}
	sort.Strings(out)
	return out
}

func severityRank(value models.Severity) int {
	return map[models.Severity]int{models.SeverityInfo: 1, models.SeverityLow: 2, models.SeverityMedium: 3, models.SeverityHigh: 4, models.SeverityCritical: 5}[value]
}
