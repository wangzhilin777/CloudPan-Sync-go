package task

type ProviderProbe struct {
	ID          string                 `json:"id"`
	ProviderKey string                 `json:"providerKey"`
	ProfileID   string                 `json:"profileId,omitempty"`
	Status      string                 `json:"status"`
	Payload     map[string]interface{} `json:"payload,omitempty"`
	CreatedAt   string                 `json:"createdAt"`
}

type ProviderStatus struct {
	ID          string                 `json:"id"`
	ProviderKey string                 `json:"providerKey"`
	Summary     map[string]interface{} `json:"summary"`
	CreatedAt   string                 `json:"createdAt"`
}

type State string

const (
	StateReady               State = "ready"
	StateRunning             State = "running"
	StatePaused              State = "paused"
	StateBlocked             State = "blocked"
	StateCompleted           State = "completed"
	StateCompletedWithErrors State = "completed_with_errors"
)

type CompletionKind string

const (
	CompletionKindRealTransfer  CompletionKind = "real_transfer"
	CompletionKindProbeOnly     CompletionKind = "probe_only"
	CompletionKindCandidateOnly CompletionKind = "candidate_only"
	CompletionKindLiveFailed    CompletionKind = "live_failed"
)

type Task struct {
	ID             string         `json:"id"`
	SourceProvider string         `json:"sourceProvider"`
	TargetProvider string         `json:"targetProvider"`
	State          State          `json:"state"`
	CompletionKind CompletionKind `json:"completionKind"`
	CreatedAt      string         `json:"createdAt"`
	UpdatedAt      string         `json:"updatedAt"`
}

type DirectoryState struct {
	Path           string `json:"path"`
	RootPath       string `json:"rootPath,omitempty"`
	Status         string `json:"status"`
	TotalItems     int    `json:"totalItems"`
	ProcessedItems int    `json:"processedItems"`
	DoneItems      int    `json:"doneItems"`
	SkippedItems   int    `json:"skippedItems"`
	FailedItems    int    `json:"failedItems"`
	LastItemPath   string `json:"lastItemPath,omitempty"`
}

type RuntimeState struct {
	ExecutionState    string           `json:"executionState"`
	CurrentRoot       string           `json:"currentRoot,omitempty"`
	CurrentDirectory  string           `json:"currentDirectory,omitempty"`
	CurrentItemPath   string           `json:"currentItemPath,omitempty"`
	LastCompletedPath string           `json:"lastCompletedPath,omitempty"`
	ProcessedCount    int              `json:"processedCount"`
	DoneCount         int              `json:"doneCount"`
	SkippedCount      int              `json:"skippedCount"`
	FailedCount       int              `json:"failedCount"`
	NextSequence      int              `json:"nextSequence"`
	DirectoryStates   []DirectoryState `json:"directoryStates,omitempty"`
}

type Item struct {
	ID     string `json:"id"`
	TaskID string `json:"taskId"`
	Path   string `json:"path"`
	Size   int64  `json:"size"`
}

type Result struct {
	ID             string                 `json:"id"`
	TaskID         string                 `json:"taskId"`
	ItemID         string                 `json:"itemId"`
	Status         string                 `json:"status"`
	Mode           string                 `json:"mode"`
	Message        string                 `json:"message"`
	ConflictAction string                 `json:"conflictAction,omitempty"`
	Payload        map[string]interface{} `json:"payload,omitempty"`
	CreatedAt      string                 `json:"createdAt"`
}
