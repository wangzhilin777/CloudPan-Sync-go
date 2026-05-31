package provider

import (
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultCatalogProviderRequestFailedContracts(t *testing.T) {
	makeLocalFile := func(t *testing.T) string {
		t.Helper()
		path := filepath.Join(t.TempDir(), "upload.bin")
		if err := os.WriteFile(path, []byte("provider-request-failed"), 0o600); err != nil {
			t.Fatalf("write local file: %v", err)
		}
		return path
	}

	t.Run("aliyun open multipart part failure stays provider_request_failed", func(t *testing.T) {
		server, _ := newAliyunOpenUploadFailingPartTestServer(t, 2)
		defer server.Close()

		originalClient := providerHTTPClient
		providerHTTPClient = server.Client()
		defer func() { providerHTTPClient = originalClient }()

		registry := NewRegistry(DefaultCatalog()...)
		entry, ok := registry.Get("aliyundrive_open")
		if !ok {
			t.Fatal("expected aliyundrive_open")
		}

		originalSize := aliyunOpenDefaultPartSize
		aliyunOpenDefaultPartSize = 10
		defer func() { aliyunOpenDefaultPartSize = originalSize }()

		result := entry.Adapter.Upload(UploadRequest{
			Profile: AuthProfile{
				ProviderKey: "aliyundrive_open",
				Token:       "token-live",
				Extra:       map[string]string{"domainId": "bj1", "driveId": "drive-1", "apiEndpoint": server.URL},
			},
			Path:           "/multipart-fail.bin",
			Name:           "multipart-fail.bin",
			LocalPath:      makeLocalFile(t),
			Size:           22,
			ConflictPolicy: ConflictPolicyAutoRenameNew,
			Strategy:       "download_upload",
		})
		assertProviderRequestFailedUploadBlocked(t, result, "open_family_real_upload")
		if got := int64MapValue(result.Payload, "failedPartNumber"); got != 2 {
			t.Fatalf("expected failedPartNumber 2, got %+v", result.Payload)
		}
		if got := int64MapValue(result.Payload, "nextPartNumber"); got != 2 {
			t.Fatalf("expected nextPartNumber 2, got %+v", result.Payload)
		}
	})

	t.Run("baidu missing uploadid stays explicit missing_uploadid", func(t *testing.T) {
		server := newBaiduMissingUploadIDTestServer(t)
		defer server.Close()

		originalClient := providerHTTPClient
		providerHTTPClient = server.Client()
		defer func() { providerHTTPClient = originalClient }()

		registry := NewRegistry(DefaultCatalog()...)
		entry, ok := registry.Get("baidu_netdisk")
		if !ok {
			t.Fatal("expected baidu_netdisk")
		}

		result := entry.Adapter.Upload(UploadRequest{
			Profile: AuthProfile{
				ProviderKey: "baidu_netdisk",
				Token:       "token-live",
				Extra:       map[string]string{"apiEndpoint": server.URL, "pcsEndpoint": server.URL},
			},
			Path:           "/missing-uploadid.bin",
			Name:           "missing-uploadid.bin",
			LocalPath:      makeLocalFile(t),
			Size:           24,
			ConflictPolicy: ConflictPolicyAutoRenameNew,
			Strategy:       "download_upload",
			MD5:            "4c7fb2d4ac8976657e13e231ad091db5",
		})
		if result.OK {
			t.Fatalf("expected missing_uploadid to stay blocked, got %+v", result)
		}
		if result.Status != "missing_uploadid" {
			t.Fatalf("expected missing_uploadid, got %+v", result)
		}
		if !containsProviderSubstring(result.Message, "uploadid") {
			t.Fatalf("expected uploadid message, got %+v", result)
		}
		if result.Mode != "baidu_family_real_upload" {
			t.Fatalf("expected baidu_family_real_upload, got %+v", result)
		}
	})
}

func assertProviderRequestFailedUploadBlocked(t *testing.T, result UploadResult, wantMode string) {
	t.Helper()
	if result.OK {
		t.Fatalf("expected provider request failure to stay blocked, got %+v", result)
	}
	if result.Status != "provider_request_failed" {
		t.Fatalf("expected provider_request_failed, got %+v", result)
	}
	if result.Mode != wantMode {
		t.Fatalf("expected mode %s, got %+v", wantMode, result)
	}
}

func newBaiduMissingUploadIDTestServer(t *testing.T) *httptest.Server {
	t.Helper()

	mustAccess := func(w http.ResponseWriter, r *http.Request) bool {
		t.Helper()
		if got := r.URL.Query().Get("access_token"); got != "token-live" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return false
		}
		return true
	}
	mustParseForm := func(r *http.Request) url.Values {
		t.Helper()
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		return r.Form
	}

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !mustAccess(w, r) {
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == baiduXPanFilePath && r.URL.Query().Get("method") == "list":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"list": []map[string]interface{}{}})
		case r.Method == http.MethodPost && r.URL.Path == baiduXPanFilePath && r.URL.Query().Get("method") == "precreate":
			form := mustParseForm(r)
			if form.Get("path") == "" {
				t.Fatalf("expected precreate path")
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"return_type": 2})
		case r.Method == http.MethodPost && r.URL.Path == baiduPCSUploadPath && r.URL.Query().Get("method") == "upload":
			mediaType, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
			if err != nil {
				t.Fatalf("parse multipart content type: %v", err)
			}
			if mediaType != "multipart/form-data" {
				t.Fatalf("expected multipart/form-data, got %s", mediaType)
			}
			reader := multipart.NewReader(r.Body, params["boundary"])
			part, err := reader.NextPart()
			if err != nil {
				t.Fatalf("read multipart part: %v", err)
			}
			if _, err := io.ReadAll(part); err != nil {
				t.Fatalf("read multipart body: %v", err)
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"md5": "4c7fb2d4ac8976657e13e231ad091db5"})
		case r.Method == http.MethodPost && r.URL.Path == baiduXPanFilePath && r.URL.Query().Get("method") == "create":
			form := mustParseForm(r)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"fs_id": "4444",
				"path":  fmt.Sprintf("%s/%s", form.Get("path"), "unexpected"),
			})
		default:
			http.NotFound(w, r)
		}
	}))
}
