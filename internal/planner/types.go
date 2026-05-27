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
	RiskKeywords        []string `json:"riskKeywords,omitempty"`
}

type ExecutionMode string

const (
	ExecutionModeLeafFirstLazy ExecutionMode = "leaf_first_lazy"
	ExecutionModePreScanFlat   ExecutionMode = "pre_scan_flat"
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
