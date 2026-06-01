## 2026-06-01 - provider_session_missing 浏览器级异常 smoke 补强

- 继续按 [docs/01-GO_REBUILD_PLAN.md](E:/Workspace/VSCode/CloudPan-Sync-go/docs/01-GO_REBUILD_PLAN.md) 中 UI smoke 与异常场景提示校验方向推进，这轮把 `provider_session_missing / manual_intervention_required` 从 API workflow 与静态契约进一步补进浏览器级 smoke。
- [internal/app/ui_smoke_test.go](E:/Workspace/VSCode/CloudPan-Sync-go/internal/app/ui_smoke_test.go) 现在会真实构造一个缺 `uploadid` 的 blocked 任务，并在控制台里校验 blocked 任务对象、浏览器内任务选择状态，以及状态页 blocked action 筛选入口。
- 回归验证已通过：`go test ./internal/app -run TestConsoleUISmokeMainline -v`、`go test ./...`、`node --check web/static/app.js`。
- 清理情况：本轮验证未遗留 `.codex_tmp*` 临时文件、额外 smoke 目录或后台测试进程；测试使用 `httptest`、`chromedp` 与 `t.TempDir()`，结束后已自动释放。
# Completed Milestones

## 2026-06-01 - provider_session_missing 前端筛选与处理契约补强

- 继续沿着 [docs/01-GO_REBUILD_PLAN.md](E:/Workspace/VSCode/CloudPan-Sync-go/docs/01-GO_REBUILD_PLAN.md) 中“状态矩阵/证据页展示”与“更复杂的后台补传编排、批量筛选与多策略调度”方向推进，这轮把新引入的 `manual_intervention_required` / `provider_session_missing` 处理链补进前端可见筛选与静态契约。
- [web/static/app.js](E:/Workspace/VSCode/CloudPan-Sync-go/web/static/app.js) 现在能把 `manual_intervention_required` 映射成 provider 会话缺口的明确处理引导，并把 `provider_session_missing` 纳入 retry class / blocked action 统计与过滤入口。
- [web/static/index.html](E:/Workspace/VSCode/CloudPan-Sync-go/web/static/index.html) 现在补上 `provider_session_missing` 的重试分类筛选项，状态页、任务页与后台补传候选池都能直接按这类失败定位。
- [internal/app/web_test.go](E:/Workspace/VSCode/CloudPan-Sync-go/internal/app/web_test.go) 继续补齐静态契约，固定前端引导文案与新筛选入口不会在后续重构中丢失。
- 回归验证已通过：`go test ./...`、`node --check web/static/app.js`。
- 清理情况：本轮验证未遗留额外后台进程、临时脚本、smoke 目录或构建残留。

## 2026-06-01 - missing_uploadid runtime 阻断与 API 回显收口

- 继续按 [docs/01-GO_REBUILD_PLAN.md](E:/Workspace/VSCode/CloudPan-Sync-go/docs/01-GO_REBUILD_PLAN.md) 中 provider 上传接口契约、runtime `retry queue / pending relay` 与 Phase 6 错误收口主线推进，这轮把 `missing_uploadid` 从普通 `retry_failed` 中拆出来，作为 provider 上传会话信息不完整的明确人工处理阻断。
- [internal/task/service.go](E:/Workspace/VSCode/CloudPan-Sync-go/internal/task/service.go) 现在会把 `missing_uploadid` 映射为 `RetryClass provider_session_missing + RetryAction manual_intervention_required`，并在 retry summary 中产出 `retry_queue_requires_provider_session_rebuild / manual_intervention_required`，避免后台自动重试反复撞同一个缺 uploadid 的会话错误。
- [internal/task/service_test.go](E:/Workspace/VSCode/CloudPan-Sync-go/internal/task/service_test.go) 新增 runtime 集成契约，固定 `missing_uploadid` 的 retry queue、blockedReason、blockedAction 和 blocked 状态。
- [internal/app/workflow_test.go](E:/Workspace/VSCode/CloudPan-Sync-go/internal/app/workflow_test.go) 新增 API workflow 契约，固定任务运行接口会把 `missing_uploadid` 的 `blockedReason / blockedAction / retryQueue` 明确回显给前端消费。
- 回归验证已通过：`go test ./internal/planner ./internal/provider ./internal/task ./internal/app`。
- 清理情况：本轮未遗留额外后台进程、临时目录、smoke 目录或构建残留；测试使用 `httptest` 与 `t.TempDir()`，结束后已自动清理，未额外停止用户自有进程。

