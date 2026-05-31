# Completed Milestones

## 2026-05-31 - 后台补传策略筛选与推荐预算展示

- 对齐 [docs/01-GO_REBUILD_PLAN.md](E:/Workspace/VSCode/CloudPan-Sync-go/docs/01-GO_REBUILD_PLAN.md) 中“更复杂的后台补传编排、批量筛选与多策略调度”方向，先补齐 `strategy` 维度的后台补传筛选链路。
- 后端 `RecoverBlockedTasksWithOptions` 新增 `strategy` 过滤条件，并通过 `recoverTaskStrategy` 从结果 payload / plan item 中回收任务策略。
- `AutoRecoverLane` 汇总新增 `strategies` 与 `sampleStrategy`，状态矩阵可以看到样本策略与策略集合。
- `/api/tasks/recover` 新增 `strategy` 请求字段，前端状态页新增“全部策略 / fast_upload / download_upload / pending_manual”筛选控件。
- 状态页自动补传筛选、请求构造、筛选摘要、重置逻辑与事件绑定已全部接入 `strategy`。
- 风险解决明细面板新增推荐补传预算展示：`RECOVER BUDGET`、`RECOVER REASON`、`SENSITIVE PROVIDERS`。
- 回归验证已通过：`go test ./...`、`node --check web/static/app.js`。
- 清理情况：本轮未启动额外后台进程；测试使用 `t.TempDir()`，未保留额外临时目录或构建残留。

## 2026-05-31 - 后台补传策略决策回显与统计展示

- 在上一里程碑的 `strategy` 筛选基础上，继续把策略维度补到后台补传结果回显链路。
- 后端 `RecoverDecision` 新增 `strategy` 字段，`RecoverResult` 新增 `strategyCounts` 聚合统计，dry-run / execute 两条结果链路都会回传命中策略。
- `appendRecoverDecision` 已把策略计数纳入结果汇总，状态矩阵最近一次后台补传结果可直接看到策略分布。
- 前端最近一次后台补传 summary 新增 `strategies ...` 汇总段，decision detail 新增 `strategy` pill，用户可以直接看到每条决策对应的实际策略。
- 服务层回归测试已补强：`TestServiceRecoverBlockedTasksWithOptionsFiltersStrategy` 现在同时断言 `strategy` 请求回显、`strategyCounts` 聚合值、以及 `decisions[0].strategy` 决策回显。
- 回归验证已通过：`go test ./internal/task ./internal/app`、`node --check web/static/app.js`。
- 清理情况：本轮未启动额外后台进程；未新增持久临时目录，测试仍使用 `t.TempDir()` 自动清理。

## 2026-05-31 - 后台补传决策级策略透传

- 把最近一次后台补传结果里的 `strategy` 继续透传到“预演该决策 / 执行该决策”按钮请求，避免用户从决策明细再次触发时丢失策略上下文。
- 前端 decision action dataset 已补上 `data-auto-recover-decision-strategy`，`currentAutoRecoverDecisionRequest()` 也会把 `strategy` 一起带回 `/api/tasks/recover`。
- 这样状态矩阵里的单条决策从查看、预演到执行，使用的是同一份策略上下文，和当前筛选链路保持一致。
- 回归验证已通过：`go test ./internal/app ./internal/task`、`node --check web/static/app.js`。
- 清理情况：本轮未启动额外后台进程；未新增临时目录或构建残留。

## 2026-05-31 - 后台补传决策级策略筛选状态同步

- 继续对齐 [docs/01-GO_REBUILD_PLAN.md](E:/Workspace/VSCode/CloudPan-Sync-go/docs/01-GO_REBUILD_PLAN.md) 中后台补传多策略调度细化目标，补齐决策级 action 与当前筛选状态之间最后一段 `strategy` 同步链路。
- `triggerAutoRecoverDecision()` 现在会把 `payload.strategy` 一起写回 `applyAutoRecoverFilters(...)`，因此从“预演该决策 / 执行该决策”进入时，页面当前筛选条件会和本次决策请求保持同一份策略上下文。
- Web smoke 断言已补强：静态资源测试现在同时校验 `data-auto-recover-decision-strategy` 数据透传，以及 `strategy: payload.strategy` 这段前端状态同步代码仍然存在。
- 回归验证已通过：`go test ./internal/app ./internal/task`、`node --check web/static/app.js`。
- 清理情况：本轮未启动额外后台进程；未生成需保留的临时目录、smoke 目录或构建残留。

