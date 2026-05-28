package provider

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestPan115FamilyAdapterValidatesAgainstLiveEndpoint(t *testing.T) {
	server, _ := newPan115FamilyTestServer(t)
	t.Cleanup(server.Close)

	originalClient := providerHTTPClient
	providerHTTPClient = server.Client()
	t.Cleanup(func() { providerHTTPClient = originalClient })

	registry := NewRegistry(DefaultCatalog()...)
	entry, ok := registry.Get("115_open")
	if !ok {
		t.Fatal("expected 115_open entry")
	}
	result := entry.Adapter.ValidateAuth(pan115TestProfile(server.URL))
	if !result.OK {
		t.Fatalf("expected validation success, got %+v", result)
	}
	if result.Mode != "pan115_family_real_auth" {
		t.Fatalf("expected pan115_family_real_auth mode, got %s", result.Mode)
	}
}

func TestPan115FamilyAdapterListsReadsAndCreatesDirectory(t *testing.T) {
	server, _ := newPan115FamilyTestServer(t)
	t.Cleanup(server.Close)

	originalClient := providerHTTPClient
	providerHTTPClient = server.Client()
	t.Cleanup(func() { providerHTTPClient = originalClient })

	registry := NewRegistry(DefaultCatalog()...)
	entry, ok := registry.Get("115_open")
	if !ok {
		t.Fatal("expected 115_open entry")
	}
	profile := pan115TestProfile(server.URL)

	list := entry.Adapter.List(ListRequest{
		Profile:  profile,
		ParentID: "0",
		PageSize: 100,
	})
	if !list.OK {
		t.Fatalf("expected list success, got %+v", list)
	}
	if len(list.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(list.Items))
	}
	guide := findProviderItemByPath(list.Items, "guide.txt")
	if guide == nil || stringMapValue(guide, "sha1") != "SHA1-GUIDE-115" {
		t.Fatalf("expected guide item, got %+v", guide)
	}

	metadata := entry.Adapter.Metadata(MetadataRequest{
		Profile: profile,
		FileID:  "file-guide-115",
		Path:    "guide.txt",
	})
	if !metadata.OK || metadata.Status != "exists" {
		t.Fatalf("expected metadata exists, got %+v", metadata)
	}
	if stringMapValue(metadata.Entry, "fileId") != "file-guide-115" {
		t.Fatalf("expected file-guide-115 metadata, got %+v", metadata.Entry)
	}

	createDir := entry.Adapter.CreateDir(CreateDirRequest{
		Profile:  profile,
		ParentID: "0",
		DirName:  "uploads",
	})
	if !createDir.OK {
		t.Fatalf("expected create dir success, got %+v", createDir)
	}
	if stringMapValue(createDir.Payload, "fileId") != "dir-new-115" {
		t.Fatalf("expected dir-new-115, got %+v", createDir.Payload)
	}
}

func TestPan115FamilyAdapterRapidUploadHit(t *testing.T) {
	server, _ := newPan115FamilyTestServer(t)
	t.Cleanup(server.Close)

	originalClient := providerHTTPClient
	providerHTTPClient = server.Client()
	t.Cleanup(func() { providerHTTPClient = originalClient })

	registry := NewRegistry(DefaultCatalog()...)
	entry, ok := registry.Get("115_open")
	if !ok {
		t.Fatal("expected 115_open entry")
	}
	profile := pan115TestProfile(server.URL)

	result := entry.Adapter.Upload(UploadRequest{
		Profile:   profile,
		Name:      "rapid.bin",
		ParentID:  "0",
		LocalPath: mustWritePan115TempFile(t, "rapid.bin", []byte("rapid-content-115")),
		Strategy:  "fast_upload",
		SHA1:      "SHA1-HIT-115",
	})
	if !result.OK {
		t.Fatalf("expected rapid upload success, got %+v", result)
	}
	if stringMapValue(result.Payload, "fileId") != "file-hit-115" {
		t.Fatalf("expected file-hit-115, got %+v", result.Payload)
	}
	if stringMapValue(result.Payload, "verifyMode") != "metadata_by_file_id" {
		t.Fatalf("expected metadata_by_file_id verify, got %+v", result.Payload)
	}
}

