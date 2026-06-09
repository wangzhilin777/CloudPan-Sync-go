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
	sourceProfileName := "UI Smoke Source 123"
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
		{
			"path":         "/demo/deleted.bin",
			"deleted":      true,
			"deletedAt":    "2026-05-29T10:00:00Z",
			"deleteReason": "source_removed",
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
	sourceWizardProfileResp := invokeJSON(t, application.routes(), http.MethodPost, "/api/auth/profiles", map[string]interface{}{"providerKey": "guangya", "authMode": "manual_token", "displayName": sourceProfileName, "token": "token-ui-smoke-source"})
	sourceWizardProfileID := sourceWizardProfileResp.Data.(map[string]interface{})["id"].(string)
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
		chromedp.Evaluate(`(() => {
			const el = document.querySelector('#language-select');
			if (!el) {
				return false;
			}
			el.value = 'en-US';
			el.dispatchEvent(new Event('change', { bubbles: true }));
			return true;
		})()`, nil),
		waitForText(`body`, "A multi-cloud transfer console focused on tasks, status, and evidence."),
		waitForText(`#login-form button[type="submit"]`, "Sign In"),
		waitForText(`#session-state`, "Signed Out"),
		chromedp.Evaluate(`(() => {
			const el = document.querySelector('#language-select');
			if (!el) {
				return false;
			}
			el.value = 'zh-CN';
			el.dispatchEvent(new Event('change', { bubbles: true }));
			return true;
		})()`, nil),
		waitForText(`#login-form button[type="submit"]`, "验证登录"),
		waitForText(`#session-state`, "未登录"),
		chromedp.SetValue(`#login-password`, "admin", chromedp.ByID),
		chromedp.Click(`#login-form button[type="submit"]`, chromedp.ByQuery),
		waitForText(`#session-state`, "已登录"),
		waitForSelectorCount(`#providers-grid .provider-card`, 10),
	)

	runStep(t, runCtx, "create and validate profile",
		chromedp.Click(`button[data-view="providers"]`, chromedp.ByQuery),
		setSelectValue(`#language-select`, "en-US"),
		waitForText(`body`, "Auth Profiles"),
		waitForText(`#profile-assist-use-openlist`, "Prefer OpenList"),
		waitForText(`#profile-submit`, "Create Auth Profile"),
		setSelectValue(`#language-select`, "zh-CN"),
		waitForText(`body`, "授权档案"),
		waitForText(`#profile-assist-summary`, "当前优先走 OpenList"),
		chromedp.Evaluate(`(() => {
			if (typeof syncAuthAssistDiscovery !== 'function') {
				return false;
			}
			syncAuthAssistDiscovery({
				kind: 'openlist',
				baseUrl: 'http://127.0.0.1:5244',
				reachable: true,
				storages: [
					{
						id: 'storage-openlist-main',
						name: 'OpenList 主存储',
						driver: 'WebDAV',
						mountPath: '/dav',
						status: 'work',
					},
				],
			});
			return true;
		})()`, nil),
		waitForText(`#profile-assist-discovery`, "OpenList 已连通"),
		chromedp.Click(`[data-assist-select-index="0"]`, chromedp.ByQuery),
		waitForValue(`#profile-display-name`, "OpenList 主存储"),
		waitForValueContains(`#profile-extra`, `"assistKind": "openlist"`),
		waitForValueContains(`#profile-extra`, `"assistStorageMountPath": "/dav"`),
		waitForText(`#flash`, "已从 OpenList 回填存储"),
		setSelectValue(`#profile-provider`, "123_open"),
		waitForText(`#profile-auth-guide`, "domainId"),
		waitForText(`#profile-auth-guide`, "driveId"),
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
		waitForText(`#profiles-table`, "网盘源"),
		waitForText(`#profiles-table`, "状态"),
		waitForText(`#profiles-table`, "手动令牌（manual_token）"),
		waitForText(`#profiles-table`, "已保存（saved）"),
		waitForText(`#profiles-table`, "验证授权"),
		chromedp.Click(`[data-profile-validate]`, chromedp.ByQuery),
		waitForText(`#profiles-table`, "已校验（verified）"),
	)

	runStep(t, runCtx, "preview and create task",
		chromedp.Click(`button[data-view="wizard"]`, chromedp.ByQuery),
		setSelectValue(`#language-select`, "en-US"),
		waitForText(`body`, "Task Preview / Create"),
		waitForText(`body`, "Source Provider"),
		waitForText(`body`, "Target Provider"),
		waitForText(`body`, "Source Directory"),
		waitForText(`body`, "Use Current Directory"),
		waitForText(`body`, "Target Directory"),
		waitForText(`body`, "Preview Result"),
		setSelectValue(`#language-select`, "zh-CN"),
		waitForText(`body`, "任务预览 / 创建"),
		setSelectValue(`#plan-source-provider`, "guangya"),
		waitForText(`#plan-source-profile`, sourceProfileName),
		setSelectValue(`#plan-source-profile`, sourceWizardProfileID),
		setSelectValue(`#plan-target-provider`, "123_open"),
		waitForText(`#plan-target-provider-insight`, "推荐风控档位"),
		waitForText(`#plan-target-provider-insight`, "平衡（balanced）"),
		waitForText(`#plan-target-provider-insight`, "网盘源默认模板"),
		waitForText(`#plan-target-profile`, profileName),
		setSelectValueByText(`#plan-target-profile`, profileName),
		waitForText(`#plan-target-profile-insight`, "123_open / 手动令牌（manual_token）"),
		waitForText(`#plan-target-profile-insight`, "授权档案内置账号默认风控"),
		waitForText(`#plan-target-profile-insight`, "附加配置项"),
		waitForText(`#plan-target-profile-insight`, "已启用字段"),
		waitForText(`#plan-target-profile-insight`, "req 1666ms"),
		chromedp.Click(`#apply-profile-default-risk`, chromedp.ByID),
		waitForText(`#flash`, "已将账号默认风控写入任务覆盖"),
		waitForValue(`#plan-risk-mode`, "custom"),
		waitForValueContains(`#plan-risk-override`, `"requestIntervalMs": 1666`),
		waitForValueContains(`#plan-risk-override`, `"retryLimit": 4`),
		setSelectValue(`#plan-risk-mode`, "fast"),
		setSelectValue(`#plan-execution-mode`, "pre_scan_flat"),
		waitForText(`#plan-execution-hint`, "先完整扫描再执行（pre_scan_flat）"),
		waitForText(`#plan-source-browser-level`, "当前层级：根目录"),
		waitForText(`#plan-target-browser-level`, "当前层级：根目录"),
		waitForButtonEnabled(`#plan-source-browser-select-current`),
		chromedp.SetValue(`#plan-selected-roots`, `["/stale-source"]`, chromedp.ByID),
		chromedp.Evaluate(`(() => document.querySelector('#plan-source-browser-select-current')?.click())()`, nil),
		waitForValueContains(`#plan-selected-roots`, `"/"`),
		waitForText(`#plan-source-browser-selection`, "将回填到选定根目录(JSON)：/"),
		waitForButtonEnabled(`#plan-target-browser-select-current`),
		chromedp.SetValue(`#plan-target-root`, "/stale-target", chromedp.ByID),
		chromedp.Evaluate(`(() => document.querySelector('#plan-target-browser-select-current')?.click())()`, nil),
		waitForValue(`#plan-target-root`, "/"),
		waitForText(`#plan-target-browser-selection`, "将回填到目标根目录：/"),
		chromedp.SetValue(`#plan-threshold`, "10", chromedp.ByID),
		chromedp.SetValue(`#plan-selected-roots`, `["/demo","/archive"]`, chromedp.ByID),
		chromedp.SetValue(`#plan-entries`, entriesJSON, chromedp.ByID),
		chromedp.Click(`#preview-plan`, chromedp.ByID),
		waitForText(`#plan-recommendation-title`, "建议执行模式：按目录逐棵推进（leaf_first_lazy） / 建议风控档位：balanced"),
		waitForText(`#plan-recommendation-reason`, "检测到多个顶层目录，按子树逐个推进会更稳妥，也更方便分批处理。"),
		chromedp.Click(`#apply-recommended-execution`, chromedp.ByID),
		waitForValue(`#plan-execution-mode`, "leaf_first_lazy"),
		waitForText(`#flash`, "已采用推荐执行模式：leaf_first_lazy"),
		chromedp.Click(`#apply-recommended-risk`, chromedp.ByID),
		waitForValue(`#plan-risk-mode`, "balanced"),
		waitForText(`#flash`, "已采用推荐风控档位：balanced"),
		waitForText(`#plan-preview-meta`, "pre_scan_flat"),
		waitForText(`#plan-preview-meta`, "选定根目录"),
		waitForText(`#plan-preview-meta`, "/demo -> /archive"),
		waitForText(`#plan-preview-meta`, "网盘源基线"),
		waitForText(`#plan-preview-meta`, "最终生效"),
		waitForText(`#plan-preview-meta`, "校准结果"),
		waitForText(`#plan-preview-meta`, "任务覆盖字段"),
		waitForText(`#plan-preview-meta`, "删除记录仅用于定位"),
		chromedp.Evaluate(`(() => {
			const button = document.querySelector('#plan-preview-meta [data-source-delete-prefill-paths]');
			if (!button) {
				throw new Error('missing source deletion preview prefill button');
			}
			button.click();
			const roots = document.querySelector('#plan-selected-roots')?.value || '';
			const entries = document.querySelector('#plan-entries')?.value || '';
			if (!roots.includes('/demo/deleted.bin') || !entries.includes('/demo/deleted.bin')) {
				throw new Error('source deletion preview prefill did not update form: roots=' + roots + '; entries=' + entries);
			}
			return true;
		})()`, nil),
		chromedp.Click(`#plan-form button[type="submit"]`, chromedp.ByQuery),
		waitForText(`#flash`, "当前只有删除记录，没有可执行条目；请先恢复源文件并重新预览"),
		chromedp.SetValue(`#plan-selected-roots`, `["/demo","/archive"]`, chromedp.ByID),
		chromedp.SetValue(`#plan-entries`, entriesJSON, chromedp.ByID),
		waitForText(`#plan-preview`, `"strategy": "fast_upload"`),
		chromedp.Click(`#plan-form button[type="submit"]`, chromedp.ByQuery),
		waitForText(`#tasks-list`, "guangya -> 123_open"),
		waitForText(`#task-summary`, "leaf_first_lazy"),
		waitForText(`#task-summary`, "选定根目录"),
		waitForText(`#task-summary`, "网盘源校准后"),
		waitForText(`#task-summary`, "任务覆盖"),
		waitForText(`#task-summary`, "校准结果"),
		waitForText(`#task-summary`, "任务覆盖字段"),
		waitForText(`#task-runtime`, "选定根目录"),
		waitForText(`#task-runtime`, "删除记录摘要"),
		waitForText(`#task-runtime`, "默认只记录，不会自动删除目标端真实文件。"),
		waitForText(`#task-detail`, `"state": "ready"`),
		chromedp.Click(`#task-runtime [data-source-delete-prefill-paths]`, chromedp.ByQuery),
		waitForValueContains(`#plan-selected-roots`, "/demo/deleted.bin"),
		waitForText(`#flash`, "已按全部删除记录重建向导范围"),
		chromedp.Click(`.tabs button[data-view="tasks"]`, chromedp.ByQuery),
		waitForText(`#task-runtime`, "删除记录摘要"),
		waitForText(`#task-directory-states`, "/demo"),
		chromedp.Click(`#task-summary [data-runtime-focus-kind="roots"]`, chromedp.ByQuery),
		waitForText(`#task-directory-filter-summary`, "当前显示"),
		chromedp.Click(`#task-directory-states [data-tree-prefill-path="/demo"]`, chromedp.ByQuery),
		waitForValueContains(`#plan-selected-roots`, "/demo"),
		chromedp.Click(`.tabs button[data-view="tasks"]`, chromedp.ByQuery),
		waitForText(`#task-summary`, "leaf_first_lazy"),
		chromedp.Click(`#task-directory-copy-visible`, chromedp.ByID),
		waitForText(`#flash`, "已复制"),
		chromedp.Click(`#task-directory-states [data-tree-focus-panel="directory"]`, chromedp.ByQuery),
		waitForValue(`#task-directory-filter-query`, "/demo"),
		chromedp.Click(`#task-directory-filter-clear`, chromedp.ByID),
		waitForText(`#task-directory-filter-summary`, "显示全部"),
	)

	runStep(t, runCtx, "pause resume run task",
		waitForButtonEnabled(`#task-pause`),
		chromedp.Click(`#task-pause`, chromedp.ByID),
		waitForText(`#task-detail`, `"state": "paused"`),
		waitForButtonEnabled(`#task-resume`),
		chromedp.Click(`#task-resume`, chromedp.ByID),
		waitForText(`#task-detail`, `"state": "ready"`),
		waitForButtonEnabled(`#task-run`),
		chromedp.Click(`#task-run`, chromedp.ByID),
		waitForText(`#task-detail`, `"state": "completed_with_errors"`),
		waitForText(`#task-detail`, `"status": "failed"`),
		waitForText(`#task-runtime`, "completed"),
		waitForText(`#task-summary`, "retry_queue_auto_retry"),
		waitForText(`#task-runtime`, "后台补传候选"),
		waitForText(`#task-resolution-guide`, "等待后台自动补传接管"),
		waitForText(`#task-resolution-guide`, "只看自动补传候选"),
		chromedp.Evaluate(`(() => document.querySelector('#task-resolution-guide [data-task-guide-intent="focus_status_auto_recover_mode"]')?.click())()`, nil),
		waitForValue(`#auto-recover-mode`, "retry_queue_auto_retry"),
		waitForText(`#flash`, "已按 retry_queue_auto_retry 收敛后台补传候选"),
		chromedp.Evaluate(`(() => document.querySelector('.tabs button[data-view="tasks"]')?.click())()`, nil),
		waitForText(`#task-resolution-guide`, "等待后台自动补传接管"),
		chromedp.Evaluate(`(() => document.querySelector('#refresh-tasks')?.click())()`, nil),
		waitForText(`#flash`, "任务列表已刷新"),
		waitForText(`#task-detail`, `"state": "completed_with_errors"`),
		setSelectValue(`#language-select`, "en-US"),
		waitForText(`body`, "Task List"),
		waitForText(`body`, "Task Details"),
		waitForText(`body`, "Run"),
		waitForText(`body`, "Retry Queue"),
		waitForText(`body`, "Runtime Checkpoints"),
		setSelectValue(`#language-select`, "zh-CN"),
		waitForText(`body`, "任务列表"),
	)

	runStep(t, runCtx, "status overview",
		chromedp.Evaluate(`(() => document.querySelector('.tabs button[data-view="status"]')?.click())()`, nil),
		waitForText(`#evidence-summary`, "自动补传任务"),
		waitForText(`#auto-retry-policy-summary`, "group 1"),
		waitForText(`#auto-recover-budget-summary`, "当前生效预算（默认）"),
		waitForText(`body`, "自动补传候选池"),
		waitForText(`body`, "协议族覆盖"),
		waitForText(`body`, "最近 Probe"),
		setSelectValue(`#language-select`, "en-US"),
		waitForText(`body`, "Runtime Evidence Summary"),
		waitForText(`body`, "Auto-Recovery Pool"),
		waitForText(`body`, "Protocol Coverage"),
		waitForText(`body`, "Provider Status Matrix"),
		waitForText(`body`, "Recent Probe"),
		waitForText(`body`, "Recent Results"),
		setSelectValue(`#language-select`, "zh-CN"),
		waitForText(`body`, "运行证据摘要"),
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
		chromedp.Evaluate(`(() => document.querySelector('#auto-recover-reset')?.click())()`, nil),
		waitForText(`#auto-recover-filter-summary`, "条后台补传候选"),
		waitForText(`#auto-recover-summary`, "recover_dry_run_ui_group"),
		chromedp.Evaluate(`(() => document.querySelector('#auto-recover-summary [data-auto-recover-preview-protocol-group="recover_dry_run_ui_group"]')?.click())()`, nil),
		waitForText(`#auto-recover-last-result-summary`, "最近预演"),
		waitForText(`#auto-recover-last-result-summary`, "预演可放行 1"),
		waitForText(`#auto-recover-last-result-summary`, "recover_dry_run_ui_target"),
		waitForText(`#auto-recover-last-result-detail`, "等待态说明"),
		waitForText(`#auto-recover-last-result-detail`, "recover_dry_run_ui_group"),
		waitForText(`#auto-recover-last-result-detail`, "预算占用："),
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
		chromedp.Evaluate(`(() => document.querySelector('.tabs button[data-view="status"]')?.click())()`, nil),
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

	runStep(t, runCtx, "manual confirmation blocked task guide",
		chromedp.Evaluate(`(() => document.querySelector('.tabs button[data-view="tasks"]')?.click())()`, nil),
		chromedp.ActionFunc(func(ctx context.Context) error {
			var payload string
			if err := chromedp.Evaluate(fmt.Sprintf(`(() => {
				const hooks = window.__cloudpanTestHooks;
				if (!hooks || !hooks.state) {
					throw new Error("task hooks unavailable");
				}
				const detail = hooks.state.tasks.find((item) => item?.task?.id === %q);
				if (!detail) {
					throw new Error("missing manual confirmation blocked task detail");
				}
				hooks.state.selectedTaskId = detail.task.id;
				return JSON.stringify(detail);
			})()`, manualTaskID), &payload).Do(ctx); err != nil {
				return err
			}
			if !strings.Contains(payload, `"retryClass":"pending_manual"`) {
				return fmt.Errorf("manual confirmation retryClass missing in payload: %s", payload)
			}
			if !strings.Contains(payload, `"blockedAction":"manual_confirmation_required"`) {
				return fmt.Errorf("manual confirmation blockedAction missing in payload: %s", payload)
			}
			return nil
		}),
		chromedp.ActionFunc(func(ctx context.Context) error {
			if err := chromedp.Evaluate(`(() => window.__cloudpanTestHooks?.renderSelectedTask?.())()`, nil).Do(ctx); err != nil {
				return err
			}
			return nil
		}),
		waitForText(`#task-resolution-guide`, "等待人工确认"),
		chromedp.Evaluate(`(() => document.querySelector('#task-resolution-guide [data-task-guide-intent="focus_status_blocked"]')?.click())()`, nil),
		waitForValue(`#auto-recover-blocked-action`, "manual_confirmation_required"),
		waitForText(`#flash`, "已按 blocked action 收敛最近重试队列"),
		waitForText(`#blocked-actions-summary`, "next-step: 人工确认后继续"),
		waitForText(`#blocked-actions-summary`, "manual_confirmation_required"),
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
		chromedp.Evaluate(`(() => document.querySelector('.tabs button[data-view="tasks"]')?.click())()`, nil),
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
		chromedp.Evaluate(`(() => document.querySelector('.tabs button[data-view="tasks"]')?.click())()`, nil),
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
		chromedp.Evaluate(`(() => document.querySelector('.tabs button[data-view="tasks"]')?.click())()`, nil),
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
		chromedp.ActionFunc(func(ctx context.Context) error {
			if err := chromedp.Evaluate(`(() => window.__cloudpanTestHooks?.renderSelectedTask?.())()`, nil).Do(ctx); err != nil {
				return err
			}
			return nil
		}),
		waitForText(`#task-resolution-guide`, "补回本地回退文件"),
		chromedp.Evaluate(`(() => document.querySelector('#task-resolution-guide [data-task-guide-intent="focus_status_blocked"]')?.click())()`, nil),
		waitForValue(`#auto-recover-blocked-action`, "restore_local_source_file"),
		waitForText(`#flash`, "已按 blocked action 收敛最近重试队列"),
		waitForText(`#blocked-actions-summary`, "next-step: 补回本地文件后继续"),
		waitForText(`#blocked-actions-summary`, "restore_local_source_file"),
	)

	runStep(t, runCtx, "auth expired blocked task",
		chromedp.Evaluate(`(() => document.querySelector('.tabs button[data-view="tasks"]')?.click())()`, nil),
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
		chromedp.ActionFunc(func(ctx context.Context) error {
			if err := chromedp.Evaluate(`(() => window.__cloudpanTestHooks?.renderSelectedTask?.())()`, nil).Do(ctx); err != nil {
				return err
			}
			return nil
		}),
		waitForText(`#task-resolution-guide`, "刷新授权档案"),
		chromedp.Evaluate(`(() => document.querySelector('#task-resolution-guide [data-task-guide-intent="focus_status_blocked"]')?.click())()`, nil),
		waitForValue(`#auto-recover-blocked-action`, "refresh_auth_profile"),
		waitForText(`#flash`, "已按 blocked action 收敛最近重试队列"),
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
		waitForValueContains(`#provider-smoke-title`, "补授权失效样本"),
		waitForValue(`#provider-smoke-category`, "failed"),
		waitForValue(`#provider-smoke-result`, "failure"),
		chromedp.Evaluate(`(() => document.querySelector('#save-provider-smoke')?.click())()`, nil),
		waitForText(`#flash`, "网盘样本 smoke 记录已保存"),
		waitForText(`#provider-smoke-records-filter-summary`, "当前显示 1 /"),
		waitForText(`#provider-smoke-records`, "补授权失效样本"),
		chromedp.Evaluate(`(() => {
		  const rows = Array.from(document.querySelectorAll('#provider-smoke-matrix .directory-row'));
		  const row = rows.find((item) => item.textContent?.includes('aliyun_123_open'));
		  row?.querySelector('[data-provider-smoke-open-record]')?.click();
		})()`, nil),
		waitForText(`#flash`, "已打开 smoke 样本并回填表单"),
		waitForText(`#provider-smoke-markdown`, "aliyun_123_open 补授权失效样本"),
		waitForText(`#provider-smoke-markdown`, "ValidateAuth"),
		chromedp.Evaluate(`(() => document.querySelector('#provider-smoke-matrix [data-provider-smoke-filter-status="pending"]')?.click())()`, nil),
		waitForText(`#flash`, "已按 pending 收敛验收矩阵"),
	)

	runStep(t, runCtx, "provider smoke matrix prefill profile risk",
		chromedp.Evaluate(`(() => document.querySelector('#provider-smoke-matrix [data-provider-smoke-prefill-profile-risk="aliyun_123_open"]')?.click())()`, nil),
		waitForText(`#flash`, "已按真实样本预填账号默认风控"),
		waitForValue(`#profile-provider`, "123_open"),
		waitForValueContains(`#profile-display-name`, "aliyun_123_open 风控模板"),
		waitForValueContains(`#profile-extra`, `riskDefaults`),
		waitForValueContains(`#profile-extra`, `riskDefaultsSourceDisplay`),
		waitForValueContains(`#profile-extra`, `Smoke Matrix aliyun_123_open (pending)`),
		waitForValueContains(`#profile-extra`, `upload_sample`),
		waitForValue(`#profile-risk-request-interval`, "1600"),
		waitForValue(`#profile-risk-directory-interval`, "2200"),
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
		waitForTextContent(`#plan-target-profile-insight`, "Smoke Matrix aliyun_123_open (pending)"),
		waitForTextContent(`#plan-target-profile-insight`, "req 1600ms"),
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

func waitForButtonEnabled(selector string) chromedp.ActionFunc {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		var matched bool
		script := fmt.Sprintf(`(() => {
			const el = document.querySelector(%q);
			return !!el && !el.disabled;
		})()`, selector)
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


