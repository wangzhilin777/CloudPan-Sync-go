package provider

import "testing"

func TestOpenFamilyAdapterRequiresDomainAndDriveForAliyun(t *testing.T) {
	registry := NewRegistry(DefaultCatalog()...)
	entry, ok := registry.Get("aliyundrive_open")
	if !ok {
		t.Fatal("expected aliyundrive_open entry")
	}
	result := entry.Adapter.ValidateAuth(AuthProfile{
		ProviderKey: "aliyundrive_open",
		Token:       "token-1",
		Extra:       map[string]string{},
	})
	if result.OK {
		t.Fatal("expected validation to fail without domainId/driveId")
	}
	if result.Status != "missing_domain_or_drive_id" {
		t.Fatalf("expected missing_domain_or_drive_id, got %s", result.Status)
	}
}

func TestOpenFamilyAdapterSupports123FastUploadCandidate(t *testing.T) {
	registry := NewRegistry(DefaultCatalog()...)
	entry, ok := registry.Get("123_open")
	if !ok {
		t.Fatal("expected 123_open entry")
	}
	check := entry.Adapter.FastUploadCheck(FastUploadCheckRequest{
		Profile: AuthProfile{
			ProviderKey: "123_open",
			Token:       "token-1",
		},
		Path: "/a.bin",
		Name: "a.bin",
		Size: 1024,
		MD5:  "abc",
	})
	if !check.OK || !check.Candidate {
		t.Fatalf("expected fast upload candidate, got %+v", check)
	}
}
