package planner

import "testing"

func TestHasAllFastInputsContracts(t *testing.T) {
	tests := []struct {
		name     string
		required []string
		entry    SourceEntry
		want     bool
	}{
		{
			name:     "md5 requirement accepts etag fallback",
			required: []string{"md5", "size"},
			entry:    SourceEntry{Path: "/demo/a.bin", Size: 12, ETag: "etag-a"},
			want:     true,
		},
		{
			name:     "gcid family requires gcid and size",
			required: []string{"gcid", "size"},
			entry:    SourceEntry{Path: "/demo/a.bin", Size: 12, GCID: "gcid-a"},
			want:     true,
		},
		{
			name:     "name requirement rejects empty path",
			required: []string{"name"},
			entry:    SourceEntry{Path: "", Size: 12, MD5: "md5-a"},
			want:     false,
		},
		{
			name:     "sha1 family rejects missing sha1",
			required: []string{"sha1", "size"},
			entry:    SourceEntry{Path: "/demo/a.bin", Size: 12},
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasAllFastInputs(tt.required, tt.entry); got != tt.want {
				t.Fatalf("hasAllFastInputs(%v, %+v) = %v, want %v", tt.required, tt.entry, got, tt.want)
			}
		})
	}
}

func TestDecideStrategyContracts(t *testing.T) {
	tests := []struct {
		name           string
		required       []string
		entry          SourceEntry
		thresholdBytes int64
		want           Strategy
	}{
		{
			name:           "etag still unlocks fast upload for md5 provider",
			required:       []string{"md5", "size"},
			entry:          SourceEntry{Path: "/demo/a.bin", Size: 128, ETag: "etag-a"},
			thresholdBytes: 1024,
			want:           StrategyFastUpload,
		},
		{
			name:           "small file falls back to download upload when fast inputs are incomplete",
			required:       []string{"sha1", "size"},
			entry:          SourceEntry{Path: "/demo/a.bin", Size: 128},
			thresholdBytes: 1024,
			want:           StrategyDownloadUpload,
		},
		{
			name:           "large file becomes pending manual without fast inputs",
			required:       []string{"gcid", "size"},
			entry:          SourceEntry{Path: "/demo/a.bin", Size: 8 * 1024 * 1024},
			thresholdBytes: 1024,
			want:           StrategyPendingManual,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := decideStrategy(tt.required, tt.entry, tt.thresholdBytes); got != tt.want {
				t.Fatalf("decideStrategy(%v, %+v, %d) = %s, want %s", tt.required, tt.entry, tt.thresholdBytes, got, tt.want)
			}
		})
	}
}
