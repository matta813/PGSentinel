package analyzer

import "github.com/matta813/pgsentinel/internal/models"

func (e *Engine) analyzeConfiguration(s models.Snapshot) []models.Finding {
	out := []models.Finding{}
	if !s.Capabilities["pg_stat_statements"] {
		out = append(out, newFinding("pgss-unavailable", s.ServerID, "", "", models.SeverityInfo, "Configuration", "Query monitoring is unavailable", "pg_stat_statements is not enabled on this PostgreSQL server.", "Query-level load, latency and regression analysis cannot be performed.", models.ConfidenceHigh, nil))
	}
	if s.Settings["track_io_timing"] == "off" {
		out = append(out, newFinding("io-timing-disabled", s.ServerID, "", "", models.SeverityInfo, "Configuration", "I/O timing is disabled", "track_io_timing is off, limiting diagnosis of storage latency.", "Read and write time attribution will be unavailable. Enabling it has overhead that should be evaluated.", models.ConfidenceHigh, nil))
	}
	return out
}
