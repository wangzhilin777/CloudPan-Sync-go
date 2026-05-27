package task

import (
	"context"
	"path/filepath"
	"testing"

	"cloudpan-sync-go/internal/auth"
	"cloudpan-sync-go/internal/planner"
	"cloudpan-sync-go/internal/provider"
	sqlitestore "cloudpan-sync-go/internal/store/sqlite"
)

func TestServiceCreateRunRetryTask(t *testing.T) {
	ctx := context.Background()
	store, err := sqlitestore.New(ctx, filepath.Join(t.TempDir(), "task.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer func() { _ = store.Close() }()

	registry := provider.NewRegistry(provider.DefaultCatalog()...)
	authSvc := auth.NewService(store, registry)
	svc := NewService(store, registry, authSvc)
	profile, err := authSvc.CreateProfile(ctx, auth.CreateProfileInput{
		ProviderKey: "123_open",
		AuthMode:    "manual_token",
		DisplayName: "123 smoke",
		Token:       "token-1",
	})
	if err != nil {
		t.Fatalf("CreateProfile() error = %v", err)
	}
	detail, err := svc.Create(ctx, CreateRequest{
		SourceProvider:  "guangya",
		TargetProvider:  "123_open",
		TargetProfileID: profile.ID,
		ThresholdMB:     10,
		Entries: []planner.SourceEntry{
			{Path: "/a.bin", Size: 1024, MD5: "abc"},
			{Path: "/b.bin", Size: 40 * 1024 * 1024},
		},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if detail.Task.State != StateReady {
		t.Fatalf("expected ready state, got %s", detail.Task.State)
	}

	running, ok, err := svc.Run(ctx, detail.Task.ID)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !ok {
		t.Fatal("expected task to exist")
	}
	if len(running.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(running.Results))
	}
	if running.Task.State != StateCompletedWithErrors {
		t.Fatalf("expected completed_with_errors, got %s", running.Task.State)
	}

	retried, ok, err := svc.Retry(ctx, detail.Task.ID)
	if err != nil {
		t.Fatalf("Retry() error = %v", err)
	}
	if !ok || retried.Task.State != StateReady {
		t.Fatalf("expected retry to reset state, ok=%v state=%s", ok, retried.Task.State)
	}
}
