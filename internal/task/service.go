package task

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"

	"cloudpan-sync-go/internal/auth"
	"cloudpan-sync-go/internal/planner"
	"cloudpan-sync-go/internal/provider"
	sqlitestore "cloudpan-sync-go/internal/store/sqlite"
)

type CreateRequest struct {
	SourceProvider  string                  `json:"sourceProvider"`
	TargetProvider  string                  `json:"targetProvider"`
	TargetProfileID string                  `json:"targetProfileId"`
	ThresholdMB     int                     `json:"thresholdMB"`
	RiskMode        planner.RiskMode        `json:"riskMode"`
	ConflictPolicy  provider.ConflictPolicy `json:"conflictPolicy"`
	SelectedRoots   []string                `json:"selectedRoots"`
	Entries         []planner.SourceEntry   `json:"entries"`
}

type Detail struct {
	Task            Task                  `json:"task"`
	Plan            planner.Plan          `json:"plan"`
	Items           []Item                `json:"items"`
	Results         []Result              `json:"results"`
	TargetProfileID string                `json:"targetProfileId,omitempty"`
	ConflictPolicy  string                `json:"conflictPolicy,omitempty"`
	SourceEntries   []planner.SourceEntry `json:"sourceEntries,omitempty"`
}

type EvidenceSummary struct {
	TotalTasks        int             `json:"totalTasks"`
	CompletedTasks    int             `json:"completedTasks"`
	FailedResultCount int             `json:"failedResultCount"`
	DoneResultCount   int             `json:"doneResultCount"`
	RecentResults     []Result        `json:"recentResults"`
	RecentProbes      []ProviderProbe `json:"recentProbes"`
}

type StatusSummary struct {
	ProviderKey     string                 `json:"providerKey"`
	ProfileCount    int                    `json:"profileCount"`
	TaskCount       int                    `json:"taskCount"`
	CompletedCount  int                    `json:"completedCount"`
	LastTaskState   string                 `json:"lastTaskState,omitempty"`
	LatestProbe     string                 `json:"latestProbe,omitempty"`
	LastObservedAt  string                 `json:"lastObservedAt,omitempty"`
	SnapshotSummary map[string]interface{} `json:"snapshotSummary,omitempty"`
}

type Service struct {
	store    *sqlitestore.Store
	registry *provider.Registry
	authSvc  *auth.Service
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
	if err := createTask(ctx, s.store, t, plan, items, req.Entries, req.TargetProfileID, string(req.ConflictPolicy)); err != nil {
		return Detail{}, err
	}
	return Detail{Task: t, Plan: plan, Items: items, Results: []Result{}, TargetProfileID: req.TargetProfileID, ConflictPolicy: string(req.ConflictPolicy), SourceEntries: req.Entries}, nil
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
	results := make([]Result, 0, len(detail.Plan.Items))
	failed := 0
	now := time.Now().UTC().Format(time.RFC3339)
	for i, item := range detail.Plan.Items {
		result := Result{
			ID:        uuid.NewString(),
			TaskID:    detail.Task.ID,
			ItemID:    detail.Items[i].ID,
			Payload:   map[string]interface{}{},
			CreatedAt: now,
		}
		localPath := lookupLocalPath(detail.SourceEntries, item.Path)
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
		}
		upload, fallbackUsed := s.executeUpload(entry, uploadReq)
		result.Mode = upload.Mode
		result.Message = upload.Message
		result.ConflictAction = conflictAction
		result.Payload["strategy"] = uploadReq.Strategy
		result.Payload["providerStatus"] = upload.Status
		if item.Sequence > 0 {
			result.Payload["sequence"] = item.Sequence
		}
		if riskProfile, ok := detail.Plan.Metadata["riskProfile"]; ok {
			result.Payload["riskProfile"] = riskProfile
		}
		if conflictAction != "" {
			result.Payload["conflictAction"] = conflictAction
		}
		if fallbackUsed {
			result.Payload["fallbackUsed"] = true
		}
		if item.Strategy == planner.StrategyPendingManual && !upload.OK {
			result.Status = "failed"
			failed++
		} else if upload.OK {
			result.Status = "done"
		} else {
			result.Status = "failed"
			failed++
		}
		results = append(results, result)
	}
	detail.Task.State = StateCompleted
	detail.Task.CompletionKind = CompletionKindProbeOnly
	if failed > 0 {
		detail.Task.State = StateCompletedWithErrors
	}
	detail.Task.UpdatedAt = now
	detail.Results = results
	if err := replaceTaskResults(ctx, s.store, detail.Task, results); err != nil {
		return Detail{}, true, err
	}
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

