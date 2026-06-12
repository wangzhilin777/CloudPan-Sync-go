package desktop

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// desktopLang 解析桌面模式的界面语言。默认中文（zh），设置 CLOUDPAN_LANG=en / en-US
// 时切换为英文。这样桌面端启动、错误和退出提示可以与控制台保持同一套双语策略，
// 而不再固定只输出中文。
func desktopLang() string {
	raw := strings.ToLower(strings.TrimSpace(os.Getenv("CLOUDPAN_LANG")))
	switch {
	case raw == "":
		return "zh"
	case strings.HasPrefix(raw, "en"):
		return "en"
	case strings.HasPrefix(raw, "zh"):
		return "zh"
	default:
		return "zh"
	}
}

func isEnglishDesktopLang() bool {
	return desktopLang() == "en"
}

// noDesktopBrowserMessage 返回“找不到独立窗口浏览器”的双语错误文案。
func noDesktopBrowserMessage() string {
	if isEnglishDesktopLang() {
		return "No Chrome / Edge browser available for dedicated-window mode."
	}
	return "未找到可用于独立窗口模式的 Chrome / Edge 浏览器"
}

// desktopReadyMessage 返回“本地服务已就绪”的双语提示。
func desktopReadyMessage(url string) string {
	if isEnglishDesktopLang() {
		return fmt.Sprintf("Desktop console service is ready: %s", url)
	}
	return fmt.Sprintf("桌面模式服务已就绪：%s", url)
}

// desktopWindowClosedMessage 返回“独立窗口已关闭，正在退出本地服务”的双语提示。
func desktopWindowClosedMessage() string {
	if isEnglishDesktopLang() {
		return "Dedicated desktop window closed; shutting down the local service."
	}
	return "桌面独立窗口已关闭，正在退出本地服务。"
}

func buildDesktopWindowOpenMessage(cause error, url string) string {
	english := isEnglishDesktopLang()
	if cause == nil {
		if english {
			return fmt.Sprintf("Desktop mode could not open the console window. Please open %s manually.", url)
		}
		return fmt.Sprintf("桌面模式未能打开控制台窗口，请手动访问 %s", url)
	}
	if errors.Is(cause, errNoDesktopBrowser) {
		if english {
			return fmt.Sprintf("No Chrome / Edge dedicated-window browser was found and the system browser fallback also failed. Please open %s manually.", url)
		}
		return fmt.Sprintf("未找到 Chrome / Edge 独立窗口浏览器，且系统浏览器兜底也失败，请手动访问 %s", url)
	}
	if english {
		return fmt.Sprintf("Desktop mode could not open a dedicated window. Please open %s manually: %v", url, cause)
	}
	return fmt.Sprintf("桌面模式未能打开独立窗口，请手动访问 %s：%v", url, cause)
}

func desktopLaunchModeMessage(mode desktopLaunchMode, url string) string {
	english := isEnglishDesktopLang()
	switch mode {
	case desktopLaunchModeApp:
		if english {
			return fmt.Sprintf("Opened the console in a Chrome / Edge dedicated window. Closing the window will stop the local service automatically; if the window did not appear, open %s manually.", url)
		}
		return fmt.Sprintf("已使用 Chrome / Edge 独立窗口打开控制台。关闭窗口后会自动退出本地服务；如窗口未弹出，可手动访问 %s", url)
	case desktopLaunchModeBrowser:
		if english {
			return fmt.Sprintf("No dedicated window is in use; fell back to the system default browser. Closing the browser tab will NOT stop the local service; to stop it, close this terminal window or press Ctrl+C. If the browser did not open, please open %s manually.", url)
		}
		return fmt.Sprintf("当前未使用独立窗口，已退回系统默认浏览器。关闭浏览器标签页不会自动退出本地服务；如需停止，请关闭当前终端窗口或按 Ctrl+C。若浏览器未自动打开，请手动访问 %s", url)
	default:
		if english {
			return fmt.Sprintf("Desktop mode started. If the panel did not open automatically, open %s manually.", url)
		}
		return fmt.Sprintf("桌面模式已启动。若面板未自动打开，请手动访问 %s", url)
	}
}