## 2026-06-01 - provider auth_invalid 与上传失败归类契约补强

- 继续按 [docs/01-GO_REBUILD_PLAN.md](E:/Workspace/VSCode/CloudPan-Sync-go/docs/01-GO_REBUILD_PLAN.md) 中 Phase 3 provider 核心上传接口契约、runtime `auth expired / retry queue` 和 Phase 6 错误收口主线推进，这轮把 provider 上传阶段 `auth_invalid / provider_request_failed / missing_uploadid` 的上游语义补成 catalog 契约，并把 runtime 对真实授权失效状态的归一化补平。
- 新增 [internal/provider/catalog_auth_invalid_contract_test.go](E:/Workspace/VSCode/CloudPan-Sync-go/internal/provider/catalog_auth_invalid_contract_test.go)，覆盖 `guangya / aliyundrive_open / 123_open / 115_open / quark / uc / xunlei / pikpak / baidu_netdisk / 189cloud` 十家 provider 的真实上传入口，固定“坏 token/cookie 进入上传调用路径时必须显式返回 `auth_invalid`，不能伪装成一般失败或假成功”。
- 新增 [internal/provider/catalog_provider_request_failed_contract_test.go](E:/Workspace/VSCode/CloudPan-Sync-go/internal/provider/catalog_provider_request_failed_contract_test.go)，固定 `aliyundrive_open` 分片上传失败仍保持 `provider_request_failed + upload checkpoint`，以及 `baidu_netdisk` precreate 缺少 `uploadid` 时保持显式 `missing_uploadid` 特例，不被吞成普通失败。
- [internal/provider/http_json.go](E:/Workspace/VSCode/CloudPan-Sync-go/internal/provider/http_json.go) 与多家 provider family 的 JSON 解码 helper 统一补上 `401/403 + 非 JSON 响应体` 兜底，避免真实未授权响应被误吞成 `decode provider json`，从而把 `auth_invalid` 错分成 `provider_request_failed`。
- [internal/task/service.go](E:/Workspace/VSCode/CloudPan-Sync-go/internal/task/service.go) 现在会把真实 provider 返回的 `auth_invalid` 归一化映射进 runtime `auth_expired` retry lane；[internal/task/service_test.go](E:/Workspace/VSCode/CloudPan-Sync-go/internal/task/service_test.go) 也补上了对应契约，固定 `auth_invalid -> refresh_auth_profile` 这条处理链。
- 回归验证已通过：`go test ./internal/planner ./internal/provider ./internal/task`。
- 清理情况：本轮未遗留额外测试缓存、smoke 目录、临时数据库或后台测试服务；测试使用 `httptest` 与 `t.TempDir()`，结束后已自动清理，未额外停止用户自有进程。

## 2026-06-01 - runtime 错误分类到 retry queue 契约补强

- 继续按 [docs/01-GO_REBUILD_PLAN.md](E:/Workspace/VSCode/CloudPan-Sync-go/docs/01-GO_REBUILD_PLAN.md) 中 runtime `pending_manual / auth expired / rate limit / local file missing / retry queue` 主线推进，这轮把 provider 返回状态到 `retryQueue` 分类与动作映射补成更直接的集成契约。
- [internal/task/service_test.go](E:/Workspace/VSCode/CloudPan-Sync-go/internal/task/service_test.go) 在现有 `TestServiceRuntimeHandlesPendingManualAuthExpiredRateLimitAndMissingLocalFile` 上新增逐项断言，固定：
- `pending_manual_requires_confirmation -> RetryClass pending_manual + RetryAction retry_after_manual_confirmation`
- `auth_expired -> RetryClass auth_expired + RetryAction refresh_auth_profile`
- `rate_limited -> RetryClass rate_limited + RetryAction retry_after_cooldown`
- `local_file_missing -> RetryClass local_file_missing + RetryAction restore_local_file`
- 同时也固定这些项在 blocked / retryable 维度上的差异，避免后续 runtime 重构时把“可自动重试”和“必须人工处理”的边界悄悄混掉。
- 回归验证已通过：`go test ./internal/planner ./internal/provider ./internal/task`。
- 清理情况：本轮未遗留额外后台进程；测试使用临时目录并自动清理，未保留 smoke 目录、临时数据库或构建残留。

## 2026-06-01 - provider 缺本地文件与 hash miss fallback 契约补强

