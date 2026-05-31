package provider

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultCatalogUploadAuthInvalidContracts(t *testing.T) {
	makeLocalFile := func(t *testing.T) string {
		t.Helper()
		path := filepath.Join(t.TempDir(), "upload.bin")
		if err := os.WriteFile(path, []byte("auth-invalid-upload"), 0o600); err != nil {
			t.Fatalf("write local file: %v", err)
		}
		return path
	}

	t.Run("guangya invalid token is rejected during upload", func(t *testing.T) {
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
		assertAuthInvalidUploadBlocked(t, entry.Adapter.Upload(UploadRequest{
			Profile: AuthProfile{
				ProviderKey: "guangya",
				Token:       "bad-token",
				Extra:       map[string]string{"apiEndpoint": server.URL, "parentId": "parent-gy"},
			},
			Path:           "/auth-invalid.bin",
			ParentID:       "parent-gy",
			Name:           "auth-invalid.bin",
			LocalPath:      makeLocalFile(t),
			Size:           16,
			ConflictPolicy: ConflictPolicyAutoRenameNew,
			Strategy:       "download_upload",
		}), "guangya_family_real_auth")
	})

	t.Run("aliyun open invalid token is rejected during upload", func(t *testing.T) {
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
		assertAuthInvalidUploadBlocked(t, entry.Adapter.Upload(UploadRequest{
			Profile: AuthProfile{
				ProviderKey: "aliyundrive_open",
				Token:       "bad-token",
				Extra:       map[string]string{"domainId": "bj1", "driveId": "drive-1", "apiEndpoint": server.URL},
			},
			Path:           "/auth-invalid.bin",
			Name:           "auth-invalid.bin",
			LocalPath:      makeLocalFile(t),
			Size:           16,
			ConflictPolicy: ConflictPolicyAutoRenameNew,
			Strategy:       "download_upload",
		}), "")
	})

	t.Run("123 open invalid token is rejected during upload", func(t *testing.T) {
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
		assertAuthInvalidUploadBlocked(t, entry.Adapter.Upload(UploadRequest{
			Profile: AuthProfile{
				ProviderKey: "123_open",
				Token:       "bad-token",
				Extra:       map[string]string{"apiEndpoint": server.URL},
			},
			Path:           "/auth-invalid.bin",
			Name:           "auth-invalid.bin",
			LocalPath:      makeLocalFile(t),
			Size:           16,
			ConflictPolicy: ConflictPolicyAutoRenameNew,
			Strategy:       "download_upload",
		}), "")
	})

	t.Run("115 invalid token is rejected during upload", func(t *testing.T) {
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
		assertAuthInvalidUploadBlocked(t, entry.Adapter.Upload(UploadRequest{
			Profile: AuthProfile{
				ProviderKey: "115_open",
				Token:       "bad-token",
				Extra:       pan115TestProfile(server.URL).Extra,
			},
			Path:           "/auth-invalid.bin",
			Name:           "auth-invalid.bin",
			LocalPath:      makeLocalFile(t),
			Size:           16,
			ConflictPolicy: ConflictPolicyAutoRenameNew,
			Strategy:       "download_upload",
		}), "")
	})

	t.Run("quark invalid token is rejected during upload", func(t *testing.T) {
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
		assertAuthInvalidUploadBlocked(t, entry.Adapter.Upload(UploadRequest{
			Profile: AuthProfile{
				ProviderKey: "quark",
				Cookie:      "bad-cookie",
				Extra:       map[string]string{"apiEndpoint": server.URL, "pwdId": "pwd-live"},
			},
			Path:           "/auth-invalid.bin",
			Name:           "auth-invalid.bin",
			LocalPath:      makeLocalFile(t),
			Size:           16,
			ConflictPolicy: ConflictPolicyAutoRenameNew,
			Strategy:       "download_upload",
		}), "")
	})

	t.Run("uc invalid token is rejected during upload", func(t *testing.T) {
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
		assertAuthInvalidUploadBlocked(t, entry.Adapter.Upload(UploadRequest{
			Profile: AuthProfile{
				ProviderKey: "uc",
				Cookie:      "bad-cookie",
				Extra:       map[string]string{"apiEndpoint": server.URL, "pwdId": "pwd-live"},
			},
			Path:           "/auth-invalid.bin",
			Name:           "auth-invalid.bin",
			LocalPath:      makeLocalFile(t),
			Size:           16,
			ConflictPolicy: ConflictPolicyAutoRenameNew,
			Strategy:       "download_upload",
		}), "")
	})

	t.Run("xunlei invalid token is rejected during upload", func(t *testing.T) {
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
		assertAuthInvalidUploadBlocked(t, entry.Adapter.Upload(UploadRequest{
			Profile: AuthProfile{
				ProviderKey: "xunlei",
				Token:       "bad-token",
				Extra:       map[string]string{"apiEndpoint": server.URL, "deviceId": "device-1"},
			},
			Path:           "/auth-invalid.bin",
			Name:           "auth-invalid.bin",
			LocalPath:      makeLocalFile(t),
			Size:           16,
			ConflictPolicy: ConflictPolicyAutoRenameNew,
			Strategy:       "download_upload",
			GCID:           "GCID-AUTH",
		}), "")
	})

	t.Run("pikpak invalid token is rejected during upload", func(t *testing.T) {
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
		assertAuthInvalidUploadBlocked(t, entry.Adapter.Upload(UploadRequest{
			Profile: AuthProfile{
				ProviderKey: "pikpak",
				Token:       "bad-token",
				Extra:       map[string]string{"apiEndpoint": server.URL, "deviceId": "device-pk"},
			},
			Path:           "/auth-invalid.bin",
			Name:           "auth-invalid.bin",
			LocalPath:      makeLocalFile(t),
			Size:           16,
			ConflictPolicy: ConflictPolicyAutoRenameNew,
			Strategy:       "download_upload",
			GCID:           "GCID-AUTH",
		}), "")
	})

	t.Run("baidu invalid token is rejected during upload", func(t *testing.T) {
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
		assertAuthInvalidUploadBlocked(t, entry.Adapter.Upload(UploadRequest{
			Profile: AuthProfile{
				ProviderKey: "baidu_netdisk",
				Token:       "bad-token",
				Extra:       map[string]string{"apiEndpoint": server.URL, "pcsEndpoint": server.URL},
			},
			Path:           "/auth-invalid.bin",
			Name:           "auth-invalid.bin",
			LocalPath:      makeLocalFile(t),
			Size:           16,
			ConflictPolicy: ConflictPolicyAutoRenameNew,
			Strategy:       "download_upload",
		}), "")
	})

	t.Run("189cloud invalid token is rejected during upload", func(t *testing.T) {
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
		profile := cloud189WritableProfile(server.URL)
		profile.Token = "bad-token"
		profile.Extra["accessToken"] = "bad-token"
		assertAuthInvalidUploadBlocked(t, entry.Adapter.Upload(UploadRequest{
			Profile:        profile,
			Path:           "/auth-invalid.bin",
			Name:           "auth-invalid.bin",
			LocalPath:      makeLocalFile(t),
			Size:           16,
			ConflictPolicy: ConflictPolicyAutoRenameNew,
			Strategy:       "download_upload",
		}), "")
	})
}

func assertAuthInvalidUploadBlocked(t *testing.T, result UploadResult, wantMode string) {
	t.Helper()
	if result.OK {
		t.Fatalf("expected auth_invalid upload to stay blocked, got %+v", result)
	}
	if result.Status != "auth_invalid" {
		t.Fatalf("expected auth_invalid, got %+v", result)
	}
	if !containsProviderSubstring(result.Message, "reject") && !containsProviderSubstring(result.Message, "auth_invalid") && !containsProviderSubstring(result.Message, "unauthorized") {
		t.Fatalf("expected auth invalid message, got %+v", result)
	}
	if wantMode != "" && result.Mode != wantMode {
		t.Fatalf("expected mode %s, got %+v", wantMode, result)
	}
}
