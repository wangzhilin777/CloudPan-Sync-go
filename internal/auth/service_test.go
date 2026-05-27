package auth

import (
	"context"
	"path/filepath"
	"testing"

	"cloudpan-sync-go/internal/provider"
	sqlitestore "cloudpan-sync-go/internal/store/sqlite"
)

func TestServiceCreateAndValidateProfile(t *testing.T) {
	ctx := context.Background()
	store, err := sqlitestore.New(ctx, filepath.Join(t.TempDir(), "auth.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer func() { _ = store.Close() }()

	svc := NewService(store, provider.NewRegistry(provider.DefaultCatalog()...))
	profile, err := svc.CreateProfile(ctx, CreateProfileInput{
		ProviderKey: "guangya",
		AuthMode:    "manual_token",
		DisplayName: "demo",
		Token:       "token-1",
	})
	if err != nil {
		t.Fatalf("CreateProfile() error = %v", err)
	}
	if profile.ID == "" {
		t.Fatal("expected profile id")
	}

	validation, ok, err := svc.ValidateProfile(ctx, profile.ID)
	if err != nil {
		t.Fatalf("ValidateProfile() error = %v", err)
	}
	if !ok || !validation.OK {
		t.Fatalf("expected successful validation, ok=%v validation=%+v", ok, validation)
	}
}
