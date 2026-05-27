package task

import (
	"context"
	"os"
	"path/filepath"
	"strings"
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
	if mode, _ := detail.Plan.Metadata["executionMode"].(planner.ExecutionMode); mode != planner.ExecutionModeLeafFirstLazy {
		t.Fatalf("expected default execution mode leaf_first_lazy, got %v", detail.Plan.Metadata["executionMode"])
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
	if mode, _ := running.Results[0].Payload["executionMode"].(string); mode != string(planner.ExecutionModeLeafFirstLazy) {
		t.Fatalf("expected result execution mode leaf_first_lazy, got %v", running.Results[0].Payload["executionMode"])
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

func TestServiceRunSkipsAlreadySyncedTargetFile(t *testing.T) {
	ctx := context.Background()
	store, err := sqlitestore.New(ctx, filepath.Join(t.TempDir(), "runtime-skip.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer func() { _ = store.Close() }()

	uploadCalls := 0
	adapter := &scriptedAdapter{
		meta: provider.Provider{
			Key:              "runtime_skip",
			DisplayName:      "Runtime Skip",
			ProtocolGroup:    "fake",
			AuthModes:        []string{"manual_token"},
			FastUploadInputs: []string{"md5", "size"},
			FallbackModes:    []string{"download_upload"},
			Status:           "planned",
		},
		capability: provider.CapabilitySet{
			SupportsAuthValidation: true,
			SupportsMetadata:       true,
			SupportsUpload:         true,
		},
		metadataFunc: func(req provider.MetadataRequest) provider.MetadataResult {
			return provider.MetadataResult{
				OperationResult: provider.OperationResult{
					OK:      true,
					Status:  "exists",
					Message: "exists",
					Mode:    "scripted_metadata",
				},
				Entry: map[string]interface{}{
					"exists": true,
					"path":   req.Path,
					"size":   int64(1024),
					"md5":    "abc",
				},
			}
		},
		uploadFunc: func(req provider.UploadRequest) provider.UploadResult {
			uploadCalls++
			return provider.UploadResult{
				OperationResult: provider.OperationResult{
					OK:      true,
					Status:  "ok",
					Message: "uploaded",
					Mode:    "scripted_upload",
				},
			}
		},
	}

	registry := provider.NewRegistry(adapter)
	authSvc := auth.NewService(store, registry)
	svc := NewService(store, registry, authSvc)
	profile, err := authSvc.CreateProfile(ctx, auth.CreateProfileInput{
		ProviderKey: "runtime_skip",
		AuthMode:    "manual_token",
		DisplayName: "runtime skip",
		Token:       "token-1",
	})
	if err != nil {
		t.Fatalf("CreateProfile() error = %v", err)
	}

	detail, err := svc.Create(ctx, CreateRequest{
		SourceProvider:  "guangya",
		TargetProvider:  "runtime_skip",
		TargetProfileID: profile.ID,
		ThresholdMB:     10,
		Entries: []planner.SourceEntry{
			{Path: "/same.bin", Size: 1024, MD5: "abc"},
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
	if uploadCalls != 0 {
		t.Fatalf("expected upload not to be called, got %d", uploadCalls)
	}
	if got := len(running.Results); got != 1 {
		t.Fatalf("expected 1 result, got %d", got)
	}
	if got := running.Results[0].Status; got != "skipped" {
		t.Fatalf("expected skipped result, got %s", got)
	}
	if got, _ := running.Results[0].Payload["syncDecision"].(string); got != "skip" {
		t.Fatalf("expected syncDecision skip, got %v", running.Results[0].Payload["syncDecision"])
	}
	if running.Runtime.SkippedCount != 1 {
		t.Fatalf("expected runtime skipped count 1, got %d", running.Runtime.SkippedCount)
	}
	dirIndex := findDirectoryState(running.Runtime.DirectoryStates, "/")
	if dirIndex >= 0 && running.Runtime.DirectoryStates[dirIndex].SkippedItems != 1 {
		t.Fatalf("expected root directory skipped count 1, got %d", running.Runtime.DirectoryStates[dirIndex].SkippedItems)
	}

	evidence, err := svc.RuntimeEvidence(ctx)
	if err != nil {
		t.Fatalf("RuntimeEvidence() error = %v", err)
	}
	if evidence.SkippedResultCount != 1 {
		t.Fatalf("expected skipped result count 1, got %d", evidence.SkippedResultCount)
	}
}

func TestServiceRunUploadsWhenTargetFingerprintDiffers(t *testing.T) {
	ctx := context.Background()
	store, err := sqlitestore.New(ctx, filepath.Join(t.TempDir(), "runtime-overwrite.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer func() { _ = store.Close() }()

	uploadCalls := 0
	adapter := &scriptedAdapter{
		meta: provider.Provider{
			Key:              "runtime_overwrite",
			DisplayName:      "Runtime Overwrite",
			ProtocolGroup:    "fake",
			AuthModes:        []string{"manual_token"},
			FastUploadInputs: []string{"md5", "size"},
			FallbackModes:    []string{"download_upload"},
			Status:           "planned",
		},
		capability: provider.CapabilitySet{
			SupportsAuthValidation: true,
			SupportsMetadata:       true,
			SupportsUpload:         true,
		},
		metadataFunc: func(req provider.MetadataRequest) provider.MetadataResult {
			return provider.MetadataResult{
				OperationResult: provider.OperationResult{
					OK:      true,
					Status:  "exists",
					Message: "exists",
					Mode:    "scripted_metadata",
				},
				Entry: map[string]interface{}{
					"exists": true,
					"path":   req.Path,
					"size":   int64(1024),
					"md5":    "different-md5",
				},
			}
		},
		uploadFunc: func(req provider.UploadRequest) provider.UploadResult {
			uploadCalls++
			return provider.UploadResult{
				OperationResult: provider.OperationResult{
					OK:      true,
					Status:  "ok",
					Message: "uploaded",
					Mode:    "scripted_upload",
				},
			}
		},
	}

	registry := provider.NewRegistry(adapter)
	authSvc := auth.NewService(store, registry)
	svc := NewService(store, registry, authSvc)
	profile, err := authSvc.CreateProfile(ctx, auth.CreateProfileInput{
		ProviderKey: "runtime_overwrite",
		AuthMode:    "manual_token",
		DisplayName: "runtime overwrite",
		Token:       "token-1",
	})
	if err != nil {
		t.Fatalf("CreateProfile() error = %v", err)
	}

	detail, err := svc.Create(ctx, CreateRequest{
		SourceProvider:  "guangya",
		TargetProvider:  "runtime_overwrite",
		TargetProfileID: profile.ID,
		ThresholdMB:     10,
		Entries: []planner.SourceEntry{
			{Path: "/changed.bin", Size: 1024, MD5: "abc"},
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
	if uploadCalls != 1 {
		t.Fatalf("expected upload to be called once, got %d", uploadCalls)
	}
	if got, _ := running.Results[0].Payload["syncDecision"].(string); got != "overwrite" {
		t.Fatalf("expected syncDecision overwrite, got %v", running.Results[0].Payload["syncDecision"])
	}
	if got := running.Results[0].Status; got != "done" {
		t.Fatalf("expected done result, got %s", got)
	}
	if got := running.Runtime.SkippedCount; got != 0 {
		t.Fatalf("expected skipped count 0, got %d", got)
	}
}

func TestServiceRunLazilyScansLeafFirstByRootSubtree(t *testing.T) {
	ctx := context.Background()
	store, err := sqlitestore.New(ctx, filepath.Join(t.TempDir(), "lazy-scan.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer func() { _ = store.Close() }()

	sourceAdapter := &scriptedAdapter{
		meta: provider.Provider{
			Key:           "source_tree",
			DisplayName:   "Source Tree",
			ProtocolGroup: "fake",
			AuthModes:     []string{"manual_token"},
			Status:        "planned",
		},
		capability: provider.CapabilitySet{
			SupportsAuthValidation: true,
			SupportsList:           true,
		},
		listFunc: func(req provider.ListRequest) provider.ListResult {
			switch req.Path {
			case "/1":
				return scriptedListResult(
					dirItem("/1/11"),
					dirItem("/1/12"),
				)
			case "/1/11":
				return scriptedListResult(
					dirItem("/1/11/111"),
					dirItem("/1/11/112"),
				)
			case "/1/11/111":
				return scriptedListResult(fileItem("/1/11/111/a.bin", 10))
			case "/1/11/112":
				return scriptedListResult(fileItem("/1/11/112/b.bin", 20))
			case "/1/12":
				return scriptedListResult(
					dirItem("/1/12/121"),
					dirItem("/1/12/123"),
				)
			case "/1/12/121":
				return scriptedListResult(fileItem("/1/12/121/c.bin", 30))
			case "/1/12/123":
				return scriptedListResult(fileItem("/1/12/123/d.bin", 40))
			case "/2":
				return scriptedListResult(dirItem("/2/22"))
			case "/2/22":
				return scriptedListResult(
					dirItem("/2/22/221"),
					dirItem("/2/22/222"),
				)
			case "/2/22/221":
				return scriptedListResult(fileItem("/2/22/221/e.bin", 50))
			case "/2/22/222":
				return scriptedListResult(fileItem("/2/22/222/f.bin", 60))
			case "/3":
				return scriptedListResult(fileItem("/3/g.bin", 70))
			default:
				return scriptedListResult()
			}
		},
	}
	targetUploads := make([]string, 0)
	targetAdapter := &scriptedAdapter{
		meta: provider.Provider{
			Key:              "target_tree",
			DisplayName:      "Target Tree",
			ProtocolGroup:    "fake",
			AuthModes:        []string{"manual_token"},
			FastUploadInputs: []string{"md5", "size"},
			FallbackModes:    []string{"download_upload"},
			Status:           "planned",
		},
		capability: provider.CapabilitySet{
			SupportsAuthValidation: true,
			SupportsUpload:         true,
		},
		uploadFunc: func(req provider.UploadRequest) provider.UploadResult {
			targetUploads = append(targetUploads, req.Path)
			return provider.UploadResult{
				OperationResult: provider.OperationResult{
					OK:      true,
					Status:  "ok",
					Message: "uploaded",
					Mode:    "scripted_upload",
				},
			}
		},
	}

	registry := provider.NewRegistry(sourceAdapter, targetAdapter)
	authSvc := auth.NewService(store, registry)
	svc := NewService(store, registry, authSvc)
	sourceProfile, err := authSvc.CreateProfile(ctx, auth.CreateProfileInput{
		ProviderKey: "source_tree",
		AuthMode:    "manual_token",
		DisplayName: "source",
		Token:       "source-token",
	})
	if err != nil {
		t.Fatalf("CreateProfile(source) error = %v", err)
	}
	targetProfile, err := authSvc.CreateProfile(ctx, auth.CreateProfileInput{
		ProviderKey: "target_tree",
		AuthMode:    "manual_token",
		DisplayName: "target",
		Token:       "target-token",
	})
	if err != nil {
		t.Fatalf("CreateProfile(target) error = %v", err)
	}

	detail, err := svc.Create(ctx, CreateRequest{
		SourceProvider:  "source_tree",
		SourceProfileID: sourceProfile.ID,
		TargetProvider:  "target_tree",
		TargetProfileID: targetProfile.ID,
		ThresholdMB:     1,
		SelectedRoots:   []string{"/1", "/2", "/3"},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if len(detail.SourceEntries) != 0 {
		t.Fatalf("expected no eager source entries at create time, got %d", len(detail.SourceEntries))
	}

	running, ok, err := svc.Run(ctx, detail.Task.ID)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !ok {
		t.Fatal("expected task to exist")
	}

	wantListOrder := []string{"/1", "/1/11", "/1/11/111", "/1/11/112", "/1/12", "/1/12/121", "/1/12/123", "/2", "/2/22", "/2/22/221", "/2/22/222", "/3"}
	if len(sourceAdapter.listCalls) != len(wantListOrder) {
		t.Fatalf("expected %d list calls, got %d: %#v", len(wantListOrder), len(sourceAdapter.listCalls), sourceAdapter.listCalls)
	}
	for i, want := range wantListOrder {
		if sourceAdapter.listCalls[i] != want {
			t.Fatalf("list call %d expected %s, got %s", i, want, sourceAdapter.listCalls[i])
		}
	}

	wantUploadOrder := []string{
		"/1/11/111/a.bin",
		"/1/11/112/b.bin",
		"/1/12/121/c.bin",
		"/1/12/123/d.bin",
		"/2/22/221/e.bin",
		"/2/22/222/f.bin",
		"/3/g.bin",
	}
	if len(targetUploads) != len(wantUploadOrder) {
		t.Fatalf("expected %d uploads, got %d: %#v", len(wantUploadOrder), len(targetUploads), targetUploads)
	}
	for i, want := range wantUploadOrder {
		if targetUploads[i] != want {
			t.Fatalf("upload order %d expected %s, got %s", i, want, targetUploads[i])
		}
	}

	if got, _ := running.Plan.Metadata["scanMode"].(string); got != "lazy_leaf_first" {
		t.Fatalf("expected lazy scan mode, got %v", running.Plan.Metadata["scanMode"])
	}
	assertExecutionModeValue(t, running.Plan.Metadata["executionMode"], planner.ExecutionModeLeafFirstLazy)
	if running.Runtime.ExecutionState != "completed" {
		t.Fatalf("expected runtime completed, got %s", running.Runtime.ExecutionState)
	}
	if running.Runtime.ProcessedCount != len(wantUploadOrder) {
		t.Fatalf("expected processed count %d, got %d", len(wantUploadOrder), running.Runtime.ProcessedCount)
	}
	rootStateIndex := findDirectoryState(running.Runtime.DirectoryStates, "/1")
	if rootStateIndex < 0 {
		t.Fatal("expected runtime directory state for /1")
	}
	if got := running.Runtime.DirectoryStates[rootStateIndex].Status; got != "completed" {
		t.Fatalf("expected root /1 completed, got %s", got)
	}
	switch trace := running.Plan.Metadata["scanTrace"].(type) {
	case []string:
		if len(trace) != len(wantListOrder) {
			t.Fatalf("expected scanTrace len %d, got %d", len(wantListOrder), len(trace))
		}
	case []interface{}:
		if len(trace) != len(wantListOrder) {
			t.Fatalf("expected scanTrace len %d, got %d", len(wantListOrder), len(trace))
		}
	default:
		t.Fatalf("expected scanTrace in metadata, got %#v", running.Plan.Metadata["scanTrace"])
	}
}

func TestServiceResumeContinuesCurrentSubtreeFromCheckpoint(t *testing.T) {
	ctx := context.Background()
	store, err := sqlitestore.New(ctx, filepath.Join(t.TempDir(), "resume.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer func() { _ = store.Close() }()

	sourceAdapter := &scriptedAdapter{
		meta: provider.Provider{
			Key:           "resume_source",
			DisplayName:   "Resume Source",
			ProtocolGroup: "fake",
			AuthModes:     []string{"manual_token"},
			Status:        "planned",
		},
		capability: provider.CapabilitySet{
			SupportsAuthValidation: true,
			SupportsList:           true,
		},
		listFunc: func(req provider.ListRequest) provider.ListResult {
			switch req.Path {
			case "/1":
				return scriptedListResult(dirItem("/1/11"), dirItem("/1/12"))
			case "/1/11":
				return scriptedListResult(fileItem("/1/11/a.bin", 10), fileItem("/1/11/b.bin", 20))
			case "/1/12":
				return scriptedListResult(fileItem("/1/12/c.bin", 30))
			case "/2":
				return scriptedListResult(fileItem("/2/d.bin", 40))
			default:
				return scriptedListResult()
			}
		},
	}
	targetUploads := make([]string, 0)
	targetAdapter := &scriptedAdapter{
		meta: provider.Provider{
			Key:              "resume_target",
			DisplayName:      "Resume Target",
			ProtocolGroup:    "fake",
			AuthModes:        []string{"manual_token"},
			FastUploadInputs: []string{"md5", "size"},
			FallbackModes:    []string{"download_upload"},
			Status:           "planned",
		},
		capability: provider.CapabilitySet{
			SupportsAuthValidation: true,
			SupportsUpload:         true,
		},
		uploadFunc: func(req provider.UploadRequest) provider.UploadResult {
			targetUploads = append(targetUploads, req.Path)
			return provider.UploadResult{
				OperationResult: provider.OperationResult{
					OK:      true,
					Status:  "ok",
					Message: "uploaded",
					Mode:    "scripted_upload",
				},
			}
		},
	}

	registry := provider.NewRegistry(sourceAdapter, targetAdapter)
	authSvc := auth.NewService(store, registry)
	svc := NewService(store, registry, authSvc)
	sourceProfile, err := authSvc.CreateProfile(ctx, auth.CreateProfileInput{
		ProviderKey: "resume_source",
		AuthMode:    "manual_token",
		DisplayName: "resume source",
		Token:       "source-token",
	})
	if err != nil {
		t.Fatalf("CreateProfile(source) error = %v", err)
	}
	targetProfile, err := authSvc.CreateProfile(ctx, auth.CreateProfileInput{
		ProviderKey: "resume_target",
		AuthMode:    "manual_token",
		DisplayName: "resume target",
		Token:       "target-token",
	})
	if err != nil {
		t.Fatalf("CreateProfile(target) error = %v", err)
	}

	created, err := svc.Create(ctx, CreateRequest{
		SourceProvider:  "resume_source",
		SourceProfileID: sourceProfile.ID,
		TargetProvider:  "resume_target",
		TargetProfileID: targetProfile.ID,
		SelectedRoots:   []string{"/1", "/2"},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if err := svc.materializeTaskEntriesIfNeeded(ctx, &created); err != nil {
		t.Fatalf("materializeTaskEntriesIfNeeded() error = %v", err)
	}

	firstResult := Result{
		ID:        "resume-result-1",
		TaskID:    created.Task.ID,
		ItemID:    created.Items[0].ID,
		Status:    "done",
		Mode:      "scripted_upload",
		Message:   "uploaded",
		CreatedAt: created.Task.CreatedAt,
		Payload: map[string]interface{}{
			"path":          created.Plan.Items[0].Path,
			"executionMode": string(planner.ExecutionModeLeafFirstLazy),
		},
	}
	created.Results = []Result{firstResult}
	created.Task.State = StatePaused
	created.Task.UpdatedAt = created.Task.CreatedAt
	created.Runtime = initializeRuntimeState(created.Plan)
	updateRuntimeAfterItem(&created, created.Plan.Items[0].Path, firstResult)
	created.Runtime.ExecutionState = "paused"
	created.Runtime.CurrentRoot = "/1"
	created.Runtime.CurrentDirectory = "/1/11"
	if err := replaceTaskDetailAndResults(ctx, store, created); err != nil {
		t.Fatalf("replaceTaskDetailAndResults() error = %v", err)
	}

	resumed, ok, err := svc.Resume(ctx, created.Task.ID)
	if err != nil {
		t.Fatalf("Resume() error = %v", err)
	}
	if !ok || resumed.Task.State != StateReady {
		t.Fatalf("expected ready after resume, ok=%v state=%s", ok, resumed.Task.State)
	}

	running, ok, err := svc.Run(ctx, created.Task.ID)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !ok {
		t.Fatal("expected resumed task to exist")
	}
	wantUploads := []string{"/1/11/b.bin", "/1/12/c.bin", "/2/d.bin"}
	if len(targetUploads) != len(wantUploads) {
		t.Fatalf("expected %d resumed uploads, got %d: %#v", len(wantUploads), len(targetUploads), targetUploads)
	}
	for i, want := range wantUploads {
		if targetUploads[i] != want {
			t.Fatalf("resumed upload %d expected %s, got %s", i, want, targetUploads[i])
		}
	}
	if running.Runtime.ProcessedCount != 4 {
		t.Fatalf("expected processed count 4 after resume, got %d", running.Runtime.ProcessedCount)
	}
	if running.Runtime.LastCompletedPath != "/2/d.bin" {
		t.Fatalf("expected last completed /2/d.bin, got %s", running.Runtime.LastCompletedPath)
	}
	dirStateIndex := findDirectoryState(running.Runtime.DirectoryStates, "/1/11")
	if dirStateIndex < 0 {
		t.Fatal("expected runtime directory state for /1/11")
	}
	if got := running.Runtime.DirectoryStates[dirStateIndex].Status; got != "completed" {
		t.Fatalf("expected /1/11 completed after resume, got %s", got)
	}
}

func TestServiceRunSupportsPreScanFlatExecutionMode(t *testing.T) {
	ctx := context.Background()
	store, err := sqlitestore.New(ctx, filepath.Join(t.TempDir(), "pre-scan.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer func() { _ = store.Close() }()

	sourceAdapter := &scriptedAdapter{
		meta: provider.Provider{
			Key:           "source_tree_flat",
			DisplayName:   "Source Tree Flat",
			ProtocolGroup: "fake",
			AuthModes:     []string{"manual_token"},
			Status:        "planned",
		},
		capability: provider.CapabilitySet{
			SupportsAuthValidation: true,
			SupportsList:           true,
		},
		listFunc: func(req provider.ListRequest) provider.ListResult {
			switch req.Path {
			case "/root":
				return scriptedListResult(
					fileItem("/root/a.bin", 10),
					dirItem("/root/sub"),
				)
			case "/root/sub":
				return scriptedListResult(
					fileItem("/root/sub/b.bin", 20),
					fileItem("/root/sub/c.bin", 30),
				)
			default:
				return scriptedListResult()
			}
		},
	}
	targetUploads := make([]string, 0)
	targetAdapter := &scriptedAdapter{
		meta: provider.Provider{
			Key:              "target_tree_flat",
			DisplayName:      "Target Tree Flat",
			ProtocolGroup:    "fake",
			AuthModes:        []string{"manual_token"},
			FastUploadInputs: []string{"md5", "size"},
			FallbackModes:    []string{"download_upload"},
			Status:           "planned",
		},
		capability: provider.CapabilitySet{
			SupportsAuthValidation: true,
			SupportsUpload:         true,
		},
		uploadFunc: func(req provider.UploadRequest) provider.UploadResult {
			targetUploads = append(targetUploads, req.Path)
			return provider.UploadResult{
				OperationResult: provider.OperationResult{
					OK:      true,
					Status:  "ok",
					Message: "uploaded",
					Mode:    "scripted_upload",
				},
			}
		},
	}

	registry := provider.NewRegistry(sourceAdapter, targetAdapter)
	authSvc := auth.NewService(store, registry)
	svc := NewService(store, registry, authSvc)
	sourceProfile, err := authSvc.CreateProfile(ctx, auth.CreateProfileInput{
		ProviderKey: "source_tree_flat",
		AuthMode:    "manual_token",
		DisplayName: "source",
		Token:       "source-token",
	})
	if err != nil {
		t.Fatalf("CreateProfile(source) error = %v", err)
	}
	targetProfile, err := authSvc.CreateProfile(ctx, auth.CreateProfileInput{
		ProviderKey: "target_tree_flat",
		AuthMode:    "manual_token",
		DisplayName: "target",
		Token:       "target-token",
	})
	if err != nil {
		t.Fatalf("CreateProfile(target) error = %v", err)
	}

	detail, err := svc.Create(ctx, CreateRequest{
		SourceProvider:  "source_tree_flat",
		SourceProfileID: sourceProfile.ID,
		TargetProvider:  "target_tree_flat",
		TargetProfileID: targetProfile.ID,
		ExecutionMode:   planner.ExecutionModePreScanFlat,
		SelectedRoots:   []string{"/root"},
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
	if got, _ := running.Plan.Metadata["scanMode"].(string); got != "pre_scan_flat" {
		t.Fatalf("expected pre_scan_flat scan mode, got %v", running.Plan.Metadata["scanMode"])
	}
	assertExecutionModeValue(t, running.Plan.Metadata["executionMode"], planner.ExecutionModePreScanFlat)
	wantUploads := []string{"/root/a.bin", "/root/sub/b.bin", "/root/sub/c.bin"}
	if len(targetUploads) != len(wantUploads) {
		t.Fatalf("expected %d uploads, got %d: %#v", len(wantUploads), len(targetUploads), targetUploads)
	}
	for i, want := range wantUploads {
		if targetUploads[i] != want {
			t.Fatalf("pre-scan upload order %d expected %s, got %s", i, want, targetUploads[i])
		}
	}
}

type scriptedAdapter struct {
	meta         provider.Provider
	capability   provider.CapabilitySet
	listFunc     func(req provider.ListRequest) provider.ListResult
	metadataFunc func(req provider.MetadataRequest) provider.MetadataResult
	uploadFunc   func(req provider.UploadRequest) provider.UploadResult
	listCalls    []string
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
	a.listCalls = append(a.listCalls, req.Path)
	if a.listFunc != nil {
		return a.listFunc(req)
	}
	return provider.ListResult{OperationResult: provider.OperationResult{OK: true, Status: "ok", Message: "ok", Mode: "scripted"}}
}

func (a *scriptedAdapter) Metadata(req provider.MetadataRequest) provider.MetadataResult {
	if a.metadataFunc != nil {
		return a.metadataFunc(req)
	}
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

func scriptedListResult(items ...map[string]interface{}) provider.ListResult {
	return provider.ListResult{
		OperationResult: provider.OperationResult{
			OK:      true,
			Status:  "ok",
			Message: "ok",
			Mode:    "scripted_list",
		},
		Items: items,
	}
}

func dirItem(path string) map[string]interface{} {
	return map[string]interface{}{
		"path":  path,
		"name":  inferNameForTest(path),
		"isDir": true,
	}
}

func fileItem(path string, size int64) map[string]interface{} {
	return map[string]interface{}{
		"path":  path,
		"name":  inferNameForTest(path),
		"size":  size,
		"isDir": false,
		"md5":   "md5-" + inferNameForTest(path),
	}
}

func inferNameForTest(path string) string {
	path = strings.ReplaceAll(path, "\\", "/")
	index := strings.LastIndex(path, "/")
	if index >= 0 && index < len(path)-1 {
		return path[index+1:]
	}
	return path
}

func assertExecutionModeValue(t *testing.T, raw interface{}, want planner.ExecutionMode) {
	t.Helper()
	switch value := raw.(type) {
	case planner.ExecutionMode:
		if value != want {
			t.Fatalf("expected execution mode %s, got %s", want, value)
		}
	case string:
		if value != string(want) {
			t.Fatalf("expected execution mode %s, got %s", want, value)
		}
	default:
		t.Fatalf("expected execution mode %s, got %#v", want, raw)
	}
}
