package app

import (
	"strings"
	"testing"
)

func TestLocalConsoleURL(t *testing.T) {
	testCases := []struct {
		name string
		addr string
		want string
	}{
		{name: "empty uses default", addr: "", want: "http://127.0.0.1:8080/"},
		{name: "wildcard host maps to loopback", addr: "0.0.0.0:9090", want: "http://127.0.0.1:9090/"},
		{name: "port only maps to loopback", addr: ":7070", want: "http://127.0.0.1:7070/"},
		{name: "localhost stays localhost", addr: "localhost:6060", want: "http://localhost:6060/"},
		{name: "lan host stays lan host", addr: "192.168.1.9:5050", want: "http://192.168.1.9:5050/"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := localConsoleURL(tc.addr); got != tc.want {
				t.Fatalf("localConsoleURL(%q) = %q, want %q", tc.addr, got, tc.want)
			}
		})
	}
}

func TestStartupLogFieldsIncludeFriendlyHints(t *testing.T) {
	app := &App{
		cfg: Config{
			Addr:    ":8080",
			DBPath:  "data/cloudpan-sync.db",
			DataDir: "data",
		},
		runtimeAddr: "127.0.0.1:18080",
	}

	fields := app.startupLogFields()
	if len(fields)%2 != 0 {
		t.Fatalf("startupLogFields should return key/value pairs, got %v", fields)
	}

	joined := make([]string, 0, len(fields))
	for _, item := range fields {
		joined = append(joined, strings.TrimSpace(item.(string)))
	}
	text := strings.Join(joined, "\n")

	for _, want := range []string{
		"local_url",
		"http://127.0.0.1:18080/",
		"lan_hint",
		"局域网访问时，请把 127.0.0.1 替换成运行这台服务机器的局域网 IP。",
		"docker_hint",
		"Docker 或 NAS 部署时，请把端口映射后的宿主机地址填写到浏览器，例如 http://NAS-IP:8080/ 。",
		"desktop_hint",
		"如使用桌面模式，程序会自动打开独立窗口；关闭窗口后会一并清理本地服务。",
		"password_hint",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected startup log fields to contain %q, got %q", want, text)
		}
	}
}
