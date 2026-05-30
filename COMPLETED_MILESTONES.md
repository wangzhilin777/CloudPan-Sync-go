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
