# CloudPan Sync Go 开发与上手说明

## 适用场景

- 适合第一次接手这个 Go 重构项目的人快速理解结构和运行方式。
- 适合把当前仓库单独发给协作者时，作为最小上手指南。

## 先看什么

- 项目目标和重构边界：`docs/01-GO_REBUILD_PLAN.md`
- 当前功能范围和进度：`docs/02-PROJECT_STATUS.md`
- 功能清单与完成度矩阵：`docs/07-FEATURE_MATRIX.md`
- API 工作流示例：`docs/04-API_WORKFLOW_EXAMPLES.md`
- Provider 接入指南：`docs/05-PROVIDER_INTEGRATION_GUIDE.md`
- 真实联调记录模板：`docs/06-REAL_PROVIDER_SMOKE_TEMPLATE.md`
- 前端资源说明：`web/README.md`

## 当前目录说明

- `cmd/cloudpan-sync`
  - 服务启动入口
- `internal/app`
  - 应用装配、路由、统一响应、配置、错误处理
- `internal/provider`
  - provider 元数据、能力模型、adapter 接口、10 家 provider 注册与协议族适配器
- `internal/auth`
  - 授权档案 CRUD、校验、持久化
- `internal/planner`
  - 任务预览与策略规划
- `internal/task`
  - 任务创建、状态流转、运行时结果、证据聚合
- `internal/store/sqlite`
  - SQLite 存储与 migration
- `web`
  - 控制台前端静态资源和嵌入逻辑
- `docs`
  - 计划文档、状态文档、上手文档

## 当前可以直接做的事

在当前仓库里，可以把工作拆成三类：

- 直接启动和演示现有主流程
- 直接调用 API 或 provider 调试接口做联调
- 在现有内核上继续补真实 provider 能力

## 当前任务执行模式怎么理解

- 当前项目是多 provider 互传，不是定向同步到单个固定目标。
- 执行模式已经作为任务级可选配置进入当前 API。
- 当前已支持：
  - `leaf_first_lazy`
  - `pre_scan_flat`
- 当前默认优先推荐的是：
  - `leaf_first_lazy`
- 这个模式的含义是：
  - 按顶层目录顺序逐棵子树推进
  - 每棵子树内部优先下探最深目录
  - 只扫描下一步真正需要传的目录，不预先拉完整目录树
- 这个模式的推荐语义是：
  - 默认优先推荐
  - 但不是强制
  - 用户和 API 都可以显式改成 `pre_scan_flat`
- `pre_scan_flat` 的含义是：
  - 先把当前选择目录下的文件项按遍历顺序收集出来
  - 再按收集顺序执行
  - 适合目录较小、希望先看到完整分析结果的场景
- 这适合：
  - 大目录
  - 风控敏感 provider
  - 需要边扫边传、边停边恢复的场景
- 当前 planner / task 结果里还会直接带出：
  - `executionMode`
  - `recommendedExecutionMode`
  - `recommendedExecutionModeReason`
  - `syncDecision`
  - `executionOrder`
  - `runtime`
  - `currentRoot`
  - `currentDirectory`
  - `lastCompletedPath`
  - `pendingCount`
  - `pendingTree`

## 当前增量 / 覆盖 / 跳过规则怎么理解

- 当前任务在真正上传前，会先对目标端做一次 `Metadata` 预检查。
- 当前运行时会得出三种判定：
  - `create`
    - 目标端还没有这个文件，或当前无法确认其存在
  - `overwrite`
    - 目标端已有同路径文件，但大小或指纹已变化
  - `skip`
    - 目标端已有同路径文件，且和源端指纹一致
- 当前首版采用保守存在判定：
  - `MetadataResult.Status == "exists"`
  - 或 `MetadataResult.Entry.exists == true`
- 如果 provider 只返回普通 `ok`，不会直接认定为“已存在”。
- 这是为了兼容当前仓库里仍存在的占位 provider，避免把所有文件误判成“已同步”。
- 当前默认同步语义仍然是：
  - 新增
  - 覆盖
  - 跳过未变化
  - 源端删除只记录，不默认删除目标端

## 当前待补传树怎么理解

- 当运行结果命中 `pending_manual` 一类场景时，当前不会把它伪装成普通失败后直接丢失上下文。
- 当前 runtime 会把这类结果聚合成 `pendingTree`：
  - 先按顶层 root 聚合
  - 再按子目录层级展开
  - 叶子文件节点会带上 `reason` 和 `providerStatus`
