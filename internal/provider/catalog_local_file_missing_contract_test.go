package provider

import "testing"

func TestDefaultCatalogLocalFileMissingContracts(t *testing.T) {
	t.Run("guangya live upload requires local file", func(t *testing.T) {
		server, _ := newGuangyaTestServer(t)
		defer server.Close()

		originalClient := providerHTTPClient
		providerHTTPClient = server.Client()
		defer func() { providerHTTPClient = originalClient }()

		registry := NewRegistry(DefaultCatalog()...)
		entry, ok := registry.Get("guangya")
		if !ok {
			t.Fatal("expected guangya")
		}
		assertLocalFileMissingUploadBlocked(t, entry.Adapter.Upload(UploadRequest{
			Profile:        guangyaTestProfile(server.URL),
			Path:           "/upload.bin",
			ParentID:       "parent-gy",
			Name:           "upload.bin",
			Size:           1024,
			ConflictPolicy: ConflictPolicyAutoRenameNew,
			Strategy:       "download_upload",
		}), "guangya_family_real_upload")
	})

	t.Run("aliyun open live upload requires local file", func(t *testing.T) {
		server, _ := newAliyunOpenUploadTestServer(t)
		defer server.Close()

		originalClient := providerHTTPClient
		providerHTTPClient = server.Client()
		defer func() { providerHTTPClient = originalClient }()

		registry := NewRegistry(DefaultCatalog()...)
		entry, ok := registry.Get("aliyundrive_open")
		if !ok {
			t.Fatal("expected aliyundrive_open")
		}
		assertLocalFileMissingUploadBlocked(t, entry.Adapter.Upload(UploadRequest{
			Profile:        AuthProfile{ProviderKey: "aliyundrive_open", Token: "token-live", Extra: map[string]string{"domainId": "bj1", "driveId": "drive-1", "apiEndpoint": server.URL}},
			Path:           "/upload.bin",
			Name:           "upload.bin",
			Size:           1024,
			ConflictPolicy: ConflictPolicyAutoRenameNew,
			Strategy:       "download_upload",
		}), "open_family_real_upload")
	})

	t.Run("123 open live upload requires local file", func(t *testing.T) {
		server, _ := newPan123OpenTestServer(t)
		defer server.Close()

		originalClient := providerHTTPClient
		providerHTTPClient = server.Client()
		defer func() { providerHTTPClient = originalClient }()

		registry := NewRegistry(DefaultCatalog()...)
		entry, ok := registry.Get("123_open")
		if !ok {
			t.Fatal("expected 123_open")
		}
		assertLocalFileMissingUploadBlocked(t, entry.Adapter.Upload(UploadRequest{
			Profile:        AuthProfile{ProviderKey: "123_open", Token: "token-live", Extra: map[string]string{"apiEndpoint": server.URL}},
			Path:           "/upload.bin",
			Name:           "upload.bin",
			Size:           1024,
			ConflictPolicy: ConflictPolicyAutoRenameNew,
			Strategy:       "download_upload",
		}), "open_family_real_upload")
	})

	t.Run("115 live upload requires local file", func(t *testing.T) {
		server, _ := newPan115FamilyTestServer(t)
		defer server.Close()

		originalClient := providerHTTPClient
		providerHTTPClient = server.Client()
		defer func() { providerHTTPClient = originalClient }()

		registry := NewRegistry(DefaultCatalog()...)
		entry, ok := registry.Get("115_open")
		if !ok {
			t.Fatal("expected 115_open")
		}
		assertLocalFileMissingUploadBlocked(t, entry.Adapter.Upload(UploadRequest{
			Profile:        pan115TestProfile(server.URL),
			Path:           "/upload.bin",
			Name:           "upload.bin",
			Size:           1024,
			ConflictPolicy: ConflictPolicyAutoRenameNew,
			Strategy:       "download_upload",
		}), "pan115_family_real_upload")
	})

	t.Run("quark live upload requires local file", func(t *testing.T) {
		server, _ := newShareFamilyTestServer(t, "quark")
		defer server.Close()

		originalClient := providerHTTPClient
		providerHTTPClient = server.Client()
		defer func() { providerHTTPClient = originalClient }()

		registry := NewRegistry(DefaultCatalog()...)
		entry, ok := registry.Get("quark")
		if !ok {
			t.Fatal("expected quark")
		}
		assertLocalFileMissingUploadBlocked(t, entry.Adapter.Upload(UploadRequest{
			Profile:        shareFamilyTestProfile(server.URL, "quark"),
			Path:           "/upload.bin",
			Name:           "upload.bin",
			Size:           1024,
			ConflictPolicy: ConflictPolicyAutoRenameNew,
			Strategy:       "download_upload",
		}), "share_family_real_upload")
	})

	t.Run("uc live upload requires local file", func(t *testing.T) {
		server, _ := newShareFamilyTestServer(t, "uc")
		defer server.Close()

		originalClient := providerHTTPClient
		providerHTTPClient = server.Client()
		defer func() { providerHTTPClient = originalClient }()

		registry := NewRegistry(DefaultCatalog()...)
		entry, ok := registry.Get("uc")
		if !ok {
			t.Fatal("expected uc")
		}
		assertLocalFileMissingUploadBlocked(t, entry.Adapter.Upload(UploadRequest{
			Profile:        shareFamilyTestProfile(server.URL, "uc"),
			Path:           "/upload.bin",
			Name:           "upload.bin",
			Size:           1024,
			ConflictPolicy: ConflictPolicyAutoRenameNew,
			Strategy:       "download_upload",
		}), "share_family_real_upload")
	})

	t.Run("baidu live upload requires local file", func(t *testing.T) {
		server, _ := newBaiduFamilyTestServer(t)
		defer server.Close()

		originalClient := providerHTTPClient
		providerHTTPClient = server.Client()
		defer func() { providerHTTPClient = originalClient }()

		registry := NewRegistry(DefaultCatalog()...)
		entry, ok := registry.Get("baidu_netdisk")
		if !ok {
			t.Fatal("expected baidu_netdisk")
		}
		assertLocalFileMissingUploadBlocked(t, entry.Adapter.Upload(UploadRequest{
			Profile:        AuthProfile{ProviderKey: "baidu_netdisk", Token: "token-live", Extra: map[string]string{"apiEndpoint": server.URL, "pcsEndpoint": server.URL}},
			Path:           "/upload.bin",
			Name:           "upload.bin",
			Size:           1024,
			ConflictPolicy: ConflictPolicyAutoRenameNew,
			Strategy:       "download_upload",
		}), "baidu_family_real_upload")
	})

	t.Run("189cloud live upload requires local file", func(t *testing.T) {
		server, _ := newCloud189TestServer(t)
		defer server.Close()

		originalClient := providerHTTPClient
		providerHTTPClient = server.Client()
		defer func() { providerHTTPClient = originalClient }()

		registry := NewRegistry(DefaultCatalog()...)
		entry, ok := registry.Get("189cloud")
		if !ok {
			t.Fatal("expected 189cloud")
		}
		assertLocalFileMissingUploadBlocked(t, entry.Adapter.Upload(UploadRequest{
			Profile:        cloud189WritableProfile(server.URL),
			Path:           "/upload.bin",
			Name:           "upload.bin",
			Size:           1024,
			ConflictPolicy: ConflictPolicyAutoRenameNew,
			Strategy:       "download_upload",
		}), "cloud189_family_real_upload")
	})

	t.Run("xunlei hash miss without local fallback stays blocked", func(t *testing.T) {
		server, _ := newXunleiHashFamilyTestServer(t)
		defer server.Close()

		originalClient := providerHTTPClient
		providerHTTPClient = server.Client()
		defer func() { providerHTTPClient = originalClient }()

		registry := NewRegistry(DefaultCatalog()...)
		entry, ok := registry.Get("xunlei")
		if !ok {
			t.Fatal("expected xunlei")
		}
		assertHashMissWithoutLocalFallbackBlocked(t, entry.Adapter.Upload(UploadRequest{
			Profile:        AuthProfile{ProviderKey: "xunlei", Token: "token-live", Extra: map[string]string{"apiEndpoint": server.URL, "deviceId": "device-1"}},
			Path:           "/upload.bin",
			Name:           "upload.bin",
			Size:           1024,
			GCID:           "GCID-UPLOAD",
			ConflictPolicy: ConflictPolicyAutoRenameNew,
			Strategy:       "download_upload",
		}), "hash_family_real_upload")
	})

	t.Run("pikpak hash miss without local fallback stays blocked", func(t *testing.T) {
		server, _ := newPikPakHashFamilyTestServer(t)
		defer server.Close()

		originalClient := providerHTTPClient
		providerHTTPClient = server.Client()
		defer func() { providerHTTPClient = originalClient }()

		registry := NewRegistry(DefaultCatalog()...)
		entry, ok := registry.Get("pikpak")
		if !ok {
			t.Fatal("expected pikpak")
		}
		assertHashMissWithoutLocalFallbackBlocked(t, entry.Adapter.Upload(UploadRequest{
			Profile:        AuthProfile{ProviderKey: "pikpak", Token: "token-live", Extra: map[string]string{"apiEndpoint": server.URL, "deviceId": "device-pk"}},
			Path:           "/upload.bin",
			Name:           "upload.bin",
			Size:           1024,
			GCID:           "GCID-UPLOAD",
			ConflictPolicy: ConflictPolicyAutoRenameNew,
			Strategy:       "download_upload",
		}), "hash_family_real_upload")
	})
}

func assertLocalFileMissingUploadBlocked(t *testing.T, result UploadResult, wantMode string) {
	t.Helper()
	if result.OK {
		t.Fatalf("expected upload to stay blocked on missing local file, got %+v", result)
	}
	if result.Status != "local_file_missing" {
		t.Fatalf("expected local_file_missing, got %+v", result)
	}
	if !containsProviderSubstring(result.Message, "local file") {
		t.Fatalf("expected message containing local file, got %+v", result)
	}
	if result.Mode != wantMode {
		t.Fatalf("expected mode %s, got %+v", wantMode, result)
	}
}

func assertHashMissWithoutLocalFallbackBlocked(t *testing.T, result UploadResult, wantMode string) {
	t.Helper()
	if result.OK {
		t.Fatalf("expected hash miss fallback to stay blocked without local file, got %+v", result)
	}
	if result.Status != "hash_miss" {
		t.Fatalf("expected hash_miss, got %+v", result)
	}
	if !containsProviderSubstring(result.Message, "no local file") {
		t.Fatalf("expected message containing no local file, got %+v", result)
	}
	if result.Mode != wantMode {
		t.Fatalf("expected mode %s, got %+v", wantMode, result)
	}
}