package desktop

import (
	"context"
	"errors"
	"fmt"
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

func Run(ctx context.Context, cfg app.Config) error {
	cfg.Addr = desktopLoopbackAddr

	application, err := app.New(ctx, cfg)
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
		runErrCh <- application.RunWithListener(ctx, listener)
	}()

	if err := waitForReady(ctx, url, 8*time.Second); err != nil {
		return fmt.Errorf("wait desktop console ready: %w", err)
	}
	if err := openDesktopWindow(url); err != nil {
		return fmt.Errorf("open desktop window: %w", err)
	}

	select {
	case <-ctx.Done():
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

func openDesktopWindow(url string) error {
	if err := openChromeAppWindow(url); err == nil {
		return nil
	}
	return openSystemBrowser(url)
}

func openChromeAppWindow(url string) error {
	chromePath, err := findChromeExecutable()
	if err != nil {
		return err
	}
	cmd := exec.Command(chromePath, buildChromeAppArgs(url)...)
	return cmd.Start()
}

func buildChromeAppArgs(url string) []string {
	return []string{
		"--app=" + url,
		"--new-window",
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
	return "", errors.New("no Chrome-compatible browser found")
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