- 继续按 [docs/01-GO_REBUILD_PLAN.md](E:/Workspace/VSCode/CloudPan-Sync-go/docs/01-GO_REBUILD_PLAN.md) 中 runtime `hash miss -> binary fallback`、`local file missing` 与 Provider 核心上传接口契约推进，这轮把各家 provider 在“缺本地文件”场景下的上传阶段行为补成 catalog 级回归约束。
- 新增 [internal/provider/catalog_local_file_missing_contract_test.go](E:/Workspace/VSCode/CloudPan-Sync-go/internal/provider/catalog_local_file_missing_contract_test.go)，复用现有本地 test server / helper profile，固定 `guangya / aliyundrive_open / 123_open / 115_open / quark / uc / baidu_netdisk / 189cloud` 在 `Upload(strategy=download_upload)` 且无 `LocalPath` 时必须返回 `local_file_missing`。
- 同一组契约也把 `xunlei / pikpak` 单独收口为真实特例：当上传阶段已拿到 `GCID` 但没有本地文件可继续 binary fallback 时，provider 必须保持 `hash_miss` 阻断，并在消息里明确说明缺少本地文件，而不是伪装成成功上传或混成其它错误。
- 这样 provider 侧“缺本地文件 / hash miss 但无 fallback 文件”语义就和 [internal/task/service.go](E:/Workspace/VSCode/CloudPan-Sync-go/internal/task/service.go) 中 runtime `local_file_missing` 结果、hash miss fallback 护栏与 retry queue 分类形成了更稳定的上下游契约。
- 回归验证已通过：`go test ./internal/provider ./internal/task`。
- 清理情况：本轮未遗留额外后台进程；测试使用本地 `httptest` server 与临时目录，结束后已自动清理，未保留 smoke 目录或构建残留。

## 2026-06-01 - provider pending_manual 上传契约补强

- 继续按 [docs/01-GO_REBUILD_PLAN.md](E:/Workspace/VSCode/CloudPan-Sync-go/docs/01-GO_REBUILD_PLAN.md) 中 “Provider 契约测试覆盖每家 provider 的核心接口” 与 runtime `pending_manual` 主线推进，这轮把各家 provider 在上传阶段对 `pending_manual` 的统一阻断语义补成 catalog 级回归契约。
- 新增 [internal/provider/catalog_pending_manual_contract_test.go](E:/Workspace/VSCode/CloudPan-Sync-go/internal/provider/catalog_pending_manual_contract_test.go)，复用现有本地 test server / helper profile，覆盖 `guangya / aliyundrive_open / 123_open / 115_open / quark / uc / xunlei / pikpak / baidu_netdisk / 189cloud` 十家 provider 的 `Upload(strategy=pending_manual)` 行为。
- 这组契约固定各家在通过最小 live mock 前置校验后，`pending_manual` 仍必须返回 `pending_manual_requires_confirmation`，不能伪装成上传成功，也不能被 auth 前置噪音掩盖成错误的主线语义。
- 这样 provider 侧“待人工确认”分支就和 [internal/task/service.go](E:/Workspace/VSCode/CloudPan-Sync-go/internal/task/service.go) 里已有的 runtime `pending_manual` 重试分类、blocked reason 与 pending tree 聚合形成了更稳定的上下游契约。
- 回归验证已通过：`go test ./internal/provider ./internal/task`。
- 清理情况：本轮未遗留额外后台进程；测试使用本地 `httptest` server 与临时目录，结束后已自动清理，未保留 smoke 目录或构建残留。

## 2026-05-31 - 运行结果执行上下文透传收口

- 继续按 [docs/01-GO_REBUILD_PLAN.md](E:/Workspace/VSCode/CloudPan-Sync-go/docs/01-GO_REBUILD_PLAN.md) 中“task runtime evidence / provider probe 也应透传当前执行模式和扫描模式”方向推进，这轮把运行结果在 `skip / create / overwrite / upload` 各分支上的上下文字段透传统一收口。
- [internal/task/service.go](E:/Workspace/VSCode/CloudPan-Sync-go/internal/task/service.go) 新增 `attachPlanContextToResult()`，统一把 `executionMode`、`scanMode`、`recommendedExecutionMode`、`recommendedExecutionModeReason`、`sourceDeletePolicy`、retry 上下文和 `riskProfile` 注入到单条运行结果，避免 `skip` 分支遗漏字段。
- 同一处实现对 `scanMode` 做了按 `executionMode` 的兜底推导，因此即使 plan metadata 尚未预先写入 `scanMode`，运行结果里仍能稳定看到 `lazy_leaf_first / pre_scan_flat` 语义。
- [internal/task/service_test.go](E:/Workspace/VSCode/CloudPan-Sync-go/internal/task/service_test.go) 补强了 `skip` 与保守 `create` 两条契约，固定运行结果必须带出 `executionMode`、`scanMode` 和 `sourceDeletePolicy`，避免后续 runtime 重构时再次出现结果字段不一致。
- 回归验证已通过：`go test ./internal/planner ./internal/provider ./internal/task`。
- 清理情况：本轮未遗留额外后台进程；测试使用临时目录并自动清理，未保留临时数据库、smoke 目录或构建产物。

