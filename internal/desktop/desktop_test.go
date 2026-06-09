package desktop

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestBuildChromeAppArgs(t *testing.T) {
	args := buildChromeAppArgs("http://127.0.0.1:8080/", "/tmp/cloudpan-sync-desktop-profile")
	if len(args) < 4 {
		t.Fatalf("expected chrome app args, got %v", args)
	}
	if args[0] != "--app=http://127.0.0.1:8080/" {
		t.Fatalf("expected app mode arg, got %v", args)
	}
	if args[2] != "--user-data-dir=/tmp/cloudpan-sync-desktop-profile" {
		t.Fatalf("expected dedicated user-data-dir arg, got %v", args)
	}
}

func TestWaitForReady(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := waitForReady(ctx, server.URL, 2*time.Second); err != nil {
		t.Fatalf("waitForReady() error = %v", err)
	}
}

func TestChromeCandidatesContainKnownBrowserNames(t *testing.T) {
	joined := strings.Join(chromeCandidates(), "\n")
	if !strings.Contains(strings.ToLower(joined), "chrome") && !strings.Contains(strings.ToLower(joined), "edge") {
		t.Fatalf("expected Chrome-compatible candidates, got %q", joined)
	}
}
