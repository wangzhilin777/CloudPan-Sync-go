package desktop

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"cloudpan-sync-go/internal/app"
)

const desktopLoopbackAddr = "127.0.0.1:0"

var errNoDesktopBrowser = errors.New("未找到可用于独立窗口模式的 Chrome / Edge 浏览器")

type desktopLaunchMode string

const (
	desktopLaunchModeApp     desktopLaunchMode = "app"
	desktopLaunchModeBrowser desktopLaunchMode = "browser"
)

func Run(ctx context.Context, cfg app.Config) error {
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	cfg.Addr = desktopLoopbackAddr

	application, err := app.New(runCtx, cfg)
	if err != nil {
		return fmt.Errorf("create desktop app: %w", err)
	}

	listener, err := net.Listen("tcp", cfg.Addr)
	if err != nil {
		return fmt.Errorf("listen desktop addr: %w", err)
	}
	url := fmt.Sprintf("http://%s/", listener.Addr().String())

	runErrCh := make(chan error, 1)
	go func() {
		runErrCh <- application.RunWithListener(runCtx, listener)
	}()

	if err := waitForReady(runCtx, url, 8*time.Second); err != nil {
		return fmt.Errorf("wait desktop console ready: %w", err)
	}
	log.Print(desktopReadyMessage(url))
	windowProc, cleanup, launchMode, err := openDesktopWindow(url)
	if err != nil {
		return fmt.Errorf("open desktop window: %w", err)
	}
	defer cleanup()
	log.Print(desktopLaunchMessage(launchMode, url))

	if windowProc != nil {
		go func() {
			_, _ = windowProc.Wait()
			log.Print(desktopWindowClosedMessage())
			cancel()
		}()
	}

	select {
	case <-runCtx.Done():
		return nil
	case err := <-runErrCh:
		return err
	}
}

func waitForReady(ctx context.Context, url string, timeout time.Duration) error {
	deadlineCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	client := &http.Client{Timeout: 1500 * time.Millisecond}
	for {
		req, err := http.NewRequestWithContext(deadlineCtx, http.MethodGet, url, nil)
		if err == nil {
			resp, reqErr := client.Do(req)
			if reqErr == nil {
				_ = resp.Body.Close()
				if resp.StatusCode >= 200 && resp.StatusCode < 500 {
					return nil
				}
			}
		}

		select {
		case <-deadlineCtx.Done():
			return deadlineCtx.Err()
		case <-ticker.C:
		}
	}
}

func openDesktopWindow(url string) (*os.Process, func(), desktopLaunchMode, error) {
	process, cleanup, err := openChromeAppWindow(url)
	if err == nil {
		return process, cleanup, desktopLaunchModeApp, nil
	}
	if fallbackErr := openSystemBrowser(url); fallbackErr == nil {
		return nil, func() {}, desktopLaunchModeBrowser, nil
	}
	return nil, func() {}, "", buildDesktopWindowOpenError(err, url)
}

func openChromeAppWindow(url string) (*os.Process, func(), error) {
	chromePath, err := findChromeExecutable()
	if err != nil {
		return nil, func() {}, err
	}
	profileDir, err := os.MkdirTemp("", "cloudpan-sync-desktop-*")
	if err != nil {
		return nil, func() {}, fmt.Errorf("create desktop browser profile: %w", err)
	}
	cleanup := func() {
		_ = os.RemoveAll(profileDir)
	}
	cmd := exec.Command(chromePath, buildChromeAppArgs(url, profileDir)...)
	if err := cmd.Start(); err != nil {
		cleanup()
		return nil, func() {}, err
	}
	return cmd.Process, cleanup, nil
}

func buildChromeAppArgs(url string, profileDir string) []string {
	return []string{
		"--app=" + url,
		"--new-window",
		"--user-data-dir=" + profileDir,
		"--window-size=1280,860",
		"--no-first-run",
		"--no-default-browser-check",
		"--disable-features=Translate,msEdgeSidebarV2",
	}
}

func findChromeExecutable() (string, error) {
	candidates := chromeCandidates()
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	if path, err := exec.LookPath("chrome"); err == nil {
		return path, nil
	}
	if path, err := exec.LookPath("msedge"); err == nil {
		return path, nil
	}
	return "", errNoDesktopBrowser
}

func chromeCandidates() []string {
	switch runtime.GOOS {
	case "windows":
		return []string{
			filepath.Join(os.Getenv("ProgramFiles"), "Google", "Chrome", "Application", "chrome.exe"),
			filepath.Join(os.Getenv("ProgramFiles(x86)"), "Google", "Chrome", "Application", "chrome.exe"),
			filepath.Join(os.Getenv("LocalAppData"), "Google", "Chrome", "Application", "chrome.exe"),
			filepath.Join(os.Getenv("ProgramFiles"), "Microsoft", "Edge", "Application", "msedge.exe"),
			filepath.Join(os.Getenv("ProgramFiles(x86)"), "Microsoft", "Edge", "Application", "msedge.exe"),
			filepath.Join(os.Getenv("LocalAppData"), "Microsoft", "Edge", "Application", "msedge.exe"),
		}
	case "darwin":
		return []string{
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
		}
	default:
		return []string{
			"/usr/bin/google-chrome",
			"/usr/bin/google-chrome-stable",
			"/usr/bin/chromium",
			"/usr/bin/chromium-browser",
			"/usr/bin/microsoft-edge",
		}
	}
}

func openSystemBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}

func buildDesktopWindowOpenError(cause error, url string) error {
	return errors.New(buildDesktopWindowOpenMessage(cause, url))
}

func desktopLaunchMessage(mode desktopLaunchMode, url string) string {
	return desktopLaunchModeMessage(mode, url)
}
