package provider

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCloud189FamilyAdapterRequiresCookie(t *testing.T) {
	registry := NewRegistry(DefaultCatalog()...)
	entry, ok := registry.Get("189cloud")
	if !ok {
		t.Fatal("expected 189cloud entry")
	}

	result := entry.Adapter.ValidateAuth(AuthProfile{
		ProviderKey: "189cloud",
	})
	if result.OK {
		t.Fatal("expected validation to fail without cookie")
	}
	if result.Status != "missing_cookie" {
		t.Fatalf("expected missing_cookie, got %s", result.Status)
	}
}

func TestCloud189FamilyAdapterSupportsFastUploadCandidate(t *testing.T) {
	server, _ := newCloud189TestServer(t)
	t.Cleanup(server.Close)

	originalClient := providerHTTPClient
	providerHTTPClient = server.Client()
	t.Cleanup(func() { providerHTTPClient = originalClient })

	registry := NewRegistry(DefaultCatalog()...)
	entry, ok := registry.Get("189cloud")
	if !ok {
		t.Fatal("expected 189cloud entry")
	}
	check := entry.Adapter.FastUploadCheck(FastUploadCheckRequest{
		Profile: cloud189TestProfile(server.URL),
		Path:    "/a.bin",
		Name:    "a.bin",
		Size:    1024,
		MD5:     "0123456789abcdef0123456789abcdef",
	})
	if !check.OK || !check.Candidate {
		t.Fatalf("expected fast upload candidate, got %+v", check)
	}
}

func TestCloud189FamilyAdapterLiveShareReadAndCreateDir(t *testing.T) {
	server, _ := newCloud189TestServer(t)
	t.Cleanup(server.Close)

	originalClient := providerHTTPClient
	providerHTTPClient = server.Client()
	t.Cleanup(func() { providerHTTPClient = originalClient })

	registry := NewRegistry(DefaultCatalog()...)
	entry, ok := registry.Get("189cloud")
	if !ok {
		t.Fatal("expected 189cloud entry")
	}
	profile := cloud189TestProfile(server.URL)

	validation := entry.Adapter.ValidateAuth(profile)
	if !validation.OK {
		t.Fatalf("expected validation success, got %+v", validation)
	}

	list := entry.Adapter.List(ListRequest{
		Profile:  profile,
		ParentID: "root-file-189",
		PageSize: 100,
	})
	if !list.OK {
		t.Fatalf("expected list success, got %+v", list)
	}
	if len(list.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(list.Items))
	}
	file := findProviderItemByPath(list.Items, "/189cloud-share/guide.txt")
	if file == nil || stringMapValue(file, "md5") != "0123456789abcdef0123456789abcdef" {
		t.Fatalf("expected guide file with md5, got %+v", file)
	}

	metadata := entry.Adapter.Metadata(MetadataRequest{
		Profile:  profile,
		ParentID: "root-file-189",
		FileID:   "file-guide-189",
	})
	if !metadata.OK || metadata.Status != "exists" {
		t.Fatalf("expected metadata exists, got %+v", metadata)
	}
	if stringMapValue(metadata.Entry, "fileId") != "file-guide-189" {
		t.Fatalf("expected file-guide-189 metadata, got %+v", metadata.Entry)
	}

	readonlyCreate := entry.Adapter.CreateDir(CreateDirRequest{
		Profile:  profile,
		ParentID: "root-file-189",
		DirName:  "uploads",
	})
	if readonlyCreate.OK || readonlyCreate.Status != "share_auth_readonly" {
		t.Fatalf("expected share_auth_readonly, got %+v", readonlyCreate)
	}

	writable := cloud189WritableProfile(server.URL)
	createDir := entry.Adapter.CreateDir(CreateDirRequest{
		Profile: writable,
		DirName: "uploads",
	})
	if !createDir.OK {
		t.Fatalf("expected create dir success, got %+v", createDir)
	}
	item, _ := createDir.Payload["item"].(map[string]interface{})
	if stringMapValue(item, "fileId") != "dir-new-189" {
		t.Fatalf("expected dir-new-189, got %+v", createDir.Payload)
	}
}

