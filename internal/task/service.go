package task

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"cloudpan-sync-go/internal/auth"
	"cloudpan-sync-go/internal/planner"
	"cloudpan-sync-go/internal/provider"
	sqlitestore "cloudpan-sync-go/internal/store/sqlite"
)

type CreateRequest struct {
	SourceProvider  string                       `json:"sourceProvider"`
	SourceProfileID string                       `json:"sourceProfileId"`
	TargetProvider  string                       `json:"targetProvider"`
	TargetProfileID string                       `json:"targetProfileId"`
	ThresholdMB     int                          `json:"thresholdMB"`
	RiskMode        planner.RiskMode             `json:"riskMode"`
	RiskOverride    *planner.RiskProfileOverride `json:"riskOverride,omitempty"`
	ExecutionMode   planner.ExecutionMode        `json:"executionMode"`
	ConflictPolicy  provider.ConflictPolicy      `json:"conflictPolicy"`
	SelectedRoots   []string                     `json:"selectedRoots"`
	Entries         []planner.SourceEntry        `json:"entries"`
}

type RetryOptions struct {
	Paths []string `json:"paths,omitempty"`
	Scope string   `json:"scope,omitempty"`
}

type RecoverOptions struct {
	Mode                  string   `json:"mode,omitempty"`
	IncludeNonRunnable    bool     `json:"includeNonRunnable,omitempty"`
	TaskID                string   `json:"taskId,omitempty"`
	ProtocolGroup         string   `json:"protocolGroup,omitempty"`
	ProviderKey           string   `json:"providerKey,omitempty"`
	ProfileID             string   `json:"profileId,omitempty"`
	RetryClass            string   `json:"retryClass,omitempty"`
	BlockedAction         string   `json:"blockedAction,omitempty"`
	RecoverState          string   `json:"recoverState,omitempty"`
	Paths                 []string `json:"paths,omitempty"`
	Path                  string   `json:"path,omitempty"`
	Scope                 string   `json:"scope,omitempty"`
	Limit                 int      `json:"limit,omitempty"`
	LimitPerMode          int      `json:"limitPerMode,omitempty"`
	LimitPerLane          int      `json:"limitPerLane,omitempty"`
	LimitPerProtocolGroup int      `json:"limitPerProtocolGroup,omitempty"`
	LimitPerProvider      int      `json:"limitPerProvider,omitempty"`
	LimitPerProfile       int      `json:"limitPerProfile,omitempty"`
}

type RecoverResult struct {
	Mode                         string `json:"mode,omitempty"`
	TaskID                       string `json:"taskId,omitempty"`
	ProtocolGroup                string `json:"protocolGroup,omitempty"`
	ProviderKey                  string `json:"providerKey,omitempty"`
	ProfileID                    string `json:"profileId,omitempty"`
	RetryClass                   string `json:"retryClass,omitempty"`
	BlockedAction                string `json:"blockedAction,omitempty"`
	RecoverState                 string `json:"recoverState,omitempty"`
	Path                         string `json:"path,omitempty"`
	Scope                        string `json:"scope,omitempty"`
	Limit                        int    `json:"limit,omitempty"`
	LimitPerMode                 int    `json:"limitPerMode,omitempty"`
	LimitPerLane                 int    `json:"limitPerLane,omitempty"`
	LimitPerProtocolGroup        int    `json:"limitPerProtocolGroup,omitempty"`
	LimitPerProvider             int    `json:"limitPerProvider,omitempty"`
	LimitPerProfile              int    `json:"limitPerProfile,omitempty"`
	MatchedCount                 int    `json:"matchedCount"`
	RecoveredCount               int    `json:"recoveredCount"`
	SkippedByLimit               int    `json:"skippedByLimit"`
	SkippedByModeBudget          int    `json:"skippedByModeBudget"`
	SkippedByLaneBudget          int    `json:"skippedByLaneBudget"`
	SkippedByProtocolGroupBudget int    `json:"skippedByProtocolGroupBudget"`
	SkippedByProviderBudget      int    `json:"skippedByProviderBudget"`
	SkippedByProfileBudget       int    `json:"skippedByProfileBudget"`
	SkippedByCooldownWait        int    `json:"skippedByCooldownWait"`
	SkippedByRetryWindowWait     int    `json:"skippedByRetryWindowWait"`
	SkippedByBlockedReason       int    `json:"skippedByBlockedReason"`
}

type Detail struct {
	Task            Task                  `json:"task"`
	Plan            planner.Plan          `json:"plan"`
	Runtime         RuntimeState          `json:"runtime"`
	Items           []Item                `json:"items"`
	Results         []Result              `json:"results"`
	SourceProfileID string                `json:"sourceProfileId,omitempty"`
	TargetProfileID string                `json:"targetProfileId,omitempty"`
	ConflictPolicy  string                `json:"conflictPolicy,omitempty"`
	SourceEntries   []planner.SourceEntry `json:"sourceEntries,omitempty"`
}

type EvidenceSummary struct {
	TotalTasks         int                `json:"totalTasks"`
	CompletedTasks     int                `json:"completedTasks"`
	BlockedTasks       int                `json:"blockedTasks"`
	AutoRecoverTasks   int                `json:"autoRecoverTasks"`
	FailedResultCount  int                `json:"failedResultCount"`
	DoneResultCount    int                `json:"doneResultCount"`
	SkippedResultCount int                `json:"skippedResultCount"`
	PendingResultCount int                `json:"pendingResultCount"`
	RiskHitCount       int                `json:"riskHitCount"`
	BlockedActions     []BlockedAction    `json:"blockedActions,omitempty"`
	AutoRecoverPool    []AutoRecoverLane  `json:"autoRecoverPool,omitempty"`
	ProtocolCoverage   []ProtocolCoverage `json:"protocolCoverage,omitempty"`
	RecentResults      []Result           `json:"recentResults"`
	RecentProbes       []ProviderProbe    `json:"recentProbes"`
}

type AutoRecoverLane struct {
	Mode                        string   `json:"mode"`
	Advice                      string   `json:"advice,omitempty"`
	TaskCount                   int      `json:"taskCount"`
	ProviderCount               int      `json:"providerCount"`
	ProfileCount                int      `json:"profileCount"`
	SuggestedProviderBudget     int      `json:"suggestedProviderBudget,omitempty"`
	SuggestedProfileBudget      int      `json:"suggestedProfileBudget,omitempty"`
	QueueItemCount              int      `json:"queueItemCount"`
	RetryableNowCount           int      `json:"retryableNowCount"`
	CooldownCount               int      `json:"cooldownCount"`
	RunnableTaskCount           int      `json:"runnableTaskCount"`
	WaitingCooldownTaskCount    int      `json:"waitingCooldownTaskCount"`
	WaitingRetryWindowTaskCount int      `json:"waitingRetryWindowTaskCount"`
	WaitingOtherTaskCount       int      `json:"waitingOtherTaskCount"`
	UploadCheckpointEligible    int      `json:"uploadCheckpointEligible"`
	ProtocolGroups              []string `json:"protocolGroups,omitempty"`
	RetryClasses                []string `json:"retryClasses,omitempty"`
	BlockedActions              []string `json:"blockedActions,omitempty"`
	ProfileIDs                  []string `json:"profileIds,omitempty"`
	PrimaryRetryClass           string   `json:"primaryRetryClass,omitempty"`
	PrimaryBlockedAction        string   `json:"primaryBlockedAction,omitempty"`
	NextRetryAt                 string   `json:"nextRetryAt,omitempty"`
	SampleTaskID                string   `json:"sampleTaskId,omitempty"`
	SampleProvider              string   `json:"sampleProvider,omitempty"`
	SampleProtocolGroup         string   `json:"sampleProtocolGroup,omitempty"`
	SampleProfileID             string   `json:"sampleProfileId,omitempty"`
}

type EvidenceReport struct {
	GeneratedAt    string                   `json:"generatedAt"`
	Title          string                   `json:"title"`
	Note           string                   `json:"note,omitempty"`
	Markdown       string                   `json:"markdown"`
	Summary        EvidenceSummary          `json:"summary"`
	Statuses       []StatusSummary          `json:"statuses"`
	SmokeSummaries []ProviderSmokeSummary   `json:"smokeSummaries,omitempty"`
	SmokeMatrix    []ProviderSmokeMatrixRow `json:"smokeMatrix,omitempty"`
	Samples        []EvidenceSample         `json:"samples"`
}

type EvidenceReportRecord struct {
	ID             string                   `json:"id"`
	GeneratedAt    string                   `json:"generatedAt"`
	Title          string                   `json:"title"`
	Note           string                   `json:"note,omitempty"`
	Markdown       string                   `json:"markdown"`
	Summary        EvidenceSummary          `json:"summary"`
	Statuses       []StatusSummary          `json:"statuses"`
	SmokeSummaries []ProviderSmokeSummary   `json:"smokeSummaries,omitempty"`
	SmokeMatrix    []ProviderSmokeMatrixRow `json:"smokeMatrix,omitempty"`
	Samples        []EvidenceSample         `json:"samples"`
}

type ProviderSmokeRecord struct {
	ID            string            `json:"id"`
	ProviderKey   string            `json:"providerKey"`
	ProtocolGroup string            `json:"protocolGroup,omitempty"`
	AuthMode      string            `json:"authMode,omitempty"`
	Category      string            `json:"category,omitempty"`
	Result        string            `json:"result"`
	Title         string            `json:"title"`
	Note          string            `json:"note,omitempty"`
	Operations    []string          `json:"operations,omitempty"`
	Environment   map[string]string `json:"environment,omitempty"`
	Markdown      string            `json:"markdown,omitempty"`
	CreatedAt     string            `json:"createdAt"`
}

type ProviderSmokeSummary struct {
	ProtocolGroup        string   `json:"protocolGroup"`
	SmokeCount           int      `json:"smokeCount"`
	SuccessCount         int      `json:"successCount"`
	FailureCount         int      `json:"failureCount"`
	ProviderCount        int      `json:"providerCount"`
	ProviderKeys         []string `json:"providerKeys,omitempty"`
	SampleRecordID       string   `json:"sampleRecordId,omitempty"`
	SampleTitle          string   `json:"sampleTitle,omitempty"`
	SampleProviderKey    string   `json:"sampleProviderKey,omitempty"`
	SampleResult         string   `json:"sampleResult,omitempty"`
	SampleCategory       string   `json:"sampleCategory,omitempty"`
	LatestSmokeAt        string   `json:"latestSmokeAt,omitempty"`
	HasRealSuccessSample bool     `json:"hasRealSuccessSample"`
}

type ProviderSmokeMatrixRow struct {
	ProtocolGroup                string   `json:"protocolGroup"`
	SmokeCount                   int      `json:"smokeCount"`
	SuccessCount                 int      `json:"successCount"`
	FailureCount                 int      `json:"failureCount"`
	ProviderCount                int      `json:"providerCount"`
	ProviderKeys                 []string `json:"providerKeys,omitempty"`
	SampleRecordID               string   `json:"sampleRecordId,omitempty"`
	SampleTitle                  string   `json:"sampleTitle,omitempty"`
	SampleProviderKey            string   `json:"sampleProviderKey,omitempty"`
	SampleCategory               string   `json:"sampleCategory,omitempty"`
	SampleResult                 string   `json:"sampleResult,omitempty"`
	LatestSmokeAt                string   `json:"latestSmokeAt,omitempty"`
	HasRealSuccessSample         bool     `json:"hasRealSuccessSample"`
	CoverageTaskCount            int      `json:"coverageTaskCount"`
	CoverageCompletedTaskCount   int      `json:"coverageCompletedTaskCount"`
	CoverageRealSuccessTaskCount int      `json:"coverageRealSuccessTaskCount"`
	CoverageProviderCount        int      `json:"coverageProviderCount"`
	CoverageProviderKeys         []string `json:"coverageProviderKeys,omitempty"`
	CoverageHasRealSuccessSample bool     `json:"coverageHasRealSuccessSample"`
	CoverageSampleTaskID         string   `json:"coverageSampleTaskId,omitempty"`
	CoverageSampleProviderKey    string   `json:"coverageSampleProviderKey,omitempty"`
	CoverageSampleTaskState      string   `json:"coverageSampleTaskState,omitempty"`
	CoverageSampleCompletionKind string   `json:"coverageSampleCompletionKind,omitempty"`
	CoverageLastObservedAt       string   `json:"coverageLastObservedAt,omitempty"`
	Accepted                     bool     `json:"accepted"`
	AcceptanceStatus             string   `json:"acceptanceStatus,omitempty"`
	AcceptanceMissing            []string `json:"acceptanceMissing,omitempty"`
	AcceptanceAdvice             string   `json:"acceptanceAdvice,omitempty"`
}

type EvidenceSample struct {
	ProviderKey       string   `json:"providerKey"`
	TaskID            string   `json:"taskId"`
	SourceProvider    string   `json:"sourceProvider"`
	TargetProvider    string   `json:"targetProvider"`
	TaskState         string   `json:"taskState"`
	CompletionKind    string   `json:"completionKind,omitempty"`
	ExecutionMode     string   `json:"executionMode,omitempty"`
	ScanMode          string   `json:"scanMode,omitempty"`
	SelectedRoots     []string `json:"selectedRoots,omitempty"`
	ScanTrace         []string `json:"scanTrace,omitempty"`
	BlockedReason     string   `json:"blockedReason,omitempty"`
	LastCompletedPath string   `json:"lastCompletedPath,omitempty"`
	ResultCount       int      `json:"resultCount"`
	CreatedAt         string   `json:"createdAt"`
}

type BlockedAction struct {
	Action         string `json:"action"`
	Advice         string `json:"advice,omitempty"`
	TaskCount      int    `json:"taskCount"`
	ProviderCount  int    `json:"providerCount"`
	NextRetryAt    string `json:"nextRetryAt,omitempty"`
	SampleTaskID   string `json:"sampleTaskId,omitempty"`
	SampleProvider string `json:"sampleProvider,omitempty"`
}

type StatusSummary struct {
	ProviderKey      string                 `json:"providerKey"`
	ProtocolGroup    string                 `json:"protocolGroup,omitempty"`
	ProfileCount     int                    `json:"profileCount"`
	TaskCount        int                    `json:"taskCount"`
	CompletedCount   int                    `json:"completedCount"`
	BlockedCount     int                    `json:"blockedCount"`
	AutoRecoverCount int                    `json:"autoRecoverCount"`
	ProtocolCoverage *ProtocolCoverage      `json:"protocolCoverage,omitempty"`
	LastTaskState    string                 `json:"lastTaskState,omitempty"`
	LatestProbe      string                 `json:"latestProbe,omitempty"`
	LastObservedAt   string                 `json:"lastObservedAt,omitempty"`
	SnapshotSummary  map[string]interface{} `json:"snapshotSummary,omitempty"`
}

type Service struct {
	store         *sqlitestore.Store
	registry      *provider.Registry
	authSvc       *auth.Service
	throttleSleep func(context.Context, time.Duration) error
}

func NewService(store *sqlitestore.Store, registry *provider.Registry, authSvc *auth.Service) *Service {
	return &Service{store: store, registry: registry, authSvc: authSvc}
}

