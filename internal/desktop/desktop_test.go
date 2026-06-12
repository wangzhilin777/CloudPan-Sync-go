package desktop

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"runtime"
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
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "--window-size=") {
		t.Fatalf("expected an explicit window size for an app-like window, got %v", args)
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

func TestBuildDesktopWindowOpenError(t *testing.T) {
	url := "http://127.0.0.1:18080/"

	testCases := []struct {
		name  string
		cause error
		want  string
	}{
		{
			name:  "no dedicated browser",
			cause: errNoDesktopBrowser,
			want:  "未找到 Chrome / Edge 独立窗口浏览器，且系统浏览器兜底也失败，请手动访问 http://127.0.0.1:18080/",
		},
		{
			name:  "generic open failure",
			cause: errors.New("browser start failed"),
			want:  "桌面模式未能打开独立窗口，请手动访问 http://127.0.0.1:18080/：browser start failed",
		},
		{
			name:  "nil cause",
			cause: nil,
			want:  "桌面模式未能打开控制台窗口，请手动访问 http://127.0.0.1:18080/",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := buildDesktopWindowOpenError(tc.cause, url).Error(); got != tc.want {
				t.Fatalf("buildDesktopWindowOpenError() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestChromeCandidatesIncludeLocalEdgePathOnWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows-only candidate assertion")
	}
	joined := strings.Join(chromeCandidates(), "\n")
	if !strings.Contains(joined, `Microsoft\Edge\Application\msedge.exe`) {
		t.Fatalf("expected LocalAppData Edge candidate in %q", joined)
	}
}

func TestDesktopLaunchMessage(t *testing.T) {
	url := "http://127.0.0.1:18080/"

	testCases := []struct {
		name string
		mode desktopLaunchMode
		want string
	}{
		{
			name: "app window mode",
			mode: desktopLaunchModeApp,
			want: "已使用 Chrome / Edge 独立窗口打开控制台。关闭窗口后会自动退出本地服务；如窗口未弹出，可手动访问 http://127.0.0.1:18080/",
		},
		{
			name: "system browser fallback",
			mode: desktopLaunchModeBrowser,
			want: "当前未使用独立窗口，已退回系统默认浏览器。关闭浏览器标签页不会自动退出本地服务；如需停止，请关闭当前终端窗口或按 Ctrl+C。若浏览器未自动打开，请手动访问 http://127.0.0.1:18080/",
		},
		{
			name: "unknown mode fallback",
			mode: "",
			want: "桌面模式已启动。若面板未自动打开，请手动访问 http://127.0.0.1:18080/",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := desktopLaunchMessage(tc.mode, url); got != tc.want {
				t.Fatalf("desktopLaunchMessage() = %q, want %q", got, tc.want)
			}
		})
	}
}