func TestPan115FamilyAdapterFallsBackToOSSBinaryUpload(t *testing.T) {
	server, state := newPan115FamilyTestServer(t)
	t.Cleanup(server.Close)

	originalClient := providerHTTPClient
	providerHTTPClient = server.Client()
	t.Cleanup(func() { providerHTTPClient = originalClient })

	registry := NewRegistry(DefaultCatalog()...)
	entry, ok := registry.Get("115_open")
	if !ok {
		t.Fatal("expected 115_open entry")
	}
	profile := pan115TestProfile(server.URL)
	localPath := mustWritePan115TempFile(t, "miss.bin", []byte("miss-content-115"))
	result := entry.Adapter.Upload(UploadRequest{
		Profile:   profile,
		Name:      "miss.bin",
		ParentID:  "0",
		LocalPath: localPath,
		Strategy:  "download_upload",
	})
	if !result.OK {
		t.Fatalf("expected oss fallback upload success, got %+v", result)
	}
	if result.Status != "ok" {
		t.Fatalf("expected ok status, got %+v", result)
	}
	if stringMapValue(result.Payload, "fileId") != "file-miss-115" {
		t.Fatalf("expected file-miss-115 session payload, got %+v", result.Payload)
	}
	if !boolMapValue(result.Payload, "usedBinaryFallback") {
		t.Fatalf("expected usedBinaryFallback, got %+v", result.Payload)
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.ossRequestCount != 1 {
		t.Fatalf("expected 1 oss upload request, got %d", state.ossRequestCount)
	}
	if state.ossMethod != http.MethodPut {
		t.Fatalf("expected PUT oss upload, got %s", state.ossMethod)
	}
	if state.ossPath != "/bucket-115/folder/miss.bin" {
		t.Fatalf("expected path-style oss upload path, got %s", state.ossPath)
	}
	if !strings.HasPrefix(state.ossAuthorization, "OSS ak:") {
		t.Fatalf("expected OSS authorization header, got %s", state.ossAuthorization)
	}
	if state.ossSecurityToken != "token" {
		t.Fatalf("expected x-oss-security-token, got %s", state.ossSecurityToken)
	}
	if got := decodePan115Base64ForTest(t, state.ossCallback); got != "{\"callbackUrl\":\"https://example.invalid\"}" {
		t.Fatalf("unexpected x-oss-callback payload: %s", got)
	}
	if got := decodePan115Base64ForTest(t, state.ossCallbackVar); got != "{\"x:var\":\"1\"}" {
		t.Fatalf("unexpected x-oss-callback-var payload: %s", got)
	}
	if state.ossBody != "miss-content-115" {
		t.Fatalf("unexpected oss request body: %s", state.ossBody)
	}
}

func TestPan115FamilyAdapterResumesCachedOSSUploadSession(t *testing.T) {
	server, state := newPan115FamilyTestServer(t)
	state.failUploadOnce = true
	t.Cleanup(server.Close)

	originalClient := providerHTTPClient
	providerHTTPClient = server.Client()
	t.Cleanup(func() { providerHTTPClient = originalClient })

	registry := NewRegistry(DefaultCatalog()...)
	entry, ok := registry.Get("115_open")
	if !ok {
		t.Fatal("expected 115_open entry")
	}
	profile := pan115TestProfile(server.URL)
	localPath := mustWritePan115TempFile(t, "resume.bin", []byte("resume-content-115"))

	first := entry.Adapter.Upload(UploadRequest{
		Profile:   profile,
		Name:      "resume.bin",
		ParentID:  "0",
		LocalPath: localPath,
		Strategy:  "download_upload",
	})
	if first.OK {
		t.Fatalf("expected first upload to fail, got %+v", first)
	}
	providerData, _ := first.Payload["providerData"].(map[string]interface{})
	if providerData == nil {
		t.Fatalf("expected providerData in failed payload, got %+v", first.Payload)
	}
	uploadSession, _ := providerData["uploadSession"].(map[string]interface{})
	if uploadSession == nil {
		t.Fatalf("expected uploadSession in providerData, got %+v", first.Payload)
	}

	second := entry.Adapter.Upload(UploadRequest{
		Profile:   profile,
		Name:      "resume.bin",
		ParentID:  "0",
		LocalPath: localPath,
		Strategy:  "download_upload",
		ResumeUpload: &ResumeUpload{
			FileID:       stringMapValue(first.Payload, "fileId"),
			ProviderData: providerData,
		},
	})
	if !second.OK {
		t.Fatalf("expected resumed upload success, got %+v", second)
	}
	if !boolMapValue(second.Payload, "resumedUpload") {
		t.Fatalf("expected resumedUpload true, got %+v", second.Payload)
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.uploadInitCount != 2 {
		t.Fatalf("expected only first attempt to call upload init twice, got %d", state.uploadInitCount)
	}
	if state.uploadTokenCount != 1 {
		t.Fatalf("expected only first attempt to fetch upload token once, got %d", state.uploadTokenCount)
	}
	if state.ossRequestCount != 2 {
		t.Fatalf("expected two oss upload attempts, got %d", state.ossRequestCount)
	}
}

func pan115TestProfile(baseURL string) AuthProfile {
	return AuthProfile{
		ProviderKey: "115_open",
		Cookie:      "cookie-live",
		Extra: map[string]string{
			"listEndpoint":           baseURL + "/files",
			"infoEndpoint":           baseURL + "/files/get_info",
			"mkdirEndpoint":          baseURL + "/files/add",
			"uploadInitEndpoint":     baseURL + "/open/upload/init",
			"uploadGetTokenEndpoint": baseURL + "/open/upload/get_token",
		},
	}
}

func mustWritePan115TempFile(t *testing.T, name string, content []byte) string {
	t.Helper()
	tmpDir := t.TempDir()
	localPath := filepath.Join(tmpDir, name)
	if err := os.WriteFile(localPath, content, 0o600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	return localPath
}

type pan115TestServerState struct {
	mu               sync.Mutex
	ossRequestCount  int
	ossMethod        string
	ossPath          string
	ossAuthorization string
	ossSecurityToken string
	ossCallback      string
	ossCallbackVar   string
	ossBody          string
	uploadInitCount  int
	uploadTokenCount int
	failUploadOnce   bool
}

func newPan115FamilyTestServer(t *testing.T) (*httptest.Server, *pan115TestServerState) {
	t.Helper()
	state := &pan115TestServerState{}
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut && strings.HasPrefix(r.URL.Path, "/bucket-115/") {
			bodyBytes, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("read oss body: %v", err)
			}
			state.mu.Lock()
			state.ossRequestCount++
			state.ossMethod = r.Method
			state.ossPath = r.URL.Path
			state.ossAuthorization = r.Header.Get("Authorization")
			state.ossSecurityToken = r.Header.Get("x-oss-security-token")
			state.ossCallback = r.Header.Get("x-oss-callback")
			state.ossCallbackVar = r.Header.Get("x-oss-callback-var")
			state.ossBody = string(bodyBytes)
			if state.failUploadOnce && state.ossRequestCount == 1 {
				state.mu.Unlock()
				http.Error(w, "temporary upload failure", http.StatusInternalServerError)
				return
			}
			state.mu.Unlock()
			w.Header().Set("ETag", "\"etag-115\"")
			w.WriteHeader(http.StatusOK)
			return
		}
		if got := r.Header.Get("Cookie"); got != "cookie-live" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/files":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"data": []map[string]interface{}{
					{"fid": "file-guide-115", "pid": "0", "n": "guide.txt", "s": 12, "sha": "SHA1-GUIDE-115", "pc": "pick-guide"},
					{"cid": "dir-docs-115", "pid": "0", "n": "docs"},
				},
			})
		case "/files/get_info":
			if r.URL.Query().Get("file_id") == "file-miss-115" {
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"data": map[string]interface{}{
						"fid": "file-miss-115", "pid": "0", "n": "miss.bin", "s": 16, "sha": "2F4EBE72AF297C0B96D2F7DD7F271300A34F6AD4", "pc": "pick-miss-115",
					},
				})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"data": map[string]interface{}{
					"fid": "file-guide-115", "pid": "0", "n": "guide.txt", "s": 12, "sha": "SHA1-GUIDE-115", "pc": "pick-guide",
				},
			})
		case "/files/add":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"data": map[string]interface{}{"cid": "dir-new-115"},
			})
		case "/open/upload/init":
			state.mu.Lock()
			state.uploadInitCount++
			state.mu.Unlock()
			if err := r.ParseForm(); err != nil {
				t.Fatalf("parse form: %v", err)
			}
			switch r.Form.Get("fileid") {
			case "SHA1-HIT-115":
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"state": true,
					"data": map[string]interface{}{
						"status":    2,
						"file_id":   "file-hit-115",
						"pick_code": "pick-hit-115",
					},
				})
			default:
				signKey := r.Form.Get("sign_key")
				if signKey == "" {
					_ = json.NewEncoder(w).Encode(map[string]interface{}{
						"state": true,
						"data": map[string]interface{}{
							"status":     7,
							"sign_key":   "sign-key-115",
							"sign_check": "0-3",
							"bucket":     "bucket-115",
							"object":     "folder/miss.bin",
							"callback": map[string]interface{}{
								"value": map[string]interface{}{
									"callback":     "{\"callbackUrl\":\"https://example.invalid\"}",
									"callback_var": "{\"x:var\":\"1\"}",
								},
							},
						},
					})
					return
				}
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"state": true,
					"data": map[string]interface{}{
						"status":    1,
						"file_id":   "file-miss-115",
						"pick_code": "pick-miss-115",
						"bucket":    "bucket-115",
						"object":    "folder/miss.bin",
						"callback": map[string]interface{}{
							"value": map[string]interface{}{
								"callback":     "{\"callbackUrl\":\"https://example.invalid\"}",
								"callback_var": "{\"x:var\":\"1\"}",
							},
						},
					},
				})
			}
		case "/open/upload/get_token":
			state.mu.Lock()
			state.uploadTokenCount++
			state.mu.Unlock()
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"state": true,
				"data": map[string]interface{}{
					"endpoint":          server.URL,
					"access_key_id":     "ak",
					"access_key_secret": "sk",
					"security_token":    "token",
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	return server, state
}

func decodePan115Base64ForTest(t *testing.T, value string) string {
	t.Helper()
	decoded, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		t.Fatalf("decode base64 header: %v", err)
	}
	return string(decoded)
}
