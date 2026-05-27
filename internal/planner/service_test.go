package planner

import (
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
			{Path: "/a.bin", Size: 1024, MD5: "abc"},
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
}
