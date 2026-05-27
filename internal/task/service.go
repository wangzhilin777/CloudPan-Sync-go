package task

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"

	"cloudpan-sync-go/internal/auth"
	"cloudpan-sync-go/internal/planner"
	"cloudpan-sync-go/internal/provider"
	sqlitestore "cloudpan-sync-go/internal/store/sqlite"
)

type CreateRequest struct {
	SourceProvider  string                  `json:"sourceProvider"`
	SourceProfileID string                  `json:"sourceProfileId"`
	TargetProvider  string                  `json:"targetProvider"`
	TargetProfileID string                  `json:"targetProfileId"`
	ThresholdMB     int                     `json:"thresholdMB"`
	RiskMode        planner.RiskMode        `json:"riskMode"`
	ExecutionMode   planner.ExecutionMode   `json:"executionMode"`
	ConflictPolicy  provider.ConflictPolicy `json:"conflictPolicy"`
	SelectedRoots   []string                `json:"selectedRoots"`
	Entries         []planner.SourceEntry   `json:"entries"`
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
	TotalTasks         int             `json:"totalTasks"`
	CompletedTasks     int             `json:"completedTasks"`
	FailedResultCount  int             `json:"failedResultCount"`
	DoneResultCount    int             `json:"doneResultCount"`
	SkippedResultCount int             `json:"skippedResultCount"`
	RecentResults      []Result        `json:"recentResults"`
	RecentProbes       []ProviderProbe `json:"recentProbes"`
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
		if err := updateTaskDetailState(ctx, s.store, detail); err != nil {
			return Detail{}, true, err
		}
	}

	results := append([]Result(nil), detail.Results...)
	startIndex := len(results)
	syncRuntimeCountsFromResults(&detail.Runtime, results)

	for i := startIndex; i < len(detail.Plan.Items); i++ {
		item := detail.Plan.Items[i]
		result := Result{
			ID:        uuid.NewString(),
			TaskID:    detail.Task.ID,
			ItemID:    detail.Items[i].ID,
			Payload:   map[string]interface{}{},
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
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
			detail.Results = results
			updateRuntimeAfterItem(&detail, item.Path, result)
			detail.Task.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
			if err := replaceTaskDetailAndResults(ctx, s.store, detail); err != nil {
				return Detail{}, true, err
			}
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
		}
		upload, fallbackUsed := s.executeUpload(entry, uploadReq)
		result.Mode = upload.Mode
		result.Message = upload.Message
		result.ConflictAction = conflictAction
		result.Payload["strategy"] = uploadReq.Strategy
		result.Payload["providerStatus"] = upload.Status
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
		detail.Results = results
		updateRuntimeAfterItem(&detail, item.Path, result)
		detail.Task.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		if err := replaceTaskDetailAndResults(ctx, s.store, detail); err != nil {
			return Detail{}, true, err
		}
	}

	failed := detail.Runtime.FailedCount
	detail.Task.State = StateCompleted
	detail.Task.CompletionKind = CompletionKindProbeOnly
	if failed > 0 {
		detail.Task.State = StateCompletedWithErrors
	}
	detail.Runtime.ExecutionState = "completed"
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
	detail.Runtime = initializeRuntimeState(detail.Plan)
	detail.Results = []Result{}
	if err := resetTaskResults(ctx, s.store, detail.Task); err != nil {
		return Detail{}, true, err
	}
	if err := updateTaskDetailState(ctx, s.store, detail); err != nil {
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
	ensureRuntimeState(&detail)
	switch target {
	case StatePaused:
		detail.Runtime.ExecutionState = "paused"
	case StateReady:
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

func initializeRuntimeState(plan planner.Plan) RuntimeState {
	directoryStates := collectDirectoryStates(plan)
	return RuntimeState{
		ExecutionState:  "idle",
		ProcessedCount:  0,
		DoneCount:       0,
		SkippedCount:    0,
		FailedCount:     0,
		NextSequence:    1,
		DirectoryStates: directoryStates,
	}
}

func ensureRuntimeState(detail *Detail) {
	if detail == nil {
		return
	}
	if len(detail.Runtime.DirectoryStates) == 0 {
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
			"runtime":                        detail.Runtime,
			"currentRoot":                    detail.Runtime.CurrentRoot,
			"currentDirectory":               detail.Runtime.CurrentDirectory,
			"lastCompletedPath":              detail.Runtime.LastCompletedPath,
			"targetProfileId":                detail.TargetProfileID,
		},
		CreatedAt: createdAt,
	}
}