func (s *Service) Create(ctx context.Context, req CreateRequest) (Detail, error) {
	plan, err := planner.BuildPreview(s.registry, planner.PreviewRequest{
		SourceProvider: req.SourceProvider,
		TargetProvider: req.TargetProvider,
		ThresholdMB:    req.ThresholdMB,
		RiskMode:       req.RiskMode,
		RiskOverride:   req.RiskOverride,
		ExecutionMode:  req.ExecutionMode,
		ConflictPolicy: req.ConflictPolicy,
		SelectedRoots:  req.SelectedRoots,
		Entries:        req.Entries,
	})
	if err != nil {
		return Detail{}, err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	t := Task{
		ID:             uuid.NewString(),
		SourceProvider: req.SourceProvider,
		TargetProvider: req.TargetProvider,
		State:          StateReady,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	items := make([]Item, 0, len(plan.Items))
	for _, planItem := range plan.Items {
		items = append(items, Item{
			ID:     uuid.NewString(),
			TaskID: t.ID,
			Path:   planItem.Path,
			Size:   planItem.Size,
		})
	}
	runtime := initializeRuntimeState(plan)
	if err := createTask(ctx, s.store, t, plan, runtime, items, req.Entries, req.SourceProfileID, req.TargetProfileID, string(req.ConflictPolicy)); err != nil {
		return Detail{}, err
	}
	return Detail{Task: t, Plan: plan, Runtime: runtime, Items: items, Results: []Result{}, SourceProfileID: req.SourceProfileID, TargetProfileID: req.TargetProfileID, ConflictPolicy: string(req.ConflictPolicy), SourceEntries: req.Entries}, nil
}

func (s *Service) List(ctx context.Context) ([]Detail, error) {
	return listTasks(ctx, s.store)
}

func (s *Service) Get(ctx context.Context, id string) (Detail, bool, error) {
	return getTask(ctx, s.store, id)
}

func (s *Service) Run(ctx context.Context, id string) (Detail, bool, error) {
	detail, ok, err := getTask(ctx, s.store, id)
	if err != nil || !ok {
		return Detail{}, ok, err
	}
	if detail.Task.State != StateReady && detail.Task.State != StatePaused {
		return Detail{}, true, fmt.Errorf("task_not_runnable")
	}
	if err := s.materializeTaskEntriesIfNeeded(ctx, &detail); err != nil {
		return Detail{}, true, err
	}
	ensureRuntimeState(&detail)
	entry, exists := s.registry.Get(detail.Task.TargetProvider)
	if !exists {
		return Detail{}, true, fmt.Errorf("provider_not_found")
	}
	var providerProfile provider.AuthProfile
	if detail.TargetProfileID != "" {
		profile, ok, err := s.authSvc.GetProfile(ctx, detail.TargetProfileID)
		if err != nil {
			return Detail{}, true, err
		}
		if !ok {
			return Detail{}, true, fmt.Errorf("target_profile_not_found")
		}
		providerProfile = provider.AuthProfile{
			ID:          profile.ID,
			ProviderKey: profile.ProviderKey,
			AuthMode:    profile.AuthMode,
			DisplayName: profile.DisplayName,
			Token:       profile.Token,
			Cookie:      profile.Cookie,
			Extra:       profile.Extra,
		}
	}
	if detail.Task.State != StateRunning {
		detail.Task.State = StateRunning
		detail.Task.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		detail.Runtime.ExecutionState = "running"
		clearPauseRequest(&detail.Runtime)
		if err := updateTaskDetailState(ctx, s.store, detail); err != nil {
			return Detail{}, true, err
		}
	}

	results := append([]Result(nil), detail.Results...)
	startIndex := len(results)
	riskProfile := riskProfileFromMetadata(detail.Plan.Metadata)
	previousItemPath := ""
	if len(results) > 0 {
		previousItemPath = stringValue(results[len(results)-1].Payload["path"])
	}
	syncRuntimeCountsFromResults(&detail.Runtime, results)
	syncRuntimeRiskEvidence(&detail.Runtime, detail.Plan.Metadata, results)
	syncRuntimePendingTree(&detail.Runtime, detail.Plan.Metadata, results)
	syncRuntimeRetryQueue(&detail.Runtime, detail.Plan.Metadata, results)
	syncRuntimeUploadCheckpoint(&detail.Runtime, results)
	applyRetryQueueSummary(&detail.Runtime, detail.Plan.Metadata)

	for i := startIndex; i < len(detail.Plan.Items); i++ {
		if paused, err := s.taskPauseRequested(ctx, detail.Task.ID); err != nil {
			return Detail{}, true, err
		} else if paused {
			detail.Task.State = StatePaused
			detail.Runtime.ExecutionState = "paused"
			clearPauseRequest(&detail.Runtime)
			detail.Task.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
			if err := replaceTaskDetailAndResults(ctx, s.store, detail); err != nil {
				return Detail{}, true, err
			}
			return detail, true, nil
		}
		item := detail.Plan.Items[i]
		result := Result{
			ID:     uuid.NewString(),
			TaskID: detail.Task.ID,
			ItemID: detail.Items[i].ID,
			Payload: map[string]interface{}{
				"path": item.Path,
			},
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
		}
		throttleEvidence, throttleErr := s.applyRuntimeThrottle(ctx, riskProfile, previousItemPath, item.Path)
		if throttleErr != nil {
			return Detail{}, true, throttleErr
		}
		if len(throttleEvidence) > 0 {
			result.Payload["throttle"] = throttleEvidence
		}
		updateRuntimeBeforeItem(&detail, item.Path, i)
		localPath := lookupLocalPath(detail.SourceEntries, item.Path)
		targetState := s.inspectTargetState(entry, providerProfile, detail.SourceEntries, item.Path, item.Size)
		result.Payload["targetState"] = targetState
		switch targetState.Decision {
		case "skip":
			result.Mode = "runtime_skip"
			result.Message = "Target already has a matching file; skip upload."
			result.Status = "skipped"
			result.Payload["providerStatus"] = "target_already_synced"
			result.Payload["syncDecision"] = "skip"
			if targetState.Reason != "" {
				result.Payload["syncDecisionReason"] = targetState.Reason
			}
			if targetState.TargetFingerprint != nil {
				result.Payload["targetFingerprint"] = targetState.TargetFingerprint
			}
			results = append(results, result)
			syncRuntimeRiskEvidence(&detail.Runtime, detail.Plan.Metadata, results)
			syncRuntimePendingTree(&detail.Runtime, detail.Plan.Metadata, results)
			syncRuntimeRetryQueue(&detail.Runtime, detail.Plan.Metadata, results)
			syncRuntimeUploadCheckpoint(&detail.Runtime, results)
			applyRetryQueueSummary(&detail.Runtime, detail.Plan.Metadata)
			detail.Results = results
			updateRuntimeAfterItem(&detail, item.Path, result)
			detail.Task.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
			if err := replaceTaskDetailAndResults(ctx, s.store, detail); err != nil {
				return Detail{}, true, err
			}
			previousItemPath = item.Path
			continue
		case "overwrite", "create":
			result.Payload["syncDecision"] = targetState.Decision
			if targetState.Reason != "" {
				result.Payload["syncDecisionReason"] = targetState.Reason
			}
			if targetState.TargetFingerprint != nil {
				result.Payload["targetFingerprint"] = targetState.TargetFingerprint
			}
		}
		conflictPolicy, conflictAction := resolveConflictPolicy(entry.Meta, provider.ConflictPolicy(detail.ConflictPolicy))
		uploadReq := provider.UploadRequest{
			Profile:        providerProfile,
			Path:           item.Path,
			ParentID:       "",
			Name:           inferUploadName(item.Path),
			Size:           item.Size,
			LocalPath:      localPath,
			ConflictPolicy: conflictPolicy,
			Strategy:       string(item.Strategy),
			MD5:            lookupMD5(detail.SourceEntries, item.Path),
			SHA1:           lookupSHA1(detail.SourceEntries, item.Path),
			GCID:           lookupGCID(detail.SourceEntries, item.Path),
			ResumeUpload:   resumeUploadForPath(detail.Plan.Metadata, item.Path),
		}
		upload, fallbackUsed, fastCheckPayload := s.executeUpload(entry, uploadReq)
		result.Mode = upload.Mode
		result.Message = upload.Message
		result.ConflictAction = conflictAction
		result.Payload["strategy"] = uploadReq.Strategy
		result.Payload["providerStatus"] = upload.Status
		if len(upload.Payload) > 0 {
			result.Payload["upload"] = copyPayloadMap(upload.Payload)
		}
		if fastCheckPayload != nil {
			result.Payload["fastCheck"] = fastCheckPayload
		}
		if executionMode, ok := detail.Plan.Metadata["executionMode"]; ok {
			result.Payload["executionMode"] = executionMode
		}
		if recommendedMode, ok := detail.Plan.Metadata["recommendedExecutionMode"]; ok {
			result.Payload["recommendedExecutionMode"] = recommendedMode
		}
		if recommendedReason, ok := detail.Plan.Metadata["recommendedExecutionModeReason"]; ok {
			result.Payload["recommendedExecutionModeReason"] = recommendedReason
		}
		if item.Sequence > 0 {
			result.Payload["sequence"] = item.Sequence
		}
		if riskProfile, ok := detail.Plan.Metadata["riskProfile"]; ok {
			result.Payload["riskProfile"] = riskProfile
		}
		if autoRecovery := runtimeAutoRecoveryPayload(detail.Runtime); autoRecovery != nil {
			result.Payload["autoRecovery"] = autoRecovery
		}
		if conflictAction != "" {
			result.Payload["conflictAction"] = conflictAction
		}
		if fallbackUsed {
			result.Payload["fallbackUsed"] = true
		}
		if item.Strategy == planner.StrategyPendingManual && !upload.OK {
			result.Status = "failed"
		} else if upload.OK {
			result.Status = "done"
		} else {
			result.Status = "failed"
		}
		results = append(results, result)
		syncRuntimeRiskEvidence(&detail.Runtime, detail.Plan.Metadata, results)
		syncRuntimePendingTree(&detail.Runtime, detail.Plan.Metadata, results)
		syncRuntimeRetryQueue(&detail.Runtime, detail.Plan.Metadata, results)
		syncRuntimeUploadCheckpoint(&detail.Runtime, results)
		applyRetryQueueSummary(&detail.Runtime, detail.Plan.Metadata)
		detail.Results = results
		updateRuntimeAfterItem(&detail, item.Path, result)
		if paused, err := s.taskPauseRequested(ctx, detail.Task.ID); err != nil {
			return Detail{}, true, err
		} else if paused {
			detail.Task.State = StatePaused
			detail.Runtime.ExecutionState = "paused"
			clearPauseRequest(&detail.Runtime)
			detail.Task.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
			if err := replaceTaskDetailAndResults(ctx, s.store, detail); err != nil {
				return Detail{}, true, err
			}
			return detail, true, nil
		}
		detail.Task.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		if err := replaceTaskDetailAndResults(ctx, s.store, detail); err != nil {
			return Detail{}, true, err
		}
		previousItemPath = item.Path
	}

	failed := detail.Runtime.FailedCount
	detail.Task.State = StateCompleted
	detail.Task.CompletionKind = completionKindFromResults(results)
	detail.Runtime.ExecutionState = "completed"
	if failed > 0 {
		if detail.Runtime.BlockedReason != "" {
			detail.Task.State = StateBlocked
			detail.Runtime.ExecutionState = "blocked"
		} else {
			detail.Task.State = StateCompletedWithErrors
		}
	}
	detail.Runtime.CurrentItemPath = ""
	detail.Runtime.CurrentDirectory = ""
	detail.Runtime.CurrentRoot = ""
	detail.Task.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	detail.Results = results
	if err := replaceTaskDetailAndResults(ctx, s.store, detail); err != nil {
		return Detail{}, true, err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	probe := buildProviderProbe(detail, providerProfile, results, now)
	if err := saveProviderProbe(ctx, s.store, probe); err != nil {
		return Detail{}, true, err
	}
	snapshot, err := buildProviderStatusSnapshot(ctx, s.store, detail, probe, now)
	if err != nil {
		return Detail{}, true, err
	}
	if err := saveProviderStatusSnapshot(ctx, s.store, snapshot); err != nil {
		return Detail{}, true, err
	}
	return detail, true, nil
}

type targetInspection struct {
	Decision          string
	Reason            string
	TargetFingerprint map[string]interface{}
}

type retryQueueSummary struct {
	ShouldBlock              bool
	BlockedReason            string
	BlockedAction            string
	BlockedAdvice            string
	NextRetryAt              string
	WindowBlocked            bool
	CanAutoRetry             bool
	RetryableNowCount        int
	CooldownCount            int
	PendingManualCount       int
	AuthExpiredCount         int
	LocalMissingCount        int
	ExhaustedCount           int
	UploadCheckpointEligible int
	AutoRecoverEligible      bool
	AutoRecoverMode          string
	AutoRecoverAdvice        string
}

type pendingTreeBuilderNode struct {
	node     PendingNode
	children map[string]*pendingTreeBuilderNode
}

type recoverCandidate struct {
	Detail            Detail
	Mode              string
	ProtocolGroup     string
	EffectiveAction   string
	PrimaryRetryClass string
	Summary           retryQueueSummary
}

func (s *Service) inspectTargetState(entry provider.Entry, profile provider.AuthProfile, sourceEntries []planner.SourceEntry, path string, size int64) targetInspection {
	if !entry.Capability.SupportsMetadata {
		return targetInspection{Decision: "create", Reason: "target_metadata_unsupported"}
	}
	metadata := entry.Adapter.Metadata(provider.MetadataRequest{
		Profile: profile,
		Path:    path,
	})
	if !metadata.OK {
		return targetInspection{Decision: "create", Reason: "target_missing_or_metadata_unavailable"}
	}
	if !targetEntryExists(metadata) {
		return targetInspection{Decision: "create", Reason: "target_missing_or_metadata_unavailable"}
	}

	sourceFingerprint := sourceFingerprintForPath(sourceEntries, path, size)
	targetFingerprint := fingerprintFromMetadata(metadata.Entry)
	if fingerprintsMatch(sourceFingerprint, targetFingerprint) {
		return targetInspection{
			Decision:          "skip",
			Reason:            "target_already_synced",
			TargetFingerprint: targetFingerprint,
		}
	}
	return targetInspection{
		Decision:          "overwrite",
		Reason:            "target_exists_but_fingerprint_changed",
		TargetFingerprint: targetFingerprint,
	}
}

func (s *Service) materializeTaskEntriesIfNeeded(ctx context.Context, detail *Detail) error {
	if len(detail.SourceEntries) > 0 {
		return nil
	}
	selectedRoots := metadataStringSlice(detail.Plan.Metadata, "selectedRoots")
	if len(selectedRoots) == 0 {
		return nil
	}
	if strings.TrimSpace(detail.SourceProfileID) == "" {
		return fmt.Errorf("source_profile_required_for_lazy_scan")
	}
	executionMode, err := executionModeFromMetadata(detail.Plan.Metadata)
	if err != nil {
		return err
	}
	sourceProfile, sourceEntry, err := s.resolveSourceProfile(ctx, detail.Task.SourceProvider, detail.SourceProfileID)
	if err != nil {
		return err
	}
	riskProfile, _ := detail.Plan.Metadata["riskProfile"].(map[string]interface{})
	pageSize := 0
	if riskProfile != nil {
		pageSize = intNumber(riskProfile["pageSize"])
	}
	if pageSize <= 0 {
		pageSize = 200
	}

	entries, trace, err := s.collectEntriesByMode(ctx, executionMode, sourceEntry, sourceProfile, selectedRoots, pageSize)
	if err != nil {
		return err
	}
	plan, err := planner.BuildPreview(s.registry, planner.PreviewRequest{
		SourceProvider: detail.Task.SourceProvider,
		TargetProvider: detail.Task.TargetProvider,
		ThresholdMB:    detail.Plan.ThresholdMB,
		RiskMode:       planner.RiskMode(metadataStringFromRisk(riskProfile, "mode")),
		RiskOverride:   riskOverrideFromMetadata(detail.Plan.Metadata),
		ExecutionMode:  executionMode,
		ConflictPolicy: provider.ConflictPolicy(detail.ConflictPolicy),
		SelectedRoots:  selectedRoots,
		Entries:        entries,
	})
	if err != nil {
		return err
	}
	if plan.Metadata == nil {
		plan.Metadata = map[string]interface{}{}
	}
	plan.Metadata["scanMode"] = scanModeForExecutionMode(executionMode)
	plan.Metadata["scanTrace"] = trace

	items := make([]Item, 0, len(plan.Items))
	for _, planItem := range plan.Items {
		items = append(items, Item{
			ID:     uuid.NewString(),
			TaskID: detail.Task.ID,
			Path:   planItem.Path,
			Size:   planItem.Size,
		})
	}

	detail.SourceEntries = entries
	detail.Plan = plan
	detail.Runtime = initializeRuntimeState(plan)
	detail.Items = items
	detail.Task.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	return replaceTaskPlanAndItems(ctx, s.store, *detail)
}

func (s *Service) Pause(ctx context.Context, id string) (Detail, bool, error) {
	detail, ok, err := getTask(ctx, s.store, id)
	if err != nil || !ok {
		return Detail{}, ok, err
	}
	ensureRuntimeState(&detail)
	switch detail.Task.State {
	case StateReady:
		detail.Task.State = StatePaused
		detail.Task.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		detail.Runtime.ExecutionState = "paused"
		clearPauseRequest(&detail.Runtime)
		if err := updateTaskDetailState(ctx, s.store, detail); err != nil {
			return Detail{}, true, err
		}
		return detail, true, nil
	case StateRunning:
		now := time.Now().UTC().Format(time.RFC3339)
		detail.Task.UpdatedAt = now
		detail.Runtime.ExecutionState = "pause_requested"
		detail.Runtime.PauseRequested = true
		detail.Runtime.PauseRequestedAt = now
		detail.Runtime.PauseRequestSource = "user"
		if err := updateTaskDetailState(ctx, s.store, detail); err != nil {
			return Detail{}, true, err
		}
		return detail, true, nil
	default:
		return Detail{}, true, fmt.Errorf("task_state_transition_not_allowed")
	}
}

func (s *Service) Resume(ctx context.Context, id string) (Detail, bool, error) {
	detail, ok, err := getTask(ctx, s.store, id)
	if err != nil || !ok {
		return Detail{}, ok, err
	}
	ensureRuntimeState(&detail)
	switch detail.Task.State {
	case StatePaused, StateBlocked:
		return s.transitionState(ctx, id, []State{detail.Task.State}, StateReady)
	case StateRunning:
		if !detail.Runtime.PauseRequested {
			return Detail{}, true, fmt.Errorf("task_state_transition_not_allowed")
		}
		detail.Task.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		detail.Runtime.ExecutionState = "running"
		clearPauseRequest(&detail.Runtime)
		if err := updateTaskDetailState(ctx, s.store, detail); err != nil {
			return Detail{}, true, err
		}
		return detail, true, nil
	default:
		return Detail{}, true, fmt.Errorf("task_state_transition_not_allowed")
	}
}

func (s *Service) Retry(ctx context.Context, id string) (Detail, bool, error) {
	return s.RetryWithOptions(ctx, id, RetryOptions{})
}

func (s *Service) RetryWithOptions(ctx context.Context, id string, opts RetryOptions) (Detail, bool, error) {
	detail, ok, err := getTask(ctx, s.store, id)
	if err != nil || !ok {
		return Detail{}, ok, err
	}
	retried, err := s.buildRetryDetail(detail, opts)
	if err != nil {
		return Detail{}, true, err
	}
	if err := rebuildTaskForRetry(ctx, s.store, retried); err != nil {
		return Detail{}, true, err
	}
	return retried, true, nil
}

func lookupMD5(entries []planner.SourceEntry, path string) string {
	for _, item := range entries {
		if item.Path == path {
			if item.MD5 != "" {
				return item.MD5
			}
			return item.ETag
		}
	}
	return ""
}

func lookupSHA1(entries []planner.SourceEntry, path string) string {
	for _, item := range entries {
		if item.Path == path {
			return item.SHA1
		}
	}
	return ""
}

func lookupGCID(entries []planner.SourceEntry, path string) string {
	for _, item := range entries {
		if item.Path == path {
			return item.GCID
		}
	}
	return ""
}

func lookupLocalPath(entries []planner.SourceEntry, path string) string {
	for _, item := range entries {
		if item.Path == path {
			return item.LocalPath
		}
	}
	return ""
}

func buildTaskItems(taskID string, plan planner.Plan) []Item {
	items := make([]Item, 0, len(plan.Items))
	for _, planItem := range plan.Items {
		items = append(items, Item{
			ID:     uuid.NewString(),
			TaskID: taskID,
			Path:   planItem.Path,
			Size:   planItem.Size,
		})
	}
	return items
}

func inferUploadName(path string) string {
	if path == "" || path == "/" {
		return "upload.bin"
	}
	lastSlash := -1
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			lastSlash = i
			break
		}
	}
	if lastSlash >= 0 && lastSlash < len(path)-1 {
		return path[lastSlash+1:]
	}
	return path
}

func (s *Service) buildRetryDetail(detail Detail, opts RetryOptions) (Detail, error) {
	previousState := detail.Task.State
	syncRuntimeRetryQueue(&detail.Runtime, detail.Plan.Metadata, detail.Results)
	syncRuntimeUploadCheckpoint(&detail.Runtime, detail.Results)
	detail.Task.State = StateReady
	detail.Task.CompletionKind = ""
	detail.Task.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	retryEntries, retryPaths, retryMode, retryBlockedUntil, retryBlockedReason := selectRetryEntries(detail, opts)
	if retryBlockedUntil != "" {
		return Detail{}, fmt.Errorf("retry_cooldown_active:%s", retryBlockedUntil)
	}
	if retryBlockedReason == "retry_selection_empty" {
		return Detail{}, fmt.Errorf("retry_selection_empty")
	}
	if retryBlockedReason != "" {
		return Detail{}, fmt.Errorf("retry_blocked:%s", retryBlockedReason)
	}
	if len(retryEntries) > 0 {
		executionMode, err := executionModeFromMetadata(detail.Plan.Metadata)
		if err != nil {
			return Detail{}, err
		}
		selectedRoots := retrySelectedRoots(metadataStringSlice(detail.Plan.Metadata, "selectedRoots"), retryPaths)
		riskProfile := riskProfileFromMetadata(detail.Plan.Metadata)
		retryAttempts := incrementRetryAttempts(detail.Plan.Metadata, retryPaths)
		retryUploadCheckpoints := buildRetryUploadCheckpointMetadata(detail.Runtime.RetryQueue, retryPaths)
		plan, err := planner.BuildPreview(s.registry, planner.PreviewRequest{
			SourceProvider: detail.Task.SourceProvider,
			TargetProvider: detail.Task.TargetProvider,
			ThresholdMB:    detail.Plan.ThresholdMB,
			RiskMode:       riskProfile.Mode,
			RiskOverride:   riskOverrideFromMetadata(detail.Plan.Metadata),
			ExecutionMode:  executionMode,
			ConflictPolicy: provider.ConflictPolicy(detail.ConflictPolicy),
			SelectedRoots:  selectedRoots,
			Entries:        retryEntries,
		})
		if err != nil {
			return Detail{}, err
		}
		if plan.Metadata == nil {
			plan.Metadata = map[string]interface{}{}
		}
		plan.Metadata["retryPendingOnly"] = true
		if retryMode == "retry_queue" || retryMode == "selected_retry_subset" || retryMode == "selected_directory_subset" {
			plan.Metadata["retryPendingOnly"] = false
		}
		plan.Metadata["retryMode"] = retryMode
		plan.Metadata["retryPendingPaths"] = retryPaths
		if len(opts.Paths) > 0 {
			plan.Metadata["retrySelectedPaths"] = normalizeSelectionPaths(opts.Paths)
			plan.Metadata["retryScope"] = strings.TrimSpace(opts.Scope)
		}
		plan.Metadata["retrySourceResultCount"] = len(detail.Results)
		plan.Metadata["retrySourceTaskState"] = string(previousState)
		plan.Metadata["retryAttempts"] = retryAttempts
		if len(retryUploadCheckpoints) > 0 {
			plan.Metadata["retryUploadCheckpoints"] = retryUploadCheckpoints
		}
		delete(plan.Metadata, "retrySummary")
		detail.Plan = plan
		detail.SourceEntries = retryEntries
		detail.Items = buildTaskItems(detail.Task.ID, plan)
	} else if detail.Plan.Metadata != nil {
		delete(detail.Plan.Metadata, "retryPendingOnly")
		delete(detail.Plan.Metadata, "retryPendingPaths")
		delete(detail.Plan.Metadata, "retrySourceResultCount")
		delete(detail.Plan.Metadata, "retrySourceTaskState")
		delete(detail.Plan.Metadata, "retryAttempts")
		delete(detail.Plan.Metadata, "retryUploadCheckpoints")
		delete(detail.Plan.Metadata, "retrySelectedPaths")
		delete(detail.Plan.Metadata, "retryScope")
		delete(detail.Plan.Metadata, "retrySummary")
	}
	detail.Runtime = initializeRuntimeState(detail.Plan)
	if checkpoint := firstRetryUploadCheckpoint(detail.Plan.Metadata, retryPaths); checkpoint != nil {
		detail.Runtime.UploadCheckpoint = checkpoint
	}
	detail.Results = []Result{}
	return detail, nil
}

func (s *Service) RuntimeEvidence(ctx context.Context) (EvidenceSummary, error) {
	return taskEvidenceSummary(ctx, s.store, s.registry.List())
}

func (s *Service) ProviderStatuses(ctx context.Context) ([]StatusSummary, error) {
	return providerStatusSummary(ctx, s.store, s.registry.List())
}

func (s *Service) ProviderSmokeSummary(ctx context.Context) ([]ProviderSmokeSummary, error) {
	records, err := s.ListProviderSmokeRecords(ctx)
	if err != nil {
		return nil, err
	}
	return summarizeProviderSmokeRecords(records), nil
}

func (s *Service) ProviderSmokeMatrix(ctx context.Context) ([]ProviderSmokeMatrixRow, error) {
	summary, err := s.RuntimeEvidence(ctx)
	if err != nil {
		return nil, err
	}
	smokeSummaries, err := s.ProviderSmokeSummary(ctx)
	if err != nil {
		return nil, err
	}
	return buildProviderSmokeMatrix(summary, smokeSummaries), nil
}

func (s *Service) EvidenceReport(ctx context.Context) (EvidenceReport, error) {
	summary, err := s.RuntimeEvidence(ctx)
	if err != nil {
		return EvidenceReport{}, err
	}
	statuses, err := s.ProviderStatuses(ctx)
	if err != nil {
		return EvidenceReport{}, err
	}
	smokeSummaries, err := s.ProviderSmokeSummary(ctx)
	if err != nil {
		return EvidenceReport{}, err
	}
	smokeMatrix := buildProviderSmokeMatrix(summary, smokeSummaries)
	details, err := s.List(ctx)
	if err != nil {
		return EvidenceReport{}, err
	}
	return buildEvidenceReport(summary, statuses, smokeSummaries, smokeMatrix, buildEvidenceSamples(details, 12), time.Now().UTC().Format(time.RFC3339), "", ""), nil
}

func (s *Service) SaveEvidenceReport(ctx context.Context, title, note string) (EvidenceReportRecord, error) {
	report, err := s.EvidenceReport(ctx)
	if err != nil {
		return EvidenceReportRecord{}, err
	}
	return saveEvidenceReport(ctx, s.store, buildEvidenceReport(report.Summary, report.Statuses, report.SmokeSummaries, report.SmokeMatrix, report.Samples, report.GeneratedAt, title, note))
}

func (s *Service) ListEvidenceReports(ctx context.Context) ([]EvidenceReportRecord, error) {
	return listEvidenceReports(ctx, s.store)
}

func (s *Service) GetEvidenceReport(ctx context.Context, id string) (EvidenceReportRecord, bool, error) {
	return getEvidenceReport(ctx, s.store, id)
}

func (s *Service) SaveProviderSmokeRecord(ctx context.Context, record ProviderSmokeRecord) (ProviderSmokeRecord, error) {
	if strings.TrimSpace(record.ProviderKey) == "" {
		return ProviderSmokeRecord{}, fmt.Errorf("provider_key_required")
	}
	if strings.TrimSpace(record.Result) == "" {
		record.Result = "success"
	}
	if strings.TrimSpace(record.Title) == "" {
		record.Title = strings.TrimSpace(record.ProviderKey) + " 真实 smoke 记录"
	}
	if strings.TrimSpace(record.Category) == "" {
		record.Category = inferProviderSmokeCategory(record)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	record.ID = uuid.NewString()
	record.Title = strings.TrimSpace(record.Title)
	record.Note = strings.TrimSpace(record.Note)
	record.ProviderKey = strings.TrimSpace(record.ProviderKey)
	record.ProtocolGroup = strings.TrimSpace(record.ProtocolGroup)
	record.AuthMode = strings.TrimSpace(record.AuthMode)
	record.Category = strings.TrimSpace(record.Category)
	record.Result = strings.TrimSpace(record.Result)
	record.CreatedAt = now
	if len(record.Operations) == 0 {
		record.Operations = []string{"ValidateAuth", "List", "Metadata", "CreateDir", "FastUploadCheck", "Upload"}
	}
	if record.Environment == nil {
		record.Environment = map[string]string{}
	}
	record.Markdown = buildProviderSmokeMarkdown(record)
	return saveProviderSmokeRecord(ctx, s.store, record)
}

func (s *Service) ListProviderSmokeRecords(ctx context.Context) ([]ProviderSmokeRecord, error) {
	return listProviderSmokeRecords(ctx, s.store)
}

func (s *Service) GetProviderSmokeRecord(ctx context.Context, id string) (ProviderSmokeRecord, bool, error) {
	return getProviderSmokeRecord(ctx, s.store, id)
}

func (s *Service) RecoverBlockedTasks(ctx context.Context) (int, error) {
	result, err := s.RecoverBlockedTasksWithOptions(ctx, RecoverOptions{})
	if err != nil {
		return result.RecoveredCount, err
	}
	return result.RecoveredCount, nil
}

func (s *Service) RecoverBlockedTasksWithOptions(ctx context.Context, opts RecoverOptions) (RecoverResult, error) {
	opts.Mode = strings.TrimSpace(opts.Mode)
	opts.TaskID = strings.TrimSpace(opts.TaskID)
	opts.ProtocolGroup = strings.TrimSpace(opts.ProtocolGroup)
	opts.ProviderKey = strings.TrimSpace(opts.ProviderKey)
	opts.ProfileID = strings.TrimSpace(opts.ProfileID)
	opts.RetryClass = strings.TrimSpace(opts.RetryClass)
	opts.BlockedAction = strings.TrimSpace(opts.BlockedAction)
	opts.RecoverState = strings.TrimSpace(opts.RecoverState)
	opts.Paths = normalizeRecoverSelectionPaths(opts.Paths, opts.Path)
	opts.Path = firstRecoverSelectionPath(opts.Paths)
	opts.Scope = strings.TrimSpace(opts.Scope)
	providers := s.registry.List()
	result := RecoverResult{
		Mode:                  opts.Mode,
		TaskID:                opts.TaskID,
		ProtocolGroup:         opts.ProtocolGroup,
		ProviderKey:           opts.ProviderKey,
		ProfileID:             opts.ProfileID,
		RetryClass:            opts.RetryClass,
		BlockedAction:         opts.BlockedAction,
		RecoverState:          opts.RecoverState,
		Path:                  opts.Path,
		Scope:                 opts.Scope,
		Limit:                 opts.Limit,
		LimitPerMode:          opts.LimitPerMode,
		LimitPerLane:          opts.LimitPerLane,
		LimitPerProtocolGroup: opts.LimitPerProtocolGroup,
		LimitPerProvider:      opts.LimitPerProvider,
		LimitPerProfile:       opts.LimitPerProfile,
	}
	items, err := s.List(ctx)
	if err != nil {
		return result, err
	}
	candidates := make([]recoverCandidate, 0)
	for _, detail := range items {
		if detail.Task.State != StateBlocked && detail.Task.State != StateCompletedWithErrors && detail.Runtime.ExecutionState != string(StateBlocked) {
			continue
		}
		if opts.TaskID != "" && detail.Task.ID != opts.TaskID {
			continue
		}
		syncRuntimeRetryQueue(&detail.Runtime, detail.Plan.Metadata, detail.Results)
		applyRetryQueueSummary(&detail.Runtime, detail.Plan.Metadata)
		summary := summarizeRetryQueueWithRisk(detail.Runtime.RetryQueue, riskProfileFromMetadata(detail.Plan.Metadata), time.Now().UTC())
		if opts.IncludeNonRunnable {
			if !shouldIncludeAutoRecoverPool(detail, summary) && !recoverStateFallbackMatch(detail, summary, opts.RecoverState) {
				continue
			}
		} else if !taskCanAutoRecover(detail) {
			continue
		}
		candidate := buildRecoverCandidate(detail, protocolGroupForProviderKey(providers, detail.Task.TargetProvider))
		if opts.Mode != "" && candidate.Mode != opts.Mode {
			continue
		}
		if opts.ProtocolGroup != "" && candidate.ProtocolGroup != recoverProtocolGroupBudgetKey(opts.ProtocolGroup) {
			continue
		}
		if opts.ProviderKey != "" && detail.Task.TargetProvider != opts.ProviderKey {
			continue
		}
		if opts.ProfileID != "" && detail.TargetProfileID != opts.ProfileID {
			continue
		}
		if opts.RetryClass != "" && !retryQueueContainsClass(detail.Runtime.RetryQueue, opts.RetryClass) {
			continue
		}
		if opts.BlockedAction != "" && !strings.EqualFold(candidate.EffectiveAction, opts.BlockedAction) {
			continue
		}
		if opts.RecoverState != "" && !recoverStateMatchesCandidate(detail, candidate.Summary, opts.RecoverState) && !recoverStateFallbackMatch(detail, candidate.Summary, opts.RecoverState) {
			continue
		}
		if len(opts.Paths) > 0 {
			retryEntries, _, _, retryBlockedUntil, retryBlockedReason := selectRetryEntries(detail, RetryOptions{
				Paths: opts.Paths,
				Scope: recoverSelectionScope(opts.Scope),
			})
			if retryBlockedUntil != "" || retryBlockedReason != "" || len(retryEntries) == 0 {
				continue
			}
		}
		candidates = append(candidates, candidate)
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		return recoverCandidateLess(candidates[i], candidates[j])
	})
	candidates = interleaveRecoverCandidatesByProviderAndProfile(candidates)
	result.MatchedCount = len(candidates)
	if opts.Limit > 0 && len(candidates) > opts.Limit {
		result.SkippedByLimit = len(candidates) - opts.Limit
		candidates = candidates[:opts.Limit]
	}
	recovered := 0
	recoveredByMode := make(map[string]int)
	recoveredByLane := make(map[string]int)
	recoveredByProtocolGroup := make(map[string]int)
	recoveredByProvider := make(map[string]int)
	recoveredByProfile := make(map[string]int)
	for _, candidate := range candidates {
		detail := candidate.Detail
		modeKey := recoverModeBudgetKey(candidate.Mode)
		if opts.LimitPerMode > 0 && recoveredByMode[modeKey] >= opts.LimitPerMode {
			result.SkippedByModeBudget++
			continue
		}
		laneKey := recoverLaneBudgetKey(candidate)
		if opts.LimitPerLane > 0 && recoveredByLane[laneKey] >= opts.LimitPerLane {
			result.SkippedByLaneBudget++
			continue
		}
		protocolGroupKey := recoverProtocolGroupBudgetKey(candidate.ProtocolGroup)
		if opts.LimitPerProtocolGroup > 0 && recoveredByProtocolGroup[protocolGroupKey] >= opts.LimitPerProtocolGroup {
			result.SkippedByProtocolGroupBudget++
			continue
		}
		providerKey := strings.TrimSpace(detail.Task.TargetProvider)
		profileID := normalizedRecoverProfileID(detail.TargetProfileID)
		providerBudget := recoverProviderBudgetWithOverride(detail, opts.LimitPerProvider)
		if providerBudget > 0 && recoveredByProvider[providerKey] >= providerBudget {
			result.SkippedByProviderBudget++
			continue
		}
		profileBudget := recoverProfileBudgetWithOverride(detail, opts.LimitPerProfile)
		if profileBudget > 0 && recoveredByProfile[recoverProfileBudgetKey(providerKey, profileID)] >= profileBudget {
			result.SkippedByProfileBudget++
			continue
		}
		if candidate.Summary.WindowBlocked {
			result.SkippedByRetryWindowWait++
			continue
		}
		retryOpts := RetryOptions{}
		if len(opts.Paths) > 0 {
			retryOpts.Paths = opts.Paths
			retryOpts.Scope = recoverSelectionScope(opts.Scope)
		}
		retried, err := s.buildRetryDetail(detail, retryOpts)
		if err != nil {
			if strings.HasPrefix(err.Error(), "retry_cooldown_active:") {
				result.SkippedByCooldownWait++
				continue
			}
			if strings.HasPrefix(err.Error(), "retry_blocked:") {
				reason := strings.TrimPrefix(err.Error(), "retry_blocked:")
				if reason == "retry_queue_waiting_for_retry_window" {
					result.SkippedByRetryWindowWait++
				} else {
					result.SkippedByBlockedReason++
				}
				continue
			}
			result.RecoveredCount = recovered
			return result, err
		}
		if err := rebuildTaskForRetry(ctx, s.store, retried); err != nil {
			result.RecoveredCount = recovered
			return result, err
		}
		markAutoRecovery(&retried, detail, autoRecoverReason(detail), time.Now().UTC().Format(time.RFC3339))
		if err := updateTaskDetailState(ctx, s.store, retried); err != nil {
			result.RecoveredCount = recovered
			return result, err
		}
		if _, _, err := s.Run(ctx, detail.Task.ID); err != nil {
			result.RecoveredCount = recovered
			return result, err
		}
		recovered++
		recoveredByMode[modeKey] = recoveredByMode[modeKey] + 1
		recoveredByLane[laneKey] = recoveredByLane[laneKey] + 1
		recoveredByProtocolGroup[protocolGroupKey] = recoveredByProtocolGroup[protocolGroupKey] + 1
		recoveredByProvider[providerKey] = recoveredByProvider[providerKey] + 1
		recoveredByProfile[recoverProfileBudgetKey(providerKey, profileID)] = recoveredByProfile[recoverProfileBudgetKey(providerKey, profileID)] + 1
	}
	result.RecoveredCount = recovered
	return result, nil
}

func recoverStateFallbackMatch(detail Detail, summary retryQueueSummary, recoverState string) bool {
	recoverState = strings.TrimSpace(recoverState)
	switch recoverState {
	case "waiting_cooldown":
		return summary.BlockedReason == "retry_queue_waiting_for_cooldown" || summary.BlockedAction == "wait_for_cooldown" || detail.Runtime.BlockedAction == "wait_for_cooldown"
	case "waiting_retry_window":
		return summary.BlockedReason == "retry_queue_waiting_for_retry_window" || summary.BlockedAction == "wait_for_retry_window" || detail.Runtime.BlockedAction == "wait_for_retry_window"
	default:
		return false
	}
}

func recoverStateMatchesCandidate(detail Detail, summary retryQueueSummary, recoverState string) bool {
	state := classifyAutoRecoverLaneState(detail, summary)
	mode := strings.TrimSpace(summary.AutoRecoverMode)
	if mode == "" {
		mode = autoRecoverReason(detail)
	}
	switch strings.TrimSpace(recoverState) {
	case "runnable_now":
		return state.runnableNow || mode == "retry_queue_auto_retry" || mode == "upload_checkpoint_auto_resume"
	case "waiting_cooldown":
		return state.waitingCooldown || mode == "cooldown_elapsed_auto_retry"
	case "waiting_retry_window":
		return state.waitingRetryWindow || mode == "retry_window_waiting_auto_retry"
	case "waiting_other":
		return state.waitingOther || (mode != "" && mode != "retry_queue_auto_retry" && mode != "upload_checkpoint_auto_resume" && mode != "cooldown_elapsed_auto_retry" && mode != "retry_window_waiting_auto_retry")
	default:
		return false
	}
}

func recoverModeBudgetKey(mode string) string {
	mode = strings.TrimSpace(mode)
	if mode == "" {
		return "_unknown_mode"
	}
	return mode
}

func recoverLaneBudgetKey(candidate recoverCandidate) string {
	parts := []string{
		recoverModeBudgetKey(candidate.Mode),
		strings.TrimSpace(candidate.PrimaryRetryClass),
		strings.TrimSpace(candidate.EffectiveAction),
	}
	return strings.Join(parts, "::")
}

func recoverProtocolGroupBudgetKey(protocolGroup string) string {
	protocolGroup = strings.TrimSpace(protocolGroup)
	if protocolGroup == "" {
		return "_unknown_protocol_group"
	}
	return protocolGroup
}

func recoverSelectionScope(scope string) string {
	scope = strings.TrimSpace(scope)
	if scope == "" {
		return "selected_retry_subset"
	}
	return scope
}

func normalizeRecoverSelectionPaths(paths []string, singlePath string) []string {
	selected := normalizeSelectionPaths(paths)
	if len(selected) > 0 {
		return selected
	}
	singlePath = normalizeScanPath(singlePath)
	if singlePath == "" {
		return nil
	}
	return []string{singlePath}
}

func firstRecoverSelectionPath(paths []string) string {
	if len(paths) == 0 {
		return ""
	}
	return normalizeScanPath(paths[0])
}

func recoverProviderBudget(detail Detail) int {
	riskProfile := riskProfileFromMetadata(detail.Plan.Metadata)
	if riskProfile.MaxConcurrent <= 0 {
		return 0
	}
	return riskProfile.MaxConcurrent
}

func recoverProviderBudgetWithOverride(detail Detail, override int) int {
	return applyRecoverBudgetOverride(recoverProviderBudget(detail), override)
}

func recoverProfileBudget(detail Detail) int {
	riskProfile := riskProfileFromMetadata(detail.Plan.Metadata)
	if riskProfile.MaxConcurrent <= 0 {
		return 0
	}
	switch strings.TrimSpace(detail.Task.TargetProvider) {
	case "baidu_netdisk", "quark", "uc", "189cloud", "115_open", "guangya":
		return 1
	}
	if riskProfile.MaxConcurrent <= 2 {
		return 1
	}
	return minInt(riskProfile.MaxConcurrent, 2)
}

func recoverProfileBudgetWithOverride(detail Detail, override int) int {
	return applyRecoverBudgetOverride(recoverProfileBudget(detail), override)
}

func applyRecoverBudgetOverride(base, override int) int {
	if override <= 0 {
		return base
	}
	if base <= 0 {
		return override
	}
	return minInt(base, override)
}

func recoverProfileBudgetKey(providerKey, profileID string) string {
	return strings.TrimSpace(providerKey) + "::" + normalizedRecoverProfileID(profileID)
}

func normalizedRecoverProfileID(profileID string) string {
	profileID = strings.TrimSpace(profileID)
	if profileID == "" {
		return "_unknown_profile"
	}
	return profileID
}

func interleaveRecoverCandidatesByProviderAndProfile(candidates []recoverCandidate) []recoverCandidate {
	if len(candidates) <= 2 {
		return candidates
	}
	reordered := make([]recoverCandidate, 0, len(candidates))
	for start := 0; start < len(candidates); {
		end := start + 1
		for end < len(candidates) && recoverCandidateSameBand(candidates[start], candidates[end]) {
			end++
		}
		reordered = append(reordered, interleaveRecoverCandidateBand(candidates[start:end])...)
		start = end
	}
	return reordered
}

func recoverCandidateSameBand(left, right recoverCandidate) bool {
	return autoRecoverModePriority(left.Mode) == autoRecoverModePriority(right.Mode) &&
		retryClassPriority(left.PrimaryRetryClass) == retryClassPriority(right.PrimaryRetryClass) &&
		blockedActionPriority(left.EffectiveAction) == blockedActionPriority(right.EffectiveAction)
}

func interleaveRecoverCandidateBand(candidates []recoverCandidate) []recoverCandidate {
	if len(candidates) <= 2 {
		return candidates
	}
	providerOrder := make([]string, 0)
	providerQueues := make(map[string]map[string][]recoverCandidate)
	profileOrders := make(map[string][]string)
	profileCursor := make(map[string]int)
	for _, candidate := range candidates {
		providerKey := strings.TrimSpace(candidate.Detail.Task.TargetProvider)
		if providerKey == "" {
			providerKey = "_unknown"
		}
		profileID := strings.TrimSpace(candidate.Detail.TargetProfileID)
		if profileID == "" {
			profileID = "_unknown_profile"
		}
		if _, ok := providerQueues[providerKey]; !ok {
			providerOrder = append(providerOrder, providerKey)
			providerQueues[providerKey] = make(map[string][]recoverCandidate)
		}
		if _, ok := providerQueues[providerKey][profileID]; !ok {
			profileOrders[providerKey] = append(profileOrders[providerKey], profileID)
		}
		providerQueues[providerKey][profileID] = append(providerQueues[providerKey][profileID], candidate)
	}
	if len(providerOrder) == 0 {
		return candidates
	}
	if len(providerOrder) == 1 {
		singleProvider := providerOrder[0]
		if len(profileOrders[singleProvider]) <= 1 {
			return candidates
		}
	}
	result := make([]recoverCandidate, 0, len(candidates))
	for {
		progressed := false
		for _, providerKey := range providerOrder {
			profiles := profileOrders[providerKey]
			if len(profiles) == 0 {
				continue
			}
			start := profileCursor[providerKey] % len(profiles)
			for offset := 0; offset < len(profiles); offset++ {
				profileID := profiles[(start+offset)%len(profiles)]
				queue := providerQueues[providerKey][profileID]
				if len(queue) == 0 {
					continue
				}
				result = append(result, queue[0])
				providerQueues[providerKey][profileID] = queue[1:]
				profileCursor[providerKey] = (start + offset + 1) % len(profiles)
				progressed = true
				break
			}
		}
		if !progressed {
			break
		}
	}
	return result
}

func retryQueueContainsClass(queue []RetryQueueItem, retryClass string) bool {
	retryClass = strings.TrimSpace(retryClass)
	if retryClass == "" {
		return true
	}
	for _, item := range queue {
		if strings.EqualFold(strings.TrimSpace(item.RetryClass), retryClass) {
			return true
		}
	}
	return false
}

func detailMatchesBlockedAction(detail Detail, blockedAction string) bool {
	blockedAction = strings.TrimSpace(blockedAction)
	if blockedAction == "" {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(effectiveBlockedAction(detail)), blockedAction)
}

func buildRecoverCandidate(detail Detail, protocolGroup string) recoverCandidate {
	mode := autoRecoverReason(detail)
	summary := summarizeRetryQueueWithRisk(detail.Runtime.RetryQueue, riskProfileFromMetadata(detail.Plan.Metadata), time.Now().UTC())
	if mode == "" {
		mode = summary.AutoRecoverMode
	}
	return recoverCandidate{
		Detail:            detail,
		Mode:              mode,
		ProtocolGroup:     recoverProtocolGroupBudgetKey(protocolGroup),
		EffectiveAction:   strings.TrimSpace(effectiveBlockedAction(detail)),
		PrimaryRetryClass: primaryRetryClass(detail.Runtime.RetryQueue),
		Summary:           summary,
	}
}

func primaryRetryClass(queue []RetryQueueItem) string {
	if len(queue) == 0 {
		return ""
	}
	priorities := map[string]int{
		"retry_failed":       0,
		"rate_limited":       1,
		"auth_expired":       2,
		"local_file_missing": 3,
		"pending_manual":     4,
	}
	bestClass := ""
	bestPriority := 99
	for _, item := range queue {
		retryClass := strings.TrimSpace(item.RetryClass)
		if retryClass == "" {
			continue
		}
		priority, ok := priorities[retryClass]
		if !ok {
			priority = 50
		}
		if bestClass == "" || priority < bestPriority || (priority == bestPriority && retryClass < bestClass) {
			bestClass = retryClass
			bestPriority = priority
		}
	}
	return bestClass
}

func retryClassPriority(retryClass string) int {
	switch strings.TrimSpace(retryClass) {
	case "retry_failed":
		return 0
	case "rate_limited":
		return 1
	case "auth_expired":
		return 2
	case "local_file_missing":
		return 3
	case "pending_manual":
		return 4
	default:
		return 9
	}
}

func blockedActionPriority(action string) int {
	switch strings.TrimSpace(action) {
	case "wait_for_cooldown":
		return 0
	case "wait_for_retry_window":
		return 1
	case "review_and_reset_retry_strategy":
		return 2
	case "refresh_auth_profile":
		return 3
	case "restore_local_source_file":
		return 4
	case "manual_confirmation_required":
		return 5
	default:
		return 9
	}
}

func recoverCandidateLess(left, right recoverCandidate) bool {
	leftModePriority := autoRecoverModePriority(left.Mode)
	rightModePriority := autoRecoverModePriority(right.Mode)
	if leftModePriority != rightModePriority {
		return leftModePriority < rightModePriority
	}
	leftClassPriority := retryClassPriority(left.PrimaryRetryClass)
	rightClassPriority := retryClassPriority(right.PrimaryRetryClass)
	if leftClassPriority != rightClassPriority {
		return leftClassPriority < rightClassPriority
	}
	leftActionPriority := blockedActionPriority(left.EffectiveAction)
	rightActionPriority := blockedActionPriority(right.EffectiveAction)
	if leftActionPriority != rightActionPriority {
		return leftActionPriority < rightActionPriority
	}
	leftNextRetryAt := strings.TrimSpace(left.Detail.Runtime.NextRetryAt)
	rightNextRetryAt := strings.TrimSpace(right.Detail.Runtime.NextRetryAt)
	if leftNextRetryAt != rightNextRetryAt {
		if leftNextRetryAt == "" {
			return true
		}
		if rightNextRetryAt == "" {
			return false
		}
		return leftNextRetryAt < rightNextRetryAt
	}
	if left.Detail.Task.UpdatedAt != right.Detail.Task.UpdatedAt {
		return left.Detail.Task.UpdatedAt < right.Detail.Task.UpdatedAt
	}
	return left.Detail.Task.ID < right.Detail.Task.ID
}

func effectiveBlockedAction(detail Detail) string {
	action := strings.TrimSpace(detail.Runtime.BlockedAction)
	if action != "" {
		return action
	}
	summary := summarizeRetryQueue(detail.Runtime.RetryQueue)
	action = strings.TrimSpace(summary.BlockedAction)
	if action != "" {
		return action
	}
	if detail.Task.State == StateBlocked && retryQueueContainsClass(detail.Runtime.RetryQueue, "rate_limited") {
		return "wait_for_cooldown"
	}
	if autoRecoverReason(detail) == "cooldown_elapsed_auto_retry" && retryQueueContainsClass(detail.Runtime.RetryQueue, "rate_limited") {
		return "wait_for_cooldown"
	}
	return ""
}

func (s *Service) transitionState(ctx context.Context, id string, allowed []State, target State) (Detail, bool, error) {
	detail, ok, err := getTask(ctx, s.store, id)
	if err != nil || !ok {
		return Detail{}, ok, err
	}
	permitted := false
	for _, state := range allowed {
		if detail.Task.State == state {
			permitted = true
			break
		}
	}
	if !permitted {
		return Detail{}, true, fmt.Errorf("task_state_transition_not_allowed")
	}
	detail.Task.State = target
	detail.Task.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	ensureRuntimeState(&detail)
	switch target {
	case StatePaused:
		detail.Runtime.ExecutionState = "paused"
		clearPauseRequest(&detail.Runtime)
	case StateReady:
		detail.Runtime.BlockedReason = ""
		detail.Runtime.BlockedAction = ""
		detail.Runtime.BlockedAdvice = ""
		detail.Runtime.NextRetryAt = ""
		clearPauseRequest(&detail.Runtime)
		if len(detail.Results) > 0 {
			detail.Runtime.ExecutionState = "ready_to_resume"
		} else {
			detail.Runtime.ExecutionState = "idle"
		}
	}
	if err := updateTaskDetailState(ctx, s.store, detail); err != nil {
		return Detail{}, true, err
	}
	return detail, true, nil
}

func (s *Service) taskPauseRequested(ctx context.Context, id string) (bool, error) {
	detail, ok, err := getTask(ctx, s.store, id)
	if err != nil || !ok {
		return false, err
	}
	return detail.Task.State == StatePaused || detail.Runtime.PauseRequested, nil
}

func clearPauseRequest(runtime *RuntimeState) {
	if runtime == nil {
		return
	}
	runtime.PauseRequested = false
	runtime.PauseRequestedAt = ""
	runtime.PauseRequestSource = ""
}

func resolveConflictPolicy(meta provider.Provider, requested provider.ConflictPolicy) (provider.ConflictPolicy, string) {
	if requested == "" {
		return provider.ConflictPolicyAutoRenameNew, ""
	}
	if requested != provider.ConflictPolicyOverwriteExisting {
		return requested, ""
	}
	if meta.SupportsOverwrite {
		return requested, ""
	}
	if meta.SupportsAutoRename {
		return provider.ConflictPolicyAutoRenameNew, "downgrade_to_auto_rename"
	}
	return requested, ""
}

func (s *Service) executeUpload(entry provider.Entry, req provider.UploadRequest) (provider.UploadResult, bool, map[string]interface{}) {
	fastCheckPayload := map[string]interface{}(nil)
	if req.Strategy == string(planner.StrategyDownloadUpload) {
		if !localFileExists(req.LocalPath) {
			return provider.UploadResult{
				OperationResult: provider.OperationResult{
					Status:  "local_file_missing",
					Message: "Local file is required for download_upload fallback.",
					Mode:    "runtime_guard",
				},
			}, false, fastCheckPayload
		}
	}

	if req.Strategy == string(planner.StrategyFastUpload) && entry.Capability.SupportsFastUpload {
		check := entry.Adapter.FastUploadCheck(provider.FastUploadCheckRequest{
			Profile:  req.Profile,
			Path:     req.Path,
			ParentID: req.ParentID,
			Name:     req.Name,
			Size:     req.Size,
			MD5:      req.MD5,
			SHA1:     req.SHA1,
			GCID:     req.GCID,
		})
		fastCheckPayload = map[string]interface{}{
			"performed": check.OK || check.Status != "",
			"candidate": check.Candidate,
			"status":    check.Status,
			"message":   check.Message,
			"mode":      check.Mode,
		}
		if !check.OK {
			return provider.UploadResult{OperationResult: check.OperationResult}, false, fastCheckPayload
		}
		if !check.Candidate {
			if !supportsFallback(entry.Meta.FallbackModes, string(planner.StrategyDownloadUpload)) {
				return provider.UploadResult{
					OperationResult: provider.OperationResult{
						Status:  "hash_miss",
						Message: "Fast upload pre-check reported no candidate.",
						Mode:    "runtime_fast_check",
					},
				}, false, fastCheckPayload
			}
			if !localFileExists(req.LocalPath) {
				return provider.UploadResult{
					OperationResult: provider.OperationResult{
						Status:  "local_file_missing",
						Message: "Fast upload pre-check fallback requires a local file.",
						Mode:    "runtime_guard",
					},
				}, false, fastCheckPayload
			}

			fallbackReq := req
			fallbackReq.Strategy = string(planner.StrategyDownloadUpload)
			fallback := entry.Adapter.Upload(fallbackReq)
			if fallback.OK {
				if fallback.Payload == nil {
					fallback.Payload = map[string]interface{}{}
				}
				fallback.Payload["fallbackFrom"] = "fast_upload_precheck"
				fallback.Message = "Fast upload pre-check reported no candidate, fallback to download_upload succeeded."
			}
			return fallback, true, fastCheckPayload
		}
	}

	upload := entry.Adapter.Upload(req)
	if upload.OK {
		return upload, false, fastCheckPayload
	}
	if req.Strategy != string(planner.StrategyFastUpload) {
		return upload, false, fastCheckPayload
	}
	if upload.Status != "hash_miss" {
		return upload, false, fastCheckPayload
	}
	if !supportsFallback(entry.Meta.FallbackModes, string(planner.StrategyDownloadUpload)) {
		return upload, false, fastCheckPayload
	}
	if !localFileExists(req.LocalPath) {
		return provider.UploadResult{
			OperationResult: provider.OperationResult{
				Status:  "local_file_missing",
				Message: "Hash miss fallback requires a local file.",
				Mode:    "runtime_guard",
			},
		}, false, fastCheckPayload
	}

	fallbackReq := req
	fallbackReq.Strategy = string(planner.StrategyDownloadUpload)
	fallback := entry.Adapter.Upload(fallbackReq)
	if fallback.OK {
		if fallback.Payload == nil {
			fallback.Payload = map[string]interface{}{}
		}
		fallback.Payload["fallbackFrom"] = string(planner.StrategyFastUpload)
		fallback.Message = "Fast upload hash miss, fallback to download_upload succeeded."
	}
	return fallback, true, fastCheckPayload
}

func supportsFallback(modes []string, expected string) bool {
	for _, mode := range modes {
		if mode == expected {
			return true
		}
	}
	return false
}

func (s *Service) applyRuntimeThrottle(ctx context.Context, riskProfile planner.RiskProfile, previousPath string, currentPath string) (map[string]interface{}, error) {
	if strings.TrimSpace(previousPath) == "" {
		return nil, nil
	}
	requestInterval := riskProfile.RequestIntervalMS
	directoryInterval := 0
	if parentDirectory(previousPath) != parentDirectory(currentPath) {
		directoryInterval = riskProfile.DirectoryIntervalMS
	}
	waitMS := maxInt(requestInterval, directoryInterval)
	if waitMS <= 0 {
		return nil, nil
	}
	if err := s.sleepForThrottle(ctx, time.Duration(waitMS)*time.Millisecond); err != nil {
		return nil, err
	}
	evidence := map[string]interface{}{
		"waitMs":            waitMS,
		"requestIntervalMs": requestInterval,
		"previousPath":      previousPath,
		"currentPath":       currentPath,
	}
	if directoryInterval > 0 {
		evidence["directoryIntervalMs"] = directoryInterval
		evidence["directoryChanged"] = true
	}
	return evidence, nil
}

func (s *Service) sleepForThrottle(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}
	if s.throttleSleep != nil {
		return s.throttleSleep(ctx, delay)
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func completionKindFromResults(results []Result) CompletionKind {
	if len(results) == 0 {
		return CompletionKindProbeOnly
	}
	hasSuccess := false
	for _, item := range results {
		if item.Status != "done" {
			continue
		}
		hasSuccess = true
		if strategy := stringValue(item.Payload["strategy"]); strategy != "" && strategy != string(planner.StrategyPendingManual) {
			return CompletionKindRealTransfer
		}
		if providerStatus := stringValue(item.Payload["providerStatus"]); providerStatus == "ok" {
			return CompletionKindRealTransfer
		}
	}
	if hasSuccess {
		return CompletionKindCandidateOnly
	}
	return CompletionKindProbeOnly
}

func copyPayloadMap(values map[string]interface{}) map[string]interface{} {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]interface{}, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}

func localFileExists(path string) bool {
	if path == "" {
		return false
	}
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func (s *Service) resolveSourceProfile(ctx context.Context, providerKey, profileID string) (provider.AuthProfile, provider.Entry, error) {
	entry, exists := s.registry.Get(providerKey)
	if !exists {
		return provider.AuthProfile{}, provider.Entry{}, fmt.Errorf("provider_not_found")
	}
	profile, ok, err := s.authSvc.GetProfile(ctx, profileID)
	if err != nil {
		return provider.AuthProfile{}, provider.Entry{}, err
	}
	if !ok {
		return provider.AuthProfile{}, provider.Entry{}, fmt.Errorf("source_profile_not_found")
	}
	return provider.AuthProfile{
		ID:          profile.ID,
		ProviderKey: profile.ProviderKey,
		AuthMode:    profile.AuthMode,
		DisplayName: profile.DisplayName,
		Token:       profile.Token,
		Cookie:      profile.Cookie,
		Extra:       profile.Extra,
	}, entry, nil
}

func (s *Service) collectLeafFirstEntries(ctx context.Context, source provider.Entry, profile provider.AuthProfile, roots []string, pageSize int) ([]planner.SourceEntry, []string, error) {
	entries := make([]planner.SourceEntry, 0)
	trace := make([]string, 0)
	visited := make(map[string]bool)

	var walk func(string) error
	walk = func(path string) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		normalized := normalizeScanPath(path)
		if visited[normalized] {
			return nil
		}
		visited[normalized] = true
		trace = append(trace, normalized)

		list := source.Adapter.List(provider.ListRequest{
			Profile:  profile,
			Path:     normalized,
			ParentID: "",
			PageSize: pageSize,
		})
		if !list.OK {
			return fmt.Errorf("%s", list.Status)
		}

		dirs := make([]string, 0)
		files := make([]planner.SourceEntry, 0)
		for _, raw := range list.Items {
			childPath := normalizeScanPath(stringMapValue(raw, "path"))
			if childPath == "" || childPath == "/" {
				continue
			}
			if boolMapValue(raw, "isDir") {
				dirs = append(dirs, childPath)
				continue
			}
			files = append(files, planner.SourceEntry{
				Path:      childPath,
				Size:      int64Number(raw["size"]),
				MD5:       firstString(raw, "md5", "etag"),
				SHA1:      stringMapValue(raw, "sha1"),
				GCID:      stringMapValue(raw, "gcid"),
				ETag:      stringMapValue(raw, "etag"),
				LocalPath: stringMapValue(raw, "localPath"),
				Raw:       raw,
			})
		}
		for _, dir := range dirs {
			if err := walk(dir); err != nil {
				return err
			}
		}
		entries = append(entries, files...)
		return nil
	}

	for _, root := range roots {
		if err := walk(root); err != nil {
			return nil, trace, err
		}
	}
	return entries, trace, nil
}

func (s *Service) collectPreScannedEntries(ctx context.Context, source provider.Entry, profile provider.AuthProfile, roots []string, pageSize int) ([]planner.SourceEntry, []string, error) {
	entries := make([]planner.SourceEntry, 0)
	trace := make([]string, 0)

	var walk func(string) error
	walk = func(path string) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		normalized := normalizeScanPath(path)
		trace = append(trace, normalized)
		list := source.Adapter.List(provider.ListRequest{
			Profile:  profile,
			Path:     normalized,
			ParentID: "",
			PageSize: pageSize,
		})
		if !list.OK {
			return fmt.Errorf("%s", list.Status)
		}
		for _, raw := range list.Items {
			childPath := normalizeScanPath(stringMapValue(raw, "path"))
			if childPath == "" || childPath == "/" {
				continue
			}
			if boolMapValue(raw, "isDir") {
				if err := walk(childPath); err != nil {
					return err
				}
				continue
			}
			entries = append(entries, planner.SourceEntry{
				Path:      childPath,
				Size:      int64Number(raw["size"]),
				MD5:       firstString(raw, "md5", "etag"),
				SHA1:      stringMapValue(raw, "sha1"),
				GCID:      stringMapValue(raw, "gcid"),
				ETag:      stringMapValue(raw, "etag"),
				LocalPath: stringMapValue(raw, "localPath"),
				Raw:       raw,
			})
		}
		return nil
	}

	for _, root := range roots {
		if err := walk(root); err != nil {
			return nil, trace, err
		}
	}
	return entries, trace, nil
}

