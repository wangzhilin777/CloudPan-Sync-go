# CloudPan Sync Go 开发与上手说明

## 适用场景

- 适合第一次接手这个 Go 重构项目的人快速理解结构和运行方式。
- 适合把当前仓库单独发给协作者时，作为最小上手指南。

## 先看什么

- 项目目标和重构边界：`docs/01-GO_REBUILD_PLAN.md`
- 当前功能范围和进度：`docs/02-PROJECT_STATUS.md`
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

## 测试入口参考

- API 主工作流回归：
  - `internal/app/workflow_test.go`
- 首页与静态资源验证：
  - `internal/app/web_test.go`
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

## 最适合继续推进的任务

- 为一个协议族接入真实登录校验
- 为一个 provider 接入真实目录和元数据查询
- 为一个 provider 接入真实 fast upload / 普通上传链路
- 为控制台补浏览器级 smoke 自动化
- 为 README 增加更完整的示例和联调说明
