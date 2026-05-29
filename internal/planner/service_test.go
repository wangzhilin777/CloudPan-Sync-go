package planner

import (
	"errors"
	"testing"

	"cloudpan-sync-go/internal/provider"
)

func TestBuildPreviewClassifiesStrategies(t *testing.T) {
	registry := provider.NewRegistry(provider.DefaultCatalog()...)
	plan, err := BuildPreview(registry, PreviewRequest{
		SourceProvider: "guangya",
		TargetProvider: "aliyundrive_open",
		ThresholdMB:    10,
		Entries: []SourceEntry{
			{Path: "/a.bin", Size: 1024, SHA1: "sha1-a"},
			{Path: "/b.bin", Size: 1024},
			{Path: "/c.bin", Size: 20 * 1024 * 1024},
		},
	})
	if err != nil {
		t.Fatalf("BuildPreview() error = %v", err)
	}
	if got := plan.Items[0].Strategy; got != StrategyFastUpload {
		t.Fatalf("expected first item fast_upload, got %s", got)
	}
	if got := plan.Items[1].Strategy; got != StrategyDownloadUpload {
		t.Fatalf("expected second item download_upload, got %s", got)
	}
	if got := plan.Items[2].Strategy; got != StrategyPendingManual {
		t.Fatalf("expected third item pending_manual, got %s", got)
	}
}

func TestBuildPreviewOrdersLeafFirstAndAssignsSequence(t *testing.T) {
	registry := provider.NewRegistry(provider.DefaultCatalog()...)
	plan, err := BuildPreview(registry, PreviewRequest{
		SourceProvider: "guangya",
		TargetProvider: "123_open",
		ThresholdMB:    10,
		Entries: []SourceEntry{
			{Path: "/root.bin", Size: 10},
			{Path: "/a/b/c.bin", Size: 10},
			{Path: "/a/b.bin", Size: 10},
		},
	})
	if err != nil {
		t.Fatalf("BuildPreview() error = %v", err)
	}

	if got := plan.Items[0].Path; got != "/a/b/c.bin" {
		t.Fatalf("expected deepest path first, got %s", got)
	}
	if got := plan.Items[1].Path; got != "/a/b.bin" {
		t.Fatalf("expected second path /a/b.bin, got %s", got)
	}
	if got := plan.Items[2].Path; got != "/root.bin" {
		t.Fatalf("expected root path last, got %s", got)
	}
	if got := plan.Items[0].Sequence; got != 1 {
		t.Fatalf("expected first sequence=1, got %d", got)
	}
	if got := plan.Items[2].Sequence; got != 3 {
		t.Fatalf("expected last sequence=3, got %d", got)
	}
	if got, _ := plan.Metadata["executionOrder"].(string); got != "leaf_first" {
		t.Fatalf("expected executionOrder leaf_first, got %v", plan.Metadata["executionOrder"])
	}
	if got, _ := plan.Metadata["executionMode"].(ExecutionMode); got != ExecutionModeLeafFirstLazy {
		t.Fatalf("expected default execution mode leaf_first_lazy, got %v", plan.Metadata["executionMode"])
	}
}

