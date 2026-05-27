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
		"riskMode":       "fast",
		"riskOverride": map[string]interface{}{
			"requestIntervalMs":   1111,
			"directoryIntervalMs": 2222,
			"retryLimit":          2,
			"riskKeywords":        []string{"rate_limited"},
		},
		"executionMode":  "pre_scan_flat",
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
	metadata := previewData["metadata"].(map[string]interface{})
	if got := metadata["executionMode"].(string); got != "pre_scan_flat" {
		t.Fatalf("expected preview executionMode pre_scan_flat, got %s", got)
	}
	if got := metadata["recommendedExecutionMode"].(string); got == "" {
		t.Fatal("expected recommendedExecutionMode in preview metadata")
	}
	if got := metadata["recommendedExecutionModeReason"].(string); got == "" {
		t.Fatal("expected recommendedExecutionModeReason in preview metadata")
	}
	riskProfile := metadata["riskProfile"].(map[string]interface{})
	if got := int(riskProfile["requestIntervalMs"].(float64)); got != 1111 {
		t.Fatalf("expected preview risk requestIntervalMs 1111, got %d", got)
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
		"riskMode":        "fast",
		"riskOverride": map[string]interface{}{
			"requestIntervalMs":   1111,
			"directoryIntervalMs": 2222,
			"retryLimit":          2,
			"riskKeywords":        []string{"rate_limited"},
		},
		"executionMode":  "pre_scan_flat",
		"conflictPolicy": "overwrite_existing",
		"selectedRoots":  []string{"/demo"},
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
	createdMetadata := taskData["plan"].(map[string]interface{})["metadata"].(map[string]interface{})
	if got := createdMetadata["executionMode"].(string); got != "pre_scan_flat" {
		t.Fatalf("expected task executionMode pre_scan_flat, got %s", got)
	}
	createdRiskProfile := createdMetadata["riskProfile"].(map[string]interface{})
	if got := int(createdRiskProfile["directoryIntervalMs"].(float64)); got != 2222 {
		t.Fatalf("expected task risk directoryIntervalMs 2222, got %d", got)
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
	firstResultPayload := results[0].(map[string]interface{})["payload"].(map[string]interface{})
	if got := firstResultPayload["executionMode"].(string); got != "pre_scan_flat" {
		t.Fatalf("expected result executionMode pre_scan_flat, got %s", got)
	}

	evidenceResp := invokeJSON(t, handler, http.MethodGet, "/api/evidence/runtime", nil)
	evidenceData := evidenceResp.Data.(map[string]interface{})
	if got := int(evidenceData["totalTasks"].(float64)); got != 1 {
		t.Fatalf("expected totalTasks=1, got %d", got)
	}
	if got := int(evidenceData["pendingResultCount"].(float64)); got != 1 {
		t.Fatalf("expected pendingResultCount=1, got %d", got)
	}
	if got := len(evidenceData["recentResults"].([]interface{})); got == 0 {
		t.Fatal("expected recentResults to be populated")
	}
	if got := len(evidenceData["recentProbes"].([]interface{})); got == 0 {
		t.Fatal("expected recentProbes to be populated")
	}
	recentProbePayload := evidenceData["recentProbes"].([]interface{})[0].(map[string]interface{})["payload"].(map[string]interface{})
	if got := recentProbePayload["executionMode"].(string); got != "pre_scan_flat" {
		t.Fatalf("expected probe executionMode pre_scan_flat, got %s", got)
	}
	if got := int(recentProbePayload["riskProfile"].(map[string]interface{})["requestIntervalMs"].(float64)); got != 1111 {
		t.Fatalf("expected probe risk requestIntervalMs 1111, got %d", got)
	}
	if got := int(recentProbePayload["pendingCount"].(float64)); got != 1 {
		t.Fatalf("expected probe pendingCount 1, got %d", got)
	}
	if got := len(recentProbePayload["pendingTree"].([]interface{})); got == 0 {
		t.Fatal("expected pendingTree in recent probe payload")
	}
	if _, ok := recentProbePayload["runtime"].(map[string]interface{}); !ok {
		t.Fatalf("expected runtime payload in recent probe, got %#v", recentProbePayload["runtime"])
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
		summary := item["snapshotSummary"].(map[string]interface{})
		if got := summary["executionMode"].(string); got != "pre_scan_flat" {
			t.Fatalf("expected status summary executionMode pre_scan_flat, got %s", got)
		}
		if got := int(summary["riskProfile"].(map[string]interface{})["directoryIntervalMs"].(float64)); got != 2222 {
			t.Fatalf("expected status summary risk directoryIntervalMs 2222, got %d", got)
		}
		if got := int(summary["pendingCount"].(float64)); got != 1 {
			t.Fatalf("expected status summary pendingCount 1, got %d", got)
		}
		if got := len(summary["pendingTree"].([]interface{})); got == 0 {
			t.Fatal("expected pendingTree in status summary")
		}
		if _, ok := summary["runtime"].(map[string]interface{}); !ok {
			t.Fatalf("expected runtime summary in status snapshot, got %#v", summary["runtime"])
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
