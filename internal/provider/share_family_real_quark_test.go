package provider

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestShareFamilyAdapterValidatesQuarkAgainstLiveEndpoint(t *testing.T) {
	server, _ := newShareFamilyTestServer(t, "quark")
	t.Cleanup(server.Close)

	originalClient := providerHTTPClient
	providerHTTPClient = server.Client()
	t.Cleanup(func() { providerHTTPClient = originalClient })

	registry := NewRegistry(DefaultCatalog()...)
	entry, ok := registry.Get("quark")
	if !ok {
		t.Fatal("expected quark entry")
	}

	result := entry.Adapter.ValidateAuth(AuthProfile{
		ProviderKey: "quark",
		Cookie:      "cookie-live",
		Extra: map[string]string{
			"pwdId":            "pwd-live",
			"apiEndpoint":      server.URL,
			"driveApiEndpoint": server.URL,
		},
	})
	if !result.OK {
		t.Fatalf("expected validation success, got %+v", result)
	}
	if result.Mode != "share_family_real_auth" {
		t.Fatalf("expected share_family_real_auth mode, got %s", result.Mode)
	}
}

func TestShareFamilyAdapterListsReadsAndCreatesQuarkDirectory(t *testing.T) {
	server, _ := newShareFamilyTestServer(t, "quark")
	t.Cleanup(server.Close)

	originalClient := providerHTTPClient
	providerHTTPClient = server.Client()
	t.Cleanup(func() { providerHTTPClient = originalClient })

	registry := NewRegistry(DefaultCatalog()...)
	entry, ok := registry.Get("quark")
	if !ok {
		t.Fatal("expected quark entry")
	}
	profile := shareFamilyTestProfile(server.URL, "quark")

	list := entry.Adapter.List(ListRequest{
		Profile:  profile,
		Path:     "/docs",
		PageSize: 100,
	})
	if !list.OK {
		t.Fatalf("expected list success, got %+v", list)
	}
	if list.Mode != "share_family_real_directory" {
		t.Fatalf("expected share_family_real_directory mode, got %s", list.Mode)
	}
	if len(list.Items) != 2 {
		t.Fatalf("expected 2 docs items, got %d", len(list.Items))
	}
	guide := findProviderItemByPath(list.Items, "/docs/guide.txt")
	if guide == nil || stringMapValue(guide, "fileId") != "file-guide-qk" {
		t.Fatalf("expected guide.txt metadata, got %+v", guide)
	}

	metadata := entry.Adapter.Metadata(MetadataRequest{
		Profile: profile,
		Path:    "/docs/guide.txt",
	})
	if !metadata.OK || metadata.Status != "exists" {
		t.Fatalf("expected metadata exists, got %+v", metadata)
	}
	if stringMapValue(metadata.Entry, "fileId") != "file-guide-qk" {
		t.Fatalf("expected file-guide-qk metadata, got %+v", metadata.Entry)
	}
	if stringMapValue(metadata.Entry, "md5") != "abcdefabcdefabcdefabcdefabcdefab" {
		t.Fatalf("expected guide md5, got %+v", metadata.Entry)
	}

	byID := entry.Adapter.Metadata(MetadataRequest{
		Profile: profile,
		Path:    "/docs/guide.txt",
		FileID:  "file-guide-qk",
	})
	if !byID.OK || stringMapValue(byID.Entry, "fileId") != "file-guide-qk" {
		t.Fatalf("expected metadata by file id, got %+v", byID)
	}

	createDir := entry.Adapter.CreateDir(CreateDirRequest{
		Profile:  profile,
		ParentID: "dir-docs-qk",
		DirName:  "uploads",
	})
	if !createDir.OK {
		t.Fatalf("expected create dir success, got %+v", createDir)
	}
	if stringMapValue(createDir.Payload, "fileId") != "dir-new-qk" {
		t.Fatalf("expected dir-new-qk result, got %+v", createDir.Payload)
	}
}