- 当前控制台的任务详情页和状态页已经能直接看到这棵树。
- 当前这两个页面还支持前端侧筛选：
  - 按路径 / 文件名快速收敛命中节点
  - 按 reason / provider status 收敛待补传原因
  - 对待补传树切到“仅叶子节点”视角，优先查看当前最末端待处理节点
- 当前这棵树的定位是：
  - 让协作者知道“哪些目录、哪些文件还待补传”
  - 为后续真正的补传队列 / 补传调度提供结构基础
- 当前 `POST /api/tasks/{id}/retry` 已支持：
  - 发现待补传项后优先缩小到待补传子集
  - 清空旧结果后只重跑这些待补传文件
  - 也支持显式传 `paths + scope`，把当前任务重建成“用户选定路径子集”的 retry 范围
  - 当前任务页里的“重试当前筛选”已经接到这条能力：
    - 可以按当前待补传树筛选结果重建 `selected_pending_subset`
    - 也可以按当前重试队列筛选结果重建 `selected_retry_subset`
- 当前 runtime 还会维护 `retryQueue`：
  - `pending_manual_requires_confirmation`
  - `rate_limited`
  - `auth_expired`
  - `local_file_missing`
- 当前每个 retry queue item 还会带上：
  - `attemptCount`
  - `retryLimit`
  - `remainingCount`
  - `exhausted`
- 当前控制台的任务详情页和状态页已经可以直接查看这些队列项，并支持：
  - 按路径 / reason / provider status 收敛命中项
  - 按 retry class 过滤
  - 按 `retryable / blocked / exhausted` 过滤
  - 从某个 retry queue item 直接定位对应待补传树
  - 从某个 retry queue item 直接收敛到同类失败项
- 当前 `rate_limited` 项会生成 `eligibleAt`，冷却未到时 `retry` 会直接拒绝过早重试
- 当前如果任务失败后只剩“冷却等待 / 人工确认 / 授权失效 / 本地文件缺失”这类项，任务会进入 `blocked`
- 当前 runtime 会补充：
  - `blockedReason`
  - `blockedAction`
  - `blockedAdvice`
  - `nextRetryAt`
- 当前 evidence / status 聚合还会补充：
  - `blockedTasks`
  - `blockedActions`
  - provider 维度的 `blockedCount`
  - `protocolCoverage`
- 当前任务详情页还会根据 `blockedAction` 生成“下一步处理”引导区，并提供跳转到授权面板、任务向导或状态矩阵的直达入口
- 当前这些任务详情引导按钮还支持：
  - 直接把当前任务的 retry queue 收敛到对应 blocked action
  - 直接把当前任务的待补传树收敛到对应 root/path
- 当前任务详情里的目录树 / 待补传树节点也支持更细动作：
  - 目录树节点可直接“按当前路径重建向导”
  - 待补传树节点可直接“重试当前路径”
  - 适合先把任务缩回某个子树，再决定继续跑还是改参数重建
- 当前状态页的 `blockedActions` 聚合摘要也已经支持：
  - 一键打开这一类阻塞动作的样本任务
  - 一键把最近重试队列收敛到当前阻塞动作
- 当前状态页还会直接展示：
  - 协议族覆盖矩阵
  - 每个协议族的真实成功样本状态
- 当前目录树和待补传树支持按 root 分组收起 / 展开，方便在大任务里快速收敛视线
- 当前目录树分组收起状态会保存在浏览器本地，刷新后仍会保留
- 当前这些直达入口已支持：
  - 自动定位当前授权档案
  - 按当前任务预填任务向导里的 provider/profile/risk/entries 参数
  - 同时回填 `selectedRoots / thresholdMB / conflictPolicy`
- 当前任务详情区还支持两个快捷动作：
  - 一键按当前任务重建向导
  - 一键复制当前任务创建参数 JSON，便于重建相似任务或做接口联调
- 当前如果 `retryLimit` 已耗尽：
  - queue item 会标记为 `exhausted`
  - 手动 `retry` 会返回 `retry_blocked`
  - 自动补传调度不会继续接管这类任务
- 当前应用内已接入单机 tick 版自动补传调度：
  - 启动时会先扫描一次候选任务
  - 后续按 tick 持续检查后台补传候选池
  - 默认会受 `CLOUDPAN_AUTO_RETRY_BATCH_LIMIT` 控制，避免单次 tick 把全部候选一次性打满
  - 自动 tick 现在还会默认附带 `CLOUDPAN_AUTO_RETRY_LIMIT_PER_MODE / _PER_LANE / _PER_PROTOCOL_GROUP / _PER_PROVIDER / _PER_PROFILE` 五级公平预算，避免单一模式、单一协议族、单一 provider 或单一账号长期霸占整轮
  - 当前调度优先级固定为：
    - `upload_checkpoint_auto_resume`
    - `retry_queue_auto_retry`
    - `cooldown_elapsed_auto_retry`
