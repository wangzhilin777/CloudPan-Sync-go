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
