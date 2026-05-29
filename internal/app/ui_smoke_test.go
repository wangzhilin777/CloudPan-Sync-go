package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

	runCtx, cancelTimeout := context.WithTimeout(browserCtx, 90*time.Second)
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
		waitForText(`#plan-preview-meta`, "pre_scan_flat"),
		waitForText(`#plan-preview-meta`, "SELECTED ROOTS"),
		waitForText(`#plan-preview-meta`, "/demo -> /archive"),
		waitForText(`#plan-preview`, `"strategy": "fast_upload"`),
		chromedp.Click(`#plan-form button[type="submit"]`, chromedp.ByQuery),
		waitForText(`#tasks-list`, "guangya -> 123_open"),
		waitForText(`#task-summary`, "pre_scan_flat"),
		waitForText(`#task-summary`, "SELECTED ROOTS"),
		waitForText(`#task-runtime`, "SELECTED ROOTS"),
		waitForText(`#task-detail`, `"state": "ready"`),
		waitForText(`#task-directory-states`, "/demo"),
		chromedp.Click(`#task-summary [data-runtime-focus-kind="roots"]`, chromedp.ByQuery),
		waitForText(`#task-directory-filter-summary`, "当前显示"),
		chromedp.Click(`#task-directory-states [data-tree-prefill-path="/demo"]`, chromedp.ByQuery),
		waitForValueContains(`#plan-selected-roots`, "/demo"),
		chromedp.Click(`button[data-view="tasks"]`, chromedp.ByQuery),
		waitForText(`#task-summary`, "pre_scan_flat"),
		chromedp.Click(`#task-directory-copy-visible`, chromedp.ByID),
		waitForText(`#flash`, "已复制"),
		chromedp.Click(`#task-directory-states [data-tree-focus-panel="directory"]`, chromedp.ByQuery),
		waitForValue(`#task-directory-filter-query`, "/demo"),
		chromedp.Click(`#task-directory-filter-clear`, chromedp.ByID),
		waitForText(`#task-directory-filter-summary`, "显示全部"),
	)

	runStep(t, runCtx, "pause resume run task",
		chromedp.Click(`#task-pause`, chromedp.ByID),
		waitForText(`#task-detail`, `"state": "paused"`),
		chromedp.Click(`#task-resume`, chromedp.ByID),
		waitForText(`#task-detail`, `"state": "ready"`),
		chromedp.Click(`#task-run`, chromedp.ByID),
		waitForText(`#task-detail`, `"state": "completed_with_errors"`),
		waitForText(`#task-detail`, `"status": "failed"`),
		waitForText(`#task-runtime`, "completed"),
		waitForText(`#task-summary`, "retry_queue_auto_retry"),
		waitForText(`#task-runtime`, "后台补传候选"),
	)

	runStep(t, runCtx, "status evidence and retry",
		chromedp.Evaluate(`(() => document.querySelector('button[data-view="status"]')?.click())()`, nil),
		waitForText(`#evidence-summary`, "Auto Recover"),
		waitForText(`#auto-retry-policy-summary`, "group 1"),
		waitForText(`#auto-recover-budget-summary`, "当前生效预算（默认）"),
		chromedp.Click(`#auto-recover-summary [data-auto-recover-apply-budgets]`, chromedp.ByQuery),
		waitForText(`#auto-recover-budget-summary`, "当前手动放行预算"),
		waitForText(`#auto-recover-budget-summary`, "provider 3"),
		waitForText(`#auto-recover-budget-summary`, "profile 2"),
		chromedp.Click(`#auto-recover-preview`, chromedp.ByID),
		waitForText(`#auto-recover-last-result-summary`, "最近预演"),
		waitForText(`#auto-recover-last-result-summary`, "可放行"),
		chromedp.SetValue(`#auto-recover-limit-per-protocol-group`, "2", chromedp.ByID),
		waitForText(`#auto-recover-budget-summary`, "group 2"),
		chromedp.Click(`#auto-recover-summary [data-auto-recover-preview-lane-mode]`, chromedp.ByQuery),
		waitForText(`#auto-recover-last-result-summary`, "laneBudget"),
		waitForText(`#auto-recover-summary`, "retry_queue_auto_retry"),
		waitForText(`#status-runtime-checkpoints`, "SELECTED ROOTS"),
		waitForText(`#status-runtime-checkpoints`, "SCAN TRACE"),
		chromedp.Click(`#status-directory-states [data-tree-sync-panel="directory"][data-tree-sync-path="/demo"]`, chromedp.ByQuery),
		waitForValue(`#status-pending-filter-query`, "/demo"),
		chromedp.Click(`#status-directory-copy-visible`, chromedp.ByID),
		waitForText(`#flash`, "已复制"),
		waitForSelectorCount(`button[data-tree-bulk-scope="status"][data-tree-bulk-panel="directory"][data-tree-bulk-action="collapse"]`, 1),
		chromedp.Click(`button[data-tree-bulk-scope="status"][data-tree-bulk-panel="directory"][data-tree-bulk-action="collapse"]`, chromedp.ByQuery),
		waitForSelectorCount(`#status-directory-states .directory-group.is-collapsed`, 1),
		waitForLocalStorageContains("cloudpan_console_tree_groups_collapsed", `status:directory:/demo`),
		chromedp.SetValue(`#report-title`, "UI Smoke 里程碑报告", chromedp.ByID),
		chromedp.SetValue(`#report-note`, "用于验证报告历史与保存流程", chromedp.ByID),
		chromedp.Click(`#save-report`, chromedp.ByID),
		waitForText(`#evidence-report`, "UI Smoke 里程碑报告"),
		waitForText(`#report-history`, "UI Smoke 里程碑报告"),
		chromedp.Click(`#report-history [data-report-view]`, chromedp.ByQuery),
		waitForText(`#evidence-report`, "UI Smoke 里程碑报告"),
		chromedp.SetValue(`#provider-smoke-provider-key`, "123_open", chromedp.ByID),
		chromedp.SetValue(`#provider-smoke-protocol-group`, "aliyun_123_open", chromedp.ByID),
		chromedp.SetValue(`#provider-smoke-auth-mode`, "manual_token", chromedp.ByID),
		setSelectValue(`#provider-smoke-category`, "browse_only"),
		chromedp.SetValue(`#provider-smoke-title`, "UI Smoke Provider Smoke", chromedp.ByID),
		chromedp.SetValue(`#provider-smoke-note`, "用于验证真实 smoke 记录保存", chromedp.ByID),
		chromedp.SetValue(`#provider-smoke-operations`, "ValidateAuth,List,Metadata", chromedp.ByID),
		chromedp.Click(`#save-provider-smoke`, chromedp.ByID),
		waitForText(`#evidence-summary`, "Accepted Groups"),
		waitForText(`#provider-smoke-summary`, "aliyun_123_open"),
		waitForText(`#provider-smoke-matrix`, "accepted"),
		waitForText(`#provider-smoke-matrix`, "aliyun_123_open"),
		waitForText(`#provider-smoke-records`, "UI Smoke Provider Smoke"),
		chromedp.Click(`#provider-smoke-matrix [data-provider-smoke-focus-group]`, chromedp.ByQuery),
		waitForText(`#provider-smoke-records-filter-summary`, "当前显示"),
		chromedp.Click(`#provider-smoke-matrix [data-provider-smoke-draft]`, chromedp.ByQuery),
		waitForValue(`#provider-smoke-protocol-group`, "aliyun_123_open"),
		waitForValueContains(`#provider-smoke-note`, "协议组"),
		chromedp.Click(`#provider-smoke-records-filter-clear`, chromedp.ByID),
		waitForText(`#provider-smoke-records-filter-summary`, "显示全部"),
		chromedp.Click(`#provider-smoke-records [data-provider-smoke-view]`, chromedp.ByQuery),
		waitForText(`#provider-smoke-markdown`, "UI Smoke Provider Smoke"),
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
