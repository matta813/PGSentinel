package analyzer

import "github.com/matta813/pgsentinel/internal/models"

type Engine struct{ Thresholds Thresholds }

func New(t Thresholds) *Engine { return &Engine{Thresholds: t} }

func (e *Engine) Analyze(s models.Snapshot) []models.Finding {
	findings := []models.Finding{}
	findings = append(findings, e.analyzeReplicationState(s)...)
	findings = append(findings, e.analyzeWALArchive(s)...)
	findings = append(findings, e.analyzeReplicationSlots(s)...)
	findings = append(findings, e.analyzeWALCheckpoints(s)...)
	findings = append(findings, e.analyzeConnections(s)...)
	findings = append(findings, e.analyzeWaitEvents(s)...)
	findings = append(findings, e.analyzeLocks(s)...)
	findings = append(findings, e.analyzeDatabases(s)...)
	findings = append(findings, e.analyzeTables(s)...)
	findings = append(findings, e.analyzeQueries(s)...)
	findings = append(findings, e.analyzeConfiguration(s)...)
	return findings
}
