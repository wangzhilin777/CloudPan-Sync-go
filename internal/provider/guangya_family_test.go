package provider

import "testing"

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

func TestGuangyaFamilyAdapterSupportsFastUploadCandidate(t *testing.T) {
	registry := NewRegistry(DefaultCatalog()...)
	entry, ok := registry.Get("guangya")
	if !ok {
		t.Fatal("expected guangya entry")
	}
	check := entry.Adapter.FastUploadCheck(FastUploadCheckRequest{
		Profile: AuthProfile{
			ProviderKey: "guangya",
			Token:       "token-guangya",
		},
		Path: "/a.bin",
		Name: "a.bin",
		Size: 1024,
		MD5:  "md5-1",
	})
	if !check.OK || !check.Candidate {
		t.Fatalf("expected fast upload candidate, got %+v", check)
	}
}
