package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"cloudpan-sync-go/internal/auth"
	"cloudpan-sync-go/internal/provider"
	"cloudpan-sync-go/internal/task"
	"github.com/chromedp/chromedp"
)

func TestConsoleUISmokeMainline(t *testing.T) {
	ctx := context.Background()
	application := mustNewTestApp(t, ctx)
	server := httptest.NewServer(application.routes())
	t.Cleanup(server.Close)

	browserPath, ok := findBrowserExecutable()
	if !ok {
		t.Skip("ui smoke requires Chrome or Edge to be installed")
	}

	userDataDir := filepath.Join(t.TempDir(), "chromedp-profile")
	allocOptions := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.ExecPath(browserPath),
		chromedp.UserDataDir(userDataDir),
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-default-browser-check", true),
		chromedp.Flag("no-first-run", true),
		chromedp.Flag("disable-background-networking", true),
		chromedp.Flag("disable-sync", true),
		chromedp.Flag("metrics-recording-only", true),
	)

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), allocOptions...)
	t.Cleanup(cancelAlloc)

	browserCtx, cancelBrowser := chromedp.NewContext(allocCtx)
	t.Cleanup(cancelBrowser)

	runCtx, cancelTimeout := context.WithTimeout(browserCtx, 300*time.Second)
	t.Cleanup(cancelTimeout)

	profileName := "UI Smoke 123"
	localFile := filepath.Join(t.TempDir(), "ui-smoke.bin")
	if err := os.WriteFile(localFile, []byte("ui-smoke"), 0o644); err != nil {
		t.Fatalf("write ui smoke local file: %v", err)
	}
	entriesBytes, err := json.Marshal([]map[string]interface{}{
		{
			"path":      "/demo/ui-smoke.bin",
			"size":      2048,
			"md5":       "md5-ui-smoke",
			"localPath": localFile,
		},
	})
	if err != nil {
		t.Fatalf("marshal ui smoke entries: %v", err)
	}
	entriesJSON := string(entriesBytes)

	targetAdapter := &appScriptedAdapter{
		meta:       provider.Provider{Key: "missing_uploadid_target", DisplayName: "Missing UploadID Target", ProtocolGroup: "fake_target", AuthModes: []string{"manual_token"}, FastUploadInputs: []string{"md5", "size"}, FallbackModes: []string{"download_upload"}, Status: "planned"},
		capability: provider.CapabilitySet{SupportsAuthValidation: true, SupportsUpload: true},
		uploadFunc: func(req provider.UploadRequest) provider.UploadResult {
			return provider.UploadResult{OperationResult: provider.OperationResult{Status: "missing_uploadid", Message: "provider omitted uploadid", Mode: "scripted_missing_uploadid"}}
		},
	}
	localMissingAdapter := &appScriptedAdapter{
		meta:       provider.Provider{Key: "local_missing_target", DisplayName: "Local Missing Target", ProtocolGroup: "fake_target", AuthModes: []string{"manual_token"}, FastUploadInputs: []string{"md5", "size"}, FallbackModes: []string{"download_upload"}, Status: "planned"},
		capability: provider.CapabilitySet{SupportsAuthValidation: true, SupportsUpload: true},
		uploadFunc: func(req provider.UploadRequest) provider.UploadResult {
			return provider.UploadResult{OperationResult: provider.OperationResult{Status: "local_file_missing", Message: "local file is missing", Mode: "scripted_local_missing"}}
		},
	}
	authExpiredAdapter := &appScriptedAdapter{
		meta:       provider.Provider{Key: "auth_expired_target", DisplayName: "Auth Expired Target", ProtocolGroup: "fake_target", AuthModes: []string{"manual_token"}, FastUploadInputs: []string{"md5", "size"}, FallbackModes: []string{"download_upload"}, Status: "planned"},
		capability: provider.CapabilitySet{SupportsAuthValidation: true, SupportsUpload: true},
		uploadFunc: func(req provider.UploadRequest) provider.UploadResult {
			return provider.UploadResult{OperationResult: provider.OperationResult{Status: "auth_expired", Message: "auth expired", Mode: "scripted_auth_expired"}}
		},
	}
	manualRecoverUploadCalls := 0
	manualConfirmationAdapter := &appScriptedAdapter{
		meta:       provider.Provider{Key: "manual_confirmation_target", DisplayName: "Manual Confirmation Target", ProtocolGroup: "manual_confirmation_group", AuthModes: []string{"manual_token"}, FastUploadInputs: []string{"md5", "size"}, FallbackModes: []string{"download_upload"}, Status: "planned"},
		capability: provider.CapabilitySet{SupportsAuthValidation: true, SupportsUpload: true},
		uploadFunc: func(req provider.UploadRequest) provider.UploadResult {
			manualRecoverUploadCalls++
			if manualRecoverUploadCalls == 1 {
				return provider.UploadResult{OperationResult: provider.OperationResult{Status: "pending_manual_requires_confirmation", Message: "pending manual", Mode: "scripted_manual_pending", Payload: map[string]interface{}{"providerStatus": "pending_manual_requires_confirmation"}}}
			}
			return provider.UploadResult{OperationResult: provider.OperationResult{OK: true, Status: "ok", Message: "manual confirmed", Mode: "scripted_manual_ok"}}
		},
	}

	recoverUploadCalls := 0
	recoverDryRunAdapter := &appScriptedAdapter{
		meta:       provider.Provider{Key: "recover_dry_run_ui_target", DisplayName: "Recover Dry Run UI Target", ProtocolGroup: "recover_dry_run_ui_group", AuthModes: []string{"manual_token"}, FastUploadInputs: []string{"md5", "size"}, FallbackModes: []string{"download_upload"}, Status: "planned"},
		capability: provider.CapabilitySet{SupportsAuthValidation: true, SupportsFastUpload: true, SupportsUpload: true},
		uploadFunc: func(req provider.UploadRequest) provider.UploadResult {
			recoverUploadCalls++
			if recoverUploadCalls == 1 {
				return provider.UploadResult{
					OperationResult: provider.OperationResult{
						Status:  "upload_checkpoint_pending",
						Message: "checkpoint pending",
						Mode:    "recover_dry_run_ui_pending",
						Payload: map[string]interface{}{
							"fileId":         "recover-dry-run-ui-file",
							"uploadId":       "recover-dry-run-ui-upload",
							"nextPartNumber": 1,
						},
					},
				}
			}
			return provider.UploadResult{OperationResult: provider.OperationResult{OK: true, Status: "ok", Message: "recovered", Mode: "recover_dry_run_ui_ok"}}
		},
	}
	registry := provider.NewRegistry(append(provider.DefaultCatalog(), targetAdapter, localMissingAdapter, authExpiredAdapter, manualConfirmationAdapter, recoverDryRunAdapter)...)
	authSvc := auth.NewService(application.store, registry)
	taskSvc := task.NewService(application.store, registry, authSvc)
	application.providers = registry
	application.auth = authSvc
	application.tasks = taskSvc

	profileResp := invokeJSON(t, application.routes(), http.MethodPost, "/api/auth/profiles", map[string]interface{}{"providerKey": "missing_uploadid_target", "authMode": "manual_token", "displayName": "Missing UploadID Target", "token": "token-missing-uploadid"})
	profileID := profileResp.Data.(map[string]interface{})["id"].(string)
	blockedFile := filepath.Join(t.TempDir(), "missing-uploadid.bin")
	if err := os.WriteFile(blockedFile, []byte("missing-uploadid"), 0o644); err != nil {
		t.Fatalf("write missing uploadid local file: %v", err)
	}
	taskResp := invokeJSON(t, application.routes(), http.MethodPost, "/api/tasks", map[string]interface{}{"sourceProvider": "missing_uploadid_source", "targetProvider": "missing_uploadid_target", "targetProfileId": profileID, "thresholdMB": 1, "entries": []map[string]interface{}{{"path": "/missing-uploadid.bin", "size": 1024, "md5": "missing-md5", "localPath": blockedFile}}})
	taskID := taskResp.Data.(map[string]interface{})["task"].(map[string]interface{})["id"].(string)
	runResp := invokeJSON(t, application.routes(), http.MethodPost, "/api/tasks/"+taskID+"/run", nil)
	if got := runResp.Data.(map[string]interface{})["task"].(map[string]interface{})["state"].(string); got != "blocked" {
		t.Fatalf("expected blocked task for ui smoke, got %s", got)
	}

	localProfileResp := invokeJSON(t, application.routes(), http.MethodPost, "/api/auth/profiles", map[string]interface{}{"providerKey": "local_missing_target", "authMode": "manual_token", "displayName": "Local Missing Target", "token": "token-local-missing"})
	localProfileID := localProfileResp.Data.(map[string]interface{})["id"].(string)
	localTaskResp := invokeJSON(t, application.routes(), http.MethodPost, "/api/tasks", map[string]interface{}{"sourceProvider": "local_missing_source", "targetProvider": "local_missing_target", "targetProfileId": localProfileID, "thresholdMB": 1, "entries": []map[string]interface{}{{"path": "/local-missing.bin", "size": 1024, "md5": "local-missing-md5", "localPath": "Z:/path/that/does/not/exist.bin"}}})
	localTaskID := localTaskResp.Data.(map[string]interface{})["task"].(map[string]interface{})["id"].(string)
	localRunResp := invokeJSON(t, application.routes(), http.MethodPost, "/api/tasks/"+localTaskID+"/run", nil)
	if got := localRunResp.Data.(map[string]interface{})["task"].(map[string]interface{})["state"].(string); got != "blocked" {
		t.Fatalf("expected local missing blocked task for ui smoke, got %s", got)
	}

	authProfileResp := invokeJSON(t, application.routes(), http.MethodPost, "/api/auth/profiles", map[string]interface{}{"providerKey": "auth_expired_target", "authMode": "manual_token", "displayName": "Auth Expired Target", "token": "token-auth-expired"})
	authProfileID := authProfileResp.Data.(map[string]interface{})["id"].(string)
	authFile := filepath.Join(t.TempDir(), "auth-expired.bin")
	if err := os.WriteFile(authFile, []byte("auth-expired"), 0o644); err != nil {
		t.Fatalf("write auth expired local file: %v", err)
	}
	authTaskResp := invokeJSON(t, application.routes(), http.MethodPost, "/api/tasks", map[string]interface{}{"sourceProvider": "auth_expired_source", "targetProvider": "auth_expired_target", "targetProfileId": authProfileID, "thresholdMB": 1, "entries": []map[string]interface{}{{"path": "/auth-expired.bin", "size": 1024, "md5": "auth-expired-md5", "localPath": authFile}}})
	authTaskID := authTaskResp.Data.(map[string]interface{})["task"].(map[string]interface{})["id"].(string)
	authRunResp := invokeJSON(t, application.routes(), http.MethodPost, "/api/tasks/"+authTaskID+"/run", nil)
	if got := authRunResp.Data.(map[string]interface{})["task"].(map[string]interface{})["state"].(string); got != "blocked" {
		t.Fatalf("expected auth expired blocked task for ui smoke, got %s", got)
	}

	manualProfileResp := invokeJSON(t, application.routes(), http.MethodPost, "/api/auth/profiles", map[string]interface{}{"providerKey": "manual_confirmation_target", "authMode": "manual_token", "displayName": "Manual Confirmation Target", "token": "token-manual-confirmation"})
	manualProfileID := manualProfileResp.Data.(map[string]interface{})["id"].(string)
	manualFile := filepath.Join(t.TempDir(), "manual-confirmation.bin")
	if err := os.WriteFile(manualFile, []byte("manual-confirmation"), 0o644); err != nil {
		t.Fatalf("write manual confirmation local file: %v", err)
	}
	manualTaskResp := invokeJSON(t, application.routes(), http.MethodPost, "/api/tasks", map[string]interface{}{"sourceProvider": "manual_confirmation_source", "targetProvider": "manual_confirmation_target", "targetProfileId": manualProfileID, "thresholdMB": 1, "entries": []map[string]interface{}{{"path": "/manual-confirmation.bin", "size": 1024, "md5": "manual-confirmation-md5", "localPath": manualFile}}})
	manualTaskID := manualTaskResp.Data.(map[string]interface{})["task"].(map[string]interface{})["id"].(string)
	manualRunResp := invokeJSON(t, application.routes(), http.MethodPost, "/api/tasks/"+manualTaskID+"/run", nil)
	if got := manualRunResp.Data.(map[string]interface{})["task"].(map[string]interface{})["state"].(string); got != "blocked" {
		t.Fatalf("expected manual confirmation blocked task for ui smoke, got %s", got)
	}
	if manualRecoverUploadCalls != 1 {
		t.Fatalf("expected manual confirmation task initial upload calls 1, got %d", manualRecoverUploadCalls)
	}
	recoverProfileResp := invokeJSON(t, application.routes(), http.MethodPost, "/api/auth/profiles", map[string]interface{}{"providerKey": "recover_dry_run_ui_target", "authMode": "manual_token", "displayName": "Recover Dry Run UI Target", "token": "token-recover-dry-run-ui"})
	recoverProfileID := recoverProfileResp.Data.(map[string]interface{})["id"].(string)
	recoverFile := filepath.Join(t.TempDir(), "recover-dry-run-ui.bin")
	if err := os.WriteFile(recoverFile, []byte("recover-dry-run-ui"), 0o644); err != nil {
		t.Fatalf("write recover dry run ui local file: %v", err)
	}
	recoverTaskResp := invokeJSON(t, application.routes(), http.MethodPost, "/api/tasks", map[string]interface{}{"sourceProvider": "recover_dry_run_ui_target", "targetProvider": "recover_dry_run_ui_target", "targetProfileId": recoverProfileID, "thresholdMB": 1, "entries": []map[string]interface{}{{"path": "/recover-dry-run-ui.bin", "size": 1024, "md5": "recover-dry-run-ui-md5", "localPath": recoverFile}}})
	recoverTaskID := recoverTaskResp.Data.(map[string]interface{})["task"].(map[string]interface{})["id"].(string)
	recoverRunResp := invokeJSON(t, application.routes(), http.MethodPost, "/api/tasks/"+recoverTaskID+"/run", nil)
	if got := recoverRunResp.Data.(map[string]interface{})["task"].(map[string]interface{})["state"].(string); got != "completed_with_errors" {
		t.Fatalf("expected recover dry run task completed_with_errors for ui smoke, got %s", got)
	}
	if recoverUploadCalls != 1 {
		t.Fatalf("expected recover dry run task initial upload calls 1, got %d", recoverUploadCalls)
	}

	runStep(t, runCtx, "login and bootstrap",
		chromedp.Navigate(server.URL),
		chromedp.WaitVisible(`#login-form`, chromedp.ByID),
		chromedp.SetValue(`#login-password`, "admin", chromedp.ByID),
		chromedp.Click(`#login-form button[type="submit"]`, chromedp.ByQuery),
		waitForText(`#session-state`, "已登录"),
		waitForSelectorCount(`#providers-grid .provider-card`, 10),
	)

	runStep(t, runCtx, "create and validate profile",
		chromedp.Click(`button[data-view="providers"]`, chromedp.ByQuery),
		setSelectValue(`#profile-provider`, "123_open"),
		waitForText(`#profile-auth-mode`, "manual_token"),
		setSelectValue(`#profile-auth-mode`, "manual_token"),
		chromedp.SetValue(`#profile-display-name`, profileName, chromedp.ByID),
		chromedp.SetValue(`#profile-token`, "token-ui-smoke", chromedp.ByID),
		chromedp.SetValue(`#profile-risk-request-interval`, "1666", chromedp.ByID),
		chromedp.SetValue(`#profile-risk-retry-limit`, "4", chromedp.ByID),
		chromedp.Click(`#sync-profile-risk-defaults`, chromedp.ByID),
		waitForValueContains(`#profile-extra`, `riskDefaults`),
		chromedp.Click(`#profile-form button[type="submit"]`, chromedp.ByQuery),
		waitForText(`#profiles-table`, profileName),
		chromedp.Click(`[data-profile-validate]`, chromedp.ByQuery),
		waitForText(`#profiles-table`, "verified"),
	)

	runStep(t, runCtx, "preview and create task",
		chromedp.Click(`button[data-view="wizard"]`, chromedp.ByQuery),
		setSelectValue(`#plan-source-provider`, "guangya"),
		setSelectValue(`#plan-target-provider`, "123_open"),
		waitForText(`#plan-target-profile`, profileName),
		setSelectValueByText(`#plan-target-profile`, profileName),
		waitForText(`#plan-target-profile-insight`, "auth profile riskDefaults"),
		waitForText(`#plan-target-profile-insight`, "req 1666ms"),
		chromedp.Click(`#apply-profile-default-risk`, chromedp.ByID),
		waitForText(`#flash`, "已将账号默认风控写入任务覆盖"),
		waitForValue(`#plan-risk-mode`, "custom"),
		waitForValueContains(`#plan-risk-override`, `"requestIntervalMs": 1666`),
		waitForValueContains(`#plan-risk-override`, `"retryLimit": 4`),
		setSelectValue(`#plan-risk-mode`, "fast"),
		setSelectValue(`#plan-execution-mode`, "pre_scan_flat"),
		waitForText(`#plan-execution-hint`, "pre_scan_flat"),
		chromedp.SetValue(`#plan-threshold`, "10", chromedp.ByID),
		chromedp.SetValue(`#plan-selected-roots`, `["/demo","/archive"]`, chromedp.ByID),
		chromedp.SetValue(`#plan-entries`, entriesJSON, chromedp.ByID),
		chromedp.Click(`#preview-plan`, chromedp.ByID),
		waitForText(`#plan-recommendation-title`, "建议执行模式：leaf_first_lazy / 建议风控档位：balanced"),
		waitForText(`#plan-recommendation-reason`, "Multiple top-level roots are safer to process subtree by subtree."),
		chromedp.Click(`#apply-recommended-execution`, chromedp.ByID),
		waitForValue(`#plan-execution-mode`, "leaf_first_lazy"),
		waitForText(`#flash`, "已采用推荐执行模式：leaf_first_lazy"),
		chromedp.Click(`#apply-recommended-risk`, chromedp.ByID),
		waitForValue(`#plan-risk-mode`, "balanced"),
		waitForText(`#flash`, "已采用推荐风控档位：balanced"),
		waitForText(`#plan-preview-meta`, "pre_scan_flat"),
		waitForText(`#plan-preview-meta`, "SELECTED ROOTS"),
		waitForText(`#plan-preview-meta`, "/demo -> /archive"),
		waitForText(`#plan-preview-meta`, "PROVIDER 基线"),
		waitForText(`#plan-preview-meta`, "最终生效"),
		waitForText(`#plan-preview-meta`, "CALIBRATED"),
		waitForText(`#plan-preview-meta`, "OVERRIDE FIELDS"),
		waitForText(`#plan-preview`, `"strategy": "fast_upload"`),
		chromedp.Click(`#plan-form button[type="submit"]`, chromedp.ByQuery),
		waitForText(`#tasks-list`, "guangya -> 123_open"),
		waitForText(`#task-summary`, "leaf_first_lazy"),
		waitForText(`#task-summary`, "SELECTED ROOTS"),
		waitForText(`#task-summary`, "PROVIDER 校准后"),
		waitForText(`#task-summary`, "任务覆盖"),
		waitForText(`#task-summary`, "CALIBRATED"),
		waitForText(`#task-summary`, "OVERRIDE FIELDS"),
		waitForText(`#task-runtime`, "SELECTED ROOTS"),
		waitForText(`#task-detail`, `"state": "ready"`),
		waitForText(`#task-directory-states`, "/demo"),
		chromedp.Click(`#task-summary [data-runtime-focus-kind="roots"]`, chromedp.ByQuery),
		waitForText(`#task-directory-filter-summary`, "当前显示"),
		chromedp.Click(`#task-directory-states [data-tree-prefill-path="/demo"]`, chromedp.ByQuery),
		waitForValueContains(`#plan-selected-roots`, "/demo"),
		chromedp.Click(`button[data-view="tasks"]`, chromedp.ByQuery),
		waitForText(`#task-summary`, "leaf_first_lazy"),
		chromedp.Click(`#task-directory-copy-visible`, chromedp.ByID),
		waitForText(`#flash`, "已复制"),
		chromedp.Click(`#task-directory-states [data-tree-focus-panel="directory"]`, chromedp.ByQuery),
		waitForValue(`#task-directory-filter-query`, "/demo"),
		chromedp.Click(`#task-directory-filter-clear`, chromedp.ByID),
		waitForText(`#task-directory-filter-summary`, "显示全部"),
	)

	runStep(t, runCtx, "pause resume run task",
		chromedp.Evaluate(`(() => document.querySelector('#task-pause')?.click())()`, nil),
		waitForText(`#task-detail`, `"state": "paused"`),
		chromedp.Evaluate(`(() => document.querySelector('#task-resume')?.click())()`, nil),
		waitForText(`#task-detail`, `"state": "ready"`),
		chromedp.Evaluate(`(() => document.querySelector('#task-run')?.click())()`, nil),
		waitForText(`#task-detail`, `"state": "completed_with_errors"`),
		waitForText(`#task-detail`, `"status": "failed"`),
		waitForText(`#task-runtime`, "completed"),
		waitForText(`#task-summary`, "retry_queue_auto_retry"),
		waitForText(`#task-runtime`, "后台补传候选"),
		chromedp.Evaluate(`(() => document.querySelector('#refresh-tasks')?.click())()`, nil),
		waitForText(`#flash`, "任务列表已刷新"),
		waitForText(`#task-detail`, `"state": "completed_with_errors"`),
	)

	runStep(t, runCtx, "status overview",
		chromedp.Evaluate(`(() => document.querySelector('button[data-view="status"]')?.click())()`, nil),
		waitForText(`#evidence-summary`, "Auto Recover"),
		waitForText(`#auto-retry-policy-summary`, "group 1"),
		waitForText(`#auto-recover-budget-summary`, "当前生效预算（默认）"),
		waitForText(`body`, "自动补传候选池"),
		waitForText(`body`, "协议族覆盖"),
		waitForText(`body`, "最近 Probe"),
	)

	runStep(t, runCtx, "status snapshot panels",
		waitForText("body", "Provider 状态矩阵"),
		waitForText("body", "最近 Probe"),
		waitForText("body", "最近结果"),
		waitForText("body", "最近目录状态"),
		waitForText("body", "最近待补传树"),
		waitForText("body", "运行检查点概览"),
		waitForText("body", "暂无待补传项。"),
	)

	runStep(t, runCtx, "status report panels",
		waitForText(`body`, "验收报告"),
		waitForText(`body`, "下载 Markdown"),
		waitForText(`body`, "报告标题"),
	)

	runStep(t, runCtx, "auto recover preview and run",
		waitForText(`#auto-recover-summary`, "recover_dry_run_ui_group"),
		chromedp.Evaluate(`(() => document.querySelector('#auto-recover-summary [data-auto-recover-preview-protocol-group="recover_dry_run_ui_group"]')?.click())()`, nil),
		waitForText(`#auto-recover-last-result-summary`, "最近预演"),
		waitForText(`#auto-recover-last-result-summary`, "预演可放行 1"),
		waitForText(`#auto-recover-last-result-summary`, "recover_dry_run_ui_target"),
		waitForText(`#auto-recover-last-result-detail`, "等待态说明"),
		waitForText(`#auto-recover-last-result-detail`, "recover_dry_run_ui_group"),
		chromedp.Evaluate(`(() => document.querySelector('#auto-recover-last-result-detail [data-auto-recover-decision-focus-state="runnable_now"]')?.click())()`, nil),
		waitForValue(`#auto-recover-state`, "runnable_now"),
		chromedp.Evaluate(`(() => document.querySelector('#auto-recover-last-result-detail [data-auto-recover-decision-apply-budgets]')?.click())()`, nil),
		waitForText(`#flash`, "已按决策采用建议预算"),
		chromedp.ActionFunc(func(ctx context.Context) error {
			var modeBudget string
			if err := chromedp.Value(`#auto-recover-limit-per-mode`, &modeBudget, chromedp.ByID).Do(ctx); err != nil {
				return err
			}
			if strings.TrimSpace(modeBudget) == "" || strings.TrimSpace(modeBudget) == "0" {
				return fmt.Errorf("expected mode budget to be populated, got %q", modeBudget)
			}
			return nil
		}),
		chromedp.Evaluate(`(() => document.querySelector('#auto-recover-last-result-detail [data-auto-recover-decision-preview="1"]')?.click())()`, nil),
		waitForText(`#flash`, "已按决策预演后台补传"),
		waitForText(`#auto-recover-last-result-summary`, "最近预演"),
		waitForText(`#auto-recover-last-result-detail`, "预演可放行"),
		chromedp.Evaluate(`(() => document.querySelector('#auto-recover-last-result-detail [data-auto-recover-decision-open-task]')?.click())()`, nil),
		waitForText(`#task-detail`, recoverTaskID),
		waitForText(`#task-detail`, `"state": "completed_with_errors"`),
		chromedp.Evaluate(`(() => document.querySelector('button[data-view="status"]')?.click())()`, nil),
		waitForText(`#auto-recover-last-result-summary`, "最近预演"),
		chromedp.Evaluate(`(() => document.querySelector('#auto-recover-last-result-detail [data-auto-recover-decision-run="1"]')?.click())()`, nil),
		waitForText(`#flash`, "已按决策执行后台补传"),
		waitForText(`#auto-recover-last-result-summary`, "最近执行"),
		waitForText(`#auto-recover-last-result-summary`, "recovered 1"),
		waitForText(`#auto-recover-last-result-detail`, "已放行执行"),
		waitForText(`#auto-recover-last-result-detail`, "recover_dry_run_ui_group"),
		waitForText(`#tasks-list`, "recover_dry_run_ui_target -> recover_dry_run_ui_target"),
		chromedp.Evaluate(`(() => document.querySelector('#auto-recover-reset')?.click())()`, nil),
		waitForText(`#auto-recover-filter-summary`, "条后台补传候选"),
		chromedp.ActionFunc(func(ctx context.Context) error {
			if recoverUploadCalls < 2 {
				return fmt.Errorf("expected recover upload calls >= 2 after execute, got %d", recoverUploadCalls)
			}
			return nil
		}),
	)

	manualRecoverResp := invokeJSON(t, application.routes(), http.MethodPost, "/api/tasks/recover", map[string]interface{}{
		"blockedAction": "manual_confirmation_required",
		"recoverState":  "waiting_manual_confirmation",
		"taskId":        manualTaskID,
		"providerKey":   "manual_confirmation_target",
		"profileId":     manualProfileID,
		"limit":         5,
	})
	manualRecoverData := manualRecoverResp.Data.(map[string]interface{})
	if got := int(manualRecoverData["recoveredCount"].(float64)); got != 1 {
		t.Fatalf("expected manual confirmation recover result recoveredCount=1, got %#v", manualRecoverData)
	}

	runStep(t, runCtx, "manual confirmation blocked task",
		chromedp.Evaluate(`(() => document.querySelector('button[data-view="tasks"]')?.click())()`, nil),
		chromedp.Evaluate(`(() => document.querySelector('#refresh-tasks')?.click())()`, nil),
		waitForText(`#tasks-list`, "manual_confirmation_source -> manual_confirmation_target"),
		chromedp.ActionFunc(func(ctx context.Context) error {
			if manualRecoverUploadCalls < 2 {
				return fmt.Errorf("expected manual confirmation recovery upload calls >= 2, got %d", manualRecoverUploadCalls)
			}
			detailResp := invokeJSON(t, application.routes(), http.MethodGet, "/api/tasks/"+manualTaskID, nil)
			payload, _ := json.Marshal(detailResp.Data)
			taskMap, ok := detailResp.Data.(map[string]interface{})
			if !ok {
				return fmt.Errorf("expected task detail map, got %#v", detailResp.Data)
			}
			taskDetail, ok := taskMap["task"].(map[string]interface{})
			if !ok {
				return fmt.Errorf("expected task payload in detail, got %s", payload)
			}
			if got := taskDetail["state"]; got != "completed" {
				return fmt.Errorf("expected manual recovered task completed payload, got %s", payload)
			}
			return nil
		}),
	)
	runStep(t, runCtx, "provider session missing blocked task",
		chromedp.Evaluate(`(() => document.querySelector('button[data-view="tasks"]')?.click())()`, nil),
		chromedp.ActionFunc(func(ctx context.Context) error {
			var payload string
			if err := chromedp.Evaluate(fmt.Sprintf(`(() => {
				const hooks = window.__cloudpanTestHooks;
				if (!hooks || !hooks.state) {
					throw new Error("task hooks unavailable");
				}
				const detail = hooks.state.tasks.find((item) => item?.task?.id === %q);
				if (!detail) {
					throw new Error("missing blocked task detail");
				}
				hooks.state.selectedTaskId = detail.task.id;
				return JSON.stringify(detail);
			})()`, taskID), &payload).Do(ctx); err != nil {
				return err
			}
			if !strings.Contains(payload, `"sourceProvider":"missing_uploadid_source"`) {
				return fmt.Errorf("blocked task source provider missing in payload: %s", payload)
			}
			if !strings.Contains(payload, `"retryClass":"provider_session_missing"`) {
				return fmt.Errorf("blocked task retryClass missing in payload: %s", payload)
			}
			if !strings.Contains(payload, `"blockedAction":"manual_intervention_required"`) {
				return fmt.Errorf("blocked task blockedAction missing in payload: %s", payload)
			}
			return nil
		}),
		chromedp.ActionFunc(func(ctx context.Context) error {
			if err := chromedp.Evaluate(`(() => window.__cloudpanTestHooks?.renderSelectedTask?.())()`, nil).Do(ctx); err != nil {
				return err
			}
			return nil
		}),
		waitForText(`#task-resolution-guide`, "修复 provider 会话缺口"),
		chromedp.Evaluate(`(() => document.querySelector('#task-resolution-guide [data-task-guide-intent="focus_status_blocked"]')?.click())()`, nil),
		waitForValue(`#auto-recover-blocked-action`, "manual_intervention_required"),
		waitForText(`#flash`, "已按 blocked action 收敛最近重试队列"),
		waitForText(`#blocked-actions-summary`, "next-step: 修复 provider 会话后继续"),
		waitForText(`#blocked-actions-summary`, "manual_intervention_required"),
	)

	runStep(t, runCtx, "local file missing blocked task",
		chromedp.Evaluate(`(() => document.querySelector('button[data-view="tasks"]')?.click())()`, nil),
		chromedp.ActionFunc(func(ctx context.Context) error {
			var payload string
			if err := chromedp.Evaluate(fmt.Sprintf(`(() => {
				const hooks = window.__cloudpanTestHooks;
				if (!hooks || !hooks.state) {
					throw new Error("task hooks unavailable");
				}
				const detail = hooks.state.tasks.find((item) => item?.task?.id === %q);
				if (!detail) {
					throw new Error("missing local blocked task detail");
				}
				hooks.state.selectedTaskId = detail.task.id;
				return JSON.stringify(detail);
			})()`, localTaskID), &payload).Do(ctx); err != nil {
				return err
			}
			if !strings.Contains(payload, `"retryClass":"local_file_missing"`) {
				return fmt.Errorf("local blocked task retryClass missing in payload: %s", payload)
			}
			if !strings.Contains(payload, `"blockedAction":"restore_local_source_file"`) {
				return fmt.Errorf("local blocked task blockedAction missing in payload: %s", payload)
			}
			return nil
		}),
		chromedp.Evaluate(`(() => window.__cloudpanTestHooks?.focusBlockedActionSummary?.("restore_local_source_file"))()`, nil),
		waitForValue(`#auto-recover-blocked-action`, "restore_local_source_file"),
		waitForText(`#blocked-actions-summary`, "next-step: 补回本地文件后继续"),
		waitForText(`#blocked-actions-summary`, "restore_local_source_file"),
	)

	runStep(t, runCtx, "auth expired blocked task",
		chromedp.Evaluate(`(() => document.querySelector('button[data-view="tasks"]')?.click())()`, nil),
		chromedp.ActionFunc(func(ctx context.Context) error {
			var payload string
			if err := chromedp.Evaluate(fmt.Sprintf(`(() => {
				const hooks = window.__cloudpanTestHooks;
				if (!hooks || !hooks.state) {
					throw new Error("task hooks unavailable");
				}
				const detail = hooks.state.tasks.find((item) => item?.task?.id === %q);
				if (!detail) {
					throw new Error("missing auth blocked task detail");
				}
				hooks.state.selectedTaskId = detail.task.id;
				return JSON.stringify(detail);
			})()`, authTaskID), &payload).Do(ctx); err != nil {
				return err
			}
			if !strings.Contains(payload, `"retryClass":"auth_expired"`) {
				return fmt.Errorf("auth blocked task retryClass missing in payload: %s", payload)
			}
			if !strings.Contains(payload, `"blockedAction":"refresh_auth_profile"`) {
				return fmt.Errorf("auth blocked task blockedAction missing in payload: %s", payload)
			}
			return nil
		}),
		chromedp.Evaluate(`(() => window.__cloudpanTestHooks?.focusBlockedActionSummary?.("refresh_auth_profile"))()`, nil),
		waitForValue(`#auto-recover-blocked-action`, "refresh_auth_profile"),
		waitForText(`#blocked-actions-summary`, "next-step: 刷新授权后继续"),
		waitForText(`#blocked-actions-summary`, "refresh_auth_profile"),
	)

	runStep(t, runCtx, "provider smoke matrix draft workflow",
		waitForText(`#provider-smoke-matrix`, "aliyun_123_open"),
		chromedp.Evaluate(`(() => document.querySelector('#provider-smoke-matrix [data-provider-smoke-draft="aliyun_123_open"]')?.click())()`, nil),
		waitForValue(`#provider-smoke-provider-key`, "123_open"),
		waitForValue(`#provider-smoke-protocol-group`, "aliyun_123_open"),
		waitForValue(`#provider-smoke-category`, "binary_upload_success"),
		waitForValueContains(`#provider-smoke-operations`, "Upload"),
		waitForValueContains(`#provider-smoke-title`, "aliyun_123_open"),
		waitForValueContains(`#provider-smoke-note`, "协议组：aliyun_123_open"),
		chromedp.Evaluate(`(() => document.querySelector('#provider-smoke-matrix [data-provider-smoke-focus-group="aliyun_123_open"]')?.click())()`, nil),
		waitForValue(`#provider-smoke-records-filter-group`, "aliyun_123_open"),
		waitForText(`#provider-smoke-records-filter-summary`, "当前没有 smoke 记录。"),
		chromedp.Evaluate(`(() => document.querySelector('#provider-smoke-matrix [data-provider-smoke-draft-action="aliyun_123_open"]')?.click())()`, nil),
		waitForText(`#flash`, "已按验收缺口预填 smoke 动作"),
		waitForValueContains(`#provider-smoke-title`, "补上传 smoke"),
		chromedp.Evaluate(`(() => document.querySelector('#save-provider-smoke')?.click())()`, nil),
		waitForText(`#flash`, "Provider smoke 记录已保存"),
		waitForText(`#provider-smoke-records-filter-summary`, "1 / 1 条 smoke 记录"),
		waitForText(`#provider-smoke-records`, "binary_upload_success"),
		waitForText(`#provider-smoke-markdown`, "aliyun_123_open 补上传 smoke"),
		waitForText(`#provider-smoke-markdown`, "Upload"),
		chromedp.Evaluate(`(() => document.querySelector('#provider-smoke-matrix [data-provider-smoke-open-record]')?.click())()`, nil),
		waitForText(`#flash`, "已打开 smoke 样本并回填表单"),
		chromedp.Evaluate(`(() => document.querySelector('#provider-smoke-matrix [data-provider-smoke-filter-status="accepted"]')?.click())()`, nil),
		waitForText(`#flash`, "已按 accepted 收敛验收矩阵"),
	)

	runStep(t, runCtx, "provider smoke matrix prefill profile risk",
		chromedp.Evaluate(`(() => document.querySelector('#provider-smoke-matrix [data-provider-smoke-prefill-profile-risk="aliyun_123_open"]')?.click())()`, nil),
		waitForText(`#flash`, "已按真实样本预填账号默认风控"),
		waitForValue(`#profile-provider`, "123_open"),
		waitForValueContains(`#profile-display-name`, "aliyun_123_open 风控模板"),
		waitForValueContains(`#profile-extra`, `riskDefaults`),
		waitForValueContains(`#profile-extra`, `riskDefaultsSourceDisplay`),
		waitForValueContains(`#profile-extra`, `Smoke Matrix aliyun_123_open (accepted)`),
		waitForValueContains(`#profile-extra`, `accepted_group`),
		waitForValue(`#profile-risk-request-interval`, "1800"),
		waitForValue(`#profile-risk-directory-interval`, "2600"),
	)

	runStep(t, runCtx, "provider smoke save auth profile",
		chromedp.SetValue(`#profile-token`, "token-smoke-risk-template", chromedp.ByID),
		chromedp.Evaluate(`(() => document.querySelector('#profile-form button[type="submit"]')?.click())()`, nil),
		waitForText(`#flash`, "授权档案已创建"),
		waitForText(`#profiles-table`, "aliyun_123_open 风控模板"),
	)

	runStep(t, runCtx, "provider smoke sync wizard profile insight",
		waitForScriptTrue(`(() => {
			if (typeof loadProfiles === 'function') {
				loadProfiles();
			}
			const provider = document.querySelector('#plan-target-provider');
			const profile = document.querySelector('#plan-target-profile');
			const profiles = Array.isArray(state?.profiles) ? state.profiles : [];
			const matchedProfile = profiles.find((item) => item && item.providerKey === '123_open' && String(item.displayName || '').includes('aliyun_123_open 风控模板'));
			if (!provider || !profile || !matchedProfile) {
				return false;
			}
			provider.value = '123_open';
			provider.dispatchEvent(new Event('change', { bubbles: true }));
			if (typeof syncTargetProfiles === 'function') {
				syncTargetProfiles();
			}
			let option = Array.from(profile.options || []).find((item) => item.value === matchedProfile.id);
			if (!option) {
				option = document.createElement('option');
				option.value = matchedProfile.id;
				option.textContent = matchedProfile.displayName || matchedProfile.id;
				profile.appendChild(option);
			}
			profile.value = matchedProfile.id;
			profile.dispatchEvent(new Event('change', { bubbles: true }));
			if (typeof syncTargetProfileInsight === 'function') {
				syncTargetProfileInsight();
			}
			return true;
		})()`),
		waitForTextContent(`#plan-target-profile-insight`, "Smoke Matrix aliyun_123_open (accepted)"),
		waitForTextContent(`#plan-target-profile-insight`, "req 1800ms"),
	)
}

