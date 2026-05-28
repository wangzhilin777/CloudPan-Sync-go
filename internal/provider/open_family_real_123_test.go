package provider

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenFamilyAdapterValidates123OpenAgainstLiveEndpoint(t *testing.T) {
	server, _ := newPan123OpenTestServer(t)
	t.Cleanup(server.Close)

	originalClient := providerHTTPClient
	providerHTTPClient = server.Client()
	t.Cleanup(func() { providerHTTPClient = originalClient })

	registry := NewRegistry(DefaultCatalog()...)
	entry, ok := registry.Get("123_open")
	if !ok {
		t.Fatal("expected 123_open entry")
	}

	result := entry.Adapter.ValidateAuth(AuthProfile{
		ProviderKey: "123_open",
		Token:       "token-live",
		Extra: map[string]string{
			"apiEndpoint": server.URL,
		},
	})
	if !result.OK {
		t.Fatalf("expected validation success, got %+v", result)
	}
	if result.Mode != "open_family_real_auth" {
		t.Fatalf("expected open_family_real_auth mode, got %s", result.Mode)
	}
}

func TestOpenFamilyAdapterListsAndReads123OpenDirectory(t *testing.T) {
	server, _ := newPan123OpenTestServer(t)
	t.Cleanup(server.Close)

	originalClient := providerHTTPClient
	providerHTTPClient = server.Client()
	t.Cleanup(func() { providerHTTPClient = originalClient })

	registry := NewRegistry(DefaultCatalog()...)
	entry, ok := registry.Get("123_open")
	if !ok {
		t.Fatal("expected 123_open entry")
	}
	profile := AuthProfile{
		ProviderKey: "123_open",
		Token:       "token-live",
		Extra: map[string]string{
			"apiEndpoint": server.URL,
		},
	}

	list := entry.Adapter.List(ListRequest{
		Profile:  profile,
		Path:     "/docs",
		PageSize: 100,
	})
	if !list.OK {
		t.Fatalf("expected list success, got %+v", list)
	}
	if list.Mode != "open_family_real_directory" {
		t.Fatalf("expected open_family_real_directory mode, got %s", list.Mode)
	}
	if len(list.Items) != 2 {
		t.Fatalf("expected 2 docs items, got %d", len(list.Items))
	}
	guide := findProviderItemByPath(list.Items, "/docs/guide.md")
	if guide == nil || stringMapValue(guide, "md5") != "etag-guide" {
		t.Fatalf("expected /docs/guide.md metadata, got %+v", guide)
	}

	metadata := entry.Adapter.Metadata(MetadataRequest{
		Profile: profile,
		Path:    "/docs/guide.md",
	})
	if !metadata.OK || metadata.Status != "exists" {
		t.Fatalf("expected metadata exists, got %+v", metadata)
	}
	if stringMapValue(metadata.Entry, "fileId") != "file-guide" {
		t.Fatalf("expected file-guide metadata, got %+v", metadata.Entry)
	}

	byID := entry.Adapter.Metadata(MetadataRequest{
		Profile:  profile,
		Path:     "/docs/guide.md",
		FileID:   "file-guide",
		ParentID: "dir-docs",
	})
	if !byID.OK || stringMapValue(byID.Entry, "fileId") != "file-guide" {
		t.Fatalf("expected metadata by file id, got %+v", byID)
	}

	createDir := entry.Adapter.CreateDir(CreateDirRequest{
		Profile:  profile,
		ParentID: "dir-docs",
		DirName:  "uploads",
	})
	if !createDir.OK {
		t.Fatalf("expected create dir success, got %+v", createDir)
	}
	if stringMapValue(createDir.Payload, "fileId") != "dir-new" {
		t.Fatalf("expected dir-new result, got %+v", createDir.Payload)
	}
}

