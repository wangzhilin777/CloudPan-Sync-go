package planner

import (
	"errors"
	"strings"
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
	if len(resolution.ProviderRiskHints) == 0 {
		t.Fatalf("expected provider risk hints in resolution, got %+v", resolution)
	}
	if len(resolution.ProviderRiskTraits) == 0 {
		t.Fatalf("expected provider risk traits in resolution, got %+v", resolution)
	}
	if resolution.ProtocolGroup != "189cloud" {
		t.Fatalf("expected protocol group 189cloud, got %+v", resolution)
	}
	if resolution.ProviderKey != "189cloud" || resolution.Mode != RiskModeSafe {
		t.Fatalf("unexpected risk profile resolution: %+v", resolution)
	}
	if resolution.Applied.RequestIntervalMS != riskProfile.RequestIntervalMS {
		t.Fatalf("expected applied risk profile to match riskProfile metadata, got %+v vs %+v", resolution.Applied, riskProfile)
	}
	if resolution.RecoverBudget.ProviderBudget != riskProfile.MaxConcurrent {
		t.Fatalf("expected provider recover budget to track maxConcurrent, got %+v vs %+v", resolution.RecoverBudget, riskProfile)
	}
	if resolution.RecoverBudget.ProtocolGroupBudget != 1 {
		t.Fatalf("expected protocol group recover budget 1 for conservative profile, got %+v", resolution.RecoverBudget)
	}
	if resolution.RecoverBudget.ProfileBudget != 1 {
		t.Fatalf("expected profile recover budget 1 for sensitive provider, got %+v", resolution.RecoverBudget)
	}
	if !containsString(resolution.RecoverBudget.SensitiveProviders, "189cloud") {
		t.Fatalf("expected sensitive providers to include 189cloud, got %#v", resolution.RecoverBudget.SensitiveProviders)
	}
}

func TestBuildPreviewDerivesRecoverBudgetPolicy(t *testing.T) {
	registry := provider.NewRegistry(provider.DefaultCatalog()...)
	plan, err := BuildPreview(registry, PreviewRequest{
		SourceProvider: "guangya",
		TargetProvider: "aliyundrive_open",
		RiskMode:       RiskModeFast,
		RiskOverride: &RiskProfileOverride{
			MaxConcurrent: intPtr(4),
		},
		Entries: []SourceEntry{{Path: "/a.bin", Size: 10, SHA1: "sha1-a"}},
	})
	if err != nil {
		t.Fatalf("BuildPreview() error = %v", err)
	}
	resolution, ok := plan.Metadata["riskProfileResolution"].(RiskProfileResolution)
	if !ok {
		t.Fatalf("expected riskProfileResolution metadata, got %#v", plan.Metadata["riskProfileResolution"])
	}
	if resolution.RecoverBudget.ProviderBudget != 4 {
		t.Fatalf("expected provider budget 4, got %+v", resolution.RecoverBudget)
	}
	if resolution.RecoverBudget.ProtocolGroupBudget != 2 {
		t.Fatalf("expected protocol group budget 2, got %+v", resolution.RecoverBudget)
	}
	if resolution.RecoverBudget.ProfileBudget != 2 {
		t.Fatalf("expected profile budget 2 for non-sensitive provider, got %+v", resolution.RecoverBudget)
	}
	if len(resolution.RecoverBudget.SensitiveProviders) != 0 {
		t.Fatalf("expected no sensitive providers, got %#v", resolution.RecoverBudget.SensitiveProviders)
	}
	if resolution.RecoverBudget.Reason == "" {
		t.Fatalf("expected recover budget reason, got %+v", resolution.RecoverBudget)
	}
}

