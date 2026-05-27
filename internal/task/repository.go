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

func taskEvidenceSummary(ctx context.Context, store *sqlitestore.Store) (EvidenceSummary, error) {
	var summary EvidenceSummary
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
		if detail.Task.State != StateBlocked {
			continue
		}
		summary.BlockedTasks++
		blockedDetails = append(blockedDetails, detail)
	}
	summary.BlockedActions = summarizeBlockedActions(blockedDetails)
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
	return summary, nil
}

func providerStatusSummary(ctx context.Context, store *sqlitestore.Store, providers []provider.Entry) ([]StatusSummary, error) {
	allDetails, err := listTasks(ctx, store)
	if err != nil {
		return nil, err
	}
	items := make([]StatusSummary, 0, len(providers))
	for _, entry := range providers {
		item := StatusSummary{ProviderKey: entry.Meta.Key}
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
		for _, detail := range allDetails {
			if detail.Task.TargetProvider != entry.Meta.Key || detail.Task.State != StateBlocked {
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
		item.SnapshotSummary["blockedCount"] = item.BlockedCount
		item.SnapshotSummary["blockedActions"] = summarizeBlockedActions(blockedDetails)
		items = append(items, item)
	}
	return items, nil
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
			"scanMode":                       detail.Plan.Metadata["scanMode"],
			"riskProfile":                    detail.Plan.Metadata["riskProfile"],
			"riskOverride":                   detail.Plan.Metadata["riskOverride"],
			"runtime":                        detail.Runtime,
			"doneCount":                      detail.Runtime.DoneCount,
			"skippedCount":                   detail.Runtime.SkippedCount,
			"failedCount":                    detail.Runtime.FailedCount,
			"pendingCount":                   detail.Runtime.PendingCount,
			"pendingTree":                    detail.Runtime.PendingTree,
			"retryableCount":                 detail.Runtime.RetryableCount,
			"blockedRetryCount":              detail.Runtime.BlockedRetryCount,
			"retryQueue":                     detail.Runtime.RetryQueue,
			"retrySummary":                   detail.Plan.Metadata["retrySummary"],
			"riskHitCount":                   detail.Runtime.RiskHitCount,
			"lastRiskStatus":                 detail.Runtime.LastRiskStatus,
			"currentRoot":                    detail.Runtime.CurrentRoot,
			"currentDirectory":               detail.Runtime.CurrentDirectory,
			"lastCompletedPath":              detail.Runtime.LastCompletedPath,
		},
		CreatedAt: createdAt,
	}, nil
}

func latestProviderStatusSnapshot(ctx context.Context, store *sqlitestore.Store, providerKey string) (ProviderStatus, bool, error) {
	row := store.DB().QueryRowContext(ctx, `
SELECT id, provider_key, summary_json, created_at
FROM provider_status_snapshots
WHERE provider_key = ?
ORDER BY created_at DESC, rowid DESC
LIMIT 1`, providerKey)
	var (
		item        ProviderStatus
		summaryJSON string
	)
	if err := row.Scan(&item.ID, &item.ProviderKey, &summaryJSON, &item.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return ProviderStatus{}, false, nil
		}
		return ProviderStatus{}, false, err
	}
	item.Summary = map[string]interface{}{}
	if summaryJSON != "" {
		if err := json.Unmarshal([]byte(summaryJSON), &item.Summary); err != nil {
			return ProviderStatus{}, false, err
		}
	}
	return item, true, nil
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