func (s *Service) collectEntriesByMode(ctx context.Context, mode planner.ExecutionMode, source provider.Entry, profile provider.AuthProfile, roots []string, pageSize int) ([]planner.SourceEntry, []string, error) {
	switch mode {
	case planner.ExecutionModePreScanFlat:
		return s.collectPreScannedEntries(ctx, source, profile, roots, pageSize)
	default:
		return s.collectLeafFirstEntries(ctx, source, profile, roots, pageSize)
	}
}

func normalizeScanPath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "/"
	}
	trimmed = strings.ReplaceAll(trimmed, "\\", "/")
	if !strings.HasPrefix(trimmed, "/") {
		trimmed = "/" + trimmed
	}
	for strings.Contains(trimmed, "//") {
		trimmed = strings.ReplaceAll(trimmed, "//", "/")
	}
	if len(trimmed) > 1 && strings.HasSuffix(trimmed, "/") {
		trimmed = strings.TrimRight(trimmed, "/")
	}
	return trimmed
}

func metadataStringSlice(values map[string]interface{}, key string) []string {
	if values == nil {
		return nil
	}
	raw, ok := values[key]
	if !ok {
		return nil
	}
	switch typed := raw.(type) {
	case []string:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			item = strings.TrimSpace(item)
			if item != "" {
				out = append(out, item)
			}
		}
		return out
	case []interface{}:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if value, ok := item.(string); ok && strings.TrimSpace(value) != "" {
				out = append(out, strings.TrimSpace(value))
			}
		}
		return out
	default:
		return nil
	}
}

