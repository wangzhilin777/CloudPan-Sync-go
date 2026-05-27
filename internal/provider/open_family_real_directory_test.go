package provider

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOpenFamilyAdapterListsAliyunLiveDirectories(t *testing.T) {
	server := newAliyunOpenDirectoryTestServer(t)
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

	rootList := entry.Adapter.List(ListRequest{
		Profile:  profile,
		Path:     "/",
		PageSize: 200,
	})
	if !rootList.OK {
		t.Fatalf("expected root list success, got %+v", rootList)
	}
	if rootList.Mode != "open_family_real_directory" {
		t.Fatalf("expected open_family_real_directory mode, got %s", rootList.Mode)
	}
	if len(rootList.Items) != 2 {
		t.Fatalf("expected 2 root items, got %d", len(rootList.Items))
	}
	docsItem := findProviderItemByPath(rootList.Items, "/docs")
	if docsItem == nil || !boolMapValue(docsItem, "isDir") {
		t.Fatalf("expected /docs directory entry, got %+v", docsItem)
	}
	rootFile := findProviderItemByPath(rootList.Items, "/root.txt")
	if rootFile == nil || stringMapValue(rootFile, "md5") != "md5-root" {
		t.Fatalf("expected /root.txt md5, got %+v", rootFile)
	}

	docsList := entry.Adapter.List(ListRequest{
		Profile:  profile,
		Path:     "/docs",
		PageSize: 200,
	})
	if !docsList.OK {
		t.Fatalf("expected /docs list success, got %+v", docsList)
	}
	if len(docsList.Items) != 2 {
		t.Fatalf("expected 2 /docs items, got %d", len(docsList.Items))
	}
	guideItem := findProviderItemByPath(docsList.Items, "/docs/guide.md")
	if guideItem == nil || stringMapValue(guideItem, "sha1") != "sha1-guide" {
		t.Fatalf("expected /docs/guide.md sha1, got %+v", guideItem)
	}
}

func TestOpenFamilyAdapterReadsAliyunLiveMetadata(t *testing.T) {
	server := newAliyunOpenDirectoryTestServer(t)
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

	metadata := entry.Adapter.Metadata(MetadataRequest{
		Profile: profile,
		Path:    "/docs/guide.md",
	})
	if !metadata.OK {
		t.Fatalf("expected metadata success, got %+v", metadata)
	}
	if metadata.Status != "exists" {
		t.Fatalf("expected exists status, got %s", metadata.Status)
	}
	if stringMapValue(metadata.Entry, "sha1") != "sha1-guide" {
		t.Fatalf("expected sha1-guide, got %+v", metadata.Entry)
	}

	byID := entry.Adapter.Metadata(MetadataRequest{
		Profile: profile,
		Path:    "/docs/guide.md",
		FileID:  "file-guide",
	})
	if !byID.OK || stringMapValue(byID.Entry, "fileId") != "file-guide" {
		t.Fatalf("expected file-guide metadata by file id, got %+v", byID)
	}

	missing := entry.Adapter.Metadata(MetadataRequest{
		Profile: profile,
		Path:    "/docs/missing.md",
	})
	if !missing.OK {
		t.Fatalf("expected missing metadata request to stay successful, got %+v", missing)
	}
	if missing.Status != "missing" {
		t.Fatalf("expected missing status, got %s", missing.Status)
	}
	if boolMapValue(missing.Entry, "exists") {
		t.Fatalf("expected missing entry exists=false, got %+v", missing.Entry)
	}
}

func TestOpenFamilyAdapterCreatesAliyunLiveDirectory(t *testing.T) {
	server := newAliyunOpenDirectoryTestServer(t)
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

	result := entry.Adapter.CreateDir(CreateDirRequest{
		Profile:  profile,
		ParentID: "dir-docs",
		DirName:  "uploads",
	})
	if !result.OK {
		t.Fatalf("expected create dir success, got %+v", result)
	}
	if stringMapValue(result.Payload, "name") != "uploads" {
		t.Fatalf("expected uploads name, got %+v", result.Payload)
	}
	if stringMapValue(result.Payload, "parentId") != "dir-docs" {
		t.Fatalf("expected parent dir-docs, got %+v", result.Payload)
	}
}

func newAliyunOpenDirectoryTestServer(t *testing.T) *httptest.Server {
	t.Helper()

	mustDecode := func(r *http.Request) map[string]interface{} {
		t.Helper()
		var payload map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		return payload
	}

	return httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if auth := r.Header.Get("Authorization"); auth != "Bearer token-live" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/v2/user/get":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"domain_id":        "bj1",
				"user_id":          "user-1",
				"default_drive_id": "drive-1",
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
						{
							"name":              "root.txt",
							"type":              "file",
							"file_id":           "file-root",
							"parent_file_id":    "root",
							"size":              12,
							"content_hash_name": "md5",
							"content_hash":      "md5-root",
						},
					},
				})
			case "dir-docs":
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"items": []map[string]interface{}{
						{
							"name":           "2026",
							"type":           "folder",
							"file_id":        "dir-2026",
							"parent_file_id": "dir-docs",
						},
						{
							"name":              "guide.md",
							"type":              "file",
							"file_id":           "file-guide",
							"parent_file_id":    "dir-docs",
							"size":              34,
							"content_hash_name": "sha1",
							"content_hash":      "sha1-guide",
						},
					},
				})
			case "dir-2026":
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"items": []map[string]interface{}{
						{
							"name":           "deep.txt",
							"type":           "file",
							"file_id":        "file-deep",
							"parent_file_id": "dir-2026",
							"size":           56,
							"md5":            "md5-deep",
						},
					},
				})
			default:
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"items": []map[string]interface{}{},
				})
			}
		case "/adrive/v1.0/openFile/get":
			payload := mustDecode(r)
			switch stringMapValue(payload, "file_id") {
			case "file-guide":
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"name":              "guide.md",
					"type":              "file",
					"file_id":           "file-guide",
					"parent_file_id":    "dir-docs",
					"size":              34,
					"content_hash_name": "sha1",
					"content_hash":      "sha1-guide",
				})
			default:
				http.NotFound(w, r)
			}
		case "/adrive/v1.0/openFile/create":
			payload := mustDecode(r)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"name":            stringMapValue(payload, "name"),
				"type":            "folder",
				"file_id":         "dir-created",
				"parent_file_id":  stringMapValue(payload, "parent_file_id"),
				"check_name_mode": stringMapValue(payload, "check_name_mode"),
			})
		default:
			http.NotFound(w, r)
		}
	}))
}

func findProviderItemByPath(items []map[string]interface{}, path string) map[string]interface{} {
	for _, item := range items {
		if stringMapValue(item, "path") == path {
			return item
		}
	}
	return nil
}