## 2026-05-31 - 后台补传 lane 级策略状态同步

- 继续对齐 [docs/01-GO_REBUILD_PLAN.md](E:/Workspace/VSCode/CloudPan-Sync-go/docs/01-GO_REBUILD_PLAN.md) 中“更复杂的后台补传编排、批量筛选与多策略调度”方向，把状态页候选池 lane 级动作也补齐 `strategy` 上下文。
- 候选池里的“只看该 lane / 预演该 lane / 执行该 lane”按钮现在都会透传 `sampleStrategy`，避免从 lane 聚焦或 lane 级预演/执行进入时丢失策略筛选条件。
- `currentAutoRecoverRequest()` 现已优先读取 `state.autoRecoverFilters`，即使按钮动作使用 `render: false` 先更新筛选状态、后立即触发请求，也不会因为 DOM 尚未刷新而漏掉当前 `strategy` / 预算条件。
- Web smoke 断言已补强：静态资源测试现在同时校验 lane 级 `data-auto-recover-focus-lane-strategy / data-auto-recover-preview-lane-strategy / data-auto-recover-run-lane-strategy` 透传，以及请求构造优先读取当前筛选 state 的实现仍然存在。
- 回归验证已通过：`go test ./internal/app ./internal/task`、`node --check web/static/app.js`。
- 清理情况：本轮未启动额外后台进程；未新增需保留的临时目录、smoke 目录或构建残留。

## 2026-05-31 - 后台补传决策明细 lane 级策略状态同步

- 继续对齐 [docs/01-GO_REBUILD_PLAN.md](E:/Workspace/VSCode/CloudPan-Sync-go/docs/01-GO_REBUILD_PLAN.md) 中后台补传多策略调度的细化目标，把“最近一次后台补传结果”里的决策明细 lane 级聚焦动作也补齐 `strategy` 上下文。
- 决策明细里的“只看该 lane”按钮现在会额外透传 `data-auto-recover-decision-focus-lane-strategy`，避免用户从单条决策收敛 lane 候选池时丢失策略筛选条件。
- `wireAutoRecoverLastResultDetail()` 现已在决策明细 lane 聚焦时同步写回 `{ mode, strategy, retryClass, blockedAction }`，并在提示文案里回显完整 lane 维度，和 summary 区域的 lane 动作保持一致。
- Web smoke 断言已补强：静态资源测试现在同时校验决策明细 lane 级 `data-auto-recover-decision-focus-lane-strategy` 透传，以及 `const strategy = button.dataset.autoRecoverDecisionFocusLaneStrategy || ""` 这段前端状态同步代码仍然存在。
- 回归验证待本次提交前执行：`go test ./internal/app ./internal/task`、`node --check web/static/app.js`。
- 清理情况：本轮验证结束后会确认未遗留额外后台进程、临时目录、smoke 目录或构建残留。

## 2026-05-31 - 后台补传协议族预演走通

- 继续对齐 [docs/01-GO_REBUILD_PLAN.md](E:/Workspace/VSCode/CloudPan-Sync-go/docs/01-GO_REBUILD_PLAN.md) 中“更复杂的后台补传编排、批量筛选与多策略调度”方向，把协议族维度的后台补传动作也补齐 dry-run 预演链路。
- 状态页 summary 里的样本协议族现在新增 `data-auto-recover-preview-protocol-group` 按钮，用户可以先按协议族收敛候选并直接预演，而不必先执行真实后台补传。
- `wireAutoRecoverSummary()` 现已在协议族预演时同步写回 `protocolGroup` 过滤条件，并走 `triggerAutoRecover({ dryRun: true })`，和 lane 级预演保持一致的交互节奏。
- Web smoke 断言已补强：静态资源测试现在同时校验协议族预演按钮透传，以及 `button.dataset.autoRecoverPreviewProtocolGroup || ""` 这段前端请求构造代码仍然存在。
- 回归验证待本次提交前执行：`go test ./internal/app ./internal/task`、`node --check web/static/app.js`。
- 清理情况：本轮验证结束后会确认未遗留额外后台进程、临时目录、smoke 目录或构建残留。


