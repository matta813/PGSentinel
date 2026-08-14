package analyzer

import (
	"crypto/sha256"
	"encoding/hex"
	"github.com/matta813/pgsentinel/internal/models"
	"time"
)

func newFinding(rule, server, database, resource string, severity models.Severity, category, title, summary, impact string, confidence models.Confidence, evidence []models.Evidence) models.Finding {
	h := sha256.Sum256([]byte(rule + "|" + server + "|" + database + "|" + resource))
	fp := hex.EncodeToString(h[:12])
	now := time.Now().UTC()
	return models.Finding{ID: fp, RuleID: rule, Fingerprint: fp, ServerID: server, Database: database, Resource: resource, Severity: severity, Category: category, Title: title, Summary: summary, Cause: "The observed workload and PostgreSQL statistics match this rule's conditions.", Impact: impact, Evidence: evidence, Suggestions: []models.Suggestion{{Title: "Review the supporting evidence and workload history before changing production."}}, Confidence: confidence, Status: "active", StartedAt: now, UpdatedAt: now}
}