func metadataStringFromRisk(values map[string]interface{}, key string) string {
	if values == nil {
		return ""
	}
	value, _ := values[key].(string)
	return value
}

func riskOverrideFromMetadata(values map[string]interface{}) *planner.RiskProfileOverride {
	if values == nil {
		return nil
	}
	raw, ok := values["riskOverride"]
	if !ok || raw == nil {
		return nil
	}
	switch typed := raw.(type) {
	case *planner.RiskProfileOverride:
		return typed
	case planner.RiskProfileOverride:
		override := typed
		return &override
	case map[string]interface{}:
		override := planner.RiskProfileOverride{}
		if value, ok := intPointerFromRaw(typed["requestIntervalMs"]); ok {
			override.RequestIntervalMS = value
		}
		if value, ok := intPointerFromRaw(typed["pageSize"]); ok {
			override.PageSize = value
		}
		if value, ok := intPointerFromRaw(typed["directoryIntervalMs"]); ok {
			override.DirectoryIntervalMS = value
		}
		if value, ok := intPointerFromRaw(typed["cooldownSeconds"]); ok {
			override.CooldownSeconds = value
		}
		if value, ok := intPointerFromRaw(typed["retryLimit"]); ok {
			override.RetryLimit = value
		}
		if value, ok := intPointerFromRaw(typed["maxConcurrent"]); ok {
			override.MaxConcurrent = value
		}
		if value, ok := intPointerFromRaw(typed["autoRetryStartHour"]); ok {
			override.AutoRetryStartHour = value
		}
		if value, ok := intPointerFromRaw(typed["autoRetryEndHour"]); ok {
			override.AutoRetryEndHour = value
		}
		override.RiskKeywords = metadataStringSlice(map[string]interface{}{"keywords": typed["riskKeywords"]}, "keywords")
		if override.RequestIntervalMS == nil && override.PageSize == nil && override.DirectoryIntervalMS == nil && override.CooldownSeconds == nil && override.RetryLimit == nil && override.MaxConcurrent == nil && override.AutoRetryStartHour == nil && override.AutoRetryEndHour == nil && len(override.RiskKeywords) == 0 {
			return nil
		}
		return &override
	default:
		return nil
	}
}

func executionModeFromMetadata(values map[string]interface{}) (planner.ExecutionMode, error) {
	if values == nil {
		return planner.ExecutionModeLeafFirstLazy, nil
	}
	switch raw := values["executionMode"].(type) {
	case planner.ExecutionMode:
		return raw, nil
	case string:
		return planner.ExecutionMode(raw), nil
	default:
		return planner.ExecutionModeLeafFirstLazy, nil
	}
}

func executionModeString(values map[string]interface{}) string {
	mode, err := executionModeFromMetadata(values)
	if err != nil {
		return ""
	}
	return string(mode)
}

func scanModeForExecutionMode(mode planner.ExecutionMode) string {
	switch mode {
	case planner.ExecutionModePreScanFlat:
		return "pre_scan_flat"
	default:
		return "lazy_leaf_first"
	}
}

func stringMapValue(values map[string]interface{}, key string) string {
	value, _ := values[key].(string)
	return value
}

func boolMapValue(values map[string]interface{}, key string) bool {
	value, _ := values[key].(bool)
	return value
}

func firstString(values map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if value := stringMapValue(values, key); value != "" {
			return value
		}
	}
	return ""
}

func intNumber(raw interface{}) int {
	switch value := raw.(type) {
	case int:
		return value
	case int64:
		return int(value)
	case float64:
		return int(value)
	default:
		return 0
	}
}

func int64Number(raw interface{}) int64 {
	switch value := raw.(type) {
	case int:
		return int64(value)
	case int64:
		return value
	case float64:
		return int64(value)
	default:
		return 0
	}
}

func intPointerFromRaw(raw interface{}) (*int, bool) {
	switch value := raw.(type) {
	case int:
		v := value
		return &v, true
	case int64:
		v := int(value)
		return &v, true
	case float64:
		v := int(value)
		return &v, true
	default:
		return nil, false
	}
}

