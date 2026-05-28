package provider

import (
	"encoding/json"
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

func TestBaiduFamilyAdapterValidatesAgainstLiveEndpoint(t *testing.T) {
	server, _ := newBaiduFamilyTestServer(t)
	t.Cleanup(server.Close)

	originalClient := providerHTTPClient
	providerHTTPClient = server.Client()
	t.Cleanup(func() { providerHTTPClient = originalClient })

	registry := NewRegistry(DefaultCatalog()...)
	entry, ok := registry.Get("baidu_netdisk")
	if !ok {
		t.Fatal("expected baidu_netdisk entry")
	}

	result := entry.Adapter.ValidateAuth(AuthProfile{
		ProviderKey: "baidu_netdisk",
		Token:       "token-live",
		Extra: map[string]string{
			"apiEndpoint": server.URL,
			"pcsEndpoint": server.URL,
		},
	})
	if !result.OK {
		t.Fatalf("expected validation success, got %+v", result)
	}
	if result.Mode != "baidu_family_real_auth" {
		t.Fatalf("expected baidu_family_real_auth mode, got %s", result.Mode)
	}
}

func TestBaiduFamilyAdapterListsReadsAndCreatesDirectory(t *testing.T) {
	server, _ := newBaiduFamilyTestServer(t)
	t.Cleanup(server.Close)

	originalClient := providerHTTPClient
	providerHTTPClient = server.Client()
	t.Cleanup(func() { providerHTTPClient = originalClient })

	registry := NewRegistry(DefaultCatalog()...)
	entry, ok := registry.Get("baidu_netdisk")
	if !ok {
		t.Fatal("expected baidu_netdisk entry")
	}
	profile := AuthProfile{
		ProviderKey: "baidu_netdisk",
		Token:       "token-live",
		Extra: map[string]string{
			"apiEndpoint": server.URL,
			"pcsEndpoint": server.URL,
		},
	}

	list := entry.Adapter.List(ListRequest{
		Profile:  profile,
		Path:     "/docs",
		ParentID: "/docs",
		PageSize: 100,
	})
	if !list.OK {
		t.Fatalf("expected list success, got %+v", list)
	}
	if list.Mode != "baidu_family_real_directory" {
		t.Fatalf("expected baidu_family_real_directory mode, got %s", list.Mode)
	}
	if len(list.Items) != 2 {
		t.Fatalf("expected 2 docs items, got %d", len(list.Items))
	}
	guide := findProviderItemByPath(list.Items, "/docs/guide.txt")
	if guide == nil || stringMapValue(guide, "md5") != "abcdefabcdefabcdefabcdefabcdefab" {
		t.Fatalf("expected guide metadata, got %+v", guide)
	}

	metadata := entry.Adapter.Metadata(MetadataRequest{
		Profile: profile,
		Path:    "/docs/guide.txt",
		FileID:  "1111",
	})
	if !metadata.OK || metadata.Status != "exists" {
		t.Fatalf("expected metadata exists, got %+v", metadata)
	}
	if stringMapValue(metadata.Entry, "fileId") != "1111" {
		t.Fatalf("expected file 1111, got %+v", metadata.Entry)
	}

	createDir := entry.Adapter.CreateDir(CreateDirRequest{
		Profile:  profile,
		ParentID: "/docs",
		DirName:  "uploads",
	})
	if !createDir.OK {
		t.Fatalf("expected create dir success, got %+v", createDir)
	}
	if stringMapValue(createDir.Payload, "fileId") != "3333" {
		t.Fatalf("expected dir fsid 3333, got %+v", createDir.Payload)
	}
}

func TestBaiduFamilyAdapterUploadsBinaryFile(t *testing.T) {
	server, state := newBaiduFamilyTestServer(t)
	t.Cleanup(server.Close)

	originalClient := providerHTTPClient
	providerHTTPClient = server.Client()
	t.Cleanup(func() { providerHTTPClient = originalClient })

	registry := NewRegistry(DefaultCatalog()...)
	entry, ok := registry.Get("baidu_netdisk")
	if !ok {
		t.Fatal("expected baidu_netdisk entry")
	}
	profile := AuthProfile{
		ProviderKey: "baidu_netdisk",
		Token:       "token-live",
		Extra: map[string]string{
			"apiEndpoint": server.URL,
			"pcsEndpoint": server.URL,
		},
	}

	tmpDir := t.TempDir()
	localPath := filepath.Join(tmpDir, "upload.bin")
	content := []byte("hello-baidu-upload")
	if err := os.WriteFile(localPath, content, 0o600); err != nil {
		t.Fatalf("write local file: %v", err)
	}

	result := entry.Adapter.Upload(UploadRequest{
		Profile:        profile,
		Path:           "/docs/upload.bin",
		Name:           "upload.bin",
		ParentID:       "/docs",
		LocalPath:      localPath,
		ConflictPolicy: ConflictPolicyOverwriteExisting,
		Strategy:       "download_upload",
	})
	if !result.OK {
		t.Fatalf("expected upload success, got %+v", result)
	}
	if result.Mode != "baidu_family_real_upload" {
		t.Fatalf("expected baidu_family_real_upload mode, got %+v", result)
	}
	if result.ConflictAction != "overwrite_downgraded_to_auto_rename" {
		t.Fatalf("expected overwrite downgrade, got %s", result.ConflictAction)
	}
	if state.lastCreatedPath != "/docs/upload (1).bin" {
		t.Fatalf("expected renamed path, got %q", state.lastCreatedPath)
	}
	if string(state.lastTmpfileBody) != string(content) {
		t.Fatalf("expected tmpfile body %q, got %q", string(content), string(state.lastTmpfileBody))
	}
	if stringMapValue(result.Payload, "fileId") != "4444" {
		t.Fatalf("expected uploaded file id, got %+v", result.Payload)
	}
	if stringMapValue(result.Payload, "verifyMode") != "metadata_by_file_id" {
		t.Fatalf("expected metadata_by_file_id verification, got %+v", result.Payload)
	}
	if intMapValue(result.Payload, "uploadedPartCount") != 1 {
		t.Fatalf("expected uploadedPartCount 1, got %+v", result.Payload)
	}
	if intMapValue(result.Payload, "failedPartNumber") != 0 {
		t.Fatalf("expected failedPartNumber 0, got %+v", result.Payload)
	}
	if intMapValue(result.Payload, "nextPartNumber") != 2 {
		t.Fatalf("expected nextPartNumber 2, got %+v", result.Payload)
	}
	if uploadedPartEvidenceLen(result.Payload["uploadedParts"]) != 1 {
		t.Fatalf("expected one uploaded part evidence, got %+v", result.Payload)
	}
}

func TestBaiduFamilyAdapterRecordsWholeObjectCheckpointOnTmpfileFailure(t *testing.T) {
	server, state := newBaiduFamilyTestServer(t)
	state.failTmpfileOnce = true
	t.Cleanup(server.Close)

	originalClient := providerHTTPClient
	providerHTTPClient = server.Client()
	t.Cleanup(func() { providerHTTPClient = originalClient })

	registry := NewRegistry(DefaultCatalog()...)
	entry, ok := registry.Get("baidu_netdisk")
	if !ok {
		t.Fatal("expected baidu_netdisk entry")
	}
	profile := AuthProfile{
		ProviderKey: "baidu_netdisk",
		Token:       "token-live",
		Extra: map[string]string{
			"apiEndpoint": server.URL,
			"pcsEndpoint": server.URL,
		},
	}

	tmpDir := t.TempDir()
	localPath := filepath.Join(tmpDir, "upload.bin")
	if err := os.WriteFile(localPath, []byte("hello-baidu-upload"), 0o600); err != nil {
		t.Fatalf("write local file: %v", err)
	}

	result := entry.Adapter.Upload(UploadRequest{
		Profile:        profile,
		Path:           "/docs/upload.bin",
		Name:           "upload.bin",
		ParentID:       "/docs",
		LocalPath:      localPath,
		ConflictPolicy: ConflictPolicyOverwriteExisting,
		Strategy:       "download_upload",
	})
	if result.OK {
		t.Fatalf("expected tmpfile failure, got %+v", result)
	}
	if stringMapValue(result.Payload, "uploadId") != "upload-1" {
		t.Fatalf("expected uploadId evidence, got %+v", result.Payload)
	}
	if intMapValue(result.Payload, "partCount") != 1 {
		t.Fatalf("expected partCount 1, got %+v", result.Payload)
	}
	if intMapValue(result.Payload, "uploadedPartCount") != 0 {
		t.Fatalf("expected uploadedPartCount 0, got %+v", result.Payload)
	}
	if intMapValue(result.Payload, "failedPartNumber") != 1 {
		t.Fatalf("expected failedPartNumber 1, got %+v", result.Payload)
	}
	if intMapValue(result.Payload, "nextPartNumber") != 1 {
		t.Fatalf("expected nextPartNumber 1, got %+v", result.Payload)
	}
}

type baiduFamilyTestState struct {
	lastCreatedPath string
	lastTmpfileBody []byte
	failTmpfileOnce bool
}

func newBaiduFamilyTestServer(t *testing.T) (*httptest.Server, *baiduFamilyTestState) {
	t.Helper()

	state := &baiduFamilyTestState{}

	rootItems := []map[string]interface{}{
		{
			"fs_id":           "1000",
			"path":            "/docs",
			"server_filename": "docs",
			"isdir":           1,
			"size":            0,
		},
	}
	docsItems := func() []map[string]interface{} {
		items := []map[string]interface{}{
			{
				"fs_id":           "1111",
				"path":            "/docs/guide.txt",
				"server_filename": "guide.txt",
				"isdir":           0,
				"size":            12,
				"md5":             "abcdefabcdefabcdefabcdefabcdefab",
			},
			{
				"fs_id":           "2222",
				"path":            "/docs/upload.bin",
				"server_filename": "upload.bin",
				"isdir":           0,
				"size":            8,
				"md5":             "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			},
		}
		if state.lastCreatedPath != "" {
			items = append(items, map[string]interface{}{
				"fs_id":           "4444",
				"path":            state.lastCreatedPath,
				"server_filename": inferName(state.lastCreatedPath, "upload.bin"),
				"isdir":           0,
				"size":            18,
				"md5":             "4c7fb2d4ac8976657e13e231ad091db5",
			})
		}
		return items
	}

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

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !mustAccess(w, r) {
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == baiduXPanFilePath && r.URL.Query().Get("method") == "list":
			switch r.URL.Query().Get("dir") {
			case "/":
				_ = json.NewEncoder(w).Encode(map[string]interface{}{"list": rootItems})
			case "/docs":
				_ = json.NewEncoder(w).Encode(map[string]interface{}{"list": docsItems()})
			default:
				_ = json.NewEncoder(w).Encode(map[string]interface{}{"list": []map[string]interface{}{}})
			}
		case r.Method == http.MethodGet && r.URL.Path == baiduXPanFilePath && r.URL.Query().Get("method") == "filemetas":
			if r.URL.Query().Get("fsids") == "[1111]" {
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"info": []map[string]interface{}{
						{
							"fs_id":           "1111",
							"path":            "/docs/guide.txt",
							"server_filename": "guide.txt",
							"isdir":           0,
							"size":            12,
							"md5":             "abcdefabcdefabcdefabcdefabcdefab",
						},
					},
				})
				return
			}
			if r.URL.Query().Get("fsids") == "[4444]" {
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"info": []map[string]interface{}{
						{
							"fs_id":           "4444",
							"path":            state.lastCreatedPath,
							"server_filename": inferName(state.lastCreatedPath, "upload.bin"),
							"isdir":           0,
							"size":            18,
							"md5":             "4c7fb2d4ac8976657e13e231ad091db5",
						},
					},
				})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"info": []map[string]interface{}{}})
		case r.Method == http.MethodPost && r.URL.Path == baiduXPanFilePath && r.URL.Query().Get("method") == "create":
			form := mustParseForm(r)
			if form.Get("isdir") == "1" {
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"fs_id": "3333",
					"path":  form.Get("path"),
				})
				return
			}
			state.lastCreatedPath = form.Get("path")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"fs_id": "4444",
				"path":  form.Get("path"),
			})
		case r.Method == http.MethodPost && r.URL.Path == baiduXPanFilePath && r.URL.Query().Get("method") == "precreate":
			form := mustParseForm(r)
			if form.Get("path") == "" {
				t.Fatalf("expected precreate path")
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"uploadid": "upload-1",
			})
		case r.Method == http.MethodPost && r.URL.Path == baiduPCSUploadPath && r.URL.Query().Get("method") == "upload":
			if state.failTmpfileOnce {
				state.failTmpfileOnce = false
				http.Error(w, "temporary tmpfile failure", http.StatusInternalServerError)
				return
			}
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
			body, err := io.ReadAll(part)
			if err != nil {
				t.Fatalf("read multipart body: %v", err)
			}
			state.lastTmpfileBody = body
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"md5": "4c7fb2d4ac8976657e13e231ad091db5",
			})
		default:
			http.NotFound(w, r)
		}
	}))

	return server, state
}
