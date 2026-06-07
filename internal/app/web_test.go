package app

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	webui "cloudpan-sync-go/web"
)

func TestHandleIndexServesHTML(t *testing.T) {
	indexHTML, err := webui.IndexHTML()
	if err != nil {
		t.Fatalf("IndexHTML() error = %v", err)
	}
	staticFS, err := webui.StaticFS()
	if err != nil {
		t.Fatalf("StaticFS() error = %v", err)
	}

	app := &App{
		logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
		webIndex:  indexHTML,
		webStatic: http.StripPrefix("/assets/", http.FileServer(http.FS(staticFS))),
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); !strings.Contains(got, "text/html") {
		t.Fatalf("expected html content type, got %q", got)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "CloudPan Sync Go Console") {
		t.Fatalf("expected console html body, got %q", body)
	}
	if !strings.Contains(body, `id="recent-results"`) {
		t.Fatalf("expected recent-results panel in html body, got %q", body)
	}
	if !strings.Contains(body, `id="recent-probes"`) {
		t.Fatalf("expected recent-probes panel in html body, got %q", body)
	}
	if !strings.Contains(body, `id="plan-execution-mode"`) {
		t.Fatalf("expected execution mode selector in html body, got %q", body)
	}
	if !strings.Contains(body, `id="plan-source-delete-policy"`) {
		t.Fatalf("expected source delete policy selector in html body, got %q", body)
	}
	if !strings.Contains(body, `id="plan-selected-roots"`) {
		t.Fatalf("expected selected roots input in html body, got %q", body)
	}
	if !strings.Contains(body, `id="profile-id"`) {
		t.Fatalf("expected profile edit hidden id in html body, got %q", body)
	}
	if !strings.Contains(body, `id="profile-risk-request-interval"`) {
		t.Fatalf("expected profile risk request interval input in html body, got %q", body)
	}
	if !strings.Contains(body, `id="profile-cancel-edit"`) {
		t.Fatalf("expected profile cancel edit button in html body, got %q", body)
	}
	if !strings.Contains(body, `id="plan-target-profile-insight"`) {
		t.Fatalf("expected target profile insight panel in html body, got %q", body)
	}
	if !strings.Contains(body, `id="plan-preview-meta"`) {
		t.Fatalf("expected preview meta panel in html body, got %q", body)
	}
	if !strings.Contains(body, `id="task-directory-states"`) {
		t.Fatalf("expected task directory states panel in html body, got %q", body)
	}
	if !strings.Contains(body, `id="task-pending-tree"`) {
		t.Fatalf("expected task pending tree panel in html body, got %q", body)
	}
	if !strings.Contains(body, `id="status-directory-states"`) {
		t.Fatalf("expected status directory states panel in html body, got %q", body)
	}
	if !strings.Contains(body, `id="status-pending-tree"`) {
		t.Fatalf("expected status pending tree panel in html body, got %q", body)
	}
	if !strings.Contains(body, `id="auto-recover-run"`) {
		t.Fatalf("expected auto recover controls in html body, got %q", body)
	}
	if !strings.Contains(body, `id="auto-recover-preview"`) {
		t.Fatalf("expected auto recover preview control in html body, got %q", body)
	}
	if !strings.Contains(body, `id="auto-recover-last-result-summary"`) {
		t.Fatalf("expected auto recover result summary in html body, got %q", body)
	}
	if !strings.Contains(body, `id="auto-recover-last-result-detail"`) {
		t.Fatalf("expected auto recover result detail in html body, got %q", body)
	}
	if !strings.Contains(body, `id="auto-recover-state"`) {
		t.Fatalf("expected auto recover state selector in html body, got %q", body)
	}
	if !strings.Contains(body, `value="waiting_auth_refresh"`) {
		t.Fatalf("expected waiting_auth_refresh option in html body, got %q", body)
	}
	if !strings.Contains(body, `value="waiting_local_restore"`) {
		t.Fatalf("expected waiting_local_restore option in html body, got %q", body)
	}
	if !strings.Contains(body, `value="waiting_manual_confirmation"`) {
		t.Fatalf("expected waiting_manual_confirmation option in html body, got %q", body)
	}
	if !strings.Contains(body, `value="waiting_retry_limit"`) {
		t.Fatalf("expected waiting_retry_limit option in html body, got %q", body)
	}
	if !strings.Contains(body, `value="provider_session_missing"`) {
		t.Fatalf("expected provider_session_missing option in html body, got %q", body)
	}
	if !strings.Contains(body, `id="risk-max-concurrent"`) {
		t.Fatalf("expected risk concurrency input in html body, got %q", body)
	}
	if !strings.Contains(body, `id="plan-recommendation-action"`) {
		t.Fatalf("expected recommendation action card in html body, got %q", body)
	}
	if !strings.Contains(body, `id="apply-recommended-execution"`) {
		t.Fatalf("expected apply recommended execution control in html body, got %q", body)
	}
	if !strings.Contains(body, `id="apply-recommended-risk"`) {
		t.Fatalf("expected apply recommended risk control in html body, got %q", body)
	}
	if !strings.Contains(body, `id="provider-capability-detail"`) {
		t.Fatalf("expected provider capability detail panel in html body, got %q", body)
	}
	if !strings.Contains(body, `id="plan-target-provider-insight"`) {
		t.Fatalf("expected target provider insight panel in html body, got %q", body)
	}
}

func TestRoutesServeStaticAssets(t *testing.T) {
	staticFS, err := webui.StaticFS()
	if err != nil {
		t.Fatalf("StaticFS() error = %v", err)
	}

	app := &App{
		logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
		webIndex:  []byte("<html><body>ok</body></html>"),
		webStatic: http.StripPrefix("/assets/", http.FileServer(http.FS(staticFS))),
	}

	req := httptest.NewRequest(http.MethodGet, "/assets/styles.css", nil)
	rec := httptest.NewRecorder()
	app.routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), ":root") {
		t.Fatalf("expected stylesheet content, got %q", rec.Body.String())
	}
}

