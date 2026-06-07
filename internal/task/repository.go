package task

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"

	"cloudpan-sync-go/internal/planner"
	"cloudpan-sync-go/internal/provider"
	sqlitestore "cloudpan-sync-go/internal/store/sqlite"
)

type taskPayload struct {
	SourceProfileID string                `json:"sourceProfileId,omitempty"`
	TargetProfileID string                `json:"targetProfileId"`
	ConflictPolicy  string                `json:"conflictPolicy"`
	Plan            planner.Plan          `json:"plan"`
	Runtime         RuntimeState          `json:"runtime"`
	Entries         []planner.SourceEntry `json:"entries,omitempty"`
}

func createTask(ctx context.Context, store *sqlitestore.Store, t Task, plan planner.Plan, runtime RuntimeState, items []Item, sourceEntries []planner.SourceEntry, sourceProfileID, targetProfileID, conflictPolicy string) error {
	tx, err := store.DB().BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	payloadJSON, err := json.Marshal(taskPayload{
		SourceProfileID: sourceProfileID,
		TargetProfileID: targetProfileID,
		ConflictPolicy:  conflictPolicy,
		Plan:            plan,
		Runtime:         runtime,
		Entries:         sourceEntries,
	})
	if err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `
INSERT INTO tasks(id, source_provider, target_provider, state, completion_kind, payload_json, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		t.ID, t.SourceProvider, t.TargetProvider, t.State, t.CompletionKind, string(payloadJSON), t.CreatedAt, t.UpdatedAt,
	); err != nil {
		return err
	}

	for i, item := range items {
		if _, err := tx.ExecContext(ctx, `
INSERT INTO task_items(id, task_id, path, size, strategy, payload_json, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?)`,
			item.ID, item.TaskID, item.Path, item.Size, plan.Items[i].Strategy, `{}`, t.CreatedAt,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func listTasks(ctx context.Context, store *sqlitestore.Store) ([]Detail, error) {
	rows, err := store.DB().QueryContext(ctx, `SELECT id FROM tasks ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	items := make([]Detail, 0, len(ids))
	for _, id := range ids {
		item, ok, err := getTask(ctx, store, id)
		if err != nil {
			return nil, err
		}
		if ok {
			items = append(items, item)
		}
	}
	return items, nil
}

func getTask(ctx context.Context, store *sqlitestore.Store, id string) (Detail, bool, error) {
	row := store.DB().QueryRowContext(ctx, `
SELECT id, source_provider, target_provider, state, completion_kind, payload_json, created_at, updated_at
FROM tasks
WHERE id = ?`, id)

	var (
		t           Task
		completion  string
		payloadJSON string
	)
	if err := row.Scan(&t.ID, &t.SourceProvider, &t.TargetProvider, &t.State, &completion, &payloadJSON, &t.CreatedAt, &t.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return Detail{}, false, nil
		}
		return Detail{}, false, err
	}
	t.CompletionKind = CompletionKind(completion)

	var payload taskPayload
	if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
		return Detail{}, false, err
	}

	items, err := getTaskItems(ctx, store, id)
	if err != nil {
		return Detail{}, false, err
	}
	results, err := getTaskResults(ctx, store, id)
	if err != nil {
		return Detail{}, false, err
	}

	return Detail{
		SourceProfileID: payload.SourceProfileID,
		Task:            t,
		Plan:            payload.Plan,
		Runtime:         payload.Runtime,
		Items:           items,
		Results:         results,
		TargetProfileID: payload.TargetProfileID,
		ConflictPolicy:  payload.ConflictPolicy,
		SourceEntries:   payload.Entries,
	}, true, nil
}

