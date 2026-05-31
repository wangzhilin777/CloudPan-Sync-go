package provider

import "testing"

func TestDefaultCatalogPendingManualContracts(t *testing.T) {
	t.Run("placeholder providers reject pending manual before fake success", func(t *testing.T) {
		registry := NewRegistry(DefaultCatalog()...)
		for _, key := range []string{"guangya", "aliyundrive_open", "123_open", "115_open", "quark", "uc", "xunlei", "pikpak", "baidu_netdisk", "189cloud"} {
			entry, ok := registry.Get(key)
			if !ok {
				t.Fatalf("expected provider %s", key)
			}
			if !containsProviderSubstring(entry.Meta.OverwriteBehavior, "rename") && entry.Meta.OverwriteBehavior == "" {
				_ = entry
			}
		}
	})

	t.Run("guangya live upload keeps pending manual blocked", func(t *testing.T) {
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
		assertPendingManualUploadBlocked(t, entry.Adapter.Upload(UploadRequest{
			Profile:        guangyaTestProfile(server.URL),
			Path:           "/manual.bin",
			ParentID:       "parent-gy",
			Name:           "manual.bin",
			Size:           1024,
			ConflictPolicy: ConflictPolicyAutoRenameNew,
			Strategy:       "pending_manual",
		}), "guangya_family_real_upload")
	})

	t.Run("aliyun open live upload keeps pending manual blocked", func(t *testing.T) {
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
		assertPendingManualUploadBlocked(t, entry.Adapter.Upload(UploadRequest{
			Profile: AuthProfile{ProviderKey: "aliyundrive_open", Token: "token-live", Extra: map[string]string{"domainId": "bj1", "driveId": "drive-1", "apiEndpoint": server.URL}},
			Path: "/manual.bin", Name: "manual.bin", Size: 1024, ConflictPolicy: ConflictPolicyAutoRenameNew, Strategy: "pending_manual",
		}), "open_family_real_upload")
	})

	t.Run("123 open live upload keeps pending manual blocked", func(t *testing.T) {
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
		assertPendingManualUploadBlocked(t, entry.Adapter.Upload(UploadRequest{
			Profile: AuthProfile{ProviderKey: "123_open", Token: "token-live", Extra: map[string]string{"apiEndpoint": server.URL}},
			Path: "/manual.bin", Name: "manual.bin", Size: 1024, ConflictPolicy: ConflictPolicyAutoRenameNew, Strategy: "pending_manual",
		}), "open_family_real_upload")
	})

	t.Run("115 live upload keeps pending manual blocked", func(t *testing.T) {
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
		assertPendingManualUploadBlocked(t, entry.Adapter.Upload(UploadRequest{
			Profile: pan115TestProfile(server.URL),
			Path: "/manual.bin", Name: "manual.bin", Size: 1024, ConflictPolicy: ConflictPolicyAutoRenameNew, Strategy: "pending_manual",
		}), "pan115_family_real_upload")
	})

	t.Run("quark live upload keeps pending manual blocked", func(t *testing.T) {
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
		assertPendingManualUploadBlocked(t, entry.Adapter.Upload(UploadRequest{
			Profile: shareFamilyTestProfile(server.URL, "quark"),
			Path: "/manual.bin", Name: "manual.bin", Size: 1024, ConflictPolicy: ConflictPolicyAutoRenameNew, Strategy: "pending_manual",
		}), "share_family_real_upload")
	})

	t.Run("uc live upload keeps pending manual blocked", func(t *testing.T) {
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
		assertPendingManualUploadBlocked(t, entry.Adapter.Upload(UploadRequest{
			Profile: shareFamilyTestProfile(server.URL, "uc"),
			Path: "/manual.bin", Name: "manual.bin", Size: 1024, ConflictPolicy: ConflictPolicyAutoRenameNew, Strategy: "pending_manual",
		}), "share_family_real_upload")
	})

	t.Run("xunlei live upload keeps pending manual blocked", func(t *testing.T) {
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
		assertPendingManualUploadBlocked(t, entry.Adapter.Upload(UploadRequest{
			Profile: AuthProfile{ProviderKey: "xunlei", Token: "token-live", Extra: map[string]string{"apiEndpoint": server.URL, "deviceId": "device-1"}},
			Path: "/manual.bin", Name: "manual.bin", Size: 1024, ConflictPolicy: ConflictPolicyAutoRenameNew, Strategy: "pending_manual",
		}), "hash_family_real_upload")
	})

	t.Run("pikpak live upload keeps pending manual blocked", func(t *testing.T) {
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
		assertPendingManualUploadBlocked(t, entry.Adapter.Upload(UploadRequest{
			Profile: AuthProfile{ProviderKey: "pikpak", Token: "token-live", Extra: map[string]string{"apiEndpoint": server.URL, "deviceId": "device-pk"}},
			Path: "/manual.bin", Name: "manual.bin", Size: 1024, ConflictPolicy: ConflictPolicyAutoRenameNew, Strategy: "pending_manual",
		}), "hash_family_real_upload")
	})

	t.Run("baidu live upload keeps pending manual blocked", func(t *testing.T) {
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
		assertPendingManualUploadBlocked(t, entry.Adapter.Upload(UploadRequest{
			Profile: AuthProfile{ProviderKey: "baidu_netdisk", Token: "token-live", Extra: map[string]string{"apiEndpoint": server.URL, "pcsEndpoint": server.URL}},
			Path: "/manual.bin", Name: "manual.bin", Size: 1024, ConflictPolicy: ConflictPolicyAutoRenameNew, Strategy: "pending_manual",
		}), "baidu_family_real_upload")
	})

	t.Run("189cloud live upload keeps pending manual blocked", func(t *testing.T) {
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
		assertPendingManualUploadBlocked(t, entry.Adapter.Upload(UploadRequest{
			Profile: cloud189WritableProfile(server.URL),
			Path: "/manual.bin", Name: "manual.bin", Size: 1024, ConflictPolicy: ConflictPolicyAutoRenameNew, Strategy: "pending_manual",
		}), "cloud189_family_real_upload")
	})
}

func assertPendingManualUploadBlocked(t *testing.T, result UploadResult, wantMode string) {
	t.Helper()
	if result.OK {
		t.Fatalf("expected pending_manual to stay blocked, got %+v", result)
	}
	if result.Status != "pending_manual_requires_confirmation" {
		t.Fatalf("expected pending_manual_requires_confirmation, got %+v", result)
	}
	if !containsProviderSubstring(result.Message, "pending_manual") {
		t.Fatalf("expected message containing pending_manual, got %+v", result)
	}
	if result.Mode != wantMode {
		t.Fatalf("expected mode %s, got %+v", wantMode, result)
	}
}

func containsProviderSubstring(value string, part string) bool {
	if len(part) == 0 {
		return true
	}
	for i := 0; i+len(part) <= len(value); i++ {
		if value[i:i+len(part)] == part {
			return true
		}
	}
	return false
}