func TestRoutesServeAppJSIncludesRetryEvidenceLabels(t *testing.T) {
	staticFS, err := webui.StaticFS()
	if err != nil {
		t.Fatalf("StaticFS() error = %v", err)
	}

	app := &App{
		logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
		webIndex:  []byte("<html><body>ok</body></html>"),
		webStatic: http.StripPrefix("/assets/", http.FileServer(http.FS(staticFS))),
	}

	req := httptest.NewRequest(http.MethodGet, "/assets/app.js", nil)
	rec := httptest.NewRecorder()
	app.routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "retrySelectedPaths") {
		t.Fatalf("expected retrySelectedPaths evidence in app.js, got %q", body)
	}
	if !strings.Contains(body, "renderUploadCheckpointResumeState") {
		t.Fatalf("expected upload checkpoint resumable-state renderer in app.js, got %q", body)
	}
	if !strings.Contains(body, "renderUploadCheckpointPartSummary") {
		t.Fatalf("expected upload checkpoint uploaded-parts summary renderer in app.js, got %q", body)
	}
	if !strings.Contains(body, "renderUploadCheckpointProviderDataSummary") {
		t.Fatalf("expected upload checkpoint provider-data summary renderer in app.js, got %q", body)
	}
	if !strings.Contains(body, "续传就绪") {
		t.Fatalf("expected upload checkpoint resumable label in app.js, got %q", body)
	}
	if !strings.Contains(body, "已传分片摘要") {
		t.Fatalf("expected upload checkpoint part summary label in app.js, got %q", body)
	}
	if !strings.Contains(body, "Provider 恢复线索") {
		t.Fatalf("expected upload checkpoint provider data label in app.js, got %q", body)
	}
	if !strings.Contains(body, "CALIBRATED") {
		t.Fatalf("expected CALIBRATED risk resolution detail in app.js, got %q", body)
	}
	if !strings.Contains(body, "recommendedExecutionMode") {
		t.Fatalf("expected recommendedExecutionMode wiring in app.js, got %q", body)
	}
	if !strings.Contains(body, "recommendedExecutionModeReason") {
		t.Fatalf("expected recommendedExecutionModeReason wiring in app.js, got %q", body)
	}
	if !strings.Contains(body, "recommendedRiskMode") {
		t.Fatalf("expected recommendedRiskMode wiring in app.js, got %q", body)
	}
	if !strings.Contains(body, "recommendedRiskModeReason") {
		t.Fatalf("expected recommendedRiskModeReason wiring in app.js, got %q", body)
	}
	if !strings.Contains(body, "风险档位") {
		t.Fatalf("expected risk mode label in app.js, got %q", body)
	}
	if !strings.Contains(body, "推荐风控") {
		t.Fatalf("expected recommended risk mode label in app.js, got %q", body)
	}
	if !strings.Contains(body, "推荐风控原因") {
		t.Fatalf("expected recommended risk mode reason label in app.js, got %q", body)
	}
	if !strings.Contains(body, "激进风险提示") {
		t.Fatalf("expected aggressive risk warning label in app.js, got %q", body)
	}
	if !strings.Contains(body, "风险节流") {
		t.Fatalf("expected risk throttle label in app.js, got %q", body)
	}
	if !strings.Contains(body, "风险模板解释") {
		t.Fatalf("expected risk resolution label in app.js, got %q", body)
	}
	if !strings.Contains(body, "parseProfileRiskDefaultsSourceFromExtra") {
		t.Fatalf("expected profile risk default source parser in app.js, got %q", body)
	}
	if !strings.Contains(body, "riskDefaultsSourceDisplay") {
		t.Fatalf("expected riskDefaultsSourceDisplay wiring in app.js, got %q", body)
	}
	if !strings.Contains(body, "Smoke Matrix") {
		t.Fatalf("expected Smoke Matrix source label in app.js, got %q", body)
	}
	if !strings.Contains(body, "renderProfileRiskDefaultSourceAdvice") {
		t.Fatalf("expected profile risk default source advice helper in app.js, got %q", body)
	}
	if !strings.Contains(body, "仍待补齐") {
		t.Fatalf("expected smoke-matrix pending advice text in app.js, got %q", body)
	}
	if !strings.Contains(body, "已按真实样本预填账号默认风控") {
		t.Fatalf("expected profile risk prefill flash text in app.js, got %q", body)
	}
	if !strings.Contains(body, "OVERRIDE FIELDS") {
		t.Fatalf("expected OVERRIDE FIELDS risk resolution detail in app.js, got %q", body)
	}
	if !strings.Contains(body, "PROVIDER HINTS") {
		t.Fatalf("expected PROVIDER HINTS risk resolution detail in app.js, got %q", body)
	}
	if !strings.Contains(body, "PROVIDER TRAITS") {
		t.Fatalf("expected PROVIDER TRAITS risk resolution detail in app.js, got %q", body)
	}
	if !strings.Contains(body, "PROFILE DEFAULT SOURCE KIND") {
		t.Fatalf("expected PROFILE DEFAULT SOURCE KIND risk resolution detail in app.js, got %q", body)
	}
	if !strings.Contains(body, "PROFILE DEFAULT BIAS") {
		t.Fatalf("expected PROFILE DEFAULT BIAS risk resolution detail in app.js, got %q", body)
	}
	if !strings.Contains(body, "来源类型 / 偏向") {
		t.Fatalf("expected kind/bias insight card label in app.js, got %q", body)
	}
	if !strings.Contains(body, "恢复预算理由") {
		t.Fatalf("expected recover budget reason insight card label in app.js, got %q", body)
	}
	if !strings.Contains(body, "renderAutoRetryWindowSource") {
		t.Fatalf("expected renderAutoRetryWindowSource helper in app.js, got %q", body)
	}
	if !strings.Contains(body, "window source") {
		t.Fatalf("expected auto retry window source text in app.js, got %q", body)
	}
	if !strings.Contains(body, "window advice") {
		t.Fatalf("expected auto retry window advice text in app.js, got %q", body)
	}
	if !strings.Contains(body, "profileDefaultKindBias") {
		t.Fatalf("expected snapshot summary profileDefaultKindBias field in app.js, got %q", body)
	}
	if !strings.Contains(body, "recoverBudgetReason") {
		t.Fatalf("expected snapshot summary recoverBudgetReason field in app.js, got %q", body)
	}
	if !strings.Contains(body, "renderBlockedSummary") {
		t.Fatalf("expected blocked summary helper in app.js, got %q", body)
	}
	if !strings.Contains(body, "阻塞摘要") {
		t.Fatalf("expected blocked summary label in app.js, got %q", body)
	}
	if !strings.Contains(body, "下一步摘要") {
		t.Fatalf("expected next-step summary label in app.js, got %q", body)
	}
	if !strings.Contains(body, "blockedSummary") {
		t.Fatalf("expected snapshot summary blockedSummary field in app.js, got %q", body)
	}
	if !strings.Contains(body, "next-step:") {
		t.Fatalf("expected blocked action next-step summary in app.js, got %q", body)
	}
	if !strings.Contains(body, "profile-kind") {
		t.Fatalf("expected profile-kind summary token in app.js, got %q", body)
	}
	if !strings.Contains(body, "profile-bias") {
		t.Fatalf("expected profile-bias summary token in app.js, got %q", body)
	}
	if !strings.Contains(body, "PROFILE DEFAULT SOURCE") {
		t.Fatalf("expected PROFILE DEFAULT SOURCE risk resolution detail in app.js, got %q", body)
	}
	if !strings.Contains(body, "PROFILE DEFAULT FIELDS") {
		t.Fatalf("expected PROFILE DEFAULT FIELDS risk resolution detail in app.js, got %q", body)
	}
	if !strings.Contains(body, "PROFILE DEFAULT ") {
		t.Fatalf("expected PROFILE DEFAULT risk resolution detail in app.js, got %q", body)
	}
	if !strings.Contains(body, "profile-default") {
		t.Fatalf("expected profile-default summary token in app.js, got %q", body)
	}
	if !strings.Contains(body, "renderSourceDeletePolicy") {
		t.Fatalf("expected source delete policy renderer in app.js, got %q", body)
	}
	if !strings.Contains(body, "mergeProfileRiskDefaultsIntoExtra") {
		t.Fatalf("expected profile risk defaults merge helper in app.js, got %q", body)
	}
	if !strings.Contains(body, "data-profile-edit") {
		t.Fatalf("expected profile edit action in app.js, got %q", body)
	}
	if !strings.Contains(body, "auth profile riskDefaults") {
		t.Fatalf("expected target profile risk defaults source text in app.js, got %q", body)
	}
	if !strings.Contains(body, "授权档案已更新") {
		t.Fatalf("expected profile updated flash text in app.js, got %q", body)
	}
	if !strings.Contains(body, "record_only（只记录，不删目标端）") {
		t.Fatalf("expected source delete policy label in app.js, got %q", body)
	}
	if !strings.Contains(body, "Selected Roots") {
		t.Fatalf("expected selected roots label in app.js, got %q", body)
	}
	if !strings.Contains(body, "Scan Trace") {
		t.Fatalf("expected scan trace label in app.js, got %q", body)
	}
	if !strings.Contains(body, "data-runtime-focus-path") {
		t.Fatalf("expected runtime focus path dataset in app.js, got %q", body)
	}
	if !strings.Contains(body, "data-runtime-focus-scope") {
		t.Fatalf("expected runtime focus scope dataset in app.js, got %q", body)
	}
	if !strings.Contains(body, "data-runtime-focus-kind") {
		t.Fatalf("expected runtime focus kind dataset in app.js, got %q", body)
	}
	if !strings.Contains(body, "sourceDeletionRecordPaths") {
		t.Fatalf("expected source deletion path helper in app.js, got %q", body)
	}
	if !strings.Contains(body, "wireSourceDeletionSummary") {
		t.Fatalf("expected source deletion summary wiring in app.js, got %q", body)
	}
	if !strings.Contains(body, "data-source-delete-prefill-path") {
		t.Fatalf("expected per-record source deletion prefill action in app.js, got %q", body)
	}
	if !strings.Contains(body, "data-source-delete-prefill-paths") {
		t.Fatalf("expected batch source deletion prefill action in app.js, got %q", body)
	}
	if !strings.Contains(body, "用此删除记录重建向导") {
		t.Fatalf("expected per-record source deletion rebuild label in app.js, got %q", body)
	}
	if !strings.Contains(body, "按全部删除记录重建向导") {
		t.Fatalf("expected batch source deletion rebuild label in app.js, got %q", body)
	}
	if !strings.Contains(body, "删除记录仅用于定位") {
		t.Fatalf("expected deletion-only preview warning in app.js, got %q", body)
	}
	if !strings.Contains(body, "wireSourceDeletionSummary(\"preview\", \"#plan-preview-meta\")") {
		t.Fatalf("expected preview source deletion wiring in app.js, got %q", body)
	}
	if !strings.Contains(body, "payloadHasOnlyDeletedEntries") {
		t.Fatalf("expected deleted-only submit guard in app.js, got %q", body)
	}
	if !strings.Contains(body, "renderRuntimePathChips") {
		t.Fatalf("expected runtime path chips renderer in app.js, got %q", body)
	}
	if !strings.Contains(body, "focusRuntimeTreeByPath") {
		t.Fatalf("expected runtime path focus helper in app.js, got %q", body)
	}
	if !strings.Contains(body, "已按扫描轨迹定位任务目录树") {
		t.Fatalf("expected task scan trace focus flash text in app.js, got %q", body)
	}
	if !strings.Contains(body, "已按选定根目录定位任务目录树") {
		t.Fatalf("expected task selected root focus flash text in app.js, got %q", body)
	}
	if !strings.Contains(body, "已按最近扫描轨迹定位状态目录树") {
		t.Fatalf("expected status scan trace focus flash text in app.js, got %q", body)
	}
	if !strings.Contains(body, "已按选定根目录定位状态目录树") {
		t.Fatalf("expected status selected root focus flash text in app.js, got %q", body)
	}
	if !strings.Contains(body, "sourceDeletePolicy") {
		t.Fatalf("expected sourceDeletePolicy field wiring in app.js, got %q", body)
	}
	if !strings.Contains(body, "waiting_auth_refresh") {
		t.Fatalf("expected waiting_auth_refresh recoverState affordance in app.js, got %q", body)
	}
	if !strings.Contains(body, "autoRecoverStateLabel") {
		t.Fatalf("expected autoRecoverStateLabel helper in app.js, got %q", body)
	}
	if !strings.Contains(body, "autoRecoverStateAdvice") {
		t.Fatalf("expected autoRecoverStateAdvice helper in app.js, got %q", body)
	}
	if !strings.Contains(body, "autoRecoverDecisionAdvice") {
		t.Fatalf("expected autoRecoverDecisionAdvice helper in app.js, got %q", body)
	}
	if !strings.Contains(body, "autoRecoverOutcomeLabel") {
		t.Fatalf("expected autoRecoverOutcomeLabel helper in app.js, got %q", body)
	}
	if !strings.Contains(body, "renderAutoRecoverOutcomeCounts") {
		t.Fatalf("expected renderAutoRecoverOutcomeCounts helper in app.js, got %q", body)
	}
	if !strings.Contains(body, "renderAutoRecoverRetryClassCounts") {
		t.Fatalf("expected renderAutoRecoverRetryClassCounts helper in app.js, got %q", body)
	}
	if !strings.Contains(body, "renderAutoRecoverRecoverStateCounts") {
		t.Fatalf("expected renderAutoRecoverRecoverStateCounts helper in app.js, got %q", body)
	}
	if !strings.Contains(body, "renderAutoRecoverBlockedActionCounts") {
		t.Fatalf("expected renderAutoRecoverBlockedActionCounts helper in app.js, got %q", body)
	}
	if !strings.Contains(body, "renderAutoRecoverProtocolGroupCounts") {
		t.Fatalf("expected renderAutoRecoverProtocolGroupCounts helper in app.js, got %q", body)
	}
	if !strings.Contains(body, "renderAutoRecoverProviderCounts") {
		t.Fatalf("expected renderAutoRecoverProviderCounts helper in app.js, got %q", body)
	}
	if !strings.Contains(body, "renderAutoRecoverProfileCounts") {
		t.Fatalf("expected renderAutoRecoverProfileCounts helper in app.js, got %q", body)
	}
	if !strings.Contains(body, "renderAutoRecoverLaneCounts") {
		t.Fatalf("expected renderAutoRecoverLaneCounts helper in app.js, got %q", body)
	}
	if !strings.Contains(body, "renderAutoRecoverSuggestedBudgets") {
		t.Fatalf("expected renderAutoRecoverSuggestedBudgets helper in app.js, got %q", body)
	}
	if !strings.Contains(body, "wireAutoRecoverLastResultDetail") {
		t.Fatalf("expected wireAutoRecoverLastResultDetail helper in app.js, got %q", body)
	}
	if !strings.Contains(body, "等待授权刷新") {
		t.Fatalf("expected waiting_auth_refresh label in app.js, got %q", body)
	}
	if !strings.Contains(body, "等待补回本地文件") {
		t.Fatalf("expected waiting_local_restore label in app.js, got %q", body)
	}
	if !strings.Contains(body, "等待人工确认") {
		t.Fatalf("expected waiting_manual_confirmation label in app.js, got %q", body)
	}
	if !strings.Contains(body, "等待重置重试策略") {
		t.Fatalf("expected waiting_retry_limit label in app.js, got %q", body)
	}
	if !strings.Contains(body, "等待态建议") {
		t.Fatalf("expected waiting state advice text in app.js, got %q", body)
	}
	if !strings.Contains(body, "blockedReason") {
		t.Fatalf("expected blockedReason detail text in app.js, got %q", body)
	}
	if !strings.Contains(body, "data-auto-recover-decision-preview") {
		t.Fatalf("expected auto recover decision preview action in app.js, got %q", body)
	}
	if !strings.Contains(body, "data-auto-recover-decision-run") {
		t.Fatalf("expected auto recover decision run action in app.js, got %q", body)
	}
	if !strings.Contains(body, "data-auto-recover-decision-open-task") {
		t.Fatalf("expected auto recover decision open task action in app.js, got %q", body)
	}
	if !strings.Contains(body, "data-auto-recover-decision-focus-state") {
		t.Fatalf("expected auto recover decision focus-state action in app.js, got %q", body)
	}
	if !strings.Contains(body, "data-auto-recover-decision-focus-lane-mode") {
		t.Fatalf("expected auto recover decision focus lane-mode action in app.js, got %q", body)
	}
	if !strings.Contains(body, "data-auto-recover-decision-apply-budgets") {
		t.Fatalf("expected auto recover decision apply budgets action in app.js, got %q", body)
	}
	if !strings.Contains(body, "data-auto-recover-decision-mode-budget") {
		t.Fatalf("expected auto recover decision mode budget wiring in app.js, got %q", body)
	}
	if !strings.Contains(body, "data-auto-recover-decision-strategy") {
		t.Fatalf("expected auto recover decision strategy wiring in app.js, got %q", body)
	}
	if !strings.Contains(body, "strategy: payload.strategy") {
		t.Fatalf("expected auto recover decision strategy filter sync in app.js, got %q", body)
	}
	if !strings.Contains(body, "data-auto-recover-decision-lane-budget") {
		t.Fatalf("expected auto recover decision lane budget wiring in app.js, got %q", body)
	}
	if !strings.Contains(body, "只看该状态") {
		t.Fatalf("expected auto recover decision focus-state label in app.js, got %q", body)
	}
	if !strings.Contains(body, "只看该 lane") {
		t.Fatalf("expected auto recover decision focus lane label in app.js, got %q", body)
	}
	if !strings.Contains(body, "采用建议预算") {
		t.Fatalf("expected auto recover decision apply budgets label in app.js, got %q", body)
	}
	if !strings.Contains(body, "renderAutoRecoverDecisionBudgetHints") {
		t.Fatalf("expected auto recover decision budget hint renderer in app.js, got %q", body)
	}
	if !strings.Contains(body, "预算占用：") {
		t.Fatalf("expected auto recover decision current budget hint text in app.js, got %q", body)
	}
	if !strings.Contains(body, "item.currentProviderBudget") {
		t.Fatalf("expected auto recover decision current provider budget wiring in app.js, got %q", body)
	}
	if !strings.Contains(body, "-> suggest") {
		t.Fatalf("expected auto recover decision current-to-suggest budget text in app.js, got %q", body)
	}
	if !strings.Contains(body, "预演该决策") {
		t.Fatalf("expected auto recover decision preview label in app.js, got %q", body)
	}
	if !strings.Contains(body, "执行该决策") {
		t.Fatalf("expected auto recover decision run label in app.js, got %q", body)
	}
	if !strings.Contains(body, "已按决策状态") {
		t.Fatalf("expected auto recover decision focus-state flash text in app.js, got %q", body)
	}
	if !strings.Contains(body, "已按决策 lane 收敛后台补传候选") {
		t.Fatalf("expected auto recover decision focus lane flash text in app.js, got %q", body)
	}
	if !strings.Contains(body, "已按决策采用建议预算") {
		t.Fatalf("expected auto recover decision apply budgets flash text in app.js, got %q", body)
	}
	if !strings.Contains(body, "已按决策预演后台补传") {
		t.Fatalf("expected auto recover decision preview flash text in app.js, got %q", body)
	}
	if !strings.Contains(body, "已按决策执行后台补传") {
		t.Fatalf("expected auto recover decision run flash text in app.js, got %q", body)
	}
	if !strings.Contains(body, "data-auto-recover-focus-lane-strategy") {
		t.Fatalf("expected auto recover summary lane strategy focus wiring in app.js, got %q", body)
	}
	if !strings.Contains(body, "data-auto-recover-preview-lane-strategy") {
		t.Fatalf("expected auto recover summary lane strategy preview wiring in app.js, got %q", body)
	}
	if !strings.Contains(body, "data-auto-recover-run-lane-strategy") {
		t.Fatalf("expected auto recover summary lane strategy run wiring in app.js, got %q", body)
	}
	if !strings.Contains(body, "const autoRetryPolicy = state.evidence?.autoRetryPolicy") {
		t.Fatalf("expected autoRetryPolicy guard in app.js, got %q", body)
	}
	if !strings.Contains(body, "data-auto-recover-preview-protocol-group") {
		t.Fatalf("expected auto recover preview protocol group wiring in app.js, got %q", body)
	}
	if !strings.Contains(body, "button.dataset.autoRecoverPreviewProtocolGroup || \"\"") {
		t.Fatalf("expected auto recover preview protocol group filter sync in app.js, got %q", body)
	}
	if !strings.Contains(body, "renderProviderSmokeMatrixControls") {
		t.Fatalf("expected provider smoke matrix controls renderer in app.js, got %q", body)
	}
	if !strings.Contains(body, "providerSmokeMatrixCounts") {
		t.Fatalf("expected provider smoke matrix counts helper in app.js, got %q", body)
	}
	if !strings.Contains(body, "data-provider-smoke-filter") {
		t.Fatalf("expected provider smoke matrix filter toolbar action in app.js, got %q", body)
	}
	if !strings.Contains(body, "已验收") {
		t.Fatalf("expected provider smoke matrix accepted label in app.js, got %q", body)
	}
	if !strings.Contains(body, "进行中") {
		t.Fatalf("expected provider smoke matrix in-progress label in app.js, got %q", body)
	}
	if !strings.Contains(body, "待补齐") {
		t.Fatalf("expected provider smoke matrix pending label in app.js, got %q", body)
	}
	if !strings.Contains(body, "data-provider-smoke-draft") {
		t.Fatalf("expected provider smoke matrix draft action in app.js, got %q", body)
	}
	if !strings.Contains(body, "data-provider-smoke-draft-action") {
		t.Fatalf("expected provider smoke matrix draft-action in app.js, got %q", body)
	}
	if !strings.Contains(body, "data-provider-smoke-prefill-profile-risk") {
		t.Fatalf("expected provider smoke matrix prefill-profile-risk action in app.js, got %q", body)
	}
	if !strings.Contains(body, "data-provider-smoke-focus-group") {
		t.Fatalf("expected provider smoke matrix focus-group action in app.js, got %q", body)
	}
	if !strings.Contains(body, "data-provider-smoke-filter-status") {
		t.Fatalf("expected provider smoke matrix status filter action in app.js, got %q", body)
	}
	if !strings.Contains(body, "data-provider-smoke-open-record") {
		t.Fatalf("expected provider smoke matrix open-record action in app.js, got %q", body)
	}
	if !strings.Contains(body, "data-provider-smoke-open-task") {
		t.Fatalf("expected provider smoke matrix open-task action in app.js, got %q", body)
	}
	if !strings.Contains(body, "异常样本：auth") {
		t.Fatalf("expected provider smoke matrix anomaly sample summary in app.js, got %q", body)
	}
	if !strings.Contains(body, "checklist:") {
		t.Fatalf("expected provider smoke checklist summary in app.js, got %q", body)
	}
	if !strings.Contains(body, "renderProviderSmokeChecklist") {
		t.Fatalf("expected provider smoke checklist helper in app.js, got %q", body)
	}
	if !strings.Contains(body, "renderProviderSmokeReadiness") {
		t.Fatalf("expected provider smoke readiness helper in app.js, got %q", body)
	}
	if !strings.Contains(body, "renderProviderSmokeGaps") {
		t.Fatalf("expected provider smoke gap helper in app.js, got %q", body)
	}
	if !strings.Contains(body, "renderUploadCheckpointReadiness") {
		t.Fatalf("expected upload checkpoint readiness helper in app.js, got %q", body)
	}
	if !strings.Contains(body, "Checkpoint Ready") {
		t.Fatalf("expected upload checkpoint readiness metric in app.js, got %q", body)
	}
	if !strings.Contains(body, "renderAutoRecoverPriorityAction") {
		t.Fatalf("expected auto recover priority-action helper in app.js, got %q", body)
	}
	if !strings.Contains(body, "Recover Priority") {
		t.Fatalf("expected auto recover priority metric in app.js, got %q", body)
	}
	if !strings.Contains(body, "renderAutoRecoverReadiness") {
		t.Fatalf("expected auto recover readiness helper in app.js, got %q", body)
	}
	if !strings.Contains(body, "renderAutoRecoverFairnessReadiness") {
		t.Fatalf("expected auto recover fairness readiness helper in app.js, got %q", body)
	}
	if !strings.Contains(body, "renderAutoRecoverFairnessPriorityAction") {
		t.Fatalf("expected auto recover fairness priority helper in app.js, got %q", body)
	}
	if !strings.Contains(body, "Recover Ready") {
		t.Fatalf("expected auto recover readiness metric in app.js, got %q", body)
	}
	if !strings.Contains(body, "Fairness Ready") {
		t.Fatalf("expected auto recover fairness readiness metric in app.js, got %q", body)
	}
	if !strings.Contains(body, "Fairness Priority") {
		t.Fatalf("expected auto recover fairness priority metric in app.js, got %q", body)
	}
	if !strings.Contains(body, "readiness:") {
		t.Fatalf("expected provider smoke readiness summary in app.js, got %q", body)
	}
	if !strings.Contains(body, "gaps:") {
		t.Fatalf("expected provider smoke gap summary in app.js, got %q", body)
	}
	if !strings.Contains(body, "renderProviderSmokeNextAction") {
		t.Fatalf("expected provider smoke next-action helper in app.js, got %q", body)
	}
	if !strings.Contains(body, "next action:") {
		t.Fatalf("expected provider smoke next-action summary in app.js, got %q", body)
	}
	if !strings.Contains(body, "renderProviderSmokePriorityAction") {
		t.Fatalf("expected provider smoke priority-action helper in app.js, got %q", body)
	}
	if !strings.Contains(body, "priority action:") {
		t.Fatalf("expected provider smoke priority-action summary in app.js, got %q", body)
	}
	if !strings.Contains(body, "priority calibration:") {
		t.Fatalf("expected provider default risk priority-calibration summary in app.js, got %q", body)
	}
	if !strings.Contains(body, "calibration readiness") {
		t.Fatalf("expected provider default risk calibration-readiness summary in app.js, got %q", body)
	}
	if !strings.Contains(body, "anomaly missing") {
		t.Fatalf("expected provider smoke matrix anomaly missing detail in app.js, got %q", body)
	}
	if !strings.Contains(body, "anomaly actions") {
		t.Fatalf("expected provider smoke matrix anomaly actions detail in app.js, got %q", body)
	}
	if !strings.Contains(body, "anomaly advice") {
		t.Fatalf("expected provider smoke matrix anomaly advice detail in app.js, got %q", body)
	}
	if !strings.Contains(body, "acceptanceActions") {
		t.Fatalf("expected provider smoke matrix acceptance actions wiring in app.js, got %q", body)
	}
	if !strings.Contains(body, "draftProviderSmokeFromMatrix") {
		t.Fatalf("expected provider smoke matrix draft helper in app.js, got %q", body)
	}
	if !strings.Contains(body, "providerSmokeDraftActionLabel") {
		t.Fatalf("expected provider smoke draft action label helper in app.js, got %q", body)
	}
	if !strings.Contains(body, "draftProviderSmokeActionFromMatrix") {
		t.Fatalf("expected provider smoke draft action helper in app.js, got %q", body)
	}
	if !strings.Contains(body, "providerSmokeDraftSpecFromRepresentative") {
		t.Fatalf("expected representative smoke draft helper in app.js, got %q", body)
	}
	if !strings.Contains(body, "补大文件样本") {
		t.Fatalf("expected representative large-file draft label in app.js, got %q", body)
	}
	if !strings.Contains(body, "补多层目录样本") {
		t.Fatalf("expected representative nested-directory draft label in app.js, got %q", body)
	}
	if !strings.Contains(body, "补重试恢复样本") {
		t.Fatalf("expected representative retry-recovery draft label in app.js, got %q", body)
	}
	if !strings.Contains(body, "profileRiskDefaultsFromSmokeMatrix") {
		t.Fatalf("expected provider smoke to profile risk helper in app.js, got %q", body)
	}
	if !strings.Contains(body, "prefillProfileRiskDefaultsFromMatrix") {
		t.Fatalf("expected provider smoke profile risk prefill helper in app.js, got %q", body)
	}
	if !strings.Contains(body, "draftProviderSmokeFromGap") {
		t.Fatalf("expected provider smoke gap draft helper in app.js, got %q", body)
	}
	if !strings.Contains(body, "openProviderSmokeRecordInMatrix") {
		t.Fatalf("expected provider smoke open-record helper in app.js, got %q", body)
	}
	if !strings.Contains(body, "focusProviderSmokeMatrixByStatus") {
		t.Fatalf("expected provider smoke matrix status focus helper in app.js, got %q", body)
	}
	if !strings.Contains(body, "focusProviderSmokeRecordsByGroup") {
		t.Fatalf("expected provider smoke matrix focus helper in app.js, got %q", body)
	}
	if !strings.Contains(body, "setProviderSmokeMatrixFilter") {
		t.Fatalf("expected provider smoke matrix filter helper in app.js, got %q", body)
	}
	if !strings.Contains(body, "已按验收矩阵预填 smoke 表单") {
		t.Fatalf("expected provider smoke matrix draft flash text in app.js, got %q", body)
	}
	if !strings.Contains(body, "已按验收缺口预填 smoke 动作") {
		t.Fatalf("expected provider smoke matrix gap draft flash text in app.js, got %q", body)
	}
	if !strings.Contains(body, "已按真实样本预填账号默认风控") {
		t.Fatalf("expected provider smoke matrix profile risk prefill flash text in app.js, got %q", body)
	}
	if !strings.Contains(body, "已打开 smoke 样本") {
		t.Fatalf("expected provider smoke matrix open-record flash text in app.js, got %q", body)
	}
	if !strings.Contains(body, "已打开 smoke 样本并回填表单") {
		t.Fatalf("expected provider smoke matrix open-record hydrate flash text in app.js, got %q", body)
	}
	if !strings.Contains(body, "已打开 blocked 摘要对应的样本任务") {
		t.Fatalf("expected provider smoke matrix open-task flash text in app.js, got %q", body)
	}
	if !strings.Contains(body, "data-report-view") {
		t.Fatalf("expected report history view action in app.js, got %q", body)
	}
	if !strings.Contains(body, "data-report-download") {
		t.Fatalf("expected report history download action in app.js, got %q", body)
	}
	if !strings.Contains(body, "selectedReportId") {
		t.Fatalf("expected selectedReportId state in app.js, got %q", body)
	}
	if !strings.Contains(body, "renderReportHistory") {
		t.Fatalf("expected report history renderer in app.js, got %q", body)
	}
	if !strings.Contains(body, "selectedEvidenceReport") {
		t.Fatalf("expected selectedEvidenceReport helper in app.js, got %q", body)
	}
	if !strings.Contains(body, "renderEvidenceUploadCheckpointSummary") {
		t.Fatalf("expected evidence upload checkpoint summary renderer in app.js, got %q", body)
	}
	if !strings.Contains(body, "renderEvidenceAutoRecoverSummary") {
		t.Fatalf("expected evidence auto recover summary renderer in app.js, got %q", body)
	}
	if !strings.Contains(body, "自动补传验收") {
		t.Fatalf("expected evidence auto recover report title in app.js, got %q", body)
	}
	if !strings.Contains(body, "autoRecoverPool") {
		t.Fatalf("expected evidence auto recover pool wiring in app.js, got %q", body)
	}
	if !strings.Contains(body, "fairness priority action") {
		t.Fatalf("expected evidence auto recover fairness priority action in app.js, got %q", body)
	}
	if !strings.Contains(body, "Upload checkpoint 默认恢复验收") {
		t.Fatalf("expected evidence upload checkpoint report title in app.js, got %q", body)
	}
	if !strings.Contains(body, "Checkpoint Resume Ready") {
		t.Fatalf("expected evidence upload checkpoint readiness metric in app.js, got %q", body)
	}
	if !strings.Contains(body, "uploadCheckpointResume") {
		t.Fatalf("expected evidence upload checkpoint resume wiring in app.js, got %q", body)
	}
	if !strings.Contains(body, "recover priority action") {
		t.Fatalf("expected evidence upload checkpoint priority action summary in app.js, got %q", body)
	}
	if !strings.Contains(body, "renderEvidenceProviderSmokeProviders") {
		t.Fatalf("expected evidence provider smoke provider renderer in app.js, got %q", body)
	}
	if !strings.Contains(body, "providerSmokeProviderCounts") {
		t.Fatalf("expected evidence provider smoke provider counts helper in app.js, got %q", body)
	}
	if !strings.Contains(body, "Provider 级真实样本验收") {
		t.Fatalf("expected provider-level smoke acceptance title in app.js, got %q", body)
	}
	if !strings.Contains(body, "Provider Ready") {
		t.Fatalf("expected provider-level smoke readiness metric in app.js, got %q", body)
	}
	if !strings.Contains(body, "providerSmokeProviders") {
		t.Fatalf("expected providerSmokeProviders report wiring in app.js, got %q", body)
	}
	if !strings.Contains(body, "provider priority action") {
		t.Fatalf("expected provider-level smoke priority action summary in app.js, got %q", body)
	}
	if !strings.Contains(body, "provider preferred sample:") || !strings.Contains(body, "provider preferred upload:") || !strings.Contains(body, "provider preferred anomaly:") {
		t.Fatalf("expected provider-level preferred sample summaries in app.js, got %q", body)
	}
	if !strings.Contains(body, "renderEvidenceRiskCalibrationSummary") {
		t.Fatalf("expected evidence risk calibration summary renderer in app.js, got %q", body)
	}
	if !strings.Contains(body, "Provider 默认风控校准") {
		t.Fatalf("expected evidence provider risk calibration title in app.js, got %q", body)
	}
	if !strings.Contains(body, "Calibration Ready") {
		t.Fatalf("expected evidence provider risk calibration readiness metric in app.js, got %q", body)
	}
	if !strings.Contains(body, "defaultRiskTemplate") {
		t.Fatalf("expected evidence provider risk template wiring in app.js, got %q", body)
	}
	if !strings.Contains(body, "priority calibration") {
		t.Fatalf("expected evidence provider risk priority calibration summary in app.js, got %q", body)
	}
	if !strings.Contains(body, "calibration missing") || !strings.Contains(body, "calibration coverage") {
		t.Fatalf("expected evidence provider risk missing/coverage summary in app.js, got %q", body)
	}
	if !strings.Contains(body, "window source") {
		t.Fatalf("expected evidence provider risk auto retry window source in app.js, got %q", body)
	}
	if !strings.Contains(body, "item.id === state.selectedReportId ? \"active\" : \"\"") {
		t.Fatalf("expected report history selected-row state in app.js, got %q", body)
	}
	if !strings.Contains(body, "cloudpan-sync-report") {
		t.Fatalf("expected report download fallback filename in app.js, got %q", body)
	}
	if !strings.Contains(body, "replace(/\\s+/g, \"-\")") {
		t.Fatalf("expected report download filename sanitizer in app.js, got %q", body)
	}
	if !strings.Contains(body, "已切换验收报告") {
		t.Fatalf("expected report history switch flash text in app.js, got %q", body)
	}
	if !strings.Contains(body, "验收报告已保存") {
		t.Fatalf("expected report save flash text in app.js, got %q", body)
	}
	if !strings.Contains(body, "验收报告已下载") {
		t.Fatalf("expected report download flash text in app.js, got %q", body)
	}
	if !strings.Contains(body, "验收报告已刷新") {
		t.Fatalf("expected report refresh flash text in app.js, got %q", body)
	}
	if !strings.Contains(body, "Provider smoke 记录已保存") {
		t.Fatalf("expected provider smoke save flash text in app.js, got %q", body)
	}
	if !strings.Contains(body, "状态矩阵已刷新") {
		t.Fatalf("expected status refresh flash text in app.js, got %q", body)
	}
	if !strings.Contains(body, "renderTaskResolutionGuide") {
		t.Fatalf("expected task resolution guide renderer in app.js, got %q", body)
	}
	if !strings.Contains(body, "wireTaskResolutionGuide") {
		t.Fatalf("expected task resolution guide wiring in app.js, got %q", body)
	}
	if !strings.Contains(body, "data-task-guide-view") {
		t.Fatalf("expected task resolution guide view action in app.js, got %q", body)
	}
	if !strings.Contains(body, "data-task-guide-intent") {
		t.Fatalf("expected task resolution guide intent action in app.js, got %q", body)
	}
	if !strings.Contains(body, "refresh_auth_profile") {
		t.Fatalf("expected refresh_auth_profile guide action in app.js, got %q", body)
	}
	if !strings.Contains(body, "restore_local_source_file") {
		t.Fatalf("expected restore_local_source_file guide action in app.js, got %q", body)
	}
	if !strings.Contains(body, "manual_confirmation_required") {
		t.Fatalf("expected manual_confirmation_required guide action in app.js, got %q", body)
	}
	if !strings.Contains(body, "manual_intervention_required") {
		t.Fatalf("expected manual_intervention_required guide action in app.js, got %q", body)
	}
	if !strings.Contains(body, "provider_session_missing") {
		t.Fatalf("expected provider_session_missing helper in app.js, got %q", body)
	}
	if !strings.Contains(body, "review_and_reset_retry_strategy") {
		t.Fatalf("expected review_and_reset_retry_strategy guide action in app.js, got %q", body)
	}
	if !strings.Contains(body, "打开授权面板") {
		t.Fatalf("expected open providers guide label in app.js, got %q", body)
	}
	if !strings.Contains(body, "按当前阻塞打开状态矩阵") {
		t.Fatalf("expected focus status blocked guide label in app.js, got %q", body)
	}
	if !strings.Contains(body, "focus_status_blocked") {
		t.Fatalf("expected focus_status_blocked guide intent in app.js, got %q", body)
	}
	if !strings.Contains(body, "focus_status_auto_recover_mode") {
		t.Fatalf("expected focus_status_auto_recover_mode guide intent in app.js, got %q", body)
	}
	if !strings.Contains(body, "upload_checkpoint_auto_resume") {
		t.Fatalf("expected upload_checkpoint_auto_resume guide content in app.js, got %q", body)
	}
	if !strings.Contains(body, "只看自动续跑候选") {
		t.Fatalf("expected upload checkpoint guide button label in app.js, got %q", body)
	}
	if !strings.Contains(body, "只看自动补传候选") {
		t.Fatalf("expected auto retry guide button label in app.js, got %q", body)
	}
	if !strings.Contains(body, "等待后台自动补传接管") {
		t.Fatalf("expected auto retry guide title in app.js, got %q", body)
	}
	if !strings.Contains(body, "renderBlockedActionsSummary") {
		t.Fatalf("expected blocked actions summary renderer in app.js, got %q", body)
	}
	if !strings.Contains(body, "data-blocked-focus-action") {
		t.Fatalf("expected blocked action focus dataset in app.js, got %q", body)
	}
	if !strings.Contains(body, "focusBlockedActionSummary") {
		t.Fatalf("expected blocked action focus helper in app.js, got %q", body)
	}
	if !strings.Contains(body, "已按 blocked action 收敛最近重试队列") {
		t.Fatalf("expected blocked action summary flash text in app.js, got %q", body)
	}
	if !strings.Contains(body, "data-provider-smoke-view") {
		t.Fatalf("expected provider smoke record view action in app.js, got %q", body)
	}
	if !strings.Contains(body, "data-provider-smoke-download") {
		t.Fatalf("expected provider smoke record download action in app.js, got %q", body)
	}
	if !strings.Contains(body, "selectedProviderSmokeId") {
		t.Fatalf("expected selectedProviderSmokeId state in app.js, got %q", body)
	}
	if !strings.Contains(body, "selectedProviderSmokeMarkdown") {
		t.Fatalf("expected selectedProviderSmokeMarkdown state in app.js, got %q", body)
	}
	if !strings.Contains(body, "renderProviderSmokeMarkdown") {
		t.Fatalf("expected provider smoke markdown renderer in app.js, got %q", body)
	}
	if !strings.Contains(body, "loadProviderSmokeMarkdown") {
		t.Fatalf("expected provider smoke markdown loader in app.js, got %q", body)
	}
	if !strings.Contains(body, "?format=markdown") {
		t.Fatalf("expected provider smoke markdown format query in app.js, got %q", body)
	}
	if !strings.Contains(body, "\"Accept\": \"text/plain\"") {
		t.Fatalf("expected provider smoke markdown Accept header in app.js, got %q", body)
	}
	if !strings.Contains(body, "item.id === state.selectedProviderSmokeId ? \"active\" : \"\"") {
		t.Fatalf("expected provider smoke selected-row state in app.js, got %q", body)
	}
	if !strings.Contains(body, "provider-smoke-markdown") {
		t.Fatalf("expected provider smoke markdown panel in app.js, got %q", body)
	}
	if !strings.Contains(body, "provider-smoke-records-filter-clear") {
		t.Fatalf("expected provider smoke filter clear action in app.js, got %q", body)
	}
	if !strings.Contains(body, "template:") {
		t.Fatalf("expected provider smoke template summary in app.js, got %q", body)
	}
	if !strings.Contains(body, "completeness") {
		t.Fatalf("expected provider smoke evidence completeness summary in app.js, got %q", body)
	}
	if !strings.Contains(body, "reuse:") {
		t.Fatalf("expected provider smoke reuse advice summary in app.js, got %q", body)
	}
	if !strings.Contains(body, "priority") {
		t.Fatalf("expected provider smoke reuse priority summary in app.js, got %q", body)
	}
	if !strings.Contains(body, "item.sampleType") || !strings.Contains(body, "item.evidenceCompleteness") || !strings.Contains(body, "item.reuseAdvice") || !strings.Contains(body, "item.autoRecoverFocus") || !strings.Contains(body, "item.regressionEntry") {
		t.Fatalf("expected provider smoke filter to include structured fields in app.js, got %q", body)
	}
	if !strings.Contains(body, "regression entry:") {
		t.Fatalf("expected provider smoke regression entry summary in app.js, got %q", body)
	}
	if !strings.Contains(body, "preferred sample:") {
		t.Fatalf("expected provider smoke preferred sample summary in app.js, got %q", body)
	}
	if !strings.Contains(body, "preferred upload:") || !strings.Contains(body, "preferred anomaly:") || !strings.Contains(body, "preferred representative:") {
		t.Fatalf("expected provider smoke preferred upload/anomaly/representative summaries in app.js, got %q", body)
	}
	if !strings.Contains(body, "auto recover focus:") {
		t.Fatalf("expected provider smoke auto recover focus summary in app.js, got %q", body)
	}
	if !strings.Contains(body, "已切换 smoke Markdown") {
		t.Fatalf("expected provider smoke markdown switch flash text in app.js, got %q", body)
	}
	if !strings.Contains(body, "smoke Markdown 已下载") {
		t.Fatalf("expected provider smoke markdown download flash text in app.js, got %q", body)
	}
	if !strings.Contains(body, "已清空 smoke 记录筛选") {
		t.Fatalf("expected provider smoke filter clear flash text in app.js, got %q", body)
	}
	if !strings.Contains(body, "data-tree-group-toggle") {
		t.Fatalf("expected tree group toggle dataset in app.js, got %q", body)
	}
	if !strings.Contains(body, "data-tree-focus-path") {
		t.Fatalf("expected tree focus path dataset in app.js, got %q", body)
	}
	if !strings.Contains(body, "data-tree-sync-path") {
		t.Fatalf("expected tree sync path dataset in app.js, got %q", body)
	}
	if !strings.Contains(body, "data-tree-prefill-path") {
		t.Fatalf("expected tree prefill path dataset in app.js, got %q", body)
	}
	if !strings.Contains(body, "data-tree-retry-path") {
		t.Fatalf("expected tree retry path dataset in app.js, got %q", body)
	}
	if !strings.Contains(body, "data-tree-copy-path") {
		t.Fatalf("expected tree copy path dataset in app.js, got %q", body)
	}
	if !strings.Contains(body, "data-tree-parent-path") {
		t.Fatalf("expected tree parent path dataset in app.js, got %q", body)
	}
	if !strings.Contains(body, "同步另一棵树") {
		t.Fatalf("expected tree sync label in app.js, got %q", body)
	}
	if !strings.Contains(body, "只看当前路径") {
		t.Fatalf("expected tree focus label in app.js, got %q", body)
	}
	if !strings.Contains(body, "wireTreeGroupToggles") {
		t.Fatalf("expected tree node action wiring in app.js, got %q", body)
	}
	if !strings.Contains(body, "task-directory-prefill-visible") {
		t.Fatalf("expected task directory visible prefill action in app.js, got %q", body)
	}
	if !strings.Contains(body, "task-retry-visible-directory") {
		t.Fatalf("expected task directory visible retry action in app.js, got %q", body)
	}
	if !strings.Contains(body, "task-directory-copy-visible") {
		t.Fatalf("expected task directory visible copy action in app.js, got %q", body)
	}
	if !strings.Contains(body, "task-pending-prefill-visible") {
		t.Fatalf("expected task pending visible prefill action in app.js, got %q", body)
	}
	if !strings.Contains(body, "task-retry-visible-pending") {
		t.Fatalf("expected task pending visible retry action in app.js, got %q", body)
	}
	if !strings.Contains(body, "task-pending-copy-visible") {
		t.Fatalf("expected task pending visible copy action in app.js, got %q", body)
	}
	if !strings.Contains(body, "status-directory-prefill-visible") {
		t.Fatalf("expected status directory visible prefill action in app.js, got %q", body)
	}
	if !strings.Contains(body, "status-retry-visible-directory") {
		t.Fatalf("expected status directory visible retry action in app.js, got %q", body)
	}
	if !strings.Contains(body, "status-directory-copy-visible") {
		t.Fatalf("expected status directory visible copy action in app.js, got %q", body)
	}
	if !strings.Contains(body, "status-pending-prefill-visible") {
		t.Fatalf("expected status pending visible prefill action in app.js, got %q", body)
	}
	if !strings.Contains(body, "status-retry-visible-pending") {
		t.Fatalf("expected status pending visible retry action in app.js, got %q", body)
	}
	if !strings.Contains(body, "status-pending-copy-visible") {
		t.Fatalf("expected status pending visible copy action in app.js, got %q", body)
	}
	if !strings.Contains(body, "task-directory-filter-clear") {
		t.Fatalf("expected task directory filter clear action in app.js, got %q", body)
	}
	if !strings.Contains(body, "task-pending-filter-clear") {
		t.Fatalf("expected task pending filter clear action in app.js, got %q", body)
	}
	if !strings.Contains(body, "status-directory-filter-clear") {
		t.Fatalf("expected status directory filter clear action in app.js, got %q", body)
	}
	if !strings.Contains(body, "status-pending-filter-clear") {
		t.Fatalf("expected status pending filter clear action in app.js, got %q", body)
	}
	if !strings.Contains(body, "已按当前任务重建向导参数") {
		t.Fatalf("expected visible selection wizard flash text in app.js, got %q", body)
	}
	if !strings.Contains(body, "已复制") {
		t.Fatalf("expected visible path copy flash text in app.js, got %q", body)
	}
	if !strings.Contains(body, "data-auto-recover-decision-focus-lane-strategy") {
		t.Fatalf("expected auto recover decision lane strategy focus wiring in app.js, got %q", body)
	}
	if !strings.Contains(body, "const filters = state.autoRecoverFilters || {}") {
		t.Fatalf("expected auto recover request to read filter state first in app.js, got %q", body)
	}
	if !strings.Contains(body, "strategy: button.dataset.autoRecoverPreviewLaneStrategy || \"\"") {
		t.Fatalf("expected auto recover preview lane strategy filter sync in app.js, got %q", body)
	}
	if !strings.Contains(body, "strategy: button.dataset.autoRecoverRunLaneStrategy || \"\"") {
		t.Fatalf("expected auto recover run lane strategy filter sync in app.js, got %q", body)
	}
	if !strings.Contains(body, "const strategy = button.dataset.autoRecoverFocusLaneStrategy || \"\"") {
		t.Fatalf("expected auto recover focus lane strategy filter sync in app.js, got %q", body)
	}
	if !strings.Contains(body, "const strategy = button.dataset.autoRecoverDecisionFocusLaneStrategy || \"\"") {
		t.Fatalf("expected auto recover decision focus lane strategy filter sync in app.js, got %q", body)
	}
	if !strings.Contains(body, "data-auto-recover-apply-mode-budget") {
		t.Fatalf("expected auto recover summary mode budget wiring in app.js, got %q", body)
	}
	if !strings.Contains(body, "data-auto-recover-preview-mode-budget") {
		t.Fatalf("expected auto recover summary preview mode budget wiring in app.js, got %q", body)
	}
	if !strings.Contains(body, "data-auto-recover-run-state") {
		t.Fatalf("expected auto recover run-state dataset in app.js, got %q", body)
	}
	if !strings.Contains(body, "data-auto-recover-run-retry-class") {
		t.Fatalf("expected auto recover run retry-class dataset in app.js, got %q", body)
	}
	if !strings.Contains(body, "data-auto-recover-run-primary-blocked-action") {
		t.Fatalf("expected auto recover run primary blocked action dataset in app.js, got %q", body)
	}
	if !strings.Contains(body, "data-auto-recover-run-blocked-action") {
		t.Fatalf("expected auto recover run blocked action dataset in app.js, got %q", body)
	}
	if !strings.Contains(body, "data-auto-recover-run-lane-mode") {
		t.Fatalf("expected auto recover run lane-mode dataset in app.js, got %q", body)
	}
	if !strings.Contains(body, "data-auto-recover-run-mode") {
		t.Fatalf("expected auto recover run mode dataset in app.js, got %q", body)
	}
	if !strings.Contains(body, "data-auto-recover-run-protocol-group") {
		t.Fatalf("expected auto recover run protocol-group dataset in app.js, got %q", body)
	}
	if !strings.Contains(body, "data-auto-recover-focus-mode") {
		t.Fatalf("expected auto recover focus mode dataset in app.js, got %q", body)
	}
	if !strings.Contains(body, "data-auto-recover-focus-lane-mode") {
		t.Fatalf("expected auto recover focus lane-mode dataset in app.js, got %q", body)
	}
	if !strings.Contains(body, "data-auto-recover-apply-budgets") {
		t.Fatalf("expected auto recover apply budgets dataset in app.js, got %q", body)
	}
	if !strings.Contains(body, "data-auto-recover-preview-lane-mode") {
		t.Fatalf("expected auto recover preview lane-mode dataset in app.js, got %q", body)
	}
	if !strings.Contains(body, "data-auto-recover-open-task") {
		t.Fatalf("expected auto recover open-task dataset in app.js, got %q", body)
	}
	if !strings.Contains(body, "执行该模式") {
		t.Fatalf("expected auto recover run mode label in app.js, got %q", body)
	}
	if !strings.Contains(body, "执行该协议族") {
		t.Fatalf("expected auto recover run protocol group label in app.js, got %q", body)
	}
	if !strings.Contains(body, "预演该协议族") {
		t.Fatalf("expected auto recover preview protocol group label in app.js, got %q", body)
	}
	if !strings.Contains(body, "预演样本协议族") {
		t.Fatalf("expected auto recover preview sample protocol group label in app.js, got %q", body)
	}
	if !strings.Contains(body, "执行样本协议族") {
		t.Fatalf("expected auto recover run sample protocol group label in app.js, got %q", body)
	}
	if !strings.Contains(body, "采用建议预算") {
		t.Fatalf("expected auto recover apply suggested budgets label in app.js, got %q", body)
	}
	if !strings.Contains(body, "预演该 lane") {
		t.Fatalf("expected auto recover preview lane label in app.js, got %q", body)
	}
	if !strings.Contains(body, "执行该阻塞动作") {
		t.Fatalf("expected auto recover run blocked action label in app.js, got %q", body)
	}
	if !strings.Contains(body, "只执行可执行态") {
		t.Fatalf("expected auto recover runnable-now label in app.js, got %q", body)
	}
	if !strings.Contains(body, "只执行等时间窗") {
		t.Fatalf("expected auto recover retry-window label in app.js, got %q", body)
	}
	if !strings.Contains(body, "只执行等补源文件") {
		t.Fatalf("expected auto recover local-restore label in app.js, got %q", body)
	}
	if !strings.Contains(body, "只执行等人工确认") {
		t.Fatalf("expected auto recover manual-confirmation label in app.js, got %q", body)
	}
	if !strings.Contains(body, "只执行重试耗尽") {
		t.Fatalf("expected auto recover retry-limit label in app.js, got %q", body)
	}
	if !strings.Contains(body, "只执行其它等待") {
		t.Fatalf("expected auto recover other-waiting label in app.js, got %q", body)
	}
	if !strings.Contains(body, "执行主重试类型") {
		t.Fatalf("expected auto recover primary retry-class label in app.js, got %q", body)
	}
	if !strings.Contains(body, "执行主阻塞动作") {
		t.Fatalf("expected auto recover primary blocked action label in app.js, got %q", body)
	}
	if !strings.Contains(body, "执行该 lane") {
		t.Fatalf("expected auto recover run lane label in app.js, got %q", body)
	}
	if !strings.Contains(body, "打开样本任务") {
		t.Fatalf("expected auto recover open sample task label in app.js, got %q", body)
	}
	if !strings.Contains(body, "currentAutoRecoverScopedRequest") {
		t.Fatalf("expected scoped auto recover request helper in app.js, got %q", body)
	}
	if !strings.Contains(body, "data-provider-smoke-draft") {
		t.Fatalf("expected provider smoke draft action in app.js, got %q", body)
	}
	if !strings.Contains(body, "data-provider-smoke-focus-group") {
		t.Fatalf("expected provider smoke focus group action in app.js, got %q", body)
	}
	if !strings.Contains(body, "data-provider-smoke-filter-status") {
		t.Fatalf("expected provider smoke status filter action in app.js, got %q", body)
	}
	if !strings.Contains(body, "focusProviderSmokeRecordsByGroup") {
		t.Fatalf("expected provider smoke focus helper in app.js, got %q", body)
	}
	if !strings.Contains(body, "draftProviderSmokeFromMatrix") {
		t.Fatalf("expected provider smoke draft helper in app.js, got %q", body)
	}
	if !strings.Contains(body, "providerSmokeMatrixFilterLabel") {
		t.Fatalf("expected provider smoke matrix filter label helper in app.js, got %q", body)
	}
	if !strings.Contains(body, "selectionSourceLabel") {
		t.Fatalf("expected selection source label helper in app.js, got %q", body)
	}
	if !strings.Contains(body, "selectionScopeLabel") {
		t.Fatalf("expected selection scope label helper in app.js, got %q", body)
	}
	if !strings.Contains(body, "后台补传筛选已执行：${selectionSourceLabel(source)} / ${selectionScopeLabel(recoverScopeFromSource(source))}") {
		t.Fatalf("expected scoped auto recover selection flash text in app.js, got %q", body)
	}
	if !strings.Contains(body, "selectionScopeLabel(\"selected_pending_subset\")") {
		t.Fatalf("expected scoped selection label content in app.js, got %q", body)
	}
	if !strings.Contains(body, "后台补传子树已执行：") {
		t.Fatalf("expected scoped auto recover subtree flash text in app.js, got %q", body)
	}
	if !strings.Contains(body, "autoRecoverScopeFromPanel") {
		t.Fatalf("expected tree panel auto recover scope helper in app.js, got %q", body)
	}
	if !strings.Contains(body, "data-tree-auto-recover-panel") {
		t.Fatalf("expected tree auto recover panel wiring in app.js, got %q", body)
	}
	if !strings.Contains(body, "outcomes ") {
		t.Fatalf("expected outcome summary text in app.js, got %q", body)
	}
	if !strings.Contains(body, "classes ") {
		t.Fatalf("expected retry class summary text in app.js, got %q", body)
	}
	if !strings.Contains(body, "states ") {
		t.Fatalf("expected recover state summary text in app.js, got %q", body)
	}
	if !strings.Contains(body, "actions ") {
		t.Fatalf("expected blocked action summary text in app.js, got %q", body)
	}
	if !strings.Contains(body, "profiles ") {
		t.Fatalf("expected profile summary text in app.js, got %q", body)
	}
	if !strings.Contains(body, "lanes ") {
		t.Fatalf("expected lane summary text in app.js, got %q", body)
	}
	if !strings.Contains(body, "suggest ") {
		t.Fatalf("expected suggested budget summary text in app.js, got %q", body)
	}
	if !strings.Contains(body, "groups ") {
		t.Fatalf("expected protocol group summary text in app.js, got %q", body)
	}
	if !strings.Contains(body, "providers ") {
		t.Fatalf("expected provider summary text in app.js, got %q", body)
	}
	if !strings.Contains(body, "risk hints:") {
		t.Fatalf("expected provider risk hints text in app.js, got %q", body)
	}
	if !strings.Contains(body, "risk traits:") {
		t.Fatalf("expected provider risk traits text in app.js, got %q", body)
	}
	if !strings.Contains(body, "default risk:") {
		t.Fatalf("expected provider default risk text in app.js, got %q", body)
	}
	if !strings.Contains(body, "recommended risk:") {
		t.Fatalf("expected provider recommended risk text in app.js, got %q", body)
	}
	if !strings.Contains(body, "calibration coverage") {
		t.Fatalf("expected provider calibration coverage text in app.js, got %q", body)
	}
	if !strings.Contains(body, "risk calibration:") {
		t.Fatalf("expected provider risk calibration text in app.js, got %q", body)
	}
	if !strings.Contains(body, "calibration missing") {
		t.Fatalf("expected provider calibration missing text in app.js, got %q", body)
	}
	if !strings.Contains(body, "recover budget:") {
		t.Fatalf("expected provider recover budget text in app.js, got %q", body)
	}
	if !strings.Contains(body, "profile risk source:") {
		t.Fatalf("expected provider profile risk source text in app.js, got %q", body)
	}
	if !strings.Contains(body, "profile risk advice:") {
		t.Fatalf("expected provider profile risk advice text in app.js, got %q", body)
	}
	if !strings.Contains(body, "renderRiskDefaultsSourceBadge") {
		t.Fatalf("expected provider risk source badge helper in app.js, got %q", body)
	}
	if !strings.Contains(body, "账号默认来源:") {
		t.Fatalf("expected profile list risk source text in app.js, got %q", body)
	}
	if !strings.Contains(body, "账号默认建议:") {
		t.Fatalf("expected profile list risk advice text in app.js, got %q", body)
	}
	if !strings.Contains(body, "budget advice") {
		t.Fatalf("expected provider budget advice text in app.js, got %q", body)
	}
	if !strings.Contains(body, "账号恢复预算建议") {
		t.Fatalf("expected profile recover budget advice text in app.js, got %q", body)
	}
	if !strings.Contains(body, "fallback:") {
		t.Fatalf("expected provider fallback text in app.js, got %q", body)
	}
	if !strings.Contains(body, "conflict:") {
		t.Fatalf("expected provider conflict text in app.js, got %q", body)
	}
	if !strings.Contains(body, "renderProviderCapabilityDetail") {
		t.Fatalf("expected provider capability detail renderer in app.js, got %q", body)
	}
	if !strings.Contains(body, "syncTargetProviderInsight") {
		t.Fatalf("expected target provider insight renderer in app.js, got %q", body)
	}
	if !strings.Contains(body, "loadProviderCapabilityDetail") {
		t.Fatalf("expected provider capability detail loader in app.js, got %q", body)
	}
	if !strings.Contains(body, "data-provider-detail-open") {
		t.Fatalf("expected provider capability detail button wiring in app.js, got %q", body)
	}
	if !strings.Contains(body, "#apply-provider-default-risk") {
		t.Fatalf("expected apply provider default risk handler in app.js, got %q", body)
	}
	if !strings.Contains(body, "#open-target-provider-capability") {
		t.Fatalf("expected open target provider capability handler in app.js, got %q", body)
	}
	if !strings.Contains(body, "#apply-recommended-risk") {
		t.Fatalf("expected apply recommended risk handler in app.js, got %q", body)
	}
	if !strings.Contains(body, "已采用 provider 推荐风控") {
		t.Fatalf("expected provider default risk flash text in app.js, got %q", body)
	}
	if !strings.Contains(body, "已打开 ") {
		t.Fatalf("expected provider capability flash text in app.js, got %q", body)
	}
	if !strings.Contains(body, "已采用推荐风控档位") {
		t.Fatalf("expected recommended risk flash text in app.js, got %q", body)
	}
	if !strings.Contains(body, "earliest ") {
		t.Fatalf("expected earliest next retry summary text in app.js, got %q", body)
	}
	if !strings.Contains(body, "waiting_local_restore") {
		t.Fatalf("expected waiting_local_restore recoverState affordance in app.js, got %q", body)
	}
	if !strings.Contains(body, "waiting_manual_confirmation") {
		t.Fatalf("expected waiting_manual_confirmation recoverState affordance in app.js, got %q", body)
	}
	if !strings.Contains(body, "waiting_provider_session") {
		t.Fatalf("expected waiting_provider_session recoverState affordance in app.js, got %q", body)
	}
	if !strings.Contains(body, "waiting_retry_limit") {
		t.Fatalf("expected waiting_retry_limit recoverState affordance in app.js, got %q", body)
	}
	if !strings.Contains(body, "<th>Retry Scope</th>") {
		t.Fatalf("expected Retry Scope column in app.js, got %q", body)
	}
	if !strings.Contains(body, "<th>Retry Paths</th>") {
		t.Fatalf("expected Retry Paths column in app.js, got %q", body)
	}
}
