# CloudPan-Sync-go

CloudPan Sync 的 Go 重构版工作区。

## 这是什么

- 这是从 Python 单体迁移到 Go 的重构工程。
- 这是一个多 provider 互传项目，不是“固定从某个来源传到某个目标”的定向工具。
- 当前仓库已经具备完整的主流程闭环：
  - Go HTTP 服务
  - SQLite 持久化
  - provider registry
  - auth / planner / task 核心链路
  - runtime evidence / provider status 聚合
  - 轻量控制台前端
- 当前更准确的定位是：
  - “可运行、可测试、可继续落地 provider 真实实现的重构基座”
  - 不是只停留在空骨架
  - 也还不是全部 provider 已真实联网打通的最终版

## 先看文档

- 重构实施计划：`docs/01-GO_REBUILD_PLAN.md`
- 当前功能与进度：`docs/02-PROJECT_STATUS.md`
- 开发与上手说明：`docs/03-DEVELOPER_GUIDE.md`
- API 工作流示例：`docs/04-API_WORKFLOW_EXAMPLES.md`
- Provider 接入指南：`docs/05-PROVIDER_INTEGRATION_GUIDE.md`
- 真实联调记录模板：`docs/06-REAL_PROVIDER_SMOKE_TEMPLATE.md`
- 功能清单与完成度矩阵：`docs/07-FEATURE_MATRIX.md`

## 推荐阅读顺序

如果这是第一次接手这个仓库，建议按下面顺序看：

1. `docs/02-PROJECT_STATUS.md`
2. `docs/07-FEATURE_MATRIX.md`
3. `docs/03-DEVELOPER_GUIDE.md`
4. `docs/04-API_WORKFLOW_EXAMPLES.md`
5. `docs/05-PROVIDER_INTEGRATION_GUIDE.md`
6. `docs/01-GO_REBUILD_PLAN.md`

## 当前已完成的核心能力

- 10 家目标 provider 的统一注册与能力声明
- 授权档案 CRUD 与校验
- 任务预览规划
- 任务创建、运行、暂停、恢复、重试
- 叶子目录优先的按需扫描执行骨架
- 目标端 metadata 预检查后的 `create / overwrite / skip` 判定闭环
- 任务级风控参数覆盖与风控命中证据
- runtime evidence 与 provider 状态矩阵
- 控制台页面：
  - 登录
  - Provider / 授权
  - 任务向导
  - 任务列表详情
  - 状态矩阵 / 证据
- Provider 辅助调试接口：
  - `POST /api/providers/{key}/list`
  - `POST /api/providers/{key}/metadata`
  - `POST /api/providers/{key}/create_dir`
  - `POST /api/providers/{key}/fast_check`

## 当前仍在继续的部分

- provider 真实接口落地仍未全部完成
- 叶子目录优先执行仍需继续补目录状态持久化、补传树和更完整的恢复能力
- UI smoke 异常场景和多 provider 样本还可继续补强
- 每个协议族至少一条真实成功样本仍待补齐

## 用一句话理解当前阶段

- 当前仓库已经可以独立启动、独立测试、独立演示主流程。
- 当前最主要的未完成项，不是“项目框架没搭完”，而是“各 provider 的真实联网能力还在逐家替换占位实现”。

## 执行模式说明

- 当前同步执行模式不是固定单一模式，而是已经正式支持任务级可选配置。
- 当前已支持两种执行模式：
  - `leaf_first_lazy`
  - `pre_scan_flat`
- `leaf_first_lazy` 是当前默认优先推荐模式，适合大目录、风控敏感 provider 和需要边扫边传的场景。
- `pre_scan_flat` 是可选模式，适合目录较小、希望先拿到完整扫描结果后再执行的场景。
- `leaf_first_lazy` 的真实含义不是“全量扫描后再按深度排序”，而是：
  - 先按顶层目录顺序逐棵子树推进
  - 每棵子树内部优先下探更深目录
  - 只扫描当前下一步真正要传的目录，不预扫整个目录树