func TestBuildPreviewCapturesProtocolGroupRiskCalibration(t *testing.T) {
	registry := provider.NewRegistry(provider.DefaultCatalog()...)
	plan, err := BuildPreview(registry, PreviewRequest{
		SourceProvider: "guangya",
		TargetProvider: "quark",
		RiskMode:       RiskModeBalanced,
		Entries:        []SourceEntry{{Path: "/a.bin", Size: 10, MD5: "md5-a"}},
	})
	if err != nil {
		t.Fatalf("BuildPreview() error = %v", err)
	}
	resolution, ok := plan.Metadata["riskProfileResolution"].(RiskProfileResolution)
	if !ok {
		t.Fatalf("expected riskProfileResolution metadata, got %#v", plan.Metadata["riskProfileResolution"])
	}
	if resolution.ProtocolGroup != "quark_uc" {
		t.Fatalf("expected protocolGroup quark_uc, got %+v", resolution)
	}
	if len(resolution.ProtocolGroupReasons) == 0 {
		t.Fatalf("expected protocolGroupReasons, got %+v", resolution)
	}
	if !containsString(resolution.ProtocolGroupReasons, "Quark / UC 协议族默认采用更保守的列表节奏、分页和并发预算。") {
		t.Fatalf("expected protocol group calibration reason, got %#v", resolution.ProtocolGroupReasons)
	}
	if len(resolution.CalibrationReasons) < len(resolution.ProtocolGroupReasons) {
		t.Fatalf("expected calibrationReasons to include protocol group reasons, got %#v", resolution.CalibrationReasons)
	}
}

func TestDescribeProviderRiskDefaultsIncludesProtocolGroupCalibration(t *testing.T) {
	registry := provider.NewRegistry(provider.DefaultCatalog()...)
	entry, ok := registry.Get("aliyundrive_open")
	if !ok {
		t.Fatal("expected aliyundrive_open in registry")
	}
	defaults := DescribeProviderRiskDefaults(entry.Meta)
	if defaults.ProtocolGroup != "aliyun_123_open" {
		t.Fatalf("expected protocolGroup aliyun_123_open, got %+v", defaults)
	}
	if len(defaults.ProtocolGroupReasons) == 0 {
		t.Fatalf("expected protocolGroupReasons, got %+v", defaults)
	}
	if defaults.Profile.PageSize > 500 {
		t.Fatalf("expected protocol group page size cap <= 500, got %+v", defaults.Profile)
	}
}

func TestProviderDefaultRiskTemplateIncludesAutoRetryWindowSummary(t *testing.T) {
	registry := provider.NewRegistry(provider.DefaultCatalog()...)
	entry, ok := registry.Get("aliyundrive_open")
	if !ok {
		t.Fatal("expected aliyundrive_open in registry")
	}
	template := ProviderDefaultRiskTemplate(entry.Meta)
	if template.AutoRetryWindowSource != "empty_until_profile_or_override" {
		t.Fatalf("expected auto retry window source empty_until_profile_or_override, got %+v", template)
	}
	if !strings.Contains(template.AutoRetryWindowAdvice, "always_on") {
		t.Fatalf("expected auto retry window advice to mention always_on, got %+v", template)
	}
}