func initializeRuntimeState(plan planner.Plan) RuntimeState {
	directoryStates := collectDirectoryStates(plan)
	return RuntimeState{
		ExecutionState:  "idle",
		ProcessedCount:  0,
		DoneCount:       0,
		SkippedCount:    0,
		FailedCount:     0,
		PendingCount:    0,
		RiskHitCount:    0,
		NextSequence:    1,
		DirectoryStates: directoryStates,
	}
}

func ensureRuntimeState(detail *Detail) {
	if detail == nil {
		return
	}
	if detail.Runtime.ExecutionState == "" &&
		detail.Runtime.NextSequence == 0 &&
		detail.Runtime.ProcessedCount == 0 &&
		detail.Runtime.DoneCount == 0 &&
		detail.Runtime.SkippedCount == 0 &&
		detail.Runtime.FailedCount == 0 &&
		detail.Runtime.PendingCount == 0 &&
		detail.Runtime.RiskHitCount == 0 &&
		len(detail.Runtime.PendingTree) == 0 &&
		len(detail.Runtime.DirectoryStates) == 0 {
		detail.Runtime = initializeRuntimeState(detail.Plan)
	}
	if detail.Runtime.ExecutionState == "" {
		detail.Runtime.ExecutionState = "idle"
	}
	if detail.Runtime.NextSequence <= 0 {
		detail.Runtime.NextSequence = len(detail.Results) + 1
	}
}

func collectDirectoryStates(plan planner.Plan) []DirectoryState {
	ordered := make([]DirectoryState, 0)
	indexByPath := make(map[string]int)
	selectedRoots := metadataStringSlice(plan.Metadata, "selectedRoots")
	scanTrace := metadataStringSlice(plan.Metadata, "scanTrace")

	addPath := func(path string) {
		path = normalizeScanPath(path)
		if path == "/" {
			return
		}
		if _, exists := indexByPath[path]; exists {
			return
		}
		root := matchRootPath(path, selectedRoots)
		indexByPath[path] = len(ordered)
		ordered = append(ordered, DirectoryState{
			Path:     path,
			RootPath: root,
			Status:   "pending",
		})
	}

	for _, root := range selectedRoots {
		addPath(root)
	}
	for _, path := range scanTrace {
		addPath(path)
	}
	for _, item := range plan.Items {
		addPath(parentDirectory(item.Path))
	}

	for idx := range ordered {
		total := 0
		for _, item := range plan.Items {
			if itemBelongsToDirectory(item.Path, ordered[idx].Path) {
				total++
			}
		}
		ordered[idx].TotalItems = total
	}
	return ordered
}

func syncRuntimeCountsFromResults(runtime *RuntimeState, results []Result) {
	if runtime == nil {
		return
	}
	runtime.ProcessedCount = len(results)
	runtime.DoneCount = 0
	runtime.SkippedCount = 0
	runtime.FailedCount = 0
	runtime.PendingCount = 0
	runtime.RiskHitCount = 0
	runtime.LastRiskStatus = ""
	runtime.BlockedReason = ""
	runtime.BlockedAction = ""
	runtime.BlockedAdvice = ""
	runtime.NextRetryAt = ""
	runtime.RiskHits = nil
	runtime.PendingTree = nil
	runtime.RetryQueue = nil
	runtime.UploadCheckpoint = nil
	runtime.RetryableCount = 0
	runtime.BlockedRetryCount = 0
	lastCompleted := ""
	for _, result := range results {
		switch result.Status {
		case "done":
			runtime.DoneCount++
		case "skipped":
			runtime.SkippedCount++
		case "failed":
			runtime.FailedCount++
		}
		if path, _ := result.Payload["path"].(string); path != "" {
			lastCompleted = path
		}
		if isPendingRelayResult(result) {
			runtime.PendingCount++
		}
		if riskHit, ok := riskHitFromPayload(result.Payload); ok {
			runtime.RiskHitCount++
			runtime.LastRiskStatus = riskHit.Status
			runtime.RiskHits = append(runtime.RiskHits, riskHit)
		}
	}
	runtime.LastCompletedPath = lastCompleted
	runtime.NextSequence = len(results) + 1
}

func updateRuntimeBeforeItem(detail *Detail, path string, index int) {
	ensureRuntimeState(detail)
	currentRoot := matchRootPath(path, metadataStringSlice(detail.Plan.Metadata, "selectedRoots"))
	currentDir := parentDirectory(path)
	detail.Runtime.ExecutionState = "running"
	detail.Runtime.CurrentRoot = currentRoot
	detail.Runtime.CurrentDirectory = currentDir
	detail.Runtime.CurrentItemPath = path
	detail.Runtime.NextSequence = index + 1
	setDirectoryInProgress(&detail.Runtime, currentRoot, path)
	setDirectoryInProgress(&detail.Runtime, currentDir, path)
}

func updateRuntimeAfterItem(detail *Detail, path string, result Result) {
	ensureRuntimeState(detail)
	result.Payload["path"] = path
	detail.Runtime.ProcessedCount++
	detail.Runtime.LastCompletedPath = path
	detail.Runtime.CurrentItemPath = ""
	detail.Runtime.NextSequence = detail.Runtime.ProcessedCount + 1
	if result.Status == "done" {
		detail.Runtime.DoneCount++
	} else if result.Status == "skipped" {
		detail.Runtime.SkippedCount++
	} else if result.Status == "failed" {
		detail.Runtime.FailedCount++
	}
	currentRoot := matchRootPath(path, metadataStringSlice(detail.Plan.Metadata, "selectedRoots"))
	currentDir := parentDirectory(path)
	applyDirectoryResult(&detail.Runtime, currentRoot, path, result.Status)
	applyDirectoryResult(&detail.Runtime, currentDir, path, result.Status)
}

func setDirectoryInProgress(runtime *RuntimeState, dirPath, itemPath string) {
	if runtime == nil {
		return
	}
	index := findDirectoryState(runtime.DirectoryStates, dirPath)
	if index < 0 {
		return
	}
	state := runtime.DirectoryStates[index]
	if state.Status == "completed" || state.Status == "completed_with_errors" {
		runtime.DirectoryStates[index] = state
		return
	}
	state.Status = "in_progress"
	state.LastItemPath = itemPath
	runtime.DirectoryStates[index] = state
}

func applyDirectoryResult(runtime *RuntimeState, dirPath, itemPath, resultStatus string) {
	if runtime == nil {
		return
	}
	index := findDirectoryState(runtime.DirectoryStates, dirPath)
	if index < 0 {
		return
	}
	state := runtime.DirectoryStates[index]
	state.ProcessedItems++
	state.LastItemPath = itemPath
	if resultStatus == "done" {
		state.DoneItems++
	} else if resultStatus == "skipped" {
		state.SkippedItems++
	} else if resultStatus == "failed" {
		state.FailedItems++
	}
	state.Status = "in_progress"
	if state.TotalItems > 0 && state.ProcessedItems >= state.TotalItems {
		if state.FailedItems > 0 {
			state.Status = "completed_with_errors"
		} else {
			state.Status = "completed"
		}
	}
	runtime.DirectoryStates[index] = state
}

func findDirectoryState(states []DirectoryState, path string) int {
	path = normalizeScanPath(path)
	for idx, state := range states {
		if normalizeScanPath(state.Path) == path {
			return idx
		}
	}
	return -1
}

func itemBelongsToDirectory(itemPath, dirPath string) bool {
	dirPath = normalizeScanPath(dirPath)
	itemPath = normalizeScanPath(itemPath)
	if dirPath == "/" {
		return true
	}
	if itemPath == dirPath {
		return true
	}
	return strings.HasPrefix(itemPath, dirPath+"/")
}

func parentDirectory(path string) string {
	normalized := normalizeScanPath(path)
	if normalized == "/" {
		return "/"
	}
	index := strings.LastIndex(normalized, "/")
	if index <= 0 {
		return "/"
	}
	return normalized[:index]
}

func matchRootPath(path string, roots []string) string {
	normalized := normalizeScanPath(path)
	for _, root := range roots {
		root = normalizeScanPath(root)
		if normalized == root || strings.HasPrefix(normalized, root+"/") {
			return root
		}
	}
	if len(roots) == 1 {
		return normalizeScanPath(roots[0])
	}
	return parentDirectory(normalized)
}

func targetEntryExists(metadata provider.MetadataResult) bool {
	if metadata.Status == "exists" {
		return true
	}
	if metadata.Entry == nil {
		return false
	}
	if exists, ok := metadata.Entry["exists"].(bool); ok {
		return exists
	}
	return false
}

func sourceFingerprintForPath(entries []planner.SourceEntry, path string, fallbackSize int64) map[string]interface{} {
	for _, item := range entries {
		if item.Path != path {
			continue
		}
		return map[string]interface{}{
			"path": item.Path,
			"size": firstNonZeroInt64(item.Size, fallbackSize),
			"md5":  firstNonEmpty(item.MD5, item.ETag),
			"sha1": item.SHA1,
			"gcid": item.GCID,
			"etag": item.ETag,
		}
	}
	return map[string]interface{}{
		"path": path,
		"size": fallbackSize,
	}
}

func fingerprintFromMetadata(entry map[string]interface{}) map[string]interface{} {
	if entry == nil {
		return nil
	}
	return map[string]interface{}{
		"path": stringMapValue(entry, "path"),
		"size": int64Number(entry["size"]),
		"md5":  firstString(entry, "md5", "etag"),
		"sha1": stringMapValue(entry, "sha1"),
		"gcid": stringMapValue(entry, "gcid"),
		"etag": stringMapValue(entry, "etag"),
	}
}

func fingerprintsMatch(source map[string]interface{}, target map[string]interface{}) bool {
	if len(source) == 0 || len(target) == 0 {
		return false
	}
	sourceSize := int64Number(source["size"])
	targetSize := int64Number(target["size"])
	if sourceSize <= 0 || targetSize <= 0 || sourceSize != targetSize {
		return false
	}
	if valuesMatch(source["md5"], target["md5"]) {
		return true
	}
	if valuesMatch(source["sha1"], target["sha1"]) {
		return true
	}
	if valuesMatch(source["gcid"], target["gcid"]) {
		return true
	}
	return false
}

func valuesMatch(left interface{}, right interface{}) bool {
	leftValue := strings.TrimSpace(stringValue(left))
	rightValue := strings.TrimSpace(stringValue(right))
	return leftValue != "" && rightValue != "" && leftValue == rightValue
}

