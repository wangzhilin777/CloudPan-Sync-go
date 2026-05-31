package planner

import (
	"testing"

	"cloudpan-sync-go/internal/provider"
)

func TestBuildPreviewRecommendationContracts(t *testing.T) {
	registry := provider.NewRegistry(provider.DefaultCatalog()...)
	tests := []struct {
		name           string
		req            PreviewRequest
		wantMode       ExecutionMode
		wantExecution  string
		wantReasonPart string
	}{
		{
			name: "multiple roots prefer leaf-first lazy",
			req: PreviewRequest{
				SourceProvider: "guangya",
				TargetProvider: "123_open",
				RiskMode:       RiskModeBalanced,
				SelectedRoots:  []string{"/alpha", "/beta"},
				Entries: []SourceEntry{
					{Path: "/alpha/a.bin", Size: 8, MD5: "md5-a"},
					{Path: "/beta/b.bin", Size: 8, MD5: "md5-b"},
				},
			},
			wantMode:       ExecutionModeLeafFirstLazy,
			wantExecution:  "leaf_first",
			wantReasonPart: "Multiple top-level roots",
		},
		{
			name: "small single-root balanced set recommends pre-scan",
			req: PreviewRequest{
				SourceProvider: "guangya",
				TargetProvider: "aliyundrive_open",
				RiskMode:       RiskModeBalanced,
				SelectedRoots:  []string{"/demo"},
				Entries: []SourceEntry{
					{Path: "/demo/a.bin", Size: 8, SHA1: "sha1-a"},
					{Path: "/demo/b.bin", Size: 8, SHA1: "sha1-b"},
				},
			},
			wantMode:       ExecutionModePreScanFlat,
			wantExecution:  "leaf_first",
			wantReasonPart: "Known small input set",
		},
		{
			name: "unknown tree defaults to leaf-first lazy",
			req: PreviewRequest{
				SourceProvider: "guangya",
				TargetProvider: "quark",
				RiskMode:       RiskModeFast,
				SelectedRoots:  []string{"/unknown"},
			},
			wantMode:       ExecutionModeLeafFirstLazy,
			wantExecution:  "leaf_first",
			wantReasonPart: "Unknown full tree size",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan, err := BuildPreview(registry, tt.req)
			if err != nil {
				t.Fatalf("BuildPreview() error = %v", err)
			}
			mode, _ := plan.Metadata["recommendedExecutionMode"].(ExecutionMode)
			if mode != tt.wantMode {
				t.Fatalf("expected recommendedExecutionMode %s, got %v", tt.wantMode, plan.Metadata["recommendedExecutionMode"])
			}
			reason, _ := plan.Metadata["recommendedExecutionModeReason"].(string)
			if reason == "" || !containsSubstring(reason, tt.wantReasonPart) {
				t.Fatalf("expected recommendation reason containing %q, got %q", tt.wantReasonPart, reason)
			}
			executionOrder, _ := plan.Metadata["executionOrder"].(string)
			if executionOrder != tt.wantExecution {
				t.Fatalf("expected executionOrder %s, got %q", tt.wantExecution, executionOrder)
			}
		})
	}
}

func TestBuildPreviewDeletedEntryRootMatchingContracts(t *testing.T) {
	registry := provider.NewRegistry(provider.DefaultCatalog()...)
	plan, err := BuildPreview(registry, PreviewRequest{
		SourceProvider:     "guangya",
		TargetProvider:     "123_open",
		SourceDeletePolicy: SourceDeletePolicyRecordOnly,
		SelectedRoots:      []string{"/alpha", "/beta"},
		Entries: []SourceEntry{
			{Path: "/alpha/live.bin", Size: 10, MD5: "md5-live"},
			{Path: "/beta/history/deleted.bin", Deleted: true, DeletedAt: "2026-05-30T10:00:00Z", DeleteReason: "source_removed"},
			{Path: "/orphan/deleted.bin", Deleted: true, DeletedAt: "2026-05-30T11:00:00Z", DeleteReason: "source_removed"},
		},
	})
	if err != nil {
		t.Fatalf("BuildPreview() error = %v", err)
	}
	records, ok := plan.Metadata["sourceDeletionRecords"].([]map[string]interface{})
	if !ok {
		t.Fatalf("expected sourceDeletionRecords metadata, got %#v", plan.Metadata["sourceDeletionRecords"])
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 source deletion records, got %#v", records)
	}
	if got := records[0]["rootPath"]; got != "/beta" {
		t.Fatalf("expected /beta rootPath for nested deleted record, got %#v", got)
	}
	if got := records[1]["rootPath"]; got != "/orphan" {
		t.Fatalf("expected parent directory fallback rootPath /orphan, got %#v", got)
	}
	if got := records[1]["name"]; got != "deleted.bin" {
		t.Fatalf("expected inferred deleted entry name deleted.bin, got %#v", got)
	}
}

func containsSubstring(value string, part string) bool {
	return len(part) == 0 || (len(value) >= len(part) && stringsContains(value, part))
}

func stringsContains(value string, part string) bool {
	for i := 0; i+len(part) <= len(value); i++ {
		if value[i:i+len(part)] == part {
			return true
		}
	}
	return false
}
