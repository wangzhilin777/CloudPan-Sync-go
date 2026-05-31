package provider

import "testing"

func TestDefaultCatalogFastUploadInputContracts(t *testing.T) {
	registry := NewRegistry(DefaultCatalog()...)
	tests := []struct {
		key             string
		wantInputs      []string
		wantFallback    []string
		wantProtocol    string
		wantAuthAtLeast int
	}{
		{key: "aliyundrive_open", wantInputs: []string{"sha1", "size"}, wantFallback: []string{"download_upload"}, wantProtocol: "aliyun_123_open", wantAuthAtLeast: 1},
		{key: "123_open", wantInputs: []string{"md5", "size"}, wantFallback: []string{"download_upload"}, wantProtocol: "aliyun_123_open", wantAuthAtLeast: 2},
		{key: "xunlei", wantInputs: []string{"gcid", "size"}, wantFallback: []string{"download_upload"}, wantProtocol: "xunlei_pikpak", wantAuthAtLeast: 2},
		{key: "guangya", wantInputs: []string{"md5", "size", "name"}, wantFallback: []string{"download_upload"}, wantProtocol: "guangya", wantAuthAtLeast: 2},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			entry, ok := registry.Get(tt.key)
			if !ok {
				t.Fatalf("expected provider %s to exist", tt.key)
			}
			if entry.Meta.ProtocolGroup != tt.wantProtocol {
				t.Fatalf("expected protocolGroup %s for %s, got %s", tt.wantProtocol, tt.key, entry.Meta.ProtocolGroup)
			}
			if !equalStringSlices(entry.Meta.FastUploadInputs, tt.wantInputs) {
				t.Fatalf("expected FastUploadInputs %#v for %s, got %#v", tt.wantInputs, tt.key, entry.Meta.FastUploadInputs)
			}
			if !equalStringSlices(entry.Meta.FallbackModes, tt.wantFallback) {
				t.Fatalf("expected FallbackModes %#v for %s, got %#v", tt.wantFallback, tt.key, entry.Meta.FallbackModes)
			}
			if len(entry.Meta.AuthModes) < tt.wantAuthAtLeast {
				t.Fatalf("expected at least %d auth modes for %s, got %#v", tt.wantAuthAtLeast, tt.key, entry.Meta.AuthModes)
			}
		})
	}
}

func equalStringSlices(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
