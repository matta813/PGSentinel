package analyzer

import "github.com/matta813/pgsentinel/internal/models"

type Score struct {
	Overall    int            `json:"overall"`
	Categories map[string]int `json:"categories"`
}

func HealthScore(findings []models.Finding) Score {
	weights := map[models.Severity]int{models.SeverityInfo: 0, models.SeverityLow: 2, models.SeverityMedium: 6, models.SeverityHigh: 15, models.SeverityCritical: 35}
	cats := map[string]int{"Performance": 100, "Vacuum": 100, "Queries": 100, "Connections": 100, "Indexes": 100, "Configuration": 100, "Replication": 100}
	total := 0
	for _, f := range findings {
		if f.Status != "active" && f.Status != "acknowledged" {
			continue
		}
		w := weights[f.Severity]
		total += w
		if _, ok := cats[f.Category]; ok {
			cats[f.Category] -= w
			if cats[f.Category] < 0 {
				cats[f.Category] = 0
			}
		}
	}
	overall := 100 - total
	if overall < 0 {
		overall = 0
	}
	return Score{Overall: overall, Categories: cats}
}