- 当前控制台任务详情页也会直接显示：
  - 当前是否处于 `pending_only` 重试范围
  - 当前重试队列里的 `retryable / blocked` 计数
  - 当前这次 retry 是不是来自用户手动选定的路径子集
  - 当前 `retryScope / retrySelectedPaths`
- 当前 `metadata.retrySummary` 还会补充：
  - `retryableNowCount / cooldownCount / pendingManualCount / authExpiredCount / localMissingCount / exhaustedCount`
  - `uploadCheckpointEligible`
  - `autoRecoverEligible / autoRecoverMode / autoRecoverAdvice`
- 当前任务详情、运行检查点和状态页快照会直接把它们展示成：
  - `后台补传候选`
  - `队列拆分`
  - `自动补传提示`
- 当前状态页的运行证据摘要还新增了：
  - `自动补传候选池`
  - `自动补传默认调度策略摘要`
  - 会按模式聚合 `taskCount / providerCount / queueItemCount / retryableNowCount / cooldownCount / uploadCheckpointEligible`
- 当前状态页还支持直接手动触发后台补传：
  - 可按 `mode`
  - 可按 `providerKey`
  - 可按本轮 `limit`
  - 适合在联调时先只放行一类候选，而不是等下一次 tick
- 这几个字段主要用来判断：
  - 当前队列是不是“冷却到期后自动重试”
  - 是不是“upload checkpoint 自动续跑”
  - 还是“仍需人工确认 / 刷新授权 / 补回本地文件”
- 当前还没有做到：
  - 在 UI 里对待补传树做更复杂的筛选和批量操作
  - 更细粒度的账号级 / 时间窗级后台补传策略编排

## 当前风控参数怎么理解

- 当前 `riskMode` 仍然保留：
  - `safe`
  - `balanced`
  - `fast`
  - `custom`
- 但现在不只是“选一个档位名”。
- 当前任务还可以额外传 `riskOverride`，覆盖：
  - `requestIntervalMs`
  - `pageSize`
  - `directoryIntervalMs`
  - `cooldownSeconds`
  - `retryLimit`
  - `maxConcurrent`
  - `autoRetryStartHour`
  - `autoRetryEndHour`
  - `riskKeywords`
- 当前 planner 会把最终生效的风险配置写进 `metadata.riskProfile`。
- 当前 planner 还会把风险解释链写进 `metadata.riskProfileResolution`：
  - `base`
  - `calibrated`
  - `applied`
  - `calibrationReasons`
  - `overrideFields`
- 默认 `riskProfile` 会先按 `safe / balanced / fast` 取基线，再按目标 provider 做保守度校准；`custom` 模式则保留为调用方完全手动控制。
- 当前控制台的预览区和任务详情区也会直接展示这条“风险模板解释”，便于看清楚到底是 provider 校准还是任务 override 改了最终节流值。
- 当前 runtime 如果命中风险关键词，还会在结果和运行时证据里写回：
  - `riskHit`
  - `riskHitCount`
  - `lastRiskStatus`
- 当前 runtime 还会按 `requestIntervalMs / directoryIntervalMs` 执行基础节流，并在被节流的 result payload 写入 `throttle` 证据。
- 当前单机自动补传调度还会尊重 `autoRetryStartHour / autoRetryEndHour`：
  - 不在时间窗内时，任务仍可手动 Retry
  - 但后台 worker 不会自动接管

### 启动服务

```powershell
go run ./cmd/cloudpan-sync
```

- 默认监听：`http://127.0.0.1:8080`
- 根路径 `/` 可打开控制台页面

### 跑测试

```powershell
go test ./...
```

### 构建

```powershell
go build ./...
```

## 当前主流程

1. 登录控制台
2. 查看 provider 列表和能力
3. 创建授权档案
4. 校验授权档案
5. 预览传输计划
6. 创建任务
7. 运行 / 暂停 / 恢复 / 重试任务
8. 查看任务结果、运行证据、provider 状态矩阵

如果任务走按需扫描模式，还需要注意：

- 创建任务时可以只给 `selectedRoots`
- 运行任务时需要 `sourceProfileId`
- 运行时才会按当前需要逐段列目录
- 当前任务 payload 已持久化：
  - 目录状态
  - 已处理数量
  - 已跳过数量
  - 当前根目录 / 当前目录
  - 上次完成路径
