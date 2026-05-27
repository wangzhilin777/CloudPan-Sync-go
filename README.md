# CloudPan-Sync-go

CloudPan Sync 的 Go 重构版工作区。

## 这是什么

- 这是从 Python 单体迁移到 Go 的重构工程。
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

## 当前已完成的核心能力

- 10 家目标 provider 的统一注册与能力声明
- 授权档案 CRUD 与校验
- 任务预览规划
- 任务创建、运行、暂停、恢复、重试
- runtime evidence 与 provider 状态矩阵
- 控制台页面：
  - 登录
  - Provider / 授权
  - 任务向导
  - 任务列表详情
  - 状态矩阵 / 证据

## 当前仍在继续的部分

- provider 真实接口落地仍未全部完成
- UI smoke 异常场景和多 provider 样本还可继续补强
- 每个协议族至少一条真实成功样本仍待补齐

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