func (s *Service) Pause(ctx context.Context, id string) (Detail, bool, error) {
	return s.transitionState(ctx, id, []State{StateReady}, StatePaused)
}

func (s *Service) Resume(ctx context.Context, id string) (Detail, bool, error) {
	return s.transitionState(ctx, id, []State{StatePaused, StateBlocked}, StateReady)
}

func (s *Service) Retry(ctx context.Context, id string) (Detail, bool, error) {
	detail, ok, err := getTask(ctx, s.store, id)
	if err != nil || !ok {
		return Detail{}, ok, err
	}
	detail.Task.State = StateReady
	detail.Task.CompletionKind = ""
	detail.Task.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	detail.Results = []Result{}
	if err := resetTaskResults(ctx, s.store, detail.Task); err != nil {
		return Detail{}, true, err
	}
	return detail, true, nil
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

func (s *Service) RuntimeEvidence(ctx context.Context) (EvidenceSummary, error) {
	return taskEvidenceSummary(ctx, s.store)
}

func (s *Service) ProviderStatuses(ctx context.Context) ([]StatusSummary, error) {
	return providerStatusSummary(ctx, s.store, s.registry.List())
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
	if err := updateTaskState(ctx, s.store, detail.Task); err != nil {
		return Detail{}, true, err
	}
	return detail, true, nil
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

func (s *Service) executeUpload(entry provider.Entry, req provider.UploadRequest) (provider.UploadResult, bool) {
	if req.Strategy == string(planner.StrategyDownloadUpload) {
		if !localFileExists(req.LocalPath) {
			return provider.UploadResult{
				OperationResult: provider.OperationResult{
					Status:  "local_file_missing",
					Message: "Local file is required for download_upload fallback.",
					Mode:    "runtime_guard",
				},
			}, false
		}
	}

	upload := entry.Adapter.Upload(req)
	if upload.OK {
		return upload, false
	}
	if req.Strategy != string(planner.StrategyFastUpload) {
		return upload, false
	}
	if upload.Status != "hash_miss" {
		return upload, false
	}
	if !supportsFallback(entry.Meta.FallbackModes, string(planner.StrategyDownloadUpload)) {
		return upload, false
	}
	if !localFileExists(req.LocalPath) {
		return provider.UploadResult{
			OperationResult: provider.OperationResult{
				Status:  "local_file_missing",
				Message: "Hash miss fallback requires a local file.",
				Mode:    "runtime_guard",
			},
		}, false
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
	return fallback, true
}

func supportsFallback(modes []string, expected string) bool {
	for _, mode := range modes {
		if mode == expected {
			return true
		}
	}
	return false
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

func buildProviderProbe(detail Detail, profile provider.AuthProfile, results []Result, createdAt string) ProviderProbe {
	doneCount := 0
	failedCount := 0
	for _, result := range results {
		switch result.Status {
		case "done":
			doneCount++
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
			"taskId":          detail.Task.ID,
			"taskState":       detail.Task.State,
			"completionKind":  detail.Task.CompletionKind,
			"doneCount":       doneCount,
			"failedCount":     failedCount,
			"resultCount":     len(results),
			"targetProfileId": detail.TargetProfileID,
		},
		CreatedAt: createdAt,
	}
}
