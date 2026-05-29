package provider

import "testing"

func TestDefaultCatalogProviderContractCoverage(t *testing.T) {
	registry := NewRegistry(DefaultCatalog()...)
	items := registry.List()
	if len(items) != 10 {
		t.Fatalf("expected 10 providers, got %d", len(items))
	}

	expectedProtocolGroups := map[string]string{
		"guangya":          "guangya",
		"aliyundrive_open": "aliyun_123_open",
		"123_open":         "aliyun_123_open",
		"115_open":         "115_open",
		"quark":            "quark_uc",
		"uc":               "quark_uc",
		"xunlei":           "xunlei_pikpak",
		"pikpak":           "xunlei_pikpak",
		"baidu_netdisk":    "baidu_netdisk",
		"189cloud":         "189cloud",
	}

	for key, protocolGroup := range expectedProtocolGroups {
		entry, ok := registry.Get(key)
		if !ok {
			t.Fatalf("expected provider %s to exist", key)
		}
		if entry.Meta.ProtocolGroup != protocolGroup {
			t.Fatalf("expected provider %s protocol group %s, got %s", key, protocolGroup, entry.Meta.ProtocolGroup)
		}
		if entry.Meta.DisplayName == "" {
			t.Fatalf("expected provider %s display name", key)
		}
		if len(entry.Meta.AuthModes) == 0 {
			t.Fatalf("expected provider %s auth modes", key)
		}
		if len(entry.Meta.FallbackModes) == 0 {
			t.Fatalf("expected provider %s fallback modes", key)
		}
		if len(entry.Meta.ConflictPolicies) == 0 {
			t.Fatalf("expected provider %s conflict policies", key)
		}
		if !entry.Capability.SupportsAuthValidation ||
			!entry.Capability.SupportsList ||
			!entry.Capability.SupportsMetadata ||
			!entry.Capability.SupportsCreateDir ||
			!entry.Capability.SupportsFastUpload ||
			!entry.Capability.SupportsUpload {
			t.Fatalf("expected provider %s to declare the full v1 capability set, got %+v", key, entry.Capability)
		}
	}
}

func TestDefaultCatalogMissingCredentialsFailValidation(t *testing.T) {
	registry := NewRegistry(DefaultCatalog()...)
	for _, entry := range registry.List() {
		result := entry.Adapter.ValidateAuth(AuthProfile{
			ProviderKey: entry.Meta.Key,
			AuthMode:    firstString(entry.Meta.AuthModes),
		})
		if result.OK {
			t.Fatalf("expected provider %s missing credentials validation to fail, got %+v", entry.Meta.Key, result)
		}
		if result.Status == "" {
			t.Fatalf("expected provider %s missing credentials status", entry.Meta.Key)
		}
	}
}

func firstString(items []string) string {
	if len(items) == 0 {
		return ""
	}
	return items[0]
}