func replaceTaskPlanAndItems(ctx context.Context, store *sqlitestore.Store, detail Detail) error {
	tx, err := store.DB().BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	payloadJSON, err := json.Marshal(taskPayload{
		SourceProfileID: detail.SourceProfileID,
		TargetProfileID: detail.TargetProfileID,
		ConflictPolicy:  detail.ConflictPolicy,
		Plan:            detail.Plan,
		Runtime:         detail.Runtime,
		Entries:         detail.SourceEntries,
	})
	if err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `UPDATE tasks SET payload_json = ?, updated_at = ? WHERE id = ?`, string(payloadJSON), detail.Task.UpdatedAt, detail.Task.ID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM task_items WHERE task_id = ?`, detail.Task.ID); err != nil {
		return err
	}
	for i, item := range detail.Items {
		if _, err := tx.ExecContext(ctx, `
INSERT INTO task_items(id, task_id, path, size, strategy, payload_json, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?)`,
			item.ID, item.TaskID, item.Path, item.Size, detail.Plan.Items[i].Strategy, `{}`, detail.Task.CreatedAt,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func rebuildTaskForRetry(ctx context.Context, store *sqlitestore.Store, detail Detail) error {
	tx, err := store.DB().BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	payloadJSON, err := json.Marshal(taskPayload{
		SourceProfileID: detail.SourceProfileID,
		TargetProfileID: detail.TargetProfileID,
		ConflictPolicy:  detail.ConflictPolicy,
		Plan:            detail.Plan,
		Runtime:         detail.Runtime,
		Entries:         detail.SourceEntries,
	})
	if err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `UPDATE tasks SET state = ?, completion_kind = ?, payload_json = ?, updated_at = ? WHERE id = ?`,
		detail.Task.State, detail.Task.CompletionKind, string(payloadJSON), detail.Task.UpdatedAt, detail.Task.ID,
	); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM task_items WHERE task_id = ?`, detail.Task.ID); err != nil {
		return err
	}
	for i, item := range detail.Items {
		if _, err := tx.ExecContext(ctx, `
INSERT INTO task_items(id, task_id, path, size, strategy, payload_json, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?)`,
			item.ID, item.TaskID, item.Path, item.Size, detail.Plan.Items[i].Strategy, `{}`, detail.Task.CreatedAt,
		); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM task_results WHERE task_id = ?`, detail.Task.ID); err != nil {
		return err
	}
	return tx.Commit()
}

func replaceTaskDetailAndResults(ctx context.Context, store *sqlitestore.Store, detail Detail) error {
	tx, err := store.DB().BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	payloadJSON, err := json.Marshal(taskPayload{
		SourceProfileID: detail.SourceProfileID,
		TargetProfileID: detail.TargetProfileID,
		ConflictPolicy:  detail.ConflictPolicy,
		Plan:            detail.Plan,
		Runtime:         detail.Runtime,
		Entries:         detail.SourceEntries,
	})
	if err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `UPDATE tasks SET state = ?, completion_kind = ?, payload_json = ?, updated_at = ? WHERE id = ?`,
		detail.Task.State, detail.Task.CompletionKind, string(payloadJSON), detail.Task.UpdatedAt, detail.Task.ID,
	); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM task_results WHERE task_id = ?`, detail.Task.ID); err != nil {
		return err
	}
	for _, result := range detail.Results {
		payloadJSON, err := json.Marshal(result.Payload)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO task_results(id, task_id, task_item_id, status, mode, message, payload_json, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			result.ID, result.TaskID, result.ItemID, result.Status, result.Mode, result.Message, string(payloadJSON), result.CreatedAt,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func replaceTaskResults(ctx context.Context, store *sqlitestore.Store, t Task, results []Result) error {
	tx, err := store.DB().BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `DELETE FROM task_results WHERE task_id = ?`, t.ID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE tasks SET state = ?, completion_kind = ?, updated_at = ? WHERE id = ?`, t.State, t.CompletionKind, t.UpdatedAt, t.ID); err != nil {
		return err
	}
	for _, result := range results {
		payloadJSON, err := json.Marshal(result.Payload)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO task_results(id, task_id, task_item_id, status, mode, message, payload_json, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			result.ID, result.TaskID, result.ItemID, result.Status, result.Mode, result.Message, string(payloadJSON), result.CreatedAt,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func resetTaskResults(ctx context.Context, store *sqlitestore.Store, t Task) error {
	tx, err := store.DB().BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `DELETE FROM task_results WHERE task_id = ?`, t.ID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE tasks SET state = ?, completion_kind = ?, updated_at = ? WHERE id = ?`, t.State, t.CompletionKind, t.UpdatedAt, t.ID); err != nil {
		return err
	}
	return tx.Commit()
}

func updateTaskDetailState(ctx context.Context, store *sqlitestore.Store, detail Detail) error {
	payloadJSON, err := json.Marshal(taskPayload{
		SourceProfileID: detail.SourceProfileID,
		TargetProfileID: detail.TargetProfileID,
		ConflictPolicy:  detail.ConflictPolicy,
		Plan:            detail.Plan,
		Runtime:         detail.Runtime,
		Entries:         detail.SourceEntries,
	})
	if err != nil {
		return err
	}
	_, err = store.DB().ExecContext(ctx, `UPDATE tasks SET state = ?, completion_kind = ?, payload_json = ?, updated_at = ? WHERE id = ?`,
		detail.Task.State, detail.Task.CompletionKind, string(payloadJSON), detail.Task.UpdatedAt, detail.Task.ID,
	)
	return err
}

type blockedActionAccumulator struct {
	action         string
	advice         string
	taskIDs        map[string]struct{}
	providers      map[string]struct{}
	nextRetryAt    string
	sampleTaskID   string
	sampleProvider string
}

type autoRecoverLaneAccumulator struct {
	mode                            string
	advice                          string
	taskIDs                         map[string]struct{}
	providers                       map[string]struct{}
	profiles                        map[string]struct{}
	protocolGroups                  map[string]struct{}
	retryClasses                    map[string]struct{}
	blockedActions                  map[string]struct{}
	strategies                      map[string]struct{}
	suggestedProtocolGroupBudget    int
	suggestedProviderBudget         int
	suggestedProfileBudget          int
	queueItemCount                  int
	retryableNowCount               int
	cooldownCount                   int
	runnableTaskCount               int
	waitingCooldownTaskCount        int
	waitingRetryWindowTaskCount     int
	waitingAuthRefreshTaskCount     int
	waitingLocalRestoreTaskCount    int
	waitingProviderSessionTaskCount int
	waitingManualTaskCount          int
	waitingRetryLimitTaskCount      int
	waitingOtherTaskCount           int
	uploadCheckpointEligible        int
	nextRetryAt                     string
	sampleTaskID                    string
	sampleProvider                  string
	sampleProtocolGroup             string
	sampleProfileID                 string
	sampleStrategy                  string
}

func minPositiveBudget(current, next int) int {
	if next <= 0 {
		return current
	}
	if current <= 0 || next < current {
		return next
	}
	return current
}

func sortedMapKeys(values map[string]struct{}) []string {
	if len(values) == 0 {
		return nil
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		if strings.TrimSpace(key) == "" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func firstStringValue(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return strings.TrimSpace(values[0])
}

func summarizeBlockedActions(details []Detail) []BlockedAction {
	if len(details) == 0 {
		return nil
	}
	accumulators := make(map[string]*blockedActionAccumulator)
	order := make([]string, 0)
	for _, detail := range details {
		ensureRuntimeState(&detail)
		action := strings.TrimSpace(detail.Runtime.BlockedAction)
		if action == "" {
			continue
		}
		acc, ok := accumulators[action]
		if !ok {
			acc = &blockedActionAccumulator{
				action:    action,
				advice:    detail.Runtime.BlockedAdvice,
				taskIDs:   make(map[string]struct{}),
				providers: make(map[string]struct{}),
			}
			accumulators[action] = acc
			order = append(order, action)
		}
		acc.taskIDs[detail.Task.ID] = struct{}{}
		acc.providers[detail.Task.TargetProvider] = struct{}{}
		if acc.advice == "" {
			acc.advice = detail.Runtime.BlockedAdvice
		}
		if acc.sampleTaskID == "" {
			acc.sampleTaskID = detail.Task.ID
			acc.sampleProvider = detail.Task.TargetProvider
		}
		if detail.Runtime.NextRetryAt != "" && (acc.nextRetryAt == "" || detail.Runtime.NextRetryAt < acc.nextRetryAt) {
			acc.nextRetryAt = detail.Runtime.NextRetryAt
		}
	}
	items := make([]BlockedAction, 0, len(order))
	for _, action := range order {
		acc := accumulators[action]
		items = append(items, BlockedAction{
			Action:         acc.action,
			Advice:         acc.advice,
			TaskCount:      len(acc.taskIDs),
			ProviderCount:  len(acc.providers),
			NextRetryAt:    acc.nextRetryAt,
			SampleTaskID:   acc.sampleTaskID,
			SampleProvider: acc.sampleProvider,
		})
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].TaskCount != items[j].TaskCount {
			return items[i].TaskCount > items[j].TaskCount
		}
		if items[i].ProviderCount != items[j].ProviderCount {
			return items[i].ProviderCount > items[j].ProviderCount
		}
		return items[i].Action < items[j].Action
	})
	return items
}

type autoRecoverLaneState struct {
	runnableNow               bool
	waitingCooldown           bool
	waitingRetryWindow        bool
	waitingAuthRefresh        bool
	waitingLocalRestore       bool
	waitingProviderSession    bool
	waitingManualConfirmation bool
	waitingRetryLimit         bool
	waitingOther              bool
}

func classifyAutoRecoverLaneState(detail Detail, summary retryQueueSummary) autoRecoverLaneState {
	state := autoRecoverLaneState{}
	switch recoverDecisionCategory(detail, summary) {
	case "runnable_now":
		state.runnableNow = true
	case "waiting_cooldown":
		state.waitingCooldown = true
	case "waiting_retry_window":
		state.waitingRetryWindow = true
	case "waiting_auth_refresh":
		state.waitingAuthRefresh = true
	case "waiting_local_restore":
		state.waitingLocalRestore = true
	case "waiting_provider_session":
		state.waitingProviderSession = true
	case "waiting_manual_confirmation":
		state.waitingManualConfirmation = true
	case "waiting_retry_limit":
		state.waitingRetryLimit = true
	case "waiting_other":
		state.waitingOther = true
	}
	return state
}

func summarizeAutoRecoverPool(details []Detail, providers []provider.Entry) ([]AutoRecoverLane, int) {
	if len(details) == 0 {
		return nil, 0
	}
	accumulators := make(map[string]*autoRecoverLaneAccumulator)
	order := make([]string, 0)
	totalTasks := 0
	candidates := make([]recoverCandidate, 0)
	for _, detail := range details {
		if detail.Task.State != StateBlocked && detail.Task.State != StateCompletedWithErrors && detail.Runtime.ExecutionState != string(StateBlocked) {
			continue
		}
		ensureRuntimeState(&detail)
		syncRuntimeRetryQueue(&detail.Runtime, detail.Plan.Metadata, detail.Results)
		applyRetryQueueSummary(&detail.Runtime, detail.Plan.Metadata)
		candidate := buildRecoverCandidate(detail, protocolGroupForProviderKey(providers, detail.Task.TargetProvider))
		if !shouldIncludeAutoRecoverPool(detail, candidate.Summary) {
			continue
		}
		if candidate.Mode == "" {
			continue
		}
		candidates = append(candidates, candidate)
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		return recoverCandidateLess(candidates[i], candidates[j])
	})
	for _, candidate := range candidates {
		detail := candidate.Detail
		totalTasks++
		acc, ok := accumulators[candidate.Mode]
		if !ok {
			acc = &autoRecoverLaneAccumulator{
				mode:           candidate.Mode,
				advice:         autoRecoverGuidance(candidate.Mode),
				taskIDs:        make(map[string]struct{}),
				providers:      make(map[string]struct{}),
				profiles:       make(map[string]struct{}),
				protocolGroups: make(map[string]struct{}),
				retryClasses:   make(map[string]struct{}),
				blockedActions: make(map[string]struct{}),
				strategies:     make(map[string]struct{}),
			}
			accumulators[candidate.Mode] = acc
			order = append(order, candidate.Mode)
		}
		acc.taskIDs[detail.Task.ID] = struct{}{}
		acc.providers[detail.Task.TargetProvider] = struct{}{}
		acc.protocolGroups[recoverProtocolGroupBudgetKey(candidate.ProtocolGroup)] = struct{}{}
		if profileID := strings.TrimSpace(detail.TargetProfileID); profileID != "" {
			acc.profiles[profileID] = struct{}{}
		}
		if blockedAction := strings.TrimSpace(candidate.EffectiveAction); blockedAction != "" {
			acc.blockedActions[blockedAction] = struct{}{}
		}
		if strategy := strings.TrimSpace(recoverTaskStrategy(detail)); strategy != "" {
			acc.strategies[strategy] = struct{}{}
		}
		if retryClass := strings.TrimSpace(candidate.PrimaryRetryClass); retryClass != "" {
			acc.retryClasses[retryClass] = struct{}{}
		} else {
			for _, item := range detail.Runtime.RetryQueue {
				retryClass := strings.TrimSpace(item.RetryClass)
				if retryClass == "" {
					continue
				}
				acc.retryClasses[retryClass] = struct{}{}
			}
		}
		acc.suggestedProtocolGroupBudget = minPositiveBudget(acc.suggestedProtocolGroupBudget, recoverProtocolGroupBudget(detail))
		acc.suggestedProviderBudget = minPositiveBudget(acc.suggestedProviderBudget, recoverProviderBudget(detail))
		acc.suggestedProfileBudget = minPositiveBudget(acc.suggestedProfileBudget, recoverProfileBudget(detail))
		acc.queueItemCount += len(detail.Runtime.RetryQueue)
		acc.retryableNowCount += candidate.Summary.RetryableNowCount
		acc.cooldownCount += candidate.Summary.CooldownCount
		laneState := classifyAutoRecoverLaneState(detail, candidate.Summary)
		if laneState.runnableNow {
			acc.runnableTaskCount++
		}
		if laneState.waitingCooldown {
			acc.waitingCooldownTaskCount++
		}
		if laneState.waitingRetryWindow {
			acc.waitingRetryWindowTaskCount++
		}
		if laneState.waitingAuthRefresh {
			acc.waitingAuthRefreshTaskCount++
		}
		if laneState.waitingLocalRestore {
			acc.waitingLocalRestoreTaskCount++
		}
		if laneState.waitingProviderSession {
			acc.waitingProviderSessionTaskCount++
		}
		if laneState.waitingManualConfirmation {
			acc.waitingManualTaskCount++
		}
		if laneState.waitingRetryLimit {
			acc.waitingRetryLimitTaskCount++
		}
		if laneState.waitingOther {
			acc.waitingOtherTaskCount++
		}
		acc.uploadCheckpointEligible += candidate.Summary.UploadCheckpointEligible
		if acc.sampleTaskID == "" {
			acc.sampleTaskID = detail.Task.ID
			acc.sampleProvider = detail.Task.TargetProvider
			acc.sampleProtocolGroup = recoverProtocolGroupBudgetKey(candidate.ProtocolGroup)
			acc.sampleProfileID = detail.TargetProfileID
			acc.sampleStrategy = recoverTaskStrategy(detail)
		}
		nextRetryAt := strings.TrimSpace(candidate.Summary.NextRetryAt)
		if nextRetryAt != "" && (acc.nextRetryAt == "" || nextRetryAt < acc.nextRetryAt) {
			acc.nextRetryAt = nextRetryAt
		}
	}
	items := make([]AutoRecoverLane, 0, len(order))
	for _, mode := range order {
		acc := accumulators[mode]
		protocolGroups := sortedMapKeys(acc.protocolGroups)
		retryClasses := sortedMapKeys(acc.retryClasses)
		blockedActions := sortedMapKeys(acc.blockedActions)
		strategies := sortedMapKeys(acc.strategies)
		profileIDs := sortedMapKeys(acc.profiles)
		sampleProtocolGroup := strings.TrimSpace(acc.sampleProtocolGroup)
		if sampleProtocolGroup == "" {
			sampleProtocolGroup = firstStringValue(protocolGroups)
		}
		items = append(items, AutoRecoverLane{
			Mode:                            acc.mode,
			Advice:                          acc.advice,
			TaskCount:                       len(acc.taskIDs),
			ProviderCount:                   len(acc.providers),
			ProfileCount:                    len(acc.profiles),
			SuggestedProtocolGroupBudget:    acc.suggestedProtocolGroupBudget,
			SuggestedProviderBudget:         acc.suggestedProviderBudget,
			SuggestedProfileBudget:          acc.suggestedProfileBudget,
			QueueItemCount:                  acc.queueItemCount,
			RetryableNowCount:               acc.retryableNowCount,
			CooldownCount:                   acc.cooldownCount,
			RunnableTaskCount:               acc.runnableTaskCount,
			WaitingCooldownTaskCount:        acc.waitingCooldownTaskCount,
			WaitingRetryWindowTaskCount:     acc.waitingRetryWindowTaskCount,
			WaitingAuthRefreshTaskCount:     acc.waitingAuthRefreshTaskCount,
			WaitingLocalRestoreTaskCount:    acc.waitingLocalRestoreTaskCount,
			WaitingProviderSessionTaskCount: acc.waitingProviderSessionTaskCount,
			WaitingManualTaskCount:          acc.waitingManualTaskCount,
			WaitingRetryLimitTaskCount:      acc.waitingRetryLimitTaskCount,
			WaitingOtherTaskCount:           acc.waitingOtherTaskCount,
			UploadCheckpointEligible:        acc.uploadCheckpointEligible,
			ProtocolGroups:                  protocolGroups,
			RetryClasses:                    retryClasses,
			BlockedActions:                  blockedActions,
			Strategies:                      strategies,
			ProfileIDs:                      profileIDs,
			PrimaryRetryClass:               firstStringValue(retryClasses),
			PrimaryBlockedAction:            firstStringValue(blockedActions),
			NextRetryAt:                     acc.nextRetryAt,
			SampleTaskID:                    acc.sampleTaskID,
			SampleProvider:                  acc.sampleProvider,
			SampleProtocolGroup:             sampleProtocolGroup,
			SampleProfileID:                 acc.sampleProfileID,
			SampleStrategy:                  acc.sampleStrategy,
		})
	}
	sort.SliceStable(items, func(i, j int) bool {
		left := recoverCandidate{
			Mode:              items[i].Mode,
			ProtocolGroup:     items[i].SampleProtocolGroup,
			EffectiveAction:   items[i].PrimaryBlockedAction,
			PrimaryRetryClass: items[i].PrimaryRetryClass,
			Detail: Detail{
				Task: Task{ID: items[i].SampleTaskID},
				Runtime: RuntimeState{
					NextRetryAt: items[i].NextRetryAt,
				},
			},
		}
		right := recoverCandidate{
			Mode:              items[j].Mode,
			ProtocolGroup:     items[j].SampleProtocolGroup,
			EffectiveAction:   items[j].PrimaryBlockedAction,
			PrimaryRetryClass: items[j].PrimaryRetryClass,
			Detail: Detail{
				Task: Task{ID: items[j].SampleTaskID},
				Runtime: RuntimeState{
					NextRetryAt: items[j].NextRetryAt,
				},
			},
		}
		return recoverCandidateLess(left, right)
	})
	return items, totalTasks
}

