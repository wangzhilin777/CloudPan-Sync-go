package provider

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestOpenFamilyAdapterChecksAliyunFastUploadBySHA1(t *testing.T) {
	server, _ := newAliyunOpenUploadTestServer(t)
	t.Cleanup(server.Close)

	originalClient := providerHTTPClient
	providerHTTPClient = server.Client()
	t.Cleanup(func() { providerHTTPClient = originalClient })

	registry := NewRegistry(DefaultCatalog()...)
	entry, ok := registry.Get("aliyundrive_open")
	if !ok {
		t.Fatal("expected aliyundrive_open entry")
	}

	okCheck := entry.Adapter.FastUploadCheck(FastUploadCheckRequest{
		Profile: AuthProfile{
			ProviderKey: "aliyundrive_open",
			Token:       "token-live",
			Extra: map[string]string{
				"domainId":    "bj1",
				"driveId":     "drive-1",
				"apiEndpoint": server.URL,
			},
		},
		Path: "/demo.bin",
		Name: "demo.bin",
		Size: 1024,
		SHA1: "sha1-hit",
	})
	if !okCheck.OK || !okCheck.Candidate {
		t.Fatalf("expected aliyun fast-upload candidate, got %+v", okCheck)
	}

	missCheck := entry.Adapter.FastUploadCheck(FastUploadCheckRequest{
		Profile: AuthProfile{
			ProviderKey: "aliyundrive_open",
			Token:       "token-live",
			Extra: map[string]string{
				"domainId":    "bj1",
				"driveId":     "drive-1",
				"apiEndpoint": server.URL,
			},
		},
		Path: "/demo.bin",
		Name: "demo.bin",
		Size: 1024,
		MD5:  "md5-only",
	})
	if !missCheck.OK || missCheck.Candidate {
		t.Fatalf("expected aliyun fast-upload to reject md5-only candidate, got %+v", missCheck)
	}
}

func TestOpenFamilyAdapterRapidUploadsAliyunFile(t *testing.T) {
	server, uploaded := newAliyunOpenUploadTestServer(t)
	t.Cleanup(server.Close)

	originalClient := providerHTTPClient
	providerHTTPClient = server.Client()
	t.Cleanup(func() { providerHTTPClient = originalClient })

	registry := NewRegistry(DefaultCatalog()...)
	entry, ok := registry.Get("aliyundrive_open")
	if !ok {
		t.Fatal("expected aliyundrive_open entry")
	}
	profile := AuthProfile{
		ProviderKey: "aliyundrive_open",
		Token:       "token-live",
		Extra: map[string]string{
			"domainId":    "bj1",
			"driveId":     "drive-1",
			"apiEndpoint": server.URL,
		},
	}

	result := entry.Adapter.Upload(UploadRequest{
		Profile:        profile,
		Path:           "/docs/rapid.bin",
		Name:           "rapid.bin",
		Size:           128,
		ConflictPolicy: ConflictPolicyAutoRenameNew,
		Strategy:       "fast_upload",
		SHA1:           "sha1-hit",
	})
	if !result.OK {
		t.Fatalf("expected rapid upload success, got %+v", result)
	}
	if result.Mode != "open_family_real_upload" {
		t.Fatalf("expected open_family_real_upload mode, got %s", result.Mode)
	}
	if !boolMapValue(result.Payload, "rapidUpload") {
		t.Fatalf("expected rapidUpload flag, got %+v", result.Payload)
	}
	if len(*uploaded) != 0 {
		t.Fatalf("expected no binary upload body for rapid upload, got %q", string(*uploaded))
	}
}