func TestDescribeProviderRiskDefaultsCatalogContract(t *testing.T) {
	registry := provider.NewRegistry(provider.DefaultCatalog()...)
	tests := []struct {
		providerKey        string
		wantProtocolGroup  string
		wantRecommended    RiskMode
		wantRequest        int
		wantPage           int
		wantDirectory      int
		wantCooldown       int
		wantRetryLimit     int
		wantMaxConcurrent  int
		wantKeyword        string
	}{
		{
			providerKey:       "aliyundrive_open",
			wantProtocolGroup: "aliyun_123_open",
			wantRecommended:   RiskModeBalanced,
			wantRequest:       800,
			wantPage:          300,
			wantDirectory:     1000,
			wantCooldown:      15,
			wantRetryLimit:    3,
			wantMaxConcurrent: 2,
			wantKeyword:       "flow_limit",
		},
		{
			providerKey:       "quark",
			wantProtocolGroup: "quark_uc",
			wantRecommended:   RiskModeSafe,
			wantRequest:       1400,
			wantPage:          120,
			wantDirectory:     2200,
			wantCooldown:      40,
			wantRetryLimit:    3,
			wantMaxConcurrent: 1,
			wantKeyword:       "captcha",
		},
		{
			providerKey:       "xunlei",
			wantProtocolGroup: "xunlei_pikpak",
			wantRecommended:   RiskModeBalanced,
			wantRequest:       800,
			wantPage:          250,
			wantDirectory:     1000,
			wantCooldown:      15,
			wantRetryLimit:    3,
			wantMaxConcurrent: 2,
			wantKeyword:       "risk_detected",
		},
		{
			providerKey:       "baidu_netdisk",
			wantProtocolGroup: "baidu_netdisk",
			wantRecommended:   RiskModeSafe,
			wantRequest:       1800,
			wantPage:          100,
			wantDirectory:     3000,
			wantCooldown:      45,
			wantRetryLimit:    2,
			wantMaxConcurrent: 1,
			wantKeyword:       "hit_risk_control",
		},
		{
			providerKey:       "189cloud",
			wantProtocolGroup: "189cloud",
			wantRecommended:   RiskModeSafe,
			wantRequest:       1200,
			wantPage:          150,
			wantDirectory:     2000,
			wantCooldown:      35,
			wantRetryLimit:    3,
			wantMaxConcurrent: 1,
			wantKeyword:       "token_expired",
		},
		{
			providerKey:       "115_open",
			wantProtocolGroup: "115_open",
			wantRecommended:   RiskModeSafe,
			wantRequest:       1000,
			wantPage:          200,
			wantDirectory:     1800,
			wantCooldown:      30,
			wantRetryLimit:    3,
			wantMaxConcurrent: 1,
			wantKeyword:       "rate_limit",
		},
		{
			providerKey:       "guangya",
			wantProtocolGroup: "guangya",
			wantRecommended:   RiskModeSafe,
			wantRequest:       900,
			wantPage:          180,
			wantDirectory:     1600,
			wantCooldown:      25,
			wantRetryLimit:    3,
			wantMaxConcurrent: 1,
			wantKeyword:       "rate_limit",
		},
	}
	for _, tt := range tests {
		t.Run(tt.providerKey, func(t *testing.T) {
			entry, ok := registry.Get(tt.providerKey)
			if !ok {
				t.Fatalf("expected provider %s in registry", tt.providerKey)
			}
			defaults := DescribeProviderRiskDefaults(entry.Meta)
			if defaults.ProtocolGroup != tt.wantProtocolGroup {
				t.Fatalf("expected protocolGroup %s, got %+v", tt.wantProtocolGroup, defaults)
			}
			if defaults.RecommendedRiskMode != tt.wantRecommended {
				t.Fatalf("expected recommended risk mode %s, got %+v", tt.wantRecommended, defaults)
			}
			if defaults.Profile.RequestIntervalMS != tt.wantRequest {
				t.Fatalf("expected requestIntervalMs %d, got %+v", tt.wantRequest, defaults.Profile)
			}
			if defaults.Profile.PageSize != tt.wantPage {
				t.Fatalf("expected pageSize %d, got %+v", tt.wantPage, defaults.Profile)
			}
			if defaults.Profile.DirectoryIntervalMS != tt.wantDirectory {
				t.Fatalf("expected directoryIntervalMs %d, got %+v", tt.wantDirectory, defaults.Profile)
			}
			if defaults.Profile.CooldownSeconds != tt.wantCooldown {
				t.Fatalf("expected cooldownSeconds %d, got %+v", tt.wantCooldown, defaults.Profile)
			}
			if defaults.Profile.RetryLimit != tt.wantRetryLimit {
				t.Fatalf("expected retryLimit %d, got %+v", tt.wantRetryLimit, defaults.Profile)
			}
			if defaults.Profile.MaxConcurrent != tt.wantMaxConcurrent {
				t.Fatalf("expected maxConcurrent %d, got %+v", tt.wantMaxConcurrent, defaults.Profile)
			}
			if defaults.Profile.AutoRetryStartHour != 0 || defaults.Profile.AutoRetryEndHour != 0 {
				t.Fatalf("expected default auto retry window to stay empty, got %+v", defaults.Profile)
			}
			if !containsString(defaults.Profile.RiskKeywords, tt.wantKeyword) {
				t.Fatalf("expected risk keyword %q in %#v", tt.wantKeyword, defaults.Profile.RiskKeywords)
			}
		})
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

func TestBuildPreviewAppliesProfileRiskDefaultsBeforeTaskOverride(t *testing.T) {
	registry := provider.NewRegistry(provider.DefaultCatalog()...)
	plan, err := BuildPreview(registry, PreviewRequest{
		SourceProvider:       "guangya",
		TargetProvider:       "123_open",
		TargetProfileID:      "profile-123",
		RiskMode:             RiskModeBalanced,
		ProfileDefaultSource: "Profile 123",
		ProfileRiskDefaults: &RiskProfileOverride{
			RequestIntervalMS:   intPtr(1666),
			DirectoryIntervalMS: intPtr(2888),
			RetryLimit:          intPtr(4),
		},
		RiskOverride: &RiskProfileOverride{
			RetryLimit: intPtr(1),
		},
		Entries: []SourceEntry{{Path: "/a.bin", Size: 10, MD5: "md5-a"}},
	})
	if err != nil {
		t.Fatalf("BuildPreview() error = %v", err)
	}
	resolution, ok := plan.Metadata["riskProfileResolution"].(RiskProfileResolution)
	if !ok {
		t.Fatalf("expected riskProfileResolution metadata, got %#v", plan.Metadata["riskProfileResolution"])
	}
	if resolution.ProfileDefaultSource != "Profile 123" {
		t.Fatalf("expected profile default source Profile 123, got %+v", resolution)
	}
	if resolution.ProfileDefaultSourceKind != "auth_profile" {
		t.Fatalf("expected auth_profile source kind, got %+v", resolution)
	}
	if resolution.ProfileDefaultBias != "mixed" {
		t.Fatalf("expected mixed profile bias, got %+v", resolution)
	}
	if resolution.ProfileApplied.RequestIntervalMS != 1666 {
		t.Fatalf("expected profileApplied request interval 1666, got %+v", resolution.ProfileApplied)
	}
	if resolution.ProfileApplied.DirectoryIntervalMS != 2888 {
		t.Fatalf("expected profileApplied directory interval 2888, got %+v", resolution.ProfileApplied)
	}
	if resolution.ProfileApplied.RetryLimit != 4 {
		t.Fatalf("expected profileApplied retry limit 4, got %+v", resolution.ProfileApplied)
	}
	if resolution.Applied.RetryLimit != 1 {
		t.Fatalf("expected task override retry limit 1, got %+v", resolution.Applied)
	}
	if len(resolution.ProfileDefaultFields) != 3 {
		t.Fatalf("expected 3 profile default fields, got %#v", resolution.ProfileDefaultFields)
	}
	if len(resolution.OverrideFields) != 1 || resolution.OverrideFields[0] != "retryLimit" {
		t.Fatalf("expected retryLimit override field, got %#v", resolution.OverrideFields)
	}
}

func TestBuildPreviewSmokeMatrixProfileDefaultsTightenRecoverBudgetAndRecommendation(t *testing.T) {
	registry := provider.NewRegistry(provider.DefaultCatalog()...)
	plan, err := BuildPreview(registry, PreviewRequest{
		SourceProvider:       "guangya",
		TargetProvider:       "123_open",
		TargetProfileID:      "profile-smoke",
		RiskMode:             RiskModeFast,
		ProfileDefaultSource: "Smoke Matrix aliyun_123_open (accepted)",
		ProfileRiskDefaults: &RiskProfileOverride{
			RequestIntervalMS:   intPtr(1800),
			DirectoryIntervalMS: intPtr(2600),
			RetryLimit:          intPtr(3),
			MaxConcurrent:       intPtr(1),
		},
		Entries: []SourceEntry{{Path: "/small.bin", Size: 10, MD5: "md5-a"}},
	})
	if err != nil {
		t.Fatalf("BuildPreview() error = %v", err)
	}
	resolution, ok := plan.Metadata["riskProfileResolution"].(RiskProfileResolution)
	if !ok {
		t.Fatalf("expected riskProfileResolution metadata, got %#v", plan.Metadata["riskProfileResolution"])
	}
	if resolution.ProfileDefaultSourceKind != "smoke_matrix" {
		t.Fatalf("expected smoke_matrix source kind, got %+v", resolution)
	}
	if resolution.ProfileDefaultBias != "more_conservative" {
		t.Fatalf("expected more_conservative profile bias, got %+v", resolution)
	}
	if resolution.RecoverBudget.ProtocolGroupBudget != 1 || resolution.RecoverBudget.ProfileBudget != 1 {
		t.Fatalf("expected tightened recover budget for smoke defaults, got %+v", resolution.RecoverBudget)
	}
	if !strings.Contains(resolution.RecoverBudget.Reason, "smoke-matrix") {
		t.Fatalf("expected smoke-matrix recover budget reason, got %+v", resolution.RecoverBudget)
	}
	if got, _ := plan.Metadata["recommendedRiskMode"].(RiskMode); got != RiskModeBalanced {
		t.Fatalf("expected recommendedRiskMode balanced after conservative smoke defaults, got %v", plan.Metadata["recommendedRiskMode"])
	}
	reason, _ := plan.Metadata["recommendedRiskModeReason"].(string)
	if !strings.Contains(reason, "smoke-matrix") {
		t.Fatalf("expected recommendedRiskModeReason to mention smoke-matrix defaults, got %#v", reason)
	}
}

func TestBuildPreviewClassifiesProfileDefaultSourceAndBias(t *testing.T) {
	registry := provider.NewRegistry(provider.DefaultCatalog()...)
	plan, err := BuildPreview(registry, PreviewRequest{
		SourceProvider:       "guangya",
		TargetProvider:       "123_open",
		RiskMode:             RiskModeBalanced,
		ProfileDefaultSource: "Smoke Matrix aliyun_123_open (accepted)",
		ProfileRiskDefaults: &RiskProfileOverride{
			RequestIntervalMS:   intPtr(1800),
			DirectoryIntervalMS: intPtr(2600),
			RetryLimit:          intPtr(3),
			MaxConcurrent:       intPtr(1),
		},
		Entries: []SourceEntry{{Path: "/a.bin", Size: 10, MD5: "md5-a"}},
	})
	if err != nil {
		t.Fatalf("BuildPreview() error = %v", err)
	}
	resolution, ok := plan.Metadata["riskProfileResolution"].(RiskProfileResolution)
	if !ok {
		t.Fatalf("expected riskProfileResolution metadata, got %#v", plan.Metadata["riskProfileResolution"])
	}
	if resolution.ProfileDefaultSourceKind != "smoke_matrix" {
		t.Fatalf("expected profile default source kind smoke_matrix, got %+v", resolution)
	}
	if resolution.ProfileDefaultBias != "more_conservative" {
		t.Fatalf("expected profile default bias more_conservative, got %+v", resolution)
	}
	if resolution.RecoverBudget.ProtocolGroupBudget != 1 || resolution.RecoverBudget.ProfileBudget != 1 {
		t.Fatalf("expected conservative smoke defaults to narrow recover budget, got %+v", resolution.RecoverBudget)
	}
	reason, _ := plan.Metadata["recommendedRiskModeReason"].(string)
	if !strings.Contains(reason, "smoke-matrix") {
		t.Fatalf("expected recommendedRiskModeReason to mention smoke-matrix defaults, got %#v", reason)
	}
}
func TestBuildPreviewPendingSmokeMatrixDefaultsStayProvisional(t *testing.T) {
	registry := provider.NewRegistry(provider.DefaultCatalog()...)
	plan, err := BuildPreview(registry, PreviewRequest{
		SourceProvider:       "guangya",
		TargetProvider:       "123_open",
		RiskMode:             RiskModeFast,
		ProfileDefaultSource: "Smoke Matrix aliyun_123_open (pending)",
		ProfileRiskDefaults: &RiskProfileOverride{
			RequestIntervalMS:   intPtr(1800),
			DirectoryIntervalMS: intPtr(2600),
			RetryLimit:          intPtr(3),
			MaxConcurrent:       intPtr(1),
		},
		Entries: []SourceEntry{{Path: "/small.bin", Size: 10, MD5: "md5-a"}},
	})
	if err != nil {
		t.Fatalf("BuildPreview() error = %v", err)
	}
	resolution, ok := plan.Metadata["riskProfileResolution"].(RiskProfileResolution)
	if !ok {
		t.Fatalf("expected riskProfileResolution metadata, got %#v", plan.Metadata["riskProfileResolution"])
	}
	if resolution.ProfileDefaultSourceKind != "smoke_matrix" {
		t.Fatalf("expected smoke_matrix source kind, got %+v", resolution)
	}
	if resolution.ProfileDefaultBias != "more_conservative" {
		t.Fatalf("expected more_conservative profile bias, got %+v", resolution)
	}
	if resolution.RecoverBudget.ProtocolGroupBudget != 1 || resolution.RecoverBudget.ProfileBudget != 1 {
		t.Fatalf("expected pending smoke defaults to fall back to low-concurrency budget only, got %+v", resolution.RecoverBudget)
	}
	if strings.Contains(resolution.RecoverBudget.Reason, "smoke-matrix") {
		t.Fatalf("expected pending smoke defaults not to use accepted smoke-matrix budget reason, got %+v", resolution.RecoverBudget)
	}
	if got, _ := plan.Metadata["recommendedRiskMode"].(RiskMode); got != RiskModeFast {
		t.Fatalf("expected pending smoke defaults to keep fast recommendation for small inputs, got %v", plan.Metadata["recommendedRiskMode"])
	}
	reason, _ := plan.Metadata["recommendedRiskModeReason"].(string)
	if strings.Contains(reason, "Accepted smoke-matrix") {
		t.Fatalf("expected pending smoke defaults not to mention accepted smoke-matrix evidence, got %#v", reason)
	}
}

func TestBuildPreviewWarnsWhenProfileDefaultsAreMoreAggressive(t *testing.T) {
	registry := provider.NewRegistry(provider.DefaultCatalog()...)
	plan, err := BuildPreview(registry, PreviewRequest{
		SourceProvider:       "guangya",
		TargetProvider:       "123_open",
		RiskMode:             RiskModeBalanced,
		ProfileDefaultSource: "Profile 123",
		ProfileRiskDefaults: &RiskProfileOverride{
			RequestIntervalMS: intPtr(300),
			PageSize:          intPtr(500),
			MaxConcurrent:     intPtr(3),
		},
		Entries: []SourceEntry{{Path: "/a.bin", Size: 10, MD5: "md5-a"}},
	})
	if err != nil {
		t.Fatalf("BuildPreview() error = %v", err)
	}
	resolution, ok := plan.Metadata["riskProfileResolution"].(RiskProfileResolution)
	if !ok {
		t.Fatalf("expected riskProfileResolution metadata, got %#v", plan.Metadata["riskProfileResolution"])
	}
	if resolution.ProfileDefaultSourceKind != "auth_profile" {
		t.Fatalf("expected profile default source kind auth_profile, got %+v", resolution)
	}
	if resolution.ProfileDefaultBias != "more_aggressive" {
		t.Fatalf("expected profile default bias more_aggressive, got %+v", resolution)
	}
	warning, _ := plan.Metadata["aggressiveRiskWarning"].(string)
	if !strings.Contains(warning, "Account profile defaults are more aggressive") {
		t.Fatalf("expected aggressive warning to mention account defaults, got %#v", warning)
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

func TestBuildPreviewIncludesRecommendedRiskSemantics(t *testing.T) {
	registry := provider.NewRegistry(provider.DefaultCatalog()...)
	plan, err := BuildPreview(registry, PreviewRequest{
		SourceProvider: "guangya",
		TargetProvider: "baidu_netdisk",
		RiskMode:       RiskModeFast,
		SelectedRoots:  []string{"/demo", "/more"},
		Entries: []SourceEntry{
			{Path: "/demo/a.bin", Size: 10, MD5: "md5-a"},
			{Path: "/more/b.bin", Size: 10, MD5: "md5-b"},
		},
	})
	if err != nil {
		t.Fatalf("BuildPreview() error = %v", err)
	}
	if got, _ := plan.Metadata["recommendedRiskMode"].(RiskMode); got != RiskModeBalanced {
		t.Fatalf("expected recommendedRiskMode balanced, got %v", plan.Metadata["recommendedRiskMode"])
	}
	reason, _ := plan.Metadata["recommendedRiskModeReason"].(string)
	if reason == "" || !strings.Contains(reason, "top-level roots") {
		t.Fatalf("expected recommendedRiskModeReason to mention multi-root pacing, got %#v", plan.Metadata["recommendedRiskModeReason"])
	}
	warning, _ := plan.Metadata["aggressiveRiskWarning"].(string)
	if warning == "" || !strings.Contains(warning, "Fast mode") {
		t.Fatalf("expected aggressiveRiskWarning to mention fast mode risk, got %#v", plan.Metadata["aggressiveRiskWarning"])
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
