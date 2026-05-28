package provider

import "testing"

func TestPan115FamilyAdapterRequiresTokenOrCookie(t *testing.T) {
	registry := NewRegistry(DefaultCatalog()...)
	entry, ok := registry.Get("115_open")
	if !ok {
		t.Fatal("expected 115_open entry")
	}

	result := entry.Adapter.ValidateAuth(AuthProfile{
		ProviderKey: "115_open",
	})
	if result.OK {
		t.Fatal("expected validation to fail without token or cookie")
	}
	if result.Status != "missing_access_token_or_cookie" {
		t.Fatalf("expected missing_access_token_or_cookie, got %s", result.Status)
	}
}

func TestPan115FamilyAdapterSupportsFastUploadCandidate(t *testing.T) {
	server, _ := newPan115FamilyTestServer(t)
	t.Cleanup(server.Close)

	originalClient := providerHTTPClient
	providerHTTPClient = server.Client()
	t.Cleanup(func() { providerHTTPClient = originalClient })

	registry := NewRegistry(DefaultCatalog()...)
	entry, ok := registry.Get("115_open")
	if !ok {
		t.Fatal("expected 115_open entry")
	}
	check := entry.Adapter.FastUploadCheck(FastUploadCheckRequest{
		Profile: AuthProfile{
			ProviderKey: "115_open",
			Cookie:      "cookie-live",
			Extra: map[string]string{
				"listEndpoint":           server.URL + "/files",
				"infoEndpoint":           server.URL + "/files/get_info",
				"mkdirEndpoint":          server.URL + "/files/add",
				"uploadInitEndpoint":     server.URL + "/open/upload/init",
				"uploadGetTokenEndpoint": server.URL + "/open/upload/get_token",
			},
		},
		Path: "/a.bin",
		Name: "a.bin",
		Size: 1024,
		SHA1: "sha1-1",
	})
	if !check.OK || !check.Candidate {
		t.Fatalf("expected fast upload candidate, got %+v", check)
	}
}
