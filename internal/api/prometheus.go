package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/matta813/pgsentinel/internal/analyzer"
	"github.com/matta813/pgsentinel/internal/models"
)

func (a *API) prometheusMetrics(w http.ResponseWriter, r *http.Request) {
	if err := a.store.DB.PingContext(r.Context()); err != nil {
		http.Error(w, "storage unavailable", http.StatusServiceUnavailable)
		return
	}
	servers, err := a.store.ListServers(r.Context())
	if err != nil {
		http.Error(w, "unable to collect server metrics", http.StatusServiceUnavailable)
		return
	}
	findings, err := a.store.ListFindings(r.Context(), "active", "")
	if err != nil {
		http.Error(w, "unable to collect finding metrics", http.StatusServiceUnavailable)
		return
	}

	statuses := map[string]int{"healthy": 0, "unreachable": 0, "unknown": 0}
	for _, server := range servers {
		if _, ok := statuses[server.Status]; ok {
			statuses[server.Status]++
		} else {
			statuses["unknown"]++
		}
	}
	severities := map[models.Severity]int{}
	for _, finding := range findings {
		severities[finding.Severity]++
	}
	score := analyzer.HealthScore(findings)

	var output strings.Builder
	fmt.Fprintln(&output, "# HELP pgsentinel_up Whether the pgsentinel storage is reachable.")
	fmt.Fprintln(&output, "# TYPE pgsentinel_up gauge")
	fmt.Fprintln(&output, "pgsentinel_up 1")
	fmt.Fprintln(&output, "# HELP pgsentinel_servers Number of configured PostgreSQL servers by status.")
	fmt.Fprintln(&output, "# TYPE pgsentinel_servers gauge")
	for _, status := range []string{"healthy", "unreachable", "unknown"} {
		fmt.Fprintf(&output, "pgsentinel_servers{status=%q} %d\n", status, statuses[status])
	}
	fmt.Fprintln(&output, "# HELP pgsentinel_findings_active Number of active findings by severity.")
	fmt.Fprintln(&output, "# TYPE pgsentinel_findings_active gauge")
	for _, severity := range []models.Severity{models.SeverityCritical, models.SeverityHigh, models.SeverityMedium, models.SeverityLow, models.SeverityInfo} {
		fmt.Fprintf(&output, "pgsentinel_findings_active{severity=%q} %d\n", severity, severities[severity])
	}
	fmt.Fprintln(&output, "# HELP pgsentinel_health_score Current aggregate health score from 0 to 100.")
	fmt.Fprintln(&output, "# TYPE pgsentinel_health_score gauge")
	fmt.Fprintf(&output, "pgsentinel_health_score %d\n", score.Overall)

	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(output.String()))
}
