package planner

import (
	"errors"
	"sort"
	"strings"

	"cloudpan-sync-go/internal/provider"
)

var ErrTargetProviderNotFound = errors.New("target_provider_not_found")
var ErrInvalidExecutionMode = errors.New("invalid_execution_mode")
var ErrInvalidSourceDeletePolicy = errors.New("invalid_source_delete_policy")

type SourceEntry struct {
	Path         string                 `json:"path"`
	Size         int64                  `json:"size"`
	Deleted      bool                   `json:"deleted,omitempty"`
	DeletedAt    string                 `json:"deletedAt,omitempty"`
	DeleteReason string                 `json:"deleteReason,omitempty"`
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
	SourceProvider     string                  `json:"sourceProvider"`
	TargetProvider     string                  `json:"targetProvider"`
	ThresholdMB        int                     `json:"thresholdMB"`
	RiskMode           RiskMode                `json:"riskMode"`
	RiskOverride       *RiskProfileOverride    `json:"riskOverride,omitempty"`
	ExecutionMode      ExecutionMode           `json:"executionMode"`
	SourceDeletePolicy SourceDeletePolicy      `json:"sourceDeletePolicy"`
	ConflictPolicy     provider.ConflictPolicy `json:"conflictPolicy"`
	SelectedRoots      []string                `json:"selectedRoots"`
	Entries            []SourceEntry           `json:"entries"`
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
	sourceDeletePolicy, err := normalizeSourceDeletePolicy(req.SourceDeletePolicy)
	if err != nil {
		return Plan{}, err
	}
	riskResolution := resolveRiskProfile(target.Meta, req.RiskMode, req.RiskOverride)
	riskProfile := riskResolution.Applied
	recommendedMode, recommendedReason := recommendExecutionMode(req, riskProfile)
	recommendedRiskMode, recommendedRiskReason, aggressiveRiskWarning := recommendRiskMode(target.Meta, req, riskResolution)
	orderedEntries := orderEntriesByMode(req.Entries, executionMode)
	deletedRecords := buildDeletedEntryMetadata(orderedEntries, req.SelectedRoots)

	items := make([]Item, 0, len(orderedEntries))
	summary := map[string]int{
		string(StrategyFastUpload):     0,
		string(StrategyDownloadUpload): 0,
		string(StrategyPendingManual):  0,
	}

	for idx, entry := range orderedEntries {
		if entry.Deleted {
			continue
		}
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
			"activeEntryCount":               len(items),
			"deletedEntryCount":              len(deletedRecords),
			"sourceDeletionRecords":          deletedRecords,
			"sourceDeletePolicy":             sourceDeletePolicy,
			"executionMode":                  executionMode,
			"recommendedExecutionMode":       recommendedMode,
			"recommendedExecutionModeReason": recommendedReason,
			"recommendedRiskMode":            recommendedRiskMode,
			"recommendedRiskModeReason":      recommendedRiskReason,
			"aggressiveRiskWarning":          aggressiveRiskWarning,
			"executionOrder":                 executionOrderForMode(executionMode),
			"riskProfile":                    riskProfile,
			"riskProfileResolution":          riskResolution,
			"riskOverride":                   req.RiskOverride,
		},
	}, nil
}

func buildDeletedEntryMetadata(entries []SourceEntry, selectedRoots []string) []map[string]interface{} {
	items := make([]map[string]interface{}, 0)
	for _, entry := range entries {
		if !entry.Deleted {
			continue
		}
		path := normalizePlannerPath(entry.Path)
		items = append(items, map[string]interface{}{
			"path":         path,
			"name":         inferEntryName(path),
			"rootPath":     plannerMatchRootPath(path, selectedRoots),
			"deletedAt":    strings.TrimSpace(entry.DeletedAt),
			"deleteReason": strings.TrimSpace(entry.DeleteReason),
		})
	}
	return items
}

func plannerMatchRootPath(path string, roots []string) string {
	normalized := normalizePlannerPath(path)
	for _, root := range roots {
		root = normalizePlannerPath(root)
		if normalized == root || strings.HasPrefix(normalized, root+"/") {
			return root
		}
	}
	if len(roots) == 1 {
		return normalizePlannerPath(roots[0])
	}
	return plannerParentDirectory(normalized)
}

func plannerParentDirectory(path string) string {
	normalized := normalizePlannerPath(path)
	if normalized == "/" {
		return "/"
	}
	index := strings.LastIndex(normalized, "/")
	if index <= 0 {
		return "/"
	}
	return normalized[:index]
}