- 当前 planner 预览结果、任务运行结果和 provider probe 证据里，都会返回：
  - 当前使用的 `executionMode`
  - 推荐模式 `recommendedExecutionMode`
  - 推荐原因 `recommendedExecutionModeReason`
- 当前 `leaf_first_lazy` 只是默认优先推荐，不是强制模式：
  - 用户仍可显式切换到 `pre_scan_flat`
  - API 也允许任务级显式传入 `executionMode`
- 当前同步判定语义已经明确为：
  - 目标端不存在：`create`
  - 目标端存在但指纹变化：`overwrite`
  - 目标端已存在且指纹一致：`skip`
- 当前首版对“已存在”的判断是保守口径：
  - provider `Metadata` 需要显式返回 `status=exists`
  - 或 `entry.exists=true`
  - 否则会按“目标不存在或元数据不可用”处理，避免占位 provider 误判全量跳过
- 当前控制台任务向导已经支持：
  - 选择 `executionMode`
  - 选择 `riskMode`
  - 通过 JSON 覆盖 `requestIntervalMs / pageSize / directoryIntervalMs / cooldownSeconds / retryLimit / riskKeywords`
  - 查看推荐模式与推荐原因
  - 在任务详情、状态矩阵、最近结果、最近 probe 中查看执行模式证据
- 当前控制台也已经支持：
  - 查看运行检查点
  - 查看目录状态清单
  - 查看当前根目录 / 当前目录 / 上次完成路径
  - 查看跳过数、失败数、风控命中数、最近风险状态
  - 查看待补传树与当前任务是否处于 `pending_only` 重试范围
- 当前还会继续补强的主要是：
  - 更细粒度的目录树交互展示
  - 真正异步 worker 下的运行中暂停
  - 源端删除记录与更完整的后台补传调度模型

## 待补传重试说明

- 当前 `pending_manual` 结果已经不只是展示出来。
- 当任务存在待补传项时，调用 `POST /api/tasks/{id}/retry` 会优先把任务缩小为“待补传子集”：
  - 仅保留待补传文件对应的 `entries`
  - 仅重建待补传文件对应的 `plan/items`
  - 清空旧结果后，以新的 `pending_only` 范围重新进入 `ready`
- 当前还会进一步按失败原因生成重试队列：
  - `pending_manual_requires_confirmation` -> `pending_only`
  - `rate_limited` -> 冷却后可重试
  - `auth_expired` -> 需要先刷新授权
  - `local_file_missing` -> 需要先补回本地文件
  - `retryLimit` 耗尽 -> 进入明确阻断，不再继续自动/手动重试
- 这意味着当前已经接通：
  - 待补传项聚合
  - 待补传树展示
  - 待补传子集重试执行
  - 基于失败原因的重试队列分类
  - `rate_limited` 的冷却阻断
  - `blocked` 运行态与 `blockedReason / nextRetryAt` 证据
  - 单机 tick 版后台自动补传调度
  - `retryLimit` 的累计次数、剩余次数与 exhausted 阻断
  - `blockedAction / blockedAdvice` 的统一处理建议
  - 状态矩阵中的 `blockedTasks / blockedActions` 聚合摘要
  - 任务详情中的“下一步处理”引导区与直达入口
  - 直达入口会自动定位当前授权档案，或按当前任务预填任务向导参数
- 当前还没有接通的是：
  - 更复杂的补传批量选择与筛选
  - 更精细的后台补传策略编排

## 快速启动

```powershell
go run ./cmd/cloudpan-sync
```

- 默认地址：`http://127.0.0.1:8080`
- 控制台入口：`/`

## 常用命令

```powershell
go test ./...
go build ./...
```

## 回归说明

- API 主工作流、runtime 场景、provider 契约测试已经在仓库内。
- 控制台主流程也已加入浏览器级 UI smoke 回归。
- UI smoke 依赖本机 Chrome 或 Edge；如需手工指定浏览器，可设置环境变量 `CHROMEDP_EXEC_PATH`。
