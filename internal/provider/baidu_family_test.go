package provider

import "testing"

func TestBaiduFamilyAdapterRequiresTokenOrCookie(t *testing.T) {
	registry := NewRegistry(DefaultCatalog()...)
	entry, ok := registry.Get("baidu_netdisk")
	if !ok {
		t.Fatal("expected baidu_netdisk entry")
	}

	result := entry.Adapter.ValidateAuth(AuthProfile{
		ProviderKey: "baidu_netdisk",
	})
	if result.OK {
		t.Fatal("expected validation to fail without token or cookie")
	}
	if result.Status != "missing_access_token_or_cookie" {
		t.Fatalf("expected missing_access_token_or_cookie, got %s", result.Status)
	}
}

func TestBaiduFamilyAdapterSupportsFastUploadCandidate(t *testing.T) {
	server, _ := newBaiduFamilyTestServer(t)
	t.Cleanup(server.Close)

	originalClient := providerHTTPClient
	providerHTTPClient = server.Client()
	t.Cleanup(func() { providerHTTPClient = originalClient })

	registry := NewRegistry(DefaultCatalog()...)
	entry, ok := registry.Get("baidu_netdisk")
	if !ok {
		t.Fatal("expected baidu_netdisk entry")
	}
	check := entry.Adapter.FastUploadCheck(FastUploadCheckRequest{
		Profile: AuthProfile{
			ProviderKey: "baidu_netdisk",
			Token:       "token-live",
			Extra: map[string]string{
				"apiEndpoint": server.URL,
				"pcsEndpoint": server.URL,
			},
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