func TestOpenFamilyAdapterUploads123OpenFileWithOverwriteDowngrade(t *testing.T) {
	server, state := newPan123OpenTestServer(t)
	t.Cleanup(server.Close)

	originalClient := providerHTTPClient
	providerHTTPClient = server.Client()
	t.Cleanup(func() { providerHTTPClient = originalClient })

	registry := NewRegistry(DefaultCatalog()...)
	entry, ok := registry.Get("123_open")
	if !ok {
		t.Fatal("expected 123_open entry")
	}
	profile := AuthProfile{
		ProviderKey: "123_open",
		Token:       "token-live",
		Extra: map[string]string{
			"apiEndpoint": server.URL,
		},
	}

	tmpDir := t.TempDir()
	localPath := filepath.Join(tmpDir, "upload.bin")
	content := []byte("hello-123pan-upload")
	if err := os.WriteFile(localPath, content, 0o600); err != nil {
		t.Fatalf("write local file: %v", err)
	}

	result := entry.Adapter.Upload(UploadRequest{
		Profile:        profile,
		Path:           "/docs/upload.bin",
		Name:           "upload.bin",
		Size:           int64(len(content)),
		LocalPath:      localPath,
		ConflictPolicy: ConflictPolicyOverwriteExisting,
		Strategy:       "download_upload",
	})
	if !result.OK {
		t.Fatalf("expected upload success, got %+v", result)
	}
	if result.Mode != "open_family_real_upload" {
		t.Fatalf("expected open_family_real_upload mode, got %s", result.Mode)
	}
	if result.ConflictAction != "overwrite_downgraded_to_auto_rename" {
		t.Fatalf("expected overwrite downgrade, got %s", result.ConflictAction)
	}
	if state.lastCreatedFilename != "upload (1).bin" {
		t.Fatalf("expected renamed upload target, got %q", state.lastCreatedFilename)
	}
	if got := string(state.uploadedBody); got != string(content) {
		t.Fatalf("expected uploaded content %q, got %q", string(content), got)
	}
	if stringMapValue(result.Payload, "fileId") != "file-uploaded" {
		t.Fatalf("expected file-uploaded, got %+v", result.Payload)
	}
	if !boolMapValue(result.Payload, "verifyOk") {
		t.Fatalf("expected verifyOk true, got %+v", result.Payload)
	}
	if stringMapValue(result.Payload, "verifyMode") != "metadata_by_file_id" {
		t.Fatalf("expected metadata_by_file_id verification, got %+v", result.Payload)
	}
}

func TestOpenFamilyAdapterFastUploads123OpenFileWhenProviderReusesHash(t *testing.T) {
	server, state := newPan123OpenTestServer(t)
	state.reuseCreate = true
	t.Cleanup(server.Close)

	originalClient := providerHTTPClient
	providerHTTPClient = server.Client()
	t.Cleanup(func() { providerHTTPClient = originalClient })

	registry := NewRegistry(DefaultCatalog()...)
	entry, ok := registry.Get("123_open")
	if !ok {
		t.Fatal("expected 123_open entry")
	}
	profile := AuthProfile{
		ProviderKey: "123_open",
		Token:       "token-live",
		Extra: map[string]string{
			"apiEndpoint": server.URL,
		},
	}

	result := entry.Adapter.Upload(UploadRequest{
		Profile:        profile,
		Path:           "/docs/rapid.bin",
		Name:           "rapid.bin",
		Size:           1234,
		MD5:            "abcdef0123456789abcdef0123456789",
		ConflictPolicy: ConflictPolicyAutoRenameNew,
		Strategy:       "fast_upload",
	})
	if !result.OK {
		t.Fatalf("expected fast upload success, got %+v", result)
	}
	if !boolMapValue(result.Payload, "rapidUpload") {
		t.Fatalf("expected rapidUpload true, got %+v", result.Payload)
	}
	if stringMapValue(result.Payload, "fileId") != "file-uploaded" {
		t.Fatalf("expected file-uploaded, got %+v", result.Payload)
	}
	if string(state.uploadedBody) != "" {
		t.Fatalf("did not expect binary PUT during fast upload, got %q", string(state.uploadedBody))
	}
	if stringMapValue(result.Payload, "verifyMode") != "metadata_by_file_id" {
		t.Fatalf("expected metadata_by_file_id verification, got %+v", result.Payload)
	}
}

func TestOpenFamilyAdapterFastUploads123OpenFileReportsHashMiss(t *testing.T) {
	server, state := newPan123OpenTestServer(t)
	t.Cleanup(server.Close)

	originalClient := providerHTTPClient
	providerHTTPClient = server.Client()
	t.Cleanup(func() { providerHTTPClient = originalClient })

	registry := NewRegistry(DefaultCatalog()...)
	entry, ok := registry.Get("123_open")
	if !ok {
		t.Fatal("expected 123_open entry")
	}
	profile := AuthProfile{
		ProviderKey: "123_open",
		Token:       "token-live",
		Extra: map[string]string{
			"apiEndpoint": server.URL,
		},
	}

	result := entry.Adapter.Upload(UploadRequest{
		Profile:        profile,
		Path:           "/docs/miss.bin",
		Name:           "miss.bin",
		Size:           5678,
		MD5:            "0123456789abcdef0123456789abcdef",
		ConflictPolicy: ConflictPolicyAutoRenameNew,
		Strategy:       "fast_upload",
	})
	if result.OK || result.Status != "hash_miss" {
		t.Fatalf("expected hash_miss, got %+v", result)
	}
	if boolMapValue(result.Payload, "rapidUpload") {
		t.Fatalf("did not expect rapidUpload true, got %+v", result.Payload)
	}
	if state.lastCreatedFilename != "miss.bin" {
		t.Fatalf("expected fast upload create target miss.bin, got %q", state.lastCreatedFilename)
	}
	if string(state.uploadedBody) != "" {
		t.Fatalf("did not expect binary PUT during hash miss, got %q", string(state.uploadedBody))
	}
}

