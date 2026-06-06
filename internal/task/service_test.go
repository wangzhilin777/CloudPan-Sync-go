package task

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

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
	if running.Task.State != StateBlocked {
		t.Fatalf("expected blocked, got %s", running.Task.State)
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
	if len(retried.Plan.Items) != 1 || retried.Plan.Items[0].Path != "/b.bin" {
		t.Fatalf("expected retry to narrow plan to pending item /b.bin, got %#v", retried.Plan.Items)
	}
	if len(retried.SourceEntries) != 1 || retried.SourceEntries[0].Path != "/b.bin" {
		t.Fatalf("expected retry to narrow source entries to /b.bin, got %#v", retried.SourceEntries)
	}
	if retryPendingOnly, _ := retried.Plan.Metadata["retryPendingOnly"].(bool); !retryPendingOnly {
		t.Fatalf("expected retryPendingOnly metadata true, got %#v", retried.Plan.Metadata["retryPendingOnly"])
	}
}

func TestServiceCreateCarriesSourceDeletionRecordsIntoRuntimeAndEvidence(t *testing.T) {
	ctx := context.Background()
	store, err := sqlitestore.New(ctx, filepath.Join(t.TempDir(), "source-deletion.db"))
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
		DisplayName: "123 deletion smoke",
		Token:       "token-delete",
	})
	if err != nil {
		t.Fatalf("CreateProfile() error = %v", err)
	}

	detail, err := svc.Create(ctx, CreateRequest{
		SourceProvider:     "guangya",
		TargetProvider:     "123_open",
		TargetProfileID:    profile.ID,
		ThresholdMB:        10,
		SourceDeletePolicy: planner.SourceDeletePolicyRecordOnly,
		SelectedRoots:      []string{"/demo"},
		Entries: []planner.SourceEntry{
			{Path: "/demo/a.bin", Size: 1024, MD5: "md5-a"},
			{Path: "/demo/deleted.bin", Deleted: true, DeletedAt: "2026-05-29T10:00:00Z", DeleteReason: "source_removed"},
		},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if got := detail.Runtime.SourceDeletionCount; got != 1 {
		t.Fatalf("expected runtime source deletion count 1, got %d", got)
	}
	if len(detail.Runtime.SourceDeletionRecords) != 1 {
		t.Fatalf("expected 1 runtime source deletion record, got %#v", detail.Runtime.SourceDeletionRecords)
	}
	if got, _ := detail.Plan.Metadata["sourceDeletePolicy"].(planner.SourceDeletePolicy); got != planner.SourceDeletePolicyRecordOnly {
		t.Fatalf("expected sourceDeletePolicy record_only, got %#v", detail.Plan.Metadata["sourceDeletePolicy"])
	}
	if got := detail.Runtime.SourceDeletionRecords[0].Path; got != "/demo/deleted.bin" {
		t.Fatalf("expected source deletion record path /demo/deleted.bin, got %s", got)
	}

	evidence, err := svc.RuntimeEvidence(ctx)
	if err != nil {
		t.Fatalf("RuntimeEvidence() error = %v", err)
	}
	if got := evidence.SourceDeletionCount; got != 1 {
		t.Fatalf("expected evidence source deletion count 1, got %d", got)
	}
}

func TestServiceCreateAppliesTargetProfileRiskDefaults(t *testing.T) {
	ctx := context.Background()
	store, err := sqlitestore.New(ctx, filepath.Join(t.TempDir(), "profile-risk-defaults.db"))
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
		DisplayName: "123 profile defaults",
		Token:       "token-profile-defaults",
		Extra: map[string]string{
			"riskDefaults": "{\"requestIntervalMs\":1666,\"directoryIntervalMs\":2888,\"retryLimit\":4,\"riskKeywords\":[\"profile_limit\"]}",
		},
	})
	if err != nil {
		t.Fatalf("CreateProfile() error = %v", err)
	}

	detail, err := svc.Create(ctx, CreateRequest{
		SourceProvider:  "guangya",
		TargetProvider:  "123_open",
		TargetProfileID: profile.ID,
		RiskMode:        planner.RiskModeBalanced,
		Entries: []planner.SourceEntry{
			{Path: "/a.bin", Size: 1024, MD5: "abc"},
		},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	riskProfile, ok := detail.Plan.Metadata["riskProfile"].(planner.RiskProfile)
	if !ok {
		t.Fatalf("expected riskProfile metadata, got %#v", detail.Plan.Metadata["riskProfile"])
	}
	if riskProfile.RequestIntervalMS != 1666 {
		t.Fatalf("expected applied requestIntervalMs 1666, got %+v", riskProfile)
	}
	if riskProfile.DirectoryIntervalMS != 2888 {
		t.Fatalf("expected applied directoryIntervalMs 2888, got %+v", riskProfile)
	}
	if riskProfile.RetryLimit != 4 {
		t.Fatalf("expected applied retryLimit 4, got %+v", riskProfile)
	}
	if got, _ := detail.Plan.Metadata["profileDefaultSource"].(string); got != "123 profile defaults" {
		t.Fatalf("expected profileDefaultSource 123 profile defaults, got %#v", detail.Plan.Metadata["profileDefaultSource"])
	}
	profileDefaults, ok := detail.Plan.Metadata["profileRiskDefaults"].(*planner.RiskProfileOverride)
	if !ok || profileDefaults == nil {
		t.Fatalf("expected profileRiskDefaults metadata, got %#v", detail.Plan.Metadata["profileRiskDefaults"])
	}
	if profileDefaults.RequestIntervalMS == nil || *profileDefaults.RequestIntervalMS != 1666 {
		t.Fatalf("expected profileRiskDefaults requestIntervalMs 1666, got %#v", profileDefaults)
	}
	resolution, ok := detail.Plan.Metadata["riskProfileResolution"].(planner.RiskProfileResolution)
	if !ok {
		t.Fatalf("expected riskProfileResolution metadata, got %#v", detail.Plan.Metadata["riskProfileResolution"])
	}
	if resolution.ProfileDefaultSource != "123 profile defaults" {
		t.Fatalf("expected resolution profileDefaultSource 123 profile defaults, got %+v", resolution)
	}
	if resolution.ProfileDefaultSourceKind != "auth_profile" {
		t.Fatalf("expected resolution profileDefaultSourceKind auth_profile, got %+v", resolution)
	}
	if resolution.ProfileDefaultBias != "mixed" {
		t.Fatalf("expected resolution profileDefaultBias mixed, got %+v", resolution)
	}
	if resolution.ProfileApplied.RequestIntervalMS != 1666 {
		t.Fatalf("expected resolution profileApplied requestIntervalMs 1666, got %+v", resolution.ProfileApplied)
	}
	if resolution.ProviderDisplayName == "" {
		t.Fatalf("expected resolution provider display name, got %+v", resolution)
	}
	if resolution.RecoverBudget.Reason == "" {
		t.Fatalf("expected resolution recover budget reason, got %+v", resolution.RecoverBudget)
	}
	if len(resolution.ProfileDefaultFields) != 4 {
		t.Fatalf("expected 4 profile default fields, got %#v", resolution.ProfileDefaultFields)
	}
}

func TestServiceProtocolCoverageSummary(t *testing.T) {
	ctx := context.Background()
	store, err := sqlitestore.New(ctx, filepath.Join(t.TempDir(), "protocol-coverage.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer func() { _ = store.Close() }()

	sourceAdapter := &scriptedAdapter{
		meta: provider.Provider{
			Key:           "coverage_source",
			DisplayName:   "Coverage Source",
			ProtocolGroup: "coverage_source_group",
			AuthModes:     []string{"manual_token"},
			Status:        "planned",
		},
		capability: provider.CapabilitySet{
			SupportsAuthValidation: true,
			SupportsList:           true,
		},
	}
	target123Adapter := &scriptedAdapter{
		meta: provider.Provider{
			Key:                "coverage_target_123",
			DisplayName:        "Coverage Target 123",
			ProtocolGroup:      "aliyun_123_open",
			AuthModes:          []string{"manual_token"},
			FastUploadInputs:   []string{"md5", "size"},
			FallbackModes:      []string{"download_upload"},
			ConflictPolicies:   []provider.ConflictPolicy{provider.ConflictPolicyOverwriteExisting, provider.ConflictPolicyAutoRenameNew},
			SupportsOverwrite:  true,
			SupportsAutoRename: true,
			Status:             "planned",
		},
		capability: provider.CapabilitySet{
			SupportsAuthValidation: true,
			SupportsMetadata:       true,
			SupportsFastUpload:     true,
			SupportsUpload:         true,
		},
		metadataFunc: func(req provider.MetadataRequest) provider.MetadataResult {
			return provider.MetadataResult{
				OperationResult: provider.OperationResult{
					OK:      true,
					Status:  "exists",
					Message: "scripted existing target",
					Mode:    "scripted_metadata",
				},
				Entry: map[string]interface{}{
					"exists": true,
					"path":   req.Path,
					"name":   inferNameForTest(req.Path),
				},
			}
		},
		uploadFunc: func(req provider.UploadRequest) provider.UploadResult {
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
	targetQuarkAdapter := &scriptedAdapter{
		meta: provider.Provider{
			Key:                "coverage_target_quark",
			DisplayName:        "Coverage Target Quark",
			ProtocolGroup:      "quark_uc",
			AuthModes:          []string{"manual_token"},
			FastUploadInputs:   []string{"md5", "size"},
			FallbackModes:      []string{"download_upload"},
			ConflictPolicies:   []provider.ConflictPolicy{provider.ConflictPolicyOverwriteExisting, provider.ConflictPolicyAutoRenameNew},
			SupportsOverwrite:  true,
			SupportsAutoRename: true,
			Status:             "planned",
		},
		capability: provider.CapabilitySet{
			SupportsAuthValidation: true,
			SupportsMetadata:       true,
			SupportsFastUpload:     true,
			SupportsUpload:         true,
		},
		metadataFunc: func(req provider.MetadataRequest) provider.MetadataResult {
			return provider.MetadataResult{
				OperationResult: provider.OperationResult{
					OK:      true,
					Status:  "exists",
					Message: "scripted existing target",
					Mode:    "scripted_metadata",
				},
				Entry: map[string]interface{}{
					"exists": true,
					"path":   req.Path,
					"name":   inferNameForTest(req.Path),
				},
			}
		},
		uploadFunc: func(req provider.UploadRequest) provider.UploadResult {
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

	registry := provider.NewRegistry(sourceAdapter, target123Adapter, targetQuarkAdapter)
	authSvc := auth.NewService(store, registry)
	svc := NewService(store, registry, authSvc)
	target123Profile, err := authSvc.CreateProfile(ctx, auth.CreateProfileInput{
		ProviderKey: "coverage_target_123",
		AuthMode:    "manual_token",
		DisplayName: "coverage target 123",
		Token:       "token-123",
	})
	if err != nil {
		t.Fatalf("CreateProfile(target123) error = %v", err)
	}
	targetQuarkProfile, err := authSvc.CreateProfile(ctx, auth.CreateProfileInput{
		ProviderKey: "coverage_target_quark",
		AuthMode:    "manual_token",
		DisplayName: "coverage target quark",
		Token:       "token-quark",
	})
	if err != nil {
		t.Fatalf("CreateProfile(targetQuark) error = %v", err)
	}

	firstTask, err := svc.Create(ctx, CreateRequest{
		SourceProvider:  "coverage_source",
		TargetProvider:  "coverage_target_123",
		TargetProfileID: target123Profile.ID,
		ThresholdMB:     10,
		Entries: []planner.SourceEntry{
			{Path: "/aliyun/sample.bin", Size: 1024, MD5: "md5-aliyun"},
		},
	})
	if err != nil {
		t.Fatalf("Create(first task) error = %v", err)
	}
	if _, _, err := svc.Run(ctx, firstTask.Task.ID); err != nil {
		t.Fatalf("Run(first task) error = %v", err)
	}

	secondTask, err := svc.Create(ctx, CreateRequest{
		SourceProvider:  "coverage_source",
		TargetProvider:  "coverage_target_quark",
		TargetProfileID: targetQuarkProfile.ID,
		ThresholdMB:     10,
		Entries: []planner.SourceEntry{
			{Path: "/quark/sample.bin", Size: 1024, MD5: "md5-quark"},
		},
	})
	if err != nil {
		t.Fatalf("Create(second task) error = %v", err)
	}
	if _, _, err := svc.Run(ctx, secondTask.Task.ID); err != nil {
		t.Fatalf("Run(second task) error = %v", err)
	}

	evidence, err := svc.RuntimeEvidence(ctx)
	if err != nil {
		t.Fatalf("RuntimeEvidence() error = %v", err)
	}
	if len(evidence.ProtocolCoverage) < 3 {
		t.Fatalf("expected protocol coverage for all groups, got %#v", evidence.ProtocolCoverage)
	}
	coverageByGroup := make(map[string]ProtocolCoverage, len(evidence.ProtocolCoverage))
	for _, item := range evidence.ProtocolCoverage {
		coverageByGroup[item.ProtocolGroup] = item
	}
	aliyunCoverage, ok := coverageByGroup["aliyun_123_open"]
	if !ok {
		t.Fatalf("expected aliyun_123_open coverage, got %#v", evidence.ProtocolCoverage)
	}
	if !aliyunCoverage.HasRealSuccessSample || aliyunCoverage.RealSuccessTaskCount != 1 || aliyunCoverage.ProviderCount != 1 {
		t.Fatalf("unexpected aliyun coverage: %#v", aliyunCoverage)
	}
	if aliyunCoverage.SampleProviderKey != "coverage_target_123" {
		t.Fatalf("expected aliyun sample provider, got %#v", aliyunCoverage)
	}
	quarkCoverage, ok := coverageByGroup["quark_uc"]
	if !ok {
		t.Fatalf("expected quark_uc coverage, got %#v", evidence.ProtocolCoverage)
	}
	if !quarkCoverage.HasRealSuccessSample || quarkCoverage.RealSuccessTaskCount != 1 || quarkCoverage.ProviderCount != 1 {
		t.Fatalf("unexpected quark coverage: %#v", quarkCoverage)
	}
	if quarkCoverage.SampleProviderKey != "coverage_target_quark" {
		t.Fatalf("expected quark sample provider, got %#v", quarkCoverage)
	}

	statuses, err := svc.ProviderStatuses(ctx)
	if err != nil {
		t.Fatalf("ProviderStatuses() error = %v", err)
	}
	var found123, foundQuark bool
	for _, item := range statuses {
		switch item.ProviderKey {
		case "coverage_target_123":
			found123 = true
			if item.ProtocolGroup != "aliyun_123_open" {
				t.Fatalf("expected protocol group aliyun_123_open, got %s", item.ProtocolGroup)
			}
			if item.ProtocolCoverage == nil || !item.ProtocolCoverage.HasRealSuccessSample {
				t.Fatalf("expected protocol coverage on provider status, got %#v", item.ProtocolCoverage)
			}
			if _, ok := item.SnapshotSummary["protocolCoverage"]; !ok {
				t.Fatalf("expected protocolCoverage in snapshot summary, got %#v", item.SnapshotSummary)
			}
		case "coverage_target_quark":
			foundQuark = true
			if item.ProtocolGroup != "quark_uc" {
				t.Fatalf("expected protocol group quark_uc, got %s", item.ProtocolGroup)
			}
			if item.ProtocolCoverage == nil || !item.ProtocolCoverage.HasRealSuccessSample {
				t.Fatalf("expected protocol coverage on provider status, got %#v", item.ProtocolCoverage)
			}
		}
	}
	if !found123 || !foundQuark {
		t.Fatalf("expected both target providers in status list, got %#v", statuses)
	}

	report, err := svc.EvidenceReport(ctx)
	if err != nil {
		t.Fatalf("EvidenceReport() error = %v", err)
	}
	if report.GeneratedAt == "" {
		t.Fatal("expected report generatedAt")
	}
	if report.Title != "CloudPan Sync Go 验收与样本报告" {
		t.Fatalf("expected default report title, got %s", report.Title)
	}
	if !strings.Contains(report.Markdown, "CloudPan Sync Go 验收与样本报告") {
		t.Fatalf("expected default title in report markdown, got %s", report.Markdown)
	}
	if !strings.Contains(report.Markdown, "## 代表任务样本") {
		t.Fatalf("expected sample section in report markdown, got %s", report.Markdown)
	}
	if !strings.Contains(report.Markdown, "## 自动补传公平性摘要") {
		t.Fatalf("expected auto recover fairness section in report markdown, got %s", report.Markdown)
	}
	if !strings.Contains(report.Markdown, "当前还没有可用于判断公平性的自动补传候选池样本") && !strings.Contains(report.Markdown, "当前没有自动补传候选池数据") && (!strings.Contains(report.Markdown, "公平性") || !strings.Contains(report.Markdown, "lane")) {
		t.Fatalf("expected fairness completion summary in report markdown, got %s", report.Markdown)
	}
	if !strings.Contains(report.Markdown, "Upload checkpoint 任务数") {
		t.Fatalf("expected upload checkpoint summary in report markdown, got %s", report.Markdown)
	}
	if !strings.Contains(report.Markdown, "| Mode | Tasks | Providers | Profiles | ProtocolGroups | Suggested Budgets | Sample Context | Advice |") &&
		!strings.Contains(report.Markdown, "- 当前没有自动补传候选池数据。") {
		t.Fatalf("expected auto recover fairness table or empty-state in report markdown, got %s", report.Markdown)
	}
	if !strings.Contains(report.Markdown, "RetryMode") || !strings.Contains(report.Markdown, "RetryScope") || !strings.Contains(report.Markdown, "RetryPaths") {
		t.Fatalf("expected retry evidence headers in report markdown, got %s", report.Markdown)
	}
	if len(report.Samples) == 0 {
		t.Fatal("expected report samples")
	}
	if report.Samples[0].ProviderKey == "" {
		t.Fatal("expected report sample provider key")
	}

	savedReport, err := svc.SaveEvidenceReport(ctx, "里程碑报告", "验收备注")
	if err != nil {
		t.Fatalf("SaveEvidenceReport() error = %v", err)
	}
	if savedReport.ID == "" {
		t.Fatal("expected saved report id")
	}
	if savedReport.Title != "里程碑报告" {
		t.Fatalf("expected saved report title 里程碑报告, got %s", savedReport.Title)
	}
	if savedReport.Note != "验收备注" {
		t.Fatalf("expected saved report note 验收备注, got %s", savedReport.Note)
	}

	listedReports, err := svc.ListEvidenceReports(ctx)
	if err != nil {
		t.Fatalf("ListEvidenceReports() error = %v", err)
	}
	if len(listedReports) == 0 {
		t.Fatal("expected evidence report history items")
	}
	if got := listedReports[0].ID; got != savedReport.ID {
		t.Fatalf("expected latest saved report id %s, got %s", savedReport.ID, got)
	}

	fetchedReport, ok, err := svc.GetEvidenceReport(ctx, savedReport.ID)
	if err != nil {
		t.Fatalf("GetEvidenceReport() error = %v", err)
	}
	if !ok {
		t.Fatal("expected saved report to be found")
	}
	if fetchedReport.Markdown == "" || !strings.Contains(fetchedReport.Markdown, "里程碑报告") {
		t.Fatalf("expected custom title in saved report markdown, got %s", fetchedReport.Markdown)
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
			SupportsFastUpload:     true,
			SupportsUpload:         true,
		},
		fastCheckFunc: func(req provider.FastUploadCheckRequest) provider.FastUploadCheckResult {
			return provider.FastUploadCheckResult{
				OperationResult: provider.OperationResult{
					OK:      true,
					Status:  "ok",
					Message: "candidate",
					Mode:    "fake_fast_check",
				},
				Candidate: true,
			}
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
					Payload: map[string]interface{}{
						"uploadId":  "upload-test-1",
						"partCount": 1,
					},
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
	if running.Task.CompletionKind != CompletionKindRealTransfer {
		t.Fatalf("expected completion kind real_transfer, got %s", running.Task.CompletionKind)
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
	fastCheck, ok := running.Results[0].Payload["fastCheck"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected fastCheck payload, got %#v", running.Results[0].Payload["fastCheck"])
	}
	if candidate, _ := fastCheck["candidate"].(bool); !candidate {
		t.Fatalf("expected fastCheck candidate=true, got %#v", fastCheck)
	}
	if status, _ := fastCheck["status"].(string); status != "ok" {
		t.Fatalf("expected fastCheck status ok, got %#v", fastCheck)
	}
	uploadPayload, ok := running.Results[0].Payload["upload"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected upload payload, got %#v", running.Results[0].Payload["upload"])
	}
	if got := uploadPayload["uploadId"].(string); got != "upload-test-1" {
		t.Fatalf("expected uploadId upload-test-1, got %#v", uploadPayload)
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

func TestServiceRuntimeFallsBackAfterFastUploadPrecheckMiss(t *testing.T) {
	ctx := context.Background()
	store, err := sqlitestore.New(ctx, filepath.Join(t.TempDir(), "runtime-fastcheck.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer func() { _ = store.Close() }()

	uploadStrategies := make([]string, 0, 2)
	adapter := &scriptedAdapter{
		meta: provider.Provider{
			Key:              "runtime_fastcheck",
			DisplayName:      "Runtime FastCheck",
			ProtocolGroup:    "fake",
			AuthModes:        []string{"manual_token"},
			FastUploadInputs: []string{"md5", "size"},
			FallbackModes:    []string{"download_upload"},
			ConflictPolicies: []provider.ConflictPolicy{provider.ConflictPolicyAutoRenameNew},
			Status:           "planned",
		},
		capability: provider.CapabilitySet{
			SupportsAuthValidation: true,
			SupportsFastUpload:     true,
			SupportsUpload:         true,
		},
		fastCheckFunc: func(req provider.FastUploadCheckRequest) provider.FastUploadCheckResult {
			return provider.FastUploadCheckResult{
				OperationResult: provider.OperationResult{
					OK:      true,
					Status:  "ok",
					Message: "not candidate",
					Mode:    "fake_fast_check",
				},
				Candidate: false,
			}
		},
		uploadFunc: func(req provider.UploadRequest) provider.UploadResult {
			uploadStrategies = append(uploadStrategies, req.Strategy)
			return provider.UploadResult{
				OperationResult: provider.OperationResult{
					OK:      true,
					Status:  "ok",
					Message: "binary ok",
					Mode:    "fake_binary",
				},
			}
		},
	}

	registry := provider.NewRegistry(adapter)
	authSvc := auth.NewService(store, registry)
	svc := NewService(store, registry, authSvc)
	profile, err := authSvc.CreateProfile(ctx, auth.CreateProfileInput{
		ProviderKey: "runtime_fastcheck",
		AuthMode:    "manual_token",
		DisplayName: "runtime fastcheck",
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
		TargetProvider:  "runtime_fastcheck",
		TargetProfileID: profile.ID,
		ThresholdMB:     0,
		ConflictPolicy:  provider.ConflictPolicyAutoRenameNew,
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
	if len(uploadStrategies) != 1 || uploadStrategies[0] != string(planner.StrategyDownloadUpload) {
		t.Fatalf("expected only download_upload to execute after precheck miss, got %#v", uploadStrategies)
	}
	if len(running.Results) != 1 {
		t.Fatalf("expected one result, got %d", len(running.Results))
	}
	if value, _ := running.Results[0].Payload["fallbackUsed"].(bool); !value {
		t.Fatalf("expected fallbackUsed payload, got %+v", running.Results[0].Payload)
	}
	fastCheck, ok := running.Results[0].Payload["fastCheck"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected fastCheck payload, got %#v", running.Results[0].Payload["fastCheck"])
	}
	if candidate, _ := fastCheck["candidate"].(bool); candidate {
		t.Fatalf("expected fastCheck candidate=false, got %#v", fastCheck)
	}
	if got := running.Results[0].Message; got != "Fast upload pre-check reported no candidate, fallback to download_upload succeeded." {
		t.Fatalf("unexpected fallback message: %s", got)
	}
}

func TestServiceRuntimePreservesUploadFailureEvidence(t *testing.T) {
	ctx := context.Background()
	store, err := sqlitestore.New(ctx, filepath.Join(t.TempDir(), "runtime-upload-evidence.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer func() { _ = store.Close() }()

	adapter := &scriptedAdapter{
		meta: provider.Provider{
			Key:              "runtime_upload_evidence",
			DisplayName:      "Runtime Upload Evidence",
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
			if req.ResumeUpload != nil {
				return provider.UploadResult{
					OperationResult: provider.OperationResult{
						OK:      true,
						Status:  "ok",
						Message: "resumed multipart upload",
						Mode:    "fake_upload_resume",
						Payload: map[string]interface{}{
							"itemPath":       req.ResumeUpload.ItemPath,
							"providerStatus": req.ResumeUpload.ProviderStatus,
							"updatedAt":      req.ResumeUpload.UpdatedAt,
							"fileId":         req.ResumeUpload.FileID,
							"uploadId":       req.ResumeUpload.UploadID,
							"resumedUpload":  true,
							"providerData":   req.ResumeUpload.ProviderData,
						},
					},
				}
			}
			return provider.UploadResult{
				OperationResult: provider.OperationResult{
					Status:  "provider_request_failed",
					Message: "multipart part failed",
					Mode:    "fake_upload_error",
					Payload: map[string]interface{}{
						"file_id":          "file-fail-1",
						"uploadId":         "upload-fail-1",
						"partCount":        3,
						"failedPartNumber": 2,
						"nextPartNumber":   2,
						"providerData": map[string]interface{}{
							"resumable": map[string]interface{}{
								"provider": "S3",
								"key":      "demo",
							},
						},
					},
				},
			}
		},
	}

	registry := provider.NewRegistry(adapter)
	authSvc := auth.NewService(store, registry)
	svc := NewService(store, registry, authSvc)
	profile, err := authSvc.CreateProfile(ctx, auth.CreateProfileInput{
		ProviderKey: "runtime_upload_evidence",
		AuthMode:    "manual_token",
		DisplayName: "runtime upload evidence",
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
		TargetProvider:  "runtime_upload_evidence",
		TargetProfileID: profile.ID,
		ThresholdMB:     1,
		ConflictPolicy:  provider.ConflictPolicyAutoRenameNew,
		Entries: []planner.SourceEntry{
			{Path: "/a.bin", Size: 1024, LocalPath: localFile},
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
	if len(running.Results) != 1 {
		t.Fatalf("expected one result, got %d", len(running.Results))
	}
	uploadPayload, ok := running.Results[0].Payload["upload"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected upload payload, got %#v", running.Results[0].Payload["upload"])
	}
	if got := intTaskValue(uploadPayload["failedPartNumber"]); got != 2 {
		t.Fatalf("expected failedPartNumber 2, got %#v", uploadPayload)
	}
	if got := intTaskValue(uploadPayload["nextPartNumber"]); got != 2 {
		t.Fatalf("expected nextPartNumber 2, got %#v", uploadPayload)
	}
	if running.Runtime.UploadCheckpoint == nil {
		t.Fatalf("expected runtime upload checkpoint, got %#v", running.Runtime)
	}
	if got := running.Runtime.UploadCheckpoint.UploadID; got != "upload-fail-1" {
		t.Fatalf("expected upload checkpoint uploadId upload-fail-1, got %#v", running.Runtime.UploadCheckpoint)
	}
	if got := running.Runtime.UploadCheckpoint.NextPartNumber; got != 2 {
		t.Fatalf("expected upload checkpoint nextPartNumber 2, got %#v", running.Runtime.UploadCheckpoint)
	}
	if got := running.Runtime.UploadCheckpoint.FileID; got != "file-fail-1" {
		t.Fatalf("expected upload checkpoint fileId file-fail-1, got %#v", running.Runtime.UploadCheckpoint)
	}
	if resumable, ok := running.Runtime.UploadCheckpoint.ProviderData["resumable"].(map[string]interface{}); !ok || stringValue(resumable["provider"]) != "S3" {
		t.Fatalf("expected upload checkpoint providerData.resumable, got %#v", running.Runtime.UploadCheckpoint.ProviderData)
	}
	if len(running.Runtime.RetryQueue) != 1 {
		t.Fatalf("expected one retry queue item, got %#v", running.Runtime.RetryQueue)
	}
	if running.Runtime.RetryQueue[0].UploadCheckpoint == nil {
		t.Fatalf("expected retry queue upload checkpoint, got %#v", running.Runtime.RetryQueue[0])
	}
	if got := running.Runtime.RetryQueue[0].UploadCheckpoint.FailedPartNumber; got != 2 {
		t.Fatalf("expected retry queue failedPartNumber 2, got %#v", running.Runtime.RetryQueue[0].UploadCheckpoint)
	}
	retried, ok, err := svc.Retry(ctx, detail.Task.ID)
	if err != nil {
		t.Fatalf("Retry() error = %v", err)
	}
	if !ok {
		t.Fatal("expected retried task to exist")
	}
	if retried.Runtime.UploadCheckpoint == nil {
		t.Fatalf("expected retried runtime upload checkpoint, got %#v", retried.Runtime)
	}
	if got := retried.Runtime.UploadCheckpoint.UploadID; got != "upload-fail-1" {
		t.Fatalf("expected retried upload checkpoint uploadId upload-fail-1, got %#v", retried.Runtime.UploadCheckpoint)
	}
	checkpoints, ok := retried.Plan.Metadata["retryUploadCheckpoints"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected retryUploadCheckpoints metadata, got %#v", retried.Plan.Metadata["retryUploadCheckpoints"])
	}
	if _, ok := checkpoints["/a.bin"]; !ok {
		t.Fatalf("expected retryUploadCheckpoints to include /a.bin, got %#v", checkpoints)
	}
	resumed, ok, err := svc.Run(ctx, retried.Task.ID)
	if err != nil {
		t.Fatalf("Run(retried) error = %v", err)
	}
	if !ok {
		t.Fatal("expected retried run task to exist")
	}
	if got := resumed.Results[0].Mode; got != "fake_upload_resume" {
		t.Fatalf("expected resumed upload mode fake_upload_resume, got %#v", resumed.Results)
	}
	resumePayload, ok := resumed.Results[0].Payload["upload"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected resumed upload payload, got %#v", resumed.Results[0].Payload["upload"])
	}
	if got, _ := resumePayload["itemPath"].(string); got != "/a.bin" {
		t.Fatalf("expected resumed upload itemPath /a.bin, got %#v", resumePayload)
	}
	if got, _ := resumePayload["providerStatus"].(string); got != "provider_request_failed" {
		t.Fatalf("expected resumed upload providerStatus provider_request_failed, got %#v", resumePayload)
	}
	if got, _ := resumePayload["updatedAt"].(string); got == "" {
		t.Fatalf("expected resumed upload updatedAt, got %#v", resumePayload)
	}
	if got, _ := resumePayload["fileId"].(string); got != "file-fail-1" {
		t.Fatalf("expected resumed upload fileId file-fail-1, got %#v", resumePayload)
	}
	if resumeProviderData, ok := resumePayload["providerData"].(map[string]interface{}); !ok || resumeProviderData == nil {
		t.Fatalf("expected resumed upload providerData, got %#v", resumePayload)
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
		RiskOverride: &planner.RiskProfileOverride{
			RequestIntervalMS: intPtrTask(1400),
			RetryLimit:        intPtrTask(1),
			RiskKeywords:      []string{"rate_limited"},
		},
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
	createdRiskProfile, ok := detail.Plan.Metadata["riskProfile"].(planner.RiskProfile)
	if !ok {
		t.Fatalf("expected planner.RiskProfile in create metadata, got %#v", detail.Plan.Metadata["riskProfile"])
	}
	if len(createdRiskProfile.RiskKeywords) != 1 || createdRiskProfile.RiskKeywords[0] != "rate_limited" {
		t.Fatalf("expected override risk keywords at create time, got %#v", createdRiskProfile.RiskKeywords)
	}

	running, ok, err := svc.Run(ctx, detail.Task.ID)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !ok {
		t.Fatal("expected task to exist")
	}
	if running.Task.State != StateBlocked {
		t.Fatalf("expected blocked, got %s", running.Task.State)
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
	if running.Runtime.RiskHitCount != 1 {
		t.Fatalf("expected runtime risk hit count 1, got %d", running.Runtime.RiskHitCount)
	}
	if running.Runtime.LastRiskStatus != "rate_limited" {
		t.Fatalf("expected last risk status rate_limited, got %s", running.Runtime.LastRiskStatus)
	}
	if running.Runtime.PendingCount != 1 {
		t.Fatalf("expected runtime pending count 1, got %d", running.Runtime.PendingCount)
	}
	if len(running.Runtime.PendingTree) != 1 {
		t.Fatalf("expected 1 pending root node, got %d", len(running.Runtime.PendingTree))
	}
	pendingRoot := running.Runtime.PendingTree[0]
	if pendingRoot.Path != "/" {
		t.Fatalf("expected pending root path /, got %s", pendingRoot.Path)
	}
	if pendingRoot.ItemCount != 1 {
		t.Fatalf("expected pending root item count 1, got %d", pendingRoot.ItemCount)
	}
	if len(pendingRoot.Children) != 1 {
		t.Fatalf("expected pending root children len 1, got %d", len(pendingRoot.Children))
	}
	pendingFile := pendingRoot.Children[0]
	if pendingFile.Path != "/pending.bin" {
		t.Fatalf("expected pending file /pending.bin, got %s", pendingFile.Path)
	}
	if pendingFile.ProviderStatus != "pending_manual_requires_confirmation" {
		t.Fatalf("expected pending file provider status pending_manual_requires_confirmation, got %s", pendingFile.ProviderStatus)
	}
	rateIndex := -1
	for idx, item := range running.Plan.Items {
		if item.Path == "/rate.bin" {
			rateIndex = idx
			break
		}
	}
	if rateIndex < 0 {
		t.Fatal("expected /rate.bin in plan items")
	}
	switch riskHit := running.Results[rateIndex].Payload["riskHit"].(type) {
	case RiskHit:
		if riskHit.Keyword != "rate_limited" {
			t.Fatalf("expected riskHit keyword rate_limited, got %v", riskHit.Keyword)
		}
	case map[string]interface{}:
		if keyword, _ := riskHit["keyword"].(string); keyword != "rate_limited" {
			t.Fatalf("expected riskHit keyword rate_limited, got %v", riskHit["keyword"])
		}
	default:
		t.Fatalf("expected riskHit payload on /rate.bin, got %#v", running.Results[rateIndex].Payload["riskHit"])
	}

	evidence, err := svc.RuntimeEvidence(ctx)
	if err != nil {
		t.Fatalf("RuntimeEvidence() error = %v", err)
	}
	if evidence.PendingResultCount != 1 {
		t.Fatalf("expected pending result count 1, got %d", evidence.PendingResultCount)
	}
	if running.Runtime.RetryableCount != 2 {
		t.Fatalf("expected retryable count 2, got %d", running.Runtime.RetryableCount)
	}
	if running.Runtime.BlockedRetryCount != 2 {
		t.Fatalf("expected blocked retry count 2, got %d", running.Runtime.BlockedRetryCount)
	}
	if len(running.Runtime.RetryQueue) != 4 {
		t.Fatalf("expected retry queue len 4, got %d", len(running.Runtime.RetryQueue))
	}
	retryItemByPath := func(path string) RetryQueueItem {
		t.Helper()
		for _, item := range running.Runtime.RetryQueue {
			if item.Path == path {
				return item
			}
		}
		t.Fatalf("expected retry queue item for %s, got %#v", path, running.Runtime.RetryQueue)
		return RetryQueueItem{}
	}
	pendingRetry := retryItemByPath("/pending.bin")
	if pendingRetry.RetryClass != "pending_manual" || pendingRetry.RetryAction != "retry_after_manual_confirmation" || !pendingRetry.Retryable || pendingRetry.Blocked {
		t.Fatalf("unexpected pending retry item: %#v", pendingRetry)
	}
	authRetry := retryItemByPath("/auth.bin")
	if authRetry.RetryClass != "auth_expired" || authRetry.RetryAction != "refresh_auth_profile" || authRetry.Retryable || !authRetry.Blocked {
		t.Fatalf("unexpected auth retry item: %#v", authRetry)
	}
	rateRetry := retryItemByPath("/rate.bin")
	if rateRetry.RetryClass != "rate_limited" || rateRetry.RetryAction != "retry_after_cooldown" || !rateRetry.Retryable || rateRetry.Blocked {
		t.Fatalf("unexpected rate-limited retry item: %#v", rateRetry)
	}
	localRetry := retryItemByPath("/missing.bin")
	if localRetry.RetryClass != "local_file_missing" || localRetry.RetryAction != "restore_local_file" || localRetry.Retryable || !localRetry.Blocked {
		t.Fatalf("unexpected local-file retry item: %#v", localRetry)
	}
}

func TestServiceRuntimeMapsProviderSessionMissingIntoManualIntervention(t *testing.T) {
	ctx := context.Background()
	store, err := sqlitestore.New(ctx, filepath.Join(t.TempDir(), "runtime-missing-uploadid.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer func() { _ = store.Close() }()

	adapter := &scriptedAdapter{
		meta: provider.Provider{
			Key:              "runtime_missing_uploadid",
			DisplayName:      "Runtime Missing UploadID",
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
			return provider.UploadResult{OperationResult: provider.OperationResult{Status: "missing_uploadid", Message: "provider omitted uploadid", Mode: "fake_missing_uploadid"}}
		},
	}

	registry := provider.NewRegistry(adapter)
	authSvc := auth.NewService(store, registry)
	svc := NewService(store, registry, authSvc)
	profile, err := authSvc.CreateProfile(ctx, auth.CreateProfileInput{ProviderKey: "runtime_missing_uploadid", AuthMode: "manual_token", DisplayName: "runtime missing uploadid", Token: "token-1"})
	if err != nil {
		t.Fatalf("CreateProfile() error = %v", err)
	}
	localFile := filepath.Join(t.TempDir(), "missing-uploadid.bin")
	if err := os.WriteFile(localFile, []byte("missing-uploadid"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	detail, err := svc.Create(ctx, CreateRequest{SourceProvider: "guangya", TargetProvider: "runtime_missing_uploadid", TargetProfileID: profile.ID, ThresholdMB: 1, Entries: []planner.SourceEntry{{Path: "/missing-uploadid.bin", Size: 1024, MD5: "abc", LocalPath: localFile}}})
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
	if running.Task.State != StateBlocked {
		t.Fatalf("expected blocked, got %s", running.Task.State)
	}
	if len(running.Runtime.RetryQueue) != 1 {
		t.Fatalf("expected retry queue len 1, got %#v", running.Runtime.RetryQueue)
	}
	item := running.Runtime.RetryQueue[0]
	if item.ProviderStatus != "missing_uploadid" {
		t.Fatalf("expected providerStatus missing_uploadid, got %#v", item)
	}
	if item.RetryClass != "provider_session_missing" || item.RetryAction != "manual_intervention_required" || item.Retryable || !item.Blocked {
		t.Fatalf("unexpected missing_uploadid retry mapping: %#v", item)
	}
	if running.Runtime.BlockedReason != "retry_queue_requires_provider_session_rebuild" {
		t.Fatalf("expected blockedReason retry_queue_requires_provider_session_rebuild, got %#v", running.Runtime.BlockedReason)
	}
	if running.Runtime.BlockedAction != "manual_intervention_required" {
		t.Fatalf("expected blockedAction manual_intervention_required, got %#v", running.Runtime.BlockedAction)
	}
}

func TestServiceRuntimeMapsAuthInvalidIntoAuthRefreshRetryQueue(t *testing.T) {
	ctx := context.Background()
	store, err := sqlitestore.New(ctx, filepath.Join(t.TempDir(), "runtime-auth-invalid.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer func() { _ = store.Close() }()

	adapter := &scriptedAdapter{
		meta: provider.Provider{
			Key:              "runtime_auth_invalid",
			DisplayName:      "Runtime Auth Invalid",
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
			return provider.UploadResult{
				OperationResult: provider.OperationResult{
					Status:  "auth_invalid",
					Message: "provider rejected token",
					Mode:    "fake_auth_invalid",
				},
			}
		},
	}

	registry := provider.NewRegistry(adapter)
	authSvc := auth.NewService(store, registry)
	svc := NewService(store, registry, authSvc)
	profile, err := authSvc.CreateProfile(ctx, auth.CreateProfileInput{
		ProviderKey: "runtime_auth_invalid",
		AuthMode:    "manual_token",
		DisplayName: "runtime auth invalid",
		Token:       "token-1",
	})
	if err != nil {
		t.Fatalf("CreateProfile() error = %v", err)
	}

	detail, err := svc.Create(ctx, CreateRequest{
		SourceProvider:  "guangya",
		TargetProvider:  "runtime_auth_invalid",
		TargetProfileID: profile.ID,
		ThresholdMB:     1,
		Entries: []planner.SourceEntry{
			{Path: "/auth-invalid.bin", Size: 1024, MD5: "abc", LocalPath: filepath.Join(t.TempDir(), "auth-invalid.bin")},
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
	if running.Task.State != StateBlocked {
		t.Fatalf("expected blocked, got %s", running.Task.State)
	}
	if len(running.Runtime.RetryQueue) != 1 {
		t.Fatalf("expected retry queue len 1, got %#v", running.Runtime.RetryQueue)
	}
	retryItem := running.Runtime.RetryQueue[0]
	if retryItem.ProviderStatus != "auth_invalid" {
		t.Fatalf("expected providerStatus auth_invalid, got %#v", retryItem)
	}
	if retryItem.RetryClass != "auth_expired" || retryItem.RetryAction != "refresh_auth_profile" || retryItem.Retryable || !retryItem.Blocked {
		t.Fatalf("unexpected auth_invalid retry mapping: %#v", retryItem)
	}
	if running.Runtime.BlockedAction != "refresh_auth_profile" {
		t.Fatalf("expected blockedAction refresh_auth_profile, got %#v", running.Runtime.BlockedAction)
	}
}

func TestServiceRuntimeBuildsPendingRelayTreeByRootAndDirectory(t *testing.T) {
	ctx := context.Background()
	store, err := sqlitestore.New(ctx, filepath.Join(t.TempDir(), "pending-tree.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer func() { _ = store.Close() }()

	adapter := &scriptedAdapter{
		meta: provider.Provider{
			Key:              "pending_tree_target",
			DisplayName:      "Pending Tree Target",
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
			if strings.HasPrefix(req.Path, "/1/") || strings.HasPrefix(req.Path, "/2/") {
				return provider.UploadResult{
					OperationResult: provider.OperationResult{
						Status:  "pending_manual_requires_confirmation",
						Message: "pending manual",
						Mode:    "fake_pending",
					},
				}
			}
			return provider.UploadResult{
				OperationResult: provider.OperationResult{
					OK:      true,
					Status:  "ok",
					Message: "ok",
					Mode:    "fake_ok",
				},
			}
		},
	}

	registry := provider.NewRegistry(adapter)
	authSvc := auth.NewService(store, registry)
	svc := NewService(store, registry, authSvc)
	profile, err := authSvc.CreateProfile(ctx, auth.CreateProfileInput{
		ProviderKey: "pending_tree_target",
		AuthMode:    "manual_token",
		DisplayName: "pending tree target",
		Token:       "token-1",
	})
	if err != nil {
		t.Fatalf("CreateProfile() error = %v", err)
	}

	detail, err := svc.Create(ctx, CreateRequest{
		SourceProvider:  "guangya",
		TargetProvider:  "pending_tree_target",
		TargetProfileID: profile.ID,
		ThresholdMB:     1,
		SelectedRoots:   []string{"/1", "/2"},
		Entries: []planner.SourceEntry{
			{Path: "/1/11/111/a.bin", Size: 10 * 1024 * 1024},
			{Path: "/1/11/112/b.bin", Size: 11 * 1024 * 1024},
			{Path: "/2/22/c.bin", Size: 12 * 1024 * 1024},
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
	if running.Runtime.PendingCount != 3 {
		t.Fatalf("expected runtime pending count 3, got %d", running.Runtime.PendingCount)
	}
	if len(running.Runtime.PendingTree) != 2 {
		t.Fatalf("expected 2 pending root nodes, got %d", len(running.Runtime.PendingTree))
	}
	firstRoot := running.Runtime.PendingTree[0]
	if firstRoot.Path != "/1" {
		t.Fatalf("expected first pending root /1, got %s", firstRoot.Path)
	}
	if firstRoot.ItemCount != 2 {
		t.Fatalf("expected root /1 item count 2, got %d", firstRoot.ItemCount)
	}
	if len(firstRoot.Children) != 1 || firstRoot.Children[0].Path != "/1/11" {
		t.Fatalf("expected root /1 to have child /1/11, got %#v", firstRoot.Children)
	}
	level2 := firstRoot.Children[0]
	if len(level2.Children) != 2 {
		t.Fatalf("expected /1/11 children len 2, got %d", len(level2.Children))
	}
	if level2.Children[0].Path != "/1/11/111" || level2.Children[1].Path != "/1/11/112" {
		t.Fatalf("expected leaf directories /1/11/111 and /1/11/112, got %#v", level2.Children)
	}
	if len(level2.Children[0].Children) != 1 || level2.Children[0].Children[0].Path != "/1/11/111/a.bin" {
		t.Fatalf("expected pending file under /1/11/111, got %#v", level2.Children[0].Children)
	}
	secondRoot := running.Runtime.PendingTree[1]
	if secondRoot.Path != "/2" {
		t.Fatalf("expected second pending root /2, got %s", secondRoot.Path)
	}
	if len(secondRoot.Children) != 1 || secondRoot.Children[0].Path != "/2/22" {
		t.Fatalf("expected root /2 child /2/22, got %#v", secondRoot.Children)
	}
	if len(secondRoot.Children[0].Children) != 1 || secondRoot.Children[0].Children[0].Path != "/2/22/c.bin" {
		t.Fatalf("expected pending file /2/22/c.bin, got %#v", secondRoot.Children[0].Children)
	}
}

func TestServicePendingTreeRespectsSelectedRootOrderWithPreScanFlat(t *testing.T) {
	ctx := context.Background()
	store, err := sqlitestore.New(ctx, filepath.Join(t.TempDir(), "pending-tree-order.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer func() { _ = store.Close() }()

	adapter := &scriptedAdapter{
		meta: provider.Provider{
			Key:              "pending_tree_order_target",
			DisplayName:      "Pending Tree Order Target",
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
			if strings.HasPrefix(req.Path, "/1/") || strings.HasPrefix(req.Path, "/2/") {
				return provider.UploadResult{
					OperationResult: provider.OperationResult{
						Status:  "pending_manual_requires_confirmation",
						Message: "pending manual",
						Mode:    "fake_pending",
					},
				}
			}
			return provider.UploadResult{
				OperationResult: provider.OperationResult{
					OK:      true,
					Status:  "ok",
					Message: "ok",
					Mode:    "fake_ok",
				},
			}
		},
	}

	registry := provider.NewRegistry(adapter)
	authSvc := auth.NewService(store, registry)
	svc := NewService(store, registry, authSvc)
	profile, err := authSvc.CreateProfile(ctx, auth.CreateProfileInput{
		ProviderKey: "pending_tree_order_target",
		AuthMode:    "manual_token",
		DisplayName: "pending tree order target",
		Token:       "token-1",
	})
	if err != nil {
		t.Fatalf("CreateProfile() error = %v", err)
	}

	detail, err := svc.Create(ctx, CreateRequest{
		SourceProvider:  "guangya",
		TargetProvider:  "pending_tree_order_target",
		TargetProfileID: profile.ID,
		ThresholdMB:     1,
		SelectedRoots:   []string{"/1", "/2"},
		ExecutionMode:   planner.ExecutionModePreScanFlat,
		Entries: []planner.SourceEntry{
			{Path: "/2/22/c.bin", Size: 12 * 1024 * 1024},
			{Path: "/1/11/111/a.bin", Size: 10 * 1024 * 1024},
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
	if len(running.Runtime.PendingTree) != 2 {
		t.Fatalf("expected 2 pending root nodes, got %d", len(running.Runtime.PendingTree))
	}
	if running.Runtime.PendingTree[0].Path != "/1" {
		t.Fatalf("expected selected root /1 to stay first, got %s", running.Runtime.PendingTree[0].Path)
	}
	if running.Runtime.PendingTree[1].Path != "/2" {
		t.Fatalf("expected selected root /2 to stay second, got %s", running.Runtime.PendingTree[1].Path)
	}
}

func TestServicePauseStopsRunningTaskBetweenItemsAndResumeContinues(t *testing.T) {
	ctx := context.Background()
	store, err := sqlitestore.New(ctx, filepath.Join(t.TempDir(), "pause-run.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer func() { _ = store.Close() }()

	startedFirstUpload := make(chan struct{})
	releaseFirstUpload := make(chan struct{})
	uploadCalls := make([]string, 0, 2)
	adapter := &scriptedAdapter{
		meta: provider.Provider{
			Key:              "pause_run_target",
			DisplayName:      "Pause Run Target",
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
			uploadCalls = append(uploadCalls, req.Path)
			if len(uploadCalls) == 1 {
				close(startedFirstUpload)
				<-releaseFirstUpload
			}
			return provider.UploadResult{
				OperationResult: provider.OperationResult{
					OK:      true,
					Status:  "ok",
					Message: "ok",
					Mode:    "fake_ok",
				},
			}
		},
	}

	registry := provider.NewRegistry(adapter)
	authSvc := auth.NewService(store, registry)
	svc := NewService(store, registry, authSvc)
	profile, err := authSvc.CreateProfile(ctx, auth.CreateProfileInput{
		ProviderKey: "pause_run_target",
		AuthMode:    "manual_token",
		DisplayName: "pause run target",
		Token:       "token-1",
	})
	if err != nil {
		t.Fatalf("CreateProfile() error = %v", err)
	}

	detail, err := svc.Create(ctx, CreateRequest{
		SourceProvider:  "guangya",
		TargetProvider:  "pause_run_target",
		TargetProfileID: profile.ID,
		ThresholdMB:     1,
		Entries: []planner.SourceEntry{
			{Path: "/a.bin", Size: 10 * 1024 * 1024, MD5: "md5-a"},
			{Path: "/b.bin", Size: 10 * 1024 * 1024, MD5: "md5-b"},
		},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	runErrCh := make(chan error, 1)
	runResultCh := make(chan Detail, 1)
	go func() {
		running, _, runErr := svc.Run(ctx, detail.Task.ID)
		if runErr != nil {
			runErrCh <- runErr
			return
		}
		runResultCh <- running
	}()

	select {
	case <-startedFirstUpload:
	case <-time.After(5 * time.Second):
		t.Fatal("expected first upload to start")
	}

	paused, ok, err := svc.Pause(ctx, detail.Task.ID)
	if err != nil {
		t.Fatalf("Pause() error = %v", err)
	}
	if !ok {
		t.Fatal("expected task to exist for pause")
	}
	if paused.Task.State != StateRunning {
		t.Fatalf("expected running task state while pause is pending, got %s", paused.Task.State)
	}
	if paused.Runtime.ExecutionState != "pause_requested" {
		t.Fatalf("expected runtime pause_requested, got %s", paused.Runtime.ExecutionState)
	}
	if !paused.Runtime.PauseRequested {
		t.Fatal("expected pauseRequested true while first upload is still running")
	}

	close(releaseFirstUpload)

	var running Detail
	select {
	case err := <-runErrCh:
		t.Fatalf("Run() error = %v", err)
	case running = <-runResultCh:
	case <-time.After(5 * time.Second):
		t.Fatal("expected paused run to return")
	}
	if running.Task.State != StatePaused {
		t.Fatalf("expected run to stop in paused state, got %s", running.Task.State)
	}
	if running.Runtime.ExecutionState != "paused" {
		t.Fatalf("expected runtime paused, got %s", running.Runtime.ExecutionState)
	}
	if running.Runtime.PauseRequested {
		t.Fatal("expected pauseRequested cleared after task reaches paused state")
	}
	if running.Runtime.ProcessedCount != 1 {
		t.Fatalf("expected processed count 1 after pause, got %d", running.Runtime.ProcessedCount)
	}
	if running.Runtime.LastCompletedPath != "/a.bin" {
		t.Fatalf("expected last completed /a.bin, got %s", running.Runtime.LastCompletedPath)
	}
	if len(running.Results) != 1 {
		t.Fatalf("expected 1 result before pause, got %d", len(running.Results))
	}

	resumed, ok, err := svc.Resume(ctx, detail.Task.ID)
	if err != nil {
		t.Fatalf("Resume() error = %v", err)
	}
	if !ok {
		t.Fatal("expected task to exist for resume")
	}
	if resumed.Task.State != StateReady {
		t.Fatalf("expected ready after resume, got %s", resumed.Task.State)
	}
	if resumed.Runtime.ExecutionState != "ready_to_resume" {
		t.Fatalf("expected ready_to_resume after resume, got %s", resumed.Runtime.ExecutionState)
	}

	finished, ok, err := svc.Run(ctx, detail.Task.ID)
	if err != nil {
		t.Fatalf("Run() after resume error = %v", err)
	}
	if !ok {
		t.Fatal("expected resumed task to exist")
	}
	if finished.Task.State != StateCompleted {
		t.Fatalf("expected completed after resume, got %s", finished.Task.State)
	}
	if len(uploadCalls) != 2 {
		t.Fatalf("expected 2 upload calls across pause/resume, got %d: %#v", len(uploadCalls), uploadCalls)
	}
	if uploadCalls[0] != "/a.bin" || uploadCalls[1] != "/b.bin" {
		t.Fatalf("expected upload order [/a.bin /b.bin], got %#v", uploadCalls)
	}
}

func TestServiceResumeCancelsPauseRequestWhileTaskStillRunning(t *testing.T) {
	ctx := context.Background()
	store, err := sqlitestore.New(ctx, filepath.Join(t.TempDir(), "pause-request-cancel.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer func() { _ = store.Close() }()

	startedFirstUpload := make(chan struct{})
	releaseFirstUpload := make(chan struct{})
	startedSecondUpload := make(chan struct{}, 1)
	adapter := &scriptedAdapter{
		meta: provider.Provider{
			Key:              "pause_request_cancel_target",
			DisplayName:      "Pause Request Cancel Target",
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
			if req.Path == "/a.bin" {
				close(startedFirstUpload)
				<-releaseFirstUpload
			}
			if req.Path == "/b.bin" {
				startedSecondUpload <- struct{}{}
			}
			return provider.UploadResult{
				OperationResult: provider.OperationResult{
					OK:      true,
					Status:  "ok",
					Message: "ok",
					Mode:    "fake_ok",
				},
			}
		},
	}

	registry := provider.NewRegistry(adapter)
	authSvc := auth.NewService(store, registry)
	svc := NewService(store, registry, authSvc)
	profile, err := authSvc.CreateProfile(ctx, auth.CreateProfileInput{
		ProviderKey: "pause_request_cancel_target",
		AuthMode:    "manual_token",
		DisplayName: "pause request cancel target",
		Token:       "token-1",
	})
	if err != nil {
		t.Fatalf("CreateProfile() error = %v", err)
	}

	detail, err := svc.Create(ctx, CreateRequest{
		SourceProvider:  "guangya",
		TargetProvider:  "pause_request_cancel_target",
		TargetProfileID: profile.ID,
		ThresholdMB:     1,
		Entries: []planner.SourceEntry{
			{Path: "/a.bin", Size: 10 * 1024 * 1024, MD5: "md5-a"},
			{Path: "/b.bin", Size: 10 * 1024 * 1024, MD5: "md5-b"},
		},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	runErrCh := make(chan error, 1)
	runResultCh := make(chan Detail, 1)
	go func() {
		running, _, runErr := svc.Run(ctx, detail.Task.ID)
		if runErr != nil {
			runErrCh <- runErr
			return
		}
		runResultCh <- running
	}()

	select {
	case <-startedFirstUpload:
	case <-time.After(5 * time.Second):
		t.Fatal("expected first upload to start")
	}

	paused, ok, err := svc.Pause(ctx, detail.Task.ID)
	if err != nil {
		t.Fatalf("Pause() error = %v", err)
	}
	if !ok {
		t.Fatal("expected task to exist for pause request")
	}
	if paused.Task.State != StateRunning || paused.Runtime.ExecutionState != "pause_requested" || !paused.Runtime.PauseRequested {
		t.Fatalf("expected running/pause_requested state, got task=%s runtime=%#v", paused.Task.State, paused.Runtime)
	}

	resumed, ok, err := svc.Resume(ctx, detail.Task.ID)
	if err != nil {
		t.Fatalf("Resume() while pause requested error = %v", err)
	}
	if !ok {
		t.Fatal("expected task to exist for resume")
	}
	if resumed.Task.State != StateRunning {
		t.Fatalf("expected running after canceling pause request, got %s", resumed.Task.State)
	}
	if resumed.Runtime.ExecutionState != "running" || resumed.Runtime.PauseRequested {
		t.Fatalf("expected running runtime after canceling pause request, got %#v", resumed.Runtime)
	}

	close(releaseFirstUpload)

	var finished Detail
	select {
	case err := <-runErrCh:
		t.Fatalf("Run() error = %v", err)
	case finished = <-runResultCh:
	case <-time.After(5 * time.Second):
		t.Fatal("expected run to finish after canceling pause request")
	}
	if finished.Task.State != StateCompleted {
		t.Fatalf("expected completed after canceling pause request, got %s", finished.Task.State)
	}
	if finished.Runtime.ExecutionState != "completed" || finished.Runtime.PauseRequested {
		t.Fatalf("expected completed runtime without pending pause, got %#v", finished.Runtime)
	}
	select {
	case <-startedSecondUpload:
	case <-time.After(5 * time.Second):
		t.Fatal("expected second upload to proceed after pause request was canceled")
	}
}

func TestServiceRetryNarrowsToPendingRelayEntriesAndReplaysOnlyPendingItems(t *testing.T) {
	ctx := context.Background()
	store, err := sqlitestore.New(ctx, filepath.Join(t.TempDir(), "retry-pending.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer func() { _ = store.Close() }()

	attempts := make(map[string]int)
	uploadCalls := make([]string, 0)
	adapter := &scriptedAdapter{
		meta: provider.Provider{
			Key:              "retry_pending_target",
			DisplayName:      "Retry Pending Target",
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
			uploadCalls = append(uploadCalls, req.Path)
			attempts[req.Path]++
			switch req.Path {
			case "/pending.bin":
				if attempts[req.Path] == 1 {
					return provider.UploadResult{
						OperationResult: provider.OperationResult{
							Status:  "pending_manual_requires_confirmation",
							Message: "pending manual",
							Mode:    "fake_pending",
						},
					}
				}
				return provider.UploadResult{
					OperationResult: provider.OperationResult{
						OK:      true,
						Status:  "ok",
						Message: "manual confirmed",
						Mode:    "fake_ok",
					},
				}
			case "/done.bin":
				return provider.UploadResult{
					OperationResult: provider.OperationResult{
						OK:      true,
						Status:  "ok",
						Message: "done",
						Mode:    "fake_ok",
					},
				}
			default:
				return provider.UploadResult{
					OperationResult: provider.OperationResult{
						Status:  "auth_expired",
						Message: "auth expired",
						Mode:    "fake_auth",
					},
				}
			}
		},
	}

	registry := provider.NewRegistry(adapter)
	authSvc := auth.NewService(store, registry)
	svc := NewService(store, registry, authSvc)
	profile, err := authSvc.CreateProfile(ctx, auth.CreateProfileInput{
		ProviderKey: "retry_pending_target",
		AuthMode:    "manual_token",
		DisplayName: "retry pending target",
		Token:       "token-1",
	})
	if err != nil {
		t.Fatalf("CreateProfile() error = %v", err)
	}

	detail, err := svc.Create(ctx, CreateRequest{
		SourceProvider:  "guangya",
		TargetProvider:  "retry_pending_target",
		TargetProfileID: profile.ID,
		ThresholdMB:     1,
		Entries: []planner.SourceEntry{
			{Path: "/pending.bin", Size: 10 * 1024 * 1024},
			{Path: "/done.bin", Size: 1024, MD5: "abc"},
			{Path: "/auth.bin", Size: 512, LocalPath: filepath.Join(t.TempDir(), "auth.bin")},
		},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	firstRun, ok, err := svc.Run(ctx, detail.Task.ID)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !ok {
		t.Fatal("expected task to exist")
	}
	if len(firstRun.Results) != 3 {
		t.Fatalf("expected 3 results on first run, got %d", len(firstRun.Results))
	}
	if firstRun.Runtime.PendingCount != 1 {
		t.Fatalf("expected first run pending count 1, got %d", firstRun.Runtime.PendingCount)
	}

	retried, ok, err := svc.Retry(ctx, detail.Task.ID)
	if err != nil {
		t.Fatalf("Retry() error = %v", err)
	}
	if !ok {
		t.Fatal("expected retried task to exist")
	}
	if len(retried.Results) != 0 {
		t.Fatalf("expected retry to clear old results, got %d", len(retried.Results))
	}
	if len(retried.Plan.Items) != 1 || retried.Plan.Items[0].Path != "/pending.bin" {
		t.Fatalf("expected retried plan to keep only /pending.bin, got %#v", retried.Plan.Items)
	}
	if len(retried.SourceEntries) != 1 || retried.SourceEntries[0].Path != "/pending.bin" {
		t.Fatalf("expected retried entries to keep only /pending.bin, got %#v", retried.SourceEntries)
	}
	if retryPendingOnly, _ := retried.Plan.Metadata["retryPendingOnly"].(bool); !retryPendingOnly {
		t.Fatalf("expected retryPendingOnly metadata true, got %#v", retried.Plan.Metadata["retryPendingOnly"])
	}
	if retryMode, _ := retried.Plan.Metadata["retryMode"].(string); retryMode != "pending_only" {
		t.Fatalf("expected retryMode pending_only, got %s", retryMode)
	}

	secondRun, ok, err := svc.Run(ctx, detail.Task.ID)
	if err != nil {
		t.Fatalf("Run() after retry error = %v", err)
	}
	if !ok {
		t.Fatal("expected retried run task to exist")
	}
	if secondRun.Task.State != StateCompleted {
		t.Fatalf("expected completed after pending-only retry, got %s", secondRun.Task.State)
	}
	if len(secondRun.Results) != 1 || secondRun.Results[0].Payload["path"] != "/pending.bin" {
		t.Fatalf("expected second run to replay only /pending.bin, got %#v", secondRun.Results)
	}
	if secondRun.Runtime.PendingCount != 0 {
		t.Fatalf("expected pending count 0 after successful pending retry, got %d", secondRun.Runtime.PendingCount)
	}
	if len(uploadCalls) != 3 {
		t.Fatalf("expected 3 upload calls total, got %d: %#v", len(uploadCalls), uploadCalls)
	}
	if uploadCalls[2] != "/pending.bin" {
		t.Fatalf("expected last upload call /pending.bin, got %s", uploadCalls[2])
	}
}

func TestServiceRetryWithOptionsSelectedPendingSubsetKeepsOnlyChosenPaths(t *testing.T) {
	ctx := context.Background()
	store, err := sqlitestore.New(ctx, filepath.Join(t.TempDir(), "retry-selected-pending.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer func() { _ = store.Close() }()

	uploadCalls := []string{}
	adapter := &scriptedAdapter{
		meta: provider.Provider{
			Key:              "retry_selected_pending_target",
			DisplayName:      "Retry Selected Pending Target",
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
			uploadCalls = append(uploadCalls, req.Path)
			if len(uploadCalls) <= 2 {
				return provider.UploadResult{
					OperationResult: provider.OperationResult{
						Status:  "pending_manual_requires_confirmation",
						Message: "pending manual",
						Mode:    "fake_pending",
					},
				}
			}
			return provider.UploadResult{
				OperationResult: provider.OperationResult{
					OK:      true,
					Status:  "ok",
					Message: "selected pending resolved",
					Mode:    "fake_selected_pending_ok",
				},
			}
		},
	}

	registry := provider.NewRegistry(adapter)
	authSvc := auth.NewService(store, registry)
	svc := NewService(store, registry, authSvc)
	profile, err := authSvc.CreateProfile(ctx, auth.CreateProfileInput{
		ProviderKey: "retry_selected_pending_target",
		AuthMode:    "manual_token",
		DisplayName: "retry selected pending target",
		Token:       "token-1",
	})
	if err != nil {
		t.Fatalf("CreateProfile() error = %v", err)
	}

	detail, err := svc.Create(ctx, CreateRequest{
		SourceProvider:  "guangya",
		TargetProvider:  "retry_selected_pending_target",
		TargetProfileID: profile.ID,
		ThresholdMB:     1,
		Entries: []planner.SourceEntry{
			{Path: "/pending-a.bin", Size: 10 * 1024 * 1024},
			{Path: "/pending-b.bin", Size: 12 * 1024 * 1024},
		},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	firstRun, ok, err := svc.Run(ctx, detail.Task.ID)
	if err != nil || !ok {
		t.Fatalf("Run() error=%v ok=%v", err, ok)
	}
	if firstRun.Runtime.PendingCount != 2 {
		t.Fatalf("expected pending count 2, got %d", firstRun.Runtime.PendingCount)
	}

	retried, ok, err := svc.RetryWithOptions(ctx, detail.Task.ID, RetryOptions{
		Paths: []string{"/pending-b.bin"},
		Scope: "selected_pending_subset",
	})
	if err != nil || !ok {
		t.Fatalf("RetryWithOptions() error=%v ok=%v", err, ok)
	}
	if len(retried.Plan.Items) != 1 || retried.Plan.Items[0].Path != "/pending-b.bin" {
		t.Fatalf("expected selected pending retry to keep only /pending-b.bin, got %#v", retried.Plan.Items)
	}
	if retryMode, _ := retried.Plan.Metadata["retryMode"].(string); retryMode != "selected_pending_subset" {
		t.Fatalf("expected retryMode selected_pending_subset, got %#v", retried.Plan.Metadata["retryMode"])
	}
	if selectedPaths, ok := retried.Plan.Metadata["retrySelectedPaths"].([]string); !ok || len(selectedPaths) != 1 || selectedPaths[0] != "/pending-b.bin" {
		t.Fatalf("expected retrySelectedPaths [/pending-b.bin], got %#v", retried.Plan.Metadata["retrySelectedPaths"])
	}

	secondRun, ok, err := svc.Run(ctx, detail.Task.ID)
	if err != nil || !ok {
		t.Fatalf("Run() retried error=%v ok=%v", err, ok)
	}
	if len(secondRun.Results) != 1 || stringValue(secondRun.Results[0].Payload["path"]) != "/pending-b.bin" {
		t.Fatalf("expected second run to execute only /pending-b.bin, got %#v", secondRun.Results)
	}
	if secondRun.Task.State != StateCompleted {
		t.Fatalf("expected completed after selected pending retry, got %s", secondRun.Task.State)
	}
}

func TestServiceRetryWithOptionsSelectedDirectorySubsetKeepsOnlyChosenSubtree(t *testing.T) {
	ctx := context.Background()
	store, err := sqlitestore.New(ctx, filepath.Join(t.TempDir(), "retry-selected-directory.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer func() { _ = store.Close() }()

	adapter := &scriptedAdapter{
		meta: provider.Provider{
			Key:              "retry_selected_directory_target",
			DisplayName:      "Retry Selected Directory Target",
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
			return provider.UploadResult{
				OperationResult: provider.OperationResult{
					OK:      true,
					Status:  "ok",
					Message: "selected directory resolved",
					Mode:    "fake_selected_directory_ok",
				},
			}
		},
	}

	registry := provider.NewRegistry(adapter)
	authSvc := auth.NewService(store, registry)
	svc := NewService(store, registry, authSvc)
	profile, err := authSvc.CreateProfile(ctx, auth.CreateProfileInput{
		ProviderKey: "retry_selected_directory_target",
		AuthMode:    "manual_token",
		DisplayName: "retry selected directory target",
		Token:       "token-1",
	})
	if err != nil {
		t.Fatalf("CreateProfile() error = %v", err)
	}

	detail, err := svc.Create(ctx, CreateRequest{
		SourceProvider:  "guangya",
		TargetProvider:  "retry_selected_directory_target",
		TargetProfileID: profile.ID,
		ThresholdMB:     1,
		SelectedRoots:   []string{"/1", "/2"},
		Entries: []planner.SourceEntry{
			{Path: "/1/11/a.bin", Size: 10 * 1024},
			{Path: "/1/12/b.bin", Size: 11 * 1024},
			{Path: "/2/21/c.bin", Size: 12 * 1024},
		},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if _, ok, err := svc.Run(ctx, detail.Task.ID); err != nil || !ok {
		t.Fatalf("Run() error=%v ok=%v", err, ok)
	}

	retried, ok, err := svc.RetryWithOptions(ctx, detail.Task.ID, RetryOptions{
		Paths: []string{"/1/11"},
		Scope: "selected_directory_subset",
	})
	if err != nil || !ok {
		t.Fatalf("RetryWithOptions() error=%v ok=%v", err, ok)
	}
	if len(retried.Plan.Items) != 1 || retried.Plan.Items[0].Path != "/1/11/a.bin" {
		t.Fatalf("expected selected directory retry to keep only /1/11/a.bin, got %#v", retried.Plan.Items)
	}
	if len(retried.SourceEntries) != 1 || retried.SourceEntries[0].Path != "/1/11/a.bin" {
		t.Fatalf("expected selected directory entries to keep only /1/11/a.bin, got %#v", retried.SourceEntries)
	}
	if retryMode, _ := retried.Plan.Metadata["retryMode"].(string); retryMode != "selected_directory_subset" {
		t.Fatalf("expected retryMode selected_directory_subset, got %#v", retried.Plan.Metadata["retryMode"])
	}
	if retryPendingOnly, _ := retried.Plan.Metadata["retryPendingOnly"].(bool); retryPendingOnly {
		t.Fatalf("expected retryPendingOnly false for selected_directory_subset, got %#v", retried.Plan.Metadata["retryPendingOnly"])
	}
	if selectedPaths, ok := retried.Plan.Metadata["retrySelectedPaths"].([]string); !ok || len(selectedPaths) != 1 || selectedPaths[0] != "/1/11" {
		t.Fatalf("expected retrySelectedPaths [/1/11], got %#v", retried.Plan.Metadata["retrySelectedPaths"])
	}
}

func TestServiceRetryWithOptionsSelectedRetrySubsetKeepsOnlyChosenPaths(t *testing.T) {
	ctx := context.Background()
	store, err := sqlitestore.New(ctx, filepath.Join(t.TempDir(), "retry-selected-queue.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer func() { _ = store.Close() }()

	uploadCalls := []string{}
	adapter := &scriptedAdapter{
		meta: provider.Provider{
			Key:              "retry_selected_queue_target",
			DisplayName:      "Retry Selected Queue Target",
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
			uploadCalls = append(uploadCalls, req.Path)
			if len(uploadCalls) <= 2 {
				return provider.UploadResult{
					OperationResult: provider.OperationResult{
						Status:  "remote_error",
						Message: "retry later",
						Mode:    "fake_remote_error",
					},
				}
			}
			return provider.UploadResult{
				OperationResult: provider.OperationResult{
					OK:      true,
					Status:  "ok",
					Message: "selected queue resolved",
					Mode:    "fake_selected_queue_ok",
				},
			}
		},
	}

	registry := provider.NewRegistry(adapter)
	authSvc := auth.NewService(store, registry)
	svc := NewService(store, registry, authSvc)
	profile, err := authSvc.CreateProfile(ctx, auth.CreateProfileInput{
		ProviderKey: "retry_selected_queue_target",
		AuthMode:    "manual_token",
		DisplayName: "retry selected queue target",
		Token:       "token-1",
	})
	if err != nil {
		t.Fatalf("CreateProfile() error = %v", err)
	}

	fileA := filepath.Join(t.TempDir(), "retry-a.bin")
	fileB := filepath.Join(t.TempDir(), "retry-b.bin")
	if err := os.WriteFile(fileA, []byte("a"), 0o644); err != nil {
		t.Fatalf("WriteFile(fileA) error = %v", err)
	}
	if err := os.WriteFile(fileB, []byte("b"), 0o644); err != nil {
		t.Fatalf("WriteFile(fileB) error = %v", err)
	}

	detail, err := svc.Create(ctx, CreateRequest{
		SourceProvider:  "guangya",
		TargetProvider:  "retry_selected_queue_target",
		TargetProfileID: profile.ID,
		ThresholdMB:     1,
		Entries: []planner.SourceEntry{
			{Path: "/retry-a.bin", Size: 1024, MD5: "md5-a", LocalPath: fileA},
			{Path: "/retry-b.bin", Size: 2048, MD5: "md5-b", LocalPath: fileB},
		},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	firstRun, ok, err := svc.Run(ctx, detail.Task.ID)
	if err != nil || !ok {
		t.Fatalf("Run() error=%v ok=%v", err, ok)
	}
	if firstRun.Task.State != StateCompletedWithErrors {
		t.Fatalf("expected completed_with_errors, got %s", firstRun.Task.State)
	}
	if len(firstRun.Runtime.RetryQueue) != 2 {
		t.Fatalf("expected retry queue len 2, got %#v", firstRun.Runtime.RetryQueue)
	}

	retried, ok, err := svc.RetryWithOptions(ctx, detail.Task.ID, RetryOptions{
		Paths: []string{"/retry-b.bin"},
		Scope: "selected_retry_subset",
	})
	if err != nil || !ok {
		t.Fatalf("RetryWithOptions() error=%v ok=%v", err, ok)
	}
	if len(retried.Plan.Items) != 1 || retried.Plan.Items[0].Path != "/retry-b.bin" {
		t.Fatalf("expected selected queue retry to keep only /retry-b.bin, got %#v", retried.Plan.Items)
	}
	if retryMode, _ := retried.Plan.Metadata["retryMode"].(string); retryMode != "selected_retry_subset" {
		t.Fatalf("expected retryMode selected_retry_subset, got %#v", retried.Plan.Metadata["retryMode"])
	}
	if retryPendingOnly, _ := retried.Plan.Metadata["retryPendingOnly"].(bool); retryPendingOnly {
		t.Fatalf("expected retryPendingOnly false for selected retry queue, got %#v", retried.Plan.Metadata["retryPendingOnly"])
	}

	secondRun, ok, err := svc.Run(ctx, detail.Task.ID)
	if err != nil || !ok {
		t.Fatalf("Run() retried error=%v ok=%v", err, ok)
	}
	if len(secondRun.Results) != 1 || stringValue(secondRun.Results[0].Payload["path"]) != "/retry-b.bin" {
		t.Fatalf("expected second run to execute only /retry-b.bin, got %#v", secondRun.Results)
	}
	if secondRun.Task.State != StateCompleted {
		t.Fatalf("expected completed after selected retry queue retry, got %s", secondRun.Task.State)
	}
}

func TestServiceRetryWithOptionsSelectedRetrySubsetKeepsCheckpointContext(t *testing.T) {
	ctx := context.Background()
	store, err := sqlitestore.New(ctx, filepath.Join(t.TempDir(), "retry-selected-checkpoint.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer func() { _ = store.Close() }()

	type uploadCall struct {
		path   string
		resume *provider.ResumeUpload
	}
	uploadCalls := make([]uploadCall, 0, 4)
	adapter := &scriptedAdapter{
		meta: provider.Provider{
			Key:              "retry_selected_checkpoint_target",
			DisplayName:      "Retry Selected Checkpoint Target",
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
			var resumeCopy *provider.ResumeUpload
			if req.ResumeUpload != nil {
				copied := *req.ResumeUpload
				resumeCopy = &copied
			}
			uploadCalls = append(uploadCalls, uploadCall{path: req.Path, resume: resumeCopy})
			if len(uploadCalls) <= 2 {
				return provider.UploadResult{
					OperationResult: provider.OperationResult{
						Status:  "provider_request_failed",
						Message: "checkpoint interrupted",
						Mode:    "fake_checkpoint_failed",
						Payload: map[string]interface{}{
							"fileId":           "file-" + strings.TrimPrefix(req.Path, "/"),
							"uploadId":         "upload-" + strings.TrimPrefix(req.Path, "/"),
							"nextPartNumber":   2,
							"failedPartNumber": 2,
							"providerData": map[string]interface{}{
								"resumable": map[string]interface{}{
									"path": req.Path,
								},
							},
						},
					},
				}
			}
			return provider.UploadResult{
				OperationResult: provider.OperationResult{
					OK:      true,
					Status:  "ok",
					Message: "selected checkpoint resolved",
					Mode:    "fake_checkpoint_resume_ok",
				},
			}
		},
	}

	registry := provider.NewRegistry(adapter)
	authSvc := auth.NewService(store, registry)
	svc := NewService(store, registry, authSvc)
	profile, err := authSvc.CreateProfile(ctx, auth.CreateProfileInput{
		ProviderKey: "retry_selected_checkpoint_target",
		AuthMode:    "manual_token",
		DisplayName: "retry selected checkpoint target",
		Token:       "token-1",
	})
	if err != nil {
		t.Fatalf("CreateProfile() error = %v", err)
	}

	fileA := filepath.Join(t.TempDir(), "retry-checkpoint-a.bin")
	fileB := filepath.Join(t.TempDir(), "retry-checkpoint-b.bin")
	if err := os.WriteFile(fileA, []byte("a"), 0o644); err != nil {
		t.Fatalf("WriteFile(fileA) error = %v", err)
	}
	if err := os.WriteFile(fileB, []byte("b"), 0o644); err != nil {
		t.Fatalf("WriteFile(fileB) error = %v", err)
	}

	detail, err := svc.Create(ctx, CreateRequest{
		SourceProvider:  "guangya",
		TargetProvider:  "retry_selected_checkpoint_target",
		TargetProfileID: profile.ID,
		ThresholdMB:     1,
		Entries: []planner.SourceEntry{
			{Path: "/retry-a.bin", Size: 1024, MD5: "md5-a", LocalPath: fileA},
			{Path: "/retry-b.bin", Size: 2048, MD5: "md5-b", LocalPath: fileB},
		},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	firstRun, ok, err := svc.Run(ctx, detail.Task.ID)
	if err != nil || !ok {
		t.Fatalf("Run() error=%v ok=%v", err, ok)
	}
	if firstRun.Task.State != StateCompletedWithErrors {
		t.Fatalf("expected completed_with_errors, got %s", firstRun.Task.State)
	}
	if len(firstRun.Runtime.RetryQueue) != 2 {
		t.Fatalf("expected retry queue len 2, got %#v", firstRun.Runtime.RetryQueue)
	}

	retried, ok, err := svc.RetryWithOptions(ctx, detail.Task.ID, RetryOptions{
		Paths: []string{"/retry-b.bin"},
		Scope: "selected_retry_subset",
	})
	if err != nil || !ok {
		t.Fatalf("RetryWithOptions() error=%v ok=%v", err, ok)
	}
	if len(retried.Plan.Items) != 1 || retried.Plan.Items[0].Path != "/retry-b.bin" {
		t.Fatalf("expected selected queue retry to keep only /retry-b.bin, got %#v", retried.Plan.Items)
	}
	checkpoints, ok := retried.Plan.Metadata["retryUploadCheckpoints"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected retryUploadCheckpoints metadata, got %#v", retried.Plan.Metadata["retryUploadCheckpoints"])
	}
	if len(checkpoints) != 1 {
		t.Fatalf("expected one filtered checkpoint, got %#v", checkpoints)
	}
	if _, exists := checkpoints["/retry-a.bin"]; exists {
		t.Fatalf("expected /retry-a.bin checkpoint to be excluded, got %#v", checkpoints)
	}
	checkpoint := uploadCheckpointFromMetadata(checkpoints["/retry-b.bin"])
	if checkpoint == nil || checkpoint.ItemPath != "/retry-b.bin" {
		t.Fatalf("expected /retry-b.bin checkpoint metadata, got %#v", checkpoints["/retry-b.bin"])
	}
	if retried.Runtime.UploadCheckpoint == nil || retried.Runtime.UploadCheckpoint.ItemPath != "/retry-b.bin" {
		t.Fatalf("expected runtime upload checkpoint for /retry-b.bin, got %#v", retried.Runtime.UploadCheckpoint)
	}

	secondRun, ok, err := svc.Run(ctx, detail.Task.ID)
	if err != nil || !ok {
		t.Fatalf("Run() retried error=%v ok=%v", err, ok)
	}
	if len(secondRun.Results) != 1 || stringValue(secondRun.Results[0].Payload["path"]) != "/retry-b.bin" {
		t.Fatalf("expected second run to execute only /retry-b.bin, got %#v", secondRun.Results)
	}
	if secondRun.Task.State != StateCompleted {
		t.Fatalf("expected completed after selected retry queue retry, got %s", secondRun.Task.State)
	}
	if len(uploadCalls) != 3 {
		t.Fatalf("expected exactly 3 upload calls, got %#v", uploadCalls)
	}
	if uploadCalls[2].path != "/retry-b.bin" {
		t.Fatalf("expected resumed call for /retry-b.bin, got %#v", uploadCalls[2])
	}
	if uploadCalls[2].resume == nil {
		t.Fatalf("expected resumed call to carry checkpoint context, got %#v", uploadCalls[2])
	}
	if uploadCalls[2].resume.ItemPath != "/retry-b.bin" || uploadCalls[2].resume.UploadID != "upload-retry-b.bin" {
		t.Fatalf("expected resumed call to target /retry-b.bin checkpoint, got %#v", uploadCalls[2].resume)
	}
}

func TestServiceAppliesRiskProfileRuntimeThrottle(t *testing.T) {
	ctx := context.Background()
	store, err := sqlitestore.New(ctx, filepath.Join(t.TempDir(), "runtime-throttle.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer func() { _ = store.Close() }()

	uploaded := []string{}
	target := &scriptedAdapter{
		meta: provider.Provider{
			Key:              "throttle_target",
			DisplayName:      "Throttle Target",
			FastUploadInputs: []string{"md5"},
			FallbackModes:    []string{"download_upload"},
			ConflictPolicies: []provider.ConflictPolicy{provider.ConflictPolicyAutoRenameNew},
		},
		capability: provider.CapabilitySet{
			SupportsAuthValidation: true,
			SupportsMetadata:       true,
			SupportsUpload:         true,
		},
		uploadFunc: func(req provider.UploadRequest) provider.UploadResult {
			uploaded = append(uploaded, req.Path)
			return provider.UploadResult{OperationResult: provider.OperationResult{OK: true, Status: "ok", Message: "ok", Mode: "scripted_upload"}}
		},
	}
	registry := provider.NewRegistry(target)
	authSvc := auth.NewService(store, registry)
	svc := NewService(store, registry, authSvc)
	sleeps := []time.Duration{}
	svc.throttleSleep = func(ctx context.Context, delay time.Duration) error {
		sleeps = append(sleeps, delay)
		return nil
	}

	localDir := t.TempDir()
	localA := filepath.Join(localDir, "a.bin")
	localB := filepath.Join(localDir, "b.bin")
	localC := filepath.Join(localDir, "c.bin")
	for _, path := range []string{localA, localB, localC} {
		if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
			t.Fatalf("write local file: %v", err)
		}
	}

	detail, err := svc.Create(ctx, CreateRequest{
		SourceProvider: "source",
		TargetProvider: "throttle_target",
		ThresholdMB:    1,
		RiskMode:       planner.RiskModeCustom,
		RiskOverride: &planner.RiskProfileOverride{
			RequestIntervalMS:   intPtrTask(7),
			DirectoryIntervalMS: intPtrTask(31),
		},
		ExecutionMode: planner.ExecutionModePreScanFlat,
		Entries: []planner.SourceEntry{
			{Path: "/a/one.bin", Size: 1, LocalPath: localA},
			{Path: "/a/two.bin", Size: 1, LocalPath: localB},
			{Path: "/b/three.bin", Size: 1, LocalPath: localC},
		},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	running, ok, err := svc.Run(ctx, detail.Task.ID)
	if err != nil || !ok {
		t.Fatalf("Run() error=%v ok=%v", err, ok)
	}
	if len(uploaded) != 3 {
		t.Fatalf("expected 3 uploads, got %#v", uploaded)
	}
	wantSleeps := []time.Duration{7 * time.Millisecond, 31 * time.Millisecond}
	if len(sleeps) != len(wantSleeps) {
		t.Fatalf("expected sleeps %#v, got %#v", wantSleeps, sleeps)
	}
	for idx, want := range wantSleeps {
		if sleeps[idx] != want {
			t.Fatalf("sleep %d expected %v, got %v", idx, want, sleeps[idx])
		}
	}
	if _, exists := running.Results[0].Payload["throttle"]; exists {
		t.Fatalf("did not expect throttle evidence on first item, got %+v", running.Results[0].Payload)
	}
	secondThrottle, _ := running.Results[1].Payload["throttle"].(map[string]interface{})
	if intNumber(secondThrottle["waitMs"]) != 7 {
		t.Fatalf("expected second throttle waitMs 7, got %+v", secondThrottle)
	}
	thirdThrottle, _ := running.Results[2].Payload["throttle"].(map[string]interface{})
	if intNumber(thirdThrottle["waitMs"]) != 31 || !boolMapValue(thirdThrottle, "directoryChanged") {
		t.Fatalf("expected directory throttle waitMs 31, got %+v", thirdThrottle)
	}
}

func TestServiceRetryQueueHonorsCooldownForRateLimitedItems(t *testing.T) {
	ctx := context.Background()
	store, err := sqlitestore.New(ctx, filepath.Join(t.TempDir(), "retry-cooldown.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer func() { _ = store.Close() }()

	adapter := &scriptedAdapter{
		meta: provider.Provider{
			Key:              "retry_cooldown_target",
			DisplayName:      "Retry Cooldown Target",
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
			return provider.UploadResult{
				OperationResult: provider.OperationResult{
					Status:  "rate_limited",
					Message: "rate limited",
					Mode:    "fake_rate",
				},
			}
		},
	}

	registry := provider.NewRegistry(adapter)
	authSvc := auth.NewService(store, registry)
	svc := NewService(store, registry, authSvc)
	profile, err := authSvc.CreateProfile(ctx, auth.CreateProfileInput{
		ProviderKey: "retry_cooldown_target",
		AuthMode:    "manual_token",
		DisplayName: "retry cooldown target",
		Token:       "token-1",
	})
	if err != nil {
		t.Fatalf("CreateProfile() error = %v", err)
	}

	detail, err := svc.Create(ctx, CreateRequest{
		SourceProvider:  "guangya",
		TargetProvider:  "retry_cooldown_target",
		TargetProfileID: profile.ID,
		ThresholdMB:     1,
		RiskOverride: &planner.RiskProfileOverride{
			CooldownSeconds: intPtrTask(3600),
		},
		Entries: []planner.SourceEntry{
			{Path: "/rate.bin", Size: 1024, MD5: "abc"},
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
	if len(running.Runtime.RetryQueue) != 1 {
		t.Fatalf("expected retry queue len 1, got %d", len(running.Runtime.RetryQueue))
	}
	if running.Runtime.RetryQueue[0].RetryClass != "rate_limited" {
		t.Fatalf("expected retry class rate_limited, got %s", running.Runtime.RetryQueue[0].RetryClass)
	}
	if running.Runtime.RetryQueue[0].CooldownTier != "fast" {
		t.Fatalf("expected cooldown tier fast, got %s", running.Runtime.RetryQueue[0].CooldownTier)
	}
	if running.Runtime.RetryQueue[0].CooldownSeconds != 3600 {
		t.Fatalf("expected cooldown seconds 3600, got %d", running.Runtime.RetryQueue[0].CooldownSeconds)
	}
	if running.Runtime.RetryQueue[0].EligibleAt == "" {
		t.Fatal("expected eligibleAt for rate-limited retry item")
	}
	if _, ok, err := svc.Retry(ctx, detail.Task.ID); err == nil || !ok {
		t.Fatalf("expected retry cooldown error with task present, ok=%v err=%v", ok, err)
	} else if !strings.HasPrefix(err.Error(), "retry_cooldown_active:") {
		t.Fatalf("expected retry_cooldown_active error, got %v", err)
	}
	if running.Task.State != StateBlocked {
		t.Fatalf("expected blocked state for cooldown-only retry queue, got %s", running.Task.State)
	}
	if running.Runtime.BlockedReason != "retry_queue_waiting_for_cooldown" {
		t.Fatalf("expected cooldown blocked reason, got %s", running.Runtime.BlockedReason)
	}
	if running.Runtime.BlockedAction != "wait_for_cooldown" {
		t.Fatalf("expected cooldown blocked action, got %s", running.Runtime.BlockedAction)
	}
	if running.Runtime.BlockedAdvice == "" {
		t.Fatal("expected blocked advice on cooldown runtime")
	}
	if running.Runtime.NextRetryAt == "" {
		t.Fatal("expected nextRetryAt on blocked runtime")
	}
	retrySummary, ok := running.Plan.Metadata["retrySummary"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected retrySummary metadata, got %#v", running.Plan.Metadata["retrySummary"])
	}
	if stringValue(retrySummary["autoRecoverMode"]) != "cooldown_elapsed_auto_retry" {
		t.Fatalf("expected cooldown auto recover mode, got %#v", retrySummary)
	}
	if intNumber(retrySummary["cooldownCount"]) != 1 {
		t.Fatalf("expected cooldownCount 1, got %#v", retrySummary)
	}
	if !boolValue(retrySummary["autoRecoverEligible"]) {
		t.Fatalf("expected autoRecoverEligible for cooldown queue, got %#v", retrySummary)
	}
}

func TestResolveCooldownBackoffUsesTieredMinimums(t *testing.T) {
	tests := []struct {
		name            string
		attemptCount    int
		cooldownSeconds int
		wantTier        string
		wantSeconds     int
	}{
		{
			name:            "fast tier uses minimum cooldown",
			attemptCount:    0,
			cooldownSeconds: 0,
			wantTier:        "fast",
			wantSeconds:     30,
		},
		{
			name:            "fast tier preserves larger override",
			attemptCount:    2,
			cooldownSeconds: 3600,
			wantTier:        "fast",
			wantSeconds:     3600,
		},
		{
			name:            "normal tier raises short cooldown",
			attemptCount:    3,
			cooldownSeconds: 45,
			wantTier:        "normal",
			wantSeconds:     300,
		},
		{
			name:            "extended tier raises short cooldown",
			attemptCount:    6,
			cooldownSeconds: 300,
			wantTier:        "extended",
			wantSeconds:     1800,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tier, seconds := resolveCooldownBackoff(tt.attemptCount, tt.cooldownSeconds)
			if tier != tt.wantTier {
				t.Fatalf("expected tier %s, got %s", tt.wantTier, tier)
			}
			if seconds != tt.wantSeconds {
				t.Fatalf("expected cooldown seconds %d, got %d", tt.wantSeconds, seconds)
			}
		})
	}
}

func TestServiceRecoverBlockedTasksRetriesEligibleCooldownQueue(t *testing.T) {
	ctx := context.Background()
	store, err := sqlitestore.New(ctx, filepath.Join(t.TempDir(), "recover-blocked.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer func() { _ = store.Close() }()

	uploadCalls := 0
	adapter := &scriptedAdapter{
		meta: provider.Provider{
			Key:              "recover_blocked_target",
			DisplayName:      "Recover Blocked Target",
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
			uploadCalls++
			if uploadCalls == 1 {
				return provider.UploadResult{
					OperationResult: provider.OperationResult{
						Status:  "rate_limited",
						Message: "rate limited",
						Mode:    "fake_rate",
					},
				}
			}
			return provider.UploadResult{
				OperationResult: provider.OperationResult{
					OK:      true,
					Status:  "ok",
					Message: "recovered",
					Mode:    "fake_ok",
				},
			}
		},
	}

	registry := provider.NewRegistry(adapter)
	authSvc := auth.NewService(store, registry)
	svc := NewService(store, registry, authSvc)
	profile, err := authSvc.CreateProfile(ctx, auth.CreateProfileInput{
		ProviderKey: "recover_blocked_target",
		AuthMode:    "manual_token",
		DisplayName: "recover blocked target",
		Token:       "token-1",
	})
	if err != nil {
		t.Fatalf("CreateProfile() error = %v", err)
	}

	created, err := svc.Create(ctx, CreateRequest{
		SourceProvider:  "guangya",
		TargetProvider:  "recover_blocked_target",
		TargetProfileID: profile.ID,
		ThresholdMB:     1,
		RiskOverride: &planner.RiskProfileOverride{
			CooldownSeconds: intPtrTask(3600),
		},
		Entries: []planner.SourceEntry{
			{Path: "/rate.bin", Size: 1024, MD5: "abc"},
		},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	running, ok, err := svc.Run(ctx, created.Task.ID)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !ok {
		t.Fatal("expected task to exist")
	}
	if running.Task.State != StateBlocked {
		t.Fatalf("expected blocked after first run, got %s", running.Task.State)
	}

	blocked, ok, err := svc.Get(ctx, created.Task.ID)
	if err != nil || !ok {
		t.Fatalf("Get() blocked task error=%v ok=%v", err, ok)
	}
	blocked.Results[0].CreatedAt = time.Now().Add(-2 * time.Hour).UTC().Format(time.RFC3339)
	syncRuntimeRetryQueue(&blocked.Runtime, blocked.Plan.Metadata, blocked.Results)
	applyRetryQueueSummary(&blocked.Runtime, blocked.Plan.Metadata)
	blocked.Task.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	if err := replaceTaskDetailAndResults(ctx, store, blocked); err != nil {
		t.Fatalf("replaceTaskDetailAndResults() error = %v", err)
	}

	recovered, err := svc.RecoverBlockedTasks(ctx)
	if err != nil {
		t.Fatalf("RecoverBlockedTasks() error = %v", err)
	}
	if recovered != 1 {
		t.Fatalf("expected recovered count 1, got %d", recovered)
	}
	finalDetail, ok, err := svc.Get(ctx, created.Task.ID)
	if err != nil || !ok {
		t.Fatalf("Get() final task error=%v ok=%v", err, ok)
	}
	if finalDetail.Task.State != StateCompleted {
		t.Fatalf("expected completed after auto recovery, got %s", finalDetail.Task.State)
	}
	if finalDetail.Runtime.BlockedReason != "" {
		t.Fatalf("expected blocked reason cleared after recovery, got %s", finalDetail.Runtime.BlockedReason)
	}
	if len(finalDetail.Results) != 1 || finalDetail.Results[0].Status != "done" {
		t.Fatalf("expected recovered done result, got %#v", finalDetail.Results)
	}
	if uploadCalls != 2 {
		t.Fatalf("expected 2 upload attempts total, got %d", uploadCalls)
	}
}

func TestServiceRecoverBlockedTasksAutoResumesUploadCheckpointQueue(t *testing.T) {
	ctx := context.Background()
	store, err := sqlitestore.New(ctx, filepath.Join(t.TempDir(), "recover-upload-session.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer func() { _ = store.Close() }()

	uploadCalls := 0
	resumeCalls := 0
	adapter := &scriptedAdapter{
		meta: provider.Provider{
			Key:              "recover_upload_session_target",
			DisplayName:      "Recover Upload Session Target",
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
			uploadCalls++
			if req.ResumeUpload != nil {
				resumeCalls++
				return provider.UploadResult{
					OperationResult: provider.OperationResult{
						OK:      true,
						Status:  "ok",
						Message: "resumed upload session",
						Mode:    "fake_resume_ok",
						Payload: map[string]interface{}{
							"fileId":       req.ResumeUpload.FileID,
							"uploadId":     req.ResumeUpload.UploadID,
							"providerData": req.ResumeUpload.ProviderData,
						},
					},
				}
			}
			return provider.UploadResult{
				OperationResult: provider.OperationResult{
					Status:  "provider_request_failed",
					Message: "upload session interrupted",
					Mode:    "fake_upload_failed",
					Payload: map[string]interface{}{
						"fileId":           "resume-file-1",
						"uploadId":         "resume-upload-1",
						"nextPartNumber":   3,
						"failedPartNumber": 3,
						"providerData": map[string]interface{}{
							"resumable": map[string]interface{}{
								"provider": "oss",
								"key":      "resume-key",
							},
						},
					},
				},
			}
		},
	}

	registry := provider.NewRegistry(adapter)
	authSvc := auth.NewService(store, registry)
	svc := NewService(store, registry, authSvc)
	profile, err := authSvc.CreateProfile(ctx, auth.CreateProfileInput{
		ProviderKey: "recover_upload_session_target",
		AuthMode:    "manual_token",
		DisplayName: "recover upload session target",
		Token:       "token-1",
	})
	if err != nil {
		t.Fatalf("CreateProfile() error = %v", err)
	}

	localFile := filepath.Join(t.TempDir(), "resume-source.bin")
	if err := os.WriteFile(localFile, []byte("resume"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	created, err := svc.Create(ctx, CreateRequest{
		SourceProvider:  "guangya",
		TargetProvider:  "recover_upload_session_target",
		TargetProfileID: profile.ID,
		ThresholdMB:     1,
		Entries: []planner.SourceEntry{
			{Path: "/resume.bin", Size: 1024, MD5: "abc", LocalPath: localFile},
		},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	firstRun, ok, err := svc.Run(ctx, created.Task.ID)
	if err != nil || !ok {
		t.Fatalf("Run() first error=%v ok=%v", err, ok)
	}
	if firstRun.Task.State != StateCompletedWithErrors {
		t.Fatalf("expected completed_with_errors after first run, got %s", firstRun.Task.State)
	}
	if len(firstRun.Runtime.RetryQueue) != 1 {
		t.Fatalf("expected retry queue len 1, got %#v", firstRun.Runtime.RetryQueue)
	}
	if firstRun.Runtime.RetryQueue[0].UploadCheckpoint == nil {
		t.Fatalf("expected upload checkpoint in retry queue, got %#v", firstRun.Runtime.RetryQueue[0])
	}
	retrySummary, ok := firstRun.Plan.Metadata["retrySummary"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected retrySummary metadata, got %#v", firstRun.Plan.Metadata["retrySummary"])
	}
	if stringValue(retrySummary["autoRecoverMode"]) != "upload_checkpoint_auto_resume" {
		t.Fatalf("expected upload checkpoint auto recover mode, got %#v", retrySummary)
	}
	if intNumber(retrySummary["uploadCheckpointEligible"]) != 1 {
		t.Fatalf("expected uploadCheckpointEligible 1, got %#v", retrySummary)
	}
	if !boolValue(retrySummary["autoRecoverEligible"]) {
		t.Fatalf("expected autoRecoverEligible for upload checkpoint queue, got %#v", retrySummary)
	}

	recovered, err := svc.RecoverBlockedTasks(ctx)
	if err != nil {
		t.Fatalf("RecoverBlockedTasks() error = %v", err)
	}
	if recovered != 1 {
		t.Fatalf("expected recovered count 1, got %d", recovered)
	}

	finalDetail, ok, err := svc.Get(ctx, created.Task.ID)
	if err != nil || !ok {
		t.Fatalf("Get() final task error=%v ok=%v", err, ok)
	}
	if finalDetail.Task.State != StateCompleted {
		t.Fatalf("expected completed after auto resume recovery, got %s", finalDetail.Task.State)
	}
	if !finalDetail.Runtime.AutoRecovered {
		t.Fatalf("expected runtime auto recovery evidence, got %#v", finalDetail.Runtime)
	}
	if finalDetail.Runtime.AutoRecoverReason != "upload_checkpoint_auto_resume" {
		t.Fatalf("expected upload checkpoint auto recovery reason, got %q", finalDetail.Runtime.AutoRecoverReason)
	}
	if finalDetail.Runtime.AutoRecoverCount != 1 {
		t.Fatalf("expected auto recover count 1, got %d", finalDetail.Runtime.AutoRecoverCount)
	}
	if finalDetail.Runtime.AutoRecoverState != string(StateCompletedWithErrors) {
		t.Fatalf("expected auto recover source state completed_with_errors, got %q", finalDetail.Runtime.AutoRecoverState)
	}
	if len(finalDetail.Results) != 1 || finalDetail.Results[0].Mode != "fake_resume_ok" {
		t.Fatalf("expected resumed result, got %#v", finalDetail.Results)
	}
	autoRecoveryPayload, ok := finalDetail.Results[0].Payload["autoRecovery"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected result auto recovery payload, got %#v", finalDetail.Results[0].Payload)
	}
	if stringValue(autoRecoveryPayload["reason"]) != "upload_checkpoint_auto_resume" {
		t.Fatalf("expected result auto recovery reason, got %#v", autoRecoveryPayload)
	}
	if intNumber(autoRecoveryPayload["count"]) != 1 {
		t.Fatalf("expected result auto recover count 1, got %#v", autoRecoveryPayload)
	}
	evidence, err := svc.RuntimeEvidence(ctx)
	if err != nil {
		t.Fatalf("RuntimeEvidence() error = %v", err)
	}
	if len(evidence.RecentProbes) == 0 {
		t.Fatal("expected recent provider probe after auto recovery")
	}
	if evidence.UploadCheckpointTaskCount != 1 {
		t.Fatalf("expected upload checkpoint task count 1, got %d", evidence.UploadCheckpointTaskCount)
	}
	if evidence.UploadCheckpointResumeTaskCount != 1 {
		t.Fatalf("expected upload checkpoint resume task count 1, got %d", evidence.UploadCheckpointResumeTaskCount)
	}
	if len(evidence.UploadCheckpointResumeSamplePaths) != 1 || evidence.UploadCheckpointResumeSamplePaths[0] != "/resume.bin" {
		t.Fatalf("expected upload checkpoint resume sample path /resume.bin, got %#v", evidence.UploadCheckpointResumeSamplePaths)
	}
	if evidence.UploadCheckpointResumeSampleProvider != "recover_upload_session_target" {
		t.Fatalf("expected upload checkpoint resume sample provider recover_upload_session_target, got %q", evidence.UploadCheckpointResumeSampleProvider)
	}
	if evidence.UploadCheckpointResumeSampleProtocol != "fake" {
		t.Fatalf("expected upload checkpoint resume sample protocol fake, got %q", evidence.UploadCheckpointResumeSampleProtocol)
	}
	if evidence.UploadCheckpointResumeSampleTaskID != finalDetail.Task.ID {
		t.Fatalf("expected upload checkpoint resume sample task %s, got %q", finalDetail.Task.ID, evidence.UploadCheckpointResumeSampleTaskID)
	}
	if evidence.UploadCheckpointResumeSampleProfileID != profile.ID {
		t.Fatalf("expected upload checkpoint resume sample profile %s, got %q", profile.ID, evidence.UploadCheckpointResumeSampleProfileID)
	}
	if evidence.UploadCheckpointResumeSampleUploadID != "resume-upload-1" {
		t.Fatalf("expected upload checkpoint resume sample upload id resume-upload-1, got %q", evidence.UploadCheckpointResumeSampleUploadID)
	}
	if evidence.UploadCheckpointResumeSampleNextPart != 3 {
		t.Fatalf("expected upload checkpoint resume sample next part 3, got %d", evidence.UploadCheckpointResumeSampleNextPart)
	}
	if evidence.UploadCheckpointResumeSamplePartCount != 0 {
		t.Fatalf("expected upload checkpoint resume sample part count 0, got %d", evidence.UploadCheckpointResumeSamplePartCount)
	}
	if evidence.UploadCheckpointResumeSampleUploaded != 0 {
		t.Fatalf("expected upload checkpoint resume sample uploaded count 0, got %d", evidence.UploadCheckpointResumeSampleUploaded)
	}
	report, err := svc.EvidenceReport(ctx)
	if err != nil {
		t.Fatalf("EvidenceReport() error = %v", err)
	}
	if !strings.Contains(report.Markdown, "Upload checkpoint 稳定性摘要") {
		t.Fatalf("expected upload checkpoint stability summary in report markdown, got %s", report.Markdown)
	}
	if !strings.Contains(report.Markdown, "已具备从既有 upload checkpoint 继续恢复的关键证据") {
		t.Fatalf("expected upload checkpoint stability summary to mention resumable evidence, got %s", report.Markdown)
	}
	if !strings.Contains(report.Markdown, "当前还没有可用于判断公平性的自动补传候选池样本") && !strings.Contains(report.Markdown, "当前没有自动补传候选池数据") && (!strings.Contains(report.Markdown, "公平性") || !strings.Contains(report.Markdown, "lane")) {
		t.Fatalf("expected fairness completion summary in report markdown, got %s", report.Markdown)
	}
	if stringValue(evidence.RecentProbes[0].Payload["autoRecoverReason"]) != "upload_checkpoint_auto_resume" {
		t.Fatalf("expected probe auto recovery reason, got %#v", evidence.RecentProbes[0].Payload)
	}
	statuses, err := svc.ProviderStatuses(ctx)
	if err != nil {
		t.Fatalf("ProviderStatuses() error = %v", err)
	}
	var foundStatus bool
	for _, status := range statuses {
		if status.ProviderKey != "recover_upload_session_target" {
			continue
		}
		foundStatus = true
		if stringValue(status.SnapshotSummary["autoRecoverReason"]) != "upload_checkpoint_auto_resume" {
			t.Fatalf("expected status auto recovery reason, got %#v", status.SnapshotSummary)
		}
	}
	if !foundStatus {
		t.Fatal("expected provider status for recover_upload_session_target")
	}
	if uploadCalls != 2 {
		t.Fatalf("expected 2 upload attempts total, got %d", uploadCalls)
	}
	if resumeCalls != 1 {
		t.Fatalf("expected 1 resume upload call, got %d", resumeCalls)
	}
}

func TestServiceRecoverBlockedTasksDoesNotAutoRetryGenericCompletedErrors(t *testing.T) {
	ctx := context.Background()
	store, err := sqlitestore.New(ctx, filepath.Join(t.TempDir(), "recover-generic-error.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer func() { _ = store.Close() }()

	uploadCalls := 0
	adapter := &scriptedAdapter{
		meta: provider.Provider{
			Key:              "recover_generic_error_target",
			DisplayName:      "Recover Generic Error Target",
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
			uploadCalls++
			return provider.UploadResult{
				OperationResult: provider.OperationResult{
					Status:  "remote_error",
					Message: "generic retryable remote error",
					Mode:    "fake_remote_error",
				},
			}
		},
	}

	registry := provider.NewRegistry(adapter)
	authSvc := auth.NewService(store, registry)
	svc := NewService(store, registry, authSvc)
	profile, err := authSvc.CreateProfile(ctx, auth.CreateProfileInput{
		ProviderKey: "recover_generic_error_target",
		AuthMode:    "manual_token",
		DisplayName: "recover generic error target",
		Token:       "token-1",
	})
	if err != nil {
		t.Fatalf("CreateProfile() error = %v", err)
	}

	created, err := svc.Create(ctx, CreateRequest{
		SourceProvider:  "guangya",
		TargetProvider:  "recover_generic_error_target",
		TargetProfileID: profile.ID,
		ThresholdMB:     1,
		Entries: []planner.SourceEntry{
			{Path: "/generic.bin", Size: 1024, MD5: "abc"},
		},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	firstRun, ok, err := svc.Run(ctx, created.Task.ID)
	if err != nil || !ok {
		t.Fatalf("Run() first error=%v ok=%v", err, ok)
	}
	if firstRun.Task.State != StateCompletedWithErrors {
		t.Fatalf("expected completed_with_errors after first run, got %s", firstRun.Task.State)
	}

	recovered, err := svc.RecoverBlockedTasks(ctx)
	if err != nil {
		t.Fatalf("RecoverBlockedTasks() error = %v", err)
	}
	if recovered != 0 {
		t.Fatalf("expected recovered count 0, got %d", recovered)
	}

	after, ok, err := svc.Get(ctx, created.Task.ID)
	if err != nil || !ok {
		t.Fatalf("Get() final task error=%v ok=%v", err, ok)
	}
	if after.Task.State != StateCompletedWithErrors {
		t.Fatalf("expected task to remain completed_with_errors, got %s", after.Task.State)
	}
	if uploadCalls != 1 {
		t.Fatalf("expected no extra upload attempts, got %d", uploadCalls)
	}
}

func TestServiceAutoRecoverPoolSummaryAndPriority(t *testing.T) {
	ctx := context.Background()
	store, err := sqlitestore.New(ctx, filepath.Join(t.TempDir(), "auto-recover-pool.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer func() { _ = store.Close() }()

	recoveryOrder := make([]string, 0, 2)
	cooldownUploadCalls := 0
	checkpointUploadCalls := 0
	cooldownAdapter := &scriptedAdapter{
		meta: provider.Provider{
			Key:              "auto_recover_cooldown_target",
			DisplayName:      "Auto Recover Cooldown Target",
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
			cooldownUploadCalls++
			if cooldownUploadCalls == 1 {
				return provider.UploadResult{
					OperationResult: provider.OperationResult{
						Status:  "rate_limited",
						Message: "rate limited",
						Mode:    "fake_rate_limit",
					},
				}
			}
			recoveryOrder = append(recoveryOrder, "cooldown")
			return provider.UploadResult{
				OperationResult: provider.OperationResult{
					OK:      true,
					Status:  "ok",
					Message: "cooldown recovered",
					Mode:    "fake_cooldown_ok",
				},
			}
		},
	}
	checkpointAdapter := &scriptedAdapter{
		meta: provider.Provider{
			Key:              "auto_recover_checkpoint_target",
			DisplayName:      "Auto Recover Checkpoint Target",
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
			checkpointUploadCalls++
			if req.ResumeUpload != nil {
				recoveryOrder = append(recoveryOrder, "checkpoint")
				return provider.UploadResult{
					OperationResult: provider.OperationResult{
						OK:      true,
						Status:  "ok",
						Message: "checkpoint resumed",
						Mode:    "fake_checkpoint_resume_ok",
						Payload: map[string]interface{}{
							"fileId":   req.ResumeUpload.FileID,
							"uploadId": req.ResumeUpload.UploadID,
						},
					},
				}
			}
			return provider.UploadResult{
				OperationResult: provider.OperationResult{
					Status:  "provider_request_failed",
					Message: "checkpoint interrupted",
					Mode:    "fake_checkpoint_failed",
					Payload: map[string]interface{}{
						"fileId":           "checkpoint-file-1",
						"uploadId":         "checkpoint-upload-1",
						"nextPartNumber":   2,
						"failedPartNumber": 2,
					},
				},
			}
		},
	}

	registry := provider.NewRegistry(cooldownAdapter, checkpointAdapter)
	authSvc := auth.NewService(store, registry)
	svc := NewService(store, registry, authSvc)
	cooldownProfile, err := authSvc.CreateProfile(ctx, auth.CreateProfileInput{
		ProviderKey: "auto_recover_cooldown_target",
		AuthMode:    "manual_token",
		DisplayName: "auto recover cooldown target",
		Token:       "token-cooldown",
	})
	if err != nil {
		t.Fatalf("CreateProfile(cooldown) error = %v", err)
	}
	checkpointProfile, err := authSvc.CreateProfile(ctx, auth.CreateProfileInput{
		ProviderKey: "auto_recover_checkpoint_target",
		AuthMode:    "manual_token",
		DisplayName: "auto recover checkpoint target",
		Token:       "token-checkpoint",
	})
	if err != nil {
		t.Fatalf("CreateProfile(checkpoint) error = %v", err)
	}

	cooldownTask, err := svc.Create(ctx, CreateRequest{
		SourceProvider:  "guangya",
		TargetProvider:  "auto_recover_cooldown_target",
		TargetProfileID: cooldownProfile.ID,
		ThresholdMB:     1,
		RiskOverride: &planner.RiskProfileOverride{
			CooldownSeconds: intPtrTask(3600),
		},
		Entries: []planner.SourceEntry{
			{Path: "/cooldown.bin", Size: 1024, MD5: "cooldown-md5"},
		},
	})
	if err != nil {
		t.Fatalf("Create(cooldown) error = %v", err)
	}
	if _, ok, err := svc.Run(ctx, cooldownTask.Task.ID); err != nil || !ok {
		t.Fatalf("Run(cooldown) error=%v ok=%v", err, ok)
	}

	localFile := filepath.Join(t.TempDir(), "checkpoint.bin")
	if err := os.WriteFile(localFile, []byte("checkpoint"), 0o644); err != nil {
		t.Fatalf("WriteFile(checkpoint) error = %v", err)
	}
	checkpointTask, err := svc.Create(ctx, CreateRequest{
		SourceProvider:  "guangya",
		TargetProvider:  "auto_recover_checkpoint_target",
		TargetProfileID: checkpointProfile.ID,
		ThresholdMB:     1,
		Entries: []planner.SourceEntry{
			{Path: "/checkpoint.bin", Size: 2048, MD5: "checkpoint-md5", LocalPath: localFile},
		},
	})
	if err != nil {
		t.Fatalf("Create(checkpoint) error = %v", err)
	}
	if _, ok, err := svc.Run(ctx, checkpointTask.Task.ID); err != nil || !ok {
		t.Fatalf("Run(checkpoint) error=%v ok=%v", err, ok)
	}

	evidence, err := svc.RuntimeEvidence(ctx)
	if err != nil {
		t.Fatalf("RuntimeEvidence() error = %v", err)
	}
	if evidence.AutoRecoverTasks != 2 {
		t.Fatalf("expected autoRecoverTasks 2, got %d", evidence.AutoRecoverTasks)
	}
	if evidence.AutoRecoverRunnableTasks != 1 ||
		evidence.AutoRecoverWaitingCooldownTasks != 1 ||
		evidence.AutoRecoverWaitingRetryWindowTasks != 0 ||
		evidence.AutoRecoverWaitingAuthRefreshTasks != 0 ||
		evidence.AutoRecoverWaitingLocalRestoreTasks != 0 ||
		evidence.AutoRecoverWaitingManualTasks != 0 ||
		evidence.AutoRecoverWaitingRetryLimitTasks != 0 ||
		evidence.AutoRecoverWaitingOtherTasks != 0 {
		t.Fatalf("expected global auto recover split 1/1/0/0/0/0/0/0, got runnable=%d cooldown=%d window=%d auth=%d local=%d manual=%d retryLimit=%d other=%d",
			evidence.AutoRecoverRunnableTasks,
			evidence.AutoRecoverWaitingCooldownTasks,
			evidence.AutoRecoverWaitingRetryWindowTasks,
			evidence.AutoRecoverWaitingAuthRefreshTasks,
			evidence.AutoRecoverWaitingLocalRestoreTasks,
			evidence.AutoRecoverWaitingManualTasks,
			evidence.AutoRecoverWaitingRetryLimitTasks,
			evidence.AutoRecoverWaitingOtherTasks,
		)
	}
	if len(evidence.AutoRecoverPool) != 2 {
		t.Fatalf("expected autoRecoverPool len 2, got %#v", evidence.AutoRecoverPool)
	}
	if evidence.AutoRecoverPool[0].Mode != "upload_checkpoint_auto_resume" {
		t.Fatalf("expected checkpoint lane first, got %#v", evidence.AutoRecoverPool)
	}
	checkpointLane := autoRecoverLaneByMode(evidence.AutoRecoverPool, "upload_checkpoint_auto_resume")
	if checkpointLane.TaskCount != 1 || checkpointLane.UploadCheckpointEligible != 1 {
		t.Fatalf("unexpected checkpoint lane: %#v", checkpointLane)
	}
	if checkpointLane.SuggestedProtocolGroupBudget != 1 {
		t.Fatalf("expected checkpoint lane protocol group budget 1, got %#v", checkpointLane)
	}
	if checkpointLane.RunnableTaskCount != 1 || checkpointLane.WaitingCooldownTaskCount != 0 || checkpointLane.WaitingRetryWindowTaskCount != 0 || checkpointLane.WaitingOtherTaskCount != 0 {
		t.Fatalf("expected checkpoint lane runnable split 1/0/0/0, got %#v", checkpointLane)
	}
	if checkpointLane.PrimaryRetryClass != "retry_failed" {
		t.Fatalf("expected checkpoint primaryRetryClass retry_failed, got %#v", checkpointLane)
	}
	if len(checkpointLane.RetryClasses) != 1 || checkpointLane.RetryClasses[0] != "retry_failed" {
		t.Fatalf("expected checkpoint lane retryClasses [retry_failed], got %#v", checkpointLane.RetryClasses)
	}
	cooldownLane := autoRecoverLaneByMode(evidence.AutoRecoverPool, "cooldown_elapsed_auto_retry")
	if cooldownLane.TaskCount != 1 || cooldownLane.CooldownCount != 1 {
		t.Fatalf("unexpected cooldown lane: %#v", cooldownLane)
	}
	if cooldownLane.RunnableTaskCount != 0 || cooldownLane.WaitingCooldownTaskCount != 1 || cooldownLane.WaitingRetryWindowTaskCount != 0 || cooldownLane.WaitingOtherTaskCount != 0 {
		t.Fatalf("expected cooldown lane runnable split 0/1/0/0, got %#v", cooldownLane)
	}
	if cooldownLane.PrimaryRetryClass != "rate_limited" || cooldownLane.PrimaryBlockedAction != "wait_for_cooldown" {
		t.Fatalf("expected cooldown primary fields rate_limited/wait_for_cooldown, got %#v", cooldownLane)
	}
	if len(cooldownLane.RetryClasses) != 1 || cooldownLane.RetryClasses[0] != "rate_limited" {
		t.Fatalf("expected cooldown lane retryClasses [rate_limited], got %#v", cooldownLane.RetryClasses)
	}
	if len(cooldownLane.BlockedActions) != 1 || cooldownLane.BlockedActions[0] != "wait_for_cooldown" {
		t.Fatalf("expected cooldown lane blockedActions [wait_for_cooldown], got %#v", cooldownLane.BlockedActions)
	}

	statuses, err := svc.ProviderStatuses(ctx)
	if err != nil {
		t.Fatalf("ProviderStatuses() error = %v", err)
	}
	for _, status := range statuses {
		switch status.ProviderKey {
		case "auto_recover_cooldown_target":
			if status.AutoRecoverCount != 1 {
				t.Fatalf("expected cooldown provider autoRecoverCount 1, got %#v", status)
			}
			if got := intNumber(status.SnapshotSummary["autoRecoverWaitingCooldownTasks"]); got != 1 {
				t.Fatalf("expected cooldown provider waiting cooldown 1, got %#v", status.SnapshotSummary["autoRecoverWaitingCooldownTasks"])
			}
		case "auto_recover_checkpoint_target":
			if status.AutoRecoverCount != 1 {
				t.Fatalf("expected checkpoint provider autoRecoverCount 1, got %#v", status)
			}
			if got := intNumber(status.SnapshotSummary["autoRecoverRunnableTasks"]); got != 1 {
				t.Fatalf("expected checkpoint provider runnable 1, got %#v", status.SnapshotSummary["autoRecoverRunnableTasks"])
			}
		}
	}

	cooldownDetail, ok, err := svc.Get(ctx, cooldownTask.Task.ID)
	if err != nil || !ok {
		t.Fatalf("Get(cooldown) error=%v ok=%v", err, ok)
	}
	cooldownDetail.Results[0].CreatedAt = time.Now().Add(-2 * time.Hour).UTC().Format(time.RFC3339)
	syncRuntimeRetryQueue(&cooldownDetail.Runtime, cooldownDetail.Plan.Metadata, cooldownDetail.Results)
	applyRetryQueueSummary(&cooldownDetail.Runtime, cooldownDetail.Plan.Metadata)
	cooldownDetail.Task.UpdatedAt = time.Now().Add(-10 * time.Minute).UTC().Format(time.RFC3339)
	if err := replaceTaskDetailAndResults(ctx, store, cooldownDetail); err != nil {
		t.Fatalf("replaceTaskDetailAndResults(cooldown) error = %v", err)
	}

	recovered, err := svc.RecoverBlockedTasks(ctx)
	if err != nil {
		t.Fatalf("RecoverBlockedTasks() error = %v", err)
	}
	if recovered != 2 {
		t.Fatalf("expected recovered count 2, got %d", recovered)
	}
	if len(recoveryOrder) != 2 {
		t.Fatalf("expected two recovery operations, got %#v", recoveryOrder)
	}
	if recoveryOrder[0] != "checkpoint" || recoveryOrder[1] != "cooldown" {
		t.Fatalf("expected checkpoint recovery before cooldown, got %#v", recoveryOrder)
	}
	if cooldownUploadCalls != 2 {
		t.Fatalf("expected cooldown upload calls 2, got %d", cooldownUploadCalls)
	}
	if checkpointUploadCalls != 2 {
		t.Fatalf("expected checkpoint upload calls 2, got %d", checkpointUploadCalls)
	}

	filtered, err := svc.RecoverBlockedTasksWithOptions(ctx, RecoverOptions{
		RetryClass: "auth_expired",
		Limit:      5,
	})
	if err != nil {
		t.Fatalf("RecoverBlockedTasksWithOptions(retryClass miss) error = %v", err)
	}
	if filtered.MatchedCount != 0 || filtered.RecoveredCount != 0 || filtered.RetryClass != "auth_expired" {
		t.Fatalf("expected retryClass miss to recover nothing, got %#v", filtered)
	}
}

func TestServiceRecoverBlockedTasksWithOptionsFiltersModeProviderAndLimit(t *testing.T) {
	ctx := context.Background()
	store, err := sqlitestore.New(ctx, filepath.Join(t.TempDir(), "auto-recover-options.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer func() { _ = store.Close() }()

	cooldownUploadCalls := 0
	checkpointUploadCalls := 0
	cooldownAdapter := &scriptedAdapter{
		meta: provider.Provider{
			Key:              "recover_options_cooldown_target",
			DisplayName:      "Recover Options Cooldown Target",
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
			cooldownUploadCalls++
			if cooldownUploadCalls == 1 {
				return provider.UploadResult{
					OperationResult: provider.OperationResult{
						Status:  "rate_limited",
						Message: "rate limited",
						Mode:    "fake_rate_limit",
					},
				}
			}
			return provider.UploadResult{
				OperationResult: provider.OperationResult{
					OK:      true,
					Status:  "ok",
					Message: "cooldown recovered",
					Mode:    "fake_cooldown_ok",
				},
			}
		},
	}
	checkpointAdapter := &scriptedAdapter{
		meta: provider.Provider{
			Key:              "recover_options_checkpoint_target",
			DisplayName:      "Recover Options Checkpoint Target",
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
			checkpointUploadCalls++
			if req.ResumeUpload != nil {
				return provider.UploadResult{
					OperationResult: provider.OperationResult{
						OK:      true,
						Status:  "ok",
						Message: "checkpoint resumed",
						Mode:    "fake_checkpoint_resume_ok",
						Payload: map[string]interface{}{
							"fileId":   req.ResumeUpload.FileID,
							"uploadId": req.ResumeUpload.UploadID,
						},
					},
				}
			}
			return provider.UploadResult{
				OperationResult: provider.OperationResult{
					Status:  "provider_request_failed",
					Message: "checkpoint interrupted",
					Mode:    "fake_checkpoint_failed",
					Payload: map[string]interface{}{
						"fileId":           "recover-options-file",
						"uploadId":         "recover-options-upload",
						"nextPartNumber":   2,
						"failedPartNumber": 2,
					},
				},
			}
		},
	}

	registry := provider.NewRegistry(cooldownAdapter, checkpointAdapter)
	authSvc := auth.NewService(store, registry)
	svc := NewService(store, registry, authSvc)
	cooldownProfile, err := authSvc.CreateProfile(ctx, auth.CreateProfileInput{
		ProviderKey: "recover_options_cooldown_target",
		AuthMode:    "manual_token",
		DisplayName: "recover options cooldown target",
		Token:       "token-cooldown",
	})
	if err != nil {
		t.Fatalf("CreateProfile(cooldown) error = %v", err)
	}
	checkpointProfile, err := authSvc.CreateProfile(ctx, auth.CreateProfileInput{
		ProviderKey: "recover_options_checkpoint_target",
		AuthMode:    "manual_token",
		DisplayName: "recover options checkpoint target",
		Token:       "token-checkpoint",
	})
	if err != nil {
		t.Fatalf("CreateProfile(checkpoint) error = %v", err)
	}

	cooldownTask, err := svc.Create(ctx, CreateRequest{
		SourceProvider:  "guangya",
		TargetProvider:  "recover_options_cooldown_target",
		TargetProfileID: cooldownProfile.ID,
		ThresholdMB:     1,
		RiskOverride: &planner.RiskProfileOverride{
			CooldownSeconds: intPtrTask(3600),
		},
		Entries: []planner.SourceEntry{
			{Path: "/cooldown.bin", Size: 1024, MD5: "cooldown-md5"},
		},
	})
	if err != nil {
		t.Fatalf("Create(cooldown) error = %v", err)
	}
	if _, ok, err := svc.Run(ctx, cooldownTask.Task.ID); err != nil || !ok {
		t.Fatalf("Run(cooldown) error=%v ok=%v", err, ok)
	}

	localFile := filepath.Join(t.TempDir(), "recover-options-checkpoint.bin")
	if err := os.WriteFile(localFile, []byte("checkpoint"), 0o644); err != nil {
		t.Fatalf("WriteFile(checkpoint) error = %v", err)
	}
	checkpointTask, err := svc.Create(ctx, CreateRequest{
		SourceProvider:  "guangya",
		TargetProvider:  "recover_options_checkpoint_target",
		TargetProfileID: checkpointProfile.ID,
		ThresholdMB:     1,
		Entries: []planner.SourceEntry{
			{Path: "/checkpoint.bin", Size: 2048, MD5: "checkpoint-md5", LocalPath: localFile},
		},
	})
	if err != nil {
		t.Fatalf("Create(checkpoint) error = %v", err)
	}
	if _, ok, err := svc.Run(ctx, checkpointTask.Task.ID); err != nil || !ok {
		t.Fatalf("Run(checkpoint) error=%v ok=%v", err, ok)
	}

	cooldownDetail, ok, err := svc.Get(ctx, cooldownTask.Task.ID)
	if err != nil || !ok {
		t.Fatalf("Get(cooldown) error=%v ok=%v", err, ok)
	}
	cooldownDetail.Results[0].CreatedAt = time.Now().Add(-2 * time.Hour).UTC().Format(time.RFC3339)
	syncRuntimeRetryQueue(&cooldownDetail.Runtime, cooldownDetail.Plan.Metadata, cooldownDetail.Results)
	applyRetryQueueSummary(&cooldownDetail.Runtime, cooldownDetail.Plan.Metadata)
	if err := replaceTaskDetailAndResults(ctx, store, cooldownDetail); err != nil {
		t.Fatalf("replaceTaskDetailAndResults(cooldown) error = %v", err)
	}

	result, err := svc.RecoverBlockedTasksWithOptions(ctx, RecoverOptions{
		Mode:  "upload_checkpoint_auto_resume",
		Limit: 1,
	})
	if err != nil {
		t.Fatalf("RecoverBlockedTasksWithOptions(checkpoint) error = %v", err)
	}
	if result.MatchedCount != 1 || result.RecoveredCount != 1 || result.SkippedByLimit != 0 {
		t.Fatalf("unexpected checkpoint recover result: %#v", result)
	}

	afterCheckpoint, ok, err := svc.Get(ctx, checkpointTask.Task.ID)
	if err != nil || !ok {
		t.Fatalf("Get(checkpoint) error=%v ok=%v", err, ok)
	}
	if afterCheckpoint.Task.State != StateCompleted {
		t.Fatalf("expected checkpoint task completed, got %s", afterCheckpoint.Task.State)
	}
	stillBlocked, ok, err := svc.Get(ctx, cooldownTask.Task.ID)
	if err != nil || !ok {
		t.Fatalf("Get(cooldown after checkpoint) error=%v ok=%v", err, ok)
	}
	if stillBlocked.Task.State != StateBlocked {
		t.Fatalf("expected cooldown task to stay blocked, got %s", stillBlocked.Task.State)
	}
	syncRuntimeRetryQueue(&stillBlocked.Runtime, stillBlocked.Plan.Metadata, stillBlocked.Results)
	applyRetryQueueSummary(&stillBlocked.Runtime, stillBlocked.Plan.Metadata)
	if effectiveBlockedAction(stillBlocked) != "wait_for_cooldown" {
		t.Fatalf("expected effective blockedAction wait_for_cooldown, got %s", effectiveBlockedAction(stillBlocked))
	}

	blockedActionOnly, err := svc.RecoverBlockedTasksWithOptions(ctx, RecoverOptions{
		BlockedAction: "wait_for_cooldown",
		Limit:         1,
	})
	if err != nil {
		t.Fatalf("RecoverBlockedTasksWithOptions(blockedAction) error = %v", err)
	}
	if blockedActionOnly.MatchedCount != 1 || blockedActionOnly.RecoveredCount != 1 || blockedActionOnly.BlockedAction != "wait_for_cooldown" {
		t.Fatalf("unexpected blockedAction recover result: %#v", blockedActionOnly)
	}

	finalCooldown, ok, err := svc.Get(ctx, cooldownTask.Task.ID)
	if err != nil || !ok {
		t.Fatalf("Get(final cooldown) error=%v ok=%v", err, ok)
	}
	if finalCooldown.Task.State != StateCompleted {
		t.Fatalf("expected cooldown task completed, got %s", finalCooldown.Task.State)
	}
	if cooldownUploadCalls != 2 {
		t.Fatalf("expected cooldown upload calls 2, got %d", cooldownUploadCalls)
	}
	if checkpointUploadCalls != 2 {
		t.Fatalf("expected checkpoint upload calls 2, got %d", checkpointUploadCalls)
	}
}

func TestServiceRecoverBlockedTasksWithOptionsFiltersStrategy(t *testing.T) {
	ctx := context.Background()
	store, err := sqlitestore.New(ctx, filepath.Join(t.TempDir(), "strategy-filter.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer func() { _ = store.Close() }()

	uploadCalls := 0
	adapter := &scriptedAdapter{
		meta: provider.Provider{
			Key:              "strategy_filter_target",
			DisplayName:      "Strategy Filter Target",
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
			uploadCalls++
			return provider.UploadResult{
				OperationResult: provider.OperationResult{
					Status:  "rate_limited",
					Message: "strategy filtered blocked",
					Mode:    "fake_rate_limit",
				},
			}
		},
	}

	registry := provider.NewRegistry(adapter)
	authSvc := auth.NewService(store, registry)
	svc := NewService(store, registry, authSvc)
	profile, err := authSvc.CreateProfile(ctx, auth.CreateProfileInput{
		ProviderKey: "strategy_filter_target",
		AuthMode:    "manual_token",
		DisplayName: "strategy filter target",
		Token:       "token-1",
	})
	if err != nil {
		t.Fatalf("CreateProfile() error = %v", err)
	}

	createBlocked := func(path string) string {
		detail, err := svc.Create(ctx, CreateRequest{
			SourceProvider:  "guangya",
			TargetProvider:  "strategy_filter_target",
			TargetProfileID: profile.ID,
			ThresholdMB:     1,
			Entries:         []planner.SourceEntry{{Path: path, Size: 1024, MD5: strings.Trim(path, "/")}},
		})
		if err != nil {
			t.Fatalf("Create(%s) error = %v", path, err)
		}
		if _, ok, err := svc.Run(ctx, detail.Task.ID); err != nil || !ok {
			t.Fatalf("Run(%s) error=%v ok=%v", path, err, ok)
		}
		blocked, ok, err := svc.Get(ctx, detail.Task.ID)
		if err != nil || !ok {
			t.Fatalf("Get(%s) error=%v ok=%v", path, err, ok)
		}
		if blocked.Task.State != StateBlocked {
			t.Fatalf("expected blocked task for %s, got %s", path, blocked.Task.State)
		}
		return detail.Task.ID
	}

	fastID := createBlocked("/fast.bin")
	downloadID := createBlocked("/download.bin")

	fastDetail, ok, err := svc.Get(ctx, fastID)
	if err != nil || !ok {
		t.Fatalf("Get(fast) error=%v ok=%v", err, ok)
	}
	fastDetail.Results[0].Payload["strategy"] = "fast_upload"
	if err := replaceTaskDetailAndResults(ctx, store, fastDetail); err != nil {
		t.Fatalf("replaceTaskDetailAndResults(fast) error = %v", err)
	}

	downloadDetail, ok, err := svc.Get(ctx, downloadID)
	if err != nil || !ok {
		t.Fatalf("Get(download) error=%v ok=%v", err, ok)
	}
	downloadDetail.Results[0].Payload["strategy"] = "download_upload"
	if err := replaceTaskDetailAndResults(ctx, store, downloadDetail); err != nil {
		t.Fatalf("replaceTaskDetailAndResults(download) error = %v", err)
	}

	result, err := svc.RecoverBlockedTasksWithOptions(ctx, RecoverOptions{
		Strategy:           "download_upload",
		DryRun:             true,
		IncludeNonRunnable: true,
		Limit:              5,
	})
	if err != nil {
		t.Fatalf("RecoverBlockedTasksWithOptions(strategy) error = %v", err)
	}
	if result.MatchedCount != 1 || result.RecoveredCount != 0 || result.SkippedByCooldownWait != 1 || result.Strategy != "download_upload" {
		t.Fatalf("unexpected strategy filter result: %#v", result)
	}
	if got := result.StrategyCounts["download_upload"]; got != 1 {
		t.Fatalf("expected strategyCounts download_upload=1, got %#v", result.StrategyCounts)
	}
	if len(result.Decisions) != 1 || result.Decisions[0].Strategy != "download_upload" {
		t.Fatalf("expected decisions to carry strategy download_upload, got %#v", result.Decisions)
	}

	fastAfter, ok, err := svc.Get(ctx, fastID)
	if err != nil || !ok {
		t.Fatalf("Get(fast after) error=%v ok=%v", err, ok)
	}
	downloadAfter, ok, err := svc.Get(ctx, downloadID)
	if err != nil || !ok {
		t.Fatalf("Get(download after) error=%v ok=%v", err, ok)
	}
	if fastAfter.Task.State != StateBlocked || downloadAfter.Task.State != StateBlocked {
		t.Fatalf("expected dry-run to keep tasks blocked, got fast=%s download=%s", fastAfter.Task.State, downloadAfter.Task.State)
	}
	if uploadCalls != 2 {
		t.Fatalf("expected two initial uploads, got %d", uploadCalls)
	}
}

func TestServiceAutoRecoverPoolShowsRetryWindowWaitingAndSkipsExecutionUntilWindow(t *testing.T) {
	ctx := context.Background()
	store, err := sqlitestore.New(ctx, filepath.Join(t.TempDir(), "auto-recover-window.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer func() { _ = store.Close() }()

	uploadCalls := 0
	windowAdapter := &scriptedAdapter{
		meta: provider.Provider{
			Key:              "auto_recover_window_target",
			DisplayName:      "Auto Recover Window Target",
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
			uploadCalls++
			if uploadCalls == 1 {
				return provider.UploadResult{
					OperationResult: provider.OperationResult{
						Status:  "rate_limited",
						Message: "window wait",
						Mode:    "fake_window_wait",
					},
				}
			}
			return provider.UploadResult{
				OperationResult: provider.OperationResult{
					OK:      true,
					Status:  "ok",
					Message: "window recovered",
					Mode:    "fake_window_ok",
				},
			}
		},
	}

	registry := provider.NewRegistry(windowAdapter)
	authSvc := auth.NewService(store, registry)
	svc := NewService(store, registry, authSvc)
	profile, err := authSvc.CreateProfile(ctx, auth.CreateProfileInput{
		ProviderKey: "auto_recover_window_target",
		AuthMode:    "manual_token",
		DisplayName: "auto recover window target",
		Token:       "token-window",
	})
	if err != nil {
		t.Fatalf("CreateProfile(window) error = %v", err)
	}

	nowHour := time.Now().UTC().Hour()
	startHour := (nowHour + 1) % 24
	endHour := (nowHour + 2) % 24
	if startHour == endHour {
		endHour = (endHour + 1) % 24
	}

	taskDetail, err := svc.Create(ctx, CreateRequest{
		SourceProvider:  "guangya",
		TargetProvider:  "auto_recover_window_target",
		TargetProfileID: profile.ID,
		ThresholdMB:     1,
		RiskOverride: &planner.RiskProfileOverride{
			CooldownSeconds:    intPtrTask(0),
			AutoRetryStartHour: intPtrTask(startHour),
			AutoRetryEndHour:   intPtrTask(endHour),
		},
		Entries: []planner.SourceEntry{
			{Path: "/window.bin", Size: 1024, MD5: "window-md5"},
		},
	})
	if err != nil {
		t.Fatalf("Create(window) error = %v", err)
	}
	running, ok, err := svc.Run(ctx, taskDetail.Task.ID)
	if err != nil || !ok {
		t.Fatalf("Run(window) error=%v ok=%v", err, ok)
	}
	running.Results[0].CreatedAt = time.Now().Add(-2 * time.Hour).UTC().Format(time.RFC3339)
	syncRuntimeRetryQueue(&running.Runtime, running.Plan.Metadata, running.Results)
	applyRetryQueueSummary(&running.Runtime, running.Plan.Metadata)
	running.Task.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	if err := replaceTaskDetailAndResults(ctx, store, running); err != nil {
		t.Fatalf("replaceTaskDetailAndResults(window) error = %v", err)
	}
	running, ok, err = svc.Get(ctx, taskDetail.Task.ID)
	if err != nil || !ok {
		t.Fatalf("Get(window) error=%v ok=%v", err, ok)
	}
	if running.Runtime.BlockedAction != "wait_for_retry_window" {
		t.Fatalf("expected blockedAction wait_for_retry_window, got %#v", running.Runtime.BlockedAction)
	}
	if running.Runtime.BlockedReason != "retry_queue_waiting_for_retry_window" {
		t.Fatalf("expected blockedReason retry_queue_waiting_for_retry_window, got %#v", running.Runtime.BlockedReason)
	}
	if strings.TrimSpace(running.Runtime.NextRetryAt) == "" {
		t.Fatalf("expected runtime nextRetryAt for waiting window, got %#v", running.Runtime)
	}

	retrySummary, ok := running.Plan.Metadata["retrySummary"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected retrySummary metadata, got %#v", running.Plan.Metadata["retrySummary"])
	}
	if stringValue(retrySummary["autoRecoverMode"]) != "retry_window_waiting_auto_retry" {
		t.Fatalf("expected retry_window_waiting_auto_retry mode, got %#v", retrySummary)
	}
	if stringValue(retrySummary["blockedAction"]) != "wait_for_retry_window" {
		t.Fatalf("expected blockedAction wait_for_retry_window, got %#v", retrySummary)
	}
	if !boolValue(retrySummary["windowBlocked"]) {
		t.Fatalf("expected windowBlocked true, got %#v", retrySummary)
	}
	if intNumber(retrySummary["autoRecoverWaitingRetryWindowTasks"]) != 1 {
		t.Fatalf("expected autoRecoverWaitingRetryWindowTasks 1, got %#v", retrySummary)
	}

	evidence, err := svc.RuntimeEvidence(ctx)
	if err != nil {
		t.Fatalf("RuntimeEvidence() error = %v", err)
	}
	windowLane := autoRecoverLaneByMode(evidence.AutoRecoverPool, "retry_window_waiting_auto_retry")
	if windowLane.Mode == "" {
		t.Fatalf("expected retry_window_waiting_auto_retry lane in %#v", evidence.AutoRecoverPool)
	}
	if windowLane.TaskCount != 1 || windowLane.PrimaryBlockedAction != "wait_for_retry_window" {
		t.Fatalf("unexpected window lane %#v", windowLane)
	}
	if windowLane.RunnableTaskCount != 0 || windowLane.WaitingCooldownTaskCount != 0 || windowLane.WaitingRetryWindowTaskCount != 1 || windowLane.WaitingOtherTaskCount != 0 {
		t.Fatalf("expected window lane runnable split 0/0/1/0, got %#v", windowLane)
	}
	if windowLane.PrimaryRetryClass != "rate_limited" {
		t.Fatalf("expected window lane primary retryClass rate_limited, got %#v", windowLane)
	}
	if strings.TrimSpace(windowLane.NextRetryAt) == "" {
		t.Fatalf("expected window lane nextRetryAt, got %#v", windowLane)
	}

	recovered, err := svc.RecoverBlockedTasks(ctx)
	if err != nil {
		t.Fatalf("RecoverBlockedTasks() error = %v", err)
	}
	if recovered != 0 {
		t.Fatalf("expected no recovery before retry window, got %d", recovered)
	}
	manual, err := svc.RecoverBlockedTasksWithOptions(ctx, RecoverOptions{Mode: "retry_window_waiting_auto_retry", Limit: 1, IncludeNonRunnable: true})
	if err != nil {
		t.Fatalf("RecoverBlockedTasksWithOptions(waiting window) error = %v", err)
	}
	if manual.MatchedCount != 1 || manual.RecoveredCount != 0 || manual.SkippedByRetryWindowWait != 1 {
		t.Fatalf("expected waiting window recover summary matched=1 recovered=0 retryWindowWait=1, got %#v", manual)
	}
	if uploadCalls != 1 {
		t.Fatalf("expected no extra upload calls before retry window, got %d", uploadCalls)
	}
}

func TestServiceRecoverBlockedTasksWithOptionsDryRunDoesNotMutateTasks(t *testing.T) {
	ctx := context.Background()
	store, err := sqlitestore.New(ctx, filepath.Join(t.TempDir(), "recover-dry-run.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer func() { _ = store.Close() }()

	uploadCalls := 0
	targetAdapter := &scriptedAdapter{
		meta: provider.Provider{
			Key:              "recover_dry_run_target",
			DisplayName:      "Recover Dry Run Target",
			ProtocolGroup:    "recover_dry_run_group",
			AuthModes:        []string{"manual_token"},
			FastUploadInputs: []string{"md5", "size"},
			FallbackModes:    []string{"download_upload"},
			Status:           "planned",
		},
		capability: provider.CapabilitySet{
			SupportsAuthValidation: true,
			SupportsFastUpload:     true,
			SupportsUpload:         true,
		},
		fastCheckFunc: func(req provider.FastUploadCheckRequest) provider.FastUploadCheckResult {
			return provider.FastUploadCheckResult{
				OperationResult: provider.OperationResult{
					OK:      true,
					Status:  "hash_miss",
					Message: "need upload",
					Mode:    "dry_run_fast_upload",
				},
			}
		},
		uploadFunc: func(req provider.UploadRequest) provider.UploadResult {
			uploadCalls++
			if uploadCalls == 1 {
				return provider.UploadResult{
					OperationResult: provider.OperationResult{
						Status:  "upload_checkpoint_pending",
						Message: "checkpoint pending",
						Mode:    "checkpoint_pending",
						Payload: map[string]interface{}{
							"fileId":         "dry-run-file",
							"uploadId":       "dry-run-upload",
							"nextPartNumber": 1,
						},
					},
				}
			}
			return provider.UploadResult{
				OperationResult: provider.OperationResult{
					OK:      true,
					Status:  "ok",
					Message: "dry run recovered",
					Mode:    "dry_run_recovered",
				},
			}
		},
	}

	registry := provider.NewRegistry(targetAdapter)
	authSvc := auth.NewService(store, registry)
	svc := NewService(store, registry, authSvc)
	profile, err := authSvc.CreateProfile(ctx, auth.CreateProfileInput{
		ProviderKey: "recover_dry_run_target",
		AuthMode:    "manual_token",
		DisplayName: "recover dry run target",
		Token:       "token-dry-run",
	})
	if err != nil {
		t.Fatalf("CreateProfile() error = %v", err)
	}

	localFile := filepath.Join(t.TempDir(), "recover-dry-run.bin")
	if err := os.WriteFile(localFile, []byte("recover-dry-run"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	created, err := svc.Create(ctx, CreateRequest{
		SourceProvider:  "recover_dry_run_target",
		TargetProvider:  "recover_dry_run_target",
		TargetProfileID: profile.ID,
		ThresholdMB:     1,
		Entries: []planner.SourceEntry{
			{Path: "/recover-dry-run.bin", Size: 1024, MD5: "recover-dry-run-md5", LocalPath: localFile},
		},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, ok, err := svc.Run(ctx, created.Task.ID); err != nil || !ok {
		t.Fatalf("Run() error=%v ok=%v", err, ok)
	}
	blocked, ok, err := svc.Get(ctx, created.Task.ID)
	if err != nil || !ok {
		t.Fatalf("Get(blocked) error=%v ok=%v", err, ok)
	}
	if blocked.Task.State != StateCompletedWithErrors {
		t.Fatalf("expected completed_with_errors before dry run, got %s", blocked.Task.State)
	}
	if uploadCalls != 1 {
		t.Fatalf("expected initial upload calls 1, got %d", uploadCalls)
	}

	preview, err := svc.RecoverBlockedTasksWithOptions(ctx, RecoverOptions{
		Mode:        "upload_checkpoint_auto_resume",
		ProviderKey: "recover_dry_run_target",
		Limit:       1,
		DryRun:      true,
	})
	if err != nil {
		t.Fatalf("RecoverBlockedTasksWithOptions(dryRun) error = %v", err)
	}
	if !preview.DryRun || preview.MatchedCount != 1 || preview.RecoveredCount != 1 {
		t.Fatalf("unexpected dry run preview result: %#v", preview)
	}
	if len(preview.Decisions) == 0 || preview.Decisions[0].Outcome != "dry_run_recoverable" {
		t.Fatalf("expected dry run decisions, got %#v", preview.Decisions)
	}
	if uploadCalls != 1 {
		t.Fatalf("expected dry run not to trigger extra upload, got %d", uploadCalls)
	}
	afterPreview, ok, err := svc.Get(ctx, created.Task.ID)
	if err != nil || !ok {
		t.Fatalf("Get(after preview) error=%v ok=%v", err, ok)
	}
	if afterPreview.Task.State != StateCompletedWithErrors {
		t.Fatalf("expected task state unchanged after dry run, got %s", afterPreview.Task.State)
	}
	if len(afterPreview.Results) != 1 {
		t.Fatalf("expected results unchanged after dry run, got %d", len(afterPreview.Results))
	}

	executed, err := svc.RecoverBlockedTasksWithOptions(ctx, RecoverOptions{
		Mode:        "upload_checkpoint_auto_resume",
		ProviderKey: "recover_dry_run_target",
		Limit:       1,
	})
	if err != nil {
		t.Fatalf("RecoverBlockedTasksWithOptions(execute) error = %v", err)
	}
	if executed.DryRun || executed.RecoveredCount != 1 {
		t.Fatalf("unexpected execute recover result: %#v", executed)
	}
	if len(executed.Decisions) == 0 || executed.Decisions[0].Outcome != "recovered" {
		t.Fatalf("expected execute decisions, got %#v", executed.Decisions)
	}
	if uploadCalls != 2 {
		t.Fatalf("expected execute to trigger second upload, got %d", uploadCalls)
	}
	completed, ok, err := svc.Get(ctx, created.Task.ID)
	if err != nil || !ok {
		t.Fatalf("Get(completed) error=%v ok=%v", err, ok)
	}
	if completed.Task.State != StateCompleted {
		t.Fatalf("expected task completed after execute, got %s", completed.Task.State)
	}
}

func TestServiceRecoverBlockedTasksWithOptionsFiltersPathSubset(t *testing.T) {
	ctx := context.Background()
	store, err := sqlitestore.New(ctx, filepath.Join(t.TempDir(), "auto-recover-path.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer func() { _ = store.Close() }()

	uploadCallsByPath := map[string]int{}
	adapter := &scriptedAdapter{
		meta: provider.Provider{
			Key:              "recover_path_target",
			DisplayName:      "Recover Path Target",
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
			uploadCallsByPath[req.Path]++
			if uploadCallsByPath[req.Path] == 1 {
				return provider.UploadResult{
					OperationResult: provider.OperationResult{
						Status:  "rate_limited",
						Message: "rate limited",
						Mode:    "fake_rate_limit",
					},
				}
			}
			return provider.UploadResult{
				OperationResult: provider.OperationResult{
					OK:      true,
					Status:  "ok",
					Message: "path recovered",
					Mode:    "fake_path_ok",
				},
			}
		},
	}

	registry := provider.NewRegistry(adapter)
	authSvc := auth.NewService(store, registry)
	svc := NewService(store, registry, authSvc)
	profile, err := authSvc.CreateProfile(ctx, auth.CreateProfileInput{
		ProviderKey: "recover_path_target",
		AuthMode:    "manual_token",
		DisplayName: "recover path target",
		Token:       "token-path",
	})
	if err != nil {
		t.Fatalf("CreateProfile(path) error = %v", err)
	}

	taskDetail, err := svc.Create(ctx, CreateRequest{
		SourceProvider:  "guangya",
		TargetProvider:  "recover_path_target",
		TargetProfileID: profile.ID,
		ThresholdMB:     1,
		RiskOverride: &planner.RiskProfileOverride{
			CooldownSeconds: intPtrTask(3600),
		},
		Entries: []planner.SourceEntry{
			{Path: "/leaf-a/one.bin", Size: 101, MD5: "leaf-a"},
			{Path: "/leaf-b/two.bin", Size: 202, MD5: "leaf-b"},
		},
	})
	if err != nil {
		t.Fatalf("Create(path) error = %v", err)
	}
	if _, ok, err := svc.Run(ctx, taskDetail.Task.ID); err != nil || !ok {
		t.Fatalf("Run(path) error=%v ok=%v", err, ok)
	}

	blocked, ok, err := svc.Get(ctx, taskDetail.Task.ID)
	if err != nil || !ok {
		t.Fatalf("Get(blocked path task) error=%v ok=%v", err, ok)
	}
	for idx := range blocked.Results {
		blocked.Results[idx].CreatedAt = time.Now().Add(-2 * time.Hour).UTC().Format(time.RFC3339)
	}
	syncRuntimeRetryQueue(&blocked.Runtime, blocked.Plan.Metadata, blocked.Results)
	applyRetryQueueSummary(&blocked.Runtime, blocked.Plan.Metadata)
	if err := replaceTaskDetailAndResults(ctx, store, blocked); err != nil {
		t.Fatalf("replaceTaskDetailAndResults(path) error = %v", err)
	}

	result, err := svc.RecoverBlockedTasksWithOptions(ctx, RecoverOptions{
		ProviderKey: "recover_path_target",
		Path:        "/leaf-a",
		Scope:       "selected_retry_subset",
		Limit:       1,
	})
	if err != nil {
		t.Fatalf("RecoverBlockedTasksWithOptions(path subset) error = %v", err)
	}
	if result.MatchedCount != 1 || result.RecoveredCount != 1 || result.Path != "/leaf-a" {
		t.Fatalf("unexpected path recover result: %#v", result)
	}

	after, ok, err := svc.Get(ctx, taskDetail.Task.ID)
	if err != nil || !ok {
		t.Fatalf("Get(path recovered task) error=%v ok=%v", err, ok)
	}
	if after.Task.State != StateCompleted {
		t.Fatalf("expected path-scoped task completed, got %s", after.Task.State)
	}
	if len(after.Results) != 1 {
		t.Fatalf("expected one result after path-scoped recovery, got %#v", after.Results)
	}
	if got := stringValue(after.Results[0].Payload["path"]); got != "/leaf-a/one.bin" {
		t.Fatalf("expected recovered result path /leaf-a/one.bin, got %s", got)
	}
	if got := uploadCallsByPath["/leaf-a/one.bin"]; got != 2 {
		t.Fatalf("expected /leaf-a/one.bin upload calls 2, got %d", got)
	}
	if got := uploadCallsByPath["/leaf-b/two.bin"]; got != 1 {
		t.Fatalf("expected /leaf-b/two.bin upload calls to remain 1, got %d", got)
	}
}

func TestServiceRecoverBlockedTasksWithOptionsSupportsMultiplePaths(t *testing.T) {
	ctx := context.Background()
	store, err := sqlitestore.New(ctx, filepath.Join(t.TempDir(), "auto-recover-paths.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer func() { _ = store.Close() }()

	uploadCallsByPath := map[string]int{}
	adapter := &scriptedAdapter{
		meta: provider.Provider{
			Key:              "recover_paths_target",
			DisplayName:      "Recover Paths Target",
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
			uploadCallsByPath[req.Path]++
			if uploadCallsByPath[req.Path] == 1 {
				return provider.UploadResult{
					OperationResult: provider.OperationResult{
						Status:  "rate_limited",
						Message: "rate limited",
						Mode:    "fake_rate_limit",
					},
				}
			}
			return provider.UploadResult{
				OperationResult: provider.OperationResult{
					OK:      true,
					Status:  "ok",
					Message: "paths recovered",
					Mode:    "fake_paths_ok",
				},
			}
		},
	}

	registry := provider.NewRegistry(adapter)
	authSvc := auth.NewService(store, registry)
	svc := NewService(store, registry, authSvc)
	profile, err := authSvc.CreateProfile(ctx, auth.CreateProfileInput{
		ProviderKey: "recover_paths_target",
		AuthMode:    "manual_token",
		DisplayName: "recover paths target",
		Token:       "token-paths",
	})
	if err != nil {
		t.Fatalf("CreateProfile(paths) error = %v", err)
	}

	taskDetail, err := svc.Create(ctx, CreateRequest{
		SourceProvider:  "guangya",
		TargetProvider:  "recover_paths_target",
		TargetProfileID: profile.ID,
		ThresholdMB:     1,
		RiskOverride: &planner.RiskProfileOverride{
			CooldownSeconds: intPtrTask(3600),
		},
		Entries: []planner.SourceEntry{
			{Path: "/leaf-a/one.bin", Size: 101, MD5: "leaf-a"},
			{Path: "/leaf-b/two.bin", Size: 202, MD5: "leaf-b"},
			{Path: "/leaf-c/three.bin", Size: 303, MD5: "leaf-c"},
		},
	})
	if err != nil {
		t.Fatalf("Create(paths) error = %v", err)
	}
	if _, ok, err := svc.Run(ctx, taskDetail.Task.ID); err != nil || !ok {
		t.Fatalf("Run(paths) error=%v ok=%v", err, ok)
	}

	blocked, ok, err := svc.Get(ctx, taskDetail.Task.ID)
	if err != nil || !ok {
		t.Fatalf("Get(blocked paths task) error=%v ok=%v", err, ok)
	}
	for idx := range blocked.Results {
		blocked.Results[idx].CreatedAt = time.Now().Add(-2 * time.Hour).UTC().Format(time.RFC3339)
	}
	syncRuntimeRetryQueue(&blocked.Runtime, blocked.Plan.Metadata, blocked.Results)
	applyRetryQueueSummary(&blocked.Runtime, blocked.Plan.Metadata)
	if err := replaceTaskDetailAndResults(ctx, store, blocked); err != nil {
		t.Fatalf("replaceTaskDetailAndResults(paths) error = %v", err)
	}

	result, err := svc.RecoverBlockedTasksWithOptions(ctx, RecoverOptions{
		ProviderKey: "recover_paths_target",
		Paths:       []string{"/leaf-a", "/leaf-c"},
		Scope:       "selected_retry_subset",
		Limit:       1,
	})
	if err != nil {
		t.Fatalf("RecoverBlockedTasksWithOptions(paths subset) error = %v", err)
	}
	if result.MatchedCount != 1 || result.RecoveredCount != 1 || result.Path != "/leaf-a" {
		t.Fatalf("unexpected paths recover result: %#v", result)
	}

	after, ok, err := svc.Get(ctx, taskDetail.Task.ID)
	if err != nil || !ok {
		t.Fatalf("Get(paths recovered task) error=%v ok=%v", err, ok)
	}
	if len(after.Results) != 2 {
		t.Fatalf("expected two recovered results after multi-path recover, got %#v", after.Results)
	}
	recoveredPaths := []string{
		stringValue(after.Results[0].Payload["path"]),
		stringValue(after.Results[1].Payload["path"]),
	}
	sort.Strings(recoveredPaths)
	if recoveredPaths[0] != "/leaf-a/one.bin" || recoveredPaths[1] != "/leaf-c/three.bin" {
		t.Fatalf("expected recovered paths [/leaf-a/one.bin /leaf-c/three.bin], got %#v", recoveredPaths)
	}
	if got := uploadCallsByPath["/leaf-b/two.bin"]; got != 1 {
		t.Fatalf("expected /leaf-b/two.bin upload calls to remain 1, got %d", got)
	}
}

func TestServiceRecoverBlockedTasksWithOptionsFiltersTaskID(t *testing.T) {
	ctx := context.Background()
	store, err := sqlitestore.New(ctx, filepath.Join(t.TempDir(), "auto-recover-task-id.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer func() { _ = store.Close() }()

	uploadCallsByPath := map[string]int{}
	adapter := &scriptedAdapter{
		meta: provider.Provider{
			Key:              "recover_task_id_target",
			DisplayName:      "Recover Task ID Target",
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
			uploadCallsByPath[req.Path]++
			if uploadCallsByPath[req.Path] == 1 {
				return provider.UploadResult{
					OperationResult: provider.OperationResult{
						Status:  "rate_limited",
						Message: "rate limited",
						Mode:    "fake_rate_limit",
					},
				}
			}
			return provider.UploadResult{
				OperationResult: provider.OperationResult{
					OK:      true,
					Status:  "ok",
					Message: "task filtered recovered",
					Mode:    "fake_task_filter_ok",
				},
			}
		},
	}

	registry := provider.NewRegistry(adapter)
	authSvc := auth.NewService(store, registry)
	svc := NewService(store, registry, authSvc)
	profile, err := authSvc.CreateProfile(ctx, auth.CreateProfileInput{
		ProviderKey: "recover_task_id_target",
		AuthMode:    "manual_token",
		DisplayName: "recover task id target",
		Token:       "token-task-id",
	})
	if err != nil {
		t.Fatalf("CreateProfile(task id) error = %v", err)
	}

	createBlockedTask := func(path string) Detail {
		detail, err := svc.Create(ctx, CreateRequest{
			SourceProvider:  "guangya",
			TargetProvider:  "recover_task_id_target",
			TargetProfileID: profile.ID,
			ThresholdMB:     1,
			RiskOverride: &planner.RiskProfileOverride{
				CooldownSeconds: intPtrTask(3600),
			},
			Entries: []planner.SourceEntry{
				{Path: path, Size: 101, MD5: filepath.Base(path)},
			},
		})
		if err != nil {
			t.Fatalf("Create(%s) error = %v", path, err)
		}
		if _, ok, err := svc.Run(ctx, detail.Task.ID); err != nil || !ok {
			t.Fatalf("Run(%s) error=%v ok=%v", path, err, ok)
		}
		blocked, ok, err := svc.Get(ctx, detail.Task.ID)
		if err != nil || !ok {
			t.Fatalf("Get(%s) error=%v ok=%v", path, err, ok)
		}
		for idx := range blocked.Results {
			blocked.Results[idx].CreatedAt = time.Now().Add(-2 * time.Hour).UTC().Format(time.RFC3339)
		}
		syncRuntimeRetryQueue(&blocked.Runtime, blocked.Plan.Metadata, blocked.Results)
		applyRetryQueueSummary(&blocked.Runtime, blocked.Plan.Metadata)
		if err := replaceTaskDetailAndResults(ctx, store, blocked); err != nil {
			t.Fatalf("replaceTaskDetailAndResults(%s) error = %v", path, err)
		}
		return blocked
	}

	first := createBlockedTask("/group-a/one.bin")
	second := createBlockedTask("/group-b/two.bin")

	result, err := svc.RecoverBlockedTasksWithOptions(ctx, RecoverOptions{
		TaskID:      second.Task.ID,
		ProviderKey: "recover_task_id_target",
		Paths:       []string{"/group-b"},
		Scope:       "selected_retry_subset",
		Limit:       1,
	})
	if err != nil {
		t.Fatalf("RecoverBlockedTasksWithOptions(task id) error = %v", err)
	}
	if result.MatchedCount != 1 || result.RecoveredCount != 1 || result.TaskID != second.Task.ID {
		t.Fatalf("unexpected task id recover result: %#v", result)
	}

	afterFirst, ok, err := svc.Get(ctx, first.Task.ID)
	if err != nil || !ok {
		t.Fatalf("Get(first after task-id recover) error=%v ok=%v", err, ok)
	}
	if afterFirst.Task.State != StateBlocked {
		t.Fatalf("expected first task to remain blocked, got %s", afterFirst.Task.State)
	}
	if len(afterFirst.Results) != 1 || stringValue(afterFirst.Results[0].Payload["path"]) != "/group-a/one.bin" {
		t.Fatalf("expected first task results unchanged, got %#v", afterFirst.Results)
	}

	afterSecond, ok, err := svc.Get(ctx, second.Task.ID)
	if err != nil || !ok {
		t.Fatalf("Get(second after task-id recover) error=%v ok=%v", err, ok)
	}
	if len(afterSecond.Results) != 1 || stringValue(afterSecond.Results[0].Payload["path"]) != "/group-b/two.bin" {
		t.Fatalf("expected second task recovered path /group-b/two.bin, got %#v", afterSecond.Results)
	}
	if got := uploadCallsByPath["/group-a/one.bin"]; got != 1 {
		t.Fatalf("expected first task path upload calls to remain 1, got %d", got)
	}
	if got := uploadCallsByPath["/group-b/two.bin"]; got != 2 {
		t.Fatalf("expected second task path upload calls to become 2, got %d", got)
	}
}

func TestServiceRecoverBlockedTasksWithOptionsRespectsProviderBudget(t *testing.T) {
	ctx := context.Background()
	store, err := sqlitestore.New(ctx, filepath.Join(t.TempDir(), "auto-recover-provider-budget.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer func() { _ = store.Close() }()

	uploadCallsByPath := map[string]int{}
	adapter := &scriptedAdapter{
		meta: provider.Provider{
			Key:              "recover_budget_target",
			DisplayName:      "Recover Budget Target",
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
			uploadCallsByPath[req.Path]++
			if uploadCallsByPath[req.Path] == 1 {
				return provider.UploadResult{
					OperationResult: provider.OperationResult{
						Status:  "rate_limited",
						Message: "rate limited",
						Mode:    "fake_rate_limit",
					},
				}
			}
			return provider.UploadResult{
				OperationResult: provider.OperationResult{
					OK:      true,
					Status:  "ok",
					Message: "budget recovered",
					Mode:    "fake_budget_ok",
				},
			}
		},
	}

	registry := provider.NewRegistry(adapter)
	authSvc := auth.NewService(store, registry)
	svc := NewService(store, registry, authSvc)
	profile, err := authSvc.CreateProfile(ctx, auth.CreateProfileInput{
		ProviderKey: "recover_budget_target",
		AuthMode:    "manual_token",
		DisplayName: "recover budget target",
		Token:       "token-budget",
	})
	if err != nil {
		t.Fatalf("CreateProfile(budget) error = %v", err)
	}

	createBlockedTask := func(name string) string {
		taskDetail, err := svc.Create(ctx, CreateRequest{
			SourceProvider:  "guangya",
			TargetProvider:  "recover_budget_target",
			TargetProfileID: profile.ID,
			ThresholdMB:     1,
			RiskOverride: &planner.RiskProfileOverride{
				CooldownSeconds: intPtrTask(3600),
				MaxConcurrent:   intPtrTask(3),
			},
			Entries: []planner.SourceEntry{
				{Path: "/" + name + ".bin", Size: 128, MD5: name + "-md5"},
			},
		})
		if err != nil {
			t.Fatalf("Create(%s) error = %v", name, err)
		}
		if _, ok, err := svc.Run(ctx, taskDetail.Task.ID); err != nil || !ok {
			t.Fatalf("Run(%s) error=%v ok=%v", name, err, ok)
		}
		blocked, ok, err := svc.Get(ctx, taskDetail.Task.ID)
		if err != nil || !ok {
			t.Fatalf("Get(%s) error=%v ok=%v", name, err, ok)
		}
		for idx := range blocked.Results {
			blocked.Results[idx].CreatedAt = time.Now().Add(-2 * time.Hour).UTC().Format(time.RFC3339)
		}
		syncRuntimeRetryQueue(&blocked.Runtime, blocked.Plan.Metadata, blocked.Results)
		applyRetryQueueSummary(&blocked.Runtime, blocked.Plan.Metadata)
		if err := replaceTaskDetailAndResults(ctx, store, blocked); err != nil {
			t.Fatalf("replaceTaskDetailAndResults(%s) error = %v", name, err)
		}
		return taskDetail.Task.ID
	}

	firstID := createBlockedTask("budget-a")
	secondID := createBlockedTask("budget-b")

	result, err := svc.RecoverBlockedTasksWithOptions(ctx, RecoverOptions{
		ProviderKey:      "recover_budget_target",
		Limit:            5,
		LimitPerProvider: 1,
	})
	if err != nil {
		t.Fatalf("RecoverBlockedTasksWithOptions(provider budget) error = %v", err)
	}
	if result.MatchedCount != 2 || result.RecoveredCount != 1 || result.SkippedByProviderBudget != 1 || result.LimitPerProvider != 1 {
		t.Fatalf("unexpected provider budget recover result: %#v", result)
	}

	firstTask, ok, err := svc.Get(ctx, firstID)
	if err != nil || !ok {
		t.Fatalf("Get(first budget task) error=%v ok=%v", err, ok)
	}
	secondTask, ok, err := svc.Get(ctx, secondID)
	if err != nil || !ok {
		t.Fatalf("Get(second budget task) error=%v ok=%v", err, ok)
	}
	completedCount := 0
	blockedCount := 0
	for _, item := range []Detail{firstTask, secondTask} {
		switch item.Task.State {
		case StateCompleted:
			completedCount++
		case StateBlocked:
			blockedCount++
		}
	}
	if completedCount != 1 || blockedCount != 1 {
		t.Fatalf("expected one completed and one blocked task, got completed=%d blocked=%d", completedCount, blockedCount)
	}
}

func TestServiceRecoverBlockedTasksWithOptionsRespectsProfileBudgetOverride(t *testing.T) {
	ctx := context.Background()
	store, err := sqlitestore.New(ctx, filepath.Join(t.TempDir(), "auto-recover-profile-budget.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer func() { _ = store.Close() }()

	recoveryOrder := make([]string, 0, 3)
	adapter := &scriptedAdapter{
		meta: provider.Provider{
			Key:              "recover_profile_budget_target",
			DisplayName:      "recover_profile_budget_target",
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
			recoveryOrder = append(recoveryOrder, req.Profile.ID+":"+req.Path)
			return provider.UploadResult{
				OperationResult: provider.OperationResult{
					OK:      true,
					Status:  "ok",
					Message: "profile budget recovered",
					Mode:    "fake_profile_budget_ok",
				},
			}
		},
	}

	registry := provider.NewRegistry(adapter)
	authSvc := auth.NewService(store, registry)
	svc := NewService(store, registry, authSvc)

	createProfile := func(name string) string {
		profile, err := authSvc.CreateProfile(ctx, auth.CreateProfileInput{
			ProviderKey: "recover_profile_budget_target",
			AuthMode:    "manual_token",
			DisplayName: name,
			Token:       "token-" + name,
		})
		if err != nil {
			t.Fatalf("CreateProfile(%s) error = %v", name, err)
		}
		return profile.ID
	}
	profileA := createProfile("profile-a")
	profileB := createProfile("profile-b")

	createBlockedTask := func(profileID, path string, offsetMinutes int) {
		now := time.Now().Add(time.Duration(offsetMinutes) * time.Minute).UTC().Format(time.RFC3339)
		detail, err := svc.Create(ctx, CreateRequest{
			SourceProvider:  "guangya",
			TargetProvider:  "recover_profile_budget_target",
			TargetProfileID: profileID,
			ThresholdMB:     1,
			RiskOverride: &planner.RiskProfileOverride{
				MaxConcurrent: intPtrTask(3),
			},
			Entries: []planner.SourceEntry{
				{Path: path, Size: 256, MD5: strings.Trim(path, "/")},
			},
		})
		if err != nil {
			t.Fatalf("Create(%s %s) error = %v", profileID, path, err)
		}
		blocked, ok, err := svc.Get(ctx, detail.Task.ID)
		if err != nil || !ok {
			t.Fatalf("Get(%s %s) error=%v ok=%v", profileID, path, err, ok)
		}
		blocked.Task.State = StateBlocked
		blocked.Task.UpdatedAt = now
		blocked.Runtime = initializeRuntimeState(blocked.Plan)
		blocked.Runtime.ExecutionState = "blocked"
		blocked.Runtime.RetryQueue = []RetryQueueItem{{
			Path:           path,
			Retryable:      true,
			RetryClass:     "retry_failed",
			ProviderStatus: "provider_request_failed",
		}}
		blocked.Results = []Result{{
			ID:        uuid.NewString(),
			TaskID:    blocked.Task.ID,
			ItemID:    blocked.Items[0].ID,
			Status:    "failed",
			Mode:      "fake_failed",
			Message:   "failed",
			CreatedAt: now,
			Payload: map[string]interface{}{
				"path":           path,
				"providerStatus": "provider_request_failed",
			},
		}}
		applyRetryQueueSummary(&blocked.Runtime, blocked.Plan.Metadata)
		if err := replaceTaskDetailAndResults(ctx, store, blocked); err != nil {
			t.Fatalf("replaceTaskDetailAndResults(%s %s) error = %v", profileID, path, err)
		}
	}

	createBlockedTask(profileA, "/a-1.bin", -182)
	createBlockedTask(profileA, "/a-2.bin", -181)
	createBlockedTask(profileB, "/b-1.bin", -180)

	result, err := svc.RecoverBlockedTasksWithOptions(ctx, RecoverOptions{Limit: 3, LimitPerProfile: 1})
	if err != nil {
		t.Fatalf("RecoverBlockedTasksWithOptions(profile budget) error = %v", err)
	}
	if result.MatchedCount != 3 || result.RecoveredCount != 2 || result.SkippedByProfileBudget != 1 || result.LimitPerProfile != 1 {
		t.Fatalf("unexpected profile budget result: %#v", result)
	}
	if len(recoveryOrder) != 2 {
		t.Fatalf("expected 2 recovery operations, got %#v", recoveryOrder)
	}
	if recoveryOrder[0] != profileA+":/a-1.bin" || recoveryOrder[1] != profileB+":/b-1.bin" {
		t.Fatalf("expected profile budget order [a1 b1], got %#v", recoveryOrder)
	}
}

func TestServiceRecoverBlockedTasksWithOptionsRespectsModeBudget(t *testing.T) {
	ctx := context.Background()
	store, err := sqlitestore.New(ctx, filepath.Join(t.TempDir(), "auto-recover-mode-budget.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer func() { _ = store.Close() }()

	recoveryOrder := make([]string, 0, 3)
	adapter := &scriptedAdapter{
		meta: provider.Provider{
			Key:              "recover_mode_budget_target",
			DisplayName:      "recover_mode_budget_target",
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
			recoveryOrder = append(recoveryOrder, req.Profile.ID+":"+req.Path)
			return provider.UploadResult{
				OperationResult: provider.OperationResult{
					OK:      true,
					Status:  "ok",
					Message: "mode budget recovered",
					Mode:    "fake_mode_budget_ok",
				},
			}
		},
	}

	registry := provider.NewRegistry(adapter)
	authSvc := auth.NewService(store, registry)
	svc := NewService(store, registry, authSvc)

	createProfile := func(name string) string {
		profile, err := authSvc.CreateProfile(ctx, auth.CreateProfileInput{
			ProviderKey: "recover_mode_budget_target",
			AuthMode:    "manual_token",
			DisplayName: name,
			Token:       "token-" + name,
		})
		if err != nil {
			t.Fatalf("CreateProfile(%s) error = %v", name, err)
		}
		return profile.ID
	}

	createBlockedTask := func(profileID, path string, offsetMinutes int) {
		now := time.Now().Add(time.Duration(offsetMinutes) * time.Minute).UTC().Format(time.RFC3339)
		detail, err := svc.Create(ctx, CreateRequest{
			SourceProvider:  "guangya",
			TargetProvider:  "recover_mode_budget_target",
			TargetProfileID: profileID,
			ThresholdMB:     1,
			RiskOverride: &planner.RiskProfileOverride{
				MaxConcurrent: intPtrTask(4),
			},
			Entries: []planner.SourceEntry{
				{Path: path, Size: 256, MD5: strings.Trim(path, "/")},
			},
		})
		if err != nil {
			t.Fatalf("Create(%s %s) error = %v", profileID, path, err)
		}
		blocked, ok, err := svc.Get(ctx, detail.Task.ID)
		if err != nil || !ok {
			t.Fatalf("Get(%s %s) error=%v ok=%v", profileID, path, err, ok)
		}
		blocked.Task.State = StateBlocked
		blocked.Task.UpdatedAt = now
		blocked.Runtime = initializeRuntimeState(blocked.Plan)
		blocked.Runtime.ExecutionState = "blocked"
		blocked.Runtime.RetryQueue = []RetryQueueItem{{
			Path:           path,
			Retryable:      true,
			RetryClass:     "retry_failed",
			ProviderStatus: "provider_request_failed",
		}}
		blocked.Results = []Result{{
			ID:        uuid.NewString(),
			TaskID:    blocked.Task.ID,
			ItemID:    blocked.Items[0].ID,
			Status:    "failed",
			Mode:      "fake_failed",
			Message:   "failed",
			CreatedAt: now,
			Payload: map[string]interface{}{
				"path":           path,
				"providerStatus": "provider_request_failed",
			},
		}}
		applyRetryQueueSummary(&blocked.Runtime, blocked.Plan.Metadata)
		if err := replaceTaskDetailAndResults(ctx, store, blocked); err != nil {
			t.Fatalf("replaceTaskDetailAndResults(%s %s) error = %v", profileID, path, err)
		}
	}

	createBlockedTask(createProfile("mode-profile-a"), "/mode-a.bin", -183)
	createBlockedTask(createProfile("mode-profile-b"), "/mode-b.bin", -182)
	createBlockedTask(createProfile("mode-profile-c"), "/mode-c.bin", -181)

	result, err := svc.RecoverBlockedTasksWithOptions(ctx, RecoverOptions{Limit: 3, LimitPerMode: 1})
	if err != nil {
		t.Fatalf("RecoverBlockedTasksWithOptions(mode budget) error = %v", err)
	}
	if result.MatchedCount != 3 || result.RecoveredCount != 1 || result.SkippedByModeBudget != 2 || result.LimitPerMode != 1 {
		t.Fatalf("unexpected mode budget result: %#v", result)
	}
	if len(recoveryOrder) != 1 || recoveryOrder[0] == "" {
		t.Fatalf("expected exactly one recovered item under mode budget, got %#v", recoveryOrder)
	}
}

func TestServiceRecoverBlockedTasksWithOptionsRespectsLaneBudget(t *testing.T) {
	ctx := context.Background()
	store, err := sqlitestore.New(ctx, filepath.Join(t.TempDir(), "auto-recover-lane-budget.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer func() { _ = store.Close() }()

	recoveryOrder := make([]string, 0, 3)
	adapter := &scriptedAdapter{
		meta: provider.Provider{
			Key:              "recover_lane_budget_target",
			DisplayName:      "recover_lane_budget_target",
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
			recoveryOrder = append(recoveryOrder, req.Profile.ID+":"+req.Path)
			return provider.UploadResult{
				OperationResult: provider.OperationResult{
					OK:      true,
					Status:  "ok",
					Message: "lane budget recovered",
					Mode:    "fake_lane_budget_ok",
				},
			}
		},
	}

	registry := provider.NewRegistry(adapter)
	authSvc := auth.NewService(store, registry)
	svc := NewService(store, registry, authSvc)

	createProfile := func(name string) string {
		profile, err := authSvc.CreateProfile(ctx, auth.CreateProfileInput{
			ProviderKey: "recover_lane_budget_target",
			AuthMode:    "manual_token",
			DisplayName: name,
			Token:       "token-" + name,
		})
		if err != nil {
			t.Fatalf("CreateProfile(%s) error = %v", name, err)
		}
		return profile.ID
	}

	createBlockedTask := func(profileID, path, retryClass string, offsetMinutes int) {
		now := time.Now().Add(time.Duration(offsetMinutes) * time.Minute).UTC().Format(time.RFC3339)
		providerStatus := "provider_request_failed"
		eligibleAt := ""
		if retryClass == "rate_limited" {
			providerStatus = "rate_limited"
			eligibleAt = time.Now().Add(-5 * time.Minute).UTC().Format(time.RFC3339)
		}
		detail, err := svc.Create(ctx, CreateRequest{
			SourceProvider:  "guangya",
			TargetProvider:  "recover_lane_budget_target",
			TargetProfileID: profileID,
			ThresholdMB:     1,
			RiskOverride: &planner.RiskProfileOverride{
				MaxConcurrent: intPtrTask(4),
			},
			Entries: []planner.SourceEntry{
				{Path: path, Size: 256, MD5: strings.Trim(path, "/")},
			},
		})
		if err != nil {
			t.Fatalf("Create(%s %s) error = %v", profileID, path, err)
		}
		blocked, ok, err := svc.Get(ctx, detail.Task.ID)
		if err != nil || !ok {
			t.Fatalf("Get(%s %s) error=%v ok=%v", profileID, path, err, ok)
		}
		blocked.Task.State = StateBlocked
		blocked.Task.UpdatedAt = now
		blocked.Runtime = initializeRuntimeState(blocked.Plan)
		blocked.Runtime.ExecutionState = "blocked"
		blocked.Runtime.RetryQueue = []RetryQueueItem{{
			Path:           path,
			Retryable:      true,
			RetryClass:     retryClass,
			ProviderStatus: providerStatus,
			EligibleAt:     eligibleAt,
		}}
		blocked.Results = []Result{{
			ID:        uuid.NewString(),
			TaskID:    blocked.Task.ID,
			ItemID:    blocked.Items[0].ID,
			Status:    "failed",
			Mode:      "fake_failed",
			Message:   "failed",
			CreatedAt: now,
			Payload: map[string]interface{}{
				"path":           path,
				"providerStatus": providerStatus,
			},
		}}
		applyRetryQueueSummary(&blocked.Runtime, blocked.Plan.Metadata)
		if err := replaceTaskDetailAndResults(ctx, store, blocked); err != nil {
			t.Fatalf("replaceTaskDetailAndResults(%s %s) error = %v", profileID, path, err)
		}
	}

	profileA := createProfile("lane-profile-a")
	profileB := createProfile("lane-profile-b")
	profileC := createProfile("lane-profile-c")
	createBlockedTask(profileA, "/lane-a.bin", "retry_failed", -183)
	createBlockedTask(profileB, "/lane-b.bin", "retry_failed", -182)
	createBlockedTask(profileC, "/lane-c.bin", "rate_limited", -181)

	result, err := svc.RecoverBlockedTasksWithOptions(ctx, RecoverOptions{Limit: 3, LimitPerLane: 1})
	if err != nil {
		t.Fatalf("RecoverBlockedTasksWithOptions(lane budget) error = %v", err)
	}
	if result.MatchedCount != 3 || result.RecoveredCount != 2 || result.SkippedByLaneBudget != 1 || result.LimitPerLane != 1 {
		t.Fatalf("unexpected lane budget result: %#v", result)
	}
	if len(recoveryOrder) != 2 {
		t.Fatalf("expected two recovered items under lane budget, got %#v", recoveryOrder)
	}
	if !strings.HasSuffix(recoveryOrder[0], ":/lane-a.bin") || !strings.HasSuffix(recoveryOrder[1], ":/lane-c.bin") {
		t.Fatalf("expected lane budget to keep one retry_failed lane and one rate_limited lane, got %#v", recoveryOrder)
	}
}

func TestServiceRecoverBlockedTasksWithOptionsFiltersRecoverState(t *testing.T) {
	ctx := context.Background()
	store, err := sqlitestore.New(ctx, filepath.Join(t.TempDir(), "auto-recover-state-filter.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer func() { _ = store.Close() }()

	uploadCallsByPath := map[string]int{}
	newAdapter := func(key string) *scriptedAdapter {
		return &scriptedAdapter{
			meta: provider.Provider{
				Key:              key,
				DisplayName:      key,
				ProtocolGroup:    "state-filter",
				AuthModes:        []string{"manual_token"},
				FastUploadInputs: []string{"md5", "size"},
				FallbackModes:    []string{"download_upload"},
				Status:           "planned",
			},
			capability: provider.CapabilitySet{SupportsAuthValidation: true, SupportsUpload: true},
			uploadFunc: func(req provider.UploadRequest) provider.UploadResult {
				uploadCallsByPath[req.Path]++
				if uploadCallsByPath[req.Path] == 1 {
					return provider.UploadResult{OperationResult: provider.OperationResult{Status: "rate_limited", Message: "rate limited", Mode: "fake_rate_limit"}}
				}
				return provider.UploadResult{OperationResult: provider.OperationResult{OK: true, Status: "ok", Message: "state recovered", Mode: "fake_state_ok"}}
			},
		}
	}

	registry := provider.NewRegistry(
		newAdapter("recover_state_runnable_target"),
		newAdapter("recover_state_window_target"),
		newAdapter("recover_state_cooldown_target"),
		newAdapter("recover_state_auth_target"),
		newAdapter("recover_state_local_target"),
		newAdapter("recover_state_provider_session_target"),
		newAdapter("recover_state_manual_target"),
		newAdapter("recover_state_retry_limit_target"),
	)
	authSvc := auth.NewService(store, registry)
	svc := NewService(store, registry, authSvc)

	createProfile := func(providerKey string) string {
		profile, err := authSvc.CreateProfile(ctx, auth.CreateProfileInput{ProviderKey: providerKey, AuthMode: "manual_token", DisplayName: providerKey, Token: "token-" + providerKey})
		if err != nil {
			t.Fatalf("CreateProfile(%s) error = %v", providerKey, err)
		}
		return profile.ID
	}

	createBlockedTask := func(providerKey, profileID, path string, override *planner.RiskProfileOverride, createdAt string) string {
		detail, err := svc.Create(ctx, CreateRequest{
			SourceProvider:  "guangya",
			TargetProvider:  providerKey,
			TargetProfileID: profileID,
			ThresholdMB:     1,
			RiskOverride:    override,
			Entries:         []planner.SourceEntry{{Path: path, Size: 128, MD5: strings.Trim(path, "/")}},
		})
		if err != nil {
			t.Fatalf("Create(%s %s) error = %v", providerKey, path, err)
		}
		if _, ok, err := svc.Run(ctx, detail.Task.ID); err != nil || !ok {
			t.Fatalf("Run(%s %s) error=%v ok=%v", providerKey, path, err, ok)
		}
		blocked, ok, err := svc.Get(ctx, detail.Task.ID)
		if err != nil || !ok {
			t.Fatalf("Get(%s %s) error=%v ok=%v", providerKey, path, err, ok)
		}
		for idx := range blocked.Results {
			blocked.Results[idx].CreatedAt = createdAt
		}
		blocked.Task.State = StateBlocked
		blocked.Task.UpdatedAt = createdAt
		syncRuntimeRetryQueue(&blocked.Runtime, blocked.Plan.Metadata, blocked.Results)
		applyRetryQueueSummary(&blocked.Runtime, blocked.Plan.Metadata)
		if err := replaceTaskDetailAndResults(ctx, store, blocked); err != nil {
			t.Fatalf("replaceTaskDetailAndResults(%s %s) error = %v", providerKey, path, err)
		}
		return detail.Task.ID
	}

	createInjectedBlockedTask := func(providerKey, profileID, path string, queue []RetryQueueItem, updatedAt string) string {
		detail, err := svc.Create(ctx, CreateRequest{
			SourceProvider:  "guangya",
			TargetProvider:  providerKey,
			TargetProfileID: profileID,
			ThresholdMB:     1,
			Entries:         []planner.SourceEntry{{Path: path, Size: 128, MD5: strings.Trim(path, "/")}},
		})
		if err != nil {
			t.Fatalf("Create(injected %s %s) error = %v", providerKey, path, err)
		}
		blocked, ok, err := svc.Get(ctx, detail.Task.ID)
		if err != nil || !ok {
			t.Fatalf("Get(injected %s %s) error=%v ok=%v", providerKey, path, err, ok)
		}
		blocked.Task.State = StateBlocked
		blocked.Task.UpdatedAt = updatedAt
		blocked.Runtime.ExecutionState = string(StateBlocked)
		blocked.Runtime.RetryQueue = queue
		applyRetryQueueSummary(&blocked.Runtime, blocked.Plan.Metadata)
		if err := replaceTaskDetailAndResults(ctx, store, blocked); err != nil {
			t.Fatalf("replaceTaskDetailAndResults(injected %s %s) error = %v", providerKey, path, err)
		}
		return detail.Task.ID
	}

	nowHour := time.Now().UTC().Hour()
	startHour := (nowHour + 1) % 24
	endHour := (nowHour + 2) % 24
	if startHour == endHour {
		endHour = (endHour + 1) % 24
	}
	windowTimestamp := time.Now().Add(-2 * time.Hour).UTC().Format(time.RFC3339)
	runnableProfile := createProfile("recover_state_runnable_target")
	runnableID := createBlockedTask("recover_state_runnable_target", runnableProfile, "/runnable.bin", &planner.RiskProfileOverride{CooldownSeconds: intPtrTask(0)}, windowTimestamp)

	runnableResult, err := svc.RecoverBlockedTasksWithOptions(ctx, RecoverOptions{RecoverState: "runnable_now", Limit: 5, IncludeNonRunnable: true})
	if err != nil {
		t.Fatalf("RecoverBlockedTasksWithOptions(runnable_now) error = %v", err)
	}
	if runnableResult.RecoverState != "runnable_now" || runnableResult.MatchedCount != 1 || runnableResult.RecoveredCount != 1 {
		t.Fatalf("unexpected runnable recover result: %#v", runnableResult)
	}

	runnableTask, ok, err := svc.Get(ctx, runnableID)
	if err != nil || !ok || runnableTask.Task.State != StateCompleted {
		t.Fatalf("expected runnable task completed, got ok=%v err=%v detail=%#v", ok, err, runnableTask)
	}

	authProfile := createProfile("recover_state_auth_target")
	localProfile := createProfile("recover_state_local_target")
	providerSessionProfile := createProfile("recover_state_provider_session_target")
	manualProfile := createProfile("recover_state_manual_target")
	retryLimitProfile := createProfile("recover_state_retry_limit_target")
	injectedTimestamp := time.Now().Add(-90 * time.Minute).UTC().Format(time.RFC3339)
	_ = createInjectedBlockedTask("recover_state_auth_target", authProfile, "/auth.bin", []RetryQueueItem{
		{Path: "/auth.bin", RetryClass: "auth_expired", RetryAction: "refresh_auth", Retryable: false, Blocked: true, RemainingCount: 1},
	}, injectedTimestamp)
	_ = createInjectedBlockedTask("recover_state_local_target", localProfile, "/local.bin", []RetryQueueItem{
		{Path: "/local.bin", RetryClass: "local_file_missing", RetryAction: "restore_local", Retryable: false, Blocked: true, RemainingCount: 1},
	}, injectedTimestamp)
	_ = createInjectedBlockedTask("recover_state_provider_session_target", providerSessionProfile, "/provider-session.bin", []RetryQueueItem{
		{Path: "/provider-session.bin", RetryClass: "provider_session_missing", RetryAction: "manual_intervention_required", Retryable: false, Blocked: true, RemainingCount: 1},
	}, injectedTimestamp)
	_ = createInjectedBlockedTask("recover_state_manual_target", manualProfile, "/manual.bin", []RetryQueueItem{
		{Path: "/manual.bin", RetryClass: "pending_manual", RetryAction: "manual_confirm", Retryable: false, Blocked: true, RemainingCount: 1},
	}, injectedTimestamp)
	_ = createInjectedBlockedTask("recover_state_retry_limit_target", retryLimitProfile, "/retry-limit.bin", []RetryQueueItem{
		{Path: "/retry-limit.bin", RetryClass: "retry_failed", RetryAction: "retry", Retryable: false, Blocked: true, Exhausted: true, AttemptCount: 3, RetryLimit: 3, RemainingCount: 0},
	}, injectedTimestamp)

	evidence, err := svc.RuntimeEvidence(ctx)
	if err != nil {
		t.Fatalf("RuntimeEvidence(state filter) error = %v", err)
	}
	authLane := autoRecoverLaneByMode(evidence.AutoRecoverPool, "auth_refresh_required")
	if authLane.Mode == "" || authLane.WaitingAuthRefreshTaskCount != 1 {
		t.Fatalf("expected auth_refresh_required lane with count 1, got %#v", authLane)
	}
	localLane := autoRecoverLaneByMode(evidence.AutoRecoverPool, "local_restore_required")
	if localLane.Mode == "" || localLane.WaitingLocalRestoreTaskCount != 1 {
		t.Fatalf("expected local_restore_required lane with count 1, got %#v", localLane)
	}
	providerSessionLane := autoRecoverLaneByMode(evidence.AutoRecoverPool, "auto_retry")
	if providerSessionLane.Mode != "auto_retry" || providerSessionLane.WaitingProviderSessionTaskCount != 1 {
		t.Fatalf("expected auto_retry lane with waiting provider session count 1, got %#v", providerSessionLane)
	}
	manualLane := autoRecoverLaneByMode(evidence.AutoRecoverPool, "manual_confirmation_required")
	if manualLane.Mode == "" || manualLane.WaitingManualTaskCount != 1 {
		t.Fatalf("expected manual_confirmation_required lane with count 1, got %#v", manualLane)
	}
	retryLimitLane := autoRecoverLaneByMode(evidence.AutoRecoverPool, "retry_limit_blocked")
	if retryLimitLane.Mode == "" || retryLimitLane.WaitingRetryLimitTaskCount != 1 {
		t.Fatalf("expected retry_limit_blocked lane with count 1, got %#v", retryLimitLane)
	}
	if evidence.AutoRecoverWaitingAuthRefreshTasks != 1 ||
		evidence.AutoRecoverWaitingLocalRestoreTasks != 1 ||
		evidence.AutoRecoverWaitingProviderSessionTasks != 1 ||
		evidence.AutoRecoverWaitingManualTasks != 1 ||
		evidence.AutoRecoverWaitingRetryLimitTasks != 1 {
		t.Fatalf("expected global detailed waiting counts 1/1/1/1/1, got auth=%d local=%d providerSession=%d manual=%d retryLimit=%d",
			evidence.AutoRecoverWaitingAuthRefreshTasks,
			evidence.AutoRecoverWaitingLocalRestoreTasks,
			evidence.AutoRecoverWaitingProviderSessionTasks,
			evidence.AutoRecoverWaitingManualTasks,
			evidence.AutoRecoverWaitingRetryLimitTasks,
		)
	}

	authResult, err := svc.RecoverBlockedTasksWithOptions(ctx, RecoverOptions{RecoverState: "waiting_auth_refresh", Limit: 5, IncludeNonRunnable: true, DryRun: true})
	if err != nil {
		t.Fatalf("RecoverBlockedTasksWithOptions(waiting_auth_refresh) error = %v", err)
	}
	if authResult.MatchedCount != 1 || authResult.RecoveredCount != 0 || authResult.SkippedByBlockedReason != 1 {
		t.Fatalf("unexpected auth refresh recover result: %#v", authResult)
	}
	if len(authResult.Decisions) == 0 || authResult.Decisions[0].RecoverState != "waiting_auth_refresh" {
		t.Fatalf("expected auth refresh decision state, got %#v", authResult.Decisions)
	}

	localResult, err := svc.RecoverBlockedTasksWithOptions(ctx, RecoverOptions{RecoverState: "waiting_local_restore", Limit: 5, IncludeNonRunnable: true, DryRun: true})
	if err != nil {
		t.Fatalf("RecoverBlockedTasksWithOptions(waiting_local_restore) error = %v", err)
	}
	if localResult.MatchedCount != 1 || localResult.RecoveredCount != 0 || localResult.SkippedByBlockedReason != 1 {
		t.Fatalf("unexpected local restore recover result: %#v", localResult)
	}
	if len(localResult.Decisions) == 0 || localResult.Decisions[0].RecoverState != "waiting_local_restore" {
		t.Fatalf("expected local restore decision state, got %#v", localResult.Decisions)
	}

	providerSessionResult, err := svc.RecoverBlockedTasksWithOptions(ctx, RecoverOptions{RecoverState: "waiting_provider_session", Limit: 5, IncludeNonRunnable: true, DryRun: true})
	if err != nil {
		t.Fatalf("RecoverBlockedTasksWithOptions(waiting_provider_session) error = %v", err)
	}
	if providerSessionResult.MatchedCount != 1 || providerSessionResult.RecoveredCount != 0 || providerSessionResult.SkippedByBlockedReason != 1 {
		t.Fatalf("unexpected provider session recover result: %#v", providerSessionResult)
	}
	if len(providerSessionResult.Decisions) == 0 || providerSessionResult.Decisions[0].RecoverState != "waiting_provider_session" {
		t.Fatalf("expected provider session decision state, got %#v", providerSessionResult.Decisions)
	}

	manualResult, err := svc.RecoverBlockedTasksWithOptions(ctx, RecoverOptions{RecoverState: "waiting_manual_confirmation", Limit: 5, IncludeNonRunnable: true, DryRun: true})
	if err != nil {
		t.Fatalf("RecoverBlockedTasksWithOptions(waiting_manual_confirmation) error = %v", err)
	}
	if manualResult.MatchedCount != 1 || manualResult.RecoveredCount != 0 || manualResult.SkippedByBlockedReason != 1 {
		t.Fatalf("unexpected manual confirmation recover result: %#v", manualResult)
	}
	if len(manualResult.Decisions) == 0 || manualResult.Decisions[0].RecoverState != "waiting_manual_confirmation" {
		t.Fatalf("expected manual confirmation decision state, got %#v", manualResult.Decisions)
	}

	retryLimitResult, err := svc.RecoverBlockedTasksWithOptions(ctx, RecoverOptions{RecoverState: "waiting_retry_limit", Limit: 5, IncludeNonRunnable: true, DryRun: true})
	if err != nil {
		t.Fatalf("RecoverBlockedTasksWithOptions(waiting_retry_limit) error = %v", err)
	}
	if retryLimitResult.MatchedCount != 1 || retryLimitResult.RecoveredCount != 0 || retryLimitResult.SkippedByBlockedReason != 1 {
		t.Fatalf("unexpected retry limit recover result: %#v", retryLimitResult)
	}
	if len(retryLimitResult.Decisions) == 0 || retryLimitResult.Decisions[0].RecoverState != "waiting_retry_limit" {
		t.Fatalf("expected retry limit decision state, got %#v", retryLimitResult.Decisions)
	}
}

func TestServiceRecoverBlockedTasksWithOptionsExecutesWaitingManualConfirmation(t *testing.T) {
	ctx := context.Background()
	store, err := sqlitestore.New(ctx, filepath.Join(t.TempDir(), "auto-recover-manual-execute.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer func() { _ = store.Close() }()

	uploadCalls := 0
	targetAdapter := &scriptedAdapter{
		meta: provider.Provider{
			Key:              "manual_confirmation_execute_target",
			DisplayName:      "Manual Confirmation Execute Target",
			ProtocolGroup:    "manual_execute_group",
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
			uploadCalls++
			if uploadCalls == 1 {
				return provider.UploadResult{
					OperationResult: provider.OperationResult{
						Status:  "pending_manual_requires_confirmation",
						Message: "pending manual",
						Mode:    "manual_execute_pending",
						Payload: map[string]interface{}{
							"providerStatus": "pending_manual_requires_confirmation",
						},
					},
				}
			}
			return provider.UploadResult{
				OperationResult: provider.OperationResult{
					OK:      true,
					Status:  "ok",
					Message: "manual confirmed",
					Mode:    "manual_execute_ok",
				},
			}
		},
	}

	registry := provider.NewRegistry(targetAdapter)
	authSvc := auth.NewService(store, registry)
	svc := NewService(store, registry, authSvc)
	profile, err := authSvc.CreateProfile(ctx, auth.CreateProfileInput{
		ProviderKey: "manual_confirmation_execute_target",
		AuthMode:    "manual_token",
		DisplayName: "Manual Confirmation Execute Target",
		Token:       "token-manual-confirmation-execute",
	})
	if err != nil {
		t.Fatalf("CreateProfile() error = %v", err)
	}

	detail, err := svc.Create(ctx, CreateRequest{
		SourceProvider:  "manual_confirmation_execute_source",
		TargetProvider:  "manual_confirmation_execute_target",
		TargetProfileID: profile.ID,
		ThresholdMB:     1,
		Entries: []planner.SourceEntry{{
			Path: "/manual.bin",
			Size: 1024,
			MD5:  "manual-md5",
		}},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	blocked, ok, err := svc.Run(ctx, detail.Task.ID)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !ok {
		t.Fatal("expected task to exist")
	}
	if blocked.Task.State != StateBlocked {
		t.Fatalf("expected blocked after pending manual, got %s", blocked.Task.State)
	}
	if uploadCalls != 1 {
		t.Fatalf("expected initial upload calls 1, got %d", uploadCalls)
	}

	result, err := svc.RecoverBlockedTasksWithOptions(ctx, RecoverOptions{RecoverState: "waiting_manual_confirmation", Limit: 5, IncludeNonRunnable: true})
	if err != nil {
		t.Fatalf("RecoverBlockedTasksWithOptions(waiting_manual_confirmation) error = %v", err)
	}
	if result.MatchedCount != 1 || result.RecoveredCount != 1 {
		t.Fatalf("expected manual confirmation recovery to execute, got %#v", result)
	}
	if len(result.Decisions) == 0 || result.Decisions[0].Outcome != "recovered" {
		t.Fatalf("expected recovered decision, got %#v", result.Decisions)
	}
	if uploadCalls != 2 {
		t.Fatalf("expected manual confirmation recovery to run upload twice, got %d", uploadCalls)
	}

	detailAfter, ok, err := svc.Get(ctx, detail.Task.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !ok {
		t.Fatal("expected task to exist after recovery")
	}
	if detailAfter.Task.State != StateCompleted {
		t.Fatalf("expected completed after manual confirmation recovery, got %s", detailAfter.Task.State)
	}
}
func TestServiceRecoverBlockedTasksWithOptionsFiltersProtocolGroup(t *testing.T) {
	ctx := context.Background()
	store, err := sqlitestore.New(ctx, filepath.Join(t.TempDir(), "auto-recover-protocol-group-filter.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer func() { _ = store.Close() }()

	uploadCallsByPath := map[string]int{}
	newAdapter := func(key, protocolGroup string) *scriptedAdapter {
		return &scriptedAdapter{
			meta: provider.Provider{
				Key:              key,
				DisplayName:      key,
				ProtocolGroup:    protocolGroup,
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
				uploadCallsByPath[req.Path]++
				if uploadCallsByPath[req.Path] == 1 {
					return provider.UploadResult{
						OperationResult: provider.OperationResult{
							Status:  "rate_limited",
							Message: "rate limited",
							Mode:    "fake_rate_limit",
						},
					}
				}
				return provider.UploadResult{
					OperationResult: provider.OperationResult{
						OK:      true,
						Status:  "ok",
						Message: "protocol group recovered",
						Mode:    "fake_protocol_group_ok",
					},
				}
			},
		}
	}

	adapterA := newAdapter("recover_protocol_group_target_a", "protocol_group_a")
	adapterB := newAdapter("recover_protocol_group_target_b", "protocol_group_b")
	registry := provider.NewRegistry(adapterA, adapterB)
	authSvc := auth.NewService(store, registry)
	svc := NewService(store, registry, authSvc)

	createProfile := func(providerKey string) string {
		profile, err := authSvc.CreateProfile(ctx, auth.CreateProfileInput{
			ProviderKey: providerKey,
			AuthMode:    "manual_token",
			DisplayName: providerKey,
			Token:       "token-" + providerKey,
		})
		if err != nil {
			t.Fatalf("CreateProfile(%s) error = %v", providerKey, err)
		}
		return profile.ID
	}

	createBlockedTask := func(providerKey, profileID, path string, offsetMinutes int) string {
		detail, err := svc.Create(ctx, CreateRequest{
			SourceProvider:  "guangya",
			TargetProvider:  providerKey,
			TargetProfileID: profileID,
			ThresholdMB:     1,
			RiskOverride: &planner.RiskProfileOverride{
				CooldownSeconds: intPtrTask(3600),
				MaxConcurrent:   intPtrTask(3),
			},
			Entries: []planner.SourceEntry{{Path: path, Size: 128, MD5: strings.Trim(path, "/")}},
		})
		if err != nil {
			t.Fatalf("Create(%s %s) error = %v", providerKey, path, err)
		}
		if _, ok, err := svc.Run(ctx, detail.Task.ID); err != nil || !ok {
			t.Fatalf("Run(%s %s) error=%v ok=%v", providerKey, path, err, ok)
		}
		blocked, ok, err := svc.Get(ctx, detail.Task.ID)
		if err != nil || !ok {
			t.Fatalf("Get(%s %s) error=%v ok=%v", providerKey, path, err, ok)
		}
		timestamp := time.Now().Add(time.Duration(offsetMinutes) * time.Minute).UTC().Format(time.RFC3339)
		blocked.Task.UpdatedAt = timestamp
		for idx := range blocked.Results {
			blocked.Results[idx].CreatedAt = timestamp
		}
		syncRuntimeRetryQueue(&blocked.Runtime, blocked.Plan.Metadata, blocked.Results)
		applyRetryQueueSummary(&blocked.Runtime, blocked.Plan.Metadata)
		if err := replaceTaskDetailAndResults(ctx, store, blocked); err != nil {
			t.Fatalf("replaceTaskDetailAndResults(%s %s) error = %v", providerKey, path, err)
		}
		return detail.Task.ID
	}

	profileA := createProfile("recover_protocol_group_target_a")
	profileB := createProfile("recover_protocol_group_target_b")
	firstID := createBlockedTask("recover_protocol_group_target_a", profileA, "/group-a.bin", -120)
	secondID := createBlockedTask("recover_protocol_group_target_b", profileB, "/group-b.bin", -119)

	result, err := svc.RecoverBlockedTasksWithOptions(ctx, RecoverOptions{
		ProtocolGroup: "protocol_group_b",
		Limit:         5,
	})
	if err != nil {
		t.Fatalf("RecoverBlockedTasksWithOptions(protocolGroup filter) error = %v", err)
	}
	if result.ProtocolGroup != "protocol_group_b" || result.MatchedCount != 1 || result.RecoveredCount != 1 {
		t.Fatalf("unexpected protocolGroup filter result: %#v", result)
	}

	firstTask, ok, err := svc.Get(ctx, firstID)
	if err != nil || !ok {
		t.Fatalf("Get(first protocol group task) error=%v ok=%v", err, ok)
	}
	secondTask, ok, err := svc.Get(ctx, secondID)
	if err != nil || !ok {
		t.Fatalf("Get(second protocol group task) error=%v ok=%v", err, ok)
	}
	if firstTask.Task.State != StateBlocked {
		t.Fatalf("expected first task to remain blocked, got %s", firstTask.Task.State)
	}
	if secondTask.Task.State != StateCompleted {
		t.Fatalf("expected second task completed, got %s", secondTask.Task.State)
	}
	if uploadCallsByPath["/group-a.bin"] != 1 {
		t.Fatalf("expected /group-a.bin upload calls 1, got %d", uploadCallsByPath["/group-a.bin"])
	}
	if uploadCallsByPath["/group-b.bin"] != 2 {
		t.Fatalf("expected /group-b.bin upload calls 2, got %d", uploadCallsByPath["/group-b.bin"])
	}
}

func TestServiceRecoverBlockedTasksWithOptionsRespectsProtocolGroupBudget(t *testing.T) {
	ctx := context.Background()
	store, err := sqlitestore.New(ctx, filepath.Join(t.TempDir(), "auto-recover-protocol-group-budget.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer func() { _ = store.Close() }()

	uploadCallsByPath := map[string]int{}
	newAdapter := func(key, protocolGroup string) *scriptedAdapter {
		return &scriptedAdapter{
			meta: provider.Provider{
				Key:              key,
				DisplayName:      key,
				ProtocolGroup:    protocolGroup,
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
				uploadCallsByPath[req.Path]++
				if uploadCallsByPath[req.Path] == 1 {
					return provider.UploadResult{
						OperationResult: provider.OperationResult{
							Status:  "rate_limited",
							Message: "rate limited",
							Mode:    "fake_rate_limit",
						},
					}
				}
				return provider.UploadResult{
					OperationResult: provider.OperationResult{
						OK:      true,
						Status:  "ok",
						Message: "protocol group budget recovered",
						Mode:    "fake_protocol_group_budget_ok",
					},
				}
			},
		}
	}

	adapterA := newAdapter("recover_protocol_group_budget_target_a", "shared_protocol_group")
	adapterB := newAdapter("recover_protocol_group_budget_target_b", "shared_protocol_group")
	adapterC := newAdapter("recover_protocol_group_budget_target_c", "other_protocol_group")
	registry := provider.NewRegistry(adapterA, adapterB, adapterC)
	authSvc := auth.NewService(store, registry)
	svc := NewService(store, registry, authSvc)

	createProfile := func(providerKey string) string {
		profile, err := authSvc.CreateProfile(ctx, auth.CreateProfileInput{
			ProviderKey: providerKey,
			AuthMode:    "manual_token",
			DisplayName: providerKey,
			Token:       "token-" + providerKey,
		})
		if err != nil {
			t.Fatalf("CreateProfile(%s) error = %v", providerKey, err)
		}
		return profile.ID
	}

	createBlockedTask := func(providerKey, profileID, path string, offsetMinutes int) string {
		detail, err := svc.Create(ctx, CreateRequest{
			SourceProvider:  "guangya",
			TargetProvider:  providerKey,
			TargetProfileID: profileID,
			ThresholdMB:     1,
			RiskOverride: &planner.RiskProfileOverride{
				CooldownSeconds: intPtrTask(3600),
				MaxConcurrent:   intPtrTask(4),
			},
			Entries: []planner.SourceEntry{{Path: path, Size: 128, MD5: strings.Trim(path, "/")}},
		})
		if err != nil {
			t.Fatalf("Create(%s %s) error = %v", providerKey, path, err)
		}
		if _, ok, err := svc.Run(ctx, detail.Task.ID); err != nil || !ok {
			t.Fatalf("Run(%s %s) error=%v ok=%v", providerKey, path, err, ok)
		}
		blocked, ok, err := svc.Get(ctx, detail.Task.ID)
		if err != nil || !ok {
			t.Fatalf("Get(%s %s) error=%v ok=%v", providerKey, path, err, ok)
		}
		timestamp := time.Now().Add(time.Duration(offsetMinutes) * time.Minute).UTC().Format(time.RFC3339)
		blocked.Task.UpdatedAt = timestamp
		for idx := range blocked.Results {
			blocked.Results[idx].CreatedAt = timestamp
		}
		syncRuntimeRetryQueue(&blocked.Runtime, blocked.Plan.Metadata, blocked.Results)
		applyRetryQueueSummary(&blocked.Runtime, blocked.Plan.Metadata)
		if err := replaceTaskDetailAndResults(ctx, store, blocked); err != nil {
			t.Fatalf("replaceTaskDetailAndResults(%s %s) error = %v", providerKey, path, err)
		}
		return detail.Task.ID
	}

	profileA := createProfile("recover_protocol_group_budget_target_a")
	profileB := createProfile("recover_protocol_group_budget_target_b")
	profileC := createProfile("recover_protocol_group_budget_target_c")
	firstID := createBlockedTask("recover_protocol_group_budget_target_a", profileA, "/shared-a.bin", -180)
	secondID := createBlockedTask("recover_protocol_group_budget_target_b", profileB, "/shared-b.bin", -179)
	thirdID := createBlockedTask("recover_protocol_group_budget_target_c", profileC, "/other-c.bin", -178)

	result, err := svc.RecoverBlockedTasksWithOptions(ctx, RecoverOptions{
		Limit:                 5,
		LimitPerProtocolGroup: 1,
	})
	if err != nil {
		t.Fatalf("RecoverBlockedTasksWithOptions(protocolGroup budget) error = %v", err)
	}
	if result.MatchedCount != 3 || result.RecoveredCount != 2 || result.SkippedByProtocolGroupBudget != 1 || result.LimitPerProtocolGroup != 1 {
		t.Fatalf("unexpected protocolGroup budget result: %#v", result)
	}

	firstTask, ok, err := svc.Get(ctx, firstID)
	if err != nil || !ok {
		t.Fatalf("Get(first protocol group budget task) error=%v ok=%v", err, ok)
	}
	secondTask, ok, err := svc.Get(ctx, secondID)
	if err != nil || !ok {
		t.Fatalf("Get(second protocol group budget task) error=%v ok=%v", err, ok)
	}
	thirdTask, ok, err := svc.Get(ctx, thirdID)
	if err != nil || !ok {
		t.Fatalf("Get(third protocol group budget task) error=%v ok=%v", err, ok)
	}
	if firstTask.Task.State != StateCompleted {
		t.Fatalf("expected first shared-group task completed, got %s", firstTask.Task.State)
	}
	if secondTask.Task.State != StateBlocked {
		t.Fatalf("expected second shared-group task blocked, got %s", secondTask.Task.State)
	}
	if thirdTask.Task.State != StateCompleted {
		t.Fatalf("expected other-group task completed, got %s", thirdTask.Task.State)
	}
	if uploadCallsByPath["/shared-a.bin"] != 2 {
		t.Fatalf("expected /shared-a.bin upload calls 2, got %d", uploadCallsByPath["/shared-a.bin"])
	}
	if uploadCallsByPath["/shared-b.bin"] != 1 {
		t.Fatalf("expected /shared-b.bin upload calls 1, got %d", uploadCallsByPath["/shared-b.bin"])
	}
	if uploadCallsByPath["/other-c.bin"] != 2 {
		t.Fatalf("expected /other-c.bin upload calls 2, got %d", uploadCallsByPath["/other-c.bin"])
	}
}

func TestServiceRecoverBlockedTasksWithOptionsInterleavesProtocolGroupsBeforeProviderBudget(t *testing.T) {
	ctx := context.Background()
	store, err := sqlitestore.New(ctx, filepath.Join(t.TempDir(), "auto-recover-protocol-group-rotation.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer func() { _ = store.Close() }()

	uploadCallsByPath := map[string]int{}
	newAdapter := func(key, protocolGroup string) *scriptedAdapter {
		return &scriptedAdapter{
			meta: provider.Provider{
				Key:              key,
				DisplayName:      key,
				ProtocolGroup:    protocolGroup,
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
				uploadCallsByPath[req.Path]++
				if uploadCallsByPath[req.Path] == 1 {
					return provider.UploadResult{
						OperationResult: provider.OperationResult{
							Status:  "rate_limited",
							Message: "rate limited",
							Mode:    "fake_rate_limit",
						},
					}
				}
				return provider.UploadResult{
					OperationResult: provider.OperationResult{
						OK:      true,
						Status:  "ok",
						Message: "protocol group interleaved recovered",
						Mode:    "fake_protocol_group_interleaved_ok",
					},
				}
			},
		}
	}

	adapterA1 := newAdapter("recover_protocol_rotate_target_a1", "rotate_group_a")
	adapterA2 := newAdapter("recover_protocol_rotate_target_a2", "rotate_group_a")
	adapterB1 := newAdapter("recover_protocol_rotate_target_b1", "rotate_group_b")
	registry := provider.NewRegistry(adapterA1, adapterA2, adapterB1)
	authSvc := auth.NewService(store, registry)
	svc := NewService(store, registry, authSvc)

	createProfile := func(providerKey string) string {
		profile, err := authSvc.CreateProfile(ctx, auth.CreateProfileInput{
			ProviderKey: providerKey,
			AuthMode:    "manual_token",
			DisplayName: providerKey,
			Token:       "token-" + providerKey,
		})
		if err != nil {
			t.Fatalf("CreateProfile(%s) error = %v", providerKey, err)
		}
		return profile.ID
	}

	createRetryableTask := func(providerKey, profileID, path string, offsetMinutes int) string {
		detail, err := svc.Create(ctx, CreateRequest{
			SourceProvider:  "guangya",
			TargetProvider:  providerKey,
			TargetProfileID: profileID,
			ThresholdMB:     1,
			RiskOverride: &planner.RiskProfileOverride{
				CooldownSeconds: intPtrTask(0),
				MaxConcurrent:   intPtrTask(4),
			},
			Entries: []planner.SourceEntry{{Path: path, Size: 128, MD5: strings.Trim(path, "/")}},
		})
		if err != nil {
			t.Fatalf("Create(%s %s) error = %v", providerKey, path, err)
		}
		if _, ok, err := svc.Run(ctx, detail.Task.ID); err != nil || !ok {
			t.Fatalf("Run(%s %s) error=%v ok=%v", providerKey, path, err, ok)
		}
		blocked, ok, err := svc.Get(ctx, detail.Task.ID)
		if err != nil || !ok {
			t.Fatalf("Get(%s %s) error=%v ok=%v", providerKey, path, err, ok)
		}
		timestamp := time.Now().Add(time.Duration(offsetMinutes) * time.Minute).UTC().Format(time.RFC3339)
		blocked.Task.UpdatedAt = timestamp
		for idx := range blocked.Results {
			blocked.Results[idx].CreatedAt = timestamp
		}
		syncRuntimeRetryQueue(&blocked.Runtime, blocked.Plan.Metadata, blocked.Results)
		applyRetryQueueSummary(&blocked.Runtime, blocked.Plan.Metadata)
		if err := replaceTaskDetailAndResults(ctx, store, blocked); err != nil {
			t.Fatalf("replaceTaskDetailAndResults(%s %s) error = %v", providerKey, path, err)
		}
		return detail.Task.ID
	}

	profileA1 := createProfile("recover_protocol_rotate_target_a1")
	profileA2 := createProfile("recover_protocol_rotate_target_a2")
	profileB1 := createProfile("recover_protocol_rotate_target_b1")
	firstID := createRetryableTask("recover_protocol_rotate_target_a1", profileA1, "/rotate-a-one.bin", -300)
	secondID := createRetryableTask("recover_protocol_rotate_target_a2", profileA2, "/rotate-a-two.bin", -299)
	thirdID := createRetryableTask("recover_protocol_rotate_target_b1", profileB1, "/rotate-b-one.bin", -298)

	result, err := svc.RecoverBlockedTasksWithOptions(ctx, RecoverOptions{
		Limit: 2,
	})
	if err != nil {
		t.Fatalf("RecoverBlockedTasksWithOptions(protocol rotation) error = %v", err)
	}
	if result.MatchedCount != 3 || result.RecoveredCount != 2 || result.SkippedByLimit != 1 {
		t.Fatalf("unexpected protocol rotation result: %#v", result)
	}

	firstTask, ok, err := svc.Get(ctx, firstID)
	if err != nil || !ok {
		t.Fatalf("Get(first protocol rotation task) error=%v ok=%v", err, ok)
	}
	secondTask, ok, err := svc.Get(ctx, secondID)
	if err != nil || !ok {
		t.Fatalf("Get(second protocol rotation task) error=%v ok=%v", err, ok)
	}
	thirdTask, ok, err := svc.Get(ctx, thirdID)
	if err != nil || !ok {
		t.Fatalf("Get(third protocol rotation task) error=%v ok=%v", err, ok)
	}
	if firstTask.Task.State != StateCompleted {
		t.Fatalf("expected first rotate_group_a task completed, got %s", firstTask.Task.State)
	}
	if secondTask.Task.State != StateBlocked {
		t.Fatalf("expected second rotate_group_a task to remain blocked, got %s", secondTask.Task.State)
	}
	if thirdTask.Task.State != StateCompleted {
		t.Fatalf("expected rotate_group_b task completed, got %s", thirdTask.Task.State)
	}
	if uploadCallsByPath["/rotate-a-one.bin"] != 2 {
		t.Fatalf("expected /rotate-a-one.bin upload calls 2, got %d", uploadCallsByPath["/rotate-a-one.bin"])
	}
	if uploadCallsByPath["/rotate-a-two.bin"] != 1 {
		t.Fatalf("expected /rotate-a-two.bin upload calls 1, got %d", uploadCallsByPath["/rotate-a-two.bin"])
	}
	if uploadCallsByPath["/rotate-b-one.bin"] != 2 {
		t.Fatalf("expected /rotate-b-one.bin upload calls 2, got %d", uploadCallsByPath["/rotate-b-one.bin"])
	}
}

func TestServiceAutoRecoverPoolSummaryIncludesProtocolGroups(t *testing.T) {
	ctx := context.Background()
	store, err := sqlitestore.New(ctx, filepath.Join(t.TempDir(), "auto-recover-protocol-groups-summary.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer func() { _ = store.Close() }()

	newAdapter := func(key, protocolGroup string) *scriptedAdapter {
		return &scriptedAdapter{
			meta: provider.Provider{
				Key:              key,
				DisplayName:      key,
				ProtocolGroup:    protocolGroup,
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
				return provider.UploadResult{
					OperationResult: provider.OperationResult{
						Status:  "rate_limited",
						Message: "rate limited",
						Mode:    "fake_rate_limit",
					},
				}
			},
		}
	}

	adapterA := newAdapter("auto_recover_protocol_summary_target_a", "summary_protocol_group_a")
	adapterB := newAdapter("auto_recover_protocol_summary_target_b", "summary_protocol_group_b")
	registry := provider.NewRegistry(adapterA, adapterB)
	authSvc := auth.NewService(store, registry)
	svc := NewService(store, registry, authSvc)

	createProfile := func(providerKey string) string {
		profile, err := authSvc.CreateProfile(ctx, auth.CreateProfileInput{
			ProviderKey: providerKey,
			AuthMode:    "manual_token",
			DisplayName: providerKey,
			Token:       "token-" + providerKey,
		})
		if err != nil {
			t.Fatalf("CreateProfile(%s) error = %v", providerKey, err)
		}
		return profile.ID
	}

	createBlockedTask := func(providerKey, profileID, path string, offsetMinutes int) {
		detail, err := svc.Create(ctx, CreateRequest{
			SourceProvider:  "guangya",
			TargetProvider:  providerKey,
			TargetProfileID: profileID,
			ThresholdMB:     1,
			RiskOverride: &planner.RiskProfileOverride{
				CooldownSeconds: intPtrTask(3600),
				MaxConcurrent:   intPtrTask(3),
			},
			Entries: []planner.SourceEntry{{Path: path, Size: 128, MD5: strings.Trim(path, "/")}},
		})
		if err != nil {
			t.Fatalf("Create(%s %s) error = %v", providerKey, path, err)
		}
		if _, ok, err := svc.Run(ctx, detail.Task.ID); err != nil || !ok {
			t.Fatalf("Run(%s %s) error=%v ok=%v", providerKey, path, err, ok)
		}
		blocked, ok, err := svc.Get(ctx, detail.Task.ID)
		if err != nil || !ok {
			t.Fatalf("Get(%s %s) error=%v ok=%v", providerKey, path, err, ok)
		}
		timestamp := time.Now().Add(time.Duration(offsetMinutes) * time.Minute).UTC().Format(time.RFC3339)
		blocked.Task.UpdatedAt = timestamp
		for idx := range blocked.Results {
			blocked.Results[idx].CreatedAt = timestamp
		}
		syncRuntimeRetryQueue(&blocked.Runtime, blocked.Plan.Metadata, blocked.Results)
		applyRetryQueueSummary(&blocked.Runtime, blocked.Plan.Metadata)
		if err := replaceTaskDetailAndResults(ctx, store, blocked); err != nil {
			t.Fatalf("replaceTaskDetailAndResults(%s %s) error = %v", providerKey, path, err)
		}
	}

	profileA := createProfile("auto_recover_protocol_summary_target_a")
	profileB := createProfile("auto_recover_protocol_summary_target_b")
	createBlockedTask("auto_recover_protocol_summary_target_a", profileA, "/summary-a.bin", -200)
	createBlockedTask("auto_recover_protocol_summary_target_b", profileB, "/summary-b.bin", -199)

	evidence, err := svc.RuntimeEvidence(ctx)
	if err != nil {
		t.Fatalf("RuntimeEvidence() error = %v", err)
	}
	lane := autoRecoverLaneByMode(evidence.AutoRecoverPool, "retry_queue_auto_retry")
	if lane.Mode == "" {
		t.Fatalf("expected retry_queue_auto_retry lane in %#v", evidence.AutoRecoverPool)
	}
	if lane.SampleProtocolGroup != "summary_protocol_group_a" {
		t.Fatalf("expected sample protocol group summary_protocol_group_a, got %#v", lane.SampleProtocolGroup)
	}
	if len(lane.ProtocolGroups) != 2 || lane.ProtocolGroups[0] != "summary_protocol_group_a" || lane.ProtocolGroups[1] != "summary_protocol_group_b" {
		t.Fatalf("unexpected protocol groups: %#v", lane.ProtocolGroups)
	}
}
func TestServiceRecoverBlockedTasksWithOptionsRoundRobinsProvidersWithinSameBand(t *testing.T) {
	ctx := context.Background()
	store, err := sqlitestore.New(ctx, filepath.Join(t.TempDir(), "auto-recover-provider-round-robin.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer func() { _ = store.Close() }()

	recoveryOrder := make([]string, 0, 3)
	newAdapter := func(key string) *scriptedAdapter {
		return &scriptedAdapter{
			meta: provider.Provider{
				Key:              key,
				DisplayName:      key,
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
				recoveryOrder = append(recoveryOrder, key+":"+req.Path)
				return provider.UploadResult{
					OperationResult: provider.OperationResult{
						OK:      true,
						Status:  "ok",
						Message: "round robin recovered",
						Mode:    "fake_round_robin_ok",
					},
				}
			},
		}
	}

	providerA := newAdapter("recover_rr_target_a")
	providerB := newAdapter("recover_rr_target_b")
	registry := provider.NewRegistry(providerA, providerB)
	authSvc := auth.NewService(store, registry)
	svc := NewService(store, registry, authSvc)

	createProfile := func(providerKey string) string {
		profile, err := authSvc.CreateProfile(ctx, auth.CreateProfileInput{
			ProviderKey: providerKey,
			AuthMode:    "manual_token",
			DisplayName: providerKey,
			Token:       "token-" + providerKey,
		})
		if err != nil {
			t.Fatalf("CreateProfile(%s) error = %v", providerKey, err)
		}
		return profile.ID
	}
	profileA := createProfile("recover_rr_target_a")
	profileB := createProfile("recover_rr_target_b")

	createBlockedTask := func(providerKey, profileID, path string, offsetMinutes int) {
		now := time.Now().Add(time.Duration(offsetMinutes) * time.Minute).UTC().Format(time.RFC3339)
		detail, err := svc.Create(ctx, CreateRequest{
			SourceProvider:  "guangya",
			TargetProvider:  providerKey,
			TargetProfileID: profileID,
			ThresholdMB:     1,
			RiskOverride: &planner.RiskProfileOverride{
				MaxConcurrent: intPtrTask(3),
			},
			Entries: []planner.SourceEntry{
				{Path: path, Size: 256, MD5: strings.Trim(path, "/")},
			},
		})
		if err != nil {
			t.Fatalf("Create(%s %s) error = %v", providerKey, path, err)
		}
		blocked, ok, err := svc.Get(ctx, detail.Task.ID)
		if err != nil || !ok {
			t.Fatalf("Get(%s %s) error=%v ok=%v", providerKey, path, err, ok)
		}
		blocked.Task.State = StateBlocked
		blocked.Task.UpdatedAt = now
		blocked.Runtime = initializeRuntimeState(blocked.Plan)
		blocked.Runtime.ExecutionState = "blocked"
		blocked.Runtime.RetryQueue = []RetryQueueItem{
			{
				Path:           path,
				Retryable:      true,
				RetryClass:     "retry_failed",
				ProviderStatus: "provider_request_failed",
			},
		}
		blocked.Results = []Result{
			{
				ID:        uuid.NewString(),
				TaskID:    blocked.Task.ID,
				ItemID:    blocked.Items[0].ID,
				Status:    "failed",
				Mode:      "fake_failed",
				Message:   "failed",
				CreatedAt: now,
				Payload: map[string]interface{}{
					"path":           path,
					"providerStatus": "provider_request_failed",
				},
			},
		}
		applyRetryQueueSummary(&blocked.Runtime, blocked.Plan.Metadata)
		if err := replaceTaskDetailAndResults(ctx, store, blocked); err != nil {
			t.Fatalf("replaceTaskDetailAndResults(%s %s) error = %v", providerKey, path, err)
		}
	}

	createBlockedTask("recover_rr_target_a", profileA, "/a-1.bin", -182)
	createBlockedTask("recover_rr_target_a", profileA, "/a-2.bin", -181)
	createBlockedTask("recover_rr_target_b", profileB, "/b-1.bin", -180)

	result, err := svc.RecoverBlockedTasksWithOptions(ctx, RecoverOptions{Limit: 3})
	if err != nil {
		t.Fatalf("RecoverBlockedTasksWithOptions(round robin) error = %v", err)
	}
	if result.MatchedCount != 3 || result.RecoveredCount != 3 {
		t.Fatalf("unexpected round robin result: %#v", result)
	}
	if len(recoveryOrder) != 3 {
		t.Fatalf("expected 3 recovery operations, got %#v", recoveryOrder)
	}
	if recoveryOrder[0] != "recover_rr_target_a:/a-1.bin" ||
		recoveryOrder[1] != "recover_rr_target_b:/b-1.bin" ||
		recoveryOrder[2] != "recover_rr_target_a:/a-2.bin" {
		t.Fatalf("expected provider round robin order [a1 b1 a2], got %#v", recoveryOrder)
	}
}

func TestServiceRecoverBlockedTasksWithOptionsRoundRobinsProfilesWithinSameProviderBand(t *testing.T) {
	ctx := context.Background()
	store, err := sqlitestore.New(ctx, filepath.Join(t.TempDir(), "auto-recover-profile-round-robin.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer func() { _ = store.Close() }()

	recoveryOrder := make([]string, 0, 3)
	adapter := &scriptedAdapter{
		meta: provider.Provider{
			Key:              "recover_rr_profile_target",
			DisplayName:      "recover_rr_profile_target",
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
			recoveryOrder = append(recoveryOrder, req.Profile.ID+":"+req.Path)
			return provider.UploadResult{
				OperationResult: provider.OperationResult{
					OK:      true,
					Status:  "ok",
					Message: "profile round robin recovered",
					Mode:    "fake_profile_round_robin_ok",
				},
			}
		},
	}

	registry := provider.NewRegistry(adapter)
	authSvc := auth.NewService(store, registry)
	svc := NewService(store, registry, authSvc)

	createProfile := func(name string) string {
		profile, err := authSvc.CreateProfile(ctx, auth.CreateProfileInput{
			ProviderKey: "recover_rr_profile_target",
			AuthMode:    "manual_token",
			DisplayName: name,
			Token:       "token-" + name,
		})
		if err != nil {
			t.Fatalf("CreateProfile(%s) error = %v", name, err)
		}
		return profile.ID
	}
	profileA := createProfile("profile-a")
	profileB := createProfile("profile-b")

	createBlockedTask := func(profileID, path string, offsetMinutes int) {
		now := time.Now().Add(time.Duration(offsetMinutes) * time.Minute).UTC().Format(time.RFC3339)
		detail, err := svc.Create(ctx, CreateRequest{
			SourceProvider:  "guangya",
			TargetProvider:  "recover_rr_profile_target",
			TargetProfileID: profileID,
			ThresholdMB:     1,
			RiskOverride: &planner.RiskProfileOverride{
				MaxConcurrent: intPtrTask(3),
			},
			Entries: []planner.SourceEntry{
				{Path: path, Size: 256, MD5: strings.Trim(path, "/")},
			},
		})
		if err != nil {
			t.Fatalf("Create(%s %s) error = %v", profileID, path, err)
		}
		blocked, ok, err := svc.Get(ctx, detail.Task.ID)
		if err != nil || !ok {
			t.Fatalf("Get(%s %s) error=%v ok=%v", profileID, path, err, ok)
		}
		blocked.Task.State = StateBlocked
		blocked.Task.UpdatedAt = now
		blocked.Runtime = initializeRuntimeState(blocked.Plan)
		blocked.Runtime.ExecutionState = "blocked"
		blocked.Runtime.RetryQueue = []RetryQueueItem{
			{
				Path:           path,
				Retryable:      true,
				RetryClass:     "retry_failed",
				ProviderStatus: "provider_request_failed",
			},
		}
		blocked.Results = []Result{
			{
				ID:        uuid.NewString(),
				TaskID:    blocked.Task.ID,
				ItemID:    blocked.Items[0].ID,
				Status:    "failed",
				Mode:      "fake_failed",
				Message:   "failed",
				CreatedAt: now,
				Payload: map[string]interface{}{
					"path":           path,
					"providerStatus": "provider_request_failed",
				},
			},
		}
		applyRetryQueueSummary(&blocked.Runtime, blocked.Plan.Metadata)
		if err := replaceTaskDetailAndResults(ctx, store, blocked); err != nil {
			t.Fatalf("replaceTaskDetailAndResults(%s %s) error = %v", profileID, path, err)
		}
	}

	createBlockedTask(profileA, "/a-1.bin", -182)
	createBlockedTask(profileA, "/a-2.bin", -181)
	createBlockedTask(profileB, "/b-1.bin", -180)

	result, err := svc.RecoverBlockedTasksWithOptions(ctx, RecoverOptions{Limit: 3})
	if err != nil {
		t.Fatalf("RecoverBlockedTasksWithOptions(profile round robin) error = %v", err)
	}
	if result.MatchedCount != 3 || result.RecoveredCount != 3 {
		t.Fatalf("unexpected profile round robin result: %#v", result)
	}
	if len(recoveryOrder) != 3 {
		t.Fatalf("expected 3 recovery operations, got %#v", recoveryOrder)
	}
	if recoveryOrder[0] != profileA+":/a-1.bin" ||
		recoveryOrder[1] != profileB+":/b-1.bin" ||
		recoveryOrder[2] != profileA+":/a-2.bin" {
		t.Fatalf("expected profile round robin order [a1 b1 a2], got %#v", recoveryOrder)
	}
}

func TestServiceRecoverBlockedTasksWithOptionsFiltersProfileID(t *testing.T) {
	ctx := context.Background()
	store, err := sqlitestore.New(ctx, filepath.Join(t.TempDir(), "auto-recover-profile-filter.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer func() { _ = store.Close() }()

	uploadCallsByPath := make(map[string]int)
	adapter := &scriptedAdapter{
		meta: provider.Provider{
			Key:              "recover_profile_filter_target",
			DisplayName:      "Recover Profile Filter Target",
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
			uploadCallsByPath[req.Path]++
			if uploadCallsByPath[req.Path] == 1 {
				return provider.UploadResult{
					OperationResult: provider.OperationResult{
						Status:  "rate_limited",
						Message: "rate limited",
						Mode:    "scripted_rate_limit",
					},
				}
			}
			return provider.UploadResult{
				OperationResult: provider.OperationResult{
					OK:      true,
					Status:  "ok",
					Message: "profile filtered recovered",
					Mode:    "scripted_profile_filter_ok",
				},
			}
		},
	}

	registry := provider.NewRegistry(adapter)
	authSvc := auth.NewService(store, registry)
	svc := NewService(store, registry, authSvc)

	createProfile := func(name string) string {
		profile, err := authSvc.CreateProfile(ctx, auth.CreateProfileInput{
			ProviderKey: "recover_profile_filter_target",
			AuthMode:    "manual_token",
			DisplayName: name,
			Token:       "token-" + name,
		})
		if err != nil {
			t.Fatalf("CreateProfile(%s) error = %v", name, err)
		}
		return profile.ID
	}
	profileA := createProfile("profile-a")
	profileB := createProfile("profile-b")

	createBlockedTask := func(profileID, path string) string {
		detail, err := svc.Create(ctx, CreateRequest{
			SourceProvider:  "guangya",
			TargetProvider:  "recover_profile_filter_target",
			TargetProfileID: profileID,
			ThresholdMB:     1,
			Entries: []planner.SourceEntry{
				{Path: path, Size: 101, MD5: path},
			},
		})
		if err != nil {
			t.Fatalf("Create(%s %s) error = %v", profileID, path, err)
		}
		if _, ok, err := svc.Run(ctx, detail.Task.ID); err != nil || !ok {
			t.Fatalf("Run(%s %s) error=%v ok=%v", profileID, path, err, ok)
		}
		if _, err := store.DB().ExecContext(ctx, `UPDATE task_results SET created_at = ? WHERE task_id = ?`,
			time.Now().Add(-2*time.Hour).UTC().Format(time.RFC3339), detail.Task.ID,
		); err != nil {
			t.Fatalf("update task_results created_at for %s error = %v", path, err)
		}
		return detail.Task.ID
	}

	firstTaskID := createBlockedTask(profileA, "/group-a/one.bin")
	secondTaskID := createBlockedTask(profileB, "/group-b/two.bin")

	result, err := svc.RecoverBlockedTasksWithOptions(ctx, RecoverOptions{
		ProfileID: profileB,
		Limit:     2,
	})
	if err != nil {
		t.Fatalf("RecoverBlockedTasksWithOptions(profile filter) error = %v", err)
	}
	if result.MatchedCount != 1 || result.RecoveredCount != 1 || result.ProfileID != profileB {
		t.Fatalf("unexpected profile filter result: %#v", result)
	}

	firstDetail, ok, err := svc.Get(ctx, firstTaskID)
	if err != nil || !ok {
		t.Fatalf("Get(first task) error=%v ok=%v", err, ok)
	}
	if len(firstDetail.Results) != 1 || stringValue(firstDetail.Results[0].Payload["path"]) != "/group-a/one.bin" {
		t.Fatalf("expected first task unchanged, got %#v", firstDetail.Results)
	}

	secondDetail, ok, err := svc.Get(ctx, secondTaskID)
	if err != nil || !ok {
		t.Fatalf("Get(second task) error=%v ok=%v", err, ok)
	}
	if len(secondDetail.Results) != 1 || stringValue(secondDetail.Results[0].Payload["path"]) != "/group-b/two.bin" {
		t.Fatalf("expected second task recovered path /group-b/two.bin, got %#v", secondDetail.Results)
	}
	if got := uploadCallsByPath["/group-a/one.bin"]; got != 1 {
		t.Fatalf("expected first task upload calls to remain 1, got %d", got)
	}
	if got := uploadCallsByPath["/group-b/two.bin"]; got != 2 {
		t.Fatalf("expected second task upload calls to become 2, got %d", got)
	}
}

func TestServiceRecoverBlockedTasksRespectsAutoRetryWindow(t *testing.T) {
	ctx := context.Background()
	store, err := sqlitestore.New(ctx, filepath.Join(t.TempDir(), "auto-retry-window.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer func() { _ = store.Close() }()

	uploadCalls := 0
	adapter := &scriptedAdapter{
		meta: provider.Provider{
			Key:              "auto_retry_window_target",
			DisplayName:      "Auto Retry Window Target",
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
			uploadCalls++
			if uploadCalls == 1 {
				return provider.UploadResult{
					OperationResult: provider.OperationResult{
						Status:  "rate_limited",
						Message: "rate limited",
						Mode:    "fake_rate_limit",
					},
				}
			}
			return provider.UploadResult{
				OperationResult: provider.OperationResult{
					OK:      true,
					Status:  "ok",
					Message: "recovered inside window",
					Mode:    "fake_window_ok",
				},
			}
		},
	}

	registry := provider.NewRegistry(adapter)
	authSvc := auth.NewService(store, registry)
	svc := NewService(store, registry, authSvc)
	profile, err := authSvc.CreateProfile(ctx, auth.CreateProfileInput{
		ProviderKey: "auto_retry_window_target",
		AuthMode:    "manual_token",
		DisplayName: "auto retry window target",
		Token:       "token-window",
	})
	if err != nil {
		t.Fatalf("CreateProfile() error = %v", err)
	}

	nowHour := time.Now().UTC().Hour()
	blockedStart := (nowHour + 2) % 24
	blockedEnd := (nowHour + 3) % 24
	created, err := svc.Create(ctx, CreateRequest{
		SourceProvider:  "guangya",
		TargetProvider:  "auto_retry_window_target",
		TargetProfileID: profile.ID,
		ThresholdMB:     1,
		RiskOverride: &planner.RiskProfileOverride{
			CooldownSeconds:    intPtrTask(3600),
			AutoRetryStartHour: intPtrTask(blockedStart),
			AutoRetryEndHour:   intPtrTask(blockedEnd),
		},
		Entries: []planner.SourceEntry{
			{Path: "/window.bin", Size: 1024, MD5: "window-md5"},
		},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	firstRun, ok, err := svc.Run(ctx, created.Task.ID)
	if err != nil || !ok {
		t.Fatalf("Run() error=%v ok=%v", err, ok)
	}
	if firstRun.Task.State != StateBlocked {
		t.Fatalf("expected blocked after first run, got %s", firstRun.Task.State)
	}

	detail, ok, err := svc.Get(ctx, created.Task.ID)
	if err != nil || !ok {
		t.Fatalf("Get() error=%v ok=%v", err, ok)
	}
	detail.Results[0].CreatedAt = time.Now().Add(-2 * time.Hour).UTC().Format(time.RFC3339)
	syncRuntimeRetryQueue(&detail.Runtime, detail.Plan.Metadata, detail.Results)
	applyRetryQueueSummary(&detail.Runtime, detail.Plan.Metadata)
	if err := replaceTaskDetailAndResults(ctx, store, detail); err != nil {
		t.Fatalf("replaceTaskDetailAndResults() error = %v", err)
	}

	recovered, err := svc.RecoverBlockedTasks(ctx)
	if err != nil {
		t.Fatalf("RecoverBlockedTasks() error = %v", err)
	}
	if recovered != 0 {
		t.Fatalf("expected recovered count 0 outside auto retry window, got %d", recovered)
	}
	afterBlocked, ok, err := svc.Get(ctx, created.Task.ID)
	if err != nil || !ok {
		t.Fatalf("Get(after blocked) error=%v ok=%v", err, ok)
	}
	if afterBlocked.Task.State != StateBlocked {
		t.Fatalf("expected task to stay blocked outside auto retry window, got %s", afterBlocked.Task.State)
	}

	allowedStart := nowHour
	allowedEnd := (nowHour + 2) % 24
	if allowedEnd == allowedStart {
		allowedEnd = (allowedStart + 1) % 24
	}
	detail.Plan.Metadata["riskProfile"] = planner.RiskProfile{
		Mode:                planner.RiskModeBalanced,
		RequestIntervalMS:   800,
		PageSize:            300,
		DirectoryIntervalMS: 1000,
		CooldownSeconds:     0,
		RetryLimit:          3,
		MaxConcurrent:       1,
		AutoRetryStartHour:  allowedStart,
		AutoRetryEndHour:    allowedEnd,
		RiskKeywords:        []string{"rate_limit"},
	}
	if err := replaceTaskDetailAndResults(ctx, store, detail); err != nil {
		t.Fatalf("replaceTaskDetailAndResults(allowed) error = %v", err)
	}

	recovered, err = svc.RecoverBlockedTasks(ctx)
	if err != nil {
		t.Fatalf("RecoverBlockedTasks(allowed) error = %v", err)
	}
	if recovered != 1 {
		t.Fatalf("expected recovered count 1 inside auto retry window, got %d", recovered)
	}
	finalDetail, ok, err := svc.Get(ctx, created.Task.ID)
	if err != nil || !ok {
		t.Fatalf("Get(final) error=%v ok=%v", err, ok)
	}
	if finalDetail.Task.State != StateCompleted {
		t.Fatalf("expected completed after allowed window recovery, got %s", finalDetail.Task.State)
	}
}

func TestServiceRetryBlockedForLocalFileMissingDoesNotResetTask(t *testing.T) {
	ctx := context.Background()
	store, err := sqlitestore.New(ctx, filepath.Join(t.TempDir(), "retry-blocked-local.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer func() { _ = store.Close() }()

	adapter := &scriptedAdapter{
		meta: provider.Provider{
			Key:              "retry_blocked_local_target",
			DisplayName:      "Retry Blocked Local Target",
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
			return provider.UploadResult{
				OperationResult: provider.OperationResult{
					Status:  "local_file_missing",
					Message: "local file missing",
					Mode:    "fake_missing",
				},
			}
		},
	}

	registry := provider.NewRegistry(adapter)
	authSvc := auth.NewService(store, registry)
	svc := NewService(store, registry, authSvc)
	profile, err := authSvc.CreateProfile(ctx, auth.CreateProfileInput{
		ProviderKey: "retry_blocked_local_target",
		AuthMode:    "manual_token",
		DisplayName: "retry blocked local target",
		Token:       "token-1",
	})
	if err != nil {
		t.Fatalf("CreateProfile() error = %v", err)
	}

	created, err := svc.Create(ctx, CreateRequest{
		SourceProvider:  "guangya",
		TargetProvider:  "retry_blocked_local_target",
		TargetProfileID: profile.ID,
		Entries: []planner.SourceEntry{
			{Path: "/missing.bin", Size: 1024, MD5: "abc", LocalPath: filepath.Join(t.TempDir(), "missing.bin")},
		},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	running, ok, err := svc.Run(ctx, created.Task.ID)
	if err != nil || !ok {
		t.Fatalf("Run() error=%v ok=%v", err, ok)
	}
	if running.Task.State != StateBlocked {
		t.Fatalf("expected blocked, got %s", running.Task.State)
	}
	if _, ok, err := svc.Retry(ctx, created.Task.ID); err == nil || !ok {
		t.Fatalf("expected retry_blocked error with task present, ok=%v err=%v", ok, err)
	} else if !strings.HasPrefix(err.Error(), "retry_blocked:retry_queue_requires_local_file_restore") {
		t.Fatalf("expected retry_queue_requires_local_file_restore, got %v", err)
	}
	after, ok, err := svc.Get(ctx, created.Task.ID)
	if err != nil || !ok {
		t.Fatalf("Get() after retry error=%v ok=%v", err, ok)
	}
	if after.Task.State != StateBlocked {
		t.Fatalf("expected task to remain blocked, got %s", after.Task.State)
	}
	if len(after.Results) != 1 || after.Results[0].Status != "failed" {
		t.Fatalf("expected failed result to remain, got %#v", after.Results)
	}
}

func TestServiceRetryQueueMarksExhaustedAfterRetryLimit(t *testing.T) {
	ctx := context.Background()
	store, err := sqlitestore.New(ctx, filepath.Join(t.TempDir(), "retry-limit.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer func() { _ = store.Close() }()

	uploadCalls := 0
	adapter := &scriptedAdapter{
		meta: provider.Provider{
			Key:              "retry_limit_target",
			DisplayName:      "Retry Limit Target",
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
			uploadCalls++
			return provider.UploadResult{
				OperationResult: provider.OperationResult{
					Status:  "remote_error",
					Message: "remote error",
					Mode:    "fake_remote_error",
				},
			}
		},
	}

	registry := provider.NewRegistry(adapter)
	authSvc := auth.NewService(store, registry)
	svc := NewService(store, registry, authSvc)
	profile, err := authSvc.CreateProfile(ctx, auth.CreateProfileInput{
		ProviderKey: "retry_limit_target",
		AuthMode:    "manual_token",
		DisplayName: "retry limit target",
		Token:       "token-1",
	})
	if err != nil {
		t.Fatalf("CreateProfile() error = %v", err)
	}

	created, err := svc.Create(ctx, CreateRequest{
		SourceProvider:  "guangya",
		TargetProvider:  "retry_limit_target",
		TargetProfileID: profile.ID,
		RiskOverride: &planner.RiskProfileOverride{
			RetryLimit: intPtrTask(1),
		},
		Entries: []planner.SourceEntry{
			{Path: "/fail.bin", Size: 1024, MD5: "abc"},
		},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	firstRun, ok, err := svc.Run(ctx, created.Task.ID)
	if err != nil || !ok {
		t.Fatalf("Run() first error=%v ok=%v", err, ok)
	}
	if firstRun.Task.State != StateCompletedWithErrors {
		t.Fatalf("expected completed_with_errors on first run, got %s", firstRun.Task.State)
	}
	if len(firstRun.Runtime.RetryQueue) != 1 {
		t.Fatalf("expected retry queue len 1, got %d", len(firstRun.Runtime.RetryQueue))
	}
	if firstRun.Runtime.RetryQueue[0].AttemptCount != 0 {
		t.Fatalf("expected attempt count 0 before retry, got %d", firstRun.Runtime.RetryQueue[0].AttemptCount)
	}
	if firstRun.Runtime.RetryQueue[0].RemainingCount != 1 {
		t.Fatalf("expected remaining count 1 before retry, got %d", firstRun.Runtime.RetryQueue[0].RemainingCount)
	}

	retried, ok, err := svc.Retry(ctx, created.Task.ID)
	if err != nil || !ok {
		t.Fatalf("Retry() error=%v ok=%v", err, ok)
	}
	attempts := retryAttemptsFromMetadata(retried.Plan.Metadata)
	if attempts["/fail.bin"] != 1 {
		t.Fatalf("expected retryAttempts for /fail.bin to be 1, got %#v", attempts)
	}

	secondRun, ok, err := svc.Run(ctx, created.Task.ID)
	if err != nil || !ok {
		t.Fatalf("Run() second error=%v ok=%v", err, ok)
	}
	if secondRun.Task.State != StateBlocked {
		t.Fatalf("expected blocked after retry limit exhausted, got %s", secondRun.Task.State)
	}
	if secondRun.Runtime.BlockedReason != "retry_queue_retry_limit_exhausted" {
		t.Fatalf("expected retry limit blocked reason, got %s", secondRun.Runtime.BlockedReason)
	}
	if secondRun.Runtime.BlockedAction != "review_and_reset_retry_strategy" {
		t.Fatalf("expected retry limit blocked action, got %s", secondRun.Runtime.BlockedAction)
	}
	if secondRun.Runtime.BlockedAdvice == "" {
		t.Fatal("expected retry limit blocked advice")
	}
	if len(secondRun.Runtime.RetryQueue) != 1 {
		t.Fatalf("expected retry queue len 1, got %d", len(secondRun.Runtime.RetryQueue))
	}
	item := secondRun.Runtime.RetryQueue[0]
	if !item.Exhausted {
		t.Fatal("expected retry queue item exhausted")
	}
	if item.AttemptCount != 1 {
		t.Fatalf("expected attempt count 1 after retry, got %d", item.AttemptCount)
	}
	if item.RemainingCount != 0 {
		t.Fatalf("expected remaining count 0 after retry, got %d", item.RemainingCount)
	}
	if item.Retryable {
		t.Fatal("expected exhausted item not retryable")
	}
	if _, ok, err := svc.Retry(ctx, created.Task.ID); err == nil || !ok {
		t.Fatalf("expected retry_blocked after retry limit exhausted, ok=%v err=%v", ok, err)
	} else if !strings.HasPrefix(err.Error(), "retry_blocked:retry_queue_retry_limit_exhausted") {
		t.Fatalf("expected retry limit exhausted error, got %v", err)
	}
	if uploadCalls != 2 {
		t.Fatalf("expected exactly 2 upload attempts, got %d", uploadCalls)
	}
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
	if got, _ := running.Results[0].Payload["executionMode"].(string); got != string(planner.ExecutionModeLeafFirstLazy) {
		t.Fatalf("expected skip result execution mode leaf_first_lazy, got %v", running.Results[0].Payload["executionMode"])
	}
	if got, _ := running.Results[0].Payload["scanMode"].(string); got != "lazy_leaf_first" {
		t.Fatalf("expected skip result scan mode lazy_leaf_first, got %v", running.Results[0].Payload["scanMode"])
	}
	if got, _ := running.Results[0].Payload["sourceDeletePolicy"].(string); got != string(planner.SourceDeletePolicyRecordOnly) {
		t.Fatalf("expected skip result source delete policy record_only, got %v", running.Results[0].Payload["sourceDeletePolicy"])
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

func TestServiceRunTreatsMetadataStatusExistsAsExplicitTargetExistence(t *testing.T) {
	ctx := context.Background()
	store, err := sqlitestore.New(ctx, filepath.Join(t.TempDir(), "runtime-status-exists.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer func() { _ = store.Close() }()

	uploadCalls := 0
	adapter := &scriptedAdapter{
		meta: provider.Provider{
			Key:              "runtime_status_exists",
			DisplayName:      "Runtime Status Exists",
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
					Message: "exists by status",
					Mode:    "scripted_metadata",
				},
				Entry: map[string]interface{}{
					"path": req.Path,
					"size": int64(1024),
					"md5":  "abc",
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
		ProviderKey: "runtime_status_exists",
		AuthMode:    "manual_token",
		DisplayName: "runtime status exists",
		Token:       "token-1",
	})
	if err != nil {
		t.Fatalf("CreateProfile() error = %v", err)
	}

	detail, err := svc.Create(ctx, CreateRequest{
		SourceProvider:  "guangya",
		TargetProvider:  "runtime_status_exists",
		TargetProfileID: profile.ID,
		ThresholdMB:     10,
		Entries: []planner.SourceEntry{
			{Path: "/same-by-status.bin", Size: 1024, MD5: "abc"},
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
		t.Fatalf("expected upload not to be called for explicit exists status, got %d", uploadCalls)
	}
	if got, _ := running.Results[0].Payload["syncDecision"].(string); got != "skip" {
		t.Fatalf("expected syncDecision skip, got %v", running.Results[0].Payload["syncDecision"])
	}
	if got, _ := running.Results[0].Payload["syncDecisionReason"].(string); got != "target_already_synced" {
		t.Fatalf("expected target_already_synced reason, got %v", running.Results[0].Payload["syncDecisionReason"])
	}
	if running.Runtime.SkippedCount != 1 {
		t.Fatalf("expected skipped count 1, got %d", running.Runtime.SkippedCount)
	}
}

func TestServiceRunTreatsAmbiguousMetadataOKAsCreate(t *testing.T) {
	ctx := context.Background()
	store, err := sqlitestore.New(ctx, filepath.Join(t.TempDir(), "runtime-ambiguous-metadata.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer func() { _ = store.Close() }()

	uploadCalls := 0
	adapter := &scriptedAdapter{
		meta: provider.Provider{
			Key:              "runtime_ambiguous_metadata",
			DisplayName:      "Runtime Ambiguous Metadata",
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
					Status:  "ok",
					Message: "generic ok must not imply existence",
					Mode:    "scripted_metadata",
				},
				Entry: map[string]interface{}{
					"path": req.Path,
					"size": int64(1024),
					"md5":  "abc",
				},
			}
		},
		uploadFunc: func(req provider.UploadRequest) provider.UploadResult {
			uploadCalls++
			return provider.UploadResult{
				OperationResult: provider.OperationResult{
					OK:      true,
					Status:  "ok",
					Message: "created",
					Mode:    "scripted_upload",
				},
			}
		},
	}

	registry := provider.NewRegistry(adapter)
	authSvc := auth.NewService(store, registry)
	svc := NewService(store, registry, authSvc)
	profile, err := authSvc.CreateProfile(ctx, auth.CreateProfileInput{
		ProviderKey: "runtime_ambiguous_metadata",
		AuthMode:    "manual_token",
		DisplayName: "runtime ambiguous metadata",
		Token:       "token-1",
	})
	if err != nil {
		t.Fatalf("CreateProfile() error = %v", err)
	}

	detail, err := svc.Create(ctx, CreateRequest{
		SourceProvider:  "guangya",
		TargetProvider:  "runtime_ambiguous_metadata",
		TargetProfileID: profile.ID,
		ThresholdMB:     10,
		Entries: []planner.SourceEntry{
			{Path: "/ambiguous-ok.bin", Size: 1024, MD5: "abc"},
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
		t.Fatalf("expected upload to be called for ambiguous metadata, got %d", uploadCalls)
	}
	if got, _ := running.Results[0].Payload["syncDecision"].(string); got != "create" {
		t.Fatalf("expected syncDecision create, got %v", running.Results[0].Payload["syncDecision"])
	}
	if got, _ := running.Results[0].Payload["executionMode"].(string); got != string(planner.ExecutionModeLeafFirstLazy) {
		t.Fatalf("expected create result execution mode leaf_first_lazy, got %v", running.Results[0].Payload["executionMode"])
	}
	if got, _ := running.Results[0].Payload["scanMode"].(string); got != "lazy_leaf_first" {
		t.Fatalf("expected create result scan mode lazy_leaf_first, got %v", running.Results[0].Payload["scanMode"])
	}
	if got, _ := running.Results[0].Payload["syncDecisionReason"].(string); got != "target_missing_or_metadata_unavailable" {
		t.Fatalf("expected conservative missing/unavailable reason, got %v", running.Results[0].Payload["syncDecisionReason"])
	}
	if running.Runtime.SkippedCount != 0 {
		t.Fatalf("expected skipped count 0, got %d", running.Runtime.SkippedCount)
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
	meta          provider.Provider
	capability    provider.CapabilitySet
	listFunc      func(req provider.ListRequest) provider.ListResult
	metadataFunc  func(req provider.MetadataRequest) provider.MetadataResult
	fastCheckFunc func(req provider.FastUploadCheckRequest) provider.FastUploadCheckResult
	uploadFunc    func(req provider.UploadRequest) provider.UploadResult
	listCalls     []string
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
	if a.fastCheckFunc != nil {
		return a.fastCheckFunc(req)
	}
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

func intTaskValue(raw interface{}) int {
	switch value := raw.(type) {
	case int:
		return value
	case int32:
		return int(value)
	case int64:
		return int(value)
	case float32:
		return int(value)
	case float64:
		return int(value)
	default:
		return 0
	}
}

func intPtrTask(value int) *int {
	return &value
}

func TestServiceProviderSmokeRecords(t *testing.T) {
	ctx := context.Background()
	store, err := sqlitestore.New(ctx, filepath.Join(t.TempDir(), "provider-smoke.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer func() { _ = store.Close() }()

	registry := provider.NewRegistry(provider.DefaultCatalog()...)
	authSvc := auth.NewService(store, registry)
	svc := NewService(store, registry, authSvc)

	record, err := svc.SaveProviderSmokeRecord(ctx, ProviderSmokeRecord{
		ProviderKey:   "123_open",
		ProtocolGroup: "aliyun_123_open",
		AuthMode:      "manual_token",
		Category:      "browse_only",
		Result:        "success",
		Title:         "123_open 真实 smoke",
		Note:          "ValidateAuth/List/Metadata",
		Operations:    []string{"ValidateAuth", "List", "Metadata"},
		Environment: map[string]string{
			"os": "windows",
		},
	})
	if err != nil {
		t.Fatalf("SaveProviderSmokeRecord() error = %v", err)
	}
	if record.ID == "" {
		t.Fatal("expected smoke record id")
	}

	items, err := svc.ListProviderSmokeRecords(ctx)
	if err != nil {
		t.Fatalf("ListProviderSmokeRecords() error = %v", err)
	}
	if len(items) == 0 {
		t.Fatal("expected smoke record history")
	}
	if got := items[0].ID; got != record.ID {
		t.Fatalf("expected latest smoke record id %s, got %s", record.ID, got)
	}

	fetched, ok, err := svc.GetProviderSmokeRecord(ctx, record.ID)
	if err != nil {
		t.Fatalf("GetProviderSmokeRecord() error = %v", err)
	}
	if !ok {
		t.Fatal("expected smoke record to exist")
	}
	if fetched.ProviderKey != "123_open" || fetched.Result != "success" {
		t.Fatalf("unexpected fetched smoke record: %#v", fetched)
	}
	if fetched.Category != "browse_only" {
		t.Fatalf("expected fetched smoke category browse_only, got %s", fetched.Category)
	}
	if len(fetched.Operations) != 3 {
		t.Fatalf("expected 3 operations, got %#v", fetched.Operations)
	}
	if !strings.Contains(fetched.Markdown, "123_open 真实 smoke") {
		t.Fatalf("expected markdown title, got %s", fetched.Markdown)
	}
	if !strings.Contains(fetched.Markdown, "## 固定记录模板") {
		t.Fatalf("expected fixed smoke template section, got %s", fetched.Markdown)
	}
	if !strings.Contains(fetched.Markdown, "phase2_smoke_template_v2") {
		t.Fatalf("expected smoke template version, got %s", fetched.Markdown)
	}
	if !strings.Contains(fetched.Markdown, "真实浏览成功样本") {
		t.Fatalf("expected browse sample type in markdown, got %s", fetched.Markdown)
	}
	if !strings.Contains(fetched.Markdown, "环境键") || !strings.Contains(fetched.Markdown, "os") {
		t.Fatalf("expected environment keys summary in markdown, got %s", fetched.Markdown)
	}
	if !strings.Contains(fetched.Markdown, "基础成功样本") {
		t.Fatalf("expected reuse advice in markdown, got %s", fetched.Markdown)
	}
	if !strings.Contains(fetched.Markdown, "代表性样本维度") || !strings.Contains(fetched.Markdown, "未标记") {
		t.Fatalf("expected representative dimension line in markdown, got %s", fetched.Markdown)
	}
	if !strings.Contains(fetched.Markdown, "自动补传/公平性关注点") {
		t.Fatalf("expected auto recover focus line in markdown, got %s", fetched.Markdown)
	}

	summary, err := svc.ProviderSmokeSummary(ctx)
	if err != nil {
		t.Fatalf("ProviderSmokeSummary() error = %v", err)
	}
	if len(summary) == 0 {
		t.Fatal("expected smoke summary")
	}
	if got := summary[0].ProtocolGroup; got != "aliyun_123_open" {
		t.Fatalf("expected smoke summary protocol group aliyun_123_open, got %s", got)
	}
	if got := summary[0].SmokeCount; got != 1 {
		t.Fatalf("expected smoke summary count 1, got %d", got)
	}
	if !summary[0].HasRealSuccessSample {
		t.Fatal("expected smoke summary success sample")
	}
	if summary[0].HasUploadSuccessSample {
		t.Fatal("expected browse-only smoke summary to keep upload success false")
	}
	if summary[0].SampleCategory != "browse_only" {
		t.Fatalf("expected smoke summary sample category browse_only, got %s", summary[0].SampleCategory)
	}

	matrix, err := svc.ProviderSmokeMatrix(ctx)
	if err != nil {
		t.Fatalf("ProviderSmokeMatrix() error = %v", err)
	}
	if len(matrix) == 0 {
		t.Fatal("expected smoke matrix")
	}
	if matrix[0].AcceptanceStatus != "in_progress" {
		t.Fatalf("expected smoke matrix in_progress status, got %s", matrix[0].AcceptanceStatus)
	}
	if matrix[0].HasUploadSuccessSample {
		t.Fatal("expected browse-only smoke matrix to keep upload success false")
	}
	if len(matrix[0].AcceptanceMissing) == 0 {
		t.Fatal("expected smoke matrix missing reasons")
	}
	if matrix[0].AcceptanceMissing[0] != "upload_smoke_success_missing" {
		t.Fatalf("expected upload_smoke_success_missing reason, got %#v", matrix[0].AcceptanceMissing)
	}
	if len(matrix[0].AcceptanceActions) == 0 {
		t.Fatal("expected smoke matrix acceptance actions")
	}
	if matrix[0].AcceptanceActions[0] != "补 1 条真实上传成功样本" {
		t.Fatalf("expected upload success acceptance action, got %#v", matrix[0].AcceptanceActions)
	}
	if !strings.Contains(matrix[0].AcceptanceAdvice, "真实上传成功样本") {
		t.Fatalf("expected advice to mention upload sample, got %s", matrix[0].AcceptanceAdvice)
	}
	if len(matrix[0].RepresentativeMissing) != 3 {
		t.Fatalf("expected representative missing 3, got %#v", matrix[0].RepresentativeMissing)
	}
	if matrix[0].AnomalyCompletedCount != 0 || matrix[0].AnomalyTargetCount != 4 {
		t.Fatalf("expected anomaly coverage 0/4, got %d/%d", matrix[0].AnomalyCompletedCount, matrix[0].AnomalyTargetCount)
	}
	if matrix[0].RepresentativeCompletedCount != 0 || matrix[0].RepresentativeTargetCount != 3 {
		t.Fatalf("expected representative coverage 0/3, got %d/%d", matrix[0].RepresentativeCompletedCount, matrix[0].RepresentativeTargetCount)
	}
	if len(matrix[0].RepresentativeActions) != 3 {
		t.Fatalf("expected representative actions 3, got %#v", matrix[0].RepresentativeActions)
	}
	if !strings.Contains(matrix[0].RepresentativeAdvice, "代表性样本建议") {
		t.Fatalf("expected representative advice, got %s", matrix[0].RepresentativeAdvice)
	}

	evidence, err := svc.RuntimeEvidence(ctx)
	if err != nil {
		t.Fatalf("RuntimeEvidence() error = %v", err)
	}
	if evidence.AcceptedSmokeGroups != 0 {
		t.Fatalf("expected accepted smoke groups 0, got %d", evidence.AcceptedSmokeGroups)
	}
	if evidence.InProgressSmokeGroups != 1 {
		t.Fatalf("expected in-progress smoke groups 1, got %d", evidence.InProgressSmokeGroups)
	}
	if evidence.PendingSmokeGroups != len(matrix)-1 {
		t.Fatalf("expected pending smoke groups %d, got %d", len(matrix)-1, evidence.PendingSmokeGroups)
	}
	if evidence.UploadSuccessGroups != 0 {
		t.Fatalf("expected upload success groups 0, got %d", evidence.UploadSuccessGroups)
	}
	if evidence.UploadSuccessSamples != 0 {
		t.Fatalf("expected upload success samples 0, got %d", evidence.UploadSuccessSamples)
	}
	if got := evidence.AcceptanceActionCounts["补 1 条真实上传成功样本"]; got < 1 {
		t.Fatalf("expected acceptance action count for upload success sample >= 1, got %d", got)
	}
}

func TestServiceProviderSmokeMatrixTracksUploadSuccessSample(t *testing.T) {
	ctx := context.Background()
	store, err := sqlitestore.New(ctx, filepath.Join(t.TempDir(), "provider-smoke-upload.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer func() { _ = store.Close() }()

	registry := provider.NewRegistry(provider.DefaultCatalog()...)
	authSvc := auth.NewService(store, registry)
	svc := NewService(store, registry, authSvc)

	_, err = svc.SaveProviderSmokeRecord(ctx, ProviderSmokeRecord{
		ProviderKey:   "123_open",
		ProtocolGroup: "aliyun_123_open",
		AuthMode:      "manual_token",
		Category:      "binary_upload_success",
		Result:        "success",
		Title:         "123_open 上传 smoke",
		Note:          "Upload success",
		Operations:    []string{"ValidateAuth", "Upload"},
	})
	if err != nil {
		t.Fatalf("SaveProviderSmokeRecord(upload) error = %v", err)
	}

	summary, err := svc.ProviderSmokeSummary(ctx)
	if err != nil {
		t.Fatalf("ProviderSmokeSummary() error = %v", err)
	}
	if len(summary) == 0 {
		t.Fatal("expected smoke summary")
	}
	if !summary[0].HasUploadSuccessSample {
		t.Fatal("expected upload success sample in smoke summary")
	}
	if summary[0].UploadSuccessCount != 1 {
		t.Fatalf("expected upload success count 1 in smoke summary, got %d", summary[0].UploadSuccessCount)
	}

	matrix, err := svc.ProviderSmokeMatrix(ctx)
	if err != nil {
		t.Fatalf("ProviderSmokeMatrix() error = %v", err)
	}
	if len(matrix) == 0 {
		t.Fatal("expected smoke matrix")
	}
	if !matrix[0].HasUploadSuccessSample {
		t.Fatal("expected upload success sample in smoke matrix")
	}
	if matrix[0].UploadSuccessCount != 1 {
		t.Fatalf("expected upload success count 1 in smoke matrix, got %d", matrix[0].UploadSuccessCount)
	}
	if len(matrix[0].AcceptanceActions) == 0 {
		t.Fatal("expected in-progress smoke matrix acceptance actions")
	}
	if !strings.Contains(matrix[0].AcceptanceActions[0], "任务覆盖样本") {
		t.Fatalf("expected in-progress smoke matrix task coverage action, got %#v", matrix[0].AcceptanceActions)
	}

	evidence, err := svc.RuntimeEvidence(ctx)
	if err != nil {
		t.Fatalf("RuntimeEvidence() error = %v", err)
	}
	if evidence.UploadSuccessGroups != 1 {
		t.Fatalf("expected runtime upload success groups 1, got %d", evidence.UploadSuccessGroups)
	}
	if evidence.AcceptedSmokeGroups != 0 {
		t.Fatalf("expected accepted smoke groups 0 without task coverage, got %d", evidence.AcceptedSmokeGroups)
	}
	if evidence.InProgressSmokeGroups != 1 {
		t.Fatalf("expected in-progress smoke groups 1, got %d", evidence.InProgressSmokeGroups)
	}
	if evidence.PendingSmokeGroups != len(matrix)-1 {
		t.Fatalf("expected pending smoke groups %d, got %d", len(matrix)-1, evidence.PendingSmokeGroups)
	}
	if evidence.UploadSuccessSamples != 1 {
		t.Fatalf("expected upload success samples 1, got %d", evidence.UploadSuccessSamples)
	}
	if got := evidence.AcceptanceActionCounts["补 1 条真实任务覆盖样本"]; got < 1 {
		t.Fatalf("expected acceptance action count for task coverage sample >= 1, got %d", got)
	}

	report, err := svc.EvidenceReport(ctx)
	if err != nil {
		t.Fatalf("EvidenceReport() error = %v", err)
	}
	if report.Summary.AcceptedSmokeGroups != 0 {
		t.Fatalf("expected report accepted smoke groups 0, got %d", report.Summary.AcceptedSmokeGroups)
	}
	if report.Summary.InProgressSmokeGroups != 1 {
		t.Fatalf("expected report in-progress smoke groups 1, got %d", report.Summary.InProgressSmokeGroups)
	}
	if report.Summary.PendingSmokeGroups != len(matrix)-1 {
		t.Fatalf("expected report pending smoke groups %d, got %d", len(matrix)-1, report.Summary.PendingSmokeGroups)
	}
	if report.Summary.UploadSuccessGroups != 1 {
		t.Fatalf("expected report upload success groups 1, got %d", report.Summary.UploadSuccessGroups)
	}
	if report.Summary.UploadSuccessSamples != 1 {
		t.Fatalf("expected report upload success samples 1, got %d", report.Summary.UploadSuccessSamples)
	}
	if !strings.Contains(report.Markdown, "已验收协议组: 0") {
		t.Fatalf("expected accepted smoke groups line in report markdown, got %s", report.Markdown)
	}
	if !strings.Contains(report.Markdown, "进行中协议组: 1") {
		t.Fatalf("expected in-progress smoke groups line in report markdown, got %s", report.Markdown)
	}
	if !strings.Contains(report.Markdown, fmt.Sprintf("待补齐协议组: %d", len(matrix)-1)) {
		t.Fatalf("expected pending smoke groups line in report markdown, got %s", report.Markdown)
	}
	if !strings.Contains(report.Markdown, "上传成功协议组: 1") {
		t.Fatalf("expected upload success groups line in report markdown, got %s", report.Markdown)
	}
	if !strings.Contains(report.Markdown, "上传成功样本数: 1") {
		t.Fatalf("expected upload success samples line in report markdown, got %s", report.Markdown)
	}
	if !strings.Contains(report.Markdown, "### 样本补齐总览") {
		t.Fatalf("expected smoke completion summary heading in report markdown, got %s", report.Markdown)
	}
	if !strings.Contains(report.Markdown, "| ProtocolGroup | Upload Success | Task Coverage | Anomaly Coverage | Representative Coverage |") {
		t.Fatalf("expected smoke completion summary table in report markdown, got %s", report.Markdown)
	}
	if !strings.Contains(report.Markdown, "### 样本完成清单") || !strings.Contains(report.Markdown, "upload ready / coverage 0/0 / anomaly 0/4 / representative 0/3") {
		t.Fatalf("expected smoke checklist summary in report markdown, got %s", report.Markdown)
	}
}

func TestServiceProviderSmokeMatrixTracksRepresentativeSamples(t *testing.T) {
	ctx := context.Background()
	store, err := sqlitestore.New(ctx, filepath.Join(t.TempDir(), "provider-smoke-representative.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer func() { _ = store.Close() }()

	registry := provider.NewRegistry(provider.DefaultCatalog()...)
	authSvc := auth.NewService(store, registry)
	svc := NewService(store, registry, authSvc)

	_, err = svc.SaveProviderSmokeRecord(ctx, ProviderSmokeRecord{
		ProviderKey:   "guangya",
		ProtocolGroup: "guangya",
		AuthMode:      "manual_token",
		Category:      "binary_upload_success",
		Result:        "success",
		Title:         "guangya 大文件 多层目录 checkpoint smoke",
		Note:          "大文件 multipart 多层目录 leaf_first checkpoint 重试恢复 续传",
		Operations:    []string{"ValidateAuth", "List", "Metadata", "Upload", "leaf_first", "checkpoint"},
	})
	if err != nil {
		t.Fatalf("SaveProviderSmokeRecord(representative) error = %v", err)
	}

	summary, err := svc.ProviderSmokeSummary(ctx)
	if err != nil {
		t.Fatalf("ProviderSmokeSummary() error = %v", err)
	}
	if len(summary) == 0 {
		t.Fatal("expected smoke summary")
	}
	if !summary[0].HasLargeFileSample || !summary[0].HasNestedDirectorySample || !summary[0].HasRetryRecoverySample {
		t.Fatalf("expected representative sample flags true, got %#v", summary[0])
	}

	matrix, err := svc.ProviderSmokeMatrix(ctx)
	if err != nil {
		t.Fatalf("ProviderSmokeMatrix() error = %v", err)
	}
	if len(matrix) == 0 {
		t.Fatal("expected smoke matrix")
	}
	if !matrix[0].HasLargeFileSample || !matrix[0].HasNestedDirectorySample || !matrix[0].HasRetryRecoverySample {
		t.Fatalf("expected matrix representative sample flags true, got %#v", matrix[0])
	}
	if len(matrix[0].RepresentativeMissing) != 0 {
		t.Fatalf("expected representative missing empty, got %#v", matrix[0].RepresentativeMissing)
	}
	if matrix[0].RepresentativeCompletedCount != 3 || matrix[0].RepresentativeTargetCount != 3 {
		t.Fatalf("expected representative coverage 3/3, got %d/%d", matrix[0].RepresentativeCompletedCount, matrix[0].RepresentativeTargetCount)
	}
	if len(matrix[0].RepresentativeActions) != 1 {
		t.Fatalf("expected representative actions len 1 when complete, got %#v", matrix[0].RepresentativeActions)
	}
	if !strings.Contains(matrix[0].RepresentativeAdvice, "代表性样本已覆盖") {
		t.Fatalf("expected representative complete advice, got %s", matrix[0].RepresentativeAdvice)
	}
	fetched, ok, err := svc.GetProviderSmokeRecord(ctx, summary[0].SampleRecordID)
	if err != nil {
		t.Fatalf("GetProviderSmokeRecord(representative) error = %v", err)
	}
	if !ok {
		t.Fatal("expected representative smoke record to exist")
	}
	if !strings.Contains(fetched.Markdown, "代表性样本维度: 大文件 / 多层目录 / 重试恢复/续传") {
		t.Fatalf("expected representative labels in markdown, got %s", fetched.Markdown)
	}
	if !strings.Contains(fetched.Markdown, "自动补传/公平性关注点") {
		t.Fatalf("expected auto recover focus in representative markdown, got %s", fetched.Markdown)
	}
}

func TestRecoverBudgetsPreferPlannerMetadataAndFallback(t *testing.T) {
	detailWithBudget := Detail{
		Task: Task{TargetProvider: "aliyundrive_open"},
		Plan: planner.Plan{
			Metadata: map[string]interface{}{
				"riskProfile": planner.RiskProfile{MaxConcurrent: 6},
				"riskProfileResolution": planner.RiskProfileResolution{
					ProviderKey: "aliyundrive_open",
					RecoverBudget: planner.RecoverBudgetPolicy{
						ProtocolGroupBudget: 2,
						ProviderBudget:      5,
						ProfileBudget:       2,
						Reason:              "test budget",
					},
				},
			},
		},
	}
	if got := recoverProtocolGroupBudget(detailWithBudget); got != 2 {
		t.Fatalf("expected metadata protocol group budget 2, got %d", got)
	}
	if got := recoverProviderBudget(detailWithBudget); got != 5 {
		t.Fatalf("expected metadata provider budget 5, got %d", got)
	}
	if got := recoverProfileBudget(detailWithBudget); got != 2 {
		t.Fatalf("expected metadata profile budget 2, got %d", got)
	}

	detailFallback := Detail{
		Task: Task{TargetProvider: "189cloud"},
		Plan: planner.Plan{
			Metadata: map[string]interface{}{
				"riskProfile": planner.RiskProfile{MaxConcurrent: 4},
			},
		},
	}
	if got := recoverProtocolGroupBudget(detailFallback); got != 2 {
		t.Fatalf("expected fallback protocol group budget 2, got %d", got)
	}
	if got := recoverProviderBudget(detailFallback); got != 4 {
		t.Fatalf("expected fallback provider budget 4, got %d", got)
	}
	if got := recoverProfileBudget(detailFallback); got != 1 {
		t.Fatalf("expected sensitive fallback profile budget 1, got %d", got)
	}
}

func boolValue(value interface{}) bool {
	typed, ok := value.(bool)
	return ok && typed
}

func autoRecoverLaneByMode(items []AutoRecoverLane, mode string) AutoRecoverLane {
	for _, item := range items {
		if item.Mode == mode {
			return item
		}
	}
	return AutoRecoverLane{}
}
