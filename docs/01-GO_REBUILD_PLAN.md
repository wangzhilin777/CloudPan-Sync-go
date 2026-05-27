# CloudPan Sync Go 重构实施计划

## Summary

- 本文档是 Go 重构版 CloudPan Sync 的唯一实施基线，用于后续开发、验收与里程碑推进。
- 项目从 Python 单体迁移到 Go，采用轻量前后端分离、SQLite、单机队列 Worker。
- Go 版不兼容 Python 旧 API，但保留核心业务语义：provider 能力模型、授权档案、规划器、任务执行、运行证据与状态聚合。
- Go 版除了 provider 统一抽象，也必须继承原始项目中重要的通用执行能力：叶子目录优先、增量/覆盖判定、补传执行、风控与频率策略。

## 重构边界

- 首版覆盖 10 家 provider：`guangya`、`aliyundrive_open`、`115_open`、`quark`、`189cloud`、`baidu_netdisk`、`uc`、`xunlei`、`pikpak`、`123_open`
- 首版保留的核心资源模型：
  - `Provider`
  - `AuthProfile`
  - `AuthValidation`
  - `TransferPlan`
  - `Task`
  - `TaskItem`
  - `TaskResult`
  - `ProviderProbe`
  - `ProviderStatus`
- 不迁移 Python 阶段的大量 `verify_*.py`、重型 markdown 导出链路和巨型单文件路由组织方式。

## 技术选型

- 语言：Go
- HTTP：标准库 `net/http`
- 持久化：SQLite
- 迁移：应用启动时执行内置 SQL migration
- 前端：轻量前后端分离，允许使用常见前端框架，但保持轻量
- 执行模型：单机队列 Worker

## 系统分层

- `cmd/cloudpan-sync`
  - 可执行入口
- `internal/app`
  - 配置、日志、HTTP 路由、统一响应、应用装配
- `internal/provider`
  - provider 接口、能力声明、provider registry
- `internal/auth`
  - 授权档案与认证验证领域模型
- `internal/planner`
  - 传输规划、策略判断、冲突策略
- `internal/task`
  - 任务、任务项、任务结果、状态机、Worker
- `internal/store`
  - SQLite 连接、migration、仓储
- `web`
  - 新版前端资源

## 核心数据模型

- `conflict_policy`
  - `overwrite_existing`
  - `auto_rename_new`
- `strategy`
  - `fast_upload`
  - `download_upload`
  - `pending_manual`
- `task_state`
  - `ready`
  - `running`
  - `paused`
  - `blocked`
  - `completed`
  - `completed_with_errors`
- `completion_kind`
  - `real_transfer`
  - `probe_only`
  - `candidate_only`
  - `live_failed`
- `risk_mode`
  - `safe`
  - `balanced`
  - `fast`
  - `custom`

## 执行模型要求

### 叶子目录优先

- Go 版必须保留原项目的“最底层目录优先”执行思想。
- 对大目录默认推荐：
  - 先向下扫描
  - 找到最底层目录
  - 立即执行该最底层目录
  - 目录之间按节流间隔继续推进
- 这样做的目的包括：
  - 降低风控压力
  - 控制内存占用
  - 让任务更容易中断恢复
  - 让目录级日志和证据更清晰

### 增量与覆盖判定

- Go 版必须内建明确的增量同步语义：
  - 新文件：上传
  - 已存在但大小 / mtime / 指纹变化：覆盖或按策略降级
  - 已同步且未变化：跳过
  - 源端已删除：默认记录，不默认删除目标端
- 这套判定应作为 planner + task runtime 的通用能力，而不是绑定单个 provider。

### 补传执行

- Go 版必须支持补传语义，而不仅是单次任务运行。
- 首版至少要具备：
  - 待补传项标记
  - 按目录层级聚合待补传项
  - 按叶子目录顺序执行补传
  - 为后续 UI 补传树预留数据结构

## 风控与频率策略要求

- Go 版必须提供通用、可配置的风控与频率策略层，而不是把节流常量散落在 provider 或 task 代码里。
- 首版固定支持：
  - `safe`
  - `balanced`
  - `fast`
  - `custom`
