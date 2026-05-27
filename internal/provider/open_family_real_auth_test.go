package provider

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOpenFamilyAdapterValidatesAliyunAgainstLiveEndpoints(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if auth := r.Header.Get("Authorization"); auth != "Bearer token-live" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/user/get":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"domain_id":        "bj1",
				"user_id":          "user-1",
				"default_drive_id": "drive-1",
			})
		case "/v2/drive/get_default_drive":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"drive_id": "drive-1",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	originalClient := providerHTTPClient
	providerHTTPClient = server.Client()
	t.Cleanup(func() { providerHTTPClient = originalClient })

	registry := NewRegistry(DefaultCatalog()...)
	entry, ok := registry.Get("aliyundrive_open")
	if !ok {
		t.Fatal("expected aliyundrive_open entry")
	}
	result := entry.Adapter.ValidateAuth(AuthProfile{
		ProviderKey: "aliyundrive_open",
		Token:       "token-live",
		Extra: map[string]string{
			"domainId":    "bj1",
			"driveId":     "drive-1",
			"apiEndpoint": server.URL,
		},
	})
	if !result.OK {
		t.Fatalf("expected validation success, got %+v", result)
	}
	if result.Mode != "open_family_real_auth" {
		t.Fatalf("expected open_family_real_auth mode, got %s", result.Mode)
	}
}

func TestOpenFamilyAdapterFailsAliyunWhenDriveIDMismatches(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/user/get":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"domain_id": "bj1",
				"user_id":   "user-1",
			})
		case "/v2/drive/get_default_drive":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"drive_id": "drive-actual",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	originalClient := providerHTTPClient
	providerHTTPClient = server.Client()
	t.Cleanup(func() { providerHTTPClient = originalClient })

	registry := NewRegistry(DefaultCatalog()...)
	entry, ok := registry.Get("aliyundrive_open")
	if !ok {
		t.Fatal("expected aliyundrive_open entry")
	}
	result := entry.Adapter.ValidateAuth(AuthProfile{
		ProviderKey: "aliyundrive_open",
		Token:       "token-live",
		Extra: map[string]string{
			"domainId":    "bj1",
			"driveId":     "drive-expected",
			"apiEndpoint": server.URL,
		},
	})
	if result.OK {
		t.Fatalf("expected validation failure, got %+v", result)
	}
	if result.Status != "drive_id_mismatch" {
		t.Fatalf("expected drive_id_mismatch, got %s", result.Status)
	}
}