func normalizePlannerPath(path string) string {
	path = strings.ReplaceAll(strings.TrimSpace(path), "\\", "/")
	if path == "" {
		return "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	path = strings.TrimRight(path, "/")
	if path == "" {
		return "/"
	}
	return path
}

func inferEntryName(path string) string {
	normalized := normalizePlannerPath(path)
	if normalized == "/" {
		return "/"
	}
	index := strings.LastIndex(normalized, "/")
	if index >= 0 && index < len(normalized)-1 {
		return normalized[index+1:]
	}
	return normalized
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
	normalizedMode := normalizeRiskMode(mode)
	profile := baseRiskProfile(normalizedMode, providerKey)
	return applyProviderRiskCalibration(providerKey, profile)
}

func ProviderDefaultRiskTemplate(meta provider.Provider) provider.RiskTemplateSummary {
	resolution := resolveRiskProfile(meta, RiskModeBalanced, nil)
	defaults := DescribeProviderRiskDefaults(meta)
	return provider.RiskTemplateSummary{
		RecommendedMode:       string(defaults.RecommendedRiskMode),
		Base:                  resolution.Base,
		Calibrated:            resolution.Calibrated,
		RecoverBudget:         resolution.RecoverBudget,
		CalibrationReasons:    append([]string(nil), defaults.CalibrationReasons...),
		ProviderRiskHints:     append([]string(nil), defaults.ProviderRiskHints...),
		ProviderRiskTraits:    append([]string(nil), defaults.ProviderRiskTraits...),
		RecommendedReason:     defaults.RecommendedRiskReason,
		AggressiveRiskWarning: defaults.AggressiveRiskWarning,
	}
}

func DescribeProviderRiskDefaults(meta provider.Provider) ProviderRiskDefaults {
	resolution := resolveRiskProfile(meta, RiskModeBalanced, nil)
	recommended := RiskModeBalanced
	reason := "Balanced is the default provider template for steady transfer pacing."
	if isRiskSensitiveProvider(meta.Key) {
		recommended = RiskModeSafe
		reason = "This provider is more risk-sensitive, so large or unknown workloads should start from safe pacing."
	}
	if len(meta.RiskHints) > 0 {
		reason = reason + " Provider hint: " + meta.RiskHints[0]
	}
	_, _, warning := recommendRiskMode(meta, PreviewRequest{RiskMode: RiskModeFast, SelectedRoots: []string{"/unknown"}}, resolution)
	return ProviderRiskDefaults{
		ProviderKey:           meta.Key,
		ProviderDisplayName:   meta.DisplayName,
		DefaultMode:           RiskModeBalanced,
		Profile:               resolution.Calibrated,
		RecoverBudget:         resolution.RecoverBudget,
		CalibrationReasons:    append([]string(nil), resolution.CalibrationReasons...),
		ProviderRiskHints:     append([]string(nil), meta.RiskHints...),
		ProviderRiskTraits:    append([]string(nil), meta.RiskTraits...),
		RecommendedRiskMode:   recommended,
		RecommendedRiskReason: reason,
		AggressiveRiskWarning: warning,
	}
}

func resolveRiskProfile(meta provider.Provider, mode RiskMode, override *RiskProfileOverride) RiskProfileResolution {
	normalizedMode := normalizeRiskMode(mode)
	base := baseRiskProfile(normalizedMode, meta.Key)
	calibrated, calibrationReasons := applyProviderRiskCalibrationWithReasons(meta.Key, base)
	applied, overrideFields := applyRiskProfileOverrideWithFields(calibrated, override)
	// 根据最终的 RiskProfile 计算恢复预算策略，统一由 Planner 提供。
	recoverBudget := deriveRecoverBudgetPolicy(meta.Key, applied)
	return RiskProfileResolution{
		ProviderKey:         meta.Key,
		ProviderDisplayName: meta.DisplayName,
		ProviderRiskHints:   append([]string(nil), meta.RiskHints...),
		ProviderRiskTraits:  append([]string(nil), meta.RiskTraits...),
		Mode:                normalizedMode,
		Base:                base,
		Calibrated:          calibrated,
		Applied:             applied,
		RecoverBudget:       recoverBudget,
		CalibrationReasons:  calibrationReasons,
		Override:            override,
		OverrideFields:      overrideFields,
	}
}

func baseRiskProfile(mode RiskMode, providerKey string) RiskProfile {
	keywords := providerRiskKeywords(providerKey)
	switch normalizeRiskMode(mode) {
	case RiskModeSafe:
		return RiskProfile{
			Mode:                RiskModeSafe,
			RequestIntervalMS:   1500,
			PageSize:            100,
			DirectoryIntervalMS: 2500,
			CooldownSeconds:     30,
			RetryLimit:          2,
			MaxConcurrent:       1,
			RiskKeywords:        keywords,
		}
	case RiskModeFast:
		return RiskProfile{
			Mode:                RiskModeFast,
			RequestIntervalMS:   250,
			PageSize:            1000,
			DirectoryIntervalMS: 300,
			CooldownSeconds:     5,
			RetryLimit:          5,
			MaxConcurrent:       4,
			RiskKeywords:        keywords,
		}
	case RiskModeCustom:
		return RiskProfile{
			Mode:                RiskModeCustom,
			RequestIntervalMS:   0,
			PageSize:            0,
			DirectoryIntervalMS: 0,
			CooldownSeconds:     0,
			RetryLimit:          0,
			MaxConcurrent:       0,
			RiskKeywords:        keywords,
		}
	default:
		return RiskProfile{
			Mode:                RiskModeBalanced,
			RequestIntervalMS:   800,
			PageSize:            300,
			DirectoryIntervalMS: 1000,
			CooldownSeconds:     15,
			RetryLimit:          3,
			MaxConcurrent:       2,
			RiskKeywords:        keywords,
		}
	}
}

func applyProviderRiskCalibration(providerKey string, profile RiskProfile) RiskProfile {
	profile, _ = applyProviderRiskCalibrationWithReasons(providerKey, profile)
	return profile
}

func applyProviderRiskCalibrationWithReasons(providerKey string, profile RiskProfile) (RiskProfile, []string) {
	if profile.Mode == RiskModeCustom {
		return profile, nil
	}
	reasons := make([]string, 0, 3)
	switch providerKey {
	case "baidu_netdisk":
		profile.RequestIntervalMS = max(profile.RequestIntervalMS, 1800)
		profile.PageSize = minPositive(profile.PageSize, 100)
		profile.DirectoryIntervalMS = max(profile.DirectoryIntervalMS, 3000)
		profile.CooldownSeconds = max(profile.CooldownSeconds, 45)
		profile.RetryLimit = minPositive(profile.RetryLimit, 2)
		profile.MaxConcurrent = minPositive(profile.MaxConcurrent, 1)
		reasons = append(reasons, "百度网盘按更保守的请求/目录间隔收敛，并降低重试上限。")
	case "quark", "uc":
		profile.RequestIntervalMS = max(profile.RequestIntervalMS, 1400)
		profile.PageSize = minPositive(profile.PageSize, 120)
		profile.DirectoryIntervalMS = max(profile.DirectoryIntervalMS, 2200)
		profile.CooldownSeconds = max(profile.CooldownSeconds, 40)
		profile.MaxConcurrent = minPositive(profile.MaxConcurrent, 1)
		reasons = append(reasons, "Quark / UC 风控更敏感，默认提高请求间隔并缩小分页。")
	case "189cloud":
		profile.RequestIntervalMS = max(profile.RequestIntervalMS, 1200)
		profile.PageSize = minPositive(profile.PageSize, 150)
		profile.DirectoryIntervalMS = max(profile.DirectoryIntervalMS, 2000)
		profile.CooldownSeconds = max(profile.CooldownSeconds, 35)
		profile.MaxConcurrent = minPositive(profile.MaxConcurrent, 1)
		reasons = append(reasons, "天翼云盘默认保守控制分页和目录推进节奏。")
	case "115_open":
		profile.RequestIntervalMS = max(profile.RequestIntervalMS, 1000)
		profile.PageSize = minPositive(profile.PageSize, 200)
		profile.DirectoryIntervalMS = max(profile.DirectoryIntervalMS, 1800)
		profile.CooldownSeconds = max(profile.CooldownSeconds, 30)
		profile.MaxConcurrent = minPositive(profile.MaxConcurrent, 1)
		reasons = append(reasons, "115 默认保守控制列表频率与目录节流。")
	case "guangya":
		profile.RequestIntervalMS = max(profile.RequestIntervalMS, 900)
		profile.PageSize = minPositive(profile.PageSize, 180)
		profile.DirectoryIntervalMS = max(profile.DirectoryIntervalMS, 1600)
		profile.CooldownSeconds = max(profile.CooldownSeconds, 25)
		profile.MaxConcurrent = minPositive(profile.MaxConcurrent, 1)
		reasons = append(reasons, "光鸭链路默认按中保守模板限制目录节奏。")
	case "xunlei", "pikpak":
		profile.RequestIntervalMS = max(profile.RequestIntervalMS, 700)
		profile.PageSize = minPositive(profile.PageSize, 250)
		profile.DirectoryIntervalMS = max(profile.DirectoryIntervalMS, 1000)
		profile.MaxConcurrent = minPositive(profile.MaxConcurrent, 2)
		reasons = append(reasons, "迅雷 / PikPak 保留较快节奏，但仍限制分页和目录切换频率。")
	case "aliyundrive_open", "123_open":
		profile.PageSize = minPositive(profile.PageSize, 500)
		profile.MaxConcurrent = minPositive(profile.MaxConcurrent, 3)
		reasons = append(reasons, "阿里云盘 / 123Open 允许更大的分页预算，适合较平滑的列表推进。")
	}
	return profile, reasons
}

func minPositive(current int, limit int) int {
	if current <= 0 {
		return current
	}
	if limit <= 0 || current < limit {
		return current
	}
	return limit
}

func deriveRecoverBudgetPolicy(providerKey string, profile RiskProfile) RecoverBudgetPolicy {
	policy := RecoverBudgetPolicy{}
	if profile.MaxConcurrent <= 0 {
		return policy
	}
	policy.ProviderBudget = profile.MaxConcurrent
	if profile.MaxConcurrent <= 2 {
		policy.ProtocolGroupBudget = 1
	} else {
		policy.ProtocolGroupBudget = minPositive(profile.MaxConcurrent, 2)
	}
	sensitive := isSensitiveRecoverBudgetProvider(providerKey)
	if sensitive {
		policy.ProfileBudget = 1
		policy.SensitiveProviders = []string{providerKey}
		policy.Reason = "Sensitive provider defaults to single-profile recover budget."
		return policy
	}
	if profile.MaxConcurrent <= 2 {
		policy.ProfileBudget = 1
		policy.Reason = "Low concurrency risk profile narrows recover budget to one profile per round."
		return policy
	}
	policy.ProfileBudget = minPositive(profile.MaxConcurrent, 2)
	policy.Reason = "Recover budgets inherit maxConcurrent and keep profile fairness within each provider."
	return policy
}

func isSensitiveRecoverBudgetProvider(providerKey string) bool {
	switch providerKey {
	case "baidu_netdisk", "quark", "uc", "189cloud", "115_open", "guangya":
		return true
	default:
		return false
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

func applyRiskProfileOverride(base RiskProfile, override *RiskProfileOverride) RiskProfile {
	base, _ = applyRiskProfileOverrideWithFields(base, override)
	return base
}

func applyRiskProfileOverrideWithFields(base RiskProfile, override *RiskProfileOverride) (RiskProfile, []string) {
	if override == nil {
		return base, nil
	}
	fields := make([]string, 0, 6)
	if override.RequestIntervalMS != nil && *override.RequestIntervalMS >= 0 {
		base.RequestIntervalMS = *override.RequestIntervalMS
		fields = append(fields, "requestIntervalMs")
	}
	if override.PageSize != nil && *override.PageSize >= 0 {
		base.PageSize = *override.PageSize
		fields = append(fields, "pageSize")
	}
	if override.DirectoryIntervalMS != nil && *override.DirectoryIntervalMS >= 0 {
		base.DirectoryIntervalMS = *override.DirectoryIntervalMS
		fields = append(fields, "directoryIntervalMs")
	}
	if override.CooldownSeconds != nil && *override.CooldownSeconds >= 0 {
		base.CooldownSeconds = *override.CooldownSeconds
		fields = append(fields, "cooldownSeconds")
	}
	if override.RetryLimit != nil && *override.RetryLimit >= 0 {
		base.RetryLimit = *override.RetryLimit
		fields = append(fields, "retryLimit")
	}
	if override.MaxConcurrent != nil && *override.MaxConcurrent >= 0 {
		base.MaxConcurrent = *override.MaxConcurrent
		fields = append(fields, "maxConcurrent")
	}
	if override.AutoRetryStartHour != nil && *override.AutoRetryStartHour >= 0 {
		base.AutoRetryStartHour = *override.AutoRetryStartHour
		fields = append(fields, "autoRetryStartHour")
	}
	if override.AutoRetryEndHour != nil && *override.AutoRetryEndHour >= 0 {
		base.AutoRetryEndHour = *override.AutoRetryEndHour
		fields = append(fields, "autoRetryEndHour")
	}
	if len(override.RiskKeywords) > 0 {
		base.RiskKeywords = append([]string(nil), override.RiskKeywords...)
		fields = append(fields, "riskKeywords")
	}
	base.AutoRetryStartHour = clamp(base.AutoRetryStartHour, 0, 23)
	base.AutoRetryEndHour = clamp(base.AutoRetryEndHour, 0, 24)
	return base, fields
}

func clamp(value, minValue, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
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

func normalizeSourceDeletePolicy(policy SourceDeletePolicy) (SourceDeletePolicy, error) {
	switch policy {
	case "", SourceDeletePolicyRecordOnly:
		return SourceDeletePolicyRecordOnly, nil
	default:
		return "", ErrInvalidSourceDeletePolicy
	}
}

func recommendExecutionMode(req PreviewRequest, riskProfile RiskProfile) (ExecutionMode, string) {
	if len(req.SelectedRoots) > 1 {
		return ExecutionModeLeafFirstLazy, "Multiple top-level roots are safer to process subtree by subtree."
	}
	if len(req.Entries) > 0 && len(req.Entries) <= 20 && len(req.SelectedRoots) <= 1 && riskProfile.Mode == RiskModeFast {
		return ExecutionModePreScanFlat, "Known small input set with aggressive risk mode can finish analysis up front."
	}
	if len(req.Entries) > 0 && len(req.Entries) <= 20 && len(req.SelectedRoots) <= 1 {
		return ExecutionModePreScanFlat, "Known small input set is suitable for up-front scan and simpler progress visibility."
	}
	if len(req.Entries) == 0 {
		return ExecutionModeLeafFirstLazy, "Unknown full tree size should default to on-demand leaf-first scanning."
	}
	return ExecutionModeLeafFirstLazy, "Leaf-first lazy scan is the preferred default for large or risk-sensitive transfers."
}

func recommendRiskMode(meta provider.Provider, req PreviewRequest, resolution RiskProfileResolution) (RiskMode, string, string) {
	selectedMode := normalizeRiskMode(req.RiskMode)
	entryCount := len(req.Entries)
	rootCount := len(req.SelectedRoots)
	recommended := RiskModeBalanced
	reason := "Balanced keeps throughput and provider risk under control for the default transfer path."
	warning := ""

	switch {
	case entryCount == 0:
		recommended = RiskModeSafe
		reason = "Unknown full tree size should start from a safer throttle profile until runtime evidence becomes clearer."
	case rootCount > 1:
		recommended = RiskModeBalanced
		reason = "Multiple top-level roots benefit from balanced pacing to keep subtree progression and recoverability stable."
	case isRiskSensitiveProvider(meta.Key):
		recommended = RiskModeSafe
		reason = "This provider family is more risk-sensitive and should default to safe pacing before raising throughput."
	case entryCount <= 20 && rootCount <= 1:
		recommended = RiskModeFast
		reason = "Known small input set can use a faster profile to finish validation and transfer with fewer long waits."
	}

	if selectedMode == RiskModeFast {
		warning = "Fast mode may increase rate-limit, captcha, or provider risk-control hits for this provider or workload."
	}
	if selectedMode == RiskModeCustom {
		warning = "Custom mode bypasses the default recommended throttle profile. Validate request pacing and retry budgets carefully."
	}
	if selectedMode == RiskModeFast && recommended != RiskModeFast {
		warning = "Fast mode may increase rate-limit, captcha, or provider risk-control hits for this provider or workload."
	}
	if warning == "" && selectedMode != recommended && selectedMode != "" {
		warning = "Current risk mode differs from the recommended profile. Review request interval, retry limit, and concurrency before large runs."
	}
	if len(meta.RiskHints) > 0 {
		reason = reason + " Provider hint: " + meta.RiskHints[0]
	}
	if warning == "" && len(meta.RiskTraits) > 0 && containsPlannerString(meta.RiskTraits, "manual_confirmation_possible") {
		warning = "This provider may still require manual confirmation on some fallback branches."
	}
	_ = resolution
	return recommended, reason, warning
}

func containsPlannerString(values []string, target string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == target {
			return true
		}
	}
	return false
}

func isRiskSensitiveProvider(providerKey string) bool {
	switch strings.TrimSpace(providerKey) {
	case "baidu_netdisk", "quark", "uc", "189cloud", "115_open", "guangya":
		return true
	default:
		return false
	}
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
