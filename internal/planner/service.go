package planner

import (
	"errors"
	"strings"

	"cloudpan-sync-go/internal/provider"
)

var ErrTargetProviderNotFound = errors.New("target_provider_not_found")

type SourceEntry struct {
	Path         string                 `json:"path"`
	Size         int64                  `json:"size"`
	MD5          string                 `json:"md5,omitempty"`
	SHA1         string                 `json:"sha1,omitempty"`
	SHA256       string                 `json:"sha256,omitempty"`
	CRC64        string                 `json:"crc64,omitempty"`
	GCID         string                 `json:"gcid,omitempty"`
	ETag         string                 `json:"etag,omitempty"`
	PickCode     string                 `json:"pickcode,omitempty"`
	BlockListMD5 []string               `json:"blockListMd5,omitempty"`
	LocalPath    string                 `json:"localPath,omitempty"`
	Raw          map[string]interface{} `json:"raw,omitempty"`
}

type PreviewRequest struct {
	SourceProvider string                  `json:"sourceProvider"`
	TargetProvider string                  `json:"targetProvider"`
	ThresholdMB    int                     `json:"thresholdMB"`
	ConflictPolicy provider.ConflictPolicy `json:"conflictPolicy"`
	SelectedRoots  []string                `json:"selectedRoots"`
	Entries        []SourceEntry           `json:"entries"`
}

func BuildPreview(registry *provider.Registry, req PreviewRequest) (Plan, error) {
	target, ok := registry.Get(req.TargetProvider)
	if !ok {
		return Plan{}, ErrTargetProviderNotFound
	}

	thresholdBytes := int64(max(req.ThresholdMB, 0)) * 1024 * 1024
	conflictPolicy := string(req.ConflictPolicy)
	if conflictPolicy == "" {
		conflictPolicy = string(provider.ConflictPolicyAutoRenameNew)
	}

	items := make([]Item, 0, len(req.Entries))
	summary := map[string]int{
		string(StrategyFastUpload):     0,
		string(StrategyDownloadUpload): 0,
		string(StrategyPendingManual):  0,
	}

	for _, entry := range req.Entries {
		strategy := decideStrategy(target.Meta.FastUploadInputs, entry, thresholdBytes)
		items = append(items, Item{
			Path:           entry.Path,
			Size:           entry.Size,
			Strategy:       strategy,
			ConflictPolicy: conflictPolicy,
		})
		summary[string(strategy)]++
	}

	return Plan{
		SourceProvider: req.SourceProvider,
		TargetProvider: req.TargetProvider,
		ThresholdMB:    max(req.ThresholdMB, 0),
		Items:          items,
		Summary:        summary,
		Metadata: map[string]interface{}{
			"selectedRoots": req.SelectedRoots,
			"entryCount":    len(req.Entries),
		},
	}, nil
}

func decideStrategy(required []string, entry SourceEntry, thresholdBytes int64) Strategy {
	if hasAllFastInputs(required, entry) {
		return StrategyFastUpload
	}
	if thresholdBytes > 0 && entry.Size <= thresholdBytes {
		return StrategyDownloadUpload
	}
	return StrategyPendingManual
}

func hasAllFastInputs(required []string, entry SourceEntry) bool {
	for _, item := range required {
		switch strings.ToLower(item) {
		case "md5":
			if strings.TrimSpace(entry.MD5) == "" && strings.TrimSpace(entry.ETag) == "" {
				return false
			}
		case "sha1":
			if strings.TrimSpace(entry.SHA1) == "" {
				return false
			}
		case "size":
			if entry.Size <= 0 {
				return false
			}
		case "name":
			if strings.TrimSpace(entry.Path) == "" {
				return false
			}
		case "gcid":
			if strings.TrimSpace(entry.GCID) == "" {
				return false
			}
		}
	}
	return true
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
