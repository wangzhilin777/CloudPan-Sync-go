package provider

import "testing"

func TestHashFamilyAdapterRequiresToken(t *testing.T) {
	registry := NewRegistry(DefaultCatalog()...)
	entry, ok := registry.Get("xunlei")
	if !ok {
		t.Fatal("expected xunlei entry")
	}
	result := entry.Adapter.ValidateAuth(AuthProfile{
		ProviderKey: "xunlei",
	})
	if result.OK {
		t.Fatal("expected validation to fail without token")
	}
	if result.Status != "missing_access_token" {
		t.Fatalf("expected missing_access_token, got %s", result.Status)
	}
}

func TestHashFamilyAdapterSupportsFastUploadCandidate(t *testing.T) {
	registry := NewRegistry(DefaultCatalog()...)
	entry, ok := registry.Get("pikpak")
	if !ok {
		t.Fatal("expected pikpak entry")
	}
	check := entry.Adapter.FastUploadCheck(FastUploadCheckRequest{
		Profile: AuthProfile{
			ProviderKey: "pikpak",
			Token:       "token-1",
		},
		Path: "/a.bin",
		Name: "a.bin",
		Size: 1024,
		GCID: "gcid-1",
	})
	if !check.OK || !check.Candidate {
		t.Fatalf("expected fast upload candidate, got %+v", check)
	}
}