func firstNonZeroInt64(values ...int64) int64 {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func normalizeEvidenceReportTitle(title string) string {
	title = strings.TrimSpace(title)
	if title == "" {
		return "CloudPan Sync Go 验收与样本报告"
	}
	return title
}

func buildEvidenceReport(summary EvidenceSummary, statuses []StatusSummary, smokeSummaries []ProviderSmokeSummary, smokeMatrix []ProviderSmokeMatrixRow, samples []EvidenceSample, generatedAt string, title string, note string) EvidenceReport {
	title = normalizeEvidenceReportTitle(title)
	note = strings.TrimSpace(note)
	var b strings.Builder
	b.WriteString("# ")
	b.WriteString(title)
	b.WriteString("\n\n")
	b.WriteString("生成时间: ")
	b.WriteString(generatedAt)
	b.WriteString("\n\n")
	if note != "" {
		b.WriteString("备注: ")
		b.WriteString(note)
		b.WriteString("\n\n")
	}
	b.WriteString("## 运行证据摘要\n\n")
	fmt.Fprintf(&b, "- 总任务数: %d\n", summary.TotalTasks)
	fmt.Fprintf(&b, "- 已完成任务: %d\n", summary.CompletedTasks)
	fmt.Fprintf(&b, "- 阻塞任务: %d\n", summary.BlockedTasks)
	fmt.Fprintf(&b, "- 成功结果: %d\n", summary.DoneResultCount)
	fmt.Fprintf(&b, "- 跳过结果: %d\n", summary.SkippedResultCount)
	fmt.Fprintf(&b, "- 待补传结果: %d\n", summary.PendingResultCount)
	fmt.Fprintf(&b, "- 失败结果: %d\n", summary.FailedResultCount)
	fmt.Fprintf(&b, "- 风控命中: %d\n", summary.RiskHitCount)
	b.WriteString("\n## 阻塞动作\n\n")
	if len(summary.BlockedActions) == 0 {
		b.WriteString("- 当前没有需要人工处理的阻塞动作。\n")
	} else {
		b.WriteString("| 动作 | 任务数 | Provider 数 | 下一次可重试 | 建议 |\n")
		b.WriteString("| --- | ---: | ---: | --- | --- |\n")
		for _, item := range summary.BlockedActions {
			fmt.Fprintf(&b, "| %s | %d | %d | %s | %s |\n",
				markdownCell(item.Action),
				item.TaskCount,
				item.ProviderCount,
				markdownCell(firstNonEmpty(item.NextRetryAt, "-")),
				markdownCell(firstNonEmpty(item.Advice, "-")),
			)
		}
	}
	b.WriteString("\n## Provider 样本矩阵\n\n")
	if len(statuses) == 0 {
		b.WriteString("- 当前没有 provider 状态样本。\n")
	} else {
		b.WriteString("| Provider | Profiles | Tasks | Completed | Blocked | Latest Probe | Last Task State | Observed At |\n")
		b.WriteString("| --- | ---: | ---: | ---: | ---: | --- | --- | --- |\n")
		for _, item := range statuses {
			fmt.Fprintf(&b, "| %s | %d | %d | %d | %d | %s | %s | %s |\n",
				markdownCell(item.ProviderKey),
				item.ProfileCount,
				item.TaskCount,
				item.CompletedCount,
				item.BlockedCount,
				markdownCell(firstNonEmpty(item.LatestProbe, "-")),
				markdownCell(firstNonEmpty(item.LastTaskState, "-")),
				markdownCell(firstNonEmpty(item.LastObservedAt, "-")),
			)
		}
	}
	b.WriteString("\n## 真实样本矩阵\n\n")
	if len(smokeMatrix) == 0 {
		b.WriteString("- 当前没有真实样本矩阵数据。\n")
	} else {
		acceptedCount := 0
		inProgressCount := 0
		pendingCount := 0
		for _, item := range smokeMatrix {
			if item.Accepted {
				acceptedCount++
				continue
			}
			switch item.AcceptanceStatus {
			case "in_progress":
				inProgressCount++
			default:
				pendingCount++
			}
		}
		fmt.Fprintf(&b, "- 已验收协议组: %d / %d\n", acceptedCount, len(smokeMatrix))
		fmt.Fprintf(&b, "- 进行中协议组: %d\n", inProgressCount)
		fmt.Fprintf(&b, "- 待补齐协议组: %d\n", pendingCount)
		b.WriteString("\n## 真实联调验收\n\n")
		b.WriteString("| ProtocolGroup | Acceptance | Missing | Advice | Smoke | Coverage | Sample | Latest Smoke |\n")
		b.WriteString("| --- | --- | --- | --- | --- | --- | --- | --- |\n")
		for _, item := range smokeMatrix {
			fmt.Fprintf(&b, "| %s | %s | %s | %s | %d/%d | %d/%d/%d | %s / %s / %s | %s |\n",
				markdownCell(firstNonEmpty(item.ProtocolGroup, "-")),
				markdownCell(firstNonEmpty(item.AcceptanceStatus, "-")),
				markdownCell(strings.Join(item.AcceptanceMissing, ", ")),
				markdownCell(firstNonEmpty(item.AcceptanceAdvice, "-")),
				item.SuccessCount,
				item.FailureCount,
				item.CoverageRealSuccessTaskCount,
				item.CoverageTaskCount,
				item.CoverageCompletedTaskCount,
				markdownCell(firstNonEmpty(item.SampleTitle, "-")),
				markdownCell(firstNonEmpty(item.SampleProviderKey, "-")),
				markdownCell(firstNonEmpty(item.SampleCategory, "-")),
				markdownCell(firstNonEmpty(item.LatestSmokeAt, "-")),
			)
		}
	}
	b.WriteString("\n## 代表任务样本\n\n")
	if len(samples) == 0 {
		b.WriteString("- 当前没有可展示的任务样本。\n")
	} else {
		b.WriteString("| Provider | Task | State | Completion | ExecMode | ScanMode | Selected Roots | Scan Trace | BlockedReason | Last Path |\n")
		b.WriteString("| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |\n")
		for _, item := range samples {
			fmt.Fprintf(&b, "| %s | %s | %s | %s | %s | %s | %s | %s | %s | %s |\n",
				markdownCell(firstNonEmpty(item.ProviderKey, "-")),
				markdownCell(firstNonEmpty(item.TaskID, "-")),
				markdownCell(firstNonEmpty(item.TaskState, "-")),
				markdownCell(firstNonEmpty(item.CompletionKind, "-")),
				markdownCell(firstNonEmpty(item.ExecutionMode, "-")),
				markdownCell(firstNonEmpty(item.ScanMode, "-")),
				markdownCell(strings.Join(item.SelectedRoots, " -> ")),
				markdownCell(strings.Join(item.ScanTrace, " -> ")),
				markdownCell(firstNonEmpty(item.BlockedReason, "-")),
				markdownCell(firstNonEmpty(item.LastCompletedPath, "-")),
			)
		}
	}
	b.WriteString("\n## 最近证据\n\n")
	if len(summary.RecentProbes) == 0 {
		b.WriteString("- 暂无 recent probe。\n")
	} else {
		for _, probe := range summary.RecentProbes {
			fmt.Fprintf(&b, "- %s %s %s %s\n",
				markdownCell(firstNonEmpty(probe.ProviderKey, "-")),
				markdownCell(firstNonEmpty(probe.Status, "-")),
				markdownCell(firstNonEmpty(probe.CreatedAt, "-")),
				markdownCell(firstNonEmpty(stringValue(probe.Payload["taskState"]), "-")),
			)
		}
	}
	return EvidenceReport{
		GeneratedAt:    generatedAt,
		Title:          title,
		Note:           note,
		Markdown:       strings.TrimSpace(b.String()),
		Summary:        summary,
		Statuses:       statuses,
		SmokeSummaries: smokeSummaries,
		SmokeMatrix:    smokeMatrix,
		Samples:        samples,
	}
}

func buildEvidenceSamples(details []Detail, limit int) []EvidenceSample {
	if limit <= 0 {
		limit = 10
	}
	samples := make([]EvidenceSample, 0, minInt(limit, len(details)))
	for _, detail := range details {
		selectedRoots := metadataStringSlice(detail.Plan.Metadata, "selectedRoots")
		scanTrace := metadataStringSlice(detail.Plan.Metadata, "scanTrace")
		samples = append(samples, EvidenceSample{
			ProviderKey:       detail.Task.TargetProvider,
			TaskID:            detail.Task.ID,
			SourceProvider:    detail.Task.SourceProvider,
			TargetProvider:    detail.Task.TargetProvider,
			TaskState:         string(detail.Task.State),
			CompletionKind:    string(detail.Task.CompletionKind),
			ExecutionMode:     executionModeString(detail.Plan.Metadata),
			ScanMode:          stringValue(detail.Plan.Metadata["scanMode"]),
			SelectedRoots:     selectedRoots,
			ScanTrace:         scanTrace,
			BlockedReason:     stringValue(detail.Runtime.BlockedReason),
			LastCompletedPath: detail.Runtime.LastCompletedPath,
			ResultCount:       len(detail.Results),
			CreatedAt:         detail.Task.CreatedAt,
		})
		if len(samples) >= limit {
			break
		}
	}
	return samples
}

func markdownCell(value string) string {
	value = strings.ReplaceAll(strings.TrimSpace(value), "|", "\\|")
	if value == "" {
		return "-"
	}
	return value
}

func inferProviderSmokeCategory(record ProviderSmokeRecord) string {
	if !strings.EqualFold(record.Result, "success") {
		return "failed"
	}
	ops := make([]string, 0, len(record.Operations))
	for _, op := range record.Operations {
		ops = append(ops, strings.TrimSpace(strings.ToLower(op)))
	}
	contains := func(name string) bool {
		for _, op := range ops {
			if op == strings.ToLower(name) {
				return true
			}
		}
		return false
	}
	switch {
	case contains("upload"):
		return "binary_upload_success"
	case contains("fastuploadcheck"):
		return "fast_upload_success"
	case contains("createdir") || contains("metadata") || contains("list"):
		return "browse_only"
	case contains("validateauth"):
		return "auth_only"
	default:
		return "partial_blocked"
	}
}

func buildProviderSmokeMarkdown(record ProviderSmokeRecord) string {
	var b strings.Builder
	title := strings.TrimSpace(record.Title)
	if title == "" {
		title = record.ProviderKey + " 真实 smoke 记录"
	}
	b.WriteString("# ")
	b.WriteString(title)
	b.WriteString("\n\n")
	b.WriteString("## 基本信息\n\n")
	fmt.Fprintf(&b, "- Provider: %s\n", markdownCell(record.ProviderKey))
	fmt.Fprintf(&b, "- 协议组: %s\n", markdownCell(firstNonEmpty(record.ProtocolGroup, "-")))
	fmt.Fprintf(&b, "- 认证方式: %s\n", markdownCell(firstNonEmpty(record.AuthMode, "-")))
	fmt.Fprintf(&b, "- 结果分类: %s\n", markdownCell(firstNonEmpty(record.Category, "-")))
	fmt.Fprintf(&b, "- 结果: %s\n", markdownCell(firstNonEmpty(record.Result, "-")))
	fmt.Fprintf(&b, "- 记录时间: %s\n", markdownCell(firstNonEmpty(record.CreatedAt, "-")))
	if record.Note != "" {
		fmt.Fprintf(&b, "- 备注: %s\n", markdownCell(record.Note))
	}
	b.WriteString("\n## 本次覆盖范围\n\n")
	if len(record.Operations) == 0 {
		b.WriteString("- 未记录具体操作。\n")
	} else {
		for _, op := range record.Operations {
			fmt.Fprintf(&b, "- %s\n", markdownCell(op))
		}
	}
	b.WriteString("\n## 认证信息摘要\n\n")
	fmt.Fprintf(&b, "- `authMode`: %s\n", markdownCell(firstNonEmpty(record.AuthMode, "-")))
	if len(record.Environment) == 0 {
		b.WriteString("- `extra` / 环境摘要：未填写\n")
	} else {
		keys := make([]string, 0, len(record.Environment))
		for key := range record.Environment {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		b.WriteString("- `extra` / 环境摘要：")
		b.WriteString(strings.Join(keys, ", "))
		b.WriteString("\n")
	}
	b.WriteString("\n## 异常与阻塞记录\n\n")
	if strings.EqualFold(record.Result, "success") {
		b.WriteString("- 本次未记录阻塞。\n")
	} else {
		b.WriteString("- 本次存在阻塞或失败，请结合控制台证据补充。\n")
	}
	b.WriteString("\n## 对代码的反推结论\n\n")
	b.WriteString("- 这条记录可以作为协议族或 provider 的真实联调样本。\n")
	b.WriteString("- 需要时可继续补充更细的请求 / 响应片段。\n")
	b.WriteString("\n## 验收结论\n\n")
	if strings.EqualFold(record.Result, "success") {
		b.WriteString("- 本次形成有效真实联调样本。\n")
	} else {
		b.WriteString("- 本次未形成完整成功样本，但仍保留排障证据。\n")
	}
	return strings.TrimSpace(b.String())
}

func summarizeProviderSmokeRecords(records []ProviderSmokeRecord) []ProviderSmokeSummary {
	if len(records) == 0 {
		return nil
	}
	type smokeGroupState struct {
		row          ProviderSmokeSummary
		providerSeen map[string]struct{}
	}
	states := make(map[string]*smokeGroupState)
	order := make([]string, 0)
	for _, record := range records {
		group := strings.TrimSpace(record.ProtocolGroup)
		if group == "" {
			group = strings.TrimSpace(record.ProviderKey)
		}
		state, ok := states[group]
		if !ok {
			state = &smokeGroupState{
				row: ProviderSmokeSummary{
					ProtocolGroup: group,
				},
				providerSeen: make(map[string]struct{}),
			}
			states[group] = state
			order = append(order, group)
		}
		state.row.SmokeCount++
		if strings.EqualFold(record.Result, "success") {
			state.row.SuccessCount++
		} else {
			state.row.FailureCount++
		}
		if _, ok := state.providerSeen[record.ProviderKey]; !ok {
			state.providerSeen[record.ProviderKey] = struct{}{}
			state.row.ProviderCount++
			state.row.ProviderKeys = append(state.row.ProviderKeys, record.ProviderKey)
		}
		if state.row.SampleRecordID == "" {
			state.row.SampleRecordID = record.ID
			state.row.SampleTitle = record.Title
			state.row.SampleProviderKey = record.ProviderKey
			state.row.SampleCategory = record.Category
			state.row.SampleResult = record.Result
			state.row.LatestSmokeAt = record.CreatedAt
		}
		if strings.EqualFold(record.Result, "success") {
			state.row.HasRealSuccessSample = true
		}
		if record.CreatedAt != "" && (state.row.LatestSmokeAt == "" || record.CreatedAt > state.row.LatestSmokeAt) {
			state.row.SampleRecordID = record.ID
			state.row.SampleTitle = record.Title
			state.row.SampleProviderKey = record.ProviderKey
			state.row.SampleCategory = record.Category
			state.row.SampleResult = record.Result
			state.row.LatestSmokeAt = record.CreatedAt
		}
	}
	rows := make([]ProviderSmokeSummary, 0, len(order))
	for _, group := range order {
		state := states[group]
		if len(state.row.ProviderKeys) == 0 {
			state.row.ProviderKeys = nil
		}
		rows = append(rows, state.row)
	}
	return rows
}

func buildProviderSmokeMatrix(summary EvidenceSummary, smokeSummaries []ProviderSmokeSummary) []ProviderSmokeMatrixRow {
	if len(smokeSummaries) == 0 && len(summary.ProtocolCoverage) == 0 {
		return nil
	}
	type matrixState struct {
		row ProviderSmokeMatrixRow
	}
	states := make(map[string]*matrixState)
	order := make([]string, 0)
	ensure := func(group string) *matrixState {
		group = strings.TrimSpace(group)
		if group == "" {
			group = "unknown"
		}
		state, ok := states[group]
		if ok {
			return state
		}
		state = &matrixState{row: ProviderSmokeMatrixRow{ProtocolGroup: group}}
		states[group] = state
		order = append(order, group)
		return state
	}
	for _, item := range smokeSummaries {
		state := ensure(item.ProtocolGroup)
		state.row.SmokeCount = item.SmokeCount
		state.row.SuccessCount = item.SuccessCount
		state.row.FailureCount = item.FailureCount
		state.row.ProviderCount = item.ProviderCount
		state.row.ProviderKeys = append([]string(nil), item.ProviderKeys...)
		state.row.SampleRecordID = item.SampleRecordID
		state.row.SampleTitle = item.SampleTitle
		state.row.SampleProviderKey = item.SampleProviderKey
		state.row.SampleCategory = item.SampleCategory
		state.row.SampleResult = item.SampleResult
		state.row.LatestSmokeAt = item.LatestSmokeAt
		state.row.HasRealSuccessSample = item.HasRealSuccessSample
	}
	for _, coverage := range summary.ProtocolCoverage {
		state := ensure(coverage.ProtocolGroup)
		state.row.CoverageTaskCount = coverage.TaskCount
		state.row.CoverageCompletedTaskCount = coverage.CompletedTaskCount
		state.row.CoverageRealSuccessTaskCount = coverage.RealSuccessTaskCount
		state.row.CoverageProviderCount = coverage.ProviderCount
		state.row.CoverageProviderKeys = append([]string(nil), coverage.ProviderKeys...)
		state.row.CoverageHasRealSuccessSample = coverage.HasRealSuccessSample
		state.row.CoverageSampleTaskID = coverage.SampleTaskID
		state.row.CoverageSampleProviderKey = coverage.SampleProviderKey
		state.row.CoverageSampleTaskState = coverage.SampleTaskState
		state.row.CoverageSampleCompletionKind = coverage.SampleCompletionKind
		state.row.CoverageLastObservedAt = coverage.LastObservedAt
	}
	for _, state := range states {
		missing := make([]string, 0, 2)
		if !state.row.HasRealSuccessSample {
			missing = append(missing, "real_smoke_success_missing")
		}
		if state.row.CoverageTaskCount == 0 {
			missing = append(missing, "task_coverage_missing")
		}
		switch {
		case state.row.HasRealSuccessSample && state.row.CoverageTaskCount > 0:
			state.row.Accepted = true
			state.row.AcceptanceStatus = "accepted"
			state.row.AcceptanceAdvice = "已具备真实 smoke 成功样本与任务覆盖，可继续补充更多边界样本。"
		case state.row.HasRealSuccessSample || state.row.CoverageHasRealSuccessSample:
			state.row.AcceptanceStatus = "in_progress"
			state.row.AcceptanceAdvice = buildAcceptanceAdvice(missing, true)
		default:
			state.row.AcceptanceStatus = "pending"
			state.row.AcceptanceAdvice = buildAcceptanceAdvice(missing, false)
		}
		if len(missing) > 0 {
			state.row.AcceptanceMissing = missing
		}
	}
	rows := make([]ProviderSmokeMatrixRow, 0, len(order))
	for _, group := range order {
		rows = append(rows, states[group].row)
	}
	return rows
}

func buildAcceptanceAdvice(missing []string, partial bool) string {
	if len(missing) == 0 {
		if partial {
			return "当前协议组已具备部分验收条件，建议继续补充另一侧样本。"
		}
		return "当前协议组已具备验收条件。"
	}
	hints := make([]string, 0, len(missing))
	for _, item := range missing {
		switch item {
		case "real_smoke_success_missing":
			hints = append(hints, "补 1 条真实 smoke 成功样本")
		case "task_coverage_missing":
			hints = append(hints, "补 1 条真实任务覆盖样本")
		default:
			hints = append(hints, item)
		}
	}
	return "建议：" + strings.Join(hints, "；")
}

func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}

func applyRiskEvidence(runtime *RuntimeState, metadata map[string]interface{}, path string, result Result) {
	if runtime == nil || result.Payload == nil {
		return
	}
	riskProfile := riskProfileFromRaw(result.Payload["riskProfile"])
	if len(riskProfile.RiskKeywords) == 0 {
		riskProfile = riskProfileFromMetadata(metadata)
	}
	riskHit, ok := detectRiskHit(riskProfile, result, path)
	if !ok {
		return
	}
	result.Payload["riskHit"] = riskHit
	runtime.RiskHitCount++
	runtime.LastRiskStatus = riskHit.Status
	runtime.RiskHits = append(runtime.RiskHits, riskHit)
}

func pendingRetryEntries(detail Detail) ([]planner.SourceEntry, []string) {
	pendingPaths := pendingRetryPaths(detail.Results)
	if len(pendingPaths) == 0 {
		return nil, nil
	}
	entryByPath := make(map[string]planner.SourceEntry, len(detail.SourceEntries))
	for _, entry := range detail.SourceEntries {
		entryByPath[normalizeScanPath(entry.Path)] = entry
	}
	sizeByPath := make(map[string]int64, len(detail.Plan.Items))
	for _, item := range detail.Plan.Items {
		sizeByPath[normalizeScanPath(item.Path)] = item.Size
	}
	filtered := make([]planner.SourceEntry, 0, len(pendingPaths))
	for _, path := range pendingPaths {
		normalized := normalizeScanPath(path)
		if entry, ok := entryByPath[normalized]; ok {
			filtered = append(filtered, entry)
			continue
		}
		filtered = append(filtered, planner.SourceEntry{
			Path: normalized,
			Size: sizeByPath[normalized],
		})
	}
	return filtered, pendingPaths
}

func normalizeSelectionPaths(paths []string) []string {
	normalized := make([]string, 0, len(paths))
	seen := make(map[string]struct{}, len(paths))
	for _, path := range paths {
		value := normalizeScanPath(path)
		if value == "" || value == "/" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}
	return normalized
}

func pathMatchesSelection(path string, selected []string) bool {
	normalized := normalizeScanPath(path)
	if normalized == "" {
		return false
	}
	for _, item := range selected {
		if item == normalized || strings.HasPrefix(normalized, item+"/") {
			return true
		}
	}
	return false
}

func retryAttemptsFromMetadata(values map[string]interface{}) map[string]int {
	attempts := make(map[string]int)
	if values == nil {
		return attempts
	}
	raw, ok := values["retryAttempts"]
	if !ok || raw == nil {
		return attempts
	}
	switch typed := raw.(type) {
	case map[string]int:
		for path, count := range typed {
			attempts[normalizeScanPath(path)] = count
		}
	case map[string]interface{}:
		for path, count := range typed {
			attempts[normalizeScanPath(path)] = intNumber(count)
		}
	}
	return attempts
}

func incrementRetryAttempts(values map[string]interface{}, retryPaths []string) map[string]int {
	attempts := retryAttemptsFromMetadata(values)
	for _, path := range retryPaths {
		normalized := normalizeScanPath(path)
		if normalized == "" || normalized == "/" {
			continue
		}
		attempts[normalized] = attempts[normalized] + 1
	}
	return attempts
}

func selectRetryEntries(detail Detail, opts RetryOptions) ([]planner.SourceEntry, []string, string, string, string) {
	selectedPaths := normalizeSelectionPaths(opts.Paths)
	if len(selectedPaths) > 0 {
		return selectRetryEntriesForSelection(detail, selectedPaths, strings.TrimSpace(opts.Scope))
	}
	pendingEntries, pendingPaths := pendingRetryEntries(detail)
	if len(pendingEntries) > 0 {
		return pendingEntries, pendingPaths, "pending_only", "", ""
	}

	eligiblePaths := make([]string, 0)
	blockedUntil := ""
	entryByPath := make(map[string]planner.SourceEntry, len(detail.SourceEntries))
	for _, entry := range detail.SourceEntries {
		entryByPath[normalizeScanPath(entry.Path)] = entry
	}
	sizeByPath := make(map[string]int64, len(detail.Plan.Items))
	for _, item := range detail.Plan.Items {
		sizeByPath[normalizeScanPath(item.Path)] = item.Size
	}
	for _, item := range detail.Runtime.RetryQueue {
		if !item.Retryable {
			continue
		}
		switch item.RetryClass {
		case "pending_manual":
			continue
		case "rate_limited":
			if item.EligibleAt != "" {
				eligibleAt, err := time.Parse(time.RFC3339, item.EligibleAt)
				if err == nil && time.Now().UTC().Before(eligibleAt) {
					if blockedUntil == "" || item.EligibleAt < blockedUntil {
						blockedUntil = item.EligibleAt
					}
					continue
				}
			}
		}
		eligiblePaths = append(eligiblePaths, item.Path)
	}
	if len(eligiblePaths) == 0 {
		if blockedUntil != "" {
			return nil, nil, "", blockedUntil, ""
		}
		summary := summarizeRetryQueue(detail.Runtime.RetryQueue)
		if summary.ShouldBlock {
			return nil, nil, "", "", summary.BlockedReason
		}
		return nil, nil, "", "", ""
	}
	filtered := make([]planner.SourceEntry, 0, len(eligiblePaths))
	for _, path := range eligiblePaths {
		normalized := normalizeScanPath(path)
		if entry, ok := entryByPath[normalized]; ok {
			filtered = append(filtered, entry)
			continue
		}
		filtered = append(filtered, planner.SourceEntry{
			Path: normalized,
			Size: sizeByPath[normalized],
		})
	}
	return filtered, eligiblePaths, "retry_queue", "", ""
}

func selectRetryEntriesForSelection(detail Detail, selectedPaths []string, scope string) ([]planner.SourceEntry, []string, string, string, string) {
	if strings.TrimSpace(scope) == "selected_directory_subset" {
		selectedEntries, selectedEntryPaths := selectedDirectoryEntries(detail, selectedPaths)
		if len(selectedEntries) == 0 {
			return nil, nil, "", "", "retry_selection_empty"
		}
		return selectedEntries, selectedEntryPaths, "selected_directory_subset", "", ""
	}

	pendingEntries, pendingPaths := pendingRetryEntries(detail)
	selectedPendingEntries := make([]planner.SourceEntry, 0, len(pendingEntries))
	selectedPendingPaths := make([]string, 0, len(pendingPaths))
	for idx, path := range pendingPaths {
		if !pathMatchesSelection(path, selectedPaths) {
			continue
		}
		selectedPendingPaths = append(selectedPendingPaths, path)
		if idx < len(pendingEntries) {
			selectedPendingEntries = append(selectedPendingEntries, pendingEntries[idx])
		}
	}
	if len(selectedPendingEntries) > 0 {
		mode := "selected_pending_subset"
		if scope != "" {
			mode = scope
		}
		return selectedPendingEntries, selectedPendingPaths, mode, "", ""
	}

	filteredQueue := make([]RetryQueueItem, 0)
	for _, item := range detail.Runtime.RetryQueue {
		if pathMatchesSelection(item.Path, selectedPaths) {
			filteredQueue = append(filteredQueue, item)
		}
	}
	if len(filteredQueue) == 0 {
		return nil, nil, "", "", "retry_selection_empty"
	}

	eligiblePaths := make([]string, 0, len(filteredQueue))
	blockedUntil := ""
	entryByPath := make(map[string]planner.SourceEntry, len(detail.SourceEntries))
	for _, entry := range detail.SourceEntries {
		entryByPath[normalizeScanPath(entry.Path)] = entry
	}
	sizeByPath := make(map[string]int64, len(detail.Plan.Items))
	for _, item := range detail.Plan.Items {
		sizeByPath[normalizeScanPath(item.Path)] = item.Size
	}
	for _, item := range filteredQueue {
		if !item.Retryable {
			continue
		}
		switch item.RetryClass {
		case "pending_manual":
			continue
		case "rate_limited":
			if item.EligibleAt != "" {
				eligibleAt, err := time.Parse(time.RFC3339, item.EligibleAt)
				if err == nil && time.Now().UTC().Before(eligibleAt) {
					if blockedUntil == "" || item.EligibleAt < blockedUntil {
						blockedUntil = item.EligibleAt
					}
					continue
				}
			}
		}
		eligiblePaths = append(eligiblePaths, item.Path)
	}
	if len(eligiblePaths) == 0 {
		if blockedUntil != "" {
			return nil, nil, "", blockedUntil, ""
		}
		summary := summarizeRetryQueue(filteredQueue)
		if summary.ShouldBlock {
			return nil, nil, "", "", summary.BlockedReason
		}
		return nil, nil, "", "", "retry_selection_empty"
	}

	filtered := make([]planner.SourceEntry, 0, len(eligiblePaths))
	for _, path := range eligiblePaths {
		normalized := normalizeScanPath(path)
		if entry, ok := entryByPath[normalized]; ok {
			filtered = append(filtered, entry)
			continue
		}
		filtered = append(filtered, planner.SourceEntry{
			Path: normalized,
			Size: sizeByPath[normalized],
		})
	}
	mode := "selected_retry_subset"
	if scope != "" {
		mode = scope
	}
	return filtered, eligiblePaths, mode, "", ""
}

func selectedDirectoryEntries(detail Detail, selectedPaths []string) ([]planner.SourceEntry, []string) {
	if len(selectedPaths) == 0 {
		return nil, nil
	}
	selectedEntries := make([]planner.SourceEntry, 0)
	selectedEntryPaths := make([]string, 0)
	seen := make(map[string]struct{})
	appendEntry := func(entry planner.SourceEntry) {
		normalized := normalizeScanPath(entry.Path)
		if normalized == "" || normalized == "/" {
			return
		}
		if _, ok := seen[normalized]; ok {
			return
		}
		seen[normalized] = struct{}{}
		entry.Path = normalized
		selectedEntries = append(selectedEntries, entry)
		selectedEntryPaths = append(selectedEntryPaths, normalized)
	}
	for _, entry := range detail.SourceEntries {
		if pathMatchesSelection(entry.Path, selectedPaths) {
			appendEntry(entry)
		}
	}
	if len(selectedEntries) > 0 {
		return selectedEntries, selectedEntryPaths
	}
	for _, item := range detail.Plan.Items {
		if !pathMatchesSelection(item.Path, selectedPaths) {
			continue
		}
		appendEntry(planner.SourceEntry{
			Path: item.Path,
			Size: item.Size,
		})
	}
	return selectedEntries, selectedEntryPaths
}

func pendingRetryPaths(results []Result) []string {
	seen := make(map[string]struct{})
	paths := make([]string, 0)
	for _, result := range results {
		if !isPendingRelayResult(result) {
			continue
		}
		path := normalizeScanPath(stringValue(result.Payload["path"]))
		if path == "" || path == "/" {
			continue
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		paths = append(paths, path)
	}
	return paths
}

func retrySelectedRoots(existingRoots []string, retryPaths []string) []string {
	selected := make([]string, 0)
	seen := make(map[string]struct{})
	appendRoot := func(root string) {
		root = normalizeScanPath(root)
		if root == "" {
			return
		}
		if _, ok := seen[root]; ok {
			return
		}
		seen[root] = struct{}{}
		selected = append(selected, root)
	}
	for _, root := range existingRoots {
		for _, path := range retryPaths {
			if isUnderRoot(path, root) {
				appendRoot(root)
				break
			}
		}
	}
	for _, path := range retryPaths {
		matched := ""
		for _, root := range existingRoots {
			if isUnderRoot(path, root) {
				matched = root
				break
			}
		}
		if matched != "" {
			appendRoot(matched)
			continue
		}
		appendRoot(parentDirectory(path))
	}
	return selected
}

func isUnderRoot(path, root string) bool {
	path = normalizeScanPath(path)
	root = normalizeScanPath(root)
	if root == "/" {
		return true
	}
	return path == root || strings.HasPrefix(path, root+"/")
}

func syncRuntimeRiskEvidence(runtime *RuntimeState, metadata map[string]interface{}, results []Result) {
	if runtime == nil {
		return
	}
	runtime.RiskHitCount = 0
	runtime.LastRiskStatus = ""
	runtime.RiskHits = nil
	riskProfile := riskProfileFromMetadata(metadata)
	for idx := range results {
		if results[idx].Payload == nil {
			results[idx].Payload = map[string]interface{}{}
		}
		riskHit, ok := riskHitFromPayload(results[idx].Payload)
		if !ok {
			riskHit, ok = detectRiskHit(riskProfile, results[idx], stringValue(results[idx].Payload["path"]))
			if ok {
				results[idx].Payload["riskHit"] = riskHit
			}
		}
		if !ok {
			continue
		}
		runtime.RiskHitCount++
		runtime.LastRiskStatus = riskHit.Status
		runtime.RiskHits = append(runtime.RiskHits, riskHit)
	}
}

func riskProfileFromMetadata(metadata map[string]interface{}) planner.RiskProfile {
	if metadata == nil {
		return planner.RiskProfile{}
	}
	return riskProfileFromRaw(metadata["riskProfile"])
}

func riskProfileFromRaw(raw interface{}) planner.RiskProfile {
	switch typed := raw.(type) {
	case planner.RiskProfile:
		return typed
	case map[string]interface{}:
		return planner.RiskProfile{
			Mode:                planner.RiskMode(stringValue(typed["mode"])),
			RequestIntervalMS:   intNumber(typed["requestIntervalMs"]),
			PageSize:            intNumber(typed["pageSize"]),
			DirectoryIntervalMS: intNumber(typed["directoryIntervalMs"]),
			CooldownSeconds:     intNumber(typed["cooldownSeconds"]),
			RetryLimit:          intNumber(typed["retryLimit"]),
			MaxConcurrent:       intNumber(typed["maxConcurrent"]),
			AutoRetryStartHour:  intNumber(typed["autoRetryStartHour"]),
			AutoRetryEndHour:    intNumber(typed["autoRetryEndHour"]),
			RiskKeywords:        metadataStringSlice(map[string]interface{}{"keywords": typed["riskKeywords"]}, "keywords"),
		}
	default:
		return planner.RiskProfile{}
	}
}

func detectRiskHit(riskProfile planner.RiskProfile, result Result, path string) (RiskHit, bool) {
	providerStatus := strings.ToLower(strings.TrimSpace(stringValue(result.Payload["providerStatus"])))
	message := strings.ToLower(strings.TrimSpace(result.Message))
	if providerStatus == "" && message == "" {
		return RiskHit{}, false
	}
	for _, keyword := range riskProfile.RiskKeywords {
		normalizedKeyword := strings.ToLower(strings.TrimSpace(keyword))
		if normalizedKeyword == "" {
			continue
		}
		if strings.Contains(providerStatus, normalizedKeyword) || strings.Contains(normalizedKeyword, providerStatus) || strings.Contains(message, normalizedKeyword) {
			return RiskHit{
				Status:      providerStatus,
				Keyword:     keyword,
				ItemPath:    path,
				Stage:       "upload",
				Message:     result.Message,
				TriggeredAt: result.CreatedAt,
			}, true
		}
	}
	for _, keyword := range []string{"rate_limit", "rate_limited", "too_many_requests", "captcha", "risk_control", "frequency_limit", "flow_limit", "forbidden"} {
		if strings.Contains(providerStatus, keyword) || strings.Contains(message, keyword) {
			return RiskHit{
				Status:      providerStatus,
				Keyword:     keyword,
				ItemPath:    path,
				Stage:       "upload",
				Message:     result.Message,
				TriggeredAt: result.CreatedAt,
			}, true
		}
	}
	return RiskHit{}, false
}

func riskHitFromPayload(payload map[string]interface{}) (RiskHit, bool) {
	if payload == nil {
		return RiskHit{}, false
	}
	raw, ok := payload["riskHit"]
	if !ok || raw == nil {
		return RiskHit{}, false
	}
	switch typed := raw.(type) {
	case RiskHit:
		return typed, true
	case map[string]interface{}:
		return RiskHit{
			Status:      stringValue(typed["status"]),
			Keyword:     stringValue(typed["keyword"]),
			ItemPath:    stringValue(typed["itemPath"]),
			Stage:       stringValue(typed["stage"]),
			Message:     stringValue(typed["message"]),
			TriggeredAt: stringValue(typed["triggeredAt"]),
		}, true
	default:
		return RiskHit{}, false
	}
}

func syncRuntimePendingTree(runtime *RuntimeState, metadata map[string]interface{}, results []Result) {
	if runtime == nil {
		return
	}
	runtime.PendingCount = 0
	runtime.PendingTree = buildPendingTree(metadata, results)
	runtime.PendingCount = countPendingNodes(runtime.PendingTree)
}

func syncRuntimeRetryQueue(runtime *RuntimeState, metadata map[string]interface{}, results []Result) {
	if runtime == nil {
		return
	}
	runtime.RetryQueue = buildRetryQueue(metadata, results)
	runtime.RetryableCount = 0
	runtime.BlockedRetryCount = 0
	for _, item := range runtime.RetryQueue {
		if item.Retryable {
			runtime.RetryableCount++
		}
		if item.Blocked {
			runtime.BlockedRetryCount++
		}
	}
}

func applyRetryQueueSummary(runtime *RuntimeState, metadata map[string]interface{}) {
	if runtime == nil {
		return
	}
	riskProfile := riskProfileFromMetadata(metadata)
	summary := summarizeRetryQueueWithRisk(runtime.RetryQueue, riskProfile, time.Now().UTC())
	runtime.BlockedReason = ""
	runtime.BlockedAction = ""
	runtime.BlockedAdvice = ""
	runtime.NextRetryAt = ""
	if summary.ShouldBlock {
		runtime.BlockedReason = summary.BlockedReason
		runtime.BlockedAction = summary.BlockedAction
		runtime.BlockedAdvice = summary.BlockedAdvice
		runtime.NextRetryAt = summary.NextRetryAt
	}
	if metadata == nil {
		return
	}
	if !summary.ShouldBlock && len(runtime.RetryQueue) == 0 {
		delete(metadata, "retrySummary")
		return
	}
	metadata["retrySummary"] = map[string]interface{}{
		"shouldBlock":              summary.ShouldBlock,
		"blockedReason":            summary.BlockedReason,
		"blockedAction":            summary.BlockedAction,
		"blockedAdvice":            summary.BlockedAdvice,
		"nextRetryAt":              summary.NextRetryAt,
		"windowBlocked":            summary.WindowBlocked,
		"canAutoRetry":             summary.CanAutoRetry,
		"queueSize":                len(runtime.RetryQueue),
		"retryableNowCount":        summary.RetryableNowCount,
		"cooldownCount":            summary.CooldownCount,
		"pendingManualCount":       summary.PendingManualCount,
		"authExpiredCount":         summary.AuthExpiredCount,
		"localMissingCount":        summary.LocalMissingCount,
		"exhaustedCount":           summary.ExhaustedCount,
		"uploadCheckpointEligible": summary.UploadCheckpointEligible,
		"autoRecoverEligible":      summary.AutoRecoverEligible,
		"autoRecoverMode":          summary.AutoRecoverMode,
		"autoRecoverAdvice":        summary.AutoRecoverAdvice,
	}
}

func autoRecoverReason(detail Detail) string {
	if detail.Task.State == StateCompletedWithErrors && retryQueueCanAutoResumeUploads(detail.Runtime.RetryQueue) {
		return "upload_checkpoint_auto_resume"
	}
	summary := summarizeRetryQueueWithRisk(detail.Runtime.RetryQueue, riskProfileFromMetadata(detail.Plan.Metadata), time.Now().UTC())
	if summary.BlockedReason == "retry_queue_waiting_for_retry_window" {
		return "retry_window_waiting_auto_retry"
	}
	if summary.BlockedReason == "retry_queue_waiting_for_cooldown" {
		return "cooldown_elapsed_auto_retry"
	}
	if summary.CanAutoRetry {
		return "retry_queue_auto_retry"
	}
	return "auto_retry"
}

func markAutoRecovery(detail *Detail, source Detail, reason, recoveredAt string) {
	if detail == nil {
		return
	}
	if recoveredAt == "" {
		recoveredAt = time.Now().UTC().Format(time.RFC3339)
	}
	count := source.Runtime.AutoRecoverCount + 1
	detail.Runtime.AutoRecovered = true
	detail.Runtime.AutoRecoverReason = reason
	detail.Runtime.AutoRecoverCount = count
	detail.Runtime.AutoRecoveredAt = recoveredAt
	detail.Runtime.AutoRecoverState = string(source.Task.State)
	if detail.Plan.Metadata == nil {
		detail.Plan.Metadata = map[string]interface{}{}
	}
	detail.Plan.Metadata["autoRecovery"] = map[string]interface{}{
		"recovered":   true,
		"reason":      reason,
		"count":       count,
		"recoveredAt": recoveredAt,
		"sourceState": string(source.Task.State),
	}
}

func runtimeAutoRecoveryPayload(runtime RuntimeState) map[string]interface{} {
	if !runtime.AutoRecovered {
		return nil
	}
	payload := map[string]interface{}{
		"recovered": true,
		"count":     runtime.AutoRecoverCount,
	}
	if runtime.AutoRecoverReason != "" {
		payload["reason"] = runtime.AutoRecoverReason
	}
	if runtime.AutoRecoveredAt != "" {
		payload["recoveredAt"] = runtime.AutoRecoveredAt
	}
	if runtime.AutoRecoverState != "" {
		payload["sourceState"] = runtime.AutoRecoverState
	}
	return payload
}

func syncRuntimeUploadCheckpoint(runtime *RuntimeState, results []Result) {
	if runtime == nil {
		return
	}
	runtime.UploadCheckpoint = nil
	for idx := len(results) - 1; idx >= 0; idx-- {
		checkpoint := uploadCheckpointFromResult(results[idx])
		if checkpoint == nil {
			continue
		}
		runtime.UploadCheckpoint = checkpoint
		return
	}
}

func blockedGuidance(reason string) (string, string) {
	switch reason {
	case "retry_queue_retry_limit_exhausted":
		return "review_and_reset_retry_strategy", "已达到任务级 retryLimit，请检查 provider 返回的失败原因、放宽策略或人工确认后再重新发起任务。"
	case "retry_queue_requires_auth_refresh":
		return "refresh_auth_profile", "目标端授权已失效，请先刷新或重建授权档案，再恢复任务。"
	case "retry_queue_requires_local_file_restore":
		return "restore_local_source_file", "本地回退文件缺失，请先补回源文件或调整执行策略，再继续重试。"
	case "retry_queue_waiting_for_cooldown":
		return "wait_for_cooldown", "当前处于风控冷却窗口，等待 nextRetryAt 后系统会尝试自动补传。"
	case "retry_queue_waiting_for_retry_window":
		return "wait_for_retry_window", "当前已满足自动补传条件，但不在允许的自动补传时间窗内，等待 nextRetryAt 后系统会自动接管。"
	case "retry_queue_pending_manual_confirmation":
		return "manual_confirmation_required", "存在 pending_manual 项，需要人工确认或等待后续真实 fallback 运行时能力。"
	default:
		return "", ""
	}
}

func autoRecoverGuidance(mode string) string {
	switch mode {
	case "upload_checkpoint_auto_resume":
		return "当前失败队列都带可恢复的 upload checkpoint，单机 worker 会优先尝试续跑上传会话。"
	case "cooldown_elapsed_auto_retry":
		return "当前队列主要受冷却窗口阻塞，窗口结束后单机 worker 会自动重试。"
	case "retry_window_waiting_auto_retry":
		return "当前队列已经满足自动补传条件，但还不在允许的自动补传时间窗内，系统会在下一个时间窗开始后自动接管。"
	case "retry_queue_auto_retry":
		return "当前队列不存在人工确认/授权/本地文件硬阻塞，满足条件时可由后台自动接管重试。"
	default:
		return ""
	}
}

func autoRecoverModePriority(mode string) int {
	switch mode {
	case "upload_checkpoint_auto_resume":
		return 0
	case "retry_queue_auto_retry":
		return 1
	case "retry_window_waiting_auto_retry":
		return 2
	case "cooldown_elapsed_auto_retry":
		return 3
	default:
		return 9
	}
}

func summarizeRetryQueue(queue []RetryQueueItem) retryQueueSummary {
	summary := retryQueueSummary{}
	if len(queue) == 0 {
		return summary
	}
	immediateRetry := 0
	cooldownCount := 0
	pendingManualCount := 0
	authExpiredCount := 0
	localMissingCount := 0
	exhaustedCount := 0
	uploadCheckpointEligible := 0
	now := time.Now().UTC()
	for _, item := range queue {
		if item.Exhausted {
			exhaustedCount++
			continue
		}
		switch item.RetryClass {
		case "pending_manual":
			pendingManualCount++
		case "auth_expired":
			authExpiredCount++
		case "local_file_missing":
			localMissingCount++
		case "rate_limited":
			if item.EligibleAt != "" {
				eligibleAt, err := time.Parse(time.RFC3339, item.EligibleAt)
				if err == nil && now.Before(eligibleAt) {
					cooldownCount++
					if summary.NextRetryAt == "" || eligibleAt.UTC().Format(time.RFC3339) < summary.NextRetryAt {
						summary.NextRetryAt = eligibleAt.UTC().Format(time.RFC3339)
					}
					continue
				}
			}
			if item.Retryable {
				immediateRetry++
			}
		default:
			if item.Retryable {
				immediateRetry++
			}
		}
		if item.Retryable && !item.Blocked && uploadCheckpointCanResume(item.UploadCheckpoint) {
			uploadCheckpointEligible++
		}
	}
	summary.RetryableNowCount = immediateRetry
	summary.CooldownCount = cooldownCount
	summary.PendingManualCount = pendingManualCount
	summary.AuthExpiredCount = authExpiredCount
	summary.LocalMissingCount = localMissingCount
	summary.ExhaustedCount = exhaustedCount
	summary.UploadCheckpointEligible = uploadCheckpointEligible
	if immediateRetry > 0 {
		summary.CanAutoRetry = pendingManualCount == 0 && authExpiredCount == 0 && localMissingCount == 0
		if retryQueueCanAutoResumeUploads(queue) {
			summary.AutoRecoverEligible = true
			summary.AutoRecoverMode = "upload_checkpoint_auto_resume"
			summary.AutoRecoverAdvice = autoRecoverGuidance(summary.AutoRecoverMode)
		} else if summary.CanAutoRetry {
			summary.AutoRecoverEligible = true
			summary.AutoRecoverMode = "retry_queue_auto_retry"
			summary.AutoRecoverAdvice = autoRecoverGuidance(summary.AutoRecoverMode)
		}
		return summary
	}
	if exhaustedCount > 0 {
		summary.ShouldBlock = true
		summary.BlockedReason = "retry_queue_retry_limit_exhausted"
		summary.BlockedAction, summary.BlockedAdvice = blockedGuidance(summary.BlockedReason)
		return summary
	}
	if authExpiredCount > 0 {
		summary.ShouldBlock = true
		summary.BlockedReason = "retry_queue_requires_auth_refresh"
		summary.BlockedAction, summary.BlockedAdvice = blockedGuidance(summary.BlockedReason)
		return summary
	}
	if localMissingCount > 0 {
		summary.ShouldBlock = true
		summary.BlockedReason = "retry_queue_requires_local_file_restore"
		summary.BlockedAction, summary.BlockedAdvice = blockedGuidance(summary.BlockedReason)
		return summary
	}
	if cooldownCount > 0 {
		summary.ShouldBlock = true
		summary.BlockedReason = "retry_queue_waiting_for_cooldown"
		summary.CanAutoRetry = pendingManualCount == 0
		summary.BlockedAction, summary.BlockedAdvice = blockedGuidance(summary.BlockedReason)
		if summary.CanAutoRetry {
			summary.AutoRecoverEligible = true
			summary.AutoRecoverMode = "cooldown_elapsed_auto_retry"
			summary.AutoRecoverAdvice = autoRecoverGuidance(summary.AutoRecoverMode)
		}
		return summary
	}
	if pendingManualCount > 0 {
		summary.ShouldBlock = true
		summary.BlockedReason = "retry_queue_pending_manual_confirmation"
		summary.BlockedAction, summary.BlockedAdvice = blockedGuidance(summary.BlockedReason)
		return summary
	}
	return summary
}

func summarizeRetryQueueWithRisk(queue []RetryQueueItem, riskProfile planner.RiskProfile, now time.Time) retryQueueSummary {
	summary := summarizeRetryQueue(queue)
	if len(queue) == 0 || !summary.CanAutoRetry || summary.ShouldBlock {
		return summary
	}
	nextWindowAt, blockedByWindow := nextAutoRetryWindowStart(riskProfile, now)
	if !blockedByWindow {
		return summary
	}
	summary.ShouldBlock = true
	summary.WindowBlocked = true
	summary.BlockedReason = "retry_queue_waiting_for_retry_window"
	summary.BlockedAction, summary.BlockedAdvice = blockedGuidance(summary.BlockedReason)
	summary.NextRetryAt = nextWindowAt
	summary.AutoRecoverEligible = true
	summary.AutoRecoverMode = "retry_window_waiting_auto_retry"
	summary.AutoRecoverAdvice = autoRecoverGuidance(summary.AutoRecoverMode)
	return summary
}

func nextAutoRetryWindowStart(riskProfile planner.RiskProfile, now time.Time) (string, bool) {
	start := riskProfile.AutoRetryStartHour
	end := riskProfile.AutoRetryEndHour
	if start <= 0 && end <= 0 {
		return "", false
	}
	if autoRetryAllowedNow(riskProfile, now) {
		return "", false
	}
	if start < 0 {
		start = 0
	}
	if end < 0 {
		end = 0
	}
	if start > 23 {
		start = 23
	}
	if end > 24 {
		end = 24
	}
	if start == end {
		return "", false
	}
	base := now.UTC()
	next := time.Date(base.Year(), base.Month(), base.Day(), start, 0, 0, 0, time.UTC)
	if start < end {
		if base.Hour() >= end {
			next = next.Add(24 * time.Hour)
		}
		return next.Format(time.RFC3339), true
	}
	if base.Hour() < start {
		return next.Format(time.RFC3339), true
	}
	return next.Add(24 * time.Hour).Format(time.RFC3339), true
}

func runtimeCanAutoRetry(runtime RuntimeState) bool {
	return runtimeCanAutoRetryWithRisk(runtime, planner.RiskProfile{})
}

func runtimeCanAutoRetryWithRisk(runtime RuntimeState, riskProfile planner.RiskProfile) bool {
	now := time.Now().UTC()
	summary := summarizeRetryQueueWithRisk(runtime.RetryQueue, riskProfile, now)
	if !summary.ShouldBlock {
		return summary.CanAutoRetry && len(runtime.RetryQueue) > 0 && autoRetryAllowedNow(riskProfile, now)
	}
	if summary.BlockedReason != "retry_queue_waiting_for_cooldown" && summary.BlockedReason != "retry_queue_waiting_for_retry_window" {
		return false
	}
	if strings.TrimSpace(summary.NextRetryAt) == "" {
		return false
	}
	nextRetryAt, err := time.Parse(time.RFC3339, summary.NextRetryAt)
	if err != nil {
		return false
	}
	if now.Before(nextRetryAt) {
		return false
	}
	return summary.CanAutoRetry && autoRetryAllowedNow(riskProfile, now)
}

func taskCanAutoRecover(detail Detail) bool {
	riskProfile := riskProfileFromMetadata(detail.Plan.Metadata)
	switch detail.Task.State {
	case StateBlocked:
		return runtimeCanAutoRetryWithRisk(detail.Runtime, riskProfile)
	case StateCompletedWithErrors:
		summary := summarizeRetryQueueWithRisk(detail.Runtime.RetryQueue, riskProfile, time.Now().UTC())
		if summary.ShouldBlock {
			return false
		}
		return retryQueueCanAutoResumeUploads(detail.Runtime.RetryQueue) && autoRetryAllowedNow(riskProfile, time.Now().UTC())
	default:
		return false
	}
}

func shouldIncludeAutoRecoverPool(detail Detail, summary retryQueueSummary) bool {
	if taskCanAutoRecover(detail) {
		return true
	}
	if summary.UploadCheckpointEligible > 0 {
		return true
	}
	if summary.AutoRecoverEligible {
		return true
	}
	if summary.CooldownCount > 0 || summary.BlockedReason == "retry_queue_waiting_for_cooldown" || detail.Runtime.BlockedAction == "wait_for_cooldown" {
		return true
	}
	if summary.WindowBlocked || summary.BlockedReason == "retry_queue_waiting_for_retry_window" || detail.Runtime.BlockedAction == "wait_for_retry_window" {
		return true
	}
	mode := autoRecoverReason(detail)
	return mode == "cooldown_elapsed_auto_retry" || mode == "retry_window_waiting_auto_retry"
}

func autoRetryAllowedNow(riskProfile planner.RiskProfile, now time.Time) bool {
	start := riskProfile.AutoRetryStartHour
	end := riskProfile.AutoRetryEndHour
	if start <= 0 && end <= 0 {
		return true
	}
	if start < 0 {
		start = 0
	}
	if end < 0 {
		end = 0
	}
	if start > 23 {
		start = 23
	}
	if end > 24 {
		end = 24
	}
	currentHour := now.UTC().Hour()
	if start == end {
		return true
	}
	if start < end {
		return currentHour >= start && currentHour < end
	}
	return currentHour >= start || currentHour < end
}

func retryQueueCanAutoResumeUploads(queue []RetryQueueItem) bool {
	if len(queue) == 0 {
		return false
	}
	hasRetryable := false
	for _, item := range queue {
		if !item.Retryable || item.Blocked || item.Exhausted {
			return false
		}
		if item.RetryClass == "pending_manual" || item.RetryClass == "auth_expired" || item.RetryClass == "local_file_missing" {
			return false
		}
		if !uploadCheckpointCanResume(item.UploadCheckpoint) {
			return false
		}
		hasRetryable = true
	}
	return hasRetryable
}

func buildRetryQueue(metadata map[string]interface{}, results []Result) []RetryQueueItem {
	if len(results) == 0 {
		return nil
	}
	queue := make([]RetryQueueItem, 0)
	selectedRoots := metadataStringSlice(metadata, "selectedRoots")
	riskProfile := riskProfileFromMetadata(metadata)
	cooldownSeconds := riskProfile.CooldownSeconds
	retryLimit := riskProfile.RetryLimit
	retryAttempts := retryAttemptsFromMetadata(metadata)
	for _, result := range results {
		if result.Status != "failed" {
			continue
		}
		path := normalizeScanPath(stringValue(result.Payload["path"]))
		if path == "" || path == "/" {
			continue
		}
		status := stringValue(result.Payload["providerStatus"])
		item := RetryQueueItem{
			Path:             path,
			RootPath:         matchRootPath(path, selectedRoots),
			ProviderStatus:   status,
			Strategy:         stringValue(result.Payload["strategy"]),
			Reason:           firstNonEmpty(stringValue(result.Payload["syncDecisionReason"]), result.Message),
			AttemptCount:     retryAttempts[path],
			RetryLimit:       retryLimit,
			UploadCheckpoint: uploadCheckpointFromResult(result),
		}
		if retryLimit > 0 {
			item.RemainingCount = maxInt(0, retryLimit-item.AttemptCount)
		}
		switch status {
		case "pending_manual_requires_confirmation":
			item.RetryClass = "pending_manual"
			item.RetryAction = "retry_after_manual_confirmation"
			item.Retryable = true
		case "rate_limited":
			item.RetryClass = "rate_limited"
			item.RetryAction = "retry_after_cooldown"
			item.Retryable = true
			item.CooldownTier, item.CooldownSeconds = resolveCooldownBackoff(item.AttemptCount, cooldownSeconds)
			if item.CooldownSeconds > 0 {
				if createdAt, err := time.Parse(time.RFC3339, result.CreatedAt); err == nil {
					item.EligibleAt = createdAt.Add(time.Duration(item.CooldownSeconds) * time.Second).UTC().Format(time.RFC3339)
				}
			}
		case "auth_expired":
			item.RetryClass = "auth_expired"
			item.RetryAction = "refresh_auth_profile"
			item.Blocked = true
		case "local_file_missing":
			item.RetryClass = "local_file_missing"
			item.RetryAction = "restore_local_file"
			item.Blocked = true
		default:
			item.RetryClass = "retry_failed"
			item.RetryAction = "retry_now"
			item.Retryable = true
		}
		if retryLimit > 0 && item.AttemptCount >= retryLimit {
			item.Retryable = false
			item.Blocked = true
			item.Exhausted = true
			item.RetryAction = "manual_intervention_required"
			item.Reason = firstNonEmpty(item.Reason, "Retry limit exhausted.")
		}
		queue = append(queue, item)
	}
	return queue
}

func resolveCooldownBackoff(attemptCount, cooldownSeconds int) (string, int) {
	switch {
	case attemptCount >= 6:
		return "extended", maxInt(cooldownSeconds, 1800)
	case attemptCount >= 3:
		return "normal", maxInt(cooldownSeconds, 300)
	default:
		return "fast", maxInt(cooldownSeconds, 30)
	}
}

func buildPendingTree(metadata map[string]interface{}, results []Result) []PendingNode {
	pendingResults := make([]Result, 0)
	for _, result := range results {
		if isPendingRelayResult(result) {
			pendingResults = append(pendingResults, result)
		}
	}
	if len(pendingResults) == 0 {
		return nil
	}

	roots := metadataStringSlice(metadata, "selectedRoots")
	rootBuilders := make(map[string]*pendingTreeBuilderNode)

	ensureNode := func(parent *pendingTreeBuilderNode, path, name, nodeType, rootPath string) *pendingTreeBuilderNode {
		if parent.children == nil {
			parent.children = make(map[string]*pendingTreeBuilderNode)
		}
		if existing, ok := parent.children[path]; ok {
			existing.node.ItemCount++
			return existing
		}
		item := &pendingTreeBuilderNode{
			node: PendingNode{
				Path:      path,
				Name:      name,
				NodeType:  nodeType,
				Status:    "pending_manual",
				RootPath:  rootPath,
				ItemCount: 1,
			},
			children: make(map[string]*pendingTreeBuilderNode),
		}
		parent.children[path] = item
		return item
	}

	for _, result := range pendingResults {
		path := normalizeScanPath(stringValue(result.Payload["path"]))
		if path == "/" {
			continue
		}
		rootPath := matchRootPath(path, roots)
		rootKey := normalizeScanPath(rootPath)
		rootBuilder, ok := rootBuilders[rootKey]
		if !ok {
			rootBuilder = &pendingTreeBuilderNode{
				node: PendingNode{
					Path:      rootKey,
					Name:      inferPendingNodeName(rootKey),
					NodeType:  "root",
					Status:    "pending_manual",
					RootPath:  rootKey,
					ItemCount: 0,
				},
				children: make(map[string]*pendingTreeBuilderNode),
			}
			rootBuilders[rootKey] = rootBuilder
		}
		rootBuilder.node.ItemCount++

		current := rootBuilder
		directoryParts := pendingDirectoryParts(rootKey, path)
		for _, dirPath := range directoryParts {
			current = ensureNode(current, dirPath, inferPendingNodeName(dirPath), "directory", rootKey)
		}

		fileNode := ensureNode(current, path, inferPendingNodeName(path), "file", rootKey)
		fileNode.node.ItemCount = 1
		fileNode.node.Reason = firstNonEmpty(stringValue(result.Payload["syncDecisionReason"]), result.Message)
		fileNode.node.ProviderStatus = stringValue(result.Payload["providerStatus"])
	}

	rootOrder := orderedPendingRootKeys(rootBuilders, roots)
	tree := make([]PendingNode, 0, len(rootOrder))
	for _, rootKey := range rootOrder {
		tree = append(tree, finalizePendingNode(rootBuilders[rootKey]))
	}
	return tree
}

func orderedPendingRootKeys(rootBuilders map[string]*pendingTreeBuilderNode, selectedRoots []string) []string {
	if len(rootBuilders) == 0 {
		return nil
	}
	ordered := make([]string, 0, len(rootBuilders))
	seen := make(map[string]bool, len(rootBuilders))
	for _, root := range selectedRoots {
		rootKey := normalizeScanPath(root)
		if _, ok := rootBuilders[rootKey]; ok && !seen[rootKey] {
			ordered = append(ordered, rootKey)
			seen[rootKey] = true
		}
	}
	remaining := make([]string, 0)
	for rootKey := range rootBuilders {
		if seen[rootKey] {
			continue
		}
		remaining = append(remaining, rootKey)
	}
	sort.SliceStable(remaining, func(i, j int) bool {
		left := remaining[i]
		right := remaining[j]
		leftDepth := pendingPathDepth(left)
		rightDepth := pendingPathDepth(right)
		if leftDepth != rightDepth {
			return leftDepth < rightDepth
		}
		return left < right
	})
	ordered = append(ordered, remaining...)
	return ordered
}

func finalizePendingNode(node *pendingTreeBuilderNode) PendingNode {
	if node == nil {
		return PendingNode{}
	}
	if len(node.children) == 0 {
		node.node.Children = nil
		return node.node
	}
	keys := make([]string, 0, len(node.children))
	for key := range node.children {
		keys = append(keys, key)
	}
	sortStringsByDepth(keys)
	children := make([]PendingNode, 0, len(keys))
	for _, key := range keys {
		children = append(children, finalizePendingNode(node.children[key]))
	}
	node.node.Children = children
	return node.node
}

func sortStringsByDepth(values []string) {
	for i := 0; i < len(values); i++ {
		for j := i + 1; j < len(values); j++ {
			if pendingNodeLess(values[j], values[i]) {
				values[i], values[j] = values[j], values[i]
			}
		}
	}
}

func pendingNodeLess(left, right string) bool {
	leftDepth := pendingPathDepth(left)
	rightDepth := pendingPathDepth(right)
	if leftDepth != rightDepth {
		return leftDepth < rightDepth
	}
	return left < right
}

func pendingPathDepth(path string) int {
	normalized := normalizeScanPath(path)
	if normalized == "/" {
		return 0
	}
	depth := 0
	for _, ch := range normalized {
		if ch == '/' {
			depth++
		}
	}
	return depth
}

func pendingDirectoryParts(rootPath, filePath string) []string {
	rootPath = normalizeScanPath(rootPath)
	filePath = normalizeScanPath(filePath)
	parent := parentDirectory(filePath)
	if parent == "/" || parent == rootPath {
		return nil
	}
	parts := make([]string, 0)
	current := parent
	for current != "/" && current != rootPath && current != "." {
		parts = append(parts, current)
		current = parentDirectory(current)
	}
	for i, j := 0, len(parts)-1; i < j; i, j = i+1, j-1 {
		parts[i], parts[j] = parts[j], parts[i]
	}
	return parts
}

func inferPendingNodeName(path string) string {
	normalized := normalizeScanPath(path)
	if normalized == "/" {
		return "/"
	}
	index := strings.LastIndex(normalized, "/")
	if index < 0 || index >= len(normalized)-1 {
		return normalized
	}
	return normalized[index+1:]
}

func isPendingRelayResult(result Result) bool {
	if result.Status != "failed" {
		return false
	}
	if stringValue(result.Payload["providerStatus"]) == "pending_manual_requires_confirmation" {
		return true
	}
	return stringValue(result.Payload["strategy"]) == string(planner.StrategyPendingManual)
}

func countPendingNodes(nodes []PendingNode) int {
	count := 0
	for _, node := range nodes {
		if node.NodeType == "file" {
			count++
		}
		count += countPendingNodes(node.Children)
	}
	return count
}

func buildProviderProbe(detail Detail, profile provider.AuthProfile, results []Result, createdAt string) ProviderProbe {
	doneCount := 0
	skippedCount := 0
	failedCount := 0
	for _, result := range results {
		switch result.Status {
		case "done":
			doneCount++
		case "skipped":
			skippedCount++
		case "failed":
			failedCount++
		}
	}
	return ProviderProbe{
		ID:          uuid.NewString(),
		ProviderKey: detail.Task.TargetProvider,
		ProfileID:   profile.ID,
		Status:      string(detail.Task.State),
		Payload: map[string]interface{}{
			"taskId":                         detail.Task.ID,
			"taskState":                      detail.Task.State,
			"completionKind":                 detail.Task.CompletionKind,
			"doneCount":                      doneCount,
			"skippedCount":                   skippedCount,
			"failedCount":                    failedCount,
			"resultCount":                    len(results),
			"executionMode":                  detail.Plan.Metadata["executionMode"],
			"recommendedExecutionMode":       detail.Plan.Metadata["recommendedExecutionMode"],
			"recommendedExecutionModeReason": detail.Plan.Metadata["recommendedExecutionModeReason"],
			"scanMode":                       detail.Plan.Metadata["scanMode"],
			"selectedRoots":                  detail.Plan.Metadata["selectedRoots"],
			"scanTrace":                      detail.Plan.Metadata["scanTrace"],
			"riskProfile":                    detail.Plan.Metadata["riskProfile"],
			"riskOverride":                   detail.Plan.Metadata["riskOverride"],
			"runtime":                        detail.Runtime,
			"pendingCount":                   detail.Runtime.PendingCount,
			"pendingTree":                    detail.Runtime.PendingTree,
			"retryableCount":                 detail.Runtime.RetryableCount,
			"blockedRetryCount":              detail.Runtime.BlockedRetryCount,
			"retryQueue":                     detail.Runtime.RetryQueue,
			"retrySummary":                   detail.Plan.Metadata["retrySummary"],
			"riskHitCount":                   detail.Runtime.RiskHitCount,
			"lastRiskStatus":                 detail.Runtime.LastRiskStatus,
			"autoRecovered":                  detail.Runtime.AutoRecovered,
			"autoRecoverReason":              detail.Runtime.AutoRecoverReason,
			"autoRecoverCount":               detail.Runtime.AutoRecoverCount,
			"autoRecoveredAt":                detail.Runtime.AutoRecoveredAt,
			"autoRecoverState":               detail.Runtime.AutoRecoverState,
			"autoRecovery":                   detail.Plan.Metadata["autoRecovery"],
			"currentRoot":                    detail.Runtime.CurrentRoot,
			"currentDirectory":               detail.Runtime.CurrentDirectory,
			"lastCompletedPath":              detail.Runtime.LastCompletedPath,
			"uploadCheckpoint":               detail.Runtime.UploadCheckpoint,
			"targetProfileId":                detail.TargetProfileID,
		},
		CreatedAt: createdAt,
	}
}

func uploadCheckpointFromResult(result Result) *UploadCheckpoint {
	if result.Status != "failed" {
		return nil
	}
	uploadPayload, ok := result.Payload["upload"].(map[string]interface{})
	if !ok || len(uploadPayload) == 0 {
		return nil
	}
	uploadID := firstNonEmpty(stringValue(uploadPayload["uploadId"]), stringValue(uploadPayload["upload_id"]))
	partCount := intNumber(uploadPayload["partCount"])
	uploadedPartCount := intNumber(uploadPayload["uploadedPartCount"])
	failedPartNumber := intNumber(uploadPayload["failedPartNumber"])
	nextPartNumber := intNumber(uploadPayload["nextPartNumber"])
	uploadedParts := uploadCheckpointParts(uploadPayload["uploadedParts"])
	providerData := uploadCheckpointProviderData(uploadPayload["providerData"])
	if len(providerData) == 0 {
		if resumable, ok := uploadPayload["resumable"].(map[string]interface{}); ok && len(resumable) > 0 {
			providerData = map[string]interface{}{
				"resumable": copyPayloadMap(resumable),
			}
		}
	}
	if uploadID == "" && partCount == 0 && uploadedPartCount == 0 && failedPartNumber == 0 && nextPartNumber == 0 && len(uploadedParts) == 0 && len(providerData) == 0 {
		return nil
	}
	return &UploadCheckpoint{
		ItemPath:          stringValue(result.Payload["path"]),
		ProviderStatus:    stringValue(result.Payload["providerStatus"]),
		FileID:            firstNonEmpty(stringValue(uploadPayload["fileId"]), stringValue(uploadPayload["file_id"])),
		UploadID:          uploadID,
		PartCount:         partCount,
		UploadedPartCount: uploadedPartCount,
		FailedPartNumber:  failedPartNumber,
		NextPartNumber:    nextPartNumber,
		UploadedParts:     uploadedParts,
		ProviderData:      providerData,
		UpdatedAt:         result.CreatedAt,
	}
}

func uploadCheckpointCanResume(checkpoint *UploadCheckpoint) bool {
	if checkpoint == nil {
		return false
	}
	return checkpoint.FileID != "" ||
		checkpoint.UploadID != "" ||
		checkpoint.PartCount > 0 ||
		checkpoint.UploadedPartCount > 0 ||
		checkpoint.FailedPartNumber > 0 ||
		checkpoint.NextPartNumber > 0 ||
		len(checkpoint.UploadedParts) > 0 ||
		len(checkpoint.ProviderData) > 0
}

func uploadCheckpointParts(raw interface{}) []map[string]interface{} {
	switch typed := raw.(type) {
	case []map[string]interface{}:
		out := make([]map[string]interface{}, 0, len(typed))
		for _, item := range typed {
			out = append(out, copyPayloadMap(item))
		}
		return out
	case []interface{}:
		out := make([]map[string]interface{}, 0, len(typed))
		for _, item := range typed {
			part, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			out = append(out, copyPayloadMap(part))
		}
		return out
	default:
		return nil
	}
}

func uploadCheckpointProviderData(raw interface{}) map[string]interface{} {
	typed, ok := raw.(map[string]interface{})
	if !ok || len(typed) == 0 {
		return nil
	}
	out := make(map[string]interface{}, len(typed))
	for key, value := range typed {
		if nested, ok := value.(map[string]interface{}); ok {
			out[key] = copyPayloadMap(nested)
			continue
		}
		out[key] = value
	}
	return out
}

func buildRetryUploadCheckpointMetadata(queue []RetryQueueItem, retryPaths []string) map[string]interface{} {
	if len(queue) == 0 || len(retryPaths) == 0 {
		return nil
	}
	allowed := make(map[string]bool, len(retryPaths))
	for _, path := range retryPaths {
		allowed[normalizeScanPath(path)] = true
	}
	items := make(map[string]interface{})
	for _, item := range queue {
		if item.UploadCheckpoint == nil {
			continue
		}
		path := normalizeScanPath(item.Path)
		if !allowed[path] {
			continue
		}
		items[path] = uploadCheckpointToMap(item.UploadCheckpoint)
	}
	if len(items) == 0 {
		return nil
	}
	return items
}

func firstRetryUploadCheckpoint(metadata map[string]interface{}, retryPaths []string) *UploadCheckpoint {
	if len(retryPaths) == 0 || metadata == nil {
		return nil
	}
	raw, ok := metadata["retryUploadCheckpoints"]
	if !ok || raw == nil {
		return nil
	}
	items, ok := raw.(map[string]interface{})
	if !ok {
		return nil
	}
	for _, path := range retryPaths {
		checkpoint := uploadCheckpointFromMetadata(items[normalizeScanPath(path)])
		if checkpoint != nil {
			return checkpoint
		}
	}
	return nil
}

func uploadCheckpointToMap(checkpoint *UploadCheckpoint) map[string]interface{} {
	if checkpoint == nil {
		return nil
	}
	out := map[string]interface{}{
		"itemPath":          checkpoint.ItemPath,
		"providerStatus":    checkpoint.ProviderStatus,
		"fileId":            checkpoint.FileID,
		"uploadId":          checkpoint.UploadID,
		"partCount":         checkpoint.PartCount,
		"uploadedPartCount": checkpoint.UploadedPartCount,
		"failedPartNumber":  checkpoint.FailedPartNumber,
		"nextPartNumber":    checkpoint.NextPartNumber,
		"updatedAt":         checkpoint.UpdatedAt,
	}
	if len(checkpoint.UploadedParts) > 0 {
		out["uploadedParts"] = uploadCheckpointParts(checkpoint.UploadedParts)
	}
	if len(checkpoint.ProviderData) > 0 {
		out["providerData"] = uploadCheckpointProviderData(checkpoint.ProviderData)
	}
	return out
}

func uploadCheckpointFromMetadata(raw interface{}) *UploadCheckpoint {
	switch typed := raw.(type) {
	case *UploadCheckpoint:
		return typed
	case UploadCheckpoint:
		checkpoint := typed
		return &checkpoint
	case map[string]interface{}:
		return &UploadCheckpoint{
			ItemPath:          stringValue(typed["itemPath"]),
			ProviderStatus:    stringValue(typed["providerStatus"]),
			FileID:            stringValue(typed["fileId"]),
			UploadID:          stringValue(typed["uploadId"]),
			PartCount:         intNumber(typed["partCount"]),
			UploadedPartCount: intNumber(typed["uploadedPartCount"]),
			FailedPartNumber:  intNumber(typed["failedPartNumber"]),
			NextPartNumber:    intNumber(typed["nextPartNumber"]),
			UploadedParts:     uploadCheckpointParts(typed["uploadedParts"]),
			ProviderData:      uploadCheckpointProviderData(typed["providerData"]),
			UpdatedAt:         stringValue(typed["updatedAt"]),
		}
	default:
		return nil
	}
}

func resumeUploadForPath(metadata map[string]interface{}, path string) *provider.ResumeUpload {
	checkpoint := firstRetryUploadCheckpoint(metadata, []string{path})
	if checkpoint == nil {
		return nil
	}
	return &provider.ResumeUpload{
		FileID:            checkpoint.FileID,
		UploadID:          checkpoint.UploadID,
		PartCount:         checkpoint.PartCount,
		UploadedPartCount: checkpoint.UploadedPartCount,
		FailedPartNumber:  checkpoint.FailedPartNumber,
		NextPartNumber:    checkpoint.NextPartNumber,
		UploadedParts:     uploadCheckpointParts(checkpoint.UploadedParts),
		ProviderData:      uploadCheckpointProviderData(checkpoint.ProviderData),
	}
}