func TestCloud189FamilyAdapterUploadRapidHitSucceeds(t *testing.T) {
	server, state := newCloud189TestServer(t)
	t.Cleanup(server.Close)

	originalClient := providerHTTPClient
	providerHTTPClient = server.Client()
	t.Cleanup(func() { providerHTTPClient = originalClient })

	localBytes := []byte("cloud189-rapid-hit")
	localFile := writeCloud189TempFile(t, "rapid-hit.bin", localBytes)
	localMD5, err := computeCloud189LocalMD5(localFile)
	if err != nil {
		t.Fatalf("compute md5: %v", err)
	}

	registry := NewRegistry(DefaultCatalog()...)
	entry, ok := registry.Get("189cloud")
	if !ok {
		t.Fatal("expected 189cloud entry")
	}

	result := entry.Adapter.Upload(UploadRequest{
		Profile:        cloud189WritableProfile(server.URL),
		ParentID:       "parent-write-189",
		Path:           "/rapid-hit.bin",
		Name:           "rapid-hit.bin",
		LocalPath:      localFile,
		Size:           int64(len(localBytes)),
		ConflictPolicy: ConflictPolicyAutoRenameNew,
		Strategy:       "fast_upload",
		MD5:            localMD5,
	})
	if !result.OK {
		t.Fatalf("expected rapid hit upload success, got %+v", result)
	}
	if stringMapValue(result.Payload, "fileId") != "file-rapid-189" {
		t.Fatalf("expected file-rapid-189, got %+v", result.Payload)
	}
	if stringMapValue(result.Payload, "verifyMode") != "commit_response_xml" {
		t.Fatalf("expected commit_response_xml, got %+v", result.Payload)
	}
	if boolMapValue(result.Payload, "usedBinaryFallback") {
		t.Fatalf("did not expect binary fallback, got %+v", result.Payload)
	}
	if state.sessionCount != 1 || state.createUploadCount != 1 || state.commitCount != 1 {
		t.Fatalf("expected session/create/commit once, got %+v", state)
	}
	if state.putCount != 0 || state.statusCount != 0 {
		t.Fatalf("did not expect PUT/status on rapid hit, got %+v", state)
	}
}

