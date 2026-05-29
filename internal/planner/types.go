package planner

type Strategy string

const (
	StrategyFastUpload     Strategy = "fast_upload"
	StrategyDownloadUpload Strategy = "download_upload"
	StrategyPendingManual  Strategy = "pending_manual"
)

type RiskMode string

const (
	RiskModeSafe     RiskMode = "safe"
	RiskModeBalanced RiskMode = "balanced"
	RiskModeFast     RiskMode = "fast"
	RiskModeCustom   RiskMode = "custom"
)

type RiskProfile struct {
	Mode                RiskMode `json:"mode"`
	RequestIntervalMS   int      `json:"requestIntervalMs"`
	PageSize            int      `json:"pageSize"`
	DirectoryIntervalMS int      `json:"directoryIntervalMs"`
	CooldownSeconds     int      `json:"cooldownSeconds"`
	RetryLimit          int      `json:"retryLimit"`
	MaxConcurrent       int      `json:"maxConcurrent,omitempty"`
	AutoRetryStartHour  int      `json:"autoRetryStartHour,omitempty"`
	AutoRetryEndHour    int      `json:"autoRetryEndHour,omitempty"`
	RiskKeywords        []string `json:"riskKeywords,omitempty"`
}

type RiskProfileOverride struct {
	RequestIntervalMS   *int     `json:"requestIntervalMs,omitempty"`
	PageSize            *int     `json:"pageSize,omitempty"`
	DirectoryIntervalMS *int     `json:"directoryIntervalMs,omitempty"`
	CooldownSeconds     *int     `json:"cooldownSeconds,omitempty"`
	RetryLimit          *int     `json:"retryLimit,omitempty"`
	MaxConcurrent       *int     `json:"maxConcurrent,omitempty"`
	AutoRetryStartHour  *int     `json:"autoRetryStartHour,omitempty"`
	AutoRetryEndHour    *int     `json:"autoRetryEndHour,omitempty"`
	RiskKeywords        []string `json:"riskKeywords,omitempty"`
}

type RiskProfileResolution struct {
	ProviderKey        string               `json:"providerKey"`
	Mode               RiskMode             `json:"mode"`
	Base               RiskProfile          `json:"base"`
	Calibrated         RiskProfile          `json:"calibrated"`
	Applied            RiskProfile          `json:"applied"`
	CalibrationReasons []string             `json:"calibrationReasons,omitempty"`
	Override           *RiskProfileOverride `json:"override,omitempty"`
	OverrideFields     []string             `json:"overrideFields,omitempty"`
}

type ExecutionMode string

const (
	ExecutionModeLeafFirstLazy ExecutionMode = "leaf_first_lazy"
	ExecutionModePreScanFlat   ExecutionMode = "pre_scan_flat"
)

type SourceDeletePolicy string

const (
	SourceDeletePolicyRecordOnly SourceDeletePolicy = "record_only"
)

type Plan struct {
	SourceProvider string                 `json:"sourceProvider"`
	TargetProvider string                 `json:"targetProvider"`
	ThresholdMB    int                    `json:"thresholdMB"`
	Items          []Item                 `json:"items"`
	Summary        map[string]int         `json:"summary"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

type Item struct {
	Path           string   `json:"path"`
	Size           int64    `json:"size"`
	Sequence       int      `json:"sequence"`
	Strategy       Strategy `json:"strategy"`
	ConflictPolicy string   `json:"conflictPolicy"`
}
