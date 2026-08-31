package api

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/matta813/pgsentinel/internal/buildinfo"
	"github.com/matta813/pgsentinel/internal/models"
	"github.com/matta813/pgsentinel/internal/storage"
)

const diagnosticFindingLimit = 1000

type diagnosticManifest struct {
	Format      string         `json:"format"`
	GeneratedAt time.Time      `json:"generatedAt"`
	Build       buildinfo.Info `json:"build"`
	Redactions  []string       `json:"redactions"`
}

func (a *API) diagnosticBundle(w http.ResponseWriter, r *http.Request) {
	generatedAt := time.Now().UTC()
	servers, err := a.store.ListServers(r.Context())
	if err != nil {
		failure(w, http.StatusInternalServerError, "Unable to export diagnostic bundle", err)
		return
	}
	findings, err := a.store.FilterFindings(r.Context(), storage.FindingFilter{Limit: diagnosticFindingLimit})
	if err != nil {
		failure(w, http.StatusInternalServerError, "Unable to export diagnostic bundle", err)
		return
	}

	freshness := make([]models.CollectionResourceStatus, 0, len(servers)*len(monitoredResources))
	for i := range servers {
		items, loadErr := a.store.ListCollectionResources(r.Context(), servers[i].ID, generatedAt)
		if loadErr != nil {
			failure(w, http.StatusInternalServerError, "Unable to export diagnostic bundle", loadErr)
			return
		}
		freshness = append(freshness, items...)
		servers[i].Password = ""
		servers[i].User = "[redacted]"
		servers[i].Host = "[redacted]"
		if servers[i].LastError != "" {
			servers[i].LastError = "[redacted: connection error present]"
		}
	}
	redactQueryText(findings)

	manifest := diagnosticManifest{
		Format: "pgsentinel-diagnostic-bundle/v1", GeneratedAt: generatedAt, Build: buildinfo.Current(),
		Redactions: []string{"database credentials", "server hosts and usernames", "connection error details", "raw SQL text"},
	}
	var output bytes.Buffer
	archive := zip.NewWriter(&output)
	for _, file := range []struct {
		name  string
		value any
	}{{"manifest.json", manifest}, {"servers.json", servers}, {"findings.json", findings}, {"freshness.json", freshness}} {
		entry, createErr := archive.Create(file.name)
		if createErr != nil {
			failure(w, http.StatusInternalServerError, "Unable to export diagnostic bundle", createErr)
			return
		}
		encoder := json.NewEncoder(entry)
		encoder.SetIndent("", "  ")
		if encodeErr := encoder.Encode(file.value); encodeErr != nil {
			failure(w, http.StatusInternalServerError, "Unable to export diagnostic bundle", encodeErr)
			return
		}
	}
	if err := archive.Close(); err != nil {
		failure(w, http.StatusInternalServerError, "Unable to export diagnostic bundle", err)
		return
	}

	filename := fmt.Sprintf("pgsentinel-diagnostics-%s.zip", generatedAt.Format("20060102T150405Z"))
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	w.Header().Set("Content-Length", fmt.Sprint(output.Len()))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(output.Bytes())
	a.audit(r, "", "diagnostic_bundle.exported", "diagnostic_bundle", "", "A redacted diagnostic bundle was exported.")
}

func redactQueryText(findings []models.Finding) {
	for i := range findings {
		for j := range findings[i].Evidence {
			label := strings.ToLower(findings[i].Evidence[j].Label)
			if strings.Contains(label, "query text") || strings.Contains(label, "statement") || strings.Contains(label, "sql") {
				findings[i].Evidence[j].Value = "[redacted]"
			}
		}
	}
}
