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
- 当前前后端基线已经支持这套模式语义，但 UI 里的完整配置入口和推荐提示还会继续补强。

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