## 2026-05-31 - 推荐模式与风险解释静态契约补强

- 继续对齐 [docs/01-GO_REBUILD_PLAN.md](E:/Workspace/VSCode/CloudPan-Sync-go/docs/01-GO_REBUILD_PLAN.md) 中“页面与 API 应给出推荐语义”要求，把前端已展示的推荐模式与风险解释字段补成 smoke 级静态契约。
- `internal/app/web_test.go` 现在会同时校验 `recommendedExecutionMode`、`recommendedExecutionModeReason`、`风险档位`、`风险节流`、`风险模板解释` 这些关键展示点仍然存在，避免后续前端回归时把推荐语义悄悄删掉。
- 这次改动不改业务逻辑，只把已经落地的推荐模式 / 风险解释展示进一步固定成可回归验证的契约。
- 回归验证待本次提交前执行：`go test ./internal/app`。
- 清理情况：本轮验证结束后会确认未遗留额外后台进程、临时目录、smoke 目录或构建残留。


## 2026-05-31 - 推荐执行模式 UI 闭环走通

- 继续对齐 [docs/01-GO_REBUILD_PLAN.md](E:/Workspace/VSCode/CloudPan-Sync-go/docs/01-GO_REBUILD_PLAN.md) 中“可选模式 + 默认推荐模式 + 推荐提示语义”要求，把推荐执行模式从静态展示补成可点击、可回填、可验证的 UI 闭环。
- `internal/app/ui_smoke_test.go` 现在会在计划预览后显式等待推荐标题和推荐原因出现，再点击 `采用推荐模式` 按钮，并校验 `#plan-execution-mode` 已被回填为推荐值 `leaf_first_lazy`。
- 这次改动不改 planner 推荐逻辑，而是把“用户真的能采用推荐模式继续创建任务”这条交互链路固定成回归测试。
- 回归验证待本次提交前执行：`go test ./internal/app`。
- 清理情况：本轮验证结束后会确认未遗留额外后台进程、临时目录、smoke 目录或构建残留。


## 2026-05-31 - 验收矩阵 UI 预填与筛选闭环走通

- 继续对齐 [docs/01-GO_REBUILD_PLAN.md](E:/Workspace/VSCode/CloudPan-Sync-go/docs/01-GO_REBUILD_PLAN.md) 中“状态矩阵与证据页展示”和“真实 provider smoke 记录与协议组聚合矩阵”方向，把真实样本矩阵从展示推进到可操作闭环。
- `internal/app/ui_smoke_test.go` 现在会在状态页等待 `provider-smoke-matrix` 渲染出 `aliyun_123_open`，再触发“预填 smoke 表单”，校验 provider、protocol group、标题和备注都被矩阵行回填。
- 同一条浏览器级 smoke 还会触发“只看该组记录”，校验 `#provider-smoke-records-filter-group` 被同步成 `aliyun_123_open`，保证验收矩阵能直接收敛下方 smoke 记录视图。
- 回归验证已通过：`go test ./internal/app`、`node --check web/static/app.js`。
- 清理情况：本轮验证中曾清理超时调试遗留的 `go/chrome` 测试进程；最终通过的验证未遗留额外后台进程、临时目录、smoke 目录或构建残留。


## 2026-05-31 - 验收矩阵前端交互静态契约补强

- 在浏览器级 smoke 的基础上，继续把验收矩阵关键交互固定成静态资源契约，避免后续前端重构时把矩阵动作钩子悄悄删掉。
- `internal/app/web_test.go` 现在会校验 `data-provider-smoke-draft`、`data-provider-smoke-focus-group`、`data-provider-smoke-filter-status`、`draftProviderSmokeFromMatrix()`、`focusProviderSmokeRecordsByGroup()` 和“已按验收矩阵预填 smoke 表单”提示语仍然存在。
- 这次改动不改 API 或业务状态，只把已经落地的真实样本矩阵操作入口补成可回归契约。
- 回归验证已通过：`go test ./internal/app`、`node --check web/static/app.js`。
- 清理情况：本轮验证中曾清理超时调试遗留的 `go/chrome` 测试进程；最终通过的验证未遗留额外后台进程、临时目录、smoke 目录或构建残留。