func TestOpenFamilyAdapterUploadsAliyunFileByBinary(t *testing.T) {
	server, uploaded := newAliyunOpenUploadTestServer(t)
	t.Cleanup(server.Close)

	originalClient := providerHTTPClient
	providerHTTPClient = server.Client()
	t.Cleanup(func() { providerHTTPClient = originalClient })

	registry := NewRegistry(DefaultCatalog()...)
	entry, ok := registry.Get("aliyundrive_open")
	if !ok {
		t.Fatal("expected aliyundrive_open entry")
	}
	profile := AuthProfile{
		ProviderKey: "aliyundrive_open",
		Token:       "token-live",
		Extra: map[string]string{
			"domainId":    "bj1",
			"driveId":     "drive-1",
			"apiEndpoint": server.URL,
		},
	}

	tmpDir := t.TempDir()
	localPath := filepath.Join(tmpDir, "upload.bin")
	if err := os.WriteFile(localPath, []byte("hello-aliyun-upload"), 0o600); err != nil {
		t.Fatalf("write local file: %v", err)
	}

	result := entry.Adapter.Upload(UploadRequest{
		Profile:        profile,
		Path:           "/docs/upload.bin",
		Name:           "upload.bin",
		Size:           int64(len("hello-aliyun-upload")),
		LocalPath:      localPath,
		ConflictPolicy: ConflictPolicyAutoRenameNew,
		Strategy:       "download_upload",
	})
	if !result.OK {
		t.Fatalf("expected binary upload success, got %+v", result)
	}
	if got := string(*uploaded); got != "hello-aliyun-upload" {
		t.Fatalf("expected uploaded body hello-aliyun-upload, got %q", got)
	}
	if stringMapValue(result.Payload, "fileId") != "file-uploaded" {
		t.Fatalf("expected file-uploaded result, got %+v", result.Payload)
	}
}

func newAliyunOpenUploadTestServer(t *testing.T) (*httptest.Server, *[]byte) {
	t.Helper()

	uploaded := []byte{}
	mustDecode := func(r *http.Request) map[string]interface{} {
		t.Helper()
		var payload map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		return payload
	}

	var baseURL string
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut && r.URL.Path == "/upload/part/1" {
			body, err := ioReadAll(r)
			if err != nil {
				t.Fatalf("read upload body: %v", err)
			}
			uploaded = body
			w.Header().Set("ETag", "\"etag-upload-1\"")
			w.WriteHeader(http.StatusOK)
			return
		}

		if auth := r.Header.Get("Authorization"); auth != "Bearer token-live" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/v2/user/get":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"domain_id": "bj1",
			})
		case "/v2/drive/get_default_drive":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"drive_id": "drive-1",
			})
		case "/adrive/v1.0/openFile/list":
			payload := mustDecode(r)
			switch stringMapValue(payload, "parent_file_id") {
			case "root":
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"items": []map[string]interface{}{
						{
							"name":           "docs",
							"type":           "folder",
							"file_id":        "dir-docs",
							"parent_file_id": "root",
						},
					},
				})
			case "dir-docs":
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"items": []map[string]interface{}{},
				})
			default:
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"items": []map[string]interface{}{},
				})
			}
		case "/adrive/v1.0/openFile/create":
			payload := mustDecode(r)
			if stringMapValue(payload, "type") == "folder" {
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"name":           stringMapValue(payload, "name"),
					"type":           "folder",
					"file_id":        "dir-created",
					"parent_file_id": stringMapValue(payload, "parent_file_id"),
				})
				return
			}
			if stringMapValue(payload, "content_hash") == "sha1-hit" {
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"name":              stringMapValue(payload, "name"),
					"type":              "file",
					"file_id":           "file-rapid",
					"parent_file_id":    stringMapValue(payload, "parent_file_id"),
					"size":              int64MapValue(payload, "size"),
					"rapid_upload":      true,
					"content_hash_name": "sha1",
					"content_hash":      "sha1-hit",
				})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"name":           stringMapValue(payload, "name"),
				"type":           "file",
				"file_id":        "file-uploaded",
				"parent_file_id": stringMapValue(payload, "parent_file_id"),
				"upload_id":      "upload-1",
				"rapid_upload":   false,
				"part_info_list": []map[string]interface{}{
					{
						"part_number": 1,
						"upload_url":  baseURL + "/upload/part/1",
					},
				},
			})
		case "/v2/file/complete":
			payload := mustDecode(r)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"name":              "upload.bin",
				"type":              "file",
				"file_id":           stringMapValue(payload, "file_id"),
				"parent_file_id":    "dir-docs",
				"size":              len(uploaded),
				"content_hash_name": "sha1",
				"content_hash":      "sha1-uploaded",
				"status":            "available",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	baseURL = server.URL

	return server, &uploaded
}

func ioReadAll(r *http.Request) ([]byte, error) {
	defer func() { _ = r.Body.Close() }()
	return io.ReadAll(r.Body)
}
