package provider

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
)

func TestGuangyaFamilyAdapterRequiresToken(t *testing.T) {
	registry := NewRegistry(DefaultCatalog()...)
	entry, ok := registry.Get("guangya")
	if !ok {
		t.Fatal("expected guangya entry")
	}

	result := entry.Adapter.ValidateAuth(AuthProfile{
		ProviderKey: "guangya",
	})
	if result.OK {
		t.Fatal("expected validation to fail without token")
	}
	if result.Status != "missing_access_token" {
		t.Fatalf("expected missing_access_token, got %s", result.Status)
	}
}

func TestGuangyaFamilyAdapterLiveDirectoryAndFastCheck(t *testing.T) {
	server, state := newGuangyaTestServer(t)
	t.Cleanup(server.Close)

	originalClient := providerHTTPClient
	providerHTTPClient = server.Client()
	t.Cleanup(func() { providerHTTPClient = originalClient })

	registry := NewRegistry(DefaultCatalog()...)
	entry, ok := registry.Get("guangya")
	if !ok {
		t.Fatal("expected guangya entry")
	}
	profile := guangyaTestProfile(server.URL)

	validation := entry.Adapter.ValidateAuth(profile)
	if !validation.OK {
		t.Fatalf("expected validation success, got %+v", validation)
	}

	list := entry.Adapter.List(ListRequest{
		Profile:  profile,
		ParentID: "parent-gy",
		PageSize: 100,
	})
	if !list.OK {
		t.Fatalf("expected list success, got %+v", list)
	}
	if len(list.Items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(list.Items))
	}
	guide := findProviderItemByPath(list.Items, "guide.txt")
	if guide == nil || stringMapValue(guide, "fileId") != "file-guide-gy" {
		t.Fatalf("expected guide file, got %+v", guide)
	}

	metadata := entry.Adapter.Metadata(MetadataRequest{
		Profile: profile,
		FileID:  "file-guide-gy",
	})
	if !metadata.OK || metadata.Status != "exists" {
		t.Fatalf("expected metadata exists, got %+v", metadata)
	}
	if stringMapValue(metadata.Entry, "md5") != "0123456789abcdef0123456789abcdef" {
		t.Fatalf("expected normalized md5, got %+v", metadata.Entry)
	}

	createDir := entry.Adapter.CreateDir(CreateDirRequest{
		Profile:  profile,
		ParentID: "parent-gy",
		DirName:  "uploads",
	})
	if !createDir.OK {
		t.Fatalf("expected create dir success, got %+v", createDir)
	}
	if stringMapValue(createDir.Payload, "fileId") != "dir-new-gy" {
		t.Fatalf("expected dir-new-gy, got %+v", createDir.Payload)
	}

	check := entry.Adapter.FastUploadCheck(FastUploadCheckRequest{
		Profile:  profile,
		ParentID: "parent-gy",
		Path:     "/guide.txt",
		Name:     "guide.txt",
		Size:     12,
		MD5:      guangyaBase64MD5Token("0123456789abcdef0123456789abcdef"),
	})
	if !check.OK || !check.Candidate {
		t.Fatalf("expected fast upload candidate, got %+v", check)
	}
	if !boolMapValue(check.Payload, "canFastUpload") {
		t.Fatalf("expected canFastUpload true, got %+v", check.Payload)
	}
	if state.deletedTaskID != "" {
		t.Fatalf("did not expect cleanup after instant hit, got %s", state.deletedTaskID)
	}
}

