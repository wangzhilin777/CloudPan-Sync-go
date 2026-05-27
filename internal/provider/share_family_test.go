package provider

import "testing"

func TestShareFamilyAdapterRequiresCookieAndPwdID(t *testing.T) {
	registry := NewRegistry(DefaultCatalog()...)
	entry, ok := registry.Get("quark")
	if !ok {
		t.Fatal("expected quark entry")
	}

	result := entry.Adapter.ValidateAuth(AuthProfile{
		ProviderKey: "quark",
	})
	if result.OK {
		t.Fatal("expected validation to fail without cookie")
	}
	if result.Status != "missing_cookie" {
		t.Fatalf("expected missing_cookie, got %s", result.Status)
	}

	result = entry.Adapter.ValidateAuth(AuthProfile{
		ProviderKey: "quark",
		Cookie:      "cookie-1",
		Extra:       map[string]string{},
	})
	if result.OK {
		t.Fatal("expected validation to fail without pwdId")
	}
	if result.Status != "missing_pwd_id" {
		t.Fatalf("expected missing_pwd_id, got %s", result.Status)
	}
}

func TestShareFamilyAdapterSupportsFastUploadCandidate(t *testing.T) {
	registry := NewRegistry(DefaultCatalog()...)
	entry, ok := registry.Get("uc")
	if !ok {
		t.Fatal("expected uc entry")
	}
	check := entry.Adapter.FastUploadCheck(FastUploadCheckRequest{
		Profile: AuthProfile{
			ProviderKey: "uc",
			Cookie:      "cookie-1",
			Extra:       map[string]string{"pwdId": "pwd-1"},
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