func summarizeAutoRecoverStateCounts(items []AutoRecoverLane) (int, int, int, int, int, int, int, int, int) {
	runnable := 0
	waitingCooldown := 0
	waitingRetryWindow := 0
	waitingAuthRefresh := 0
	waitingLocalRestore := 0
	waitingProviderSession := 0
	waitingManual := 0
	waitingRetryLimit := 0
	waitingOther := 0
	for _, item := range items {
		runnable += item.RunnableTaskCount
		waitingCooldown += item.WaitingCooldownTaskCount
		waitingRetryWindow += item.WaitingRetryWindowTaskCount
		waitingAuthRefresh += item.WaitingAuthRefreshTaskCount
		waitingLocalRestore += item.WaitingLocalRestoreTaskCount
		waitingProviderSession += item.WaitingProviderSessionTaskCount
		waitingManual += item.WaitingManualTaskCount
		waitingRetryLimit += item.WaitingRetryLimitTaskCount
		waitingOther += item.WaitingOtherTaskCount
	}
	return runnable, waitingCooldown, waitingRetryWindow, waitingAuthRefresh, waitingLocalRestore, waitingProviderSession, waitingManual, waitingRetryLimit, waitingOther
}

func taskEvidenceSummary(ctx context.Context, store *sqlitestore.Store, providers []provider.Entry) (EvidenceSummary, error) {
	var summary EvidenceSummary
	protocolGroups := make(map[string]string, len(providers))
	for _, entry := range providers {
		protocolGroups[entry.Meta.Key] = strings.TrimSpace(entry.Meta.ProtocolGroup)
	}
	if err := store.DB().QueryRowContext(ctx, `SELECT COUNT(1) FROM tasks`).Scan(&summary.TotalTasks); err != nil {
		return summary, err
	}
	if err := store.DB().QueryRowContext(ctx, `SELECT COUNT(1) FROM tasks WHERE state IN ('completed', 'completed_with_errors')`).Scan(&summary.CompletedTasks); err != nil {
		return summary, err
	}
	if err := store.DB().QueryRowContext(ctx, `SELECT COUNT(1) FROM task_results WHERE status = 'failed'`).Scan(&summary.FailedResultCount); err != nil {
		return summary, err
	}
	if err := store.DB().QueryRowContext(ctx, `SELECT COUNT(1) FROM task_results WHERE status = 'done'`).Scan(&summary.DoneResultCount); err != nil {
		return summary, err
	}
	if err := store.DB().QueryRowContext(ctx, `SELECT COUNT(1) FROM task_results WHERE status = 'skipped'`).Scan(&summary.SkippedResultCount); err != nil {
		return summary, err
	}
	rows, err := store.DB().QueryContext(ctx, `SELECT status, payload_json FROM task_results WHERE payload_json IS NOT NULL AND payload_json != ''`)
	if err != nil {
		return summary, err
	}
	defer rows.Close()
	for rows.Next() {
		var (
			status      string
			payloadJSON string
		)
		if err := rows.Scan(&status, &payloadJSON); err != nil {
			return summary, err
		}
		payload := map[string]interface{}{}
		if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
			return summary, err
		}
		if isPendingRelayResult(Result{Status: status, Payload: payload}) {
			summary.PendingResultCount++
		}
		if _, ok := riskHitFromPayload(payload); ok {
			summary.RiskHitCount++
		}
	}
	if err := rows.Err(); err != nil {
		return summary, err
	}
	details, err := listTasks(ctx, store)
	if err != nil {
		return summary, err
	}
	blockedDetails := make([]Detail, 0)
	for _, detail := range details {
		ensureRuntimeState(&detail)
		summary.SourceDeletionCount += detail.Runtime.SourceDeletionCount
		if (detail.Runtime.UploadCheckpoint != nil && uploadCheckpointCanResume(detail.Runtime.UploadCheckpoint)) || detailHasUploadCheckpointEvidence(detail.Results) || detail.Runtime.AutoRecoverReason == "upload_checkpoint_auto_resume" {
			summary.UploadCheckpointTaskCount++
		}
		if detail.Runtime.AutoRecovered && detail.Runtime.AutoRecoverReason == "upload_checkpoint_auto_resume" {
			summary.UploadCheckpointResumeTaskCount++
			checkpoint := detail.Runtime.UploadCheckpoint
			if checkpoint == nil {
				retryPaths := []string{}
				if path := normalizeScanPath(lastCompletedResultPath(detail.Results)); path != "" {
					retryPaths = append(retryPaths, path)
				}
				if len(retryPaths) == 0 {
					retryPaths = metadataStringSlice(detail.Plan.Metadata, "retrySelectedPaths")
				}
				checkpoint = firstRetryUploadCheckpoint(detail.Plan.Metadata, retryPaths)
			}
			if checkpoint == nil {
				for idx := len(detail.Results) - 1; idx >= 0; idx-- {
					if candidate := uploadCheckpointFromResult(detail.Results[idx]); candidate != nil {
						checkpoint = candidate
						break
					}
				}
			}
			if detail.Runtime.UploadCheckpoint != nil {
				if path := normalizeScanPath(detail.Runtime.UploadCheckpoint.ItemPath); path != "" {
					summary.UploadCheckpointResumeSamplePaths = appendUniqueString(summary.UploadCheckpointResumeSamplePaths, path)
				}
			} else if path := normalizeScanPath(lastCompletedResultPath(detail.Results)); path != "" {
				summary.UploadCheckpointResumeSamplePaths = appendUniqueString(summary.UploadCheckpointResumeSamplePaths, path)
			}
			if summary.UploadCheckpointResumeSampleTaskID == "" {
				summary.UploadCheckpointResumeSampleTaskID = detail.Task.ID
				summary.UploadCheckpointResumeSampleProvider = detail.Task.TargetProvider
				summary.UploadCheckpointResumeSampleProtocol = firstNonEmpty(protocolGroups[detail.Task.TargetProvider], strings.TrimSpace(detail.Task.TargetProvider))
				summary.UploadCheckpointResumeSampleProfileID = detail.TargetProfileID
				if checkpoint != nil {
					summary.UploadCheckpointResumeSampleUploadID = strings.TrimSpace(checkpoint.UploadID)
					summary.UploadCheckpointResumeSampleNextPart = checkpoint.NextPartNumber
					summary.UploadCheckpointResumeSamplePartCount = checkpoint.PartCount
					summary.UploadCheckpointResumeSampleUploaded = checkpoint.UploadedPartCount
				}
			}
		}
		if detail.Task.State != StateBlocked {
			continue
		}
		summary.BlockedTasks++
		blockedDetails = append(blockedDetails, detail)
	}
	summary.BlockedActions = summarizeBlockedActions(blockedDetails)
	summary.AutoRecoverPool, summary.AutoRecoverTasks = summarizeAutoRecoverPool(details, providers)
	summary.AutoRecoverRunnableTasks,
		summary.AutoRecoverWaitingCooldownTasks,
		summary.AutoRecoverWaitingRetryWindowTasks,
		summary.AutoRecoverWaitingAuthRefreshTasks,
		summary.AutoRecoverWaitingLocalRestoreTasks,
		summary.AutoRecoverWaitingProviderSessionTasks,
		summary.AutoRecoverWaitingManualTasks,
		summary.AutoRecoverWaitingRetryLimitTasks,
		summary.AutoRecoverWaitingOtherTasks = summarizeAutoRecoverStateCounts(summary.AutoRecoverPool)
	results, err := recentTaskResults(ctx, store, 10)
	if err != nil {
		return summary, err
	}
	probes, err := recentProviderProbes(ctx, store, 10)
	if err != nil {
		return summary, err
	}
	summary.RecentResults = results
	summary.RecentProbes = probes
	if len(details) > 0 {
		latest := details[0]
		summary.ExecutionMode = executionModeString(latest.Plan.Metadata)
		summary.ScanMode = stringValue(latest.Plan.Metadata["scanMode"])
		summary.SourceDeletePolicy = sourceDeletePolicyString(latest.Plan.Metadata["sourceDeletePolicy"])
		if snapshot, ok, err := latestProviderStatusSnapshot(ctx, store, latest.Task.TargetProvider); err != nil {
			return summary, err
		} else if ok {
			if summary.ExecutionMode == "" {
				summary.ExecutionMode = executionModeString(snapshot.Summary)
			}
			if summary.ScanMode == "" {
				summary.ScanMode = stringValue(snapshot.Summary["scanMode"])
			}
			if summary.SourceDeletePolicy == "" {
				summary.SourceDeletePolicy = sourceDeletePolicyString(snapshot.Summary["sourceDeletePolicy"])
			}
		}
	}
	if snapshot, ok, err := mostRecentProviderStatusSnapshot(ctx, store); err != nil {
		return summary, err
	} else if ok {
		if summary.ExecutionMode == "" {
			summary.ExecutionMode = executionModeString(snapshot.Summary)
		}
		if summary.ScanMode == "" {
			summary.ScanMode = stringValue(snapshot.Summary["scanMode"])
		}
		if summary.SourceDeletePolicy == "" {
			summary.SourceDeletePolicy = sourceDeletePolicyString(snapshot.Summary["sourceDeletePolicy"])
		}
	}
	coverage, _, err := protocolCoverageSummary(providers, details)
	if err != nil {
		return summary, err
	}
	summary.ProtocolCoverage = coverage
	return summary, nil
}