- 任务在运行中可以协作式暂停：当前 item 完成后会检查暂停请求，再落盘为 `paused`，恢复后会继续后续 item
- 如果任务已经带有部分结果，再次运行时会从未完成项继续，而不是整任务从头重跑
- 当前控制台中可以直接看到：
  - 运行检查点
  - 目录状态树
  - 待补传树
  - 真实样本矩阵
  - 真实联调验收判定
  - 真实联调验收缺失原因提示
  - 真实联调验收补齐建议
  - 首页证据摘要里的验收计数
  - 当前根目录 / 当前目录 / 上次完成路径

## 当前 API 范围

- `POST /api/session/login`
- `GET /api/providers`
- `GET /api/providers/{key}/capabilities`
- `GET/POST/PATCH/DELETE /api/auth/profiles`
- `POST /api/auth/profiles/{id}/validate`
- `POST /api/plans/preview`
- `GET/POST /api/tasks`
- `GET /api/tasks/{id}`
- `POST /api/tasks/{id}/run`
- `POST /api/tasks/{id}/pause`
- `POST /api/tasks/{id}/resume`
- `POST /api/tasks/{id}/retry`
- `GET /api/evidence/runtime`
- `GET /api/status/providers`

创建任务时当前新增一个重要字段：

- `sourceProfileId`
  - 当任务要走按需扫描时，需要它来在运行阶段对 source provider 执行 `List`
- `riskMode`
  - 当前控制台任务向导已支持直接选择
- `riskOverride`
  - 当前控制台任务向导已支持通过 JSON 输入覆盖节流参数和风险关键词
- `executionMode`
  - 不传时默认按 `leaf_first_lazy`
  - 传 `pre_scan_flat` 时会切换到预扫描平铺模式
  - 当前控制台任务向导也已支持直接选择，并会展示推荐模式与推荐原因

## 当前额外可用的 Provider 调试接口

这些接口不属于普通业务主流程，但很适合开发阶段直接验证 provider 适配器：

- `POST /api/providers/{key}/list`
- `POST /api/providers/{key}/metadata`
- `POST /api/providers/{key}/create_dir`
- `POST /api/providers/{key}/fast_check`

建议直接参考：`docs/04-API_WORKFLOW_EXAMPLES.md`

## 测试入口参考

- API 主工作流回归：
  - `internal/app/workflow_test.go`
- 首页与静态资源验证：
  - `internal/app/web_test.go`
- 控制台 UI smoke：
  - `internal/app/ui_smoke_test.go`
- planner 策略测试：
  - `internal/planner/service_test.go`
- auth 档案测试：
  - `internal/auth/service_test.go`
- runtime 场景覆盖：
  - `internal/task/service_test.go`
- provider 契约级测试：
  - `internal/provider/*_test.go`

## 接手开发时的注意点

- 当前很多 provider 已经“接了接口”，但并不代表已经接了真实平台能力。
- 如果要继续落地 provider，不建议推翻现有内核，而是沿着现有抽象往里替换真实实现。
- 当前最应该保留的核心设计：
  - provider registry
  - auth profile 持久化模型
  - planner 统一决策入口
  - task runtime + evidence 聚合
  - 控制台只消费 Go API 的边界

## 如果你想快速判断现在做到哪里

- 看“宏观完成度”：
  - `docs/02-PROJECT_STATUS.md`
- 看“逐模块功能和未完成项”：
  - `docs/07-FEATURE_MATRIX.md`
- 看“怎么自己跑一遍”：
  - 本文档 + `docs/04-API_WORKFLOW_EXAMPLES.md`

## UI smoke 说明

- 浏览器级主流程 smoke 已纳入 `go test ./...`。
- 测试依赖本机存在 Chrome 或 Edge，可通过环境变量 `CHROMEDP_EXEC_PATH` 指定浏览器路径。
- 当前覆盖的主流程：
  - 登录
  - 授权档案创建
  - 授权档案校验
  - 任务预览
  - 任务创建
  - 启动 / 暂停 / 恢复 / 重试
  - 状态矩阵与运行证据展示

## 最适合继续推进的任务

- 为一个协议族接入真实登录校验
- 为一个 provider 接入真实目录和元数据查询
- 为一个 provider 接入真实 fast upload / 普通上传链路
- 补执行模式的目录状态持久化、补传树和 UI 模式提示
- 补协议族覆盖矩阵的真实样本沉淀与验收
- 补更细粒度的目录树节点级交互
- 为控制台扩展异常场景和多 provider 的 UI smoke
- 为 README 增加更完整的示例和联调说明