func TestCloud189FamilyAdapterUploadHashMissFallsBackToBinary(t *testing.T) {
	server, state := newCloud189TestServer(t)
	t.Cleanup(server.Close)

	originalClient := providerHTTPClient
	providerHTTPClient = server.Client()
	t.Cleanup(func() { providerHTTPClient = originalClient })

	localBytes := []byte("cloud189-hash-miss-binary")
	localFile := writeCloud189TempFile(t, "binary-fallback.bin", localBytes)
	localMD5, err := computeCloud189LocalMD5(localFile)
	if err != nil {
		t.Fatalf("compute md5: %v", err)
	}

	registry := NewRegistry(DefaultCatalog()...)
	entry, ok := registry.Get("189cloud")
	if !ok {
		t.Fatal("expected 189cloud entry")
	}

	result := entry.Adapter.Upload(UploadRequest{
		Profile:        cloud189WritableProfile(server.URL),
		ParentID:       "parent-write-189",
		Path:           "/binary-fallback.bin",
		Name:           "binary-fallback.bin",
		LocalPath:      localFile,
		Size:           int64(len(localBytes)),
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
	if stringMapValue(result.Payload, "verifyMode") != "commit_response_xml_after_binary_put" {
		t.Fatalf("expected commit_response_xml_after_binary_put, got %+v", result.Payload)
	}
	verifyPayload, _ := result.Payload["verifyPayload"].(map[string]interface{})
	if stringMapValue(verifyPayload, "fileId") != "file-fallback-189" {
		t.Fatalf("expected file-fallback-189 verify payload, got %+v", verifyPayload)
	}
	if state.sessionCount != 1 || state.createUploadCount != 1 || state.statusCount != 1 || state.putCount != 1 || state.commitCount != 1 {
		t.Fatalf("expected full fallback chain once, got %+v", state)
	}
}

func TestCloud189FamilyAdapterResumesCachedBinaryUploadSession(t *testing.T) {
	server, state := newCloud189TestServer(t)
	state.failPutOnce = true
	t.Cleanup(server.Close)

	originalClient := providerHTTPClient
	providerHTTPClient = server.Client()
	t.Cleanup(func() { providerHTTPClient = originalClient })

	localBytes := []byte("cloud189-resume-binary")
	localFile := writeCloud189TempFile(t, "resume-binary.bin", localBytes)
	localMD5, err := computeCloud189LocalMD5(localFile)
	if err != nil {
		t.Fatalf("compute md5: %v", err)
	}

	registry := NewRegistry(DefaultCatalog()...)
	entry, ok := registry.Get("189cloud")
	if !ok {
		t.Fatal("expected 189cloud entry")
	}

	first := entry.Adapter.Upload(UploadRequest{
		Profile:        cloud189WritableProfile(server.URL),
		ParentID:       "parent-write-189",
		Path:           "/binary-fallback.bin",
		Name:           "binary-fallback.bin",
		LocalPath:      localFile,
		Size:           int64(len(localBytes)),
		ConflictPolicy: ConflictPolicyAutoRenameNew,
		Strategy:       "download_upload",
		MD5:            localMD5,
	})
	if first.OK {
		t.Fatalf("expected first upload to fail, got %+v", first)
	}
	providerData, _ := first.Payload["providerData"].(map[string]interface{})
	if providerData == nil {
		t.Fatalf("expected providerData in failed payload, got %+v", first.Payload)
	}
	if int64MapValue(providerData, "uploadFileId") != 18902 {
		t.Fatalf("expected cached uploadFileId 18902, got %+v", providerData)
	}

	second := entry.Adapter.Upload(UploadRequest{
		Profile:        cloud189WritableProfile(server.URL),
		ParentID:       "parent-write-189",
		Path:           "/binary-fallback.bin",
		Name:           "binary-fallback.bin",
		LocalPath:      localFile,
		Size:           int64(len(localBytes)),
		ConflictPolicy: ConflictPolicyAutoRenameNew,
		Strategy:       "download_upload",
		MD5:            localMD5,
		ResumeUpload: &ResumeUpload{
			ProviderData: providerData,
		},
	})
	if !second.OK {
		t.Fatalf("expected resumed upload success, got %+v", second)
	}
	if !boolMapValue(second.Payload, "resumedUpload") {
		t.Fatalf("expected resumedUpload true, got %+v", second.Payload)
	}
	if state.createUploadCount != 1 {
		t.Fatalf("expected create upload only once, got %+v", state)
	}
	if state.putCount != 2 || state.statusCount != 1 || state.commitCount != 1 {
		t.Fatalf("expected resumed flow to reuse cached session, got %+v", state)
	}
	if state.sessionCount != 2 {
		t.Fatalf("expected auth session refresh twice, got %+v", state)
	}
}

type cloud189TestState struct {
	sessionCount      int
	createUploadCount int
	statusCount       int
	putCount          int
	commitCount       int
	lastUploadFileID  string
	lastTargetName    string
	failPutOnce       bool
}

func cloud189TestProfile(baseURL string) AuthProfile {
	return AuthProfile{
		ProviderKey: "189cloud",
		Cookie:      "SESSION=cloud189",
		Extra: map[string]string{
			"shareCode":            "share-189",
			"shareInfoEndpoint":    baseURL + "/share/info",
			"checkAccessEndpoint":  baseURL + "/share/check_access",
			"listEndpoint":         baseURL + "/share/list",
			"createDirEndpoint":    baseURL + "/create/folder",
			"authSessionEndpoint":  baseURL + "/upload/session",
			"createUploadEndpoint": baseURL + "/upload/create",
			"uploadStatusEndpoint": baseURL + "/upload/status",
		},
	}
}

func cloud189WritableProfile(baseURL string) AuthProfile {
	profile := cloud189TestProfile(baseURL)
	profile.Token = "token-189"
	profile.Extra["signature"] = "sig-189"
	profile.Extra["date"] = "2026-05-28T10:11:12Z"
	profile.Extra["parentId"] = "parent-write-189"
	return profile
}

func writeCloud189TempFile(t *testing.T, name string, data []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	return path
}

func newCloud189TestServer(t *testing.T) (*httptest.Server, *cloud189TestState) {
	t.Helper()

	state := &cloud189TestState{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/upload/put/") {
			state.putCount++
			if got := r.Header.Get("Edrive-UploadFileId"); got == "" {
				t.Fatalf("expected Edrive-UploadFileId header")
			}
			if state.failPutOnce && state.putCount == 1 {
				http.Error(w, "temporary put failure", http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/upload/commit/") {
			state.commitCount++
			w.Header().Set("Content-Type", "application/xml")
			fileID := "file-rapid-189"
			if strings.Contains(r.URL.Path, "fallback") {
				fileID = "file-fallback-189"
			}
			_, _ = w.Write([]byte(`<root><id>` + fileID + `</id><name>` + state.lastTargetName + `</name><size>24</size><md5>0123456789abcdef0123456789abcdef</md5><createDate>2026-05-28 10:11:12</createDate></root>`))
			return
		}

		if got := r.Header.Get("Cookie"); !strings.Contains(got, "SESSION=cloud189") {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/share/info":
			_ = r.ParseForm()
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"res_code": "0",
				"data": map[string]interface{}{
					"shareId":   "share-id-189",
					"shareMode": 1,
					"fileId":    "root-file-189",
				},
			})
		case "/share/check_access":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"res_code": "0",
				"data": map[string]interface{}{
					"shareId": "share-id-189",
				},
			})
		case "/share/list":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"res_code": "0",
				"data": map[string]interface{}{
					"fileListAO": map[string]interface{}{
						"folderList": []map[string]interface{}{
							{"id": "dir-docs-189", "name": "docs"},
						},
						"fileList": []map[string]interface{}{
							{"id": "file-guide-189", "name": "guide.txt", "size": 12, "md5": "0123456789abcdef0123456789abcdef"},
						},
					},
				},
			})
		case "/create/folder":
			if got := r.Header.Get("AccessToken"); got != "token-189" {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			if got := r.Header.Get("Signature"); got != "sig-189" {
				http.Error(w, "bad signature", http.StatusForbidden)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"res_code": "0",
				"data": map[string]interface{}{
					"createFolderVO": map[string]interface{}{
						"id": "dir-new-189",
					},
				},
			})
		case "/upload/session":
			state.sessionCount++
			if got := r.URL.Query().Get("accessToken"); got != "token-189" {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"sessionKey":    "session-key-189",
				"sessionSecret": "session-secret-189",
			})
		case "/upload/create":
			state.createUploadCount++
			_ = r.ParseForm()
			targetName := r.Form.Get("fileName")
			state.lastTargetName = targetName
			fileDataExists := 1
			uploadFileID := "18901"
			baseURL := "http://" + r.Host
			uploadPath := baseURL + "/upload/put/rapid"
			commitPath := baseURL + "/upload/commit/rapid"
			if targetName == "binary-fallback.bin" {
				fileDataExists = 0
				uploadFileID = "18902"
				uploadPath = baseURL + "/upload/put/fallback"
				commitPath = baseURL + "/upload/commit/fallback"
			}
			state.lastUploadFileID = uploadFileID
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"uploadFileId":   uploadFileID,
				"fileUploadUrl":  uploadPath,
				"fileCommitUrl":  commitPath,
				"fileDataExists": fileDataExists,
			})
		case "/upload/status":
			state.statusCount++
			baseURL := "http://" + r.Host
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"data": map[string]interface{}{
					"uploadFileId":   state.lastUploadFileID,
					"fileUploadUrl":  baseURL + "/upload/put/fallback",
					"fileCommitUrl":  baseURL + "/upload/commit/fallback",
					"fileDataExists": 1,
					"dataSize":       24,
					"size":           0,
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))

	return server, state
}