func lastCompletedResultPath(results []Result) string {
	for idx := len(results) - 1; idx >= 0; idx-- {
		if path := strings.TrimSpace(stringValue(results[idx].Payload["path"])); path != "" {
			return path
		}
	}
	return ""
}

func detailHasUploadCheckpointEvidence(results []Result) bool {
	for _, result := range results {
		if uploadCheckpointFromResult(result) != nil {
			return true
		}
	}
	return false
}

func appendUniqueString(items []string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return items
	}
	for _, item := range items {
		if item == value {
			return items
		}
	}
	return append(items, value)
}

func providerStatusSummary(ctx context.Context, store *sqlitestore.Store, providers []provider.Entry) ([]StatusSummary, error) {
	allDetails, err := listTasks(ctx, store)
	if err != nil {
		return nil, err
	}
	_, coverageByGroup, err := protocolCoverageSummary(providers, allDetails)
	if err != nil {
		return nil, err
	}
	items := make([]StatusSummary, 0, len(providers))
	for _, entry := range providers {
		item := StatusSummary{ProviderKey: entry.Meta.Key, ProtocolGroup: protocolGroupForProvider(entry)}
		if err := store.DB().QueryRowContext(ctx, `SELECT COUNT(1) FROM auth_profiles WHERE provider_key = ?`, entry.Meta.Key).Scan(&item.ProfileCount); err != nil {
			return nil, err
		}
		if err := store.DB().QueryRowContext(ctx, `SELECT COUNT(1) FROM tasks WHERE target_provider = ?`, entry.Meta.Key).Scan(&item.TaskCount); err != nil {
			return nil, err
		}
		if err := store.DB().QueryRowContext(ctx, `SELECT COUNT(1) FROM tasks WHERE target_provider = ? AND state IN ('completed', 'completed_with_errors')`, entry.Meta.Key).Scan(&item.CompletedCount); err != nil {
			return nil, err
		}
		blockedDetails := make([]Detail, 0)
		providerDetails := make([]Detail, 0)
		for _, detail := range allDetails {
			if detail.Task.TargetProvider != entry.Meta.Key {
				continue
			}
			providerDetails = append(providerDetails, detail)
			if detail.Task.State != StateBlocked {
				continue
			}
			item.BlockedCount++
			blockedDetails = append(blockedDetails, detail)
		}
		snapshot, ok, err := latestProviderStatusSnapshot(ctx, store, entry.Meta.Key)
		if err != nil {
			return nil, err
		}
		if ok {
			item.LastObservedAt = snapshot.CreatedAt
			item.SnapshotSummary = snapshot.Summary
			item.LastTaskState = stringValue(snapshot.Summary["lastTaskState"])
			item.LatestProbe = stringValue(snapshot.Summary["latestProbe"])
		}
		if item.SnapshotSummary == nil {
			item.SnapshotSummary = map[string]interface{}{}
		}
		if coverage, ok := coverageByGroup[protocolGroupForProvider(entry)]; ok {
			coverageCopy := coverage
			item.ProtocolCoverage = &coverageCopy
			item.SnapshotSummary["protocolCoverage"] = coverageCopy
		}
		item.SnapshotSummary["blockedCount"] = item.BlockedCount
		item.SnapshotSummary["blockedActions"] = summarizeBlockedActions(blockedDetails)
		autoRecoverPool, autoRecoverCount := summarizeAutoRecoverPool(providerDetails, providers)
		item.AutoRecoverCount = autoRecoverCount
		item.SnapshotSummary["autoRecoverPool"] = autoRecoverPool
		item.SnapshotSummary["autoRecoverCount"] = item.AutoRecoverCount
		runnable, waitingCooldown, waitingRetryWindow, waitingAuthRefresh, waitingLocalRestore, waitingProviderSession, waitingManual, waitingRetryLimit, waitingOther := summarizeAutoRecoverStateCounts(autoRecoverPool)
		item.SnapshotSummary["autoRecoverRunnableTasks"] = runnable
		item.SnapshotSummary["autoRecoverWaitingCooldownTasks"] = waitingCooldown
		item.SnapshotSummary["autoRecoverWaitingRetryWindowTasks"] = waitingRetryWindow
		item.SnapshotSummary["autoRecoverWaitingAuthRefreshTasks"] = waitingAuthRefresh
		item.SnapshotSummary["autoRecoverWaitingLocalRestoreTasks"] = waitingLocalRestore
		item.SnapshotSummary["autoRecoverWaitingProviderSessionTasks"] = waitingProviderSession
		item.SnapshotSummary["autoRecoverWaitingManualTasks"] = waitingManual
		item.SnapshotSummary["autoRecoverWaitingRetryLimitTasks"] = waitingRetryLimit
		item.SnapshotSummary["autoRecoverWaitingOtherTasks"] = waitingOther
		items = append(items, item)
	}
	return items, nil
}

