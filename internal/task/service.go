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
	ConflictPolicy  provider.ConflictPolicy `json:"conflictPolicy"`
	SelectedRoots   []string                `json:"selectedRoots"`
	Entries         []planner.SourceEntry   `json:"entries"`
}

type Detail struct {
	Task            Task                  `json:"task"`
	Plan            planner.Plan          `json:"plan"`
	Items           []Item                `json:"items"`
	Results         []Result              `json:"results"`
	SourceProfileID string                `json:"sourceProfileId,omitempty"`
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
	if err := createTask(ctx, s.store, t, plan, items, req.Entries, req.SourceProfileID, req.TargetProfileID, string(req.ConflictPolicy)); err != nil {
		return Detail{}, err
	}
	return Detail{Task: t, Plan: plan, Items: items, Results: []Result{}, SourceProfileID: req.SourceProfileID, TargetProfileID: req.TargetProfileID, ConflictPolicy: string(req.ConflictPolicy), SourceEntries: req.Entries}, nil
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

	entries, trace, err := s.collectLeafFirstEntries(ctx, sourceEntry, sourceProfile, selectedRoots, pageSize)
	if err != nil {
		return err
	}
	plan, err := planner.BuildPreview(s.registry, planner.PreviewRequest{
		SourceProvider: detail.Task.SourceProvider,
		TargetProvider: detail.Task.TargetProvider,
		ThresholdMB:    detail.Plan.ThresholdMB,
		RiskMode:       planner.RiskMode(metadataStringFromRisk(riskProfile, "mode")),
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
	plan.Metadata["scanMode"] = "lazy_leaf_first"
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
