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
	if !strings.Contains(body, `id="plan-preview-meta"`) {
		t.Fatalf("expected preview meta panel in html body, got %q", body)
	}
	if !strings.Contains(body, `id="task-directory-states"`) {
		t.Fatalf("expected task directory states panel in html body, got %q", body)
	}
	if !strings.Contains(body, `id="status-directory-states"`) {
		t.Fatalf("expected status directory states panel in html body, got %q", body)
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