func protocolCoverageSummary(providers []provider.Entry, details []Detail) ([]ProtocolCoverage, map[string]ProtocolCoverage, error) {
	if len(providers) == 0 {
		return nil, map[string]ProtocolCoverage{}, nil
	}

	type coverageState struct {
		row          ProtocolCoverage
		providerSeen map[string]struct{}
		sampleTaken  bool
	}

	states := make(map[string]*coverageState)
	order := make([]string, 0)
	for _, entry := range providers {
		group := protocolGroupForProvider(entry)
		state, ok := states[group]
		if !ok {
			state = &coverageState{
				row: ProtocolCoverage{
					ProtocolGroup: group,
				},
				providerSeen: make(map[string]struct{}),
			}
			states[group] = state
			order = append(order, group)
		}
		if _, ok := state.providerSeen[entry.Meta.Key]; ok {
			continue
		}
		state.providerSeen[entry.Meta.Key] = struct{}{}
		state.row.ProviderCount++
		state.row.ProviderKeys = append(state.row.ProviderKeys, entry.Meta.Key)
	}

	for _, detail := range details {
		entry, ok := providerEntryByKey(providers, detail.Task.TargetProvider)
		if !ok {
			continue
		}
		group := protocolGroupForProvider(entry)
		state, ok := states[group]
		if !ok {
			continue
		}
		state.row.TaskCount++
		if detail.Task.State == StateCompleted || detail.Task.State == StateCompletedWithErrors {
			state.row.CompletedTaskCount++
		}
		if detail.Task.CompletionKind != CompletionKindRealTransfer {
			continue
		}
		state.row.RealSuccessTaskCount++
		if !state.sampleTaken {
			state.sampleTaken = true
			state.row.SampleTaskID = detail.Task.ID
			state.row.SampleProviderKey = detail.Task.TargetProvider
			state.row.SampleTaskState = string(detail.Task.State)
			state.row.SampleCompletionKind = string(detail.Task.CompletionKind)
			state.row.LastObservedAt = detail.Task.UpdatedAt
		}
	}

	rows := make([]ProtocolCoverage, 0, len(order))
	coverageByGroup := make(map[string]ProtocolCoverage, len(order))
	for _, group := range order {
		state := states[group]
		state.row.HasRealSuccessSample = state.row.RealSuccessTaskCount > 0
		if len(state.row.ProviderKeys) == 0 {
			state.row.ProviderKeys = nil
		}
		rows = append(rows, state.row)
		coverageByGroup[group] = state.row
	}
	return rows, coverageByGroup, nil
}