func TestShareFamilyAdapterQuarkUploadFastHitSucceeds(t *testing.T) {
	server, state := newShareFamilyTestServer(t, "quark")
	t.Cleanup(server.Close)

	originalClient := providerHTTPClient
	providerHTTPClient = server.Client()
	t.Cleanup(func() { providerHTTPClient = originalClient })

	localFile := writeShareFamilyTempFile(t, "upload-hit.bin", []byte("quark-hit-content"))
	localMD5, _, err := computeShareFamilyLocalHashes(localFile)
	if err != nil {
		t.Fatalf("compute hashes: %v", err)
	}

	registry := NewRegistry(DefaultCatalog()...)
	entry, ok := registry.Get("quark")
	if !ok {
		t.Fatal("expected quark entry")
	}

	result := entry.Adapter.Upload(UploadRequest{
		Profile:        shareFamilyTestProfile(server.URL, "quark"),
		Path:           "/docs/upload-hit.bin",
		Name:           "upload-hit.bin",
		ParentID:       "dir-docs-qk",
		Size:           int64(len([]byte("quark-hit-content"))),
		LocalPath:      localFile,
		ConflictPolicy: ConflictPolicyAutoRenameNew,
		Strategy:       "fast_upload",
		MD5:            localMD5,
	})
	if !result.OK {
		t.Fatalf("expected upload success, got %+v", result)
	}
	if result.Mode != "share_family_real_upload" {
		t.Fatalf("expected share_family_real_upload mode, got %+v", result)
	}
	if boolMapValue(result.Payload, "usedBinaryFallback") {
		t.Fatalf("did not expect binary fallback, got %+v", result.Payload)
	}
	if stringMapValue(result.Payload, "fileId") != "file-hit-qk" {
		t.Fatalf("expected file-hit-qk, got %+v", result.Payload)
	}
	if stringMapValue(result.Payload, "verifyMode") != "finish_response" {
		t.Fatalf("expected finish_response verify mode, got %+v", result.Payload)
	}
	if state.uploadPreCount != 1 || state.uploadHashCount != 1 || state.uploadFinishCount != 1 {
		t.Fatalf("expected pre/hash/finish once, got %+v", state)
	}
	if state.uploadAuthCount != 0 || state.ossPutCount != 0 || state.ossCommitCount != 0 {
		t.Fatalf("did not expect binary fallback calls, got %+v", state)
	}
}

func TestShareFamilyAdapterQuarkUploadHashMissFallsBackToBinary(t *testing.T) {
	server, state := newShareFamilyTestServer(t, "quark")
	t.Cleanup(server.Close)

	originalClient := providerHTTPClient
	providerHTTPClient = server.Client()
	t.Cleanup(func() { providerHTTPClient = originalClient })

	localBytes := []byte("quark-hash-miss-binary-data")
	localFile := writeShareFamilyTempFile(t, "upload-miss.bin", localBytes)
	localMD5, _, err := computeShareFamilyLocalHashes(localFile)
	if err != nil {
		t.Fatalf("compute hashes: %v", err)
	}

	registry := NewRegistry(DefaultCatalog()...)
	entry, ok := registry.Get("quark")
	if !ok {
		t.Fatal("expected quark entry")
	}

	result := entry.Adapter.Upload(UploadRequest{
		Profile:        shareFamilyTestProfile(server.URL, "quark"),
		Path:           "/docs/upload-miss.bin",
		Name:           "upload-miss.bin",
		ParentID:       "dir-docs-qk",
		Size:           int64(len(localBytes)),
		LocalPath:      localFile,
		ConflictPolicy: ConflictPolicyAutoRenameNew,
		Strategy:       "download_upload",
		MD5:            localMD5,
	})
	if !result.OK {
		t.Fatalf("expected binary fallback upload success, got %+v", result)
	}
	if !boolMapValue(result.Payload, "usedBinaryFallback") {
		t.Fatalf("expected usedBinaryFallback true, got %+v", result.Payload)
	}
	if stringMapValue(result.Payload, "verifyMode") != "finish_response_after_binary_upload" {
		t.Fatalf("expected binary verify mode, got %+v", result.Payload)
	}
	if stringMapValue(result.Payload, "fileId") != "file-miss-qk" {
		t.Fatalf("expected file-miss-qk, got %+v", result.Payload)
	}
	if state.uploadPreCount != 1 || state.uploadHashCount != 1 || state.uploadFinishCount != 1 {
		t.Fatalf("expected pre/hash/finish once, got %+v", state)
	}
	if state.ossPutCount < 2 {
		t.Fatalf("expected multipart PUTs, got %+v", state)
	}
	if state.uploadAuthCount != state.ossPutCount+1 {
		t.Fatalf("expected auth per part plus commit, got %+v", state)
	}
	if state.ossCommitCount != 1 {
		t.Fatalf("expected one OSS commit, got %+v", state)
	}
}