func TestBuildPreviewIncludesRiskProfileDefaults(t *testing.T) {
	registry := provider.NewRegistry(provider.DefaultCatalog()...)
	plan, err := BuildPreview(registry, PreviewRequest{
		SourceProvider: "guangya",
		TargetProvider: "189cloud",
		RiskMode:       RiskModeSafe,
		Entries:        []SourceEntry{{Path: "/a.bin", Size: 10}},
	})
	if err != nil {
		t.Fatalf("BuildPreview() error = %v", err)
	}

	riskProfile, ok := plan.Metadata["riskProfile"].(RiskProfile)
	if !ok {
		t.Fatalf("expected riskProfile metadata, got %#v", plan.Metadata["riskProfile"])
	}
	if riskProfile.Mode != RiskModeSafe {
		t.Fatalf("expected risk mode safe, got %s", riskProfile.Mode)
	}
	if riskProfile.RequestIntervalMS <= 0 {
		t.Fatalf("expected request interval > 0, got %d", riskProfile.RequestIntervalMS)
	}
	if len(riskProfile.RiskKeywords) == 0 {
		t.Fatal("expected provider risk keywords")
	}
	resolution, ok := plan.Metadata["riskProfileResolution"].(RiskProfileResolution)
	if !ok {
		t.Fatalf("expected riskProfileResolution metadata, got %#v", plan.Metadata["riskProfileResolution"])
	}
	if resolution.ProviderKey != "189cloud" || resolution.Mode != RiskModeSafe {
		t.Fatalf("unexpected risk profile resolution: %+v", resolution)
	}
	if resolution.Applied.RequestIntervalMS != riskProfile.RequestIntervalMS {
		t.Fatalf("expected applied risk profile to match riskProfile metadata, got %+v vs %+v", resolution.Applied, riskProfile)
	}
}

func TestBuildPreviewCalibratesRiskProfileByProvider(t *testing.T) {
	registry := provider.NewRegistry(provider.DefaultCatalog()...)
	tests := []struct {
		name              string
		targetProvider    string
		riskMode          RiskMode
		wantRequestMin    int
		wantPageMax       int
		wantDirectoryMin  int
		wantCooldownMin   int
		wantRetryLimitMax int
		wantKeyword       string
	}{
		{
			name:              "baidu safe is conservative",
			targetProvider:    "baidu_netdisk",
			riskMode:          RiskModeSafe,
			wantRequestMin:    1800,
			wantPageMax:       100,
			wantDirectoryMin:  3000,
			wantCooldownMin:   45,
			wantRetryLimitMax: 2,
			wantKeyword:       "hit_risk_control",
		},
		{
			name:             "quark balanced slows down risky web auth",
			targetProvider:   "quark",
			riskMode:         RiskModeBalanced,
			wantRequestMin:   1400,
			wantPageMax:      120,
			wantDirectoryMin: 2200,
			wantCooldownMin:  40,
			wantKeyword:      "captcha",
		},
		{
			name:             "aliyun fast keeps larger page budget",
			targetProvider:   "aliyundrive_open",
			riskMode:         RiskModeFast,
			wantRequestMin:   250,
			wantPageMax:      500,
			wantDirectoryMin: 300,
			wantCooldownMin:  5,
			wantKeyword:      "flow_limit",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan, err := BuildPreview(registry, PreviewRequest{
				SourceProvider: "guangya",
				TargetProvider: tt.targetProvider,
				RiskMode:       tt.riskMode,
				Entries:        []SourceEntry{{Path: "/a.bin", Size: 10}},
			})
			if err != nil {
				t.Fatalf("BuildPreview() error = %v", err)
			}
			riskProfile, ok := plan.Metadata["riskProfile"].(RiskProfile)
			if !ok {
				t.Fatalf("expected riskProfile metadata, got %#v", plan.Metadata["riskProfile"])
			}
			if riskProfile.RequestIntervalMS < tt.wantRequestMin {
				t.Fatalf("expected request interval >= %d, got %d", tt.wantRequestMin, riskProfile.RequestIntervalMS)
			}
			if riskProfile.PageSize > tt.wantPageMax {
				t.Fatalf("expected page size <= %d, got %d", tt.wantPageMax, riskProfile.PageSize)
			}
			if riskProfile.DirectoryIntervalMS < tt.wantDirectoryMin {
				t.Fatalf("expected directory interval >= %d, got %d", tt.wantDirectoryMin, riskProfile.DirectoryIntervalMS)
			}
			if riskProfile.CooldownSeconds < tt.wantCooldownMin {
				t.Fatalf("expected cooldown >= %d, got %d", tt.wantCooldownMin, riskProfile.CooldownSeconds)
			}
			if tt.wantRetryLimitMax > 0 && riskProfile.RetryLimit > tt.wantRetryLimitMax {
				t.Fatalf("expected retryLimit <= %d, got %d", tt.wantRetryLimitMax, riskProfile.RetryLimit)
			}
			if riskProfile.MaxConcurrent <= 0 {
				t.Fatalf("expected maxConcurrent > 0, got %d", riskProfile.MaxConcurrent)
			}
			if !containsString(riskProfile.RiskKeywords, tt.wantKeyword) {
				t.Fatalf("expected keyword %q in %#v", tt.wantKeyword, riskProfile.RiskKeywords)
			}
		})
	}
}

