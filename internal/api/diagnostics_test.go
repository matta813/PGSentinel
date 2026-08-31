package api

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/matta813/pgsentinel/internal/models"
)

func TestRedactQueryText(t *testing.T) {
	findings := []models.Finding{{Evidence: []models.Evidence{{Label: "Query text", Value: "select secret"}, {Label: "Calls", Value: "12"}}}}
	redactQueryText(findings)
	if findings[0].Evidence[0].Value != "[redacted]" || findings[0].Evidence[1].Value != "12" {
		t.Fatalf("unexpected evidence redaction: %#v", findings[0].Evidence)
	}
}

func TestDiagnosticBundleIsBoundedAndRedacted(t *testing.T) {
	h := testAPI(t)
	response := httptest.NewRecorder()
	h.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/servers", strings.NewReader(`{"name":"production","host":"db.internal","user":"monitor","password":"super-secret","tags":["production"]}`)))
	if response.Code != http.StatusCreated {
		t.Fatalf("create server=%d %s", response.Code, response.Body.String())
	}

	response = httptest.NewRecorder()
	h.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/diagnostic-bundle", nil))
	if response.Code != http.StatusOK || response.Header().Get("Content-Type") != "application/zip" {
		t.Fatalf("bundle=%d %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Header().Get("Content-Disposition"), "pgsentinel-diagnostics-") {
		t.Fatalf("unexpected disposition: %q", response.Header().Get("Content-Disposition"))
	}
	reader, err := zip.NewReader(bytes.NewReader(response.Body.Bytes()), int64(response.Body.Len()))
	if err != nil {
		t.Fatal(err)
	}
	files := map[string][]byte{}
	for _, file := range reader.File {
		opened, openErr := file.Open()
		if openErr != nil {
			t.Fatal(openErr)
		}
		body, readErr := io.ReadAll(opened)
		_ = opened.Close()
		if readErr != nil {
			t.Fatal(readErr)
		}
		files[file.Name] = body
	}
	for _, name := range []string{"manifest.json", "servers.json", "findings.json", "freshness.json"} {
		if !json.Valid(files[name]) {
			t.Fatalf("%s is missing or invalid", name)
		}
	}
	joined := string(bytes.Join([][]byte{files["manifest.json"], files["servers.json"], files["findings.json"], files["freshness.json"]}, nil))
	for _, secret := range []string{"db.internal", "monitor", "super-secret"} {
		if strings.Contains(joined, secret) {
			t.Fatalf("bundle leaked %q: %s", secret, joined)
		}
	}
	if !strings.Contains(joined, "[redacted]") {
		t.Fatalf("bundle does not document redaction: %s", joined)
	}

	response = httptest.NewRecorder()
	h.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/audit-events?action=diagnostic_bundle.exported", nil))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "diagnostic_bundle.exported") {
		t.Fatalf("audit=%d %s", response.Code, response.Body.String())
	}
}