func TestShareFamilyAdapterQuarkUploadResumeMultipartFallback(t *testing.T) {
	server, state := newShareFamilyTestServer(t, "quark")
	state.failPartNumber = 2
	t.Cleanup(server.Close)

	originalClient := providerHTTPClient
	providerHTTPClient = server.Client()
	t.Cleanup(func() { providerHTTPClient = originalClient })

	localBytes := []byte("quark-resume-multipart-binary-data")
	localFile := writeShareFamilyTempFile(t, "upload-resume.bin", localBytes)
	localMD5, localSHA1, err := computeShareFamilyLocalHashes(localFile)
	if err != nil {
		t.Fatalf("compute hashes: %v", err)
	}

	registry := NewRegistry(DefaultCatalog()...)
	entry, ok := registry.Get("quark")
	if !ok {
		t.Fatal("expected quark entry")
	}

	first := entry.Adapter.Upload(UploadRequest{
		Profile:        shareFamilyTestProfile(server.URL, "quark"),
		Path:           "/docs/upload-resume.bin",
		Name:           "upload-resume.bin",
		ParentID:       "dir-docs-qk",
		Size:           int64(len(localBytes)),
		LocalPath:      localFile,
		ConflictPolicy: ConflictPolicyAutoRenameNew,
		Strategy:       "download_upload",
		MD5:            localMD5,
	})
	if first.OK {
		t.Fatalf("expected first upload to fail for resume scenario, got %+v", first)
	}
	if got := intMapValue(first.Payload, "failedPartNumber"); got != 2 {
		t.Fatalf("expected failedPartNumber 2, got %+v", first.Payload)
	}
	if got := intMapValue(first.Payload, "nextPartNumber"); got != 2 {
		t.Fatalf("expected nextPartNumber 2, got %+v", first.Payload)
	}
	if got := intMapValue(first.Payload, "uploadedPartCount"); got != 1 {
		t.Fatalf("expected uploadedPartCount 1, got %+v", first.Payload)
	}
	providerData, _ := first.Payload["providerData"].(map[string]interface{})
	if providerData == nil {
		t.Fatalf("expected providerData in first failure payload, got %+v", first.Payload)
	}
	partCount := intMapValue(first.Payload, "partCount")
	state.failPartNumber = 0

	second := entry.Adapter.Upload(UploadRequest{
		Profile:        shareFamilyTestProfile(server.URL, "quark"),
		Path:           "/docs/upload-resume.bin",
		Name:           "upload-resume.bin",
		ParentID:       "dir-docs-qk",
		Size:           int64(len(localBytes)),
		LocalPath:      localFile,
		ConflictPolicy: ConflictPolicyAutoRenameNew,
		Strategy:       "download_upload",
		MD5:            localMD5,
		ResumeUpload: &ResumeUpload{
			FileID:            stringMapValue(first.Payload, "fileId"),
			UploadID:          stringMapValue(first.Payload, "uploadId"),
			PartCount:         intMapValue(first.Payload, "partCount"),
			UploadedPartCount: intMapValue(first.Payload, "uploadedPartCount"),
			FailedPartNumber:  intMapValue(first.Payload, "failedPartNumber"),
			NextPartNumber:    intMapValue(first.Payload, "nextPartNumber"),
			UploadedParts:     mapSliceValue(first.Payload, "uploadedParts"),
			ProviderData:      providerData,
		},
	})
	if !second.OK {
		t.Fatalf("expected resumed upload success, got %+v", second)
	}
	if !boolMapValue(second.Payload, "resumedUpload") {
		t.Fatalf("expected resumedUpload true, got %+v", second.Payload)
	}
	if got := intMapValue(second.Payload, "uploadedPartCount"); got != partCount {
		t.Fatalf("expected uploadedPartCount %d, got %+v", partCount, second.Payload)
	}
	fallbackPayload, _ := second.Payload["uploadFallback"].(map[string]interface{})
	if got := intMapValue(fallbackPayload, "uploadedPartCount"); got != partCount {
		t.Fatalf("expected uploadFallback uploadedPartCount %d, got %+v", partCount, fallbackPayload)
	}
	if state.uploadPreCount != 1 || state.uploadHashCount != 1 || state.uploadFinishCount != 1 {
		t.Fatalf("expected resume flow to reuse original pre/hash and only finish once, got %+v", state)
	}
	if state.ossPutCount < partCount {
		t.Fatalf("expected at least %d OSS puts across fail+resume, got %+v", partCount, state)
	}
	if state.uploadAuthCount != state.ossPutCount+1 {
		t.Fatalf("expected one upload auth per OSS put plus final commit auth, got %+v", state)
	}
	if state.ossCommitCount != 1 {
		t.Fatalf("expected one final commit after resume, got %+v", state)
	}
	if !strings.Contains(strings.ToLower(stringMapValue(second.Payload, "resolvedTargetName")), "upload-resume.bin") {
		t.Fatalf("expected resolved target name preserved, got %+v", second.Payload)
	}
	if got := stringMapValue(providerData, "md5"); got != localMD5 {
		t.Fatalf("expected providerData md5 %s, got %+v", localMD5, providerData)
	}
	if got := stringMapValue(providerData, "sha1"); got != localSHA1 {
		t.Fatalf("expected providerData sha1 %s, got %+v", localSHA1, providerData)
	}
}

