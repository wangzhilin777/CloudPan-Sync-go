package task

import (
	"context"
	"os"
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
	if got := detail.Plan.Items[0].Path; got != "/a.bin" {
		t.Fatalf("expected leaf-first item order to keep /a.bin first, got %s", got)
	}
	switch riskProfile := detail.Plan.Metadata["riskProfile"].(type) {
	case planner.RiskProfile:
		if riskProfile.Mode != planner.RiskModeBalanced {
			t.Fatalf("expected default balanced risk mode, got %s", riskProfile.Mode)
		}
	case map[string]interface{}:
		if mode, _ := riskProfile["mode"].(string); mode != string(planner.RiskModeBalanced) {
			t.Fatalf("expected default balanced risk mode, got %v", riskProfile["mode"])
		}
	default:
		t.Fatalf("expected riskProfile metadata, got %#v", detail.Plan.Metadata["riskProfile"])
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
	if sequence, _ := running.Results[0].Payload["sequence"].(int); sequence != 1 {
		t.Fatalf("expected first result sequence=1, got %+v", running.Results[0].Payload["sequence"])
	}
	if running.Task.State != StateCompletedWithErrors {
		t.Fatalf("expected completed_with_errors, got %s", running.Task.State)
	}
	evidence, err := svc.RuntimeEvidence(ctx)
	if err != nil {
		t.Fatalf("RuntimeEvidence() error = %v", err)
	}
	if evidence.TotalTasks != 1 {
		t.Fatalf("expected TotalTasks=1, got %d", evidence.TotalTasks)
	}
	if len(evidence.RecentResults) == 0 {
		t.Fatal("expected recent task results to be populated")
	}
	if len(evidence.RecentProbes) == 0 {
		t.Fatal("expected recent provider probes to be populated")
	}
	statuses, err := svc.ProviderStatuses(ctx)
	if err != nil {
		t.Fatalf("ProviderStatuses() error = %v", err)
	}
	foundTarget := false
	for _, item := range statuses {
		if item.ProviderKey == "123_open" {
			foundTarget = true
			if item.LastTaskState == "" {
				t.Fatal("expected target provider to include LastTaskState")
			}
			if item.LatestProbe == "" {
				t.Fatal("expected target provider to include LatestProbe")
			}
		}
	}
	if !foundTarget {
		t.Fatal("expected 123_open in provider statuses")
	}

	retried, ok, err := svc.Retry(ctx, detail.Task.ID)
	if err != nil {
		t.Fatalf("Retry() error = %v", err)
	}
	if !ok || retried.Task.State != StateReady {
		t.Fatalf("expected retry to reset state, ok=%v state=%s", ok, retried.Task.State)
	}
}

func TestServiceRuntimeHandlesFallbackAndConflictDowngrade(t *testing.T) {
	ctx := context.Background()
	store, err := sqlitestore.New(ctx, filepath.Join(t.TempDir(), "runtime.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer func() { _ = store.Close() }()

	adapter := &scriptedAdapter{
		meta: provider.Provider{
			Key:                "runtime_fake",
			DisplayName:        "Runtime Fake",
			ProtocolGroup:      "fake",
			AuthModes:          []string{"manual_token"},
			FastUploadInputs:   []string{"md5", "size"},
			FallbackModes:      []string{"download_upload"},
			ConflictPolicies:   []provider.ConflictPolicy{provider.ConflictPolicyOverwriteExisting, provider.ConflictPolicyAutoRenameNew},
			SupportsOverwrite:  false,
			SupportsAutoRename: true,
			OverwriteBehavior:  "downgrade_to_auto_rename",
			Status:             "planned",
		},
		capability: provider.CapabilitySet{
			SupportsAuthValidation: true,
			SupportsUpload:         true,
		},
		uploadFunc: func(req provider.UploadRequest) provider.UploadResult {
			if req.Strategy == string(planner.StrategyFastUpload) {
				return provider.UploadResult{
					OperationResult: provider.OperationResult{
						Status:  "hash_miss",
						Message: "hash miss",
						Mode:    "fake_fast",
					},
				}
			}
			return provider.UploadResult{
				OperationResult: provider.OperationResult{
					OK:      true,
					Status:  "ok",
					Message: "fallback upload ok",
					Mode:    "fake_binary",
				},
			}
		},
	}

	registry := provider.NewRegistry(adapter)
	authSvc := auth.NewService(store, registry)
	svc := NewService(store, registry, authSvc)
	profile, err := authSvc.CreateProfile(ctx, auth.CreateProfileInput{
		ProviderKey: "runtime_fake",
		AuthMode:    "manual_token",
		DisplayName: "runtime fake",
		Token:       "token-1",
	})
	if err != nil {
		t.Fatalf("CreateProfile() error = %v", err)
	}

	localFile := filepath.Join(t.TempDir(), "source.bin")
	if err := os.WriteFile(localFile, []byte("hello"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	detail, err := svc.Create(ctx, CreateRequest{
		SourceProvider:  "guangya",
		TargetProvider:  "runtime_fake",
		TargetProfileID: profile.ID,
		ThresholdMB:     0,
		RiskMode:        planner.RiskModeFast,
		ConflictPolicy:  provider.ConflictPolicyOverwriteExisting,
		Entries: []planner.SourceEntry{
			{Path: "/a.bin", Size: 1024, MD5: "abc", LocalPath: localFile},
		},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	running, ok, err := svc.Run(ctx, detail.Task.ID)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !ok {
		t.Fatal("expected task to exist")
	}
	if running.Task.State != StateCompleted {
		t.Fatalf("expected completed, got %s", running.Task.State)
	}
	if len(running.Results) != 1 {
		t.Fatalf("expected one result, got %d", len(running.Results))
	}
	if running.Results[0].Status != "done" {
		t.Fatalf("expected done result, got %s", running.Results[0].Status)
	}
	if running.Results[0].ConflictAction != "downgrade_to_auto_rename" {
		t.Fatalf("expected conflict downgrade, got %s", running.Results[0].ConflictAction)
	}
	if value, _ := running.Results[0].Payload["fallbackUsed"].(bool); !value {
		t.Fatalf("expected fallbackUsed payload, got %+v", running.Results[0].Payload)
	}
	riskProfile, ok := running.Results[0].Payload["riskProfile"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected result riskProfile payload, got %#v", running.Results[0].Payload["riskProfile"])
	}
	if mode, _ := riskProfile["mode"].(string); mode != string(planner.RiskModeFast) {
		t.Fatalf("expected fast risk mode, got %v", riskProfile["mode"])
	}
}

func TestServiceRuntimeHandlesPendingManualAuthExpiredRateLimitAndMissingLocalFile(t *testing.T) {
	ctx := context.Background()
	store, err := sqlitestore.New(ctx, filepath.Join(t.TempDir(), "runtime-errors.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer func() { _ = store.Close() }()

	adapter := &scriptedAdapter{
		meta: provider.Provider{
			Key:              "runtime_errors",
			DisplayName:      "Runtime Errors",
			ProtocolGroup:    "fake",
			AuthModes:        []string{"manual_token"},
			FastUploadInputs: []string{"md5", "size"},
			FallbackModes:    []string{"download_upload"},
			ConflictPolicies: []provider.ConflictPolicy{provider.ConflictPolicyAutoRenameNew},
			Status:           "planned",
		},
		capability: provider.CapabilitySet{
			SupportsAuthValidation: true,
			SupportsUpload:         true,
		},
		uploadFunc: func(req provider.UploadRequest) provider.UploadResult {
			if req.Strategy == string(planner.StrategyPendingManual) {
				return provider.UploadResult{
					OperationResult: provider.OperationResult{
						Status:  "pending_manual_requires_confirmation",
						Message: "pending manual",
						Mode:    "fake_pending",
					},
				}
			}
			switch req.Path {
			case "/auth.bin":
				return provider.UploadResult{OperationResult: provider.OperationResult{Status: "auth_expired", Message: "auth expired", Mode: "fake_auth"}}
			case "/rate.bin":
				return provider.UploadResult{OperationResult: provider.OperationResult{Status: "rate_limited", Message: "rate limited", Mode: "fake_rate"}}
			default:
				return provider.UploadResult{OperationResult: provider.OperationResult{OK: true, Status: "ok", Message: "ok", Mode: "fake_ok"}}
			}
		},
	}

	registry := provider.NewRegistry(adapter)
	authSvc := auth.NewService(store, registry)
	svc := NewService(store, registry, authSvc)
	profile, err := authSvc.CreateProfile(ctx, auth.CreateProfileInput{
		ProviderKey: "runtime_errors",
		AuthMode:    "manual_token",
		DisplayName: "runtime errors",
		Token:       "token-1",
	})
	if err != nil {
		t.Fatalf("CreateProfile() error = %v", err)
	}

	detail, err := svc.Create(ctx, CreateRequest{
		SourceProvider:  "guangya",
		TargetProvider:  "runtime_errors",
		TargetProfileID: profile.ID,
		ThresholdMB:     1,
		Entries: []planner.SourceEntry{
			{Path: "/pending.bin", Size: 10 * 1024 * 1024},
			{Path: "/auth.bin", Size: 1024, MD5: "abc", LocalPath: filepath.Join(t.TempDir(), "auth.bin")},
			{Path: "/rate.bin", Size: 1024, MD5: "abc", LocalPath: filepath.Join(t.TempDir(), "rate.bin")},
			{Path: "/missing.bin", Size: 512, LocalPath: filepath.Join(t.TempDir(), "missing.bin")},
		},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	running, ok, err := svc.Run(ctx, detail.Task.ID)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !ok {
		t.Fatal("expected task to exist")
	}
	if running.Task.State != StateCompletedWithErrors {
		t.Fatalf("expected completed_with_errors, got %s", running.Task.State)
	}
	if len(running.Results) != 4 {
		t.Fatalf("expected 4 results, got %d", len(running.Results))
	}

	assertResultStatus := func(path string, wantStatus string, wantProviderStatus string) {
		t.Helper()
		for idx, item := range running.Plan.Items {
			if item.Path != path {
				continue
			}
			got := running.Results[idx]
			if got.Status != wantStatus {
				t.Fatalf("path %s expected result status %s, got %s", path, wantStatus, got.Status)
			}
			if providerStatus, _ := got.Payload["providerStatus"].(string); providerStatus != wantProviderStatus {
				t.Fatalf("path %s expected providerStatus %s, got %s", path, wantProviderStatus, providerStatus)
			}
			return
		}
		t.Fatalf("path %s not found", path)
	}

	assertResultStatus("/pending.bin", "failed", "pending_manual_requires_confirmation")
	assertResultStatus("/auth.bin", "failed", "auth_expired")
	assertResultStatus("/rate.bin", "failed", "rate_limited")
	assertResultStatus("/missing.bin", "failed", "local_file_missing")
}

type scriptedAdapter struct {
	meta       provider.Provider
	capability provider.CapabilitySet
	uploadFunc func(req provider.UploadRequest) provider.UploadResult
}

func (a *scriptedAdapter) Meta() provider.Provider {
	return a.meta
}

func (a *scriptedAdapter) Capabilities() provider.CapabilitySet {
	return a.capability
}

func (a *scriptedAdapter) ValidateAuth(profile provider.AuthProfile) provider.OperationResult {
	return provider.OperationResult{OK: true, Status: "verified", Message: "ok", Mode: "scripted"}
}

func (a *scriptedAdapter) List(req provider.ListRequest) provider.ListResult {
	return provider.ListResult{OperationResult: provider.OperationResult{OK: true, Status: "ok", Message: "ok", Mode: "scripted"}}
}

func (a *scriptedAdapter) Metadata(req provider.MetadataRequest) provider.MetadataResult {
	return provider.MetadataResult{OperationResult: provider.OperationResult{OK: true, Status: "ok", Message: "ok", Mode: "scripted"}}
}

func (a *scriptedAdapter) CreateDir(req provider.CreateDirRequest) provider.OperationResult {
	return provider.OperationResult{OK: true, Status: "ok", Message: "ok", Mode: "scripted"}
}

func (a *scriptedAdapter) FastUploadCheck(req provider.FastUploadCheckRequest) provider.FastUploadCheckResult {
	return provider.FastUploadCheckResult{OperationResult: provider.OperationResult{OK: true, Status: "ok", Message: "ok", Mode: "scripted"}, Candidate: true}
}

func (a *scriptedAdapter) Upload(req provider.UploadRequest) provider.UploadResult {
	if a.uploadFunc == nil {
		return provider.UploadResult{OperationResult: provider.OperationResult{OK: true, Status: "ok", Message: "ok", Mode: "scripted"}}
	}
	return a.uploadFunc(req)
}