## 2026-05-31 - 多根推荐优先级与元数据保守判定契约补强

- 继续按 [docs/01-GO_REBUILD_PLAN.md](E:/Workspace/VSCode/CloudPan-Sync-go/docs/01-GO_REBUILD_PLAN.md) 中“多个顶层目录按勾选顺序逐棵子树推进”和“目标明确存在的保守判定”两条核心执行语义推进，本轮一次补齐 planner 推荐优先级和 runtime metadata 判定边界。
- [internal/planner/service.go](E:/Workspace/VSCode/CloudPan-Sync-go/internal/planner/service.go) 调整 `recommendExecutionMode()` 优先级，确保多 `selectedRoots` 即使在 `fast` 风险模式和小输入集下，也优先推荐 `leaf_first_lazy`，避免错误推荐成预扫模式。
- [internal/planner/recommendation_contract_test.go](E:/Workspace/VSCode/CloudPan-Sync-go/internal/planner/recommendation_contract_test.go) 新增 fast 模式多根契约，固定“多顶层目录优先子树逐棵推进”的推荐原因与 `leaf_first` 执行顺序。
- [internal/task/service_test.go](E:/Workspace/VSCode/CloudPan-Sync-go/internal/task/service_test.go) 新增 runtime metadata 契约，固定 `MetadataResult.Status == "exists"` 可作为明确存在并跳过上传，同时普通 `Status == "ok"` 且未声明 `entry.exists=true` 时必须保守按 `create` 处理，避免占位 provider 把文件误判为已同步。
- 回归验证已通过：`go test ./internal/planner ./internal/provider ./internal/task`。
- 清理情况：本轮未启动额外后台进程；测试使用 `t.TempDir()` 自动清理，未保留 smoke 目录、临时数据库或构建产物。

## 2026-05-31 - 快传输入与冲突策略主线契约补强

- 继续按 [docs/01-GO_REBUILD_PLAN.md](E:/Workspace/VSCode/CloudPan-Sync-go/docs/01-GO_REBUILD_PLAN.md) 中 Phase 2 / Phase 3 的 planner-provider-task 主线推进，把快传输入判定、provider 快传能力声明和 runtime 冲突策略降级一起补成可回归契约。
- 新增 [internal/planner/fastupload_contract_test.go](E:/Workspace/VSCode/CloudPan-Sync-go/internal/planner/fastupload_contract_test.go)，固定 `md5` provider 可接受 `etag` 兜底、`gcid + size` 家族快传输入、缺失指纹时小文件走 `download_upload`、大文件进入 `pending_manual` 的 planner 判定。
- 新增 [internal/provider/catalog_fastupload_contract_test.go](E:/Workspace/VSCode/CloudPan-Sync-go/internal/provider/catalog_fastupload_contract_test.go)，固定 `aliyundrive_open / 123_open / xunlei / guangya` 的 `FastUploadInputs`、`FallbackModes`、协议组和授权模式数量，避免 catalog 重构时破坏协议族能力声明。
- 新增 [internal/task/conflict_policy_contract_test.go](E:/Workspace/VSCode/CloudPan-Sync-go/internal/task/conflict_policy_contract_test.go)，固定 runtime `resolveConflictPolicy()` 的默认 auto-rename、provider 支持 overwrite 时保留、仅支持 auto-rename 时降级，以及两者都不支持时不伪造降级的行为。
- 回归验证已通过：`go test ./internal/planner ./internal/provider ./internal/task`。
- 清理情况：本轮未启动额外后台进程；测试使用临时目录并自动清理，未保留 smoke 目录或构建产物。

## 2026-05-31 - planner 推荐模式与 provider overwrite 契约补强

