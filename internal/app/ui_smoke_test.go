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
		chromedp.SetValue(`#plan-selected-roots`, `["/demo"]`, chromedp.ByID),
		chromedp.SetValue(`#plan-entries`, entriesJSON, chromedp.ByID),
		chromedp.Click(`#preview-plan`, chromedp.ByID),
		waitForText(`#plan-preview-meta`, "pre_scan_flat"),
		waitForText(`#plan-preview`, `"strategy": "fast_upload"`),
		chromedp.Click(`#plan-form button[type="submit"]`, chromedp.ByQuery),
		waitForText(`#tasks-list`, "guangya -> 123_open"),
		waitForText(`#task-summary`, "pre_scan_flat"),
		waitForText(`#task-detail`, `"state": "ready"`),
		waitForText(`#task-detail`, `"executionMode": "pre_scan_flat"`),
		waitForText(`#task-directory-states`, "/demo"),
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
	)

	runStep(t, runCtx, "status evidence and retry",
		chromedp.Click(`button[data-view="status"]`, chromedp.ByQuery),
		waitForText(`#status-table`, "123_open"),
		waitForText(`#status-table`, "pre_scan_flat"),
		waitForText(`#recent-results`, "failed"),
		waitForText(`#recent-results`, "pre_scan_flat"),
		waitForText(`#recent-probes`, "completed"),
		waitForText(`#recent-probes`, "pre_scan_flat"),
		waitForText(`#status-runtime-checkpoints`, "completed"),
		waitForText(`#status-directory-states`, "/demo"),
		waitForText(`#evidence-summary`, "Total Tasks"),
		chromedp.Click(`button[data-view="tasks"]`, chromedp.ByQuery),
		chromedp.Click(`#task-retry`, chromedp.ByID),
		waitForText(`#task-detail`, `"state": "ready"`),
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
			chromedp.WithPollingTimeout(15*time.Second),
		).Do(ctx)
	})
}

func waitForSelectorCount(selector string, minCount int) chromedp.ActionFunc {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		var matched bool
		script := fmt.Sprintf(`(() => document.querySelectorAll(%q).length >= %d)()`, selector, minCount)
		return chromedp.Poll(script, &matched,
			chromedp.WithPollingInterval(120*time.Millisecond),
			chromedp.WithPollingTimeout(15*time.Second),
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