func TestGuangyaFamilyAdapterFastCheckGCIDMissCleansTaskAndUploadPreflightIsHonest(t *testing.T) {
	server, state := newGuangyaTestServer(t)
	t.Cleanup(server.Close)

	originalClient := providerHTTPClient
	providerHTTPClient = server.Client()
	t.Cleanup(func() { providerHTTPClient = originalClient })

	registry := NewRegistry(DefaultCatalog()...)
	entry, ok := registry.Get("guangya")
	if !ok {
		t.Fatal("expected guangya entry")
	}
	profile := guangyaTestProfile(server.URL)

	check := entry.Adapter.FastUploadCheck(FastUploadCheckRequest{
		Profile:  profile,
		ParentID: "parent-gy",
		Path:     "/big.bin",
		Name:     "big.bin",
		Size:     1024,
		GCID:     "1111111111111111111111111111111111111111",
	})
	if !check.OK || !check.Candidate {
		t.Fatalf("expected gcid candidate, got %+v", check)
	}
	if boolMapValue(check.Payload, "canFastUpload") {
		t.Fatalf("expected provider miss, got %+v", check.Payload)
	}
	if state.deletedTaskID != "task-gcid-gy" {
		t.Fatalf("expected cleanup of task-gcid-gy, got %s", state.deletedTaskID)
	}

	upload := entry.Adapter.Upload(UploadRequest{
		Profile:  profile,
		ParentID: "parent-gy",
		Path:     "/big.bin",
		Name:     "big.bin",
		Size:     1024,
		Strategy: "download_upload",
	})
	if upload.OK {
		t.Fatalf("expected honest preflight-only upload result, got %+v", upload)
	}
	if upload.Status != "missing_binary_upload_runtime" {
		t.Fatalf("expected missing_binary_upload_runtime, got %+v", upload)
	}
}

func TestGuangyaFamilyAdapterUploadFastHitSucceedsAndVerifiesByList(t *testing.T) {
	server, _ := newGuangyaTestServer(t)
	t.Cleanup(server.Close)

	originalClient := providerHTTPClient
	providerHTTPClient = server.Client()
	t.Cleanup(func() { providerHTTPClient = originalClient })

	registry := NewRegistry(DefaultCatalog()...)
	entry, ok := registry.Get("guangya")
	if !ok {
		t.Fatal("expected guangya entry")
	}
	profile := guangyaTestProfile(server.URL)

	result := entry.Adapter.Upload(UploadRequest{
		Profile:  profile,
		ParentID: "parent-gy",
		Path:     "/guide.txt",
		Name:     "guide.txt",
		Size:     12,
		MD5:      guangyaBase64MD5Token("0123456789abcdef0123456789abcdef"),
		Strategy: "fast_upload",
	})
	if !result.OK {
		t.Fatalf("expected fast upload success, got %+v", result)
	}
	if result.ConflictAction != "overwrite_downgraded_to_auto_rename" {
		t.Fatalf("expected overwrite downgrade conflict action, got %+v", result)
	}
	if stringMapValue(result.Payload, "user.resolvedTargetName") != "guide (2).txt" {
		t.Fatalf("expected renamed target, got %+v", result.Payload)
	}
	if stringMapValue(result.Payload, "drive.verifyMode") != "list_by_parent_name" {
		t.Fatalf("expected verify by list, got %+v", result.Payload)
	}
	if !boolMapValue(result.Payload, "drive.verifyOk") {
		t.Fatalf("expected verify ok, got %+v", result.Payload)
	}
}

