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
	staticFS, err := webui.StaticFS()
	if err != nil {
		t.Fatalf("StaticFS() error = %v", err)
	}

	app := &App{
		logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
		webIndex:  []byte("<html><body>ok</body></html>"),
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
	if !strings.Contains(rec.Body.String(), "<html>") {
		t.Fatalf("expected html body, got %q", rec.Body.String())
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
