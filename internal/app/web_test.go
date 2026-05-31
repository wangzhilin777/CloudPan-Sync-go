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
	if !strings.Contains(body, `id="risk-max-concurrent"`) {
		t.Fatalf("expected risk concurrency input in html body, got %q", body)
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
	if !strings.Contains(body, "CALIBRATED") {
		t.Fatalf("expected CALIBRATED risk resolution detail in app.js, got %q", body)
	}
	if !strings.Contains(body, "recommendedExecutionMode") {
		t.Fatalf("expected recommendedExecutionMode wiring in app.js, got %q", body)
	}
	if !strings.Contains(body, "recommendedExecutionModeReason") {
		t.Fatalf("expected recommendedExecutionModeReason wiring in app.js, got %q", body)
	}
	if !strings.Contains(body, "风险档位") {
		t.Fatalf("expected risk mode label in app.js, got %q", body)
	}
	if !strings.Contains(body, "风险节流") {
		t.Fatalf("expected risk throttle label in app.js, got %q", body)
	}
	if !strings.Contains(body, "风险模板解释") {
		t.Fatalf("expected risk resolution label in app.js, got %q", body)
	}
	if !strings.Contains(body, "OVERRIDE FIELDS") {
		t.Fatalf("expected OVERRIDE FIELDS risk resolution detail in app.js, got %q", body)
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
	if !strings.Contains(body, "data-auto-recover-focus-lane-strategy") {
		t.Fatalf("expected auto recover summary lane strategy focus wiring in app.js, got %q", body)
	}
	if !strings.Contains(body, "data-auto-recover-preview-lane-strategy") {
		t.Fatalf("expected auto recover summary lane strategy preview wiring in app.js, got %q", body)
	}
	if !strings.Contains(body, "data-auto-recover-run-lane-strategy") {
		t.Fatalf("expected auto recover summary lane strategy run wiring in app.js, got %q", body)
	}
	if !strings.Contains(body, "data-auto-recover-preview-protocol-group") {
		t.Fatalf("expected auto recover preview protocol group wiring in app.js, got %q", body)
	}
	if !strings.Contains(body, "button.dataset.autoRecoverPreviewProtocolGroup || \"\"") {
		t.Fatalf("expected auto recover preview protocol group filter sync in app.js, got %q", body)
	}
	if !strings.Contains(body, "data-provider-smoke-draft") {
		t.Fatalf("expected provider smoke matrix draft action in app.js, got %q", body)
	}
	if !strings.Contains(body, "data-provider-smoke-focus-group") {
		t.Fatalf("expected provider smoke matrix focus-group action in app.js, got %q", body)
	}
	if !strings.Contains(body, "data-provider-smoke-filter-status") {
		t.Fatalf("expected provider smoke matrix status filter action in app.js, got %q", body)
	}
	if !strings.Contains(body, "draftProviderSmokeFromMatrix") {
		t.Fatalf("expected provider smoke matrix draft helper in app.js, got %q", body)
	}
	if !strings.Contains(body, "focusProviderSmokeRecordsByGroup") {
		t.Fatalf("expected provider smoke matrix focus helper in app.js, got %q", body)
	}
	if !strings.Contains(body, "已按验收矩阵预填 smoke 表单") {
		t.Fatalf("expected provider smoke matrix draft flash text in app.js, got %q", body)
	}
	if !strings.Contains(body, "data-report-view") {
		t.Fatalf("expected report history view action in app.js, got %q", body)
	}
	if !strings.Contains(body, "data-report-download") {
		t.Fatalf("expected report history download action in app.js, got %q", body)
	}
	if !strings.Contains(body, "renderReportHistory") {
		t.Fatalf("expected report history renderer in app.js, got %q", body)
	}
	if !strings.Contains(body, "selectedEvidenceReport") {
		t.Fatalf("expected selectedEvidenceReport helper in app.js, got %q", body)
	}
	if !strings.Contains(body, "已切换验收报告") {
		t.Fatalf("expected report history switch flash text in app.js, got %q", body)
	}
	if !strings.Contains(body, "验收报告已保存") {
		t.Fatalf("expected report save flash text in app.js, got %q", body)
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
	if !strings.Contains(body, "review_and_reset_retry_strategy") {
		t.Fatalf("expected review_and_reset_retry_strategy guide action in app.js, got %q", body)
	}
	if !strings.Contains(body, "打开授权面板") {
		t.Fatalf("expected open providers guide label in app.js, got %q", body)
	}
	if !strings.Contains(body, "查看状态矩阵") {
		t.Fatalf("expected open status guide label in app.js, got %q", body)
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
	if !strings.Contains(body, "provider-smoke-records-filter-clear") {
		t.Fatalf("expected provider smoke filter clear action in app.js, got %q", body)
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
	if !strings.Contains(body, "后台补传筛选已执行：paths") {
		t.Fatalf("expected scoped auto recover selection flash text in app.js, got %q", body)
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
	if !strings.Contains(body, "earliest ") {
		t.Fatalf("expected earliest next retry summary text in app.js, got %q", body)
	}
	if !strings.Contains(body, "waiting_local_restore") {
		t.Fatalf("expected waiting_local_restore recoverState affordance in app.js, got %q", body)
	}
	if !strings.Contains(body, "waiting_manual_confirmation") {
		t.Fatalf("expected waiting_manual_confirmation recoverState affordance in app.js, got %q", body)
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
