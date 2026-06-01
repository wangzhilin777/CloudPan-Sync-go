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
		meta: provider.Provider{Key: "missing_uploadid_target", DisplayName: "Missing UploadID Target", ProtocolGroup: "fake_target", AuthModes: []string{"manual_token"}, FastUploadInputs: []string{"md5", "size"}, FallbackModes: []string{"download_upload"}, Status: "planned"},
		capability: provider.CapabilitySet{SupportsAuthValidation: true, SupportsUpload: true},
		uploadFunc: func(req provider.UploadRequest) provider.UploadResult {
			return provider.UploadResult{OperationResult: provider.OperationResult{Status: "missing_uploadid", Message: "provider omitted uploadid", Mode: "scripted_missing_uploadid"}}
		},
	}
	registry := provider.NewRegistry(append(provider.DefaultCatalog(), targetAdapter)...)
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
		setSelectValue(`#plan-risk-mode`, "fast"),
		setSelectValue(`#plan-execution-mode`, "pre_scan_flat"),
		waitForText(`#plan-execution-hint`, "pre_scan_flat"),
		chromedp.SetValue(`#plan-threshold`, "10", chromedp.ByID),
		chromedp.SetValue(`#plan-selected-roots`, `["/demo","/archive"]`, chromedp.ByID),
		chromedp.SetValue(`#plan-entries`, entriesJSON, chromedp.ByID),
		chromedp.Click(`#preview-plan`, chromedp.ByID),
		waitForText(`#plan-recommendation-title`, "建议切换到：leaf_first_lazy"),
		waitForText(`#plan-recommendation-reason`, "Multiple top-level roots are safer to process subtree by subtree."),
		chromedp.Click(`#apply-recommended-execution`, chromedp.ByID),
		waitForValue(`#plan-execution-mode`, "leaf_first_lazy"),
		waitForText(`#flash`, "已采用推荐执行模式：leaf_first_lazy"),
		waitForText(`#plan-preview-meta`, "pre_scan_flat"),
		waitForText(`#plan-preview-meta`, "SELECTED ROOTS"),
		waitForText(`#plan-preview-meta`, "/demo -> /archive"),
		waitForText(`#plan-preview-meta`, "CALIBRATED"),
		waitForText(`#plan-preview-meta`, "OVERRIDE FIELDS"),
		waitForText(`#plan-preview`, `"strategy": "fast_upload"`),
		chromedp.Click(`#plan-form button[type="submit"]`, chromedp.ByQuery),
		waitForText(`#tasks-list`, "guangya -> 123_open"),
		waitForText(`#task-summary`, "leaf_first_lazy"),
		waitForText(`#task-summary`, "SELECTED ROOTS"),
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

	runStep(t, runCtx, "provider session missing blocked task",
		chromedp.Evaluate(`(() => document.querySelector('button[data-view="tasks"]')?.click())()`, nil),
		waitForText(`#tasks-list`, "missing_uploadid_source -> missing_uploadid_target"),
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
		chromedp.Evaluate(`(() => window.__cloudpanTestHooks?.focusBlockedActionSummary?.("manual_intervention_required"))()`, nil),
		waitForValue(`#auto-recover-blocked-action`, "manual_intervention_required"),
		waitForText(`#status-retry-filter-summary`, "当前显示"),
	)
	runStep(t, runCtx, "provider smoke matrix workflow",
		waitForText(`#provider-smoke-matrix`, "aliyun_123_open"),
		chromedp.Evaluate(`(() => document.querySelector('#provider-smoke-matrix [data-provider-smoke-draft="aliyun_123_open"]')?.click())()`, nil),
		waitForValue(`#provider-smoke-provider-key`, "123_open"),
		waitForValue(`#provider-smoke-protocol-group`, "aliyun_123_open"),
		waitForValueContains(`#provider-smoke-title`, "aliyun_123_open"),
		waitForValueContains(`#provider-smoke-note`, "协议组：aliyun_123_open"),
		chromedp.Evaluate(`(() => document.querySelector('#provider-smoke-matrix [data-provider-smoke-focus-group="aliyun_123_open"]')?.click())()`, nil),
		waitForValue(`#provider-smoke-records-filter-group`, "aliyun_123_open"),
		waitForText(`#provider-smoke-records-filter-summary`, "当前没有 smoke 记录。"),
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