func TestShareFamilyAdapterValidatesAndUploadsUCAgainstLiveEndpoint(t *testing.T) {
	server, state := newShareFamilyTestServer(t, "uc")
	t.Cleanup(server.Close)

	originalClient := providerHTTPClient
	providerHTTPClient = server.Client()
	t.Cleanup(func() { providerHTTPClient = originalClient })

	localFile := writeShareFamilyTempFile(t, "uc-upload.bin", []byte("uc-hit-content"))
	localMD5, _, err := computeShareFamilyLocalHashes(localFile)
	if err != nil {
		t.Fatalf("compute hashes: %v", err)
	}

	registry := NewRegistry(DefaultCatalog()...)
	entry, ok := registry.Get("uc")
	if !ok {
		t.Fatal("expected uc entry")
	}
	profile := shareFamilyTestProfile(server.URL, "uc")

	validate := entry.Adapter.ValidateAuth(profile)
	if !validate.OK {
		t.Fatalf("expected uc validation success, got %+v", validate)
	}

	list := entry.Adapter.List(ListRequest{
		Profile: profile,
		Path:    "/docs",
	})
	if !list.OK || len(list.Items) != 2 {
		t.Fatalf("expected uc list success, got %+v", list)
	}

	metadata := entry.Adapter.Metadata(MetadataRequest{
		Profile: profile,
		Path:    "/docs/guide.txt",
	})
	if !metadata.OK || stringMapValue(metadata.Entry, "md5") != "abcdefabcdefabcdefabcdefabcdefab" {
		t.Fatalf("expected uc metadata md5, got %+v", metadata)
	}

	createDir := entry.Adapter.CreateDir(CreateDirRequest{
		Profile:  profile,
		ParentID: "dir-docs-qk",
		DirName:  "uploads",
	})
	if !createDir.OK || stringMapValue(createDir.Payload, "fileId") != "dir-new-qk" {
		t.Fatalf("expected uc create dir success, got %+v", createDir)
	}

	result := entry.Adapter.Upload(UploadRequest{
		Profile:        profile,
		Path:           "/docs/uc-upload.bin",
		Name:           "uc-upload.bin",
		ParentID:       "dir-docs-qk",
		Size:           int64(len([]byte("uc-hit-content"))),
		LocalPath:      localFile,
		ConflictPolicy: ConflictPolicyAutoRenameNew,
		Strategy:       "fast_upload",
		MD5:            localMD5,
	})
	if !result.OK {
		t.Fatalf("expected uc upload success, got %+v", result)
	}
	if stringMapValue(result.Payload, "fileId") != "file-hit-uc" {
		t.Fatalf("expected file-hit-uc, got %+v", result.Payload)
	}
	if state.uploadPreCount != 1 || state.uploadHashCount != 1 || state.uploadFinishCount != 1 {
		t.Fatalf("expected uc pre/hash/finish once, got %+v", state)
	}
}