func TestGuangyaFamilyAdapterUploadGCIDFlashHitUsesUploadInfoAndMetadata(t *testing.T) {
	server, state := newGuangyaTestServer(t)
	state.flashHit = true
	state.uploadInfoFileID = "file-uploaded-gcid-gy"
	t.Cleanup(server.Close)

	originalClient := providerHTTPClient
	providerHTTPClient = server.Client()
	t.Cleanup(func() { providerHTTPClient = originalClient })

	registry := NewRegistry(DefaultCatalog()...)
	entry, ok := registry.Get("guangya")
	if !ok {
		t.Fatal("expected guangya entry")
	}
	profile := guangyaTestProfile(server.URL)
	profile.Extra["uploadInfoEndpoint"] = server.URL + "/upload_info"

	result := entry.Adapter.Upload(UploadRequest{
		Profile:  profile,
		ParentID: "parent-gy",
		Path:     "/flash.bin",
		Name:     "flash.bin",
		Size:     2048,
		GCID:     "1111111111111111111111111111111111111111",
		Strategy: "fast_upload",
	})
	if !result.OK {
		t.Fatalf("expected gcid flash upload success, got %+v", result)
	}
	if stringMapValue(result.Payload, "drive.verifyMode") != "metadata_by_file_id" {
		t.Fatalf("expected verify by metadata, got %+v", result.Payload)
	}
	if stringMapValue(result.Payload, "fileId") != "" {
		t.Fatalf("did not expect top-level fileId without explicit mapping, got %+v", result.Payload)
	}
	uploadInfo, _ := result.Payload["drive.uploadInfoResponse"].(map[string]interface{})
	if stringMapValue(uploadInfo, "fileId") != "file-uploaded-gcid-gy" {
		t.Fatalf("expected uploadInfo file id, got %+v", result.Payload)
	}
	if state.deletedTaskID != "" {
		t.Fatalf("did not expect delete task on flash hit, got %s", state.deletedTaskID)
	}
}

func TestGuangyaFamilyAdapterUploadSmallBinaryUsesUploadInfo(t *testing.T) {
	server, state := newGuangyaTestServer(t)
	state.uploadInfoFileID = "file-small-upload-gy"
	t.Cleanup(server.Close)

	originalClient := providerHTTPClient
	providerHTTPClient = server.Client()
	t.Cleanup(func() { providerHTTPClient = originalClient })

	localFile := writeGuangyaTempFile(t, 256*1024)

	registry := NewRegistry(DefaultCatalog()...)
	entry, ok := registry.Get("guangya")
	if !ok {
		t.Fatal("expected guangya entry")
	}
	profile := guangyaTestProfile(server.URL)

	result := entry.Adapter.Upload(UploadRequest{
		Profile:   profile,
		ParentID:  "parent-gy",
		Path:      "/small.bin",
		Name:      "small.bin",
		LocalPath: localFile,
		Strategy:  "download_upload",
	})
	if !result.OK {
		t.Fatalf("expected small binary upload success, got %+v", result)
	}
	if !state.smallUploadUsedMD5 {
		t.Fatalf("expected small upload to send md5 in upload_token, state=%+v", state)
	}
	if stringMapValue(result.Payload, "drive.verifyMode") != "metadata_by_file_id" {
		t.Fatalf("expected verify by metadata, got %+v", result.Payload)
	}
}

func TestGuangyaFamilyAdapterUploadMultipartFallbackCompletes(t *testing.T) {
	server, state := newGuangyaTestServer(t)
	state.multipartMode = true
	state.uploadInfoFileID = "file-multipart-upload-gy"
	t.Cleanup(server.Close)

	originalClient := providerHTTPClient
	providerHTTPClient = server.Client()
	t.Cleanup(func() { providerHTTPClient = originalClient })

	localFile := writeGuangyaTempFile(t, 2*1024*1024)

	registry := NewRegistry(DefaultCatalog()...)
	entry, ok := registry.Get("guangya")
	if !ok {
		t.Fatal("expected guangya entry")
	}
	profile := guangyaTestProfile(server.URL)

	result := entry.Adapter.Upload(UploadRequest{
		Profile:   profile,
		ParentID:  "parent-gy",
		Path:      "/multipart.bin",
		Name:      "multipart.bin",
		LocalPath: localFile,
		Strategy:  "download_upload",
	})
	if !result.OK {
		t.Fatalf("expected multipart upload success, got %+v", result)
	}
	if !state.ossInitiated || !state.ossCompleted {
		t.Fatalf("expected oss multipart lifecycle, state=%+v", state)
	}
	if state.ossPartCount == 0 {
		t.Fatalf("expected uploaded parts, state=%+v", state)
	}
	if !boolMapValue(result.Payload, "drive.usedBinaryFallback") {
		t.Fatalf("expected usedBinaryFallback true, got %+v", result.Payload)
	}
	if stringMapValue(result.Payload, "drive.verifyMode") != "metadata_by_file_id" {
		t.Fatalf("expected verify by metadata, got %+v", result.Payload)
	}
}

