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
	providerEntry := providerItems[0].(map[string]interface{})
	providerMeta := providerEntry["meta"].(map[string]interface{})
	if hints, ok := providerMeta["riskHints"].([]interface{}); !ok || len(hints) == 0 {
		t.Fatalf("expected provider riskHints in providers list, got %#v", providerMeta["riskHints"])
	}
	if traits, ok := providerMeta["riskTraits"].([]interface{}); !ok || len(traits) == 0 {
		t.Fatalf("expected provider riskTraits in providers list, got %#v", providerMeta["riskTraits"])
	}
	defaultRiskTemplate, ok := providerMeta["defaultRiskTemplate"].(map[string]interface{})
	if !ok || defaultRiskTemplate == nil {
		t.Fatalf("expected provider defaultRiskTemplate in providers list, got %#v", providerMeta["defaultRiskTemplate"])
	}
	if got := defaultRiskTemplate["recommendedMode"].(string); got == "" {
		t.Fatalf("expected provider recommendedMode in defaultRiskTemplate, got %#v", defaultRiskTemplate["recommendedMode"])
	}
	if _, ok := defaultRiskTemplate["calibrated"].(map[string]interface{}); !ok {
		t.Fatalf("expected provider calibrated defaultRiskTemplate profile, got %#v", defaultRiskTemplate["calibrated"])
	}
	if reasons, ok := defaultRiskTemplate["calibrationReasons"].([]interface{}); !ok || len(reasons) == 0 {
		t.Fatalf("expected provider calibrationReasons in defaultRiskTemplate, got %#v", defaultRiskTemplate["calibrationReasons"])
	}

	capabilityResp := invokeJSON(t, handler, http.MethodGet, "/api/providers/123_open/capabilities", nil)
	capabilityData := capabilityResp.Data.(map[string]interface{})
	capabilityProvider := capabilityData["provider"].(map[string]interface{})
	capabilityRiskTemplate, ok := capabilityProvider["defaultRiskTemplate"].(map[string]interface{})
	if !ok || capabilityRiskTemplate == nil {
		t.Fatalf("expected provider defaultRiskTemplate in capability response, got %#v", capabilityProvider["defaultRiskTemplate"])
	}
	if got := capabilityRiskTemplate["recommendedReason"].(string); got == "" {
		t.Fatalf("expected provider defaultRiskTemplate recommendedReason, got %#v", capabilityRiskTemplate["recommendedReason"])
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
	if _, ok := riskResolution["base"].(map[string]interface{}); !ok {
		t.Fatalf("expected preview risk resolution base, got %#v", riskResolution["base"])
	}
	if _, ok := riskResolution["calibrated"].(map[string]interface{}); !ok {
		t.Fatalf("expected preview risk resolution calibrated, got %#v", riskResolution["calibrated"])
	}
	if _, ok := riskResolution["applied"].(map[string]interface{}); !ok {
		t.Fatalf("expected preview risk resolution applied, got %#v", riskResolution["applied"])
	}
	if reasons, ok := riskResolution["calibrationReasons"].([]interface{}); !ok || len(reasons) == 0 {
		t.Fatalf("expected preview calibrationReasons, got %#v", riskResolution["calibrationReasons"])
	}
	if hints, ok := riskResolution["providerRiskHints"].([]interface{}); !ok || len(hints) == 0 {
		t.Fatalf("expected preview providerRiskHints, got %#v", riskResolution["providerRiskHints"])
	}
	if traits, ok := riskResolution["providerRiskTraits"].([]interface{}); !ok || len(traits) == 0 {
		t.Fatalf("expected preview providerRiskTraits, got %#v", riskResolution["providerRiskTraits"])
	}
	if got := metadata["recommendedRiskMode"].(string); got == "" {
		t.Fatal("expected recommendedRiskMode in preview metadata")
	}
	if got := metadata["recommendedRiskModeReason"].(string); got == "" {
		t.Fatal("expected recommendedRiskModeReason in preview metadata")
	}
	if got := metadata["aggressiveRiskWarning"].(string); got == "" {
		t.Fatal("expected aggressiveRiskWarning in preview metadata")
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
	if got := createdMetadata["recommendedRiskMode"].(string); got == "" {
		t.Fatal("expected task recommendedRiskMode")
	}
	if got := createdMetadata["recommendedRiskModeReason"].(string); got == "" {
		t.Fatal("expected task recommendedRiskModeReason")
	}
	if got := createdMetadata["aggressiveRiskWarning"].(string); got == "" {
		t.Fatal("expected task aggressiveRiskWarning")
	}
	if _, ok := createdResolution["base"].(map[string]interface{}); !ok {
		t.Fatalf("expected task risk resolution base, got %#v", createdResolution["base"])
	}
	if _, ok := createdResolution["calibrated"].(map[string]interface{}); !ok {
		t.Fatalf("expected task risk resolution calibrated, got %#v", createdResolution["calibrated"])
	}
	if _, ok := createdResolution["applied"].(map[string]interface{}); !ok {
		t.Fatalf("expected task risk resolution applied, got %#v", createdResolution["applied"])
	}
	if hints, ok := createdResolution["providerRiskHints"].([]interface{}); !ok || len(hints) == 0 {
		t.Fatalf("expected task providerRiskHints, got %#v", createdResolution["providerRiskHints"])
	}
	if traits, ok := createdResolution["providerRiskTraits"].([]interface{}); !ok || len(traits) == 0 {
		t.Fatalf("expected task providerRiskTraits, got %#v", createdResolution["providerRiskTraits"])
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
	if got := firstResultPayload["sourceDeletePolicy"].(string); got != "record_only" {
		t.Fatalf("expected result sourceDeletePolicy record_only, got %s", got)
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
	if got := int(evidenceData["autoRecoverTasks"].(float64)); got != 1 {
		t.Fatalf("expected autoRecoverTasks=1, got %d", got)
	}
	if got := int(evidenceData["autoRecoverRunnableTasks"].(float64)); got != 0 {
		t.Fatalf("expected autoRecoverRunnableTasks=0, got %d", got)
	}
	if got := int(evidenceData["autoRecoverWaitingCooldownTasks"].(float64)); got != 0 {
		t.Fatalf("expected autoRecoverWaitingCooldownTasks=0, got %d", got)
	}
	if got := int(evidenceData["autoRecoverWaitingRetryWindowTasks"].(float64)); got != 0 {
		t.Fatalf("expected autoRecoverWaitingRetryWindowTasks=0, got %d", got)
	}
	if got := int(evidenceData["autoRecoverWaitingAuthRefreshTasks"].(float64)); got != 0 {
		t.Fatalf("expected autoRecoverWaitingAuthRefreshTasks=0, got %d", got)
	}
	if got := int(evidenceData["autoRecoverWaitingLocalRestoreTasks"].(float64)); got != 1 {
		t.Fatalf("expected autoRecoverWaitingLocalRestoreTasks=1, got %d", got)
	}
	if got := int(evidenceData["autoRecoverWaitingManualTasks"].(float64)); got != 0 {
		t.Fatalf("expected autoRecoverWaitingManualTasks=0, got %d", got)
	}
	if got := int(evidenceData["autoRecoverWaitingRetryLimitTasks"].(float64)); got != 0 {
		t.Fatalf("expected autoRecoverWaitingRetryLimitTasks=0, got %d", got)
	}
	if got := int(evidenceData["autoRecoverWaitingOtherTasks"].(float64)); got != 0 {
		t.Fatalf("expected autoRecoverWaitingOtherTasks=0, got %d", got)
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
	if got, ok := evidenceData["executionMode"].(string); !ok || got != "pre_scan_flat" {
		t.Fatalf("expected evidence executionMode pre_scan_flat, got %s", got)
	}
	if got, ok := evidenceData["sourceDeletePolicy"].(string); !ok || got != "record_only" {
		t.Fatalf("expected evidence sourceDeletePolicy record_only, got %s", got)
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
	recentResultPayload := evidenceData["recentResults"].([]interface{})[0].(map[string]interface{})["payload"].(map[string]interface{})
	if got := recentResultPayload["sourceDeletePolicy"].(string); got != "record_only" {
		t.Fatalf("expected recent result sourceDeletePolicy record_only, got %s", got)
	}
	if got := len(evidenceData["recentProbes"].([]interface{})); got == 0 {
		t.Fatal("expected recentProbes to be populated")
	}
	recentProbePayload := evidenceData["recentProbes"].([]interface{})[0].(map[string]interface{})["payload"].(map[string]interface{})
	if got := recentProbePayload["executionMode"].(string); got != "pre_scan_flat" {
		t.Fatalf("expected probe executionMode pre_scan_flat, got %s", got)
	}
	if got, ok := recentProbePayload["scanMode"].(string); !ok || got != "pre_scan_flat" {
		t.Fatalf("expected probe scanMode pre_scan_flat, got %s", got)
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
	if got := int(recentProbePayload["retrySummary"].(map[string]interface{})["autoRecoverWaitingLocalRestoreTasks"].(float64)); got != 1 {
		t.Fatalf("expected probe autoRecoverWaitingLocalRestoreTasks 1, got %d", got)
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
		if got, ok := summary["scanMode"].(string); !ok || got != "pre_scan_flat" {
			t.Fatalf("expected status summary scanMode pre_scan_flat, got %s", got)
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
		riskResolution, ok := summary["riskProfileResolution"].(map[string]interface{})
		if !ok || riskResolution == nil {
			t.Fatalf("expected status summary riskProfileResolution, got %#v", summary["riskProfileResolution"])
		}
		if _, ok := riskResolution["applied"].(map[string]interface{}); !ok {
			t.Fatalf("expected status summary risk resolution applied, got %#v", riskResolution["applied"])
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
		if got := int(item["autoRecoverCount"].(float64)); got != 1 {
			t.Fatalf("expected provider autoRecoverCount 1, got %d", got)
		}
		if got := int(summary["autoRecoverRunnableTasks"].(float64)); got != 0 {
			t.Fatalf("expected status summary autoRecoverRunnableTasks 0, got %d", got)
		}
		if got := int(summary["autoRecoverWaitingCooldownTasks"].(float64)); got != 0 {
			t.Fatalf("expected status summary autoRecoverWaitingCooldownTasks 0, got %d", got)
		}
		if got := int(summary["autoRecoverWaitingRetryWindowTasks"].(float64)); got != 0 {
			t.Fatalf("expected status summary autoRecoverWaitingRetryWindowTasks 0, got %d", got)
		}
		if got := int(summary["autoRecoverWaitingAuthRefreshTasks"].(float64)); got != 0 {
			t.Fatalf("expected status summary autoRecoverWaitingAuthRefreshTasks 0, got %d", got)
		}
		if got := int(summary["autoRecoverWaitingLocalRestoreTasks"].(float64)); got != 1 {
			t.Fatalf("expected status summary autoRecoverWaitingLocalRestoreTasks 1, got %d", got)
		}
		if got := int(summary["autoRecoverWaitingManualTasks"].(float64)); got != 0 {
			t.Fatalf("expected status summary autoRecoverWaitingManualTasks 0, got %d", got)
		}
		if got := int(summary["autoRecoverWaitingRetryLimitTasks"].(float64)); got != 0 {
			t.Fatalf("expected status summary autoRecoverWaitingRetryLimitTasks 0, got %d", got)
		}
		if got := int(summary["autoRecoverWaitingOtherTasks"].(float64)); got != 0 {
			t.Fatalf("expected status summary autoRecoverWaitingOtherTasks 0, got %d", got)
		}
		if got := int(summary["retrySummary"].(map[string]interface{})["autoRecoverWaitingLocalRestoreTasks"].(float64)); got != 1 {
			t.Fatalf("expected status retrySummary autoRecoverWaitingLocalRestoreTasks 1, got %d", got)
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
		"category":      "binary_upload_success",
		"result":        "success",
		"title":         "工作流真实 upload smoke",
		"note":          "ValidateAuth/List/Metadata/Upload",
		"operations":    []string{"ValidateAuth", "List", "Metadata", "Upload"},
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
	if got := smokeData["category"].(string); got != "binary_upload_success" {
		t.Fatalf("expected smoke category binary_upload_success, got %s", got)
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
	if got := smokeSummary[0].(map[string]interface{})["sampleCategory"].(string); got != "binary_upload_success" {
		t.Fatalf("expected smoke summary sampleCategory binary_upload_success, got %s", got)
	}
	if got := smokeSummary[0].(map[string]interface{})["hasRealSuccessSample"].(bool); !got {
		t.Fatal("expected smoke summary hasRealSuccessSample true")
	}
	if got := smokeSummary[0].(map[string]interface{})["hasUploadSuccessSample"].(bool); !got {
		t.Fatal("expected upload smoke summary hasUploadSuccessSample true")
	}
	if got := smokeSummary[0].(map[string]interface{})["uploadSuccessCount"].(float64); got != 1 {
		t.Fatalf("expected uploadSuccessCount 1, got %v", got)
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
	if got := smokeMatrix[0].(map[string]interface{})["hasUploadSuccessSample"].(bool); !got {
		t.Fatal("expected upload smoke matrix hasUploadSuccessSample true")
	}
	if got := smokeMatrix[0].(map[string]interface{})["uploadSuccessCount"].(float64); got != 1 {
		t.Fatalf("expected matrix uploadSuccessCount 1, got %v", got)
	}
	if got := smokeMatrix[0].(map[string]interface{})["acceptanceStatus"].(string); got != "accepted" {
		t.Fatalf("expected smoke matrix acceptanceStatus accepted, got %s", got)
	}
	if got := smokeMatrix[0].(map[string]interface{})["accepted"].(bool); !got {
		t.Fatal("expected smoke matrix accepted true")
	}
	if got := len(smokeMatrix[0].(map[string]interface{})["acceptanceActions"].([]interface{})); got == 0 {
		t.Fatal("expected accepted smoke matrix acceptance actions")
	}
	foundMissing := false
	for _, raw := range smokeMatrix {
		item := raw.(map[string]interface{})
		if item["acceptanceStatus"].(string) == "pending" {
			if got := len(item["acceptanceMissing"].([]interface{})); got == 0 {
				t.Fatal("expected pending smoke matrix row to include missing reasons")
			}
			if got := len(item["acceptanceActions"].([]interface{})); got == 0 {
				t.Fatal("expected pending smoke matrix row to include acceptance actions")
			}
			foundMissing = true
			break
		}
	}
	if !foundMissing {
		t.Fatal("expected at least one pending smoke matrix row")
	}
	if counts, ok := evidenceData["acceptanceActionCounts"].(map[string]interface{}); !ok {
		t.Fatalf("expected acceptanceActionCounts map in evidence, got %#v", evidenceData["acceptanceActionCounts"])
	} else if got, ok := counts["补 1 条真实任务覆盖样本"].(float64); !ok || got < 1 {
		t.Fatalf("expected acceptanceActionCounts to include follow-up action, got %#v", counts)
	}

	smokeMarkdown := invokeText(t, handler, http.MethodGet, "/api/provider-smokes/"+smokeID+"?format=markdown", nil)
	if !strings.Contains(smokeMarkdown, "工作流真实 upload smoke") {
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
	if !strings.Contains(reportData["markdown"].(string), "Actions") {
		t.Fatalf("expected report markdown to include acceptance actions column, got %s", reportData["markdown"].(string))
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
	reportAfterRetryResp := invokeJSON(t, handler, http.MethodGet, "/api/evidence/report", nil)
	reportAfterRetryMarkdown := reportAfterRetryResp.Data.(map[string]interface{})["markdown"].(string)
	if !strings.Contains(reportAfterRetryMarkdown, "RetryMode") || !strings.Contains(reportAfterRetryMarkdown, "RetryScope") || !strings.Contains(reportAfterRetryMarkdown, "RetryPaths") {
		t.Fatalf("expected retry evidence headers in report markdown after retry, got %s", reportAfterRetryMarkdown)
	}
	if !strings.Contains(reportAfterRetryMarkdown, "selected_pending_subset") {
		t.Fatalf("expected report markdown to include selected_pending_subset after retry, got %s", reportAfterRetryMarkdown)
	}
	if !strings.Contains(reportAfterRetryMarkdown, "/demo/pending.bin") {
		t.Fatalf("expected report markdown to include retried path /demo/pending.bin, got %s", reportAfterRetryMarkdown)
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

	runRetryResp := invokeJSON(t, handler, http.MethodPost, "/api/tasks/"+taskID+"/run", nil)
	runRetryData := runRetryResp.Data.(map[string]interface{})
	retriedResults := runRetryData["results"].([]interface{})
	if len(retriedResults) != 1 {
		t.Fatalf("expected retried run to keep only 1 result, got %d", len(retriedResults))
	}
	retriedResultPayload := retriedResults[0].(map[string]interface{})["payload"].(map[string]interface{})
	if got := retriedResultPayload["retryMode"].(string); got != "selected_directory_subset" {
		t.Fatalf("expected retried result retryMode selected_directory_subset, got %s", got)
	}
	if got := retriedResultPayload["retryScope"].(string); got != "selected_directory_subset" {
		t.Fatalf("expected retried result retryScope selected_directory_subset, got %s", got)
	}
	retriedSelectedPaths, ok := retriedResultPayload["retrySelectedPaths"].([]interface{})
	if !ok || len(retriedSelectedPaths) != 1 || retriedSelectedPaths[0].(string) != "/1/11" {
		t.Fatalf("expected retried result retrySelectedPaths [/1/11], got %#v", retriedResultPayload["retrySelectedPaths"])
	}

	evidenceResp := invokeJSON(t, handler, http.MethodGet, "/api/evidence/runtime", nil)
	evidenceData := evidenceResp.Data.(map[string]interface{})
	evidenceRecentResultPayload := evidenceData["recentResults"].([]interface{})[0].(map[string]interface{})["payload"].(map[string]interface{})
	if got := evidenceRecentResultPayload["retryMode"].(string); got != "selected_directory_subset" {
		t.Fatalf("expected evidence recent result retryMode selected_directory_subset, got %s", got)
	}
	if got := evidenceRecentResultPayload["retryScope"].(string); got != "selected_directory_subset" {
		t.Fatalf("expected evidence recent result retryScope selected_directory_subset, got %s", got)
	}
	evidenceRecentResultSelectedPaths, ok := evidenceRecentResultPayload["retrySelectedPaths"].([]interface{})
	if !ok || len(evidenceRecentResultSelectedPaths) != 1 || evidenceRecentResultSelectedPaths[0].(string) != "/1/11" {
		t.Fatalf("expected evidence recent result retrySelectedPaths [/1/11], got %#v", evidenceRecentResultPayload["retrySelectedPaths"])
	}
	evidenceRecentProbePayload := evidenceData["recentProbes"].([]interface{})[0].(map[string]interface{})["payload"].(map[string]interface{})
	if got := evidenceRecentProbePayload["retryMode"].(string); got != "selected_directory_subset" {
		t.Fatalf("expected evidence recent probe retryMode selected_directory_subset, got %s", got)
	}
	if got := evidenceRecentProbePayload["retryScope"].(string); got != "selected_directory_subset" {
		t.Fatalf("expected evidence recent probe retryScope selected_directory_subset, got %s", got)
	}
	evidenceRecentProbeSelectedPaths, ok := evidenceRecentProbePayload["retrySelectedPaths"].([]interface{})
	if !ok || len(evidenceRecentProbeSelectedPaths) != 1 || evidenceRecentProbeSelectedPaths[0].(string) != "/1/11" {
		t.Fatalf("expected evidence recent probe retrySelectedPaths [/1/11], got %#v", evidenceRecentProbePayload["retrySelectedPaths"])
	}

	statusResp := invokeJSON(t, handler, http.MethodGet, "/api/status/providers", nil)
	statusItems := statusResp.Data.(map[string]interface{})["items"].([]interface{})
	foundTargetStatus := false
	for _, raw := range statusItems {
		item := raw.(map[string]interface{})
		if item["providerKey"].(string) != "guangya" {
			continue
		}
		foundTargetStatus = true
		summary := item["snapshotSummary"].(map[string]interface{})
		if got := summary["retryMode"].(string); got != "selected_directory_subset" {
			t.Fatalf("expected status summary retryMode selected_directory_subset, got %s", got)
		}
		if got := summary["retryScope"].(string); got != "selected_directory_subset" {
			t.Fatalf("expected status summary retryScope selected_directory_subset, got %s", got)
		}
		statusSelectedPaths, ok := summary["retrySelectedPaths"].([]interface{})
		if !ok || len(statusSelectedPaths) != 1 || statusSelectedPaths[0].(string) != "/1/11" {
			t.Fatalf("expected status summary retrySelectedPaths [/1/11], got %#v", summary["retrySelectedPaths"])
		}
	}
	if !foundTargetStatus {
		t.Fatal("expected guangya provider status after selected directory retry")
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
	if counts, ok := recoverData["outcomeCounts"].(map[string]interface{}); !ok {
		t.Fatalf("expected outcomeCounts map, got %#v", recoverData["outcomeCounts"])
	} else if got := int(counts["recovered"].(float64)); got != 1 {
		t.Fatalf("expected recovered outcome count 1, got %d", got)
	}
	if counts, ok := recoverData["retryClassCounts"].(map[string]interface{}); !ok {
		t.Fatalf("expected retryClassCounts map, got %#v", recoverData["retryClassCounts"])
	} else if got := int(counts["retry_failed"].(float64)); got != 1 {
		t.Fatalf("expected retry_failed class count 1, got %d", got)
	}
	if counts, ok := recoverData["recoverStateCounts"].(map[string]interface{}); !ok {
		t.Fatalf("expected recoverStateCounts map, got %#v", recoverData["recoverStateCounts"])
	} else if got := int(counts["runnable_now"].(float64)); got != 1 {
		t.Fatalf("expected runnable_now state count 1, got %d", got)
	}
	if counts, ok := recoverData["protocolGroupCounts"].(map[string]interface{}); !ok {
		t.Fatalf("expected protocolGroupCounts map, got %#v", recoverData["protocolGroupCounts"])
	} else if got := int(counts["fake_target"].(float64)); got != 1 {
		t.Fatalf("expected fake_target protocol group count 1, got %d", got)
	}
	if counts, ok := recoverData["providerCounts"].(map[string]interface{}); !ok {
		t.Fatalf("expected providerCounts map, got %#v", recoverData["providerCounts"])
	} else if got := int(counts["recover_api_target"].(float64)); got != 1 {
		t.Fatalf("expected recover_api_target provider count 1, got %d", got)
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

func TestAppWorkflowSurfacesMissingUploadIDAsManualIntervention(t *testing.T) {
	ctx := context.Background()
	application := mustNewTestApp(t, ctx)
	handler := application.routes()

	targetAdapter := &appScriptedAdapter{
		meta:       provider.Provider{Key: "missing_uploadid_target", DisplayName: "Missing UploadID Target", ProtocolGroup: "fake_target", AuthModes: []string{"manual_token"}, FastUploadInputs: []string{"md5", "size"}, FallbackModes: []string{"download_upload"}, Status: "planned"},
		capability: provider.CapabilitySet{SupportsAuthValidation: true, SupportsUpload: true},
		uploadFunc: func(req provider.UploadRequest) provider.UploadResult {
			return provider.UploadResult{OperationResult: provider.OperationResult{Status: "missing_uploadid", Message: "provider omitted uploadid", Mode: "scripted_missing_uploadid"}}
		},
	}
	registry := provider.NewRegistry(targetAdapter)
	authSvc := auth.NewService(application.store, registry)
	taskSvc := task.NewService(application.store, registry, authSvc)
	application.providers = registry
	application.auth = authSvc
	application.tasks = taskSvc
	handler = application.routes()

	profileResp := invokeJSON(t, handler, http.MethodPost, "/api/auth/profiles", map[string]interface{}{"providerKey": "missing_uploadid_target", "authMode": "manual_token", "displayName": "Missing UploadID Target", "token": "token-missing-uploadid"})
	profileID := profileResp.Data.(map[string]interface{})["id"].(string)
	localFile := filepath.Join(t.TempDir(), "missing-uploadid.bin")
	if err := os.WriteFile(localFile, []byte("missing-uploadid"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	taskResp := invokeJSON(t, handler, http.MethodPost, "/api/tasks", map[string]interface{}{"sourceProvider": "missing_uploadid_source", "targetProvider": "missing_uploadid_target", "targetProfileId": profileID, "thresholdMB": 1, "entries": []map[string]interface{}{{"path": "/missing-uploadid.bin", "size": 1024, "md5": "missing-md5", "localPath": localFile}}})
	taskID := taskResp.Data.(map[string]interface{})["task"].(map[string]interface{})["id"].(string)
	runResp := invokeJSON(t, handler, http.MethodPost, "/api/tasks/"+taskID+"/run", nil)
	runData := runResp.Data.(map[string]interface{})
	if got := runData["task"].(map[string]interface{})["state"].(string); got != "blocked" {
		t.Fatalf("expected blocked, got %s", got)
	}
	runtimeData := runData["runtime"].(map[string]interface{})
	if got := runtimeData["blockedReason"].(string); got != "retry_queue_requires_provider_session_rebuild" {
		t.Fatalf("expected blockedReason retry_queue_requires_provider_session_rebuild, got %s", got)
	}
	if got := runtimeData["blockedAction"].(string); got != "manual_intervention_required" {
		t.Fatalf("expected blockedAction manual_intervention_required, got %s", got)
	}
	if got := runtimeData["blockedAdvice"].(string); got == "" {
		t.Fatal("expected blockedAdvice on runtime payload")
	}
	retryQueue := runtimeData["retryQueue"].([]interface{})
	if len(retryQueue) != 1 {
		t.Fatalf("expected retryQueue len 1, got %#v", retryQueue)
	}
	item := retryQueue[0].(map[string]interface{})
	if got := item["providerStatus"].(string); got != "missing_uploadid" {
		t.Fatalf("expected providerStatus missing_uploadid, got %s", got)
	}
	if got := item["retryClass"].(string); got != "provider_session_missing" {
		t.Fatalf("expected retryClass provider_session_missing, got %s", got)
	}
	if got := item["retryAction"].(string); got != "manual_intervention_required" {
		t.Fatalf("expected retryAction manual_intervention_required, got %s", got)
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
	if got := int(recoverData["suggestedLimitPerMode"].(float64)); got != 1 {
		t.Fatalf("expected suggestedLimitPerMode 1, got %d", got)
	}
	if got := int(recoverData["suggestedLimitPerLane"].(float64)); got != 1 {
		t.Fatalf("expected suggestedLimitPerLane 1, got %d", got)
	}
	if got := int(recoverData["suggestedLimitPerProtocolGroup"].(float64)); got != 1 {
		t.Fatalf("expected suggestedLimitPerProtocolGroup 1, got %d", got)
	}
	if got := int(recoverData["suggestedLimitPerProvider"].(float64)); got != 2 {
		t.Fatalf("expected suggestedLimitPerProvider 2, got %d", got)
	}
	if got := int(recoverData["suggestedLimitPerProfile"].(float64)); got != 1 {
		t.Fatalf("expected suggestedLimitPerProfile 1, got %d", got)
	}
	if decisions, ok := recoverData["decisions"].([]interface{}); !ok || len(decisions) < 2 {
		t.Fatalf("expected recover decisions with budget skip, got %#v", recoverData["decisions"])
	} else {
		var skipped map[string]interface{}
		for _, raw := range decisions {
			decision := raw.(map[string]interface{})
			if decision["outcome"].(string) == "skipped_provider_budget" {
				skipped = decision
				break
			}
		}
		if skipped == nil {
			t.Fatalf("expected skipped_provider_budget decision, got %#v", decisions)
		}
		if got := int(skipped["suggestedProviderBudget"].(float64)); got != 2 {
			t.Fatalf("expected skipped decision suggestedProviderBudget 2, got %d", got)
		}
		if got := int(skipped["suggestedModeBudget"].(float64)); got != 1 {
			t.Fatalf("expected skipped decision suggestedModeBudget 1, got %d", got)
		}
		if got := int(skipped["suggestedLaneBudget"].(float64)); got != 1 {
			t.Fatalf("expected skipped decision suggestedLaneBudget 1, got %d", got)
		}
		if got := skipped["advice"].(string); !strings.Contains(got, "provider 预算提高到 2") {
			t.Fatalf("expected skipped decision advice to mention provider budget 2, got %s", got)
		}
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
	if counts, ok := previewData["outcomeCounts"].(map[string]interface{}); !ok {
		t.Fatalf("expected preview outcomeCounts map, got %#v", previewData["outcomeCounts"])
	} else if got := int(counts["dry_run_recoverable"].(float64)); got != 1 {
		t.Fatalf("expected preview dry_run_recoverable count 1, got %d", got)
	}
	if counts, ok := previewData["retryClassCounts"].(map[string]interface{}); !ok {
		t.Fatalf("expected preview retryClassCounts map, got %#v", previewData["retryClassCounts"])
	} else if got := int(counts["retry_failed"].(float64)); got != 1 {
		t.Fatalf("expected preview retry_failed count 1, got %d", got)
	}
	if counts, ok := previewData["recoverStateCounts"].(map[string]interface{}); !ok {
		t.Fatalf("expected preview recoverStateCounts map, got %#v", previewData["recoverStateCounts"])
	} else if got := int(counts["runnable_now"].(float64)); got != 1 {
		t.Fatalf("expected preview runnable_now count 1, got %d", got)
	}
	if counts, ok := previewData["protocolGroupCounts"].(map[string]interface{}); !ok {
		t.Fatalf("expected preview protocolGroupCounts map, got %#v", previewData["protocolGroupCounts"])
	} else if got := int(counts["recover_dry_run_api_group"].(float64)); got != 1 {
		t.Fatalf("expected preview recover_dry_run_api_group count 1, got %d", got)
	}
	if counts, ok := previewData["providerCounts"].(map[string]interface{}); !ok {
		t.Fatalf("expected preview providerCounts map, got %#v", previewData["providerCounts"])
	} else if got := int(counts["recover_dry_run_api_target"].(float64)); got != 1 {
		t.Fatalf("expected preview recover_dry_run_api_target count 1, got %d", got)
	}
	if counts, ok := previewData["profileCounts"].(map[string]interface{}); !ok {
		t.Fatalf("expected preview profileCounts map, got %#v", previewData["profileCounts"])
	} else if got := int(counts[profileID].(float64)); got != 1 {
		t.Fatalf("expected preview profile count 1, got %d", got)
	}
	if counts, ok := previewData["laneCounts"].(map[string]interface{}); !ok {
		t.Fatalf("expected preview laneCounts map, got %#v", previewData["laneCounts"])
	} else if got := int(counts["upload_checkpoint_auto_resume::retry_failed::"].(float64)); got != 1 {
		t.Fatalf("expected preview lane count 1, got %d", got)
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
	if counts, ok := recoverData["outcomeCounts"].(map[string]interface{}); !ok {
		t.Fatalf("expected execute outcomeCounts map, got %#v", recoverData["outcomeCounts"])
	} else if got := int(counts["recovered"].(float64)); got != 1 {
		t.Fatalf("expected execute recovered count 1, got %d", got)
	}
	if counts, ok := recoverData["retryClassCounts"].(map[string]interface{}); !ok {
		t.Fatalf("expected execute retryClassCounts map, got %#v", recoverData["retryClassCounts"])
	} else if got := int(counts["retry_failed"].(float64)); got != 1 {
		t.Fatalf("expected execute retry_failed count 1, got %d", got)
	}
	if counts, ok := recoverData["recoverStateCounts"].(map[string]interface{}); !ok {
		t.Fatalf("expected execute recoverStateCounts map, got %#v", recoverData["recoverStateCounts"])
	} else if got := int(counts["runnable_now"].(float64)); got != 1 {
		t.Fatalf("expected execute runnable_now count 1, got %d", got)
	}
	if counts, ok := recoverData["protocolGroupCounts"].(map[string]interface{}); !ok {
		t.Fatalf("expected execute protocolGroupCounts map, got %#v", recoverData["protocolGroupCounts"])
	} else if got := int(counts["recover_dry_run_api_group"].(float64)); got != 1 {
		t.Fatalf("expected execute recover_dry_run_api_group count 1, got %d", got)
	}
	if counts, ok := recoverData["providerCounts"].(map[string]interface{}); !ok {
		t.Fatalf("expected execute providerCounts map, got %#v", recoverData["providerCounts"])
	} else if got := int(counts["recover_dry_run_api_target"].(float64)); got != 1 {
		t.Fatalf("expected execute recover_dry_run_api_target count 1, got %d", got)
	}
	if counts, ok := recoverData["profileCounts"].(map[string]interface{}); !ok {
		t.Fatalf("expected execute profileCounts map, got %#v", recoverData["profileCounts"])
	} else if got := int(counts[profileID].(float64)); got != 1 {
		t.Fatalf("expected execute profile count 1, got %d", got)
	}
	if counts, ok := recoverData["laneCounts"].(map[string]interface{}); !ok {
		t.Fatalf("expected execute laneCounts map, got %#v", recoverData["laneCounts"])
	} else if got := int(counts["upload_checkpoint_auto_resume::retry_failed::"].(float64)); got != 1 {
		t.Fatalf("expected execute lane count 1, got %d", got)
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

func TestAppRecoverTasksEndpointSupportsSelectedDirectorySubsetScope(t *testing.T) {
	ctx := context.Background()
	application := mustNewTestApp(t, ctx)

	uploadCallsByPath := map[string]int{}
	targetAdapter := &appScriptedAdapter{
		meta: provider.Provider{
			Key:              "recover_directory_scope_target",
			DisplayName:      "Recover Directory Scope Target",
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
					Message: "directory scope recovered",
					Mode:    "scripted_directory_scope_ok",
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
		"providerKey": "recover_directory_scope_target",
		"authMode":    "manual_token",
		"displayName": "Recover Directory Scope Target",
		"token":       "token-recover-directory-scope",
	})
	profileID := profileResp.Data.(map[string]interface{})["id"].(string)

	taskResp := invokeJSON(t, handler, http.MethodPost, "/api/tasks", map[string]interface{}{
		"sourceProvider":  "recover_directory_scope_source",
		"targetProvider":  "recover_directory_scope_target",
		"targetProfileId": profileID,
		"thresholdMB":     1,
		"entries": []map[string]interface{}{
			{"path": "/scope-a/one.bin", "size": 101, "md5": "scope-a"},
			{"path": "/scope-b/two.bin", "size": 202, "md5": "scope-b"},
			{"path": "/scope-c/three.bin", "size": 303, "md5": "scope-c"},
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
		"providerKey":      "recover_directory_scope_target",
		"paths":            []string{"/scope-a", "/scope-c"},
		"scope":            "selected_directory_subset",
		"limitPerMode":     2,
		"limitPerLane":     2,
		"limitPerProvider": 2,
		"limitPerProfile":  2,
	})
	recoverData := recoverResp.Data.(map[string]interface{})
	if got := recoverData["scope"].(string); got != "selected_directory_subset" {
		t.Fatalf("expected scope selected_directory_subset, got %s", got)
	}
	if got := int(recoverData["matchedCount"].(float64)); got != 1 {
		t.Fatalf("expected matchedCount 1 task, got %d", got)
	}
	if got := int(recoverData["recoveredCount"].(float64)); got != 1 {
		t.Fatalf("expected recoveredCount 1 task, got %d", got)
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
	if paths[0] != "/scope-a/one.bin" || paths[1] != "/scope-c/three.bin" {
		t.Fatalf("expected recovered paths [/scope-a/one.bin /scope-c/three.bin], got %#v", paths)
	}
	if got := uploadCallsByPath["/scope-b/two.bin"]; got != 1 {
		t.Fatalf("expected /scope-b/two.bin upload calls to remain 1, got %d", got)
	}
	if got := int(recoverData["limitPerMode"].(float64)); got != 2 {
		t.Fatalf("expected limitPerMode 2, got %d", got)
	}
	if got := int(recoverData["limitPerLane"].(float64)); got != 2 {
		t.Fatalf("expected limitPerLane 2, got %d", got)
	}
}

func TestAppRecoverTasksEndpointSupportsSelectedPendingSubsetScope(t *testing.T) {
	ctx := context.Background()
	application := mustNewTestApp(t, ctx)
	uploadCalls := []string{}

	targetAdapter := &appScriptedAdapter{
		meta: provider.Provider{
			Key:              "recover_pending_scope_target",
			DisplayName:      "Recover Pending Scope Target",
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
			uploadCalls = append(uploadCalls, req.Path)
			if len(uploadCalls) <= 2 {
				return provider.UploadResult{
					OperationResult: provider.OperationResult{
						Status:  "pending_manual_requires_confirmation",
						Message: "pending manual",
						Mode:    "scripted_pending_manual",
						Payload: map[string]interface{}{
							"providerStatus": "pending_manual_requires_confirmation",
						},
					},
				}
			}
			return provider.UploadResult{
				OperationResult: provider.OperationResult{
					OK:      true,
					Status:  "ok",
					Message: "pending scope recovered",
					Mode:    "scripted_pending_scope_ok",
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
		"providerKey": "recover_pending_scope_target",
		"authMode":    "manual_token",
		"displayName": "Recover Pending Scope Target",
		"token":       "token-recover-pending-scope",
	})
	profileID := profileResp.Data.(map[string]interface{})["id"].(string)

	taskResp := invokeJSON(t, handler, http.MethodPost, "/api/tasks", map[string]interface{}{
		"sourceProvider":  "recover_pending_scope_source",
		"targetProvider":  "recover_pending_scope_target",
		"targetProfileId": profileID,
		"thresholdMB":     1,
		"entries": []map[string]interface{}{
			{"path": "/pending-a/one.bin", "size": 111, "md5": "pending-a"},
			{"path": "/pending-b/two.bin", "size": 222, "md5": "pending-b"},
		},
	})
	taskID := taskResp.Data.(map[string]interface{})["task"].(map[string]interface{})["id"].(string)

	runResp := invokeJSON(t, handler, http.MethodPost, "/api/tasks/"+taskID+"/run", nil)
	if got := runResp.Data.(map[string]interface{})["task"].(map[string]interface{})["state"].(string); got != "blocked" {
		t.Fatalf("expected blocked, got %s", got)
	}

	recoverResp := invokeJSON(t, handler, http.MethodPost, "/api/tasks/recover", map[string]interface{}{
		"taskId":           taskID,
		"providerKey":      "recover_pending_scope_target",
		"paths":            []string{"/pending-a/one.bin"},
		"scope":            "selected_pending_subset",
		"limitPerMode":     1,
		"limitPerLane":     1,
		"limitPerProvider": 1,
		"limitPerProfile":  1,
	})
	recoverData := recoverResp.Data.(map[string]interface{})
	if got := recoverData["scope"].(string); got != "selected_pending_subset" {
		t.Fatalf("expected scope selected_pending_subset, got %s", got)
	}
	if got := int(recoverData["matchedCount"].(float64)); got != 1 {
		t.Fatalf("expected matchedCount 1, got %d", got)
	}
	if got := int(recoverData["recoveredCount"].(float64)); got != 1 {
		t.Fatalf("expected recoveredCount 1, got %d", got)
	}

	detailResp := invokeJSON(t, handler, http.MethodGet, "/api/tasks/"+taskID, nil)
	results := detailResp.Data.(map[string]interface{})["results"].([]interface{})
	if len(results) != 1 {
		t.Fatalf("expected 1 recovered result, got %#v", results)
	}
	if got := results[0].(map[string]interface{})["payload"].(map[string]interface{})["path"].(string); got != "/pending-a/one.bin" {
		t.Fatalf("expected recovered pending path /pending-a/one.bin, got %s", got)
	}
	if got := len(uploadCalls); got != 3 {
		t.Fatalf("expected 3 upload calls, got %d", got)
	}
	if got := int(recoverData["limitPerProfile"].(float64)); got != 1 {
		t.Fatalf("expected limitPerProfile 1, got %d", got)
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

func TestAppRecoverTasksEndpointFiltersWaitingRetryWindowState(t *testing.T) {
	ctx := context.Background()
	application := mustNewTestApp(t, ctx)

	uploadCalls := 0
	adapter := &appScriptedAdapter{
		meta: provider.Provider{
			Key:              "recover_waiting_window_api_target",
			DisplayName:      "Recover Waiting Window API Target",
			ProtocolGroup:    "recover_waiting_window_group",
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
			return provider.UploadResult{
				OperationResult: provider.OperationResult{
					Status:  "rate_limited",
					Message: "rate limited",
					Mode:    "fake_rate_limit",
				},
			}
		},
	}

	registry := provider.NewRegistry(adapter)
	authSvc := auth.NewService(application.store, registry)
	taskSvc := task.NewService(application.store, registry, authSvc)
	application.providers = registry
	application.auth = authSvc
	application.tasks = taskSvc
	handler := application.routes()

	profileResp := invokeJSON(t, handler, http.MethodPost, "/api/auth/profiles", map[string]interface{}{
		"providerKey": "recover_waiting_window_api_target",
		"authMode":    "manual_token",
		"displayName": "Recover Waiting Window API Target",
		"token":       "token-recover-waiting-window-api",
	})
	profileID := profileResp.Data.(map[string]interface{})["id"].(string)

	nowHour := time.Now().UTC().Hour()
	startHour := (nowHour + 2) % 24
	endHour := (startHour + 1) % 24
	if endHour == nowHour {
		endHour = (endHour + 1) % 24
	}

	taskResp := invokeJSON(t, handler, http.MethodPost, "/api/tasks", map[string]interface{}{
		"sourceProvider":  "recover_waiting_window_api_target",
		"targetProvider":  "recover_waiting_window_api_target",
		"targetProfileId": profileID,
		"thresholdMB":     1,
		"riskOverride": map[string]interface{}{
			"cooldownSeconds":    0,
			"autoRetryStartHour": startHour,
			"autoRetryEndHour":   endHour,
		},
		"entries": []map[string]interface{}{
			{"path": "/waiting-window.bin", "size": 1024, "md5": "waiting-window-md5"},
		},
	})
	taskID := taskResp.Data.(map[string]interface{})["task"].(map[string]interface{})["id"].(string)

	runResp := invokeJSON(t, handler, http.MethodPost, "/api/tasks/"+taskID+"/run", nil)
	if got := runResp.Data.(map[string]interface{})["task"].(map[string]interface{})["state"].(string); got != "blocked" {
		t.Fatalf("expected blocked task, got %s", got)
	}
	if _, err := application.store.DB().ExecContext(ctx, `UPDATE task_results SET created_at = ? WHERE task_id = ?`,
		time.Now().Add(-2*time.Hour).UTC().Format(time.RFC3339), taskID,
	); err != nil {
		t.Fatalf("update task_results created_at error = %v", err)
	}

	previewResp := invokeJSON(t, handler, http.MethodPost, "/api/tasks/recover", map[string]interface{}{
		"recoverState": "waiting_retry_window",
		"dryRun":       true,
		"limit":        1,
	})
	previewData := previewResp.Data.(map[string]interface{})
	if got := previewData["recoverState"].(string); got != "waiting_retry_window" {
		t.Fatalf("expected recoverState waiting_retry_window, got %s", got)
	}
	if got := int(previewData["matchedCount"].(float64)); got != 1 {
		t.Fatalf("expected matchedCount 1, got %d", got)
	}
	if got := int(previewData["recoveredCount"].(float64)); got != 0 {
		t.Fatalf("expected recoveredCount 0, got %d", got)
	}
	if got := int(previewData["skippedByRetryWindowWait"].(float64)); got != 1 {
		t.Fatalf("expected skippedByRetryWindowWait 1, got %d", got)
	}
	if decisions, ok := previewData["decisions"].([]interface{}); !ok || len(decisions) == 0 {
		t.Fatalf("expected preview decisions, got %#v", previewData["decisions"])
	} else if got := decisions[0].(map[string]interface{})["outcome"].(string); got != "waiting_retry_window" {
		t.Fatalf("expected preview outcome waiting_retry_window, got %s", got)
	} else {
		decision := decisions[0].(map[string]interface{})
		if got := decision["blockedReason"].(string); got != "retry_queue_waiting_for_retry_window" {
			t.Fatalf("expected waiting window blockedReason retry_queue_waiting_for_retry_window, got %s", got)
		}
		if got := decision["advice"].(string); !strings.Contains(got, "自动补传时间窗") {
			t.Fatalf("expected waiting window advice to mention auto retry window, got %s", got)
		}
	}
	if counts, ok := previewData["outcomeCounts"].(map[string]interface{}); !ok {
		t.Fatalf("expected waiting window outcomeCounts map, got %#v", previewData["outcomeCounts"])
	} else if got := int(counts["waiting_retry_window"].(float64)); got != 1 {
		t.Fatalf("expected waiting_retry_window count 1, got %d", got)
	}
	if counts, ok := previewData["retryClassCounts"].(map[string]interface{}); !ok {
		t.Fatalf("expected waiting window retryClassCounts map, got %#v", previewData["retryClassCounts"])
	} else if got := int(counts["rate_limited"].(float64)); got != 1 {
		t.Fatalf("expected rate_limited class count 1, got %d", got)
	}
	if counts, ok := previewData["recoverStateCounts"].(map[string]interface{}); !ok {
		t.Fatalf("expected waiting window recoverStateCounts map, got %#v", previewData["recoverStateCounts"])
	} else if got := int(counts["waiting_retry_window"].(float64)); got != 1 {
		t.Fatalf("expected waiting_retry_window state count 1, got %d", got)
	}
	if counts, ok := previewData["blockedActionCounts"].(map[string]interface{}); !ok {
		t.Fatalf("expected waiting window blockedActionCounts map, got %#v", previewData["blockedActionCounts"])
	} else if got := int(counts["wait_for_retry_window"].(float64)); got != 1 {
		t.Fatalf("expected wait_for_retry_window action count 1, got %d", got)
	}
	if counts, ok := previewData["protocolGroupCounts"].(map[string]interface{}); !ok {
		t.Fatalf("expected waiting window protocolGroupCounts map, got %#v", previewData["protocolGroupCounts"])
	} else if got := int(counts["recover_waiting_window_group"].(float64)); got != 1 {
		t.Fatalf("expected recover_waiting_window_group count 1, got %d", got)
	}
	if counts, ok := previewData["providerCounts"].(map[string]interface{}); !ok {
		t.Fatalf("expected waiting window providerCounts map, got %#v", previewData["providerCounts"])
	} else if got := int(counts["recover_waiting_window_api_target"].(float64)); got != 1 {
		t.Fatalf("expected recover_waiting_window_api_target count 1, got %d", got)
	}
	if counts, ok := previewData["profileCounts"].(map[string]interface{}); !ok {
		t.Fatalf("expected waiting window profileCounts map, got %#v", previewData["profileCounts"])
	} else if got := int(counts[profileID].(float64)); got != 1 {
		t.Fatalf("expected waiting window profile count 1, got %d", got)
	}
	if counts, ok := previewData["laneCounts"].(map[string]interface{}); !ok {
		t.Fatalf("expected waiting window laneCounts map, got %#v", previewData["laneCounts"])
	} else if got := int(counts["retry_window_waiting_auto_retry::rate_limited::wait_for_retry_window"].(float64)); got != 1 {
		t.Fatalf("expected waiting window lane count 1, got %d", got)
	}
	if got, ok := previewData["earliestNextRetryAt"].(string); !ok || got == "" {
		t.Fatalf("expected earliestNextRetryAt string, got %#v", previewData["earliestNextRetryAt"])
	}
	if uploadCalls != 1 {
		t.Fatalf("expected preview not to trigger extra upload, got %d", uploadCalls)
	}

	detailResp := invokeJSON(t, handler, http.MethodGet, "/api/tasks/"+taskID, nil)
	if got := detailResp.Data.(map[string]interface{})["task"].(map[string]interface{})["state"].(string); got != "blocked" {
		t.Fatalf("expected task to stay blocked after preview, got %s", got)
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