## 2026-05-31 - 验收报告历史交互静态契约补强

- 继续沿着 [docs/01-GO_REBUILD_PLAN.md](E:/Workspace/VSCode/CloudPan-Sync-go/docs/01-GO_REBUILD_PLAN.md) 中“状态矩阵/证据页展示”方向补齐验收报告链路，把报告历史切换与下载入口固定成前端静态契约。
- `internal/app/web_test.go` 现在会额外校验 `data-report-view`、`data-report-download`、`renderReportHistory()`、`selectedEvidenceReport()`、`验收报告已保存`、`已切换验收报告` 这些关键钩子和提示语仍然存在。
- 这次改动不改 API 或持久化逻辑，只把已经落地的报告历史交互入口补成可回归约束，避免后续前端重构时丢失历史查看/下载链路。
- 回归验证已通过：`go test ./internal/app`、`node --check web/static/app.js`。
- 清理情况：本轮验证未遗留额外后台进程、临时目录、smoke 目录或构建残留。


## 2026-05-31 - blocked action 处理引导静态契约补强

- 继续沿着 [docs/01-GO_REBUILD_PLAN.md](E:/Workspace/VSCode/CloudPan-Sync-go/docs/01-GO_REBUILD_PLAN.md) 中“状态矩阵/证据页展示”和“blockedAction / blockedAdvice 便于定位处理步骤”方向，补齐任务详情“下一步处理”引导区的前端契约。
- `internal/app/web_test.go` 现在会额外校验 `renderTaskResolutionGuide()`、`wireTaskResolutionGuide()`、`data-task-guide-view`、`data-task-guide-intent`，以及 `refresh_auth_profile`、`restore_local_source_file`、`manual_confirmation_required`、`review_and_reset_retry_strategy` 这些关键 blocked action 分支仍然存在。
- 同一组契约也会兜住“打开授权面板”“查看状态矩阵”等处理入口文案，避免后续前端重构时把任务详情里的处理引导和直达按钮悄悄删掉。
- 回归验证已通过：`go test ./internal/app -run TestRoutesServeAppJSIncludesRetryEvidenceLabels -v`、`go test ./internal/app -run TestConsoleUISmokeMainline -v`、`node --check web/static/app.js`。
- 清理情况：本轮验证未遗留额外后台进程、临时目录、smoke 目录或构建残留。


## 2026-05-31 - blocked 聚合与 smoke 记录动作静态契约补强

- 继续沿着 [docs/01-GO_REBUILD_PLAN.md](E:/Workspace/VSCode/CloudPan-Sync-go/docs/01-GO_REBUILD_PLAN.md) 中“状态矩阵/证据页展示”和“真实 provider smoke 记录与协议组聚合矩阵”方向，把状态页 blocked 聚合和 smoke 记录区的前端动作补成静态契约。
- `internal/app/web_test.go` 现在会额外校验 `renderBlockedActionsSummary()`、`data-blocked-focus-action`、`focusBlockedActionSummary()`、`已按 blocked action 收敛最近重试队列`，保证状态页 blocked 聚合看板的聚焦入口仍然存在。
- 同一组契约也会兜住 `data-provider-smoke-view`、`data-provider-smoke-download`、`provider-smoke-records-filter-clear`，以及“已切换 smoke Markdown”“smoke Markdown 已下载”“已清空 smoke 记录筛选”等提示语，避免后续前端重构时把 smoke 记录查看/下载/清空筛选链路删掉。
- 回归验证已通过：`go test ./internal/app -run TestRoutesServeAppJSIncludesRetryEvidenceLabels -v`、`go test ./internal/app -run TestConsoleUISmokeMainline -v`、`node --check web/static/app.js`。
- 清理情况：本轮验证未遗留额外后台进程、临时目录、smoke 目录或构建残留。


## 2026-05-31 - 目录树与待补传可见动作静态契约补强