func providerEntryByKey(entries []provider.Entry, key string) (provider.Entry, bool) {
	for _, entry := range entries {
		if entry.Meta.Key == key {
			return entry, true
		}
	}
	return provider.Entry{}, false
}

func protocolGroupForProviderKey(entries []provider.Entry, key string) string {
	if entry, ok := providerEntryByKey(entries, key); ok {
		return protocolGroupForProvider(entry)
	}
	return recoverProtocolGroupBudgetKey(key)
}

func protocolGroupForProvider(entry provider.Entry) string {
	group := strings.TrimSpace(entry.Meta.ProtocolGroup)
	if group == "" {
		return recoverProtocolGroupBudgetKey(entry.Meta.Key)
	}
	return group
}

func getTaskItems(ctx context.Context, store *sqlitestore.Store, taskID string) ([]Item, error) {
	rows, err := store.DB().QueryContext(ctx, `
SELECT id, task_id, path, size
FROM task_items
WHERE task_id = ?
ORDER BY created_at ASC`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []Item{}
	for rows.Next() {
		var item Item
		if err := rows.Scan(&item.ID, &item.TaskID, &item.Path, &item.Size); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func getTaskResults(ctx context.Context, store *sqlitestore.Store, taskID string) ([]Result, error) {
	rows, err := store.DB().QueryContext(ctx, `
SELECT id, task_id, task_item_id, status, mode, message, payload_json, created_at
FROM task_results
WHERE task_id = ?
ORDER BY created_at ASC`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []Result{}
	for rows.Next() {
		var (
			item        Result
			payloadJSON string
		)
		if err := rows.Scan(&item.ID, &item.TaskID, &item.ItemID, &item.Status, &item.Mode, &item.Message, &payloadJSON, &item.CreatedAt); err != nil {
			return nil, err
		}
		item.Payload = map[string]interface{}{}
		if payloadJSON != "" {
			if err := json.Unmarshal([]byte(payloadJSON), &item.Payload); err != nil {
				return nil, err
			}
			item.ConflictAction = stringValue(item.Payload["conflictAction"])
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func saveProviderProbe(ctx context.Context, store *sqlitestore.Store, probe ProviderProbe) error {
	payloadJSON, err := json.Marshal(probe.Payload)
	if err != nil {
		return err
	}
	_, err = store.DB().ExecContext(ctx, `
INSERT INTO provider_probes(id, provider_key, profile_id, status, payload_json, created_at)
VALUES (?, ?, ?, ?, ?, ?)`,
		probe.ID, probe.ProviderKey, probe.ProfileID, probe.Status, string(payloadJSON), probe.CreatedAt,
	)
	return err
}

func saveProviderStatusSnapshot(ctx context.Context, store *sqlitestore.Store, snapshot ProviderStatus) error {
	summaryJSON, err := json.Marshal(snapshot.Summary)
	if err != nil {
		return err
	}
	_, err = store.DB().ExecContext(ctx, `
INSERT INTO provider_status_snapshots(id, provider_key, summary_json, created_at)
VALUES (?, ?, ?, ?)`,
		snapshot.ID, snapshot.ProviderKey, string(summaryJSON), snapshot.CreatedAt,
	)
	return err
}

func saveEvidenceReport(ctx context.Context, store *sqlitestore.Store, report EvidenceReport) (EvidenceReportRecord, error) {
	record := EvidenceReportRecord{
		ID:                     uuid.NewString(),
		GeneratedAt:            report.GeneratedAt,
		Title:                  report.Title,
		Note:                   report.Note,
		Markdown:               report.Markdown,
		Summary:                report.Summary,
		Statuses:               report.Statuses,
		SmokeSummaries:         report.SmokeSummaries,
		SmokeMatrix:            report.SmokeMatrix,
		ProviderSmokeProviders: report.ProviderSmokeProviders,
		Samples:                report.Samples,
	}
	summaryJSON, err := json.Marshal(record.Summary)
	if err != nil {
		return EvidenceReportRecord{}, err
	}
	statusesJSON, err := json.Marshal(record.Statuses)
	if err != nil {
		return EvidenceReportRecord{}, err
	}
	smokeSummariesJSON, err := json.Marshal(record.SmokeSummaries)
	if err != nil {
		return EvidenceReportRecord{}, err
	}
	smokeMatrixJSON, err := json.Marshal(record.SmokeMatrix)
	if err != nil {
		return EvidenceReportRecord{}, err
	}
	providerSmokeProvidersJSON, err := json.Marshal(record.ProviderSmokeProviders)
	if err != nil {
		return EvidenceReportRecord{}, err
	}
	samplesJSON, err := json.Marshal(record.Samples)
	if err != nil {
		return EvidenceReportRecord{}, err
	}
	_, err = store.DB().ExecContext(ctx, `
INSERT INTO evidence_reports(id, generated_at, title, note, markdown, summary_json, statuses_json, smoke_summaries_json, smoke_matrix_json, provider_smoke_providers_json, samples_json)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		record.ID, record.GeneratedAt, record.Title, record.Note, record.Markdown, string(summaryJSON), string(statusesJSON), string(smokeSummariesJSON), string(smokeMatrixJSON), string(providerSmokeProvidersJSON), string(samplesJSON),
	)
	if err != nil {
		return EvidenceReportRecord{}, err
	}
	return record, nil
}

func listEvidenceReports(ctx context.Context, store *sqlitestore.Store) ([]EvidenceReportRecord, error) {
	rows, err := store.DB().QueryContext(ctx, `
SELECT id, generated_at, title, note, markdown, summary_json, statuses_json, smoke_summaries_json, smoke_matrix_json, provider_smoke_providers_json, samples_json
FROM evidence_reports
ORDER BY generated_at DESC, rowid DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]EvidenceReportRecord, 0)
	for rows.Next() {
		var (
			item                       EvidenceReportRecord
			title                      string
			note                       string
			summaryJSON                string
			statusesJSON               string
			smokeSummariesJSON         string
			smokeMatrixJSON            string
			providerSmokeProvidersJSON string
			samplesJSON                string
		)
		if err := rows.Scan(&item.ID, &item.GeneratedAt, &title, &note, &item.Markdown, &summaryJSON, &statusesJSON, &smokeSummariesJSON, &smokeMatrixJSON, &providerSmokeProvidersJSON, &samplesJSON); err != nil {
			return nil, err
		}
		item.Title = title
		item.Note = note
		if err := json.Unmarshal([]byte(summaryJSON), &item.Summary); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(statusesJSON), &item.Statuses); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(smokeSummariesJSON), &item.SmokeSummaries); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(smokeMatrixJSON), &item.SmokeMatrix); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(providerSmokeProvidersJSON), &item.ProviderSmokeProviders); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(samplesJSON), &item.Samples); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func getEvidenceReport(ctx context.Context, store *sqlitestore.Store, id string) (EvidenceReportRecord, bool, error) {
	row := store.DB().QueryRowContext(ctx, `
SELECT id, generated_at, title, note, markdown, summary_json, statuses_json, smoke_summaries_json, smoke_matrix_json, provider_smoke_providers_json, samples_json
FROM evidence_reports
WHERE id = ?`, id)

	var (
		item                       EvidenceReportRecord
		summaryJSON                string
		statusesJSON               string
		smokeSummariesJSON         string
		smokeMatrixJSON            string
		providerSmokeProvidersJSON string
		samplesJSON                string
	)
	if err := row.Scan(&item.ID, &item.GeneratedAt, &item.Title, &item.Note, &item.Markdown, &summaryJSON, &statusesJSON, &smokeSummariesJSON, &smokeMatrixJSON, &providerSmokeProvidersJSON, &samplesJSON); err != nil {
		if err == sql.ErrNoRows {
			return EvidenceReportRecord{}, false, nil
		}
		return EvidenceReportRecord{}, false, err
	}
	if err := json.Unmarshal([]byte(summaryJSON), &item.Summary); err != nil {
		return EvidenceReportRecord{}, false, err
	}
	if err := json.Unmarshal([]byte(statusesJSON), &item.Statuses); err != nil {
		return EvidenceReportRecord{}, false, err
	}
	if err := json.Unmarshal([]byte(smokeSummariesJSON), &item.SmokeSummaries); err != nil {
		return EvidenceReportRecord{}, false, err
	}
	if err := json.Unmarshal([]byte(smokeMatrixJSON), &item.SmokeMatrix); err != nil {
		return EvidenceReportRecord{}, false, err
	}
	if err := json.Unmarshal([]byte(providerSmokeProvidersJSON), &item.ProviderSmokeProviders); err != nil {
		return EvidenceReportRecord{}, false, err
	}
	if err := json.Unmarshal([]byte(samplesJSON), &item.Samples); err != nil {
		return EvidenceReportRecord{}, false, err
	}
	return item, true, nil
}

func saveProviderSmokeRecord(ctx context.Context, store *sqlitestore.Store, record ProviderSmokeRecord) (ProviderSmokeRecord, error) {
	operationsJSON, err := json.Marshal(record.Operations)
	if err != nil {
		return ProviderSmokeRecord{}, err
	}
	environmentJSON, err := json.Marshal(record.Environment)
	if err != nil {
		return ProviderSmokeRecord{}, err
	}
	_, err = store.DB().ExecContext(ctx, `
INSERT INTO provider_smoke_records(id, provider_key, protocol_group, auth_mode, category, result, title, note, markdown, operations_json, environment_json, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		record.ID, record.ProviderKey, record.ProtocolGroup, record.AuthMode, record.Category, record.Result, record.Title, record.Note, record.Markdown, string(operationsJSON), string(environmentJSON), record.CreatedAt,
	)
	if err != nil {
		return ProviderSmokeRecord{}, err
	}
	return record, nil
}

func listProviderSmokeRecords(ctx context.Context, store *sqlitestore.Store) ([]ProviderSmokeRecord, error) {
	rows, err := store.DB().QueryContext(ctx, `
SELECT id, provider_key, protocol_group, auth_mode, category, result, title, note, markdown, operations_json, environment_json, created_at
FROM provider_smoke_records
ORDER BY created_at DESC, rowid DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]ProviderSmokeRecord, 0)
	for rows.Next() {
		var (
			item            ProviderSmokeRecord
			operationsJSON  string
			environmentJSON string
		)
		if err := rows.Scan(&item.ID, &item.ProviderKey, &item.ProtocolGroup, &item.AuthMode, &item.Category, &item.Result, &item.Title, &item.Note, &item.Markdown, &operationsJSON, &environmentJSON, &item.CreatedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(operationsJSON), &item.Operations); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(environmentJSON), &item.Environment); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func getProviderSmokeRecord(ctx context.Context, store *sqlitestore.Store, id string) (ProviderSmokeRecord, bool, error) {
	row := store.DB().QueryRowContext(ctx, `
SELECT id, provider_key, protocol_group, auth_mode, category, result, title, note, markdown, operations_json, environment_json, created_at
FROM provider_smoke_records
WHERE id = ?`, id)

	var (
		item            ProviderSmokeRecord
		operationsJSON  string
		environmentJSON string
	)
	if err := row.Scan(&item.ID, &item.ProviderKey, &item.ProtocolGroup, &item.AuthMode, &item.Category, &item.Result, &item.Title, &item.Note, &item.Markdown, &operationsJSON, &environmentJSON, &item.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return ProviderSmokeRecord{}, false, nil
		}
		return ProviderSmokeRecord{}, false, err
	}
	if err := json.Unmarshal([]byte(operationsJSON), &item.Operations); err != nil {
		return ProviderSmokeRecord{}, false, err
	}
	if err := json.Unmarshal([]byte(environmentJSON), &item.Environment); err != nil {
		return ProviderSmokeRecord{}, false, err
	}
	return item, true, nil
}

func buildProviderStatusSnapshot(ctx context.Context, store *sqlitestore.Store, detail Detail, probe ProviderProbe, createdAt string) (ProviderStatus, error) {
	var (
		profileCount   int
		taskCount      int
		completedCount int
	)
	if err := store.DB().QueryRowContext(ctx, `SELECT COUNT(1) FROM auth_profiles WHERE provider_key = ?`, detail.Task.TargetProvider).Scan(&profileCount); err != nil {
		return ProviderStatus{}, err
	}
	if err := store.DB().QueryRowContext(ctx, `SELECT COUNT(1) FROM tasks WHERE target_provider = ?`, detail.Task.TargetProvider).Scan(&taskCount); err != nil {
		return ProviderStatus{}, err
	}
	if err := store.DB().QueryRowContext(ctx, `SELECT COUNT(1) FROM tasks WHERE target_provider = ? AND state IN ('completed', 'completed_with_errors')`, detail.Task.TargetProvider).Scan(&completedCount); err != nil {
		return ProviderStatus{}, err
	}
	return ProviderStatus{
		ID:          uuid.NewString(),
		ProviderKey: detail.Task.TargetProvider,
		Summary: map[string]interface{}{
			"profileCount":                   profileCount,
			"taskCount":                      taskCount,
			"completedCount":                 completedCount,
			"lastTaskId":                     detail.Task.ID,
			"lastTaskState":                  detail.Task.State,
			"latestProbe":                    probe.Status,
			"executionMode":                  detail.Plan.Metadata["executionMode"],
			"recommendedExecutionMode":       detail.Plan.Metadata["recommendedExecutionMode"],
			"recommendedExecutionModeReason": detail.Plan.Metadata["recommendedExecutionModeReason"],
			"sourceDeletePolicy":             detail.Plan.Metadata["sourceDeletePolicy"],
			"scanMode":                       scanModeValue(detail.Plan.Metadata),
			"retryMode":                      detail.Plan.Metadata["retryMode"],
			"retryScope":                     detail.Plan.Metadata["retryScope"],
			"retrySelectedPaths":             detail.Plan.Metadata["retrySelectedPaths"],
			"retrySelectedPathCount":         detail.Plan.Metadata["retrySelectedPathCount"],
			"riskProfile":                    detail.Plan.Metadata["riskProfile"],
			"riskProfileResolution":          detail.Plan.Metadata["riskProfileResolution"],
			"riskOverride":                   detail.Plan.Metadata["riskOverride"],
			"runtime":                        detail.Runtime,
			"doneCount":                      detail.Runtime.DoneCount,
			"skippedCount":                   detail.Runtime.SkippedCount,
			"failedCount":                    detail.Runtime.FailedCount,
			"pendingCount":                   detail.Runtime.PendingCount,
			"sourceDeletionCount":            detail.Runtime.SourceDeletionCount,
			"sourceDeletionRecords":          detail.Runtime.SourceDeletionRecords,
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
		},
		CreatedAt: createdAt,
	}, nil
}

func scanModeValue(values map[string]interface{}) string {
	if mode := stringValue(values["scanMode"]); mode != "" {
		return mode
	}
	executionMode, err := executionModeFromMetadata(values)
	if err != nil {
		return ""
	}
	return scanModeForExecutionMode(executionMode)
}

func latestProviderStatusSnapshot(ctx context.Context, store *sqlitestore.Store, providerKey string) (ProviderStatus, bool, error) {
	rows, err := store.DB().QueryContext(ctx, `
SELECT id, provider_key, summary_json, created_at
FROM provider_status_snapshots
WHERE provider_key = ?
ORDER BY created_at DESC, rowid DESC
`, providerKey)
	if err != nil {
		return ProviderStatus{}, false, err
	}
	defer rows.Close()

	var fallback *ProviderStatus
	for rows.Next() {
		var (
			item        ProviderStatus
			summaryJSON string
		)
		if err := rows.Scan(&item.ID, &item.ProviderKey, &summaryJSON, &item.CreatedAt); err != nil {
			return ProviderStatus{}, false, err
		}
		item.Summary = map[string]interface{}{}
		if summaryJSON != "" {
			if err := json.Unmarshal([]byte(summaryJSON), &item.Summary); err != nil {
				return ProviderStatus{}, false, err
			}
		}
		if fallback == nil {
			copyItem := item
			fallback = &copyItem
		}
		if providerStatusHasRuntimeEvidence(item.Summary) {
			return item, true, nil
		}
	}
	if err := rows.Err(); err != nil {
		return ProviderStatus{}, false, err
	}
	if fallback == nil {
		return ProviderStatus{}, false, nil
	}
	return *fallback, true, nil
}

func mostRecentProviderStatusSnapshot(ctx context.Context, store *sqlitestore.Store) (ProviderStatus, bool, error) {
	rows, err := store.DB().QueryContext(ctx, `
SELECT id, provider_key, summary_json, created_at
FROM provider_status_snapshots
ORDER BY created_at DESC, rowid DESC
`)
	if err != nil {
		return ProviderStatus{}, false, err
	}
	defer rows.Close()

	var fallback *ProviderStatus
	for rows.Next() {
		var (
			item        ProviderStatus
			summaryJSON string
		)
		if err := rows.Scan(&item.ID, &item.ProviderKey, &summaryJSON, &item.CreatedAt); err != nil {
			return ProviderStatus{}, false, err
		}
		item.Summary = map[string]interface{}{}
		if summaryJSON != "" {
			if err := json.Unmarshal([]byte(summaryJSON), &item.Summary); err != nil {
				return ProviderStatus{}, false, err
			}
		}
		if fallback == nil {
			copyItem := item
			fallback = &copyItem
		}
		if providerStatusHasRuntimeEvidence(item.Summary) {
			return item, true, nil
		}
	}
	if err := rows.Err(); err != nil {
		return ProviderStatus{}, false, err
	}
	if fallback == nil {
		return ProviderStatus{}, false, nil
	}
	return *fallback, true, nil
}

func recentProviderProbes(ctx context.Context, store *sqlitestore.Store, limit int) ([]ProviderProbe, error) {
	rows, err := store.DB().QueryContext(ctx, fmt.Sprintf(`
SELECT id, provider_key, profile_id, status, payload_json, created_at
FROM provider_probes
ORDER BY created_at DESC, rowid DESC
LIMIT %d`, limit))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []ProviderProbe{}
	for rows.Next() {
		var (
			item        ProviderProbe
			payloadJSON string
		)
		if err := rows.Scan(&item.ID, &item.ProviderKey, &item.ProfileID, &item.Status, &payloadJSON, &item.CreatedAt); err != nil {
			return nil, err
		}
		item.Payload = map[string]interface{}{}
		if payloadJSON != "" {
			if err := json.Unmarshal([]byte(payloadJSON), &item.Payload); err != nil {
				return nil, err
			}
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func recentTaskResults(ctx context.Context, store *sqlitestore.Store, limit int) ([]Result, error) {
	rows, err := store.DB().QueryContext(ctx, fmt.Sprintf(`
SELECT id, task_id, task_item_id, status, mode, message, payload_json, created_at
FROM task_results
ORDER BY created_at DESC, rowid DESC
LIMIT %d`, limit))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []Result{}
	for rows.Next() {
		var (
			item        Result
			payloadJSON string
		)
		if err := rows.Scan(&item.ID, &item.TaskID, &item.ItemID, &item.Status, &item.Mode, &item.Message, &payloadJSON, &item.CreatedAt); err != nil {
			return nil, err
		}
		item.Payload = map[string]interface{}{}
		if payloadJSON != "" {
			if err := json.Unmarshal([]byte(payloadJSON), &item.Payload); err != nil {
				return nil, err
			}
			item.ConflictAction = stringValue(item.Payload["conflictAction"])
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func stringValue(raw interface{}) string {
	value, _ := raw.(string)
	return value
}

func sourceDeletePolicyString(raw interface{}) string {
	switch value := raw.(type) {
	case string:
		return value
	case planner.SourceDeletePolicy:
		return string(value)
	default:
		return ""
	}
}

func providerStatusHasRuntimeEvidence(summary map[string]interface{}) bool {
	if summary == nil {
		return false
	}
	if stringValue(summary["scanMode"]) != "" {
		return true
	}
	if stringValue(summary["executionMode"]) != "" {
		return true
	}
	if _, ok := summary["runtime"].(map[string]interface{}); ok {
		return true
	}
	return false
}