- 继续按 [docs/01-GO_REBUILD_PLAN.md](E:/Workspace/VSCode/CloudPan-Sync-go/docs/01-GO_REBUILD_PLAN.md) 的 Phase 2 / Phase 3 主线推进，这轮不再停留在前端护栏，而是补核心 planner/provider 契约测试，直接收紧推荐模式、源删记录归根和 provider overwrite 语义。
- 新增 [internal/planner/recommendation_contract_test.go](E:/Workspace/VSCode/CloudPan-Sync-go/internal/planner/recommendation_contract_test.go)，把 `recommendedExecutionMode` / `recommendedExecutionModeReason` / `executionOrder` 的关键推荐分支，以及 deleted entry `rootPath` 回收规则固定成回归契约。
- 新增 [internal/provider/catalog_overwrite_contract_test.go](E:/Workspace/VSCode/CloudPan-Sync-go/internal/provider/catalog_overwrite_contract_test.go)，把 `aliyundrive_open / 123_open / 189cloud / 115_open` 的 `SupportsOverwrite`、`SupportsAutoRename`、`OverwriteBehavior` 和 `ConflictPolicies` 声明固定成 catalog 级契约。
- 这次改动不覆盖你当前 worktree 里已存在但无正文 diff 的 planner/provider 假脏文件，而是通过新增测试文件继续把主线里已落地的能力变成可验证基线。
- 回归验证已通过：`go test ./internal/planner ./internal/provider`。
- 清理情况：本轮未启动额外后台进程；未生成需清理的临时目录、smoke 目录或构建产物。

## 2026-05-31 - 报告历史与 smoke Markdown 下载静态契约补强

- 继续沿着 [docs/01-GO_REBUILD_PLAN.md](E:/Workspace/VSCode/CloudPan-Sync-go/docs/01-GO_REBUILD_PLAN.md) 中“状态矩阵/证据页展示”和“真实 provider smoke 记录”方向，把验收报告历史、下载文件名、smoke Markdown 查看/下载链路补成更完整的前端静态契约。
- `internal/app/web_test.go` 现在会额外校验 `selectedReportId`、报告历史 active 选中态、`selectedEvidenceReport()`、`cloudpan-sync-report` 默认下载文件名和空白替换规则仍然存在。
- 同一组契约也会兜住 `selectedProviderSmokeId`、`selectedProviderSmokeMarkdown`、`renderProviderSmokeMarkdown()`、`loadProviderSmokeMarkdown()`、`?format=markdown`、`Accept: text/plain`、smoke 记录 active 选中态和 `provider-smoke-markdown` 面板。
- 这次改动不改 API 或持久化逻辑，只把已落地的报告/Markdown 交互链路固定成回归保护，避免后续前端重构时丢失查看、选中和下载语义。
- 回归验证待本次提交前执行：`go test ./internal/app -run TestRoutesServeAppJSIncludesRetryEvidenceLabels -v`、`node --check web/static/app.js`。
- 清理情况：本轮未遗留额外后台进程、临时目录、smoke 目录或构建残留；测试使用 `httptest` 与 `t.TempDir()`，结束后已自动清理，未额外停止用户自有进程。

## 2026-05-31 - 源端删除策略与运行路径聚焦静态契约补强

- 继续沿着 [docs/01-GO_REBUILD_PLAN.md](E:/Workspace/VSCode/CloudPan-Sync-go/docs/01-GO_REBUILD_PLAN.md) 中“源端删除默认仅记录、不默认删除目标端”和“页面/运行证据应表达执行模式与扫描路径”方向，把这批已落地的前端展示语义补成可回归静态契约。
- `internal/app/web_test.go` 现在会额外校验 `plan-selected-roots`、`renderSourceDeletePolicy()`、`record_only（只记录，不删目标端）`、`Selected Roots`、`Scan Trace`、`renderRuntimePathChips()`、`focusRuntimeTreeByPath()`，以及任务/状态页运行路径聚焦提示语仍然存在。
- 同一组契约也会兜住 `data-runtime-focus-path / scope / kind` 和 `sourceDeletePolicy` 字段透传，避免后续前端重构时把运行路径聚焦和源端删除语义悄悄删掉。
- 浏览器级 smoke 这轮只保留稳定主线，并额外补上计划预览阶段 `record_only / 只记录，不删目标端` 的展示断言；更细的运行路径点击流仍在单独稳定中，避免把整包验证绑死在易抖的浏览器交互上。
- 回归验证已通过：`go test ./internal/app -run TestRoutesServeAppJSIncludesRetryEvidenceLabels -v`、`node --check web/static/app.js`。
- 额外验证情况：`go test ./internal/app -run TestConsoleUISmokeMainline -v` 在本轮长跑中未收敛，当前仍需后续单独稳定。
- 清理情况：本轮已主动清理测试拉起的 `go/chrome` 进程；未新增需保留的临时目录、smoke 目录或构建产物。

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
- 清理情况：本轮未遗留额外后台进程、临时目录、smoke 目录或构建残留；测试使用 `httptest` 与 `t.TempDir()`，结束后已自动清理，未额外停止用户自有进程。

