package app

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestAppWorkflowMainline(t *testing.T) {
	ctx := context.Background()
	application := mustNewTestApp(t, ctx)
	handler := application.routes()

	loginResp := invokeJSON(t, handler, http.MethodPost, "/api/session/login", map[string]interface{}{
		"password": "admin",
	})
	if !loginResp.OK {
		t.Fatal("expected login response ok")
	}

	providersResp := invokeJSON(t, handler, http.MethodGet, "/api/providers", nil)
	providerItems := providersResp.Data.(map[string]interface{})["items"].([]interface{})
	if len(providerItems) != 10 {
		t.Fatalf("expected 10 providers, got %d", len(providerItems))
	}

	profileResp := invokeJSON(t, handler, http.MethodPost, "/api/auth/profiles", map[string]interface{}{
		"providerKey": "guangya",
		"authMode":    "manual_token",
		"displayName": "Workflow Guangya",
		"token":       "token-workflow",
		"extra":       map[string]interface{}{},
	})
	profileData := profileResp.Data.(map[string]interface{})
	profileID := profileData["id"].(string)
	if profileID == "" {
		t.Fatal("expected profile id")
	}

	validateResp := invokeJSON(t, handler, http.MethodPost, "/api/auth/profiles/"+profileID+"/validate", nil)
	if got := validateResp.Data.(map[string]interface{})["status"].(string); got != "verified" {
		t.Fatalf("expected verified validation, got %s", got)
	}

	previewResp := invokeJSON(t, handler, http.MethodPost, "/api/plans/preview", map[string]interface{}{
		"sourceProvider": "baidu_netdisk",
		"targetProvider": "guangya",
		"thresholdMB":    1,
		"conflictPolicy": "overwrite_existing",
		"selectedRoots":  []string{"/demo"},
		"entries": []map[string]interface{}{
			{"path": "/demo/a.bin", "size": 2048, "md5": "md5-a"},
			{"path": "/demo/missing.bin", "size": 512},
			{"path": "/demo/pending.bin", "size": 10 * 1024 * 1024},
		},
	})
	previewData := previewResp.Data.(map[string]interface{})
	items := previewData["items"].([]interface{})
	if len(items) != 3 {
		t.Fatalf("expected 3 preview items, got %d", len(items))
	}

	localFile := filepath.Join(t.TempDir(), "existing.bin")
	if err := os.WriteFile(localFile, []byte("workflow"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	taskResp := invokeJSON(t, handler, http.MethodPost, "/api/tasks", map[string]interface{}{
		"sourceProvider":  "baidu_netdisk",
		"targetProvider":  "guangya",
		"targetProfileId": profileID,
		"thresholdMB":     1,
		"conflictPolicy":  "overwrite_existing",
		"selectedRoots":   []string{"/demo"},
		"entries": []map[string]interface{}{
			{"path": "/demo/a.bin", "size": 2048, "md5": "md5-a", "localPath": localFile},
			{"path": "/demo/missing.bin", "size": 512},
			{"path": "/demo/pending.bin", "size": 10 * 1024 * 1024},
		},
	})
	taskData := taskResp.Data.(map[string]interface{})
	taskID := taskData["task"].(map[string]interface{})["id"].(string)
	if taskID == "" {
		t.Fatal("expected task id")
	}

	pauseResp := invokeJSON(t, handler, http.MethodPost, "/api/tasks/"+taskID+"/pause", nil)
	if got := pauseResp.Data.(map[string]interface{})["task"].(map[string]interface{})["state"].(string); got != "paused" {
		t.Fatalf("expected paused, got %s", got)
	}

	resumeResp := invokeJSON(t, handler, http.MethodPost, "/api/tasks/"+taskID+"/resume", nil)
	if got := resumeResp.Data.(map[string]interface{})["task"].(map[string]interface{})["state"].(string); got != "ready" {
		t.Fatalf("expected ready after resume, got %s", got)
	}

	runResp := invokeJSON(t, handler, http.MethodPost, "/api/tasks/"+taskID+"/run", nil)
	runData := runResp.Data.(map[string]interface{})
	if got := runData["task"].(map[string]interface{})["state"].(string); got != "completed_with_errors" {
		t.Fatalf("expected completed_with_errors, got %s", got)
	}
	results := runData["results"].([]interface{})
	if len(results) != 3 {
		t.Fatalf("expected 3 task results, got %d", len(results))
	}

	evidenceResp := invokeJSON(t, handler, http.MethodGet, "/api/evidence/runtime", nil)
	evidenceData := evidenceResp.Data.(map[string]interface{})
	if got := int(evidenceData["totalTasks"].(float64)); got != 1 {
		t.Fatalf("expected totalTasks=1, got %d", got)
	}
	if got := len(evidenceData["recentResults"].([]interface{})); got == 0 {
		t.Fatal("expected recentResults to be populated")
	}
	if got := len(evidenceData["recentProbes"].([]interface{})); got == 0 {
		t.Fatal("expected recentProbes to be populated")
	}

	statusResp := invokeJSON(t, handler, http.MethodGet, "/api/status/providers", nil)
	statusItems := statusResp.Data.(map[string]interface{})["items"].([]interface{})
	foundGuangya := false
	for _, raw := range statusItems {
		item := raw.(map[string]interface{})
		if item["providerKey"].(string) != "guangya" {
			continue
		}
		foundGuangya = true
		if item["latestProbe"].(string) == "" {
			t.Fatal("expected latestProbe for guangya")
		}
		if item["lastTaskState"].(string) == "" {
			t.Fatal("expected lastTaskState for guangya")
		}
	}
	if !foundGuangya {
		t.Fatal("expected guangya in provider statuses")
	}

	retryResp := invokeJSON(t, handler, http.MethodPost, "/api/tasks/"+taskID+"/retry", nil)
	if got := retryResp.Data.(map[string]interface{})["task"].(map[string]interface{})["state"].(string); got != "ready" {
		t.Fatalf("expected ready after retry, got %s", got)
	}
}

func mustNewTestApp(t *testing.T, ctx context.Context) *App {
	t.Helper()

	cfg := Config{
		AppName:       "CloudPan Sync Go Test",
		Env:           "test",
		Addr:          ":0",
		DataDir:       t.TempDir(),
		DBPath:        filepath.Join(t.TempDir(), "app.db"),
		AdminPassword: "admin",
		LogLevel:      slog.LevelError,
	}
	app, err := New(ctx, cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	app.logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	t.Cleanup(func() {
		_ = app.store.Close()
	})
	return app
}

func invokeJSON(t *testing.T, handler http.Handler, method string, path string, body interface{}) Envelope {
	t.Helper()

	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("Marshal() error = %v", err)
		}
		reader = bytes.NewReader(payload)
	}

	req := httptest.NewRequest(method, path, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	var envelope Envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("Unmarshal() error = %v body=%s", err, rec.Body.String())
	}
	if rec.Code >= 400 {
		t.Fatalf("expected success for %s %s, got status=%d body=%s", method, path, rec.Code, rec.Body.String())
	}
	return envelope
}