- 策略层至少要能承载：
  - 请求间隔
  - 页面大小
  - 目录间隔
  - 冷却时间
  - 重试次数
  - 风控关键词
- 同时支持：
  - provider 默认模板
  - 任务级覆盖
  - 后续 UI 配置入口

## API 设计

- `POST /api/session/login`
- `GET /api/providers`
- `GET /api/providers/{key}/capabilities`
- `GET /api/auth/profiles`
- `POST /api/auth/profiles`
- `PATCH /api/auth/profiles/{id}`
- `DELETE /api/auth/profiles/{id}`
- `POST /api/auth/profiles/{id}/validate`
- `POST /api/plans/preview`
- `GET /api/tasks`
- `POST /api/tasks`
- `GET /api/tasks/{id}`
- `POST /api/tasks/{id}/run`
- `POST /api/tasks/{id}/pause`
- `POST /api/tasks/{id}/resume`
- `POST /api/tasks/{id}/retry`
- `GET /api/evidence/runtime`
- `GET /api/status/providers`

## SQLite 表

- `auth_profiles`
- `auth_validations`
- `tasks`
- `task_items`
- `task_results`
- `provider_probes`
- `provider_status_snapshots`

## Provider 分组策略

- 协议族 1：`Aliyun + 123`
- 协议族 2：`Xunlei + PikPak`
- 协议族 3：`Quark + UC`
- 独立链路：
  - `Baidu`
  - `115`
  - `189Cloud`
  - `Guangya`

## 分阶段实施顺序

### Phase 1 - 项目骨架

- 初始化 Go module
- 建立目录结构
- 接入配置系统、日志、HTTP 服务、SQLite migration
- 建立统一错误模型、统一响应模型、provider registry

### Phase 2 - 核心内核

- 实现 `Provider` 接口与能力声明
- 实现 `AuthProfile / AuthValidation / TransferPlan / Task / TaskItem / TaskResult / ProviderStatus`
- 实现 auth profile CRUD、planner、task queue、runtime evidence 聚合、status matrix 聚合
- 建立执行模型内核：
  - 叶子目录优先
  - 增量 / 覆盖 / 跳过判定
  - 补传项建模
- 建立风控策略内核：
  - 风控档位
  - provider 默认模板
  - 任务级节流参数

### Phase 3 - Provider 落地

- 每家 provider 统一实现：
  - `ValidateAuth`
  - `List`
  - `Metadata`
  - `CreateDir`
  - `FastUploadCheck`
  - `Upload`

### Phase 4 - 前端

- 重做登录、Provider/授权、任务向导、任务列表详情、状态矩阵/证据页
- 前端只消费 Go 新 API

### Phase 5 - 联调与切换

- 用真实 auth profile 做最小联调
- 每个协议族至少打通一条真实成功样本
- Python 仓库转为参考资料来源

### Phase 6 - 执行模型与风控收口

- 把真实 provider 能力接入到统一执行模型
- 打通叶子目录优先、补传执行、目录间隔控制
- 打通按 provider 默认模板和任务自定义模板运行
- 让运行证据里能看见执行策略和风控命中情况

## 测试与验收标准

- 单元测试覆盖：
  - 指纹归一化
  - planner 策略与阈值判断
  - conflict policy 降级规则
  - deepest-first 顺序
  - 叶子目录执行顺序
  - 增量 / 覆盖 / 跳过判定
  - 补传项聚合与重试
  - 风控档位到具体节流参数的映射
  - auth profile 脱敏与持久化
  - task 状态机
- Provider 契约测试覆盖每家 provider 的核心接口
- Runtime 集成测试覆盖：
  - `fast_upload`
  - `hash miss -> binary fallback`
  - `overwrite downgrade`
  - `pending_manual`
  - `auth expired`
  - `rate limit`
  - `leaf-first execution`
  - `retry queue / pending relay`
  - `local file missing`
- UI smoke 覆盖：
  - 登录
  - 授权档案创建
  - 任务预览
  - 启动/暂停/恢复/重试
  - 状态矩阵与证据页展示

## 明确不做

- 不兼容 Python 旧 API 路由与旧页面逻辑
- 不迁移 Python 的海量 `verify_*.py`
- 首版不做 Postgres
- 首版不做分布式 Worker
- 首版不做复杂导出系统