## 2026-05-31 - 后台补传协议族预演走通

- 继续对齐 [docs/01-GO_REBUILD_PLAN.md](E:/Workspace/VSCode/CloudPan-Sync-go/docs/01-GO_REBUILD_PLAN.md) 中“更复杂的后台补传编排、批量筛选与多策略调度”方向，把协议族维度的后台补传动作也补齐 dry-run 预演链路。
- 状态页 summary 里的样本协议族现在新增 `data-auto-recover-preview-protocol-group` 按钮，用户可以先按协议族收敛候选并直接预演，而不必先执行真实后台补传。
- `wireAutoRecoverSummary()` 现已在协议族预演时同步写回 `protocolGroup` 过滤条件，并走 `triggerAutoRecover({ dryRun: true })`，和 lane 级预演保持一致的交互节奏。
- Web smoke 断言已补强：静态资源测试现在同时校验协议族预演按钮透传，以及 `button.dataset.autoRecoverPreviewProtocolGroup || ""` 这段前端请求构造代码仍然存在。
- 回归验证待本次提交前执行：`go test ./internal/app ./internal/task`、`node --check web/static/app.js`。
- 清理情况：本轮未遗留额外后台进程、临时目录、smoke 目录或构建残留；测试使用 `httptest` 与 `t.TempDir()`，结束后已自动清理，未额外停止用户自有进程。


## 2026-05-31 - 推荐模式与风险解释静态契约补强

- 继续对齐 [docs/01-GO_REBUILD_PLAN.md](E:/Workspace/VSCode/CloudPan-Sync-go/docs/01-GO_REBUILD_PLAN.md) 中“页面与 API 应给出推荐语义”要求，把前端已展示的推荐模式与风险解释字段补成 smoke 级静态契约。
- `internal/app/web_test.go` 现在会同时校验 `recommendedExecutionMode`、`recommendedExecutionModeReason`、`风险档位`、`风险节流`、`风险模板解释` 这些关键展示点仍然存在，避免后续前端回归时把推荐语义悄悄删掉。
- 这次改动不改业务逻辑，只把已经落地的推荐模式 / 风险解释展示进一步固定成可回归验证的契约。
- 回归验证待本次提交前执行：`go test ./internal/app`。
- 清理情况：本轮未遗留额外后台进程、临时目录、smoke 目录或构建残留；测试使用 `httptest` 与 `t.TempDir()`，结束后已自动清理，未额外停止用户自有进程。


## 2026-05-31 - 推荐执行模式 UI 闭环走通

- 继续对齐 [docs/01-GO_REBUILD_PLAN.md](E:/Workspace/VSCode/CloudPan-Sync-go/docs/01-GO_REBUILD_PLAN.md) 中“可选模式 + 默认推荐模式 + 推荐提示语义”要求，把推荐执行模式从静态展示补成可点击、可回填、可验证的 UI 闭环。
- `internal/app/ui_smoke_test.go` 现在会在计划预览后显式等待推荐标题和推荐原因出现，再点击 `采用推荐模式` 按钮，并校验 `#plan-execution-mode` 已被回填为推荐值 `leaf_first_lazy`。
- 这次改动不改 planner 推荐逻辑，而是把“用户真的能采用推荐模式继续创建任务”这条交互链路固定成回归测试。
- 回归验证待本次提交前执行：`go test ./internal/app`。
- 清理情况：本轮未遗留额外后台进程、临时目录、smoke 目录或构建残留；测试使用 `httptest` 与 `t.TempDir()`，结束后已自动清理，未额外停止用户自有进程。


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


## 2026-05-31 - 状态页刷新与保存反馈静态契约补强

