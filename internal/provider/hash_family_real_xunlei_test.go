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

func TestHashFamilyAdapterValidatesXunleiAgainstLiveEndpoint(t *testing.T) {
	server, _ := newXunleiHashFamilyTestServer(t)
	t.Cleanup(server.Close)

	originalClient := providerHTTPClient
	providerHTTPClient = server.Client()
	t.Cleanup(func() { providerHTTPClient = originalClient })

	registry := NewRegistry(DefaultCatalog()...)
	entry, ok := registry.Get("xunlei")
	if !ok {
		t.Fatal("expected xunlei entry")
	}

	result := entry.Adapter.ValidateAuth(AuthProfile{
		ProviderKey: "xunlei",
		Token:       "token-live",
		Extra: map[string]string{
			"apiEndpoint": server.URL,
			"deviceId":    "device-1",
		},
	})
	if !result.OK {
		t.Fatalf("expected validation success, got %+v", result)
	}
	if result.Mode != "hash_family_real_auth" {
		t.Fatalf("expected hash_family_real_auth mode, got %s", result.Mode)
	}
}

func TestHashFamilyAdapterListsReadsAndCreatesXunleiDirectory(t *testing.T) {
	server, _ := newXunleiHashFamilyTestServer(t)
	t.Cleanup(server.Close)

	originalClient := providerHTTPClient
	providerHTTPClient = server.Client()
	t.Cleanup(func() { providerHTTPClient = originalClient })

	registry := NewRegistry(DefaultCatalog()...)
	entry, ok := registry.Get("xunlei")
	if !ok {
		t.Fatal("expected xunlei entry")
	}
	profile := AuthProfile{
		ProviderKey: "xunlei",
		Token:       "token-live",
		Extra: map[string]string{
			"apiEndpoint": server.URL,
			"deviceId":    "device-1",
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
	if list.Mode != "hash_family_real_directory" {
		t.Fatalf("expected hash_family_real_directory mode, got %s", list.Mode)
	}
	if len(list.Items) != 2 {
		t.Fatalf("expected 2 docs items, got %d", len(list.Items))
	}
	guide := findProviderItemByPath(list.Items, "/docs/guide.txt")
	if guide == nil || stringMapValue(guide, "gcid") != "GCID-GUIDE" {
		t.Fatalf("expected guide.txt metadata, got %+v", guide)
	}

	metadata := entry.Adapter.Metadata(MetadataRequest{
		Profile: profile,
		Path:    "/docs/guide.txt",
	})
	if !metadata.OK || metadata.Status != "exists" {
		t.Fatalf("expected metadata exists, got %+v", metadata)
	}
	if stringMapValue(metadata.Entry, "fileId") != "file-guide" {
		t.Fatalf("expected file-guide metadata, got %+v", metadata.Entry)
	}

	byID := entry.Adapter.Metadata(MetadataRequest{
		Profile:  profile,
		Path:     "/docs/guide.txt",
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

func TestHashFamilyAdapterRapidUploadsXunleiFile(t *testing.T) {
	server, state := newXunleiHashFamilyTestServer(t)
	t.Cleanup(server.Close)

	originalClient := providerHTTPClient
	providerHTTPClient = server.Client()
	t.Cleanup(func() { providerHTTPClient = originalClient })

	registry := NewRegistry(DefaultCatalog()...)
	entry, ok := registry.Get("xunlei")
	if !ok {
		t.Fatal("expected xunlei entry")
	}
	profile := AuthProfile{
		ProviderKey: "xunlei",
		Token:       "token-live",
		Extra: map[string]string{
			"apiEndpoint": server.URL,
			"deviceId":    "device-1",
		},
	}

	result := entry.Adapter.Upload(UploadRequest{
		Profile:        profile,
		Path:           "/docs/upload.bin",
		Name:           "upload.bin",
		ParentID:       "dir-docs",
		Size:           1024,
		GCID:           "GCID-RAPID",
		ConflictPolicy: ConflictPolicyOverwriteExisting,
		Strategy:       "fast_upload",
	})
	if !result.OK {
		t.Fatalf("expected rapid upload success, got %+v", result)
	}
	if result.Mode != "hash_family_real_upload" {
		t.Fatalf("expected hash_family_real_upload mode, got %s", result.Mode)
	}
	if result.ConflictAction != "overwrite_downgraded_to_auto_rename" {
		t.Fatalf("expected overwrite downgrade, got %s", result.ConflictAction)
	}
	if state.lastCreatedFilename != "upload (1).bin" {
		t.Fatalf("expected renamed upload target, got %q", state.lastCreatedFilename)
	}
	if stringMapValue(result.Payload, "fileId") != "file-uploaded" {
		t.Fatalf("expected file-uploaded, got %+v", result.Payload)
	}
	if boolMapValue(result.Payload, "usedBinaryFallback") {
		t.Fatalf("expected rapid path without binary fallback, got %+v", result.Payload)
	}
}

func TestHashFamilyAdapterFallsBackToResumableUploadForXunlei(t *testing.T) {
	server, state := newXunleiHashFamilyTestServer(t)
	t.Cleanup(server.Close)

	originalClient := providerHTTPClient
	providerHTTPClient = server.Client()
	t.Cleanup(func() { providerHTTPClient = originalClient })

	registry := NewRegistry(DefaultCatalog()...)
	entry, ok := registry.Get("xunlei")
	if !ok {
		t.Fatal("expected xunlei entry")
	}
	profile := AuthProfile{
		ProviderKey: "xunlei",
		Token:       "token-live",
		Extra: map[string]string{
			"apiEndpoint": server.URL,
			"deviceId":    "device-1",
		},
	}

	tmpDir := t.TempDir()
	localPath := filepath.Join(tmpDir, "upload.bin")
	if err := os.WriteFile(localPath, []byte("xunlei-binary-fallback"), 0o600); err != nil {
		t.Fatalf("write local file: %v", err)
	}

	result := entry.Adapter.Upload(UploadRequest{
		Profile:        profile,
		Path:           "/docs/binary.bin",
		Name:           "binary.bin",
		ParentID:       "dir-docs",
		LocalPath:      localPath,
		ConflictPolicy: ConflictPolicyAutoRenameNew,
		Strategy:       "download_upload",
	})
	if !result.OK {
		t.Fatalf("expected resumable fallback upload success, got %+v", result)
	}
	if !state.binaryFallbackUsed {
		t.Fatalf("expected binary fallback uploader to be used")
	}
	if string(state.lastUploadedBody) != "xunlei-binary-fallback" {
		t.Fatalf("expected uploaded fallback body, got %q", string(state.lastUploadedBody))
	}
	if !boolMapValue(result.Payload, "usedBinaryFallback") {
		t.Fatalf("expected usedBinaryFallback true, got %+v", result.Payload)
	}
	if stringMapValue(result.Payload, "verifyMode") != "metadata_by_file_id" {
		t.Fatalf("expected metadata_by_file_id verification, got %+v", result.Payload)
	}
}

func TestHashFamilyAdapterResumesExistingResumableSessionForXunlei(t *testing.T) {
	server, state := newXunleiHashFamilyTestServer(t)
	state.failUploadOnce = true
	t.Cleanup(server.Close)

	originalClient := providerHTTPClient
	providerHTTPClient = server.Client()
	t.Cleanup(func() { providerHTTPClient = originalClient })

	registry := NewRegistry(DefaultCatalog()...)
	entry, ok := registry.Get("xunlei")
	if !ok {
		t.Fatal("expected xunlei entry")
	}
	profile := AuthProfile{
		ProviderKey: "xunlei",
		Token:       "token-live",
		Extra: map[string]string{
			"apiEndpoint": server.URL,
			"deviceId":    "device-1",
		},
	}

	localPath := filepath.Join(t.TempDir(), "resume.bin")
	if err := os.WriteFile(localPath, []byte("xunlei-resume-binary"), 0o600); err != nil {
		t.Fatalf("write local file: %v", err)
	}

	first := entry.Adapter.Upload(UploadRequest{
		Profile:        profile,
		Path:           "/docs/resume.bin",
		Name:           "resume.bin",
		ParentID:       "dir-docs",
		LocalPath:      localPath,
		ConflictPolicy: ConflictPolicyAutoRenameNew,
		Strategy:       "download_upload",
	})
	if first.OK {
		t.Fatalf("expected first resumable upload to fail, got %+v", first)
	}
	providerData, _ := first.Payload["providerData"].(map[string]interface{})
	if providerData == nil {
		t.Fatalf("expected providerData in failed payload, got %+v", first.Payload)
	}
	resumable, _ := providerData["resumable"].(map[string]interface{})
	if resumable == nil {
		t.Fatalf("expected resumable providerData, got %+v", first.Payload)
	}
	if got := intMapValue(first.Payload, "partCount"); got != 1 {
		t.Fatalf("expected whole-object checkpoint partCount 1, got %+v", first.Payload)
	}
	if got := intMapValue(first.Payload, "failedPartNumber"); got != 1 {
		t.Fatalf("expected whole-object checkpoint failedPartNumber 1, got %+v", first.Payload)
	}
	if got := intMapValue(first.Payload, "nextPartNumber"); got != 1 {
		t.Fatalf("expected whole-object checkpoint nextPartNumber 1, got %+v", first.Payload)
	}
	if got := stringMapValue(first.Payload, "uploadId"); got != "folder/demo.bin" {
		t.Fatalf("expected resumable uploadId fallback from object key, got %+v", first.Payload)
	}

	second := entry.Adapter.Upload(UploadRequest{
		Profile:        profile,
		Path:           "/docs/resume.bin",
		Name:           "resume.bin",
		ParentID:       "dir-docs",
		LocalPath:      localPath,
		ConflictPolicy: ConflictPolicyAutoRenameNew,
		Strategy:       "download_upload",
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
	if got := intMapValue(second.Payload, "uploadedPartCount"); got != 1 {
		t.Fatalf("expected resumed whole-object uploadedPartCount 1, got %+v", second.Payload)
	}
	if got := intMapValue(second.Payload, "nextPartNumber"); got != 2 {
		t.Fatalf("expected resumed whole-object nextPartNumber 2, got %+v", second.Payload)
	}
	if state.createUploadCount != 1 {
		t.Fatalf("expected create upload only once, state=%+v", state)
	}
	if state.uploadAttemptCount != 2 {
		t.Fatalf("expected two upload attempts, state=%+v", state)
	}
}

type xunleiHashFamilyTestState struct {
	lastCreatedFilename string
	lastUploadedBody    []byte
	binaryFallbackUsed  bool
	createUploadCount   int
	uploadAttemptCount  int
	failUploadOnce      bool
}

func newXunleiHashFamilyTestServer(t *testing.T) (*httptest.Server, *xunleiHashFamilyTestState) {
	t.Helper()

	state := &xunleiHashFamilyTestState{}

	rootItems := []map[string]interface{}{
		{
			"id":        "dir-docs",
			"parent_id": "",
			"name":      "docs",
			"kind":      "drive#folder",
			"size":      0,
		},
	}
	docsItems := func() []map[string]interface{} {
		items := []map[string]interface{}{
			{
				"id":        "file-guide",
				"parent_id": "dir-docs",
				"name":      "guide.txt",
				"kind":      "drive#file",
				"size":      12,
				"hash":      "GCID-GUIDE",
			},
			{
				"id":        "file-existing-upload",
				"parent_id": "dir-docs",
				"name":      "upload.bin",
				"kind":      "drive#file",
				"size":      8,
				"hash":      "GCID-EXISTING",
			},
		}
		if state.lastCreatedFilename != "" {
			hashValue := "GCID-RAPID"
			if state.binaryFallbackUsed {
				hashValue = strings.ToUpper("56C5A6DFA4EB724E8A7194C792D0C7C2C52F396A")
			}
			items = append(items, map[string]interface{}{
				"id":        "file-uploaded",
				"parent_id": "dir-docs",
				"name":      state.lastCreatedFilename,
				"kind":      "drive#file",
				"size":      19,
				"hash":      hashValue,
			})
		}
		return items
	}

	mustAuth := func(w http.ResponseWriter, r *http.Request) bool {
		t.Helper()
		if got := r.Header.Get("Authorization"); got != "Bearer token-live" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return false
		}
		if got := r.Header.Get("x-client-id"); got != xunleiClientID {
			t.Fatalf("expected x-client-id %q, got %q", xunleiClientID, got)
		}
		if got := r.Header.Get("x-device-id"); got != "device-1" {
			t.Fatalf("expected x-device-id device-1, got %q", got)
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
		if r.Method == http.MethodPut && r.URL.Path == "/bucket-xl/folder/demo.bin" {
			state.uploadAttemptCount++
			if state.failUploadOnce && state.uploadAttemptCount == 1 {
				http.Error(w, "temporary upload failure", http.StatusInternalServerError)
				return
			}
			state.binaryFallbackUsed = true
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("read s3 fallback body: %v", err)
			}
			state.lastUploadedBody = body
			if !strings.HasPrefix(r.Header.Get("Authorization"), "AWS4-HMAC-SHA256 ") {
				t.Fatalf("expected aws sigv4 authorization, got %q", r.Header.Get("Authorization"))
			}
			w.WriteHeader(http.StatusOK)
			return
		}
		if !mustAuth(w, r) {
			return
		}
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/drive/v1/files":
			parentID := r.URL.Query().Get("parent_id")
			switch parentID {
			case "":
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"files": rootItems,
				})
			case "dir-docs":
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"files": docsItems(),
				})
			default:
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"files": []map[string]interface{}{},
				})
			}
		case r.Method == http.MethodPost && r.URL.Path == "/drive/v1/files":
			payload := mustDecode(r)
			kind := stringMapValue(payload, "kind")
			if kind == "drive#folder" {
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"id":        "dir-new",
					"parent_id": stringMapValue(payload, "parent_id"),
					"name":      stringMapValue(payload, "name"),
					"kind":      "drive#folder",
				})
				return
			}
			state.createUploadCount++
			state.lastCreatedFilename = stringMapValue(payload, "name")
			hashValue := strings.ToUpper(stringMapValue(payload, "hash"))
			if hashValue == "GCID-RAPID" {
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"upload_type": "UPLOAD_TYPE_RESUMABLE",
					"file": map[string]interface{}{
						"id":        "file-uploaded",
						"parent_id": "dir-docs",
						"name":      state.lastCreatedFilename,
						"kind":      "drive#file",
						"size":      1024,
						"hash":      "GCID-RAPID",
					},
				})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"upload_type": "UPLOAD_TYPE_RESUMABLE",
				"file": map[string]interface{}{
					"id":        "file-uploaded",
					"parent_id": "dir-docs",
					"name":      state.lastCreatedFilename,
					"kind":      "drive#file",
					"size":      19,
					"hash":      hashValue,
				},
				"resumable": map[string]interface{}{
					"provider": "S3",
					"params": map[string]interface{}{
						"access_key_id":     "key",
						"access_key_secret": "secret",
						"security_token":    "token",
						"bucket":            "bucket-xl",
						"endpoint":          server.URL,
						"key":               "folder/demo.bin",
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))

	return server, state
}
