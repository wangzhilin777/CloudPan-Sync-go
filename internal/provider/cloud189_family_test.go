package provider

import "testing"

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
	registry := NewRegistry(DefaultCatalog()...)
	entry, ok := registry.Get("189cloud")
	if !ok {
		t.Fatal("expected 189cloud entry")
	}
	check := entry.Adapter.FastUploadCheck(FastUploadCheckRequest{
		Profile: AuthProfile{
			ProviderKey: "189cloud",
			Cookie:      "SESSION=cloud189",
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