func TestGuangyaFamilyAdapterResumesMultipartFallbackFromCheckpoint(t *testing.T) {
	server, state := newGuangyaTestServer(t)
	state.multipartMode = true
	state.failPartNumberOnce = 2
	state.uploadInfoFileID = "file-multipart-upload-gy"
	t.Cleanup(server.Close)

	originalClient := providerHTTPClient
	providerHTTPClient = server.Client()
	t.Cleanup(func() { providerHTTPClient = originalClient })

	localFile := writeGuangyaTempFile(t, 6*1024*1024)

	registry := NewRegistry(DefaultCatalog()...)
	entry, ok := registry.Get("guangya")
	if !ok {
		t.Fatal("expected guangya entry")
	}
	profile := guangyaTestProfile(server.URL)

	first := entry.Adapter.Upload(UploadRequest{
		Profile:   profile,
		ParentID:  "parent-gy",
		Path:      "/resume.bin",
		Name:      "resume.bin",
		LocalPath: localFile,
		Strategy:  "download_upload",
	})
	if first.OK {
		t.Fatalf("expected first multipart upload to fail for resume scenario, got %+v", first)
	}
	uploadCheckpoint, _ := first.Payload["drive.multipartUpload"].(map[string]interface{})
	if uploadCheckpoint == nil {
		t.Fatalf("expected multipart upload checkpoint payload, got %+v", first.Payload)
	}
	if stringMapValue(uploadCheckpoint, "uploadId") != "upload-oss-gy" {
		t.Fatalf("expected upload checkpoint uploadId, got %+v", first.Payload)
	}
	if int(int64MapValue(uploadCheckpoint, "failedPartNumber")) != 2 {
		t.Fatalf("expected failedPartNumber 2, got %+v", first.Payload)
	}
	if int(int64MapValue(uploadCheckpoint, "nextPartNumber")) != 2 {
		t.Fatalf("expected nextPartNumber 2, got %+v", first.Payload)
	}
	uploadedPartCount := 0
	switch typed := uploadCheckpoint["uploadedParts"].(type) {
	case []interface{}:
		uploadedPartCount = len(typed)
	case []map[string]interface{}:
		uploadedPartCount = len(typed)
	}
	if uploadedPartCount != 1 {
		t.Fatalf("expected one uploaded part before failure, got %+v", first.Payload)
	}

	resume := &ResumeUpload{
		UploadID:          stringMapValue(uploadCheckpoint, "uploadId"),
		PartCount:         2,
		UploadedPartCount: 1,
		FailedPartNumber:  2,
		NextPartNumber:    2,
		UploadedParts: []map[string]interface{}{
			{"partNumber": 1, "etag": "etag-part-gy"},
		},
	}
	second := entry.Adapter.Upload(UploadRequest{
		Profile:      profile,
		ParentID:     "parent-gy",
		Path:         "/resume.bin",
		Name:         "resume.bin",
		LocalPath:    localFile,
		Strategy:     "download_upload",
		ResumeUpload: resume,
	})
	if !second.OK {
		t.Fatalf("expected resumed multipart upload success, got %+v", second)
	}
	secondCheckpoint, _ := second.Payload["drive.multipartUpload"].(map[string]interface{})
	if secondCheckpoint == nil {
		t.Fatalf("expected multipart upload payload after resume, got %+v", second.Payload)
	}
	if !boolMapValue(secondCheckpoint, "resumedUpload") {
		t.Fatalf("expected resumedUpload true, got %+v", second.Payload)
	}
	if state.ossInitiatedCount != 1 {
		t.Fatalf("expected only one multipart initiate across resume flow, state=%+v", state)
	}
	if state.partUploadAttempts[1] != 1 || state.partUploadAttempts[2] != 2 {
		t.Fatalf("expected part1 once and part2 retried, state=%+v", state)
	}
}