- 继续沿着 [docs/01-GO_REBUILD_PLAN.md](E:/Workspace/VSCode/CloudPan-Sync-go/docs/01-GO_REBUILD_PLAN.md) 中“状态矩阵/证据页展示”方向，把状态页和证据区常用刷新/保存反馈语义补成前端静态契约。
- `internal/app/web_test.go` 现在会额外校验“验收报告已刷新”“Provider smoke 记录已保存”“状态矩阵已刷新”这些提示语仍然存在，和已有的任务刷新、报告保存、报告下载、smoke Markdown 查看/下载提示一起形成更完整的反馈链路约束。
- 这次改动不改刷新或持久化逻辑，只把现有用户反馈语义补成可回归契约，避免后续前端改动时把关键反馈文案静默删掉。
- 回归验证已通过：`go test ./internal/app -run TestRoutesServeAppJSIncludesRetryEvidenceLabels -v`、`go test ./internal/app -run TestConsoleUISmokeMainline -v`、`node --check web/static/app.js`。
- 清理情况：本轮验证未遗留额外后台进程、临时目录、smoke 目录或构建残留。


## 2026-05-31 - 后台补传按状态与主阻塞动作执行契约补强

- 继续对齐 [docs/01-GO_REBUILD_PLAN.md](E:/Workspace/VSCode/CloudPan-Sync-go/docs/01-GO_REBUILD_PLAN.md) 中“更复杂的后台补传编排、批量筛选与多策略调度”方向，把状态页 summary 里按执行状态、主失败类型、主阻塞动作直接执行的入口补成静态契约。
- `internal/app/web_test.go` 现在会额外校验 `data-auto-recover-run-state`、`data-auto-recover-run-retry-class`、`data-auto-recover-run-primary-blocked-action`、`data-auto-recover-run-blocked-action` 这些关键数据集仍然存在，保证“只执行等冷却/等授权/主重试类型/主阻塞动作”等 summary 动作不会被前端重构悄悄删掉。
- 这次改动不改后台补传调度逻辑，只把已经落地的直执行入口补成可回归约束，让 auto recover summary 的批量执行入口更稳。
- 回归验证已通过：`go test ./internal/app -run TestRoutesServeAppJSIncludesRetryEvidenceLabels -v`、`go test ./internal/app -run TestConsoleUISmokeMainline -v`、`node --check web/static/app.js`。
- 清理情况：本轮验证未遗留额外后台进程、临时目录、smoke 目录或构建残留。


## 2026-05-31 - 后台补传 lane 执行与样本任务直达契约补强

- 继续对齐 [docs/01-GO_REBUILD_PLAN.md](E:/Workspace/VSCode/CloudPan-Sync-go/docs/01-GO_REBUILD_PLAN.md) 中“更复杂的后台补传编排、批量筛选与多策略调度”方向，把 auto recover summary 中 lane 级执行和样本任务直达入口补成静态契约。
- `internal/app/web_test.go` 现在会额外校验 `data-auto-recover-run-lane-mode`、`data-auto-recover-open-task`，以及“执行主重试类型”“执行主阻塞动作”“执行该 lane”“打开样本任务”这些关键动作文案仍然存在，避免后续前端重构时把 summary 的直达入口悄悄删掉。
- 这次改动不改后台补传调度逻辑，只把已经落地的 lane 级执行与样本任务跳转入口补成可回归约束，让状态页 auto recover 看板的动作链更完整。
- 回归验证已通过：`go test ./internal/app -run TestRoutesServeAppJSIncludesRetryEvidenceLabels -v`、`go test ./internal/app -run TestConsoleUISmokeMainline -v`、`node --check web/static/app.js`。
- 清理情况：本轮验证未遗留额外后台进程、临时目录、smoke 目录或构建残留。


## 2026-05-31 - 后台补传模式协议族预算动作契约补强

- 继续对齐 [docs/01-GO_REBUILD_PLAN.md](E:/Workspace/VSCode/CloudPan-Sync-go/docs/01-GO_REBUILD_PLAN.md) 中“更复杂的后台补传编排、批量筛选与多策略调度”方向，把 auto recover summary 里按模式、协议族、建议预算和 lane 预演的动作入口补成静态契约。
- `internal/app/web_test.go` 现在会额外校验 `data-auto-recover-run-mode`、`data-auto-recover-run-protocol-group`、`data-auto-recover-focus-mode`、`data-auto-recover-focus-lane-mode`、`data-auto-recover-apply-budgets`、`data-auto-recover-preview-lane-mode` 这些关键数据集仍然存在，并兜住“执行该模式”“执行该协议族”“预演该协议族”“采用建议预算”“预演该 lane”等动作文案。
- 这次改动不改后台补传调度逻辑，只把已经落地的 summary 多入口补成更完整的可回归约束，让 auto recover 看板在模式/协议族/预算/预演维度的动作链更稳。
- 回归验证已通过：`go test ./internal/app -run TestRoutesServeAppJSIncludesRetryEvidenceLabels -v`、`go test ./internal/app -run TestConsoleUISmokeMainline -v`、`node --check web/static/app.js`。
- 清理情况：本轮验证未遗留额外后台进程、临时目录、smoke 目录或构建残留。