func TestBuildPreviewCustomRiskModeSkipsProviderCalibration(t *testing.T) {
	registry := provider.NewRegistry(provider.DefaultCatalog()...)
	plan, err := BuildPreview(registry, PreviewRequest{
		SourceProvider: "guangya",
		TargetProvider: "baidu_netdisk",
		RiskMode:       RiskModeCustom,
		Entries:        []SourceEntry{{Path: "/a.bin", Size: 10}},
	})
	if err != nil {
		t.Fatalf("BuildPreview() error = %v", err)
	}
	riskProfile, ok := plan.Metadata["riskProfile"].(RiskProfile)
	if !ok {
		t.Fatalf("expected riskProfile metadata, got %#v", plan.Metadata["riskProfile"])
	}
	if riskProfile.RequestIntervalMS != 0 || riskProfile.PageSize != 0 || riskProfile.DirectoryIntervalMS != 0 || riskProfile.CooldownSeconds != 0 || riskProfile.RetryLimit != 0 {
		t.Fatalf("expected custom mode to keep zero baseline before override, got %+v", riskProfile)
	}
	if !containsString(riskProfile.RiskKeywords, "hit_risk_control") {
		t.Fatalf("expected provider risk keywords to remain in custom mode, got %#v", riskProfile.RiskKeywords)
	}
}

func TestBuildPreviewAppliesRiskOverride(t *testing.T) {
	registry := provider.NewRegistry(provider.DefaultCatalog()...)
	plan, err := BuildPreview(registry, PreviewRequest{
		SourceProvider: "guangya",
		TargetProvider: "189cloud",
		RiskMode:       RiskModeBalanced,
		RiskOverride: &RiskProfileOverride{
			RequestIntervalMS:   intPtr(1200),
			PageSize:            intPtr(88),
			DirectoryIntervalMS: intPtr(2200),
			CooldownSeconds:     intPtr(45),
			RetryLimit:          intPtr(1),
			MaxConcurrent:       intPtr(1),
			AutoRetryStartHour:  intPtr(1),
			AutoRetryEndHour:    intPtr(7),
			RiskKeywords:        []string{"rate_limited", "captcha"},
		},
		Entries: []SourceEntry{{Path: "/a.bin", Size: 10}},
	})
	if err != nil {
		t.Fatalf("BuildPreview() error = %v", err)
	}

	riskProfile, ok := plan.Metadata["riskProfile"].(RiskProfile)
	if !ok {
		t.Fatalf("expected riskProfile metadata, got %#v", plan.Metadata["riskProfile"])
	}
	if riskProfile.RequestIntervalMS != 1200 {
		t.Fatalf("expected requestIntervalMs 1200, got %d", riskProfile.RequestIntervalMS)
	}
	if riskProfile.PageSize != 88 {
		t.Fatalf("expected pageSize 88, got %d", riskProfile.PageSize)
	}
	if riskProfile.DirectoryIntervalMS != 2200 {
		t.Fatalf("expected directoryIntervalMs 2200, got %d", riskProfile.DirectoryIntervalMS)
	}
	if riskProfile.CooldownSeconds != 45 {
		t.Fatalf("expected cooldownSeconds 45, got %d", riskProfile.CooldownSeconds)
	}
	if riskProfile.RetryLimit != 1 {
		t.Fatalf("expected retryLimit 1, got %d", riskProfile.RetryLimit)
	}
	if riskProfile.MaxConcurrent != 1 {
		t.Fatalf("expected maxConcurrent 1, got %d", riskProfile.MaxConcurrent)
	}
	if riskProfile.AutoRetryStartHour != 1 || riskProfile.AutoRetryEndHour != 7 {
		t.Fatalf("expected auto retry window 1-7, got %+v", riskProfile)
	}
	if len(riskProfile.RiskKeywords) != 2 || riskProfile.RiskKeywords[0] != "rate_limited" {
		t.Fatalf("expected override risk keywords, got %#v", riskProfile.RiskKeywords)
	}
	resolution, ok := plan.Metadata["riskProfileResolution"].(RiskProfileResolution)
	if !ok {
		t.Fatalf("expected riskProfileResolution metadata, got %#v", plan.Metadata["riskProfileResolution"])
	}
	if len(resolution.CalibrationReasons) == 0 {
		t.Fatalf("expected calibration reasons, got %+v", resolution)
	}
	if len(resolution.OverrideFields) != 9 {
		t.Fatalf("expected 9 override fields, got %#v", resolution.OverrideFields)
	}
	if resolution.Applied.RetryLimit != 1 || resolution.Applied.PageSize != 88 {
		t.Fatalf("expected applied risk profile to reflect override, got %+v", resolution.Applied)
	}
}

