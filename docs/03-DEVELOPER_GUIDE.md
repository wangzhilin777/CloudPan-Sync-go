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
  - `executionOrder`

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
- 为控制台扩展异常场景和多 provider 的 UI smoke
- 为 README 增加更完整的示例和联调说明