## 2026-05-31 - 后台补传决策明细动作契约补强

- 继续对齐 [docs/01-GO_REBUILD_PLAN.md](E:/Workspace/VSCode/CloudPan-Sync-go/docs/01-GO_REBUILD_PLAN.md) 中“更复杂的后台补传编排、批量筛选与多策略调度”方向，把最近一次后台补传结果明细里的聚焦、预算采用、预演和执行动作补成静态契约。
- `internal/app/web_test.go` 现在会额外校验 `data-auto-recover-decision-focus-state`、`data-auto-recover-decision-focus-lane-mode`、`data-auto-recover-decision-apply-budgets` 这些关键数据集仍然存在，并兜住“只看该状态”“只看该 lane”“采用建议预算”“预演该决策”“执行该决策”以及对应的提示语。
- 这次改动不改后台补传调度逻辑，只把已经落地的决策明细动作链补成可回归约束，让最近一次后台补传结果从查看到聚焦、预算回填、预演和执行都更稳。
- 回归验证已通过：`go test ./internal/app -run TestRoutesServeAppJSIncludesRetryEvidenceLabels -v`、`go test ./internal/app -run TestConsoleUISmokeMainline -v`、`node --check web/static/app.js`。
- 清理情况：本轮验证未遗留额外后台进程、临时目录、smoke 目录或构建残留。


## 2026-05-31 - 验收矩阵工具栏筛选契约补强

- 继续沿着 [docs/01-GO_REBUILD_PLAN.md](E:/Workspace/VSCode/CloudPan-Sync-go/docs/01-GO_REBUILD_PLAN.md) 中“真实 provider smoke 记录与协议组聚合矩阵”方向，把验收矩阵顶部工具栏筛选入口补成静态契约。
- `internal/app/web_test.go` 现在会额外校验 `renderProviderSmokeMatrixControls()`、`providerSmokeMatrixCounts()`、`data-provider-smoke-filter`，以及“已验收”“进行中”“待补齐”这些筛选文案仍然存在，避免后续前端重构时把矩阵工具栏筛选入口悄悄删掉。
- 这次改动不改验收矩阵聚合逻辑，只把已经落地的筛选工具栏补成可回归约束，让协议组验收矩阵的筛选入口更稳。
- 回归验证已通过：`go test ./internal/app -run TestRoutesServeAppJSIncludesRetryEvidenceLabels -v`、`go test ./internal/app -run TestConsoleUISmokeMainline -v`、`node --check web/static/app.js`。
- 清理情况：本轮验证未遗留额外后台进程、临时目录、smoke 目录或构建残留。


## 2026-05-31 - 补传等待态与样本协议族动作契约补强

- 继续对齐 [docs/01-GO_REBUILD_PLAN.md](E:/Workspace/VSCode/CloudPan-Sync-go/docs/01-GO_REBUILD_PLAN.md) 中“更复杂的后台补传编排、批量筛选与多策略调度”方向，把 auto recover summary 里等待态、样本协议族和阻塞动作相关入口补成静态契约。
- `internal/app/web_test.go` 现在会额外校验“执行该阻塞动作”“预演样本协议族”“执行样本协议族”“只执行可执行态”“只执行等时间窗”“只执行等补源文件”“只执行等人工确认”“只执行重试耗尽”“只执行其它等待”这些关键动作文案仍然存在。
- 这次改动不改后台补传调度逻辑，只把已经落地的等待态和样本协议族直达入口补成可回归约束，让 auto recover 看板在不同等待态和协议族维度的动作链更完整。
- 回归验证已通过：`go test ./internal/app -run TestRoutesServeAppJSIncludesRetryEvidenceLabels -v`、`go test ./internal/app -run TestConsoleUISmokeMainline -v`、`node --check web/static/app.js`。
- 清理情况：本轮验证未遗留额外后台进程、临时目录、smoke 目录或构建残留。

