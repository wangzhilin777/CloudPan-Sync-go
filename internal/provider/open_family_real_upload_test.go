package provider

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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
		t.Fatalf("expected no binary upload body for rapid upload, got %#v", *uploaded)
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
	if len(*uploaded) != 1 {
		t.Fatalf("expected one upload part, got %#v", *uploaded)
	}
	if got := string((*uploaded)[0]); got != "hello-aliyun-upload" {
		t.Fatalf("expected uploaded body hello-aliyun-upload, got %q", got)
	}
	if stringMapValue(result.Payload, "fileId") != "file-uploaded" {
		t.Fatalf("expected file-uploaded result, got %+v", result.Payload)
	}
	if got := int64MapValue(result.Payload, "partCount"); got != 1 {
		t.Fatalf("expected partCount 1, got %+v", result.Payload)
	}
}

func TestOpenFamilyAdapterUploadsAliyunFileByMultipart(t *testing.T) {
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
	localPath := filepath.Join(tmpDir, "multipart.bin")
	content := []byte("abcdefghijklmnopqrstuvwxyz")
	if err := os.WriteFile(localPath, content, 0o600); err != nil {
		t.Fatalf("write local file: %v", err)
	}

	originalSize := aliyunOpenDefaultPartSize
	aliyunOpenDefaultPartSize = 10
	t.Cleanup(func() { aliyunOpenDefaultPartSize = originalSize })

	result := entry.Adapter.Upload(UploadRequest{
		Profile:        profile,
		Path:           "/docs/multipart.bin",
		Name:           "multipart.bin",
		Size:           int64(len(content)),
		LocalPath:      localPath,
		ConflictPolicy: ConflictPolicyAutoRenameNew,
		Strategy:       "download_upload",
	})
	if !result.OK {
		t.Fatalf("expected multipart upload success, got %+v", result)
	}
	if len(*uploaded) != 3 {
		t.Fatalf("expected 3 upload parts, got %#v", *uploaded)
	}
	if got := string((*uploaded)[0]); got != "abcdefghij" {
		t.Fatalf("unexpected first part: %q", got)
	}
	if got := string((*uploaded)[1]); got != "klmnopqrst" {
		t.Fatalf("unexpected second part: %q", got)
	}
	if got := string((*uploaded)[2]); got != "uvwxyz" {
		t.Fatalf("unexpected third part: %q", got)
	}
	if got := int64MapValue(result.Payload, "partCount"); got != 3 {
		t.Fatalf("expected partCount 3, got %+v", result.Payload)
	}
	if got := uploadedPartEvidenceLen(result.Payload["uploadedParts"]); got != 3 {
		t.Fatalf("expected uploadedParts evidence len 3, got %#v", result.Payload["uploadedParts"])
	}
	if got := int64MapValue(result.Payload, "uploadedPartCount"); got != 3 {
		t.Fatalf("expected uploadedPartCount 3, got %+v", result.Payload)
	}
	if got := int64MapValue(result.Payload, "nextPartNumber"); got != 4 {
		t.Fatalf("expected nextPartNumber 4, got %+v", result.Payload)
	}
}

func TestOpenFamilyAdapterReturnsMultipartFailureEvidence(t *testing.T) {
	server, _ := newAliyunOpenUploadFailingPartTestServer(t, 2)
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
	localPath := filepath.Join(tmpDir, "multipart-fail.bin")
	content := []byte("abcdefghijklmnopqrstuvwxyz")
	if err := os.WriteFile(localPath, content, 0o600); err != nil {
		t.Fatalf("write local file: %v", err)
	}

	originalSize := aliyunOpenDefaultPartSize
	aliyunOpenDefaultPartSize = 10
	t.Cleanup(func() { aliyunOpenDefaultPartSize = originalSize })

	result := entry.Adapter.Upload(UploadRequest{
		Profile:        profile,
		Path:           "/docs/multipart-fail.bin",
		Name:           "multipart-fail.bin",
		Size:           int64(len(content)),
		LocalPath:      localPath,
		ConflictPolicy: ConflictPolicyAutoRenameNew,
		Strategy:       "download_upload",
	})
	if result.OK {
		t.Fatalf("expected multipart upload failure, got %+v", result)
	}
	if result.Status != "provider_request_failed" {
		t.Fatalf("expected provider_request_failed, got %+v", result)
	}
	if got := int64MapValue(result.Payload, "failedPartNumber"); got != 2 {
		t.Fatalf("expected failedPartNumber 2, got %+v", result.Payload)
	}
	if got := int64MapValue(result.Payload, "nextPartNumber"); got != 2 {
		t.Fatalf("expected nextPartNumber 2, got %+v", result.Payload)
	}
	if got := int64MapValue(result.Payload, "uploadedPartCount"); got != 0 {
		t.Fatalf("expected uploadedPartCount absent or 0 on failure, got %+v", result.Payload)
	}
	if got := uploadedPartEvidenceLen(result.Payload["uploadedParts"]); got != 1 {
		t.Fatalf("expected one uploaded part evidence, got %#v", result.Payload["uploadedParts"])
	}
}