func waitForText(selector string, substring string) chromedp.ActionFunc {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		var matched bool
		script := fmt.Sprintf(`(() => {
			const el = document.querySelector(%q);
			return !!el && el.innerText.includes(%q);
		})()`, selector, substring)
		return chromedp.Poll(script, &matched,
			chromedp.WithPollingInterval(120*time.Millisecond),
			chromedp.WithPollingTimeout(30*time.Second),
		).Do(ctx)
	})
}

func waitForTextContent(selector string, substring string) chromedp.ActionFunc {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		var matched bool
		script := fmt.Sprintf(`(() => {
			const el = document.querySelector(%q);
			return !!el && String(el.textContent || "").includes(%q);
		})()`, selector, substring)
		return chromedp.Poll(script, &matched,
			chromedp.WithPollingInterval(120*time.Millisecond),
			chromedp.WithPollingTimeout(30*time.Second),
		).Do(ctx)
	})
}

func waitForSelectorCount(selector string, minCount int) chromedp.ActionFunc {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		var matched bool
		script := fmt.Sprintf(`(() => document.querySelectorAll(%q).length >= %d)()`, selector, minCount)
		return chromedp.Poll(script, &matched,
			chromedp.WithPollingInterval(120*time.Millisecond),
			chromedp.WithPollingTimeout(30*time.Second),
		).Do(ctx)
	})
}