func TestBuildPreviewSupportsPreScanFlatMode(t *testing.T) {
	registry := provider.NewRegistry(provider.DefaultCatalog()...)
	plan, err := BuildPreview(registry, PreviewRequest{
		SourceProvider: "guangya",
		TargetProvider: "123_open",
		ExecutionMode:  ExecutionModePreScanFlat,
		Entries: []SourceEntry{
			{Path: "/root.bin", Size: 10},
			{Path: "/a/b/c.bin", Size: 10},
			{Path: "/a/b.bin", Size: 10},
		},
	})
	if err != nil {
		t.Fatalf("BuildPreview() error = %v", err)
	}
	if got := plan.Items[0].Path; got != "/root.bin" {
		t.Fatalf("expected pre-scan mode to preserve input order, got %s", got)
	}
	if got, _ := plan.Metadata["executionMode"].(ExecutionMode); got != ExecutionModePreScanFlat {
		t.Fatalf("expected pre_scan_flat mode, got %v", plan.Metadata["executionMode"])
	}
	if got, _ := plan.Metadata["executionOrder"].(string); got != "pre_scan_flat" {
		t.Fatalf("expected executionOrder pre_scan_flat, got %v", plan.Metadata["executionOrder"])
	}
}

func TestBuildPreviewTracksSourceDeletionRecords(t *testing.T) {
	registry := provider.NewRegistry(provider.DefaultCatalog()...)
	plan, err := BuildPreview(registry, PreviewRequest{
		SourceProvider:     "guangya",
		TargetProvider:     "123_open",
		SourceDeletePolicy: SourceDeletePolicyRecordOnly,
		SelectedRoots:      []string{"/demo"},
		Entries: []SourceEntry{
			{Path: "/demo/live/a.bin", Size: 10, MD5: "md5-a"},
			{Path: "/demo/deleted.bin", Deleted: true, DeletedAt: "2026-05-29T10:00:00Z", DeleteReason: "source_removed"},
		},
	})
	if err != nil {
		t.Fatalf("BuildPreview() error = %v", err)
	}
	if len(plan.Items) != 1 {
		t.Fatalf("expected only active entries to become plan items, got %d", len(plan.Items))
	}
	if got := plan.Items[0].Path; got != "/demo/live/a.bin" {
		t.Fatalf("expected active entry path /demo/live/a.bin, got %s", got)
	}
	if got, _ := plan.Metadata["activeEntryCount"].(int); got != 1 {
		t.Fatalf("expected activeEntryCount 1, got %#v", plan.Metadata["activeEntryCount"])
	}
	if got, _ := plan.Metadata["deletedEntryCount"].(int); got != 1 {
		t.Fatalf("expected deletedEntryCount 1, got %#v", plan.Metadata["deletedEntryCount"])
	}
	if got, _ := plan.Metadata["sourceDeletePolicy"].(SourceDeletePolicy); got != SourceDeletePolicyRecordOnly {
		t.Fatalf("expected sourceDeletePolicy record_only, got %#v", plan.Metadata["sourceDeletePolicy"])
	}
	records, ok := plan.Metadata["sourceDeletionRecords"].([]map[string]interface{})
	if !ok {
		t.Fatalf("expected sourceDeletionRecords metadata, got %#v", plan.Metadata["sourceDeletionRecords"])
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 source deletion record, got %#v", records)
	}
	if records[0]["path"] != "/demo/deleted.bin" {
		t.Fatalf("expected deleted path /demo/deleted.bin, got %#v", records[0]["path"])
	}
	if records[0]["rootPath"] != "/demo" {
		t.Fatalf("expected deleted rootPath /demo, got %#v", records[0]["rootPath"])
	}
	if records[0]["deleteReason"] != "source_removed" {
		t.Fatalf("expected deleteReason source_removed, got %#v", records[0]["deleteReason"])
	}
}