type shareFamilyTestState struct {
	providerKey       string
	uploadPreCount    int
	uploadHashCount   int
	uploadAuthCount   int
	uploadFinishCount int
	ossPutCount       int
	ossCommitCount    int
	failPartNumber    int
	taskNames         map[string]string
	taskObjKeys       map[string]string
	taskFileIDs       map[string]string
}

func shareFamilyTestProfile(serverURL string, providerKey string) AuthProfile {
	return AuthProfile{
		ProviderKey: providerKey,
		Cookie:      "cookie-live",
		Extra: map[string]string{
			"pwdId":            "pwd-live",
			"apiEndpoint":      serverURL,
			"driveApiEndpoint": serverURL,
		},
	}
}

func writeShareFamilyTempFile(t *testing.T, name string, data []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	return path
}

func newQuarkShareFamilyTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	server, _ := newShareFamilyTestServer(t, "quark")
	return server
}

func newUCShareFamilyTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	server, _ := newShareFamilyTestServer(t, "uc")
	return server
}

func newShareFamilyTestServer(t *testing.T, providerKey string) (*httptest.Server, *shareFamilyTestState) {
	t.Helper()

	state := &shareFamilyTestState{
		providerKey: providerKey,
		taskNames:   map[string]string{},
		taskObjKeys: map[string]string{},
		taskFileIDs: map[string]string{},
	}

	rootItems := []map[string]interface{}{
		{
			"fid":       "dir-docs-qk",
			"file_name": "docs",
			"dir":       true,
			"size":      0,
		},
	}
	docsItems := []map[string]interface{}{
		{
			"fid":             "file-guide-qk",
			"share_fid_token": "token-guide-qk",
			"file_name":       "guide.txt",
			"dir":             false,
			"size":            12,
		},
		{
			"fid":       "dir-sub-qk",
			"file_name": "sub",
			"dir":       true,
			"size":      0,
		},
	}

	mustCookie := func(w http.ResponseWriter, r *http.Request) bool {
		t.Helper()
		if got := r.Header.Get("Cookie"); got != "cookie-live" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return false
		}
		if providerKey == "uc" {
			if got := r.Header.Get("User-Agent"); !strings.Contains(got, "UcCloudDrivePC") {
				t.Fatalf("expected uc user-agent, got %q", got)
			}
		} else {
			if got := r.Header.Get("User-Agent"); !strings.Contains(got, "QuarkCloudDrivePC") {
				t.Fatalf("expected quark user-agent, got %q", got)
			}
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

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/objects/") {
			switch r.Method {
			case http.MethodPut:
				state.ossPutCount++
				if state.failPartNumber > 0 && strings.Contains(r.URL.RawQuery, "partNumber="+strconv.Itoa(state.failPartNumber)) {
					http.Error(w, "mock part failure", http.StatusInternalServerError)
					return
				}
				w.Header().Set("ETag", "\"etag-part\"")
				w.WriteHeader(http.StatusOK)
			case http.MethodPost:
				state.ossCommitCount++
				w.WriteHeader(http.StatusOK)
			default:
				http.NotFound(w, r)
			}
			return
		}

		if !mustCookie(w, r) {
			return
		}
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/1/clouddrive/share/sharepage/token":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"code": 0,
				"data": map[string]interface{}{
					"stoken": "stoken-live",
				},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/1/clouddrive/share/sharepage/detail":
			parentID := r.URL.Query().Get("pdir_fid")
			switch parentID {
			case "0", "":
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"code": 0,
					"data": map[string]interface{}{
						"list":  rootItems,
						"total": len(rootItems),
					},
				})
			case "dir-docs-qk":
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"code": 0,
					"data": map[string]interface{}{
						"list":  docsItems,
						"total": len(docsItems),
					},
				})
			default:
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"code": 0,
					"data": map[string]interface{}{
						"list":  []map[string]interface{}{},
						"total": 0,
					},
				})
			}
		case r.Method == http.MethodPost && r.URL.Path == "/1/clouddrive/file/download":
			payload := mustDecode(r)
			fids, _ := payload["fids"].([]interface{})
			if len(fids) != 1 || strings.TrimSpace(fids[0].(string)) != "file-guide-qk" {
				t.Fatalf("unexpected download payload %+v", payload)
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"code": 0,
				"data": map[string]interface{}{
					"items": []map[string]interface{}{
						{
							"fid":  "file-guide-qk",
							"md5":  "abcdefabcdefabcdefabcdefabcdefab",
							"etag": "abcdefabcdefabcdefabcdefabcdefab",
						},
					},
				},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/1/clouddrive/file":
			payload := mustDecode(r)
			if stringMapValue(payload, "file_name") != "uploads" {
				t.Fatalf("unexpected create dir payload %+v", payload)
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"code": 0,
				"data": map[string]interface{}{
					"fid":       "dir-new-qk",
					"file_name": "uploads",
					"dir":       true,
					"size":      0,
				},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/1/clouddrive/file/upload/pre":
			state.uploadPreCount++
			payload := mustDecode(r)
			fileName := stringMapValue(payload, "file_name")
			taskID := "task-" + providerKey + "-" + fileName
			state.taskNames[taskID] = fileName
			state.taskObjKeys[taskID] = "objects/" + fileName
			switch fileName {
			case "upload-hit.bin":
				state.taskFileIDs[taskID] = "file-hit-qk"
			case "upload-miss.bin":
				state.taskFileIDs[taskID] = "file-miss-qk"
			case "uc-upload.bin":
				state.taskFileIDs[taskID] = "file-hit-uc"
			default:
				state.taskFileIDs[taskID] = "file-generic-" + providerKey
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"code": 0,
				"data": map[string]interface{}{
					"task_id":    taskID,
					"obj_key":    state.taskObjKeys[taskID],
					"bucket":     "test-bucket",
					"upload_id":  "upload-" + providerKey,
					"upload_url": serverURLForProviderRequest(r),
					"part_size":  8,
					"fid":        state.taskFileIDs[taskID],
					"auth_info": map[string]interface{}{
						"token": "auth-info-" + providerKey,
					},
					"callback": map[string]interface{}{
						"callbackUrl": "https://callback.invalid/" + providerKey,
					},
				},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/1/clouddrive/file/update/hash":
			state.uploadHashCount++
			payload := mustDecode(r)
			taskID := stringMapValue(payload, "task_id")
			fileName := state.taskNames[taskID]
			finish := fileName == "upload-hit.bin" || fileName == "uc-upload.bin"
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"code": 0,
				"data": map[string]interface{}{
					"finish": finish,
				},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/1/clouddrive/file/upload/auth":
			state.uploadAuthCount++
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"code": 0,
				"data": map[string]interface{}{
					"auth_key": "OSS mock-auth-key",
				},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/1/clouddrive/file/upload/finish":
			state.uploadFinishCount++
			payload := mustDecode(r)
			taskID := stringMapValue(payload, "task_id")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"code": 0,
				"data": map[string]interface{}{
					"fid":       state.taskFileIDs[taskID],
					"file_name": state.taskNames[taskID],
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))

	return server, state
}

func serverURLForProviderRequest(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}