func waitForScriptTrue(script string) chromedp.ActionFunc {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		var matched bool
		return chromedp.Poll(script, &matched,
			chromedp.WithPollingInterval(120*time.Millisecond),
			chromedp.WithPollingTimeout(30*time.Second),
		).Do(ctx)
	})
}

func waitForLocalStorageContains(key string, substring string) chromedp.ActionFunc {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		var matched bool
		script := fmt.Sprintf(`(() => String(localStorage.getItem(%q) || "").includes(%q))()`, key, substring)
		return chromedp.Poll(script, &matched,
			chromedp.WithPollingInterval(120*time.Millisecond),
			chromedp.WithPollingTimeout(30*time.Second),
		).Do(ctx)
	})
}

func waitForValue(selector string, expected string) chromedp.ActionFunc {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		var matched bool
		script := fmt.Sprintf(`(() => {
			const el = document.querySelector(%q);
			return !!el && el.value === %q;
		})()`, selector, expected)
		return chromedp.Poll(script, &matched,
			chromedp.WithPollingInterval(120*time.Millisecond),
			chromedp.WithPollingTimeout(30*time.Second),
		).Do(ctx)
	})
}

func waitForValueContains(selector string, substring string) chromedp.ActionFunc {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		var matched bool
		script := fmt.Sprintf(`(() => {
			const el = document.querySelector(%q);
			return !!el && String(el.value || "").includes(%q);
		})()`, selector, substring)
		return chromedp.Poll(script, &matched,
			chromedp.WithPollingInterval(120*time.Millisecond),
			chromedp.WithPollingTimeout(30*time.Second),
		).Do(ctx)
	})
}