func TestBuildPreviewRejectsInvalidSourceDeletePolicy(t *testing.T) {
	registry := provider.NewRegistry(provider.DefaultCatalog()...)
	_, err := BuildPreview(registry, PreviewRequest{
		SourceProvider:     "guangya",
		TargetProvider:     "123_open",
		SourceDeletePolicy: SourceDeletePolicy("delete_target"),
		Entries:            []SourceEntry{{Path: "/a.bin", Size: 10}},
	})
	if err == nil {
		t.Fatal("expected invalid source delete policy error")
	}
	if !errors.Is(err, ErrInvalidSourceDeletePolicy) {
		t.Fatalf("expected ErrInvalidSourceDeletePolicy, got %v", err)
	}
}

func TestBuildPreviewClampsAutoRetryWindowOverride(t *testing.T) {
	registry := provider.NewRegistry(provider.DefaultCatalog()...)
	plan, err := BuildPreview(registry, PreviewRequest{
		SourceProvider: "guangya",
		TargetProvider: "189cloud",
		RiskMode:       RiskModeBalanced,
		RiskOverride: &RiskProfileOverride{
			AutoRetryStartHour: intPtr(99),
			AutoRetryEndHour:   intPtr(88),
		},
		Entries: []SourceEntry{{Path: "/a.bin", Size: 10}},
	})
	if err != nil {
		t.Fatalf("BuildPreview() error = %v", err)
	}

	riskProfile, ok := plan.Metadata["riskProfile"].(RiskProfile)
	if !ok {
		t.Fatalf("expected riskProfile metadata, got %#v", plan.Metadata["riskProfile"])
	}
	if riskProfile.AutoRetryStartHour != 23 || riskProfile.AutoRetryEndHour != 24 {
		t.Fatalf("expected clamped auto retry window 23-24, got %+v", riskProfile)
	}
	resolution, ok := plan.Metadata["riskProfileResolution"].(RiskProfileResolution)
	if !ok {
		t.Fatalf("expected riskProfileResolution metadata, got %#v", plan.Metadata["riskProfileResolution"])
	}
	if resolution.Applied.AutoRetryStartHour != 23 || resolution.Applied.AutoRetryEndHour != 24 {
		t.Fatalf("expected clamped applied auto retry window 23-24, got %+v", resolution.Applied)
	}
}

func TestBuildPreviewRejectsInvalidExecutionMode(t *testing.T) {
	registry := provider.NewRegistry(provider.DefaultCatalog()...)
	_, err := BuildPreview(registry, PreviewRequest{
		SourceProvider: "guangya",
		TargetProvider: "123_open",
		ExecutionMode:  ExecutionMode("bad_mode"),
	})
	if err == nil {
		t.Fatal("expected invalid execution mode error")
	}
	if !errors.Is(err, ErrInvalidExecutionMode) {
		t.Fatalf("expected ErrInvalidExecutionMode, got %v", err)
	}
}

func intPtr(value int) *int {
	return &value
}

func containsString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}
