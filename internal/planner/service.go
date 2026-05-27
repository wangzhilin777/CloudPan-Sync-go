package planner

import (
	"errors"
	"sort"
	"strings"

	"cloudpan-sync-go/internal/provider"
)

var ErrTargetProviderNotFound = errors.New("target_provider_not_found")
var ErrInvalidExecutionMode = errors.New("invalid_execution_mode")

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
	RiskMode       RiskMode                `json:"riskMode"`
	ExecutionMode  ExecutionMode           `json:"executionMode"`
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
	executionMode, err := normalizeExecutionMode(req.ExecutionMode)
	if err != nil {
		return Plan{}, err
	}
	riskProfile := defaultRiskProfile(target.Meta.Key, req.RiskMode)
	recommendedMode, recommendedReason := recommendExecutionMode(req, riskProfile)
	orderedEntries := orderEntriesByMode(req.Entries, executionMode)

	items := make([]Item, 0, len(orderedEntries))
	summary := map[string]int{
		string(StrategyFastUpload):     0,
		string(StrategyDownloadUpload): 0,
		string(StrategyPendingManual):  0,
	}

	for idx, entry := range orderedEntries {
		strategy := decideStrategy(target.Meta.FastUploadInputs, entry, thresholdBytes)
		items = append(items, Item{
			Path:           entry.Path,
			Size:           entry.Size,
			Sequence:       idx + 1,
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
			"selectedRoots":                  req.SelectedRoots,
			"entryCount":                     len(req.Entries),
			"executionMode":                  executionMode,
			"recommendedExecutionMode":       recommendedMode,
			"recommendedExecutionModeReason": recommendedReason,
			"executionOrder":                 executionOrderForMode(executionMode),
			"riskProfile":                    riskProfile,
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

func orderEntriesLeafFirst(entries []SourceEntry) []SourceEntry {
	ordered := append([]SourceEntry(nil), entries...)
	sort.SliceStable(ordered, func(i, j int) bool {
		leftDepth := pathDepth(ordered[i].Path)
		rightDepth := pathDepth(ordered[j].Path)
		if leftDepth != rightDepth {
			return leftDepth > rightDepth
		}
		return ordered[i].Path < ordered[j].Path
	})
	return ordered
}

func orderEntriesByMode(entries []SourceEntry, mode ExecutionMode) []SourceEntry {
	switch mode {
	case ExecutionModePreScanFlat:
		return append([]SourceEntry(nil), entries...)
	default:
		return orderEntriesLeafFirst(entries)
	}
}

func executionOrderForMode(mode ExecutionMode) string {
	switch mode {
	case ExecutionModePreScanFlat:
		return "pre_scan_flat"
	default:
		return "leaf_first"
	}
}

func pathDepth(path string) int {
	trimmed := strings.Trim(path, "/\\ ")
	if trimmed == "" {
		return 0
	}
	depth := 1
	for _, ch := range trimmed {
		if ch == '/' || ch == '\\' {
			depth++
		}
	}
	return depth
}

func defaultRiskProfile(providerKey string, mode RiskMode) RiskProfile {
	switch normalizeRiskMode(mode) {
	case RiskModeSafe:
		return RiskProfile{
			Mode:                RiskModeSafe,
			RequestIntervalMS:   1500,
			PageSize:            100,
			DirectoryIntervalMS: 2500,
			CooldownSeconds:     30,
			RetryLimit:          2,
			RiskKeywords:        providerRiskKeywords(providerKey),
		}
	case RiskModeFast:
		return RiskProfile{
			Mode:                RiskModeFast,
			RequestIntervalMS:   250,
			PageSize:            1000,
			DirectoryIntervalMS: 300,
			CooldownSeconds:     5,
			RetryLimit:          5,
			RiskKeywords:        providerRiskKeywords(providerKey),
		}
	case RiskModeCustom:
		return RiskProfile{
			Mode:                RiskModeCustom,
			RequestIntervalMS:   0,
			PageSize:            0,
			DirectoryIntervalMS: 0,
			CooldownSeconds:     0,
			RetryLimit:          0,
			RiskKeywords:        providerRiskKeywords(providerKey),
		}
	default:
		return RiskProfile{
			Mode:                RiskModeBalanced,
			RequestIntervalMS:   800,
			PageSize:            300,
			DirectoryIntervalMS: 1000,
			CooldownSeconds:     15,
			RetryLimit:          3,
			RiskKeywords:        providerRiskKeywords(providerKey),
		}
	}
}

func normalizeRiskMode(mode RiskMode) RiskMode {
	switch mode {
	case RiskModeSafe, RiskModeFast, RiskModeCustom:
		return mode
	default:
		return RiskModeBalanced
	}
}

func normalizeExecutionMode(mode ExecutionMode) (ExecutionMode, error) {
	switch mode {
	case "", ExecutionModeLeafFirstLazy:
		return ExecutionModeLeafFirstLazy, nil
	case ExecutionModePreScanFlat:
		return ExecutionModePreScanFlat, nil
	default:
		return "", ErrInvalidExecutionMode
	}
}

func recommendExecutionMode(req PreviewRequest, riskProfile RiskProfile) (ExecutionMode, string) {
	if len(req.Entries) > 0 && len(req.Entries) <= 20 && len(req.SelectedRoots) <= 1 && riskProfile.Mode == RiskModeFast {
		return ExecutionModePreScanFlat, "Known small input set with aggressive risk mode can finish analysis up front."
	}
	if len(req.Entries) > 0 && len(req.Entries) <= 20 && len(req.SelectedRoots) <= 1 {
		return ExecutionModePreScanFlat, "Known small input set is suitable for up-front scan and simpler progress visibility."
	}
	if len(req.SelectedRoots) > 1 {
		return ExecutionModeLeafFirstLazy, "Multiple top-level roots are safer to process subtree by subtree."
	}
	if len(req.Entries) == 0 {
		return ExecutionModeLeafFirstLazy, "Unknown full tree size should default to on-demand leaf-first scanning."
	}
	return ExecutionModeLeafFirstLazy, "Leaf-first lazy scan is the preferred default for large or risk-sensitive transfers."
}

func providerRiskKeywords(providerKey string) []string {
	switch providerKey {
	case "aliyundrive_open", "123_open":
		return []string{"429", "too_many_requests", "flow_limit"}
	case "quark", "uc":
		return []string{"risk_control", "captcha", "forbidden"}
	case "xunlei", "pikpak":
		return []string{"frequency_limit", "risk_detected", "forbidden"}
	case "189cloud":
		return []string{"rate_limit", "token_expired", "too_many_requests"}
	case "baidu_netdisk":
		return []string{"hit_risk_control", "captcha", "too_many_requests"}
	default:
		return []string{"rate_limit", "too_many_requests"}
	}
}
