package app

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	webui "cloudpan-sync-go/web"
)

func TestHandleIndexServesHTML(t *testing.T) {
	indexHTML, err := webui.IndexHTML()
	if err != nil {
		t.Fatalf("IndexHTML() error = %v", err)
	}
	staticFS, err := webui.StaticFS()
	if err != nil {
		t.Fatalf("StaticFS() error = %v", err)
	}

	app := &App{
		logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
		webIndex:  indexHTML,
		webStatic: http.StripPrefix("/assets/", http.FileServer(http.FS(staticFS))),
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); !strings.Contains(got, "text/html") {
		t.Fatalf("expected html content type, got %q", got)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "CloudPan Sync Go Console") {
		t.Fatalf("expected console html body, got %q", body)
	}
	if !strings.Contains(body, `id="recent-results"`) {
		t.Fatalf("expected recent-results panel in html body, got %q", body)
	}
	if !strings.Contains(body, `id="recent-probes"`) {
		t.Fatalf("expected recent-probes panel in html body, got %q", body)
	}
	if !strings.Contains(body, `id="plan-execution-mode"`) {
		t.Fatalf("expected execution mode selector in html body, got %q", body)
	}
	if !strings.Contains(body, `id="plan-source-delete-policy"`) {
		t.Fatalf("expected source delete policy selector in html body, got %q", body)
	}
	if !strings.Contains(body, `id="plan-preview-meta"`) {
		t.Fatalf("expected preview meta panel in html body, got %q", body)
	}
	if !strings.Contains(body, `id="task-directory-states"`) {
		t.Fatalf("expected task directory states panel in html body, got %q", body)
	}
	if !strings.Contains(body, `id="task-pending-tree"`) {
		t.Fatalf("expected task pending tree panel in html body, got %q", body)
	}
	if !strings.Contains(body, `id="status-directory-states"`) {
		t.Fatalf("expected status directory states panel in html body, got %q", body)
	}
	if !strings.Contains(body, `id="status-pending-tree"`) {
		t.Fatalf("expected status pending tree panel in html body, got %q", body)
	}
	if !strings.Contains(body, `id="auto-recover-run"`) {
		t.Fatalf("expected auto recover controls in html body, got %q", body)
	}
	if !strings.Contains(body, `id="auto-recover-preview"`) {
		t.Fatalf("expected auto recover preview control in html body, got %q", body)
	}
	if !strings.Contains(body, `id="auto-recover-last-result-summary"`) {
		t.Fatalf("expected auto recover result summary in html body, got %q", body)
	}
	if !strings.Contains(body, `id="auto-recover-last-result-detail"`) {
		t.Fatalf("expected auto recover result detail in html body, got %q", body)
	}
	if !strings.Contains(body, `id="auto-recover-state"`) {
		t.Fatalf("expected auto recover state selector in html body, got %q", body)
	}
	if !strings.Contains(body, `value="waiting_auth_refresh"`) {
		t.Fatalf("expected waiting_auth_refresh option in html body, got %q", body)
	}
	if !strings.Contains(body, `value="waiting_local_restore"`) {
		t.Fatalf("expected waiting_local_restore option in html body, got %q", body)
	}
	if !strings.Contains(body, `value="waiting_manual_confirmation"`) {
		t.Fatalf("expected waiting_manual_confirmation option in html body, got %q", body)
	}
	if !strings.Contains(body, `value="waiting_retry_limit"`) {
		t.Fatalf("expected waiting_retry_limit option in html body, got %q", body)
	}
	if !strings.Contains(body, `id="risk-max-concurrent"`) {
		t.Fatalf("expected risk concurrency input in html body, got %q", body)
	}
}

func TestRoutesServeStaticAssets(t *testing.T) {
	staticFS, err := webui.StaticFS()
	if err != nil {
		t.Fatalf("StaticFS() error = %v", err)
	}

	app := &App{
		logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
		webIndex:  []byte("<html><body>ok</body></html>"),
		webStatic: http.StripPrefix("/assets/", http.FileServer(http.FS(staticFS))),
	}

	req := httptest.NewRequest(http.MethodGet, "/assets/styles.css", nil)
	rec := httptest.NewRecorder()
	app.routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), ":root") {
		t.Fatalf("expected stylesheet content, got %q", rec.Body.String())
	}
}

func TestRoutesServeAppJSIncludesRetryEvidenceLabels(t *testing.T) {
	staticFS, err := webui.StaticFS()
	if err != nil {
		t.Fatalf("StaticFS() error = %v", err)
	}

	app := &App{
		logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
		webIndex:  []byte("<html><body>ok</body></html>"),
		webStatic: http.StripPrefix("/assets/", http.FileServer(http.FS(staticFS))),
	}

	req := httptest.NewRequest(http.MethodGet, "/assets/app.js", nil)
	rec := httptest.NewRecorder()
	app.routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "retrySelectedPaths") {
		t.Fatalf("expected retrySelectedPaths evidence in app.js, got %q", body)
	}
	if !strings.Contains(body, "CALIBRATED") {
		t.Fatalf("expected CALIBRATED risk resolution detail in app.js, got %q", body)
	}
	if !strings.Contains(body, "OVERRIDE FIELDS") {
		t.Fatalf("expected OVERRIDE FIELDS risk resolution detail in app.js, got %q", body)
	}
	if !strings.Contains(body, "waiting_auth_refresh") {
		t.Fatalf("expected waiting_auth_refresh recoverState affordance in app.js, got %q", body)
	}
	if !strings.Contains(body, "waiting_local_restore") {
		t.Fatalf("expected waiting_local_restore recoverState affordance in app.js, got %q", body)
	}
	if !strings.Contains(body, "waiting_manual_confirmation") {
		t.Fatalf("expected waiting_manual_confirmation recoverState affordance in app.js, got %q", body)
	}
	if !strings.Contains(body, "waiting_retry_limit") {
		t.Fatalf("expected waiting_retry_limit recoverState affordance in app.js, got %q", body)
	}
	if !strings.Contains(body, "<th>Retry Scope</th>") {
		t.Fatalf("expected Retry Scope column in app.js, got %q", body)
	}
	if !strings.Contains(body, "<th>Retry Paths</th>") {
		t.Fatalf("expected Retry Paths column in app.js, got %q", body)
	}
}
