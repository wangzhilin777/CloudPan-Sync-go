# CloudPan-Sync-go

CloudPan Sync 多网盘互传控制台。

- 当前仓库 README 仅保留项目、开发和测试信息。
- 赞助商展示、收款码、打赏二维码和类似入口已从仓库说明中移除。

## 这是什么

- 这是一个多 provider 互传项目，不是“固定从某个来源传到某个目标”的定向工具。
- 当前主要提供：
  - 多 provider 之间的任务创建、运行、暂停、恢复与重试
  - 统一的授权档案、任务预览、运行证据和状态矩阵
  - 基于 SQLite 的本地持久化与浏览器控制台操作入口

## 主要功能

- 10 家目标 provider 的统一注册与能力声明
- 授权档案 CRUD 与校验
- 任务预览规划
- 任务创建、运行、暂停、恢复、重试
- 叶子目录优先与 `pre_scan_flat` 两种执行模式
- `create / overwrite / skip` 同步判定
- 源端删除仅记录、不默认删除目标端
- runtime evidence 与 provider 状态矩阵
- 后台自动补传与待补传树展示
- 控制台页面：登录、Provider / 授权、任务向导、任务列表详情、状态矩阵 / 证据

## 快速启动

```powershell
go run ./cmd/cloudpan-sync
```

- 默认地址：`http://127.0.0.1:8080`
- 控制台入口：`/`
- 默认管理员密码：`admin`

## Docker 启动

### 本地构建镜像

```powershell
docker build -t cloudpan-sync-go .
```

### 运行容器

```powershell
docker run --rm -p 8080:8080 -v ${PWD}/.cloudpan-sync-go:/data -e CLOUDPAN_ADMIN_PASSWORD=admin cloudpan-sync-go
```

- 容器默认监听 `8080`
- 默认数据目录是 `/data`
- 默认 SQLite 文件路径是 `/data/cloudpan-sync.db`
- 可通过环境变量覆盖：`CLOUDPAN_ADDR`、`CLOUDPAN_DATA_DIR`、`CLOUDPAN_DB_PATH`

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
  - 源端删除策略：当前显式任务级配置为 `sourceDeletePolicy=record_only`
    - 含义是“只记录源端删除事件，不默认删除目标端”
    - 当前首版仅支持这一种策略
- 当前首版对“已存在”的判断是保守口径：
  - provider `Metadata` 需要显式返回 `status=exists`
  - 或 `entry.exists=true`
  - 否则会按“目标不存在或元数据不可用”处理，避免占位 provider 误判全量跳过
- 当前控制台任务向导已经支持：
  - 选择 `executionMode`
  - 选择 `sourceDeletePolicy`
  - 选择 `riskMode`
  - 通过 JSON 覆盖 `requestIntervalMs / pageSize / directoryIntervalMs / cooldownSeconds / retryLimit / riskKeywords`
  - 查看推荐模式与推荐原因
  - 在任务详情、状态矩阵、最近结果、最近 probe 中查看执行模式证据
- 当前控制台也已经支持：
  - 查看运行检查点
  - 查看源端删除记录数量与样本
  - 查看 retry subset 的 `retryMode / retryScope / retrySelectedPaths`，并在最近结果、最近 probe、状态矩阵里回看这条证据链
  - 查看重试队列分类、attempt / retryLimit / remainingCount / eligibleAt
  - 查看目录状态清单
  - 查看当前根目录 / 当前目录 / 上次完成路径
  - 查看跳过数、失败数、风控命中数、最近风险状态
  - 查看待补传树与当前任务是否处于 `pending_only` 重试范围
  - 在预览、任务详情、状态矩阵和验收报告里看到“源端删除只记录、不默认删目标端”的证据
  - 在任务详情页和状态页按路径 / 状态 / reason 筛选目录树与待补传树
  - 在任务详情页和状态页按 retry class / retry state 筛选重试队列项
  - 从重试队列一键定位对应待补传树，或一键收敛到同类失败项
  - 从状态页 blocked 聚合摘要一键跳到样本任务，或一键收敛到当前阻塞动作
  - 从任务详情 blocked 引导卡片一键收敛当前任务的重试队列或待补传树
  - 对待补传树切到“仅叶子节点”视角，快速收敛当前最需要处理的末端目录/文件
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
  - 当前任务向导预填会同步带上 provider/profile/risk/selectedRoots/entries
  - 任务详情还支持一键“按当前任务重建向导”和“复制任务创建参数”
- 当前状态页已经可以直接看到后台自动补传的默认五级预算，以及当前手动放行时实际生效的 `group / provider / profile` 预算摘要，便于做协议族级排障和分批联调
- 当前 `POST /api/tasks/recover` 还支持 `dryRun=true` 预演；控制台可先预演当前筛选，或从候选 lane 一键套用建议预算后预演，再决定是否真正放行
- 预演和实际放行现在都会返回并展示 `decisions` 明细，能直接看出每个样本任务是“可放行 / 已放行 / 超出预算 / 等冷却 / 等时间窗 / 其它阻塞”的哪一种

## 相关文档

- 项目实施计划：`docs/01-GO_REBUILD_PLAN.md`
- 当前功能与进度：`docs/02-PROJECT_STATUS.md`
- 开发与上手说明：`docs/03-DEVELOPER_GUIDE.md`
- API 工作流示例：`docs/04-API_WORKFLOW_EXAMPLES.md`
- Provider 接入指南：`docs/05-PROVIDER_INTEGRATION_GUIDE.md`
- 真实联调记录模板：`docs/06-REAL_PROVIDER_SMOKE_TEMPLATE.md`
- 功能清单与完成度矩阵：`docs/07-FEATURE_MATRIX.md`

## GitHub 打包

- 仓库已补充 `.github/workflows/docker-package.yml`。
- 推送 `main` 上与 Docker / Go 构建相关的改动后，会自动执行一次 Docker 打包。
- 也可以在 GitHub Actions 页面手动触发 `docker-package`。
- 当前工作流会产出一个 Docker 镜像 tar 包 artifact：`cloudpan-sync-go-image`。

## 常用命令

```powershell
go test ./...
go build ./...
```

## 回归说明

- API 主工作流、runtime 场景、provider 契约测试已经在仓库内。
- 控制台主流程也已加入浏览器级 UI smoke 回归。
- UI smoke 依赖本机 Chrome 或 Edge；如需手工指定浏览器，可设置环境变量 `CHROMEDP_EXEC_PATH`。

