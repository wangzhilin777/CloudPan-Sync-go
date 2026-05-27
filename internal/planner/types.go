package planner

type Strategy string

const (
	StrategyFastUpload     Strategy = "fast_upload"
	StrategyDownloadUpload Strategy = "download_upload"
	StrategyPendingManual  Strategy = "pending_manual"
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
	Strategy       Strategy `json:"strategy"`
	ConflictPolicy string   `json:"conflictPolicy"`
}