func TestOpenFamilyAdapterResumesMultipartUploadFromCheckpoint(t *testing.T) {
	server, uploaded := newAliyunOpenResumeUploadTestServer(t)
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
	localPath := filepath.Join(tmpDir, "multipart-resume.bin")
	content := []byte("abcdefghijklmnopqrstuvwxyz")
	if err := os.WriteFile(localPath, content, 0o600); err != nil {
		t.Fatalf("write local file: %v", err)
	}

	originalSize := aliyunOpenDefaultPartSize
	aliyunOpenDefaultPartSize = 10
	t.Cleanup(func() { aliyunOpenDefaultPartSize = originalSize })

	result := entry.Adapter.Upload(UploadRequest{
		Profile:        profile,
		Path:           "/docs/multipart-resume.bin",
		Name:           "multipart-resume.bin",
		Size:           int64(len(content)),
		LocalPath:      localPath,
		ConflictPolicy: ConflictPolicyAutoRenameNew,
		Strategy:       "download_upload",
		ResumeUpload: &ResumeUpload{
			FileID:         "file-uploaded",
			UploadID:       "upload-1",
			PartCount:      3,
			NextPartNumber: 2,
			UploadedParts: []map[string]interface{}{
				{"partNumber": 1, "start": 0, "end": 10, "size": 10},
			},
		},
	})
	if !result.OK {
		t.Fatalf("expected resumed multipart upload success, got %+v", result)
	}
	if len(*uploaded) != 2 {
		t.Fatalf("expected 2 resumed upload parts, got %#v", *uploaded)
	}
	if got := string((*uploaded)[0]); got != "klmnopqrst" {
		t.Fatalf("unexpected resumed second part: %q", got)
	}
	if got := string((*uploaded)[1]); got != "uvwxyz" {
		t.Fatalf("unexpected resumed third part: %q", got)
	}
	if resumed, _ := result.Payload["resumedUpload"].(bool); !resumed {
		t.Fatalf("expected resumedUpload payload, got %+v", result.Payload)
	}
	if got := int64MapValue(result.Payload, "uploadedPartCount"); got != 3 {
		t.Fatalf("expected resumed uploadedPartCount 3, got %+v", result.Payload)
	}
	if got := uploadedPartEvidenceLen(result.Payload["uploadedParts"]); got != 3 {
		t.Fatalf("expected resumed uploadedParts evidence len 3, got %#v", result.Payload["uploadedParts"])
	}
}

func newAliyunOpenUploadTestServer(t *testing.T) (*httptest.Server, *[][]byte) {
	t.Helper()

	uploaded := make([][]byte, 0)
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
		if r.Method == http.MethodPut && strings.HasPrefix(r.URL.Path, "/upload/part/") {
			body, err := ioReadAll(r)
			if err != nil {
				t.Fatalf("read upload body: %v", err)
			}
			uploaded = append(uploaded, body)
			partID := strings.TrimPrefix(r.URL.Path, "/upload/part/")
			w.Header().Set("ETag", "\"etag-upload-"+partID+"\"")
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
				"part_info_list": buildUploadPartInfoForTest(baseURL, partInfoMapSlice(payload, "part_info_list")),
			})
		case "/v2/file/complete":
			payload := mustDecode(r)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"name":              "upload.bin",
				"type":              "file",
				"file_id":           stringMapValue(payload, "file_id"),
				"parent_file_id":    "dir-docs",
				"size":              totalUploadedLength(uploaded),
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