type guangyaTestServerState struct {
	deletedTaskID      string
	flashHit           bool
	uploadInfoFileID   string
	instantHitUploaded bool
	smallUploadUsedMD5 bool
	multipartMode      bool
	ossInitiated       bool
	ossCompleted       bool
	ossPartCount       int
	ossInitiatedCount  int
	failPartNumberOnce int
	partUploadAttempts map[int]int
}

func newGuangyaTestServer(t *testing.T) (*httptest.Server, *guangyaTestServerState) {
	t.Helper()
	state := &guangyaTestServerState{
		partUploadAttempts: map[int]int{},
	}
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/oss/") {
			switch r.URL.Path {
			case "/oss/upload/multipart.bin":
				switch r.Method {
				case http.MethodPost:
					if _, ok := r.URL.Query()["uploads"]; ok {
						state.ossInitiated = true
						state.ossInitiatedCount++
						w.Header().Set("Content-Type", "application/xml")
						_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><InitiateMultipartUploadResult><UploadId>upload-oss-gy</UploadId></InitiateMultipartUploadResult>`))
						return
					}
					if r.URL.Query().Get("uploadId") == "upload-oss-gy" {
						state.ossCompleted = true
						w.Header().Set("Content-Type", "application/xml")
						_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><CompleteMultipartUploadResult><ETag>etag-complete-gy</ETag></CompleteMultipartUploadResult>`))
						return
					}
				case http.MethodPut:
					partNumber, _ := strconv.Atoi(r.URL.Query().Get("partNumber"))
					state.partUploadAttempts[partNumber]++
					if state.failPartNumberOnce > 0 && partNumber == state.failPartNumberOnce && state.partUploadAttempts[partNumber] == 1 {
						http.Error(w, "multipart part failed", http.StatusInternalServerError)
						return
					}
					state.ossPartCount++
					w.Header().Set("ETag", `"etag-part-gy"`)
					w.WriteHeader(http.StatusOK)
					return
				}
			}
			http.NotFound(w, r)
			return
		}
		if got := r.Header.Get("Authorization"); got != "Bearer token-guangya" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/list":
			items := []map[string]interface{}{
				{"fileId": "dir-docs-gy", "name": "docs", "isDir": true, "parentId": "parent-gy"},
				{"fileId": "file-guide-gy", "name": "guide.txt", "size": 12, "md5": "0123456789abcdef0123456789abcdef", "parentId": "parent-gy"},
				{"fileId": "file-guide-renamed-gy", "name": "guide (1).txt", "size": 12, "md5": "0123456789abcdef0123456789abcdef", "parentId": "parent-gy"},
			}
			if state.instantHitUploaded {
				items = append(items, map[string]interface{}{
					"fileId":   "file-guide-renamed-2-gy",
					"name":     "guide (2).txt",
					"size":     12,
					"md5":      "0123456789abcdef0123456789abcdef",
					"parentId": "parent-gy",
				})
			}
			if state.flashHit {
				items = append(items, map[string]interface{}{
					"fileId":   "file-uploaded-gcid-gy",
					"name":     "flash.bin",
					"size":     2048,
					"md5":      "fedcba9876543210fedcba9876543210",
					"parentId": "parent-gy",
				})
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"data": map[string]interface{}{
					"items": items,
				},
			})
		case "/meta":
			var payload map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode meta body: %v", err)
			}
			fileID := strings.TrimSpace(stringMapValue(payload, "fileId"))
			switch fileID {
			case "file-uploaded-gcid-gy":
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"data": map[string]interface{}{
						"fileId":   "file-uploaded-gcid-gy",
						"filename": "flash.bin",
						"size":     2048,
						"md5":      "fedcba9876543210fedcba9876543210",
						"gcid":     "1111111111111111111111111111111111111111",
					},
				})
			case "file-small-upload-gy":
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"data": map[string]interface{}{
						"fileId":   "file-small-upload-gy",
						"filename": "small.bin",
						"size":     256 * 1024,
						"md5":      "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					},
				})
			case "file-multipart-upload-gy":
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"data": map[string]interface{}{
						"fileId":   "file-multipart-upload-gy",
						"filename": "multipart.bin",
						"size":     2 * 1024 * 1024,
						"md5":      "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
					},
				})
			default:
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"data": map[string]interface{}{
						"fileId":   "file-guide-gy",
						"filename": "guide.txt",
						"size":     12,
						"md5":      "0123456789abcdef0123456789abcdef",
						"gcid":     "1111111111111111111111111111111111111111",
					},
				})
			}
		case "/mkdir":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"data": map[string]interface{}{
					"dirId": "dir-new-gy",
				},
			})
		case "/res_token":
			var payload map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode res token body: %v", err)
			}
			res, _ := payload["res"].(map[string]interface{})
			if strings.TrimSpace(stringMapValue(res, "md5")) != "" {
				if strings.Contains(strings.TrimSpace(stringMapValue(payload, "name")), "guide") {
					state.instantHitUploaded = true
					_ = json.NewEncoder(w).Encode(map[string]interface{}{
						"code":   156,
						"taskId": "task-md5-gy",
					})
					return
				}
				state.smallUploadUsedMD5 = true
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"data": map[string]interface{}{
						"taskId": "task-small-gy",
					},
				})
				return
			}
			if state.multipartMode {
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"data": map[string]interface{}{
						"taskId":       "task-multipart-gy",
						"bucketName":   "bucket-gy",
						"objectPath":   "upload/multipart.bin",
						"fullEndPoint": server.URL + "/oss",
						"creds": map[string]interface{}{
							"accessKeyID":     "access-gy",
							"secretAccessKey": "secret-gy",
							"sessionToken":    "session-gy",
						},
					},
				})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"code":   0,
				"taskId": "task-gcid-gy",
			})
		case "/flash_check":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"data": map[string]interface{}{
					"canFlashUpload": state.flashHit,
				},
			})
		case "/upload_info":
			switch r.URL.Query().Get("uploadId") {
			default:
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"fileId": state.uploadInfoFileID,
				"data": map[string]interface{}{
					"fileId": state.uploadInfoFileID,
				},
			})
		case "/delete_task":
			var payload map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode delete task body: %v", err)
			}
			taskIDs, _ := payload["taskIds"].([]interface{})
			if len(taskIDs) > 0 {
				state.deletedTaskID = strings.TrimSpace(taskIDs[0].(string))
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"code": 0,
			})
		default:
			http.NotFound(w, r)
		}
	}))
	return server, state
}