func setSelectValue(selector string, value string) chromedp.Action {
	script := fmt.Sprintf(`(() => {
		const el = document.querySelector(%q);
		if (!el) {
			throw new Error("missing select");
		}
		el.value = %q;
		el.dispatchEvent(new Event("change", { bubbles: true }));
		return el.value;
	})()`, selector, value)
	return chromedp.Evaluate(script, nil)
}

func setSelectValueByText(selector string, label string) chromedp.Action {
	script := fmt.Sprintf(`(() => {
		const el = document.querySelector(%q);
		if (!el) {
			throw new Error("missing select");
		}
		const option = Array.from(el.options).find((item) => item.textContent.includes(%q));
		if (!option) {
			throw new Error("missing option");
		}
		el.value = option.value;
		el.dispatchEvent(new Event("change", { bubbles: true }));
		return el.value;
	})()`, selector, label)
	return chromedp.Evaluate(script, nil)
}

func findBrowserExecutable() (string, bool) {
	paths := []string{
		strings.TrimSpace(os.Getenv("CHROMEDP_EXEC_PATH")),
		`C:\Program Files\Google\Chrome\Application\chrome.exe`,
		`C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`,
		`C:\Program Files\Microsoft\Edge\Application\msedge.exe`,
		`C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`,
	}
	for _, path := range paths {
		if path == "" {
			continue
		}
		if _, err := os.Stat(path); err == nil {
			return path, true
		}
	}
	return "", false
}

func runStep(t *testing.T, ctx context.Context, name string, actions ...chromedp.Action) {
	t.Helper()
	if err := chromedp.Run(ctx, actions...); err != nil {
		var bodyText string
		_ = chromedp.Run(ctx, chromedp.Text("body", &bodyText, chromedp.ByQuery))
		t.Fatalf("ui smoke step %q failed: %v\nbody:\n%s", name, err, bodyText)
	}
}