func newAliyunOpenUploadFailingPartTestServer(t *testing.T, failingPart int) (*httptest.Server, *[][]byte) {
	t.Helper()

	uploaded := make([][]byte, 0)
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
		if r.Method == http.MethodPut && strings.HasPrefix(r.URL.Path, "/upload/part/") {
			partID := strings.TrimPrefix(r.URL.Path, "/upload/part/")
			if partID == strconv.Itoa(failingPart) {
				http.Error(w, "part failed", http.StatusInternalServerError)
				return
			}
			body, err := ioReadAll(r)
			if err != nil {
				t.Fatalf("read upload body: %v", err)
			}
			uploaded = append(uploaded, body)
			w.Header().Set("ETag", "\"etag-upload-"+partID+"\"")
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
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"domain_id": "bj1"})
		case "/v2/drive/get_default_drive":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"drive_id": "drive-1"})
		case "/adrive/v1.0/openFile/list":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"items": []map[string]interface{}{{"name": "docs", "type": "folder", "file_id": "dir-docs", "parent_file_id": "root"}},
			})
		case "/adrive/v1.0/openFile/create":
			payload := mustDecode(r)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"name":           stringMapValue(payload, "name"),
				"type":           "file",
				"file_id":        "file-uploaded",
				"parent_file_id": stringMapValue(payload, "parent_file_id"),
				"upload_id":      "upload-1",
				"rapid_upload":   false,
				"part_info_list": buildUploadPartInfoForTest(baseURL, partInfoMapSlice(payload, "part_info_list")),
			})
		default:
			http.NotFound(w, r)
		}
	}))
	baseURL = server.URL

	return server, &uploaded
}

func newAliyunOpenResumeUploadTestServer(t *testing.T) (*httptest.Server, *[][]byte) {
	t.Helper()

	uploaded := make([][]byte, 0)
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
		if r.Method == http.MethodPut && strings.HasPrefix(r.URL.Path, "/upload/part/") {
			body, err := ioReadAll(r)
			if err != nil {
				t.Fatalf("read upload body: %v", err)
			}
			uploaded = append(uploaded, body)
			partID := strings.TrimPrefix(r.URL.Path, "/upload/part/")
			w.Header().Set("ETag", "\"etag-upload-"+partID+"\"")
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
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"domain_id": "bj1"})
		case "/v2/drive/get_default_drive":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"drive_id": "drive-1"})
		case "/adrive/v1.0/openFile/list":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"items": []map[string]interface{}{{"name": "docs", "type": "folder", "file_id": "dir-docs", "parent_file_id": "root"}},
			})
		case "/v2/file/get_upload_url":
			payload := mustDecode(r)
			if stringMapValue(payload, "file_id") != "file-uploaded" || stringMapValue(payload, "upload_id") != "upload-1" {
				t.Fatalf("unexpected resume payload: %+v", payload)
			}
			partInfo := partInfoMapSlice(payload, "part_info_list")
			if len(partInfo) != 2 || int64MapValue(partInfo[0], "part_number") != 2 || int64MapValue(partInfo[1], "part_number") != 3 {
				t.Fatalf("unexpected resume part_info_list: %+v", partInfo)
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"file_id":        "file-uploaded",
				"upload_id":      "upload-1",
				"part_info_list": buildUploadPartInfoForTest(baseURL, partInfo),
			})
		case "/v2/file/complete":
			payload := mustDecode(r)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"name":              "multipart-resume.bin",
				"type":              "file",
				"file_id":           stringMapValue(payload, "file_id"),
				"parent_file_id":    "dir-docs",
				"size":              26,
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

func buildUploadPartInfoForTest(baseURL string, requested []map[string]interface{}) []map[string]interface{} {
	if len(requested) == 0 {
		requested = []map[string]interface{}{{"part_number": 1}}
	}
	items := make([]map[string]interface{}, 0, len(requested))
	for idx, item := range requested {
		partNumber := int64MapValue(item, "part_number")
		if partNumber <= 0 {
			partNumber = int64(idx + 1)
		}
		items = append(items, map[string]interface{}{
			"part_number": partNumber,
			"upload_url":  baseURL + "/upload/part/" + strconv.FormatInt(partNumber, 10),
		})
	}
	return items
}

func totalUploadedLength(parts [][]byte) int {
	total := 0
	for _, part := range parts {
		total += len(part)
	}
	return total
}

func uploadedPartEvidenceLen(raw interface{}) int {
	switch typed := raw.(type) {
	case []map[string]interface{}:
		return len(typed)
	case []interface{}:
		return len(typed)
	default:
		return 0
	}
}

func ioReadAll(r *http.Request) ([]byte, error) {
	defer func() { _ = r.Body.Close() }()
	return io.ReadAll(r.Body)
}