- 继续沿着 [docs/01-GO_REBUILD_PLAN.md](E:/Workspace/VSCode/CloudPan-Sync-go/docs/01-GO_REBUILD_PLAN.md) 中“状态矩阵/证据页展示”和“目录树/待补传树辅助操作”方向，把树节点动作和可见筛选动作补成前端静态契约。
- `internal/app/web_test.go` 现在会额外校验 `data-tree-group-toggle`、`data-tree-focus-path`、`data-tree-sync-path`、`data-tree-prefill-path`、`data-tree-retry-path`、`data-tree-copy-path`、`data-tree-parent-path`，以及 `wireTreeGroupToggles()`、`同步另一棵树`、`只看当前路径` 这些树节点动作钩子与文案仍然存在。
- 同一组契约也会兜住 `task-directory-prefill-visible`、`task-retry-visible-directory`、`task-directory-copy-visible`、`task-pending-prefill-visible`、`task-retry-visible-pending`、`task-pending-copy-visible`、`status-directory-prefill-visible`、`status-retry-visible-directory`、`status-directory-copy-visible`、`status-pending-prefill-visible`、`status-retry-visible-pending`、`status-pending-copy-visible`，以及筛选清空按钮和“已按当前任务重建向导参数”“已复制”等提示语，避免后续前端重构时把可见路径操作链路删掉。
- 回归验证已通过：`go test ./internal/app -run TestRoutesServeAppJSIncludesRetryEvidenceLabels -v`、`go test ./internal/app -run TestConsoleUISmokeMainline -v`、`node --check web/static/app.js`。
- 清理情况：本轮验证未遗留额外后台进程、临时目录、smoke 目录或构建残留。


## 2026-05-31 - 验收矩阵样本直达动作静态契约补强

- 继续沿着 [docs/01-GO_REBUILD_PLAN.md](E:/Workspace/VSCode/CloudPan-Sync-go/docs/01-GO_REBUILD_PLAN.md) 中“状态矩阵/证据页展示”和“真实 provider smoke 记录与协议组聚合矩阵”方向，把验收矩阵中直达样本的动作入口固定成前端静态契约。
- `internal/app/web_test.go` 现在会额外校验 `data-provider-smoke-open-record`、`data-provider-smoke-open-task`、`setProviderSmokeMatrixFilter()`，以及“已打开 smoke 样本”“已打开 blocked 摘要对应的样本任务”这些关键提示语仍然存在，避免后续前端重构时把矩阵里的直达样本入口悄悄删掉。
- 这次改动不改 API 或业务状态，只把已经落地的矩阵动作入口补成可回归约束，让协议组验收矩阵和 smoke / 样本任务之间的跳转关系更稳。
- 回归验证已通过：`go test ./internal/app -run TestRoutesServeAppJSIncludesRetryEvidenceLabels -v`、`go test ./internal/app -run TestConsoleUISmokeMainline -v`、`node --check web/static/app.js`。
- 清理情况：本轮验证未遗留额外后台进程、临时目录、smoke 目录或构建残留。


## 2026-05-31 - 验收报告下载入口静态契约补强

- 继续沿着 [docs/01-GO_REBUILD_PLAN.md](E:/Workspace/VSCode/CloudPan-Sync-go/docs/01-GO_REBUILD_PLAN.md) 中“状态矩阵/证据页展示”方向，把验收报告下载链路补成可回归的前端静态契约。
- `internal/app/web_test.go` 现在会额外校验“验收报告已下载”这条提示语仍然存在，与已落地的 `data-report-download`、`renderReportHistory()`、`selectedEvidenceReport()` 一起兜住报告下载交互。
- 这次改动不改报告生成或持久化逻辑，只把现有下载入口补成更完整的契约，避免后续前端改动时把下载反馈语义弄丢。
- 回归验证已通过：`go test ./internal/app -run TestRoutesServeAppJSIncludesRetryEvidenceLabels -v`、`go test ./internal/app -run TestConsoleUISmokeMainline -v`、`node --check web/static/app.js`。
- 清理情况：本轮验证未遗留额外后台进程、临时目录、smoke 目录或构建残留。