func guangyaTestProfile(baseURL string) AuthProfile {
	return AuthProfile{
		ProviderKey: "guangya",
		Token:       "token-guangya",
		Extra: map[string]string{
			"parentId":                    "parent-gy",
			"fileId":                      "file-guide-gy",
			"listEndpoint":                baseURL + "/list",
			"metadataEndpoint":            baseURL + "/meta",
			"createDirEndpoint":           baseURL + "/mkdir",
			"resCenterTokenEndpoint":      baseURL + "/res_token",
			"checkCanFlashUploadEndpoint": baseURL + "/flash_check",
			"deleteUploadTaskEndpoint":    baseURL + "/delete_task",
			"uploadInfoEndpoint":          baseURL + "/upload_info",
		},
	}
}

func guangyaBase64MD5Token(hexMD5 string) string {
	raw := make([]byte, 16)
	for idx := 0; idx < 16; idx++ {
		value, _ := strconv.ParseUint(hexMD5[idx*2:idx*2+2], 16, 8)
		raw[idx] = byte(value)
	}
	return strings.TrimRight(base64.StdEncoding.EncodeToString(raw), "=")
}

func writeGuangyaTempFile(t *testing.T, size int) string {
	t.Helper()
	file, err := os.CreateTemp(t.TempDir(), "guangya-upload-*")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	defer file.Close()
	content := bytes.Repeat([]byte("a"), size)
	if _, err := file.Write(content); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	return file.Name()
}
