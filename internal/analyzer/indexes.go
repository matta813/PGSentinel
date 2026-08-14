package analyzer

import (
	"fmt"
	"gitlab.scruzzi.com/root/postgresqlui/internal/models"
	"regexp"
	"strings"
)

var indexBody = regexp.MustCompile(`(?i) USING \w+ \(([^)]+)\)(?: INCLUDE \(([^)]+)\))?(?: WHERE (.+))?$`)

func IndexFindings(serverID string, indexes []models.IndexStat) []models.Finding {
	out := []models.Finding{}
	groups := map[string][]models.IndexStat{}
	for _, i := range indexes {
		if i.Scans == 0 && !i.Primary && !i.Unique && i.SizeBytes >= 100*1024*1024 {
			out = append(out, newFinding("unused-index", serverID, i.Database, i.Schema+"."+i.Table+"."+i.Index, models.SeverityLow, "Indexes", "Potential unused index", fmt.Sprintf("%s has no observed scans.", i.Index), "Unused indexes consume storage and add write overhead.", models.ConfidenceMedium, []models.Evidence{{Label: "Index size", Value: fmt.Sprintf("%.1f MB", i.SizeBytes/1024/1024)}, {Label: "Observed scans", Value: "0"}}))
		}
		m := indexBody.FindStringSubmatch(i.Definition)
		if len(m) > 0 {
			key := i.Database + "/" + i.Schema + "/" + i.Table + "/" + strings.Join(m[1:], "|")
			groups[key] = append(groups[key], i)
		}
	}
	for _, g := range groups {
		if len(g) < 2 {
			continue
		}
		out = append(out, newFinding("duplicate-index", serverID, g[0].Database, g[0].Schema+"."+g[0].Table, models.SeverityMedium, "Indexes", "Potential duplicate indexes", fmt.Sprintf("%s and %s have matching indexed columns, includes and predicate.", g[0].Index, g[1].Index), "Duplicate indexes waste disk and increase write amplification. Never drop an index without checking constraints and workload history.", models.ConfidenceHigh, []models.Evidence{{Label: "Index A", Value: g[0].Index}, {Label: "Index B", Value: g[1].Index}}))
	}
	return out
}
