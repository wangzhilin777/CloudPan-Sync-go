package task

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

type Item struct {
	ID     string `json:"id"`
	TaskID string `json:"taskId"`
	Path   string `json:"path"`
	Size   int64  `json:"size"`
}

type Result struct {
	ID        string `json:"id"`
	TaskID    string `json:"taskId"`
	ItemID    string `json:"itemId"`
	Status    string `json:"status"`
	Mode      string `json:"mode"`
	Message   string `json:"message"`
	CreatedAt string `json:"createdAt"`
}