type pan123OpenTestState struct {
	lastCreatedFilename string
	uploadedBody        []byte
	reuseCreate         bool
	lastCreatedSize     int64
	lastCreatedMD5      string
}

func newPan123OpenTestServer(t *testing.T) (*httptest.Server, *pan123OpenTestState) {
	t.Helper()

	state := &pan123OpenTestState{}

	rootItems := []map[string]interface{}{
		{
			"fileId":       "dir-docs",
			"parentFileId": "0",
			"filename":     "docs",
			"type":         1,
			"size":         0,
		},
	}
	docsItems := []map[string]interface{}{
		{
			"fileId":       "file-guide",
			"parentFileId": "dir-docs",
			"filename":     "guide.md",
			"type":         0,
			"size":         12,
			"etag":         "etag-guide",
		},
		{
			"fileId":       "file-existing-upload",
			"parentFileId": "dir-docs",
			"filename":     "upload.bin",
			"type":         0,
			"size":         8,
			"etag":         "etag-existing",
		},
	}

	mustAuth := func(w http.ResponseWriter, r *http.Request) bool {
		t.Helper()
		if got := r.Header.Get("Authorization"); got != "Bearer token-live" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return false
		}
		if got := r.Header.Get("Platform"); got != "open_platform" {
			t.Fatalf("expected Platform open_platform, got %q", got)
		}
		return true
	}

	mustDecode := func(r *http.Request) map[string]interface{} {
		t.Helper()
		var payload map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		return payload
	}

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/upload-put/") {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("read put body: %v", err)
			}
			state.uploadedBody = body
			w.Header().Set("ETag", "\"etag-uploaded\"")
			w.WriteHeader(http.StatusOK)
			return
		}
		if !mustAuth(w, r) {
			return
		}
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2/file/list":
			parentID := r.URL.Query().Get("parentFileId")
			switch parentID {
			case "0":
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"data": map[string]interface{}{
						"fileList": rootItems,
					},
				})
			case "dir-docs":
				items := docsItems
				if state.lastCreatedFilename != "" {
					fileSize := int64(len(state.uploadedBody))
					if state.lastCreatedSize > 0 {
						fileSize = state.lastCreatedSize
					}
					etag := "etag-uploaded"
					if state.lastCreatedMD5 != "" {
						etag = state.lastCreatedMD5
					}
					items = append(items, map[string]interface{}{
						"fileId":       "file-uploaded",
						"parentFileId": "dir-docs",
						"filename":     state.lastCreatedFilename,
						"type":         0,
						"size":         fileSize,
						"etag":         etag,
					})
				}
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"data": map[string]interface{}{
						"fileList": items,
					},
				})
			default:
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"data": map[string]interface{}{
						"fileList": []map[string]interface{}{},
					},
				})
			}
		case r.Method == http.MethodPost && r.URL.Path == "/upload/v1/file/mkdir":
			payload := mustDecode(r)
			if stringMapValue(payload, "name") != "uploads" {
				t.Fatalf("expected mkdir uploads, got %+v", payload)
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"data": map[string]interface{}{
					"dirID": "dir-new",
				},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/upload/v1/oss/file/create":
			payload := mustDecode(r)
			state.lastCreatedFilename = stringMapValue(payload, "filename")
			state.lastCreatedSize = int64MapValue(payload, "size")
			state.lastCreatedMD5 = stringMapValue(payload, "etag")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"data": map[string]interface{}{
					"preuploadID": "preupload-1",
					"fileID":      "file-uploaded",
					"reuse":       state.reuseCreate,
				},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/upload/v1/oss/file/get_upload_url":
			payload := mustDecode(r)
			if stringMapValue(payload, "preuploadID") != "preupload-1" {
				t.Fatalf("expected preupload-1, got %+v", payload)
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"data": map[string]interface{}{
					"presignedURL": server.URL + "/upload-put/preupload-1",
				},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/upload/v1/oss/file/upload_complete":
			payload := mustDecode(r)
			if stringMapValue(payload, "preuploadID") != "preupload-1" {
				t.Fatalf("expected upload_complete preupload-1, got %+v", payload)
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"data": map[string]interface{}{
					"completed": true,
					"fileID":    "file-uploaded",
				},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/upload/v1/oss/file/upload_async_result":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"data": map[string]interface{}{
					"completed": true,
					"fileID":    "file-uploaded",
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))

	return server, state
}
