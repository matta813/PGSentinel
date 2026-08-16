package analyzer

import (
	"testing"

	"github.com/matta813/pgsentinel/internal/models"
)

func TestAcknowledgedFindingsStillAffectHealthScore(t *testing.T) {
	finding := models.Finding{Severity: models.SeverityHigh, Category: "Connections", Status: "acknowledged"}
	score := HealthScore([]models.Finding{finding})
	if score.Overall != 85 || score.Categories["Connections"] != 85 {
		t.Fatalf("acknowledged finding disappeared from score: %#v", score)
	}
	finding.Status = "resolved"
	if score := HealthScore([]models.Finding{finding}); score.Overall != 100 {
		t.Fatalf("resolved finding affected score: %#v", score)
	}
}
