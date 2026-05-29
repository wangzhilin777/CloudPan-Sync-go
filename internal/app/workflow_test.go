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
	"sort"
	"strings"
	"testing"
	"time"

	"cloudpan-sync-go/internal/auth"
	"cloudpan-sync-go/internal/provider"
	sqlitestore "cloudpan-sync-go/internal/store/sqlite"
	"cloudpan-sync-go/internal/task"
)

func TestAppWorkflowMainline(t *testing.T) {
	ctx := context.Background()
	application := mustNewTestApp(t, ctx)
	handler := application.routes()
	providerServer, _ := newAppPan123OpenTestServer(t)
	t.Cleanup(providerServer.Close)

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
		"providerKey": "123_open",
		"authMode":    "manual_token",
		"displayName": "Workflow 123",
		"token":       "token-workflow",
		"extra": map[string]interface{}{
			"apiEndpoint": providerServer.URL,
		},
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
		"sourceProvider":     "baidu_netdisk",
		"targetProvider":     "123_open",
		"thresholdMB":        1,
		"riskMode":           "fast",
		"sourceDeletePolicy": "record_only",
		"riskOverride": map[string]interface{}{
			"requestIntervalMs":   1111,
			"directoryIntervalMs": 2222,
			"retryLimit":          2,
			"maxConcurrent":       1,
			"autoRetryStartHour":  1,
			"autoRetryEndHour":    7,
			"riskKeywords":        []string{"rate_limited"},
		},
		"executionMode":  "pre_scan_flat",
		"conflictPolicy": "overwrite_existing",
		"selectedRoots":  []string{"/demo"},
		"entries": []map[string]interface{}{
			{"path": "/demo/a.bin", "size": 2048, "md5": "md5-a"},
			{"path": "/demo/deleted.bin", "deleted": true, "deletedAt": "2026-05-29T10:00:00Z", "deleteReason": "source_removed"},
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
	if got := int(metadata["deletedEntryCount"].(float64)); got != 1 {
		t.Fatalf("expected preview deletedEntryCount 1, got %d", got)
	}
	deletionRecords := metadata["sourceDeletionRecords"].([]interface{})
	if len(deletionRecords) != 1 {
		t.Fatalf("expected preview sourceDeletionRecords 1, got %#v", deletionRecords)
	}
	if got := metadata["executionMode"].(string); got != "pre_scan_flat" {
		t.Fatalf("expected preview executionMode pre_scan_flat, got %s", got)
	}
	if got := metadata["sourceDeletePolicy"].(string); got != "record_only" {
		t.Fatalf("expected preview sourceDeletePolicy record_only, got %s", got)
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
	riskResolution := metadata["riskProfileResolution"].(map[string]interface{})
	if got := riskResolution["providerKey"].(string); got != "123_open" {
		t.Fatalf("expected preview risk providerKey 123_open, got %s", got)
	}
	overrideFields := riskResolution["overrideFields"].([]interface{})
	if len(overrideFields) < 5 {
		t.Fatalf("expected preview override fields, got %#v", overrideFields)
	}

	localFile := filepath.Join(t.TempDir(), "existing.bin")
	if err := os.WriteFile(localFile, []byte("workflow"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	taskResp := invokeJSON(t, handler, http.MethodPost, "/api/tasks", map[string]interface{}{
		"sourceProvider":     "baidu_netdisk",
		"targetProvider":     "123_open",
		"targetProfileId":    profileID,
		"thresholdMB":        1,
		"riskMode":           "fast",
		"sourceDeletePolicy": "record_only",
		"riskOverride": map[string]interface{}{
			"requestIntervalMs":   1111,
			"directoryIntervalMs": 2222,
			"retryLimit":          2,
			"maxConcurrent":       1,
			"autoRetryStartHour":  1,
			"autoRetryEndHour":    7,
			"riskKeywords":        []string{"rate_limited"},
		},
		"executionMode":  "pre_scan_flat",
		"conflictPolicy": "overwrite_existing",
		"selectedRoots":  []string{"/demo"},
		"entries": []map[string]interface{}{
			{"path": "/demo/a.bin", "size": 2048, "md5": "md5-a", "localPath": localFile},
			{"path": "/demo/deleted.bin", "deleted": true, "deletedAt": "2026-05-29T10:00:00Z", "deleteReason": "source_removed"},
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
	if got := int(createdMetadata["deletedEntryCount"].(float64)); got != 1 {
		t.Fatalf("expected task deletedEntryCount 1, got %d", got)
	}
	if got := createdMetadata["executionMode"].(string); got != "pre_scan_flat" {
		t.Fatalf("expected task executionMode pre_scan_flat, got %s", got)
	}
	if got := createdMetadata["sourceDeletePolicy"].(string); got != "record_only" {
		t.Fatalf("expected task sourceDeletePolicy record_only, got %s", got)
	}
	createdRiskProfile := createdMetadata["riskProfile"].(map[string]interface{})
	if got := int(createdRiskProfile["directoryIntervalMs"].(float64)); got != 2222 {
		t.Fatalf("expected task risk directoryIntervalMs 2222, got %d", got)
	}
	if got := int(createdRiskProfile["maxConcurrent"].(float64)); got != 1 {
		t.Fatalf("expected task risk maxConcurrent 1, got %d", got)
	}
	createdResolution := createdMetadata["riskProfileResolution"].(map[string]interface{})
	if got := createdResolution["providerKey"].(string); got != "123_open" {
		t.Fatalf("expected task risk providerKey 123_open, got %s", got)
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
	if got := runData["task"].(map[string]interface{})["state"].(string); got != "blocked" {
		t.Fatalf("expected blocked, got %s", got)
	}
	results := runData["results"].([]interface{})
	if len(results) != 3 {
		t.Fatalf("expected 3 task results, got %d", len(results))
	}
	firstResultPayload := results[0].(map[string]interface{})["payload"].(map[string]interface{})
	if got := firstResultPayload["executionMode"].(string); got != "pre_scan_flat" {
		t.Fatalf("expected result executionMode pre_scan_flat, got %s", got)
	}
	runtimeData := runData["runtime"].(map[string]interface{})
	if got := int(runtimeData["sourceDeletionCount"].(float64)); got != 1 {
		t.Fatalf("expected runtime sourceDeletionCount 1, got %d", got)
	}
	if got := len(runtimeData["sourceDeletionRecords"].([]interface{})); got != 1 {
		t.Fatalf("expected runtime sourceDeletionRecords 1, got %d", got)
	}
	if got := runtimeData["blockedReason"].(string); got != "retry_queue_requires_local_file_restore" {
		t.Fatalf("expected blockedReason retry_queue_requires_local_file_restore, got %s", got)
	}
	if got := runtimeData["blockedAction"].(string); got != "restore_local_source_file" {
		t.Fatalf("expected blockedAction restore_local_source_file, got %s", got)
	}
	if got := runtimeData["blockedAdvice"].(string); got == "" {
		t.Fatal("expected blockedAdvice on runtime payload")
	}

	evidenceResp := invokeJSON(t, handler, http.MethodGet, "/api/evidence/runtime", nil)
	evidenceData := evidenceResp.Data.(map[string]interface{})
	if got := int(evidenceData["totalTasks"].(float64)); got != 1 {
		t.Fatalf("expected totalTasks=1, got %d", got)
	}
	if got := int(evidenceData["blockedTasks"].(float64)); got != 1 {
		t.Fatalf("expected blockedTasks=1, got %d", got)
	}
	if got := int(evidenceData["autoRecoverTasks"].(float64)); got != 0 {
		t.Fatalf("expected autoRecoverTasks=0, got %d", got)
	}
	autoRetryPolicy := evidenceData["autoRetryPolicy"].(map[string]interface{})
	if got := autoRetryPolicy["tick"].(string); got == "" {
		t.Fatal("expected autoRetryPolicy.tick in evidence summary")
	}
	if got := int(autoRetryPolicy["batchLimit"].(float64)); got != 3 {
		t.Fatalf("expected autoRetryPolicy.batchLimit 3, got %d", got)
	}
	if got := int(autoRetryPolicy["limitPerLane"].(float64)); got != 1 {
		t.Fatalf("expected autoRetryPolicy.limitPerLane 1, got %d", got)
	}
	if got := int(autoRetryPolicy["limitPerProtocolGroup"].(float64)); got != 1 {
		t.Fatalf("expected autoRetryPolicy.limitPerProtocolGroup 1, got %d", got)
	}
	if got := int(evidenceData["pendingResultCount"].(float64)); got != 1 {
		t.Fatalf("expected pendingResultCount=1, got %d", got)
	}
	if got := int(evidenceData["sourceDeletionCount"].(float64)); got != 1 {
		t.Fatalf("expected sourceDeletionCount=1, got %d", got)
	}
	blockedActions := evidenceData["blockedActions"].([]interface{})
	if len(blockedActions) == 0 {
		t.Fatal("expected blockedActions in evidence summary")
	}
	if got := blockedActions[0].(map[string]interface{})["action"].(string); got != "restore_local_source_file" {
		t.Fatalf("expected blocked action restore_local_source_file, got %s", got)
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
	if got := recentProbePayload["sourceDeletePolicy"].(string); got != "record_only" {
		t.Fatalf("expected probe sourceDeletePolicy record_only, got %s", got)
	}
	if got := int(recentProbePayload["riskProfile"].(map[string]interface{})["requestIntervalMs"].(float64)); got != 1111 {
		t.Fatalf("expected probe risk requestIntervalMs 1111, got %d", got)
	}
	if got := int(recentProbePayload["pendingCount"].(float64)); got != 1 {
		t.Fatalf("expected probe pendingCount 1, got %d", got)
	}
	if got := int(recentProbePayload["retryableCount"].(float64)); got != 1 {
		t.Fatalf("expected probe retryableCount 1, got %d", got)
	}
	if got := int(recentProbePayload["blockedRetryCount"].(float64)); got != 1 {
		t.Fatalf("expected probe blockedRetryCount 1, got %d", got)
	}
	if got := recentProbePayload["taskState"].(string); got != "blocked" {
		t.Fatalf("expected probe taskState blocked, got %s", got)
	}
	if got := recentProbePayload["retrySummary"].(map[string]interface{})["blockedAction"].(string); got != "restore_local_source_file" {
		t.Fatalf("expected probe blockedAction restore_local_source_file, got %s", got)
	}
	if got := len(recentProbePayload["pendingTree"].([]interface{})); got == 0 {
		t.Fatal("expected pendingTree in recent probe payload")
	}
	if _, ok := recentProbePayload["runtime"].(map[string]interface{}); !ok {
		t.Fatalf("expected runtime payload in recent probe, got %#v", recentProbePayload["runtime"])
	}
	protocolCoverage := evidenceData["protocolCoverage"].([]interface{})
	if len(protocolCoverage) == 0 {
		t.Fatal("expected protocolCoverage in evidence summary")
	}
	foundCoverage := false
	for _, raw := range protocolCoverage {
		item := raw.(map[string]interface{})
		if item["protocolGroup"].(string) != "aliyun_123_open" {
			continue
		}
		foundCoverage = true
		if got := int(item["providerCount"].(float64)); got == 0 {
			t.Fatalf("expected providerCount for aliyun_123_open, got %#v", item)
		}
		if got := int(item["taskCount"].(float64)); got == 0 {
			t.Fatalf("expected taskCount for aliyun_123_open, got %#v", item)
		}
	}
	if !foundCoverage {
		t.Fatalf("expected aliyun_123_open in protocolCoverage, got %#v", protocolCoverage)
	}

	statusResp := invokeJSON(t, handler, http.MethodGet, "/api/status/providers", nil)
	statusItems := statusResp.Data.(map[string]interface{})["items"].([]interface{})
	foundProvider := false
	for _, raw := range statusItems {
		item := raw.(map[string]interface{})
		if item["providerKey"].(string) != "123_open" {
			continue
		}
		foundProvider = true
		if item["latestProbe"].(string) == "" {
			t.Fatal("expected latestProbe for 123_open")
		}
		if item["lastTaskState"].(string) == "" {
			t.Fatal("expected lastTaskState for 123_open")
		}
		summary := item["snapshotSummary"].(map[string]interface{})
		if got := item["protocolGroup"].(string); got != "aliyun_123_open" {
			t.Fatalf("expected provider protocolGroup aliyun_123_open, got %s", got)
		}
		coverage, ok := item["protocolCoverage"].(map[string]interface{})
		if !ok || coverage == nil {
			t.Fatalf("expected protocolCoverage on provider status, got %#v", item)
		}
		summaryCoverage, ok := summary["protocolCoverage"].(map[string]interface{})
		if !ok || summaryCoverage == nil {
			t.Fatalf("expected protocolCoverage in snapshot summary, got %#v", summary)
		}
		if got := summaryCoverage["protocolGroup"].(string); got != "aliyun_123_open" {
			t.Fatalf("expected snapshot protocolCoverage aliyun_123_open, got %s", got)
		}
		if got := summary["executionMode"].(string); got != "pre_scan_flat" {
			t.Fatalf("expected status summary executionMode pre_scan_flat, got %s", got)
		}
		if got := summary["sourceDeletePolicy"].(string); got != "record_only" {
			t.Fatalf("expected status summary sourceDeletePolicy record_only, got %s", got)
		}
		if got := int(summary["riskProfile"].(map[string]interface{})["directoryIntervalMs"].(float64)); got != 2222 {
			t.Fatalf("expected status summary risk directoryIntervalMs 2222, got %d", got)
		}
		if got := int(summary["pendingCount"].(float64)); got != 1 {
			t.Fatalf("expected status summary pendingCount 1, got %d", got)
		}
		if got := int(summary["sourceDeletionCount"].(float64)); got != 1 {
			t.Fatalf("expected status summary sourceDeletionCount 1, got %d", got)
		}
		if got := int(summary["retryableCount"].(float64)); got != 1 {
			t.Fatalf("expected status summary retryableCount 1, got %d", got)
		}
		if got := summary["runtime"].(map[string]interface{})["blockedReason"].(string); got != "retry_queue_requires_local_file_restore" {
			t.Fatalf("expected status summary blockedReason retry_queue_requires_local_file_restore, got %s", got)
		}
		if got := summary["retrySummary"].(map[string]interface{})["blockedAction"].(string); got != "restore_local_source_file" {
			t.Fatalf("expected status summary blockedAction restore_local_source_file, got %s", got)
		}
		if got := int(item["blockedCount"].(float64)); got != 1 {
			t.Fatalf("expected provider blockedCount 1, got %d", got)
		}
		if got := int(item["autoRecoverCount"].(float64)); got != 0 {
			t.Fatalf("expected provider autoRecoverCount 0, got %d", got)
		}
		blockedActions, ok := summary["blockedActions"].([]interface{})
		if !ok || len(blockedActions) == 0 {
			t.Fatalf("expected status summary blockedActions, got %#v", summary["blockedActions"])
		}
		if got := blockedActions[0].(map[string]interface{})["action"].(string); got != "restore_local_source_file" {
			t.Fatalf("expected status summary blockedActions[0] restore_local_source_file, got %s", got)
		}
		if got := len(summary["pendingTree"].([]interface{})); got == 0 {
			t.Fatal("expected pendingTree in status summary")
		}
		if _, ok := summary["runtime"].(map[string]interface{}); !ok {
			t.Fatalf("expected runtime summary in status snapshot, got %#v", summary["runtime"])
		}
	}
	if !foundProvider {
		t.Fatal("expected 123_open in provider statuses")
	}

	reportResp := invokeJSON(t, handler, http.MethodGet, "/api/evidence/report", nil)
	reportData := reportResp.Data.(map[string]interface{})
	if got := reportData["title"].(string); got != "CloudPan Sync Go 验收与样本报告" {
		t.Fatalf("expected default report title, got %s", got)
	}
	if got := reportData["markdown"].(string); !strings.Contains(got, "CloudPan Sync Go 验收与样本报告") {
		t.Fatalf("expected default title in report markdown, got %s", got)
	}
	if got := reportData["markdown"].(string); !strings.Contains(got, "## 代表任务样本") {
		t.Fatalf("expected sample section in report markdown, got %s", got)
	}

	savedReportResp := invokeJSON(t, handler, http.MethodPost, "/api/evidence/report", map[string]interface{}{
		"title": "工作流验收报告",
		"note":  "用于验证报告保存与历史记录",
	})
	savedReport := savedReportResp.Data.(map[string]interface{})
	savedReportID := savedReport["id"].(string)
	if savedReportID == "" {
		t.Fatal("expected saved report id")
	}
	if got := savedReport["title"].(string); got != "工作流验收报告" {
		t.Fatalf("expected saved report title, got %s", got)
	}
	if got := savedReport["note"].(string); got != "用于验证报告保存与历史记录" {
		t.Fatalf("expected saved report note, got %s", got)
	}

	reportListResp := invokeJSON(t, handler, http.MethodGet, "/api/evidence/reports", nil)
	reportList := reportListResp.Data.(map[string]interface{})["items"].([]interface{})
	if len(reportList) == 0 {
		t.Fatal("expected evidence report history items")
	}
	if got := reportList[0].(map[string]interface{})["id"].(string); got != savedReportID {
		t.Fatalf("expected latest saved report id %s, got %s", savedReportID, got)
	}

	reportByIDResp := invokeJSON(t, handler, http.MethodGet, "/api/evidence/reports/"+savedReportID, nil)
	reportByID := reportByIDResp.Data.(map[string]interface{})
	if got := reportByID["title"].(string); got != "工作流验收报告" {
		t.Fatalf("expected saved report title from get-by-id, got %s", got)
	}
	if got := reportByID["markdown"].(string); !strings.Contains(got, "工作流验收报告") {
		t.Fatalf("expected custom title in saved report markdown, got %s", got)
	}
	if got := len(reportByID["smokeMatrix"].([]interface{})); got == 0 {
		t.Fatal("expected saved report to persist smokeMatrix")
	}

	smokeResp := invokeJSON(t, handler, http.MethodPost, "/api/provider-smokes", map[string]interface{}{
		"providerKey":   "123_open",
		"protocolGroup": "aliyun_123_open",
		"authMode":      "manual_token",
		"category":      "browse_only",
		"result":        "success",
		"title":         "工作流真实 smoke",
		"note":          "ValidateAuth/List/Metadata",
		"operations":    []string{"ValidateAuth", "List", "Metadata"},
		"environment": map[string]interface{}{
			"os": "windows",
		},
	})
	smokeData := smokeResp.Data.(map[string]interface{})
	smokeID := smokeData["id"].(string)
	if smokeID == "" {
		t.Fatal("expected smoke record id")
	}
	if got := smokeData["providerKey"].(string); got != "123_open" {
		t.Fatalf("expected smoke providerKey 123_open, got %s", got)
	}
	if got := smokeData["category"].(string); got != "browse_only" {
		t.Fatalf("expected smoke category browse_only, got %s", got)
	}

	smokeListResp := invokeJSON(t, handler, http.MethodGet, "/api/provider-smokes", nil)
	smokeList := smokeListResp.Data.(map[string]interface{})["items"].([]interface{})
	if len(smokeList) == 0 {
		t.Fatal("expected smoke record history items")
	}
	if got := smokeList[0].(map[string]interface{})["id"].(string); got != smokeID {
		t.Fatalf("expected latest smoke id %s, got %s", smokeID, got)
	}

	smokeSummaryResp := invokeJSON(t, handler, http.MethodGet, "/api/provider-smokes/summary", nil)
	smokeSummary := smokeSummaryResp.Data.(map[string]interface{})["items"].([]interface{})
	if len(smokeSummary) == 0 {
		t.Fatal("expected smoke summary items")
	}
	if got := smokeSummary[0].(map[string]interface{})["protocolGroup"].(string); got != "aliyun_123_open" {
		t.Fatalf("expected smoke summary protocolGroup aliyun_123_open, got %s", got)
	}
	if got := smokeSummary[0].(map[string]interface{})["sampleCategory"].(string); got != "browse_only" {
		t.Fatalf("expected smoke summary sampleCategory browse_only, got %s", got)
	}
	if got := smokeSummary[0].(map[string]interface{})["hasRealSuccessSample"].(bool); !got {
		t.Fatal("expected smoke summary hasRealSuccessSample true")
	}

	smokeMatrixResp := invokeJSON(t, handler, http.MethodGet, "/api/provider-smokes/matrix", nil)
	smokeMatrix := smokeMatrixResp.Data.(map[string]interface{})["items"].([]interface{})
	if len(smokeMatrix) == 0 {
		t.Fatal("expected smoke matrix items")
	}
	if got := smokeMatrix[0].(map[string]interface{})["protocolGroup"].(string); got != "aliyun_123_open" {
		t.Fatalf("expected smoke matrix protocolGroup aliyun_123_open, got %s", got)
	}
	if got := smokeMatrix[0].(map[string]interface{})["coverageRealSuccessTaskCount"].(float64); got == 0 {
		t.Fatal("expected smoke matrix coverageRealSuccessTaskCount to be non-zero")
	}
	if got := smokeMatrix[0].(map[string]interface{})["acceptanceStatus"].(string); got != "accepted" {
		t.Fatalf("expected smoke matrix acceptanceStatus accepted, got %s", got)
	}
	if got := smokeMatrix[0].(map[string]interface{})["accepted"].(bool); !got {
		t.Fatal("expected smoke matrix accepted true")
	}
	foundMissing := false
	for _, raw := range smokeMatrix {
		item := raw.(map[string]interface{})
		if item["acceptanceStatus"].(string) == "pending" {
			if got := len(item["acceptanceMissing"].([]interface{})); got == 0 {
				t.Fatal("expected pending smoke matrix row to include missing reasons")
			}
			foundMissing = true
			break
		}
	}
	if !foundMissing {
		t.Fatal("expected at least one pending smoke matrix row")
	}

	smokeMarkdown := invokeText(t, handler, http.MethodGet, "/api/provider-smokes/"+smokeID+"?format=markdown", nil)
	if !strings.Contains(smokeMarkdown, "工作流真实 smoke") {
		t.Fatalf("expected smoke markdown title, got %s", smokeMarkdown)
	}
	if !strings.Contains(reportData["markdown"].(string), "## 真实样本矩阵") {
		t.Fatalf("expected report markdown to include smoke matrix section, got %s", reportData["markdown"].(string))
	}
	if !strings.Contains(reportData["markdown"].(string), "## 真实联调验收") {
		t.Fatalf("expected report markdown to include acceptance section, got %s", reportData["markdown"].(string))
	}
	if !strings.Contains(reportData["markdown"].(string), "Selected Roots") || !strings.Contains(reportData["markdown"].(string), "Scan Trace") {
		t.Fatalf("expected report markdown to include scan trace context, got %s", reportData["markdown"].(string))
	}
	if !strings.Contains(reportData["markdown"].(string), "Missing") {
		t.Fatalf("expected report markdown to include missing reasons column, got %s", reportData["markdown"].(string))
	}
	if !strings.Contains(reportData["markdown"].(string), "已验收协议组") || !strings.Contains(reportData["markdown"].(string), "进行中协议组") || !strings.Contains(reportData["markdown"].(string), "待补齐协议组") {
		t.Fatalf("expected report markdown to include acceptance counters, got %s", reportData["markdown"].(string))
	}

	retryResp := invokeJSON(t, handler, http.MethodPost, "/api/tasks/"+taskID+"/retry", map[string]interface{}{
		"paths": []string{"/demo/pending.bin"},
		"scope": "selected_pending_subset",
	})
	retryData := retryResp.Data.(map[string]interface{})
	if got := retryData["task"].(map[string]interface{})["state"].(string); got != "ready" {
		t.Fatalf("expected ready after retry, got %s", got)
	}
	retriedPlan := retryData["plan"].(map[string]interface{})
	if got := len(retriedPlan["items"].([]interface{})); got != 1 {
		t.Fatalf("expected retry to keep only 1 pending item, got %d", got)
	}
	retryMetadata := retriedPlan["metadata"].(map[string]interface{})
	if retryPendingOnly, _ := retryMetadata["retryPendingOnly"].(bool); !retryPendingOnly {
		t.Fatalf("expected retryPendingOnly metadata true, got %#v", retryMetadata["retryPendingOnly"])
	}
	if retryMode, _ := retryMetadata["retryMode"].(string); retryMode != "selected_pending_subset" {
		t.Fatalf("expected retryMode selected_pending_subset, got %s", retryMode)
	}
	selectedPaths, ok := retryMetadata["retrySelectedPaths"].([]interface{})
	if !ok || len(selectedPaths) != 1 || selectedPaths[0].(string) != "/demo/pending.bin" {
		t.Fatalf("expected retrySelectedPaths [/demo/pending.bin], got %#v", retryMetadata["retrySelectedPaths"])
	}
}

func TestAppPlanPreviewRejectsInvalidSourceDeletePolicy(t *testing.T) {
	ctx := context.Background()
	application := mustNewTestApp(t, ctx)
	handler := application.routes()

	envelope, statusCode := invokeJSONError(t, handler, http.MethodPost, "/api/plans/preview", map[string]interface{}{
		"sourceProvider":     "guangya",
		"targetProvider":     "123_open",
		"sourceDeletePolicy": "delete_target",
		"entries": []map[string]interface{}{
			{"path": "/demo/a.bin", "size": 128, "md5": "md5-a"},
		},
	})
	if statusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", statusCode)
	}
	if got := envelope.Error.Code; got != "invalid_source_delete_policy" {
		t.Fatalf("expected invalid_source_delete_policy, got %s", got)
	}
}

func TestAppRetrySelectedDirectorySubsetKeepsChosenSubtree(t *testing.T) {
	ctx := context.Background()
	application := mustNewTestApp(t, ctx)
	targetAdapter := &appScriptedAdapter{
		meta: provider.Provider{
			Key:           "guangya",
			DisplayName:   "Retry Directory Target",
			ProtocolGroup: "fake_target",
			AuthModes:     []string{"manual_token"},
			Status:        "planned",
		},
		capability: provider.CapabilitySet{
			SupportsAuthValidation: true,
			SupportsUpload:         true,
			SupportsFastUpload:     true,
		},
		uploadFunc: func(req provider.UploadRequest) provider.UploadResult {
			return provider.UploadResult{
				OperationResult: provider.OperationResult{
					OK:      true,
					Status:  "ok",
					Message: "selected directory ok",
					Mode:    "fake_selected_directory_ok",
				},
			}
		},
	}
	adapters := provider.DefaultCatalog()
	adapters = append(adapters, targetAdapter)
	registry := provider.NewRegistry(adapters...)
	authSvc := auth.NewService(application.store, registry)
	taskSvc := task.NewService(application.store, registry, authSvc)
	application.providers = registry
	application.auth = authSvc
	application.tasks = taskSvc
	handler := application.routes()

	profileResp := invokeJSON(t, handler, http.MethodPost, "/api/auth/profiles", map[string]interface{}{
		"providerKey": "guangya",
		"authMode":    "manual_token",
		"displayName": "Retry Directory Target",
		"token":       "token-retry-directory",
	})
	profileID := profileResp.Data.(map[string]interface{})["id"].(string)

	taskResp := invokeJSON(t, handler, http.MethodPost, "/api/tasks", map[string]interface{}{
		"sourceProvider":  "baidu_netdisk",
		"targetProvider":  "guangya",
		"targetProfileId": profileID,
		"thresholdMB":     1,
		"selectedRoots":   []string{"/1", "/2"},
		"entries": []map[string]interface{}{
			{"path": "/1/11/a.bin", "size": 512},
			{"path": "/1/12/b.bin", "size": 768},
			{"path": "/2/21/c.bin", "size": 1024},
		},
	})
	taskID := taskResp.Data.(map[string]interface{})["task"].(map[string]interface{})["id"].(string)

	runResp := invokeJSON(t, handler, http.MethodPost, "/api/tasks/"+taskID+"/run", nil)
	if got := runResp.Data.(map[string]interface{})["task"].(map[string]interface{})["state"].(string); got != "completed" && got != "completed_with_errors" {
		t.Fatalf("expected completed or completed_with_errors, got %s", got)
	}

	retryResp := invokeJSON(t, handler, http.MethodPost, "/api/tasks/"+taskID+"/retry", map[string]interface{}{
		"paths": []string{"/1/11"},
		"scope": "selected_directory_subset",
	})
	retryData := retryResp.Data.(map[string]interface{})
	if got := retryData["task"].(map[string]interface{})["state"].(string); got != "ready" {
		t.Fatalf("expected ready after directory retry, got %s", got)
	}
	retriedPlan := retryData["plan"].(map[string]interface{})
	if got := len(retriedPlan["items"].([]interface{})); got != 1 {
		t.Fatalf("expected retry to keep only 1 directory item, got %d", got)
	}
	if got := retriedPlan["items"].([]interface{})[0].(map[string]interface{})["path"].(string); got != "/1/11/a.bin" {
		t.Fatalf("expected selected directory item /1/11/a.bin, got %s", got)
	}
	retryMetadata := retriedPlan["metadata"].(map[string]interface{})
	if retryMode, _ := retryMetadata["retryMode"].(string); retryMode != "selected_directory_subset" {
		t.Fatalf("expected retryMode selected_directory_subset, got %s", retryMode)
	}
	if retryPendingOnly, _ := retryMetadata["retryPendingOnly"].(bool); retryPendingOnly {
		t.Fatalf("expected retryPendingOnly false, got %#v", retryMetadata["retryPendingOnly"])
	}
	selectedPaths, ok := retryMetadata["retrySelectedPaths"].([]interface{})
	if !ok || len(selectedPaths) != 1 || selectedPaths[0].(string) != "/1/11" {
		t.Fatalf("expected retrySelectedPaths [/1/11], got %#v", retryMetadata["retrySelectedPaths"])
	}
}

func TestAppProviderUploadEndpoint(t *testing.T) {
	ctx := context.Background()
	application := mustNewTestApp(t, ctx)
	handler := application.routes()
	providerServer, _ := newAppPan123OpenTestServer(t)
	t.Cleanup(providerServer.Close)

	profileResp := invokeJSON(t, handler, http.MethodPost, "/api/auth/profiles", map[string]interface{}{
		"providerKey": "123_open",
		"authMode":    "manual_token",
		"displayName": "Upload 123",
		"token":       "token-upload",
		"extra": map[string]interface{}{
			"apiEndpoint": providerServer.URL,
		},
	})
	profileID := profileResp.Data.(map[string]interface{})["id"].(string)
	if profileID == "" {
		t.Fatal("expected profile id")
	}

	localFile := filepath.Join(t.TempDir(), "provider-upload.bin")
	if err := os.WriteFile(localFile, []byte("provider-upload"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	uploadResp := invokeJSON(t, handler, http.MethodPost, "/api/providers/123_open/upload", map[string]interface{}{
		"profileId":      profileID,
		"path":           "/demo/provider-upload.bin",
		"name":           "provider-upload.bin",
		"size":           15,
		"localPath":      localFile,
		"conflictPolicy": "auto_rename_new",
		"strategy":       "download_upload",
		"md5":            "md5-upload",
	})
	if !uploadResp.OK {
		t.Fatal("expected provider upload response ok")
	}
	data := uploadResp.Data.(map[string]interface{})
	if got := data["status"].(string); got != "ok" {
		t.Fatalf("expected provider upload status ok, got %s", got)
	}
	if got := data["mode"].(string); got != "open_family_real_upload" {
		t.Fatalf("expected provider upload mode open_family_real_upload, got %s", got)
	}
}

func TestAppRetryBlockedReturnsStructuredError(t *testing.T) {
	ctx := context.Background()
	application := mustNewTestApp(t, ctx)
	handler := application.routes()

	profileResp := invokeJSON(t, handler, http.MethodPost, "/api/auth/profiles", map[string]interface{}{
		"providerKey": "guangya",
		"authMode":    "manual_token",
		"displayName": "Retry Blocked Guangya",
		"token":       "token-retry-blocked",
	})
	profileID := profileResp.Data.(map[string]interface{})["id"].(string)

	taskResp := invokeJSON(t, handler, http.MethodPost, "/api/tasks", map[string]interface{}{
		"sourceProvider":  "baidu_netdisk",
		"targetProvider":  "guangya",
		"targetProfileId": profileID,
		"thresholdMB":     1,
		"entries": []map[string]interface{}{
			{"path": "/only-missing.bin", "size": 512},
		},
	})
	taskID := taskResp.Data.(map[string]interface{})["task"].(map[string]interface{})["id"].(string)

	runResp := invokeJSON(t, handler, http.MethodPost, "/api/tasks/"+taskID+"/run", nil)
	if got := runResp.Data.(map[string]interface{})["task"].(map[string]interface{})["state"].(string); got != "blocked" {
		t.Fatalf("expected blocked, got %s", got)
	}

	errorResp, status := invokeJSONError(t, handler, http.MethodPost, "/api/tasks/"+taskID+"/retry", nil)
	if status != http.StatusBadRequest {
		t.Fatalf("expected bad request, got %d", status)
	}
	if errorResp.Error == nil || errorResp.Error.Code != "retry_blocked" {
		t.Fatalf("expected retry_blocked error, got %#v", errorResp.Error)
	}
}

func TestAppRetrySelectionEmptyReturnsStructuredError(t *testing.T) {
	ctx := context.Background()
	application := mustNewTestApp(t, ctx)
	sourceAdapter := &appScriptedAdapter{
		meta: provider.Provider{
			Key:           "retry_selection_source",
			DisplayName:   "Retry Selection Source",
			ProtocolGroup: "fake_source",
			AuthModes:     []string{"manual_token"},
			Status:        "planned",
		},
		capability: provider.CapabilitySet{
			SupportsAuthValidation: true,
		},
	}
	targetAdapter := &appScriptedAdapter{
		meta: provider.Provider{
			Key:              "retry_selection_target",
			DisplayName:      "Retry Selection Target",
			ProtocolGroup:    "fake_target",
			AuthModes:        []string{"manual_token"},
			FastUploadInputs: []string{"md5", "size"},
			FallbackModes:    []string{"download_upload"},
			Status:           "planned",
		},
		capability: provider.CapabilitySet{
			SupportsAuthValidation: true,
			SupportsUpload:         true,
		},
		uploadFunc: func(req provider.UploadRequest) provider.UploadResult {
			return provider.UploadResult{
				OperationResult: provider.OperationResult{
					Status:  "remote_error",
					Message: "retry later",
					Mode:    "fake_remote_error",
				},
			}
		},
	}
	registry := provider.NewRegistry(sourceAdapter, targetAdapter)
	authSvc := auth.NewService(application.store, registry)
	taskSvc := task.NewService(application.store, registry, authSvc)
	application.providers = registry
	application.auth = authSvc
	application.tasks = taskSvc

	handler := application.routes()

	profileResp := invokeJSON(t, handler, http.MethodPost, "/api/auth/profiles", map[string]interface{}{
		"providerKey": "retry_selection_target",
		"authMode":    "manual_token",
		"displayName": "Retry Selection Target",
		"token":       "token-retry-selection",
	})
	profileID := profileResp.Data.(map[string]interface{})["id"].(string)

	localFile := filepath.Join(t.TempDir(), "retry-selection.bin")
	if err := os.WriteFile(localFile, []byte("retry-selection"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	taskResp := invokeJSON(t, handler, http.MethodPost, "/api/tasks", map[string]interface{}{
		"sourceProvider":  "retry_selection_source",
		"targetProvider":  "retry_selection_target",
		"targetProfileId": profileID,
		"thresholdMB":     1,
		"entries": []map[string]interface{}{
			{"path": "/retry.bin", "size": 1024, "md5": "retry-md5", "localPath": localFile},
		},
	})
	taskID := taskResp.Data.(map[string]interface{})["task"].(map[string]interface{})["id"].(string)

	runResp := invokeJSON(t, handler, http.MethodPost, "/api/tasks/"+taskID+"/run", nil)
	if got := runResp.Data.(map[string]interface{})["task"].(map[string]interface{})["state"].(string); got != "completed_with_errors" {
		t.Fatalf("expected completed_with_errors, got %s", got)
	}

	errorResp, status := invokeJSONError(t, handler, http.MethodPost, "/api/tasks/"+taskID+"/retry", map[string]interface{}{
		"paths": []string{"/not-found.bin"},
		"scope": "selected_pending_subset",
	})
	if status != http.StatusBadRequest {
		t.Fatalf("expected bad request, got %d", status)
	}
	if errorResp.Error == nil || errorResp.Error.Code != "retry_selection_empty" {
		t.Fatalf("expected retry_selection_empty error, got %#v", errorResp.Error)
	}
}

func TestAppRecoverTasksEndpointReturnsSummary(t *testing.T) {
	ctx := context.Background()
	application := mustNewTestApp(t, ctx)
	sourceAdapter := &appScriptedAdapter{
		meta: provider.Provider{
			Key:           "recover_api_source",
			DisplayName:   "Recover API Source",
			ProtocolGroup: "fake_source",
			AuthModes:     []string{"manual_token"},
			Status:        "planned",
		},
		capability: provider.CapabilitySet{
			SupportsAuthValidation: true,
		},
	}
	uploadCalls := 0
	targetAdapter := &appScriptedAdapter{
		meta: provider.Provider{
			Key:              "recover_api_target",
			DisplayName:      "Recover API Target",
			ProtocolGroup:    "fake_target",
			AuthModes:        []string{"manual_token"},
			FastUploadInputs: []string{"md5", "size"},
			FallbackModes:    []string{"download_upload"},
			Status:           "planned",
		},
		capability: provider.CapabilitySet{
			SupportsAuthValidation: true,
			SupportsUpload:         true,
		},
		uploadFunc: func(req provider.UploadRequest) provider.UploadResult {
			uploadCalls++
			if req.ResumeUpload != nil {
				return provider.UploadResult{
					OperationResult: provider.OperationResult{
						OK:      true,
						Status:  "ok",
						Message: "resumed",
						Mode:    "scripted_resume_ok",
						Payload: map[string]interface{}{
							"fileId":   req.ResumeUpload.FileID,
							"uploadId": req.ResumeUpload.UploadID,
						},
					},
				}
			}
			return provider.UploadResult{
				OperationResult: provider.OperationResult{
					Status:  "provider_request_failed",
					Message: "resume later",
					Mode:    "scripted_resume_later",
					Payload: map[string]interface{}{
						"fileId":           "recover-api-file",
						"uploadId":         "recover-api-upload",
						"nextPartNumber":   2,
						"failedPartNumber": 2,
					},
				},
			}
		},
	}
	registry := provider.NewRegistry(sourceAdapter, targetAdapter)
	authSvc := auth.NewService(application.store, registry)
	taskSvc := task.NewService(application.store, registry, authSvc)
	application.providers = registry
	application.auth = authSvc
	application.tasks = taskSvc

	handler := application.routes()

	profileResp := invokeJSON(t, handler, http.MethodPost, "/api/auth/profiles", map[string]interface{}{
		"providerKey": "recover_api_target",
		"authMode":    "manual_token",
		"displayName": "Recover API Target",
		"token":       "token-recover-api",
	})
	profileID := profileResp.Data.(map[string]interface{})["id"].(string)

	localFile := filepath.Join(t.TempDir(), "recover-api.bin")
	if err := os.WriteFile(localFile, []byte("recover-api"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	taskResp := invokeJSON(t, handler, http.MethodPost, "/api/tasks", map[string]interface{}{
		"sourceProvider":  "recover_api_source",
		"targetProvider":  "recover_api_target",
		"targetProfileId": profileID,
		"thresholdMB":     1,
		"entries": []map[string]interface{}{
			{"path": "/recover.bin", "size": 1024, "md5": "recover-md5", "localPath": localFile},
		},
	})
	taskID := taskResp.Data.(map[string]interface{})["task"].(map[string]interface{})["id"].(string)

	runResp := invokeJSON(t, handler, http.MethodPost, "/api/tasks/"+taskID+"/run", nil)
	if got := runResp.Data.(map[string]interface{})["task"].(map[string]interface{})["state"].(string); got != "completed_with_errors" {
		t.Fatalf("expected completed_with_errors, got %s", got)
	}

	recoverResp := invokeJSON(t, handler, http.MethodPost, "/api/tasks/recover", map[string]interface{}{
		"mode":          "upload_checkpoint_auto_resume",
		"providerKey":   "recover_api_target",
		"blockedAction": "",
		"limit":         1,
	})
	recoverData := recoverResp.Data.(map[string]interface{})
	if got := int(recoverData["matchedCount"].(float64)); got != 1 {
		t.Fatalf("expected matchedCount 1, got %d", got)
	}
	if got := int(recoverData["recoveredCount"].(float64)); got != 1 {
		t.Fatalf("expected recoveredCount 1, got %d", got)
	}
	if got := recoverData["mode"].(string); got != "upload_checkpoint_auto_resume" {
		t.Fatalf("expected mode upload_checkpoint_auto_resume, got %s", got)
	}
	if raw, ok := recoverData["blockedAction"]; ok {
		if got, _ := raw.(string); got != "" {
			t.Fatalf("expected blockedAction empty, got %s", got)
		}
	}

	detailResp := invokeJSON(t, handler, http.MethodGet, "/api/tasks/"+taskID, nil)
	if got := detailResp.Data.(map[string]interface{})["task"].(map[string]interface{})["state"].(string); got != "completed" {
		t.Fatalf("expected completed after recover endpoint, got %s", got)
	}
	if uploadCalls != 2 {
		t.Fatalf("expected 2 upload calls, got %d", uploadCalls)
	}
}

func TestAppRecoverTasksEndpointReportsProviderBudgetSkips(t *testing.T) {
	ctx := context.Background()
	application := mustNewTestApp(t, ctx)

	uploadCallsByPath := map[string]int{}
	targetAdapter := &appScriptedAdapter{
		meta: provider.Provider{
			Key:              "recover_budget_api_target",
			DisplayName:      "Recover Budget API Target",
			ProtocolGroup:    "fake_target",
			AuthModes:        []string{"manual_token"},
			FastUploadInputs: []string{"md5", "size"},
			FallbackModes:    []string{"download_upload"},
			Status:           "planned",
		},
		capability: provider.CapabilitySet{
			SupportsAuthValidation: true,
			SupportsUpload:         true,
		},
		uploadFunc: func(req provider.UploadRequest) provider.UploadResult {
			uploadCallsByPath[req.Path]++
			if req.ResumeUpload != nil {
				return provider.UploadResult{
					OperationResult: provider.OperationResult{
						OK:      true,
						Status:  "ok",
						Message: "budget recovered",
						Mode:    "scripted_budget_ok",
						Payload: map[string]interface{}{
							"fileId":   req.ResumeUpload.FileID,
							"uploadId": req.ResumeUpload.UploadID,
						},
					},
				}
			}
			if uploadCallsByPath[req.Path] == 1 {
				return provider.UploadResult{
					OperationResult: provider.OperationResult{
						Status:  "provider_request_failed",
						Message: "resume later",
						Mode:    "scripted_resume_later",
						Payload: map[string]interface{}{
							"fileId":           "budget-file-" + strings.TrimPrefix(strings.ReplaceAll(req.Path, "/", "-"), "-"),
							"uploadId":         "budget-upload-" + strings.TrimPrefix(strings.ReplaceAll(req.Path, "/", "-"), "-"),
							"nextPartNumber":   2,
							"failedPartNumber": 2,
						},
					},
				}
			}
			t.Fatalf("unexpected non-resume upload for %s", req.Path)
			return provider.UploadResult{}
		},
	}

	registry := provider.NewRegistry(targetAdapter)
	authSvc := auth.NewService(application.store, registry)
	taskSvc := task.NewService(application.store, registry, authSvc)
	application.providers = registry
	application.auth = authSvc
	application.tasks = taskSvc
	handler := application.routes()

	profileResp := invokeJSON(t, handler, http.MethodPost, "/api/auth/profiles", map[string]interface{}{
		"providerKey": "recover_budget_api_target",
		"authMode":    "manual_token",
		"displayName": "Recover Budget API Target",
		"token":       "token-recover-budget-api",
	})
	profileID := profileResp.Data.(map[string]interface{})["id"].(string)

	createBlockedTask := func(path string) string {
		fileName := strings.TrimPrefix(strings.ReplaceAll(path, "/", "_"), "_")
		localFile := filepath.Join(t.TempDir(), fileName)
		if err := os.WriteFile(localFile, []byte(path), 0o644); err != nil {
			t.Fatalf("WriteFile(%s) error = %v", path, err)
		}
		taskResp := invokeJSON(t, handler, http.MethodPost, "/api/tasks", map[string]interface{}{
			"sourceProvider":  "recover_budget_api_source",
			"targetProvider":  "recover_budget_api_target",
			"targetProfileId": profileID,
			"thresholdMB":     1,
			"riskOverride": map[string]interface{}{
				"maxConcurrent": 1,
			},
			"entries": []map[string]interface{}{
				{"path": path, "size": 1024, "md5": "md5-" + fileName, "localPath": localFile},
			},
		})
		taskID := taskResp.Data.(map[string]interface{})["task"].(map[string]interface{})["id"].(string)
		runResp := invokeJSON(t, handler, http.MethodPost, "/api/tasks/"+taskID+"/run", nil)
		if got := runResp.Data.(map[string]interface{})["task"].(map[string]interface{})["state"].(string); got != "completed_with_errors" {
			t.Fatalf("expected completed_with_errors, got %s", got)
		}
		return taskID
	}

	firstID := createBlockedTask("/budget-a.bin")
	secondID := createBlockedTask("/budget-b.bin")

	recoverResp := invokeJSON(t, handler, http.MethodPost, "/api/tasks/recover", map[string]interface{}{
		"providerKey": "recover_budget_api_target",
		"limit":       5,
	})
	recoverData := recoverResp.Data.(map[string]interface{})
	if got := int(recoverData["matchedCount"].(float64)); got != 2 {
		t.Fatalf("expected matchedCount 2, got %d", got)
	}
	if got := int(recoverData["recoveredCount"].(float64)); got != 1 {
		t.Fatalf("expected recoveredCount 1, got %d", got)
	}
	if got := int(recoverData["skippedByProviderBudget"].(float64)); got != 1 {
		t.Fatalf("expected skippedByProviderBudget 1, got %d", got)
	}

	firstDetail := invokeJSON(t, handler, http.MethodGet, "/api/tasks/"+firstID, nil)
	secondDetail := invokeJSON(t, handler, http.MethodGet, "/api/tasks/"+secondID, nil)
	states := []string{
		firstDetail.Data.(map[string]interface{})["task"].(map[string]interface{})["state"].(string),
		secondDetail.Data.(map[string]interface{})["task"].(map[string]interface{})["state"].(string),
	}
	completed := 0
	pendingRecover := 0
	for _, state := range states {
		if state == "completed" {
			completed++
		}
		if state == "completed_with_errors" {
			pendingRecover++
		}
	}
	if completed != 1 || pendingRecover != 1 {
		t.Fatalf("expected one completed and one blocked task, got %#v", states)
	}
}

func TestAppRecoverTasksEndpointDryRunDoesNotMutateTask(t *testing.T) {
	ctx := context.Background()
	store, err := sqlitestore.New(ctx, filepath.Join(t.TempDir(), "recover-dry-run-api.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer func() { _ = store.Close() }()

	uploadCalls := 0
	targetAdapter := &appScriptedAdapter{
		meta: provider.Provider{
			Key:              "recover_dry_run_api_target",
			DisplayName:      "Recover Dry Run API Target",
			ProtocolGroup:    "recover_dry_run_api_group",
			AuthModes:        []string{"manual_token"},
			FastUploadInputs: []string{"md5", "size"},
			FallbackModes:    []string{"download_upload"},
			Status:           "planned",
		},
		capability: provider.CapabilitySet{
			SupportsAuthValidation: true,
			SupportsFastUpload:     true,
			SupportsUpload:         true,
		},
		uploadFunc: func(req provider.UploadRequest) provider.UploadResult {
			uploadCalls++
			if uploadCalls == 1 {
				return provider.UploadResult{
					OperationResult: provider.OperationResult{
						Status:  "upload_checkpoint_pending",
						Message: "checkpoint pending",
						Mode:    "recover_dry_run_api_pending",
						Payload: map[string]interface{}{
							"fileId":         "recover-dry-run-api-file",
							"uploadId":       "recover-dry-run-api-upload",
							"nextPartNumber": 1,
						},
					},
				}
			}
			return provider.UploadResult{
				OperationResult: provider.OperationResult{
					OK:      true,
					Status:  "ok",
					Message: "recovered",
					Mode:    "recover_dry_run_api_ok",
				},
			}
		},
	}

	registry := provider.NewRegistry(targetAdapter)
	authSvc := auth.NewService(store, registry)
	tasks := task.NewService(store, registry, authSvc)
	application := &App{
		logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
		store:     store,
		providers: registry,
		auth:      authSvc,
		tasks:     tasks,
		webIndex:  []byte("<html><body>ok</body></html>"),
		webStatic: http.NewServeMux(),
	}
	handler := application.routes()

	profileResp := invokeJSON(t, handler, http.MethodPost, "/api/auth/profiles", map[string]interface{}{
		"providerKey": "recover_dry_run_api_target",
		"authMode":    "manual_token",
		"displayName": "Recover Dry Run API Target",
		"token":       "token-recover-dry-run-api",
	})
	profileID := profileResp.Data.(map[string]interface{})["id"].(string)

	localFile := filepath.Join(t.TempDir(), "recover-dry-run-api.bin")
	if err := os.WriteFile(localFile, []byte("recover-dry-run-api"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	taskResp := invokeJSON(t, handler, http.MethodPost, "/api/tasks", map[string]interface{}{
		"sourceProvider":  "recover_dry_run_api_target",
		"targetProvider":  "recover_dry_run_api_target",
		"targetProfileId": profileID,
		"thresholdMB":     1,
		"entries": []map[string]interface{}{
			{"path": "/recover-dry-run-api.bin", "size": 1024, "md5": "recover-dry-run-api-md5", "localPath": localFile},
		},
	})
	taskID := taskResp.Data.(map[string]interface{})["task"].(map[string]interface{})["id"].(string)

	runResp := invokeJSON(t, handler, http.MethodPost, "/api/tasks/"+taskID+"/run", nil)
	if got := runResp.Data.(map[string]interface{})["task"].(map[string]interface{})["state"].(string); got != "completed_with_errors" {
		t.Fatalf("expected completed_with_errors, got %s", got)
	}
	if uploadCalls != 1 {
		t.Fatalf("expected initial upload calls 1, got %d", uploadCalls)
	}

	previewResp := invokeJSON(t, handler, http.MethodPost, "/api/tasks/recover", map[string]interface{}{
		"mode":        "upload_checkpoint_auto_resume",
		"providerKey": "recover_dry_run_api_target",
		"limit":       1,
		"dryRun":      true,
	})
	previewData := previewResp.Data.(map[string]interface{})
	if got, _ := previewData["dryRun"].(bool); !got {
		t.Fatalf("expected dryRun true, got %#v", previewData["dryRun"])
	}
	if decisions, ok := previewData["decisions"].([]interface{}); !ok || len(decisions) == 0 {
		t.Fatalf("expected preview decisions, got %#v", previewData["decisions"])
	} else if got := decisions[0].(map[string]interface{})["outcome"].(string); got != "dry_run_recoverable" {
		t.Fatalf("expected preview outcome dry_run_recoverable, got %s", got)
	}
	if got := int(previewData["matchedCount"].(float64)); got != 1 {
		t.Fatalf("expected matchedCount 1, got %d", got)
	}
	if got := int(previewData["recoveredCount"].(float64)); got != 1 {
		t.Fatalf("expected recoveredCount 1 for preview, got %d", got)
	}
	if uploadCalls != 1 {
		t.Fatalf("expected preview not to trigger extra upload, got %d", uploadCalls)
	}

	detailAfterPreview := invokeJSON(t, handler, http.MethodGet, "/api/tasks/"+taskID, nil)
	if got := detailAfterPreview.Data.(map[string]interface{})["task"].(map[string]interface{})["state"].(string); got != "completed_with_errors" {
		t.Fatalf("expected state unchanged after preview, got %s", got)
	}

	recoverResp := invokeJSON(t, handler, http.MethodPost, "/api/tasks/recover", map[string]interface{}{
		"mode":        "upload_checkpoint_auto_resume",
		"providerKey": "recover_dry_run_api_target",
		"limit":       1,
	})
	recoverData := recoverResp.Data.(map[string]interface{})
	if got, _ := recoverData["dryRun"].(bool); got {
		t.Fatalf("expected dryRun false for execute, got %#v", recoverData["dryRun"])
	}
	if decisions, ok := recoverData["decisions"].([]interface{}); !ok || len(decisions) == 0 {
		t.Fatalf("expected execute decisions, got %#v", recoverData["decisions"])
	} else if got := decisions[0].(map[string]interface{})["outcome"].(string); got != "recovered" {
		t.Fatalf("expected execute outcome recovered, got %s", got)
	}
	if got := int(recoverData["recoveredCount"].(float64)); got != 1 {
		t.Fatalf("expected recoveredCount 1, got %d", got)
	}
	if uploadCalls != 2 {
		t.Fatalf("expected execute to trigger second upload, got %d", uploadCalls)
	}

	detailResp := invokeJSON(t, handler, http.MethodGet, "/api/tasks/"+taskID, nil)
	if got := detailResp.Data.(map[string]interface{})["task"].(map[string]interface{})["state"].(string); got != "completed" {
		t.Fatalf("expected completed after execute, got %s", got)
	}
}

func TestAppRecoverTasksEndpointSupportsMultiplePaths(t *testing.T) {
	ctx := context.Background()
	application := mustNewTestApp(t, ctx)

	uploadCallsByPath := map[string]int{}
	targetAdapter := &appScriptedAdapter{
		meta: provider.Provider{
			Key:              "recover_paths_api_target",
			DisplayName:      "Recover Paths API Target",
			ProtocolGroup:    "fake_target",
			AuthModes:        []string{"manual_token"},
			FastUploadInputs: []string{"md5", "size"},
			FallbackModes:    []string{"download_upload"},
			Status:           "planned",
		},
		capability: provider.CapabilitySet{
			SupportsAuthValidation: true,
			SupportsUpload:         true,
		},
		uploadFunc: func(req provider.UploadRequest) provider.UploadResult {
			uploadCallsByPath[req.Path]++
			if uploadCallsByPath[req.Path] == 1 {
				return provider.UploadResult{
					OperationResult: provider.OperationResult{
						Status:  "rate_limited",
						Message: "rate limited",
						Mode:    "scripted_rate_limit",
					},
				}
			}
			return provider.UploadResult{
				OperationResult: provider.OperationResult{
					OK:      true,
					Status:  "ok",
					Message: "paths recovered",
					Mode:    "scripted_paths_ok",
				},
			}
		},
	}

	registry := provider.NewRegistry(targetAdapter)
	authSvc := auth.NewService(application.store, registry)
	taskSvc := task.NewService(application.store, registry, authSvc)
	application.providers = registry
	application.auth = authSvc
	application.tasks = taskSvc
	handler := application.routes()

	profileResp := invokeJSON(t, handler, http.MethodPost, "/api/auth/profiles", map[string]interface{}{
		"providerKey": "recover_paths_api_target",
		"authMode":    "manual_token",
		"displayName": "Recover Paths API Target",
		"token":       "token-recover-paths-api",
	})
	profileID := profileResp.Data.(map[string]interface{})["id"].(string)

	taskResp := invokeJSON(t, handler, http.MethodPost, "/api/tasks", map[string]interface{}{
		"sourceProvider":  "recover_paths_api_source",
		"targetProvider":  "recover_paths_api_target",
		"targetProfileId": profileID,
		"thresholdMB":     1,
		"entries": []map[string]interface{}{
			{"path": "/leaf-a/one.bin", "size": 101, "md5": "leaf-a"},
			{"path": "/leaf-b/two.bin", "size": 202, "md5": "leaf-b"},
			{"path": "/leaf-c/three.bin", "size": 303, "md5": "leaf-c"},
		},
	})
	taskID := taskResp.Data.(map[string]interface{})["task"].(map[string]interface{})["id"].(string)

	runResp := invokeJSON(t, handler, http.MethodPost, "/api/tasks/"+taskID+"/run", nil)
	if got := runResp.Data.(map[string]interface{})["task"].(map[string]interface{})["state"].(string); got != "blocked" {
		t.Fatalf("expected blocked, got %s", got)
	}
	if _, err := application.store.DB().ExecContext(ctx, `UPDATE task_results SET created_at = ? WHERE task_id = ?`,
		time.Now().Add(-2*time.Hour).UTC().Format(time.RFC3339), taskID,
	); err != nil {
		t.Fatalf("update task_results created_at error = %v", err)
	}

	recoverResp := invokeJSON(t, handler, http.MethodPost, "/api/tasks/recover", map[string]interface{}{
		"providerKey": "recover_paths_api_target",
		"paths":       []string{"/leaf-a", "/leaf-c"},
		"scope":       "selected_retry_subset",
		"limit":       1,
	})
	recoverData := recoverResp.Data.(map[string]interface{})
	if got := int(recoverData["matchedCount"].(float64)); got != 1 {
		t.Fatalf("expected matchedCount 1, got %d", got)
	}
	if got := int(recoverData["recoveredCount"].(float64)); got != 1 {
		t.Fatalf("expected recoveredCount 1, got %d", got)
	}

	detailResp := invokeJSON(t, handler, http.MethodGet, "/api/tasks/"+taskID, nil)
	results := detailResp.Data.(map[string]interface{})["results"].([]interface{})
	if len(results) != 2 {
		t.Fatalf("expected 2 recovered results, got %#v", results)
	}
	paths := []string{
		results[0].(map[string]interface{})["payload"].(map[string]interface{})["path"].(string),
		results[1].(map[string]interface{})["payload"].(map[string]interface{})["path"].(string),
	}
	sort.Strings(paths)
	if paths[0] != "/leaf-a/one.bin" || paths[1] != "/leaf-c/three.bin" {
		t.Fatalf("expected recovered paths [/leaf-a/one.bin /leaf-c/three.bin], got %#v", paths)
	}
	if got := uploadCallsByPath["/leaf-b/two.bin"]; got != 1 {
		t.Fatalf("expected /leaf-b/two.bin upload calls to remain 1, got %d", got)
	}
}

func TestAppRecoverTasksEndpointFiltersTaskID(t *testing.T) {
	ctx := context.Background()
	application := mustNewTestApp(t, ctx)

	uploadCallsByPath := map[string]int{}
	targetAdapter := &appScriptedAdapter{
		meta: provider.Provider{
			Key:              "recover_task_id_api_target",
			DisplayName:      "Recover Task ID API Target",
			ProtocolGroup:    "fake_target",
			AuthModes:        []string{"manual_token"},
			FastUploadInputs: []string{"md5", "size"},
			FallbackModes:    []string{"download_upload"},
			Status:           "planned",
		},
		capability: provider.CapabilitySet{
			SupportsAuthValidation: true,
			SupportsUpload:         true,
		},
		uploadFunc: func(req provider.UploadRequest) provider.UploadResult {
			uploadCallsByPath[req.Path]++
			if uploadCallsByPath[req.Path] == 1 {
				return provider.UploadResult{
					OperationResult: provider.OperationResult{
						Status:  "rate_limited",
						Message: "rate limited",
						Mode:    "scripted_rate_limit",
					},
				}
			}
			return provider.UploadResult{
				OperationResult: provider.OperationResult{
					OK:      true,
					Status:  "ok",
					Message: "task filtered recovered",
					Mode:    "scripted_task_filter_ok",
				},
			}
		},
	}

	registry := provider.NewRegistry(targetAdapter)
	authSvc := auth.NewService(application.store, registry)
	taskSvc := task.NewService(application.store, registry, authSvc)
	application.providers = registry
	application.auth = authSvc
	application.tasks = taskSvc
	handler := application.routes()

	profileResp := invokeJSON(t, handler, http.MethodPost, "/api/auth/profiles", map[string]interface{}{
		"providerKey": "recover_task_id_api_target",
		"authMode":    "manual_token",
		"displayName": "Recover Task ID API Target",
		"token":       "token-recover-task-id-api",
	})
	profileID := profileResp.Data.(map[string]interface{})["id"].(string)

	createBlockedTask := func(path string) string {
		taskResp := invokeJSON(t, handler, http.MethodPost, "/api/tasks", map[string]interface{}{
			"sourceProvider":  "recover_task_id_api_source",
			"targetProvider":  "recover_task_id_api_target",
			"targetProfileId": profileID,
			"thresholdMB":     1,
			"entries": []map[string]interface{}{
				{"path": path, "size": 101, "md5": path},
			},
		})
		taskID := taskResp.Data.(map[string]interface{})["task"].(map[string]interface{})["id"].(string)
		runResp := invokeJSON(t, handler, http.MethodPost, "/api/tasks/"+taskID+"/run", nil)
		if got := runResp.Data.(map[string]interface{})["task"].(map[string]interface{})["state"].(string); got != "blocked" {
			t.Fatalf("expected blocked task for %s, got %s", path, got)
		}
		if _, err := application.store.DB().ExecContext(ctx, `UPDATE task_results SET created_at = ? WHERE task_id = ?`,
			time.Now().Add(-2*time.Hour).UTC().Format(time.RFC3339), taskID,
		); err != nil {
			t.Fatalf("update task_results created_at for %s error = %v", path, err)
		}
		return taskID
	}

	firstTaskID := createBlockedTask("/group-a/one.bin")
	secondTaskID := createBlockedTask("/group-b/two.bin")

	recoverResp := invokeJSON(t, handler, http.MethodPost, "/api/tasks/recover", map[string]interface{}{
		"taskId":           secondTaskID,
		"providerKey":      "recover_task_id_api_target",
		"profileId":        profileID,
		"recoverState":     "runnable_now",
		"paths":            []string{"/group-b"},
		"scope":            "selected_retry_subset",
		"limit":            1,
		"limitPerMode":     1,
		"limitPerLane":     1,
		"limitPerProvider": 1,
		"limitPerProfile":  1,
	})
	recoverData := recoverResp.Data.(map[string]interface{})
	if got := int(recoverData["matchedCount"].(float64)); got != 1 {
		t.Fatalf("expected matchedCount 1, got %d", got)
	}
	if got := int(recoverData["recoveredCount"].(float64)); got != 1 {
		t.Fatalf("expected recoveredCount 1, got %d", got)
	}
	if got := recoverData["taskId"].(string); got != secondTaskID {
		t.Fatalf("expected taskId %s, got %s", secondTaskID, got)
	}
	if got := recoverData["profileId"].(string); got != profileID {
		t.Fatalf("expected profileId %s, got %s", profileID, got)
	}
	if got := recoverData["recoverState"].(string); got != "runnable_now" {
		t.Fatalf("expected recoverState runnable_now, got %s", got)
	}
	if got := int(recoverData["limitPerProvider"].(float64)); got != 1 {
		t.Fatalf("expected limitPerProvider 1, got %d", got)
	}
	if got := int(recoverData["limitPerProfile"].(float64)); got != 1 {
		t.Fatalf("expected limitPerProfile 1, got %d", got)
	}
	if got := int(recoverData["limitPerMode"].(float64)); got != 1 {
		t.Fatalf("expected limitPerMode 1, got %d", got)
	}
	if got := int(recoverData["limitPerLane"].(float64)); got != 1 {
		t.Fatalf("expected limitPerLane 1, got %d", got)
	}

	firstDetail := invokeJSON(t, handler, http.MethodGet, "/api/tasks/"+firstTaskID, nil)
	firstResults := firstDetail.Data.(map[string]interface{})["results"].([]interface{})
	if len(firstResults) != 1 || firstResults[0].(map[string]interface{})["payload"].(map[string]interface{})["path"].(string) != "/group-a/one.bin" {
		t.Fatalf("expected first task results unchanged, got %#v", firstResults)
	}

	secondDetail := invokeJSON(t, handler, http.MethodGet, "/api/tasks/"+secondTaskID, nil)
	secondResults := secondDetail.Data.(map[string]interface{})["results"].([]interface{})
	if len(secondResults) != 1 || secondResults[0].(map[string]interface{})["payload"].(map[string]interface{})["path"].(string) != "/group-b/two.bin" {
		t.Fatalf("expected second task recovered path /group-b/two.bin, got %#v", secondResults)
	}
	if got := uploadCallsByPath["/group-a/one.bin"]; got != 1 {
		t.Fatalf("expected first task upload calls to remain 1, got %d", got)
	}
	if got := uploadCallsByPath["/group-b/two.bin"]; got != 2 {
		t.Fatalf("expected second task upload calls to become 2, got %d", got)
	}
}

func TestAppRecoverTasksEndpointFiltersProtocolGroup(t *testing.T) {
	ctx := context.Background()
	application := mustNewTestApp(t, ctx)

	uploadCallsByPath := map[string]int{}
	newAdapter := func(key, protocolGroup string) *appScriptedAdapter {
		return &appScriptedAdapter{
			meta: provider.Provider{
				Key:              key,
				DisplayName:      key,
				ProtocolGroup:    protocolGroup,
				AuthModes:        []string{"manual_token"},
				FastUploadInputs: []string{"md5", "size"},
				FallbackModes:    []string{"download_upload"},
				Status:           "planned",
			},
			capability: provider.CapabilitySet{
				SupportsAuthValidation: true,
				SupportsUpload:         true,
			},
			uploadFunc: func(req provider.UploadRequest) provider.UploadResult {
				uploadCallsByPath[req.Path]++
				if uploadCallsByPath[req.Path] == 1 {
					return provider.UploadResult{
						OperationResult: provider.OperationResult{
							Status:  "rate_limited",
							Message: "rate limited",
							Mode:    "fake_rate_limit",
						},
					}
				}
				return provider.UploadResult{
					OperationResult: provider.OperationResult{
						OK:      true,
						Status:  "ok",
						Message: "protocol group recovered",
						Mode:    "fake_protocol_group_ok",
					},
				}
			},
		}
	}

	adapterA := newAdapter("recover_protocol_group_api_target_a", "recover_protocol_group_a")
	adapterB := newAdapter("recover_protocol_group_api_target_b", "recover_protocol_group_b")
	registry := provider.NewRegistry(adapterA, adapterB)
	authSvc := auth.NewService(application.store, registry)
	taskSvc := task.NewService(application.store, registry, authSvc)
	application.providers = registry
	application.auth = authSvc
	application.tasks = taskSvc
	handler := application.routes()

	createProfile := func(providerKey string) string {
		resp := invokeJSON(t, handler, http.MethodPost, "/api/auth/profiles", map[string]interface{}{
			"providerKey": providerKey,
			"authMode":    "manual_token",
			"displayName": providerKey,
			"token":       "token-" + providerKey,
		})
		return resp.Data.(map[string]interface{})["id"].(string)
	}

	createBlockedTask := func(providerKey, profileID, path string) string {
		taskResp := invokeJSON(t, handler, http.MethodPost, "/api/tasks", map[string]interface{}{
			"sourceProvider":  "recover_protocol_group_api_source",
			"targetProvider":  providerKey,
			"targetProfileId": profileID,
			"thresholdMB":     1,
			"entries": []map[string]interface{}{{
				"path": path,
				"size": 101,
				"md5":  path,
			}},
		})
		taskID := taskResp.Data.(map[string]interface{})["task"].(map[string]interface{})["id"].(string)
		runResp := invokeJSON(t, handler, http.MethodPost, "/api/tasks/"+taskID+"/run", nil)
		if got := runResp.Data.(map[string]interface{})["task"].(map[string]interface{})["state"].(string); got != "blocked" {
			t.Fatalf("expected blocked task for %s, got %s", path, got)
		}
		if _, err := application.store.DB().ExecContext(ctx, `UPDATE task_results SET created_at = ? WHERE task_id = ?`,
			time.Now().Add(-2*time.Hour).UTC().Format(time.RFC3339), taskID,
		); err != nil {
			t.Fatalf("update task_results created_at for %s error = %v", path, err)
		}
		return taskID
	}

	profileA := createProfile("recover_protocol_group_api_target_a")
	profileB := createProfile("recover_protocol_group_api_target_b")
	firstTaskID := createBlockedTask("recover_protocol_group_api_target_a", profileA, "/api-group-a.bin")
	secondTaskID := createBlockedTask("recover_protocol_group_api_target_b", profileB, "/api-group-b.bin")

	recoverResp := invokeJSON(t, handler, http.MethodPost, "/api/tasks/recover", map[string]interface{}{
		"protocolGroup":         "recover_protocol_group_b",
		"limit":                 2,
		"limitPerProtocolGroup": 1,
	})
	recoverData := recoverResp.Data.(map[string]interface{})
	if got := recoverData["protocolGroup"].(string); got != "recover_protocol_group_b" {
		t.Fatalf("expected protocolGroup recover_protocol_group_b, got %s", got)
	}
	if got := int(recoverData["matchedCount"].(float64)); got != 1 {
		t.Fatalf("expected matchedCount 1, got %d", got)
	}
	if got := int(recoverData["recoveredCount"].(float64)); got != 1 {
		t.Fatalf("expected recoveredCount 1, got %d", got)
	}
	if got := int(recoverData["limitPerProtocolGroup"].(float64)); got != 1 {
		t.Fatalf("expected limitPerProtocolGroup 1, got %d", got)
	}

	firstDetail := invokeJSON(t, handler, http.MethodGet, "/api/tasks/"+firstTaskID, nil)
	if got := firstDetail.Data.(map[string]interface{})["task"].(map[string]interface{})["state"].(string); got != "blocked" {
		t.Fatalf("expected first task blocked, got %s", got)
	}
	secondDetail := invokeJSON(t, handler, http.MethodGet, "/api/tasks/"+secondTaskID, nil)
	if got := secondDetail.Data.(map[string]interface{})["task"].(map[string]interface{})["state"].(string); got != "completed" {
		t.Fatalf("expected second task completed, got %s", got)
	}
	if uploadCallsByPath["/api-group-a.bin"] != 1 {
		t.Fatalf("expected /api-group-a.bin upload calls 1, got %d", uploadCallsByPath["/api-group-a.bin"])
	}
	if uploadCallsByPath["/api-group-b.bin"] != 2 {
		t.Fatalf("expected /api-group-b.bin upload calls 2, got %d", uploadCallsByPath["/api-group-b.bin"])
	}
}
func mustNewTestApp(t *testing.T, ctx context.Context) *App {
	t.Helper()

	cfg := Config{
		AppName:                        "CloudPan Sync Go Test",
		Env:                            "test",
		Addr:                           ":0",
		DataDir:                        t.TempDir(),
		DBPath:                         filepath.Join(t.TempDir(), "app.db"),
		AdminPassword:                  "admin",
		LogLevel:                       slog.LevelError,
		AutoRetryTick:                  3 * time.Second,
		AutoRetryBatchLimit:            3,
		AutoRetryLimitPerMode:          1,
		AutoRetryLimitPerLane:          1,
		AutoRetryLimitPerProtocolGroup: 1,
		AutoRetryLimitPerProvider:      1,
		AutoRetryLimitPerProfile:       1,
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

type appScriptedAdapter struct {
	meta         provider.Provider
	capability   provider.CapabilitySet
	validateFunc func(provider.AuthProfile) provider.OperationResult
	uploadFunc   func(provider.UploadRequest) provider.UploadResult
}

func (a *appScriptedAdapter) Meta() provider.Provider { return a.meta }

func (a *appScriptedAdapter) Capabilities() provider.CapabilitySet { return a.capability }

func (a *appScriptedAdapter) ValidateAuth(profile provider.AuthProfile) provider.OperationResult {
	if a.validateFunc != nil {
		return a.validateFunc(profile)
	}
	return provider.OperationResult{OK: true, Status: "verified", Message: "verified", Mode: "scripted_validate"}
}

func (a *appScriptedAdapter) List(req provider.ListRequest) provider.ListResult {
	return provider.ListResult{OperationResult: provider.OperationResult{OK: true, Status: "ok", Message: "ok", Mode: "scripted_list"}}
}

func (a *appScriptedAdapter) Metadata(req provider.MetadataRequest) provider.MetadataResult {
	return provider.MetadataResult{OperationResult: provider.OperationResult{OK: true, Status: "missing", Message: "missing", Mode: "scripted_metadata"}}
}

func (a *appScriptedAdapter) CreateDir(req provider.CreateDirRequest) provider.OperationResult {
	return provider.OperationResult{OK: true, Status: "ok", Message: "ok", Mode: "scripted_create_dir"}
}

func (a *appScriptedAdapter) FastUploadCheck(req provider.FastUploadCheckRequest) provider.FastUploadCheckResult {
	return provider.FastUploadCheckResult{
		OperationResult: provider.OperationResult{OK: true, Status: "hash_miss", Message: "hash miss", Mode: "scripted_fast_check"},
		Candidate:       false,
	}
}

func (a *appScriptedAdapter) Upload(req provider.UploadRequest) provider.UploadResult {
	if a.uploadFunc != nil {
		return a.uploadFunc(req)
	}
	return provider.UploadResult{OperationResult: provider.OperationResult{OK: true, Status: "ok", Message: "ok", Mode: "scripted_upload"}}
}

type appPan123OpenTestState struct {
	lastCreatedFilename string
	uploadedBody        []byte
}

func newAppPan123OpenTestServer(t *testing.T) (*httptest.Server, *appPan123OpenTestState) {
	t.Helper()

	state := &appPan123OpenTestState{}
	rootItems := []map[string]interface{}{
		{"fileId": "dir-demo", "parentFileId": "0", "filename": "demo", "type": 1, "size": 0},
	}
	demoItems := func() []map[string]interface{} {
		items := []map[string]interface{}{}
		if state.lastCreatedFilename != "" {
			items = append(items, map[string]interface{}{
				"fileId":       "file-uploaded",
				"parentFileId": "dir-demo",
				"filename":     state.lastCreatedFilename,
				"type":         0,
				"size":         len(state.uploadedBody),
				"etag":         "etag-uploaded",
			})
		}
		return items
	}

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/upload-put/") {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("read upload body: %v", err)
			}
			state.uploadedBody = body
			w.Header().Set("ETag", `"etag-uploaded"`)
			w.WriteHeader(http.StatusOK)
			return
		}
		if got := r.Header.Get("Authorization"); !strings.HasPrefix(got, "Bearer ") {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2/file/list":
			parentID := r.URL.Query().Get("parentFileId")
			if parentID == "" || parentID == "0" {
				_ = json.NewEncoder(w).Encode(map[string]interface{}{"code": 0, "data": map[string]interface{}{"fileList": rootItems}})
				return
			}
			if parentID == "dir-demo" {
				_ = json.NewEncoder(w).Encode(map[string]interface{}{"code": 0, "data": map[string]interface{}{"fileList": demoItems()}})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"code": 0, "data": map[string]interface{}{"fileList": []map[string]interface{}{}}})
		case r.Method == http.MethodPost && r.URL.Path == "/upload/v1/oss/file/create":
			payload := decodeAppTestJSON(t, r)
			state.lastCreatedFilename = appTestString(payload["filename"])
			if state.lastCreatedFilename == "" {
				state.lastCreatedFilename = appTestString(payload["fileName"])
			}
			if state.lastCreatedFilename == "" {
				state.lastCreatedFilename = appTestString(payload["name"])
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"code": 0, "data": map[string]interface{}{"preuploadID": "preupload-app-1", "fileID": "file-uploaded"}})
		case r.Method == http.MethodPost && r.URL.Path == "/upload/v1/oss/file/get_upload_url":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"code": 0, "data": map[string]interface{}{"presignedURL": server.URL + "/upload-put/preupload-app-1"}})
		case r.Method == http.MethodPost && r.URL.Path == "/upload/v1/oss/file/upload_complete":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"code": 0, "data": map[string]interface{}{"completed": true, "fileID": "file-uploaded"}})
		case r.Method == http.MethodPost && r.URL.Path == "/upload/v1/oss/file/upload_async_result":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"code": 0, "data": map[string]interface{}{"completed": true, "fileID": "file-uploaded"}})
		default:
			http.NotFound(w, r)
		}
	}))
	return server, state
}

func decodeAppTestJSON(t *testing.T, r *http.Request) map[string]interface{} {
	t.Helper()
	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		t.Fatalf("decode request payload: %v", err)
	}
	return payload
}

func appTestString(value interface{}) string {
	text, _ := value.(string)
	return strings.TrimSpace(text)
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

func invokeJSONError(t *testing.T, handler http.Handler, method string, path string, body interface{}) (Envelope, int) {
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
	if rec.Code < 400 {
		t.Fatalf("expected error for %s %s, got status=%d body=%s", method, path, rec.Code, rec.Body.String())
	}
	return envelope, rec.Code
}

func invokeText(t *testing.T, handler http.Handler, method string, path string, body interface{}) string {
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
	if rec.Code >= 400 {
		t.Fatalf("expected success for %s %s, got status=%d body=%s", method, path, rec.Code, rec.Body.String())
	}
	return rec.Body.String()
}
