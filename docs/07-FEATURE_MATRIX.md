# CloudPan Sync Go 功能清单与完成度矩阵

## 文档目的

- 这份文档专门回答两个问题：
  - 这个项目当前到底能做什么
  - 这个项目还有哪些功能没做完
- 如果仓库要单独分享给别人，建议把本文档当作第一份“功能地图”。

## 当前阶段结论

- 当前仓库已经不是空骨架。
- 当前已经具备完整的后端主链路、SQLite 持久化、前端控制台、API 工作流和测试回归基础。
- 当前项目语义是“多 provider 互传”，不是“固定定向同步到单个目标端”。
- 当前最主要的缺口集中在三块：
  - provider 真实联网实现
  - 同步执行模型
  - 风控与频率策略

## 功能矩阵

| 模块 | 当前可用内容 | 已完成状态 | 进行中 | 待完成 |
| --- | --- | --- | --- | --- |
| 项目骨架 | Go 服务、配置、日志、路由、静态资源、SQLite migration | 已完成 | 无 | 无 |
| 统一 API | 登录、provider 列表、能力查询、auth profile、plans、tasks、evidence、status | 已完成 | 无 | 后续仅按真实联调补细节 |
| Provider 注册表 | 10 家 provider 已注册，能力声明统一 | 已完成 | 无 | 后续逐家替换真实实现 |
| Provider 调试接口 | `list`、`metadata`、`create_dir`、`fast_check` 已暴露为 API | 已完成 | 联调口径补强 | 后续可增加上传调试入口 |
| Auth Profile | CRUD、校验、脱敏持久化、按 provider 映射 | 已完成 | 真实 provider 字段口径补充 | 更多真实鉴权模式 |
| Planner | 预览计划、策略判定、阈值判断、冲突降级、deepest-first | 已完成 | 将升级为通用执行模型入口 | 结合真实 provider 与增量规则继续校准 |
| 同步执行模型 | 已有 deepest-first 基础语义、fallback、运行时结果写库、按需扫描骨架 | 部分完成 | `leaf-first lazy scan` 已接入任务运行链路 | 增量 / 覆盖 / 跳过闭环、补传树执行、目录状态持久化 |
| 风控与频率策略 | 已有 `rate_limited` 场景测试和少量行为语义 | 部分完成 | 风控档位与模板设计待落地 | `safe / balanced / fast / custom`、provider 默认模板、任务级节流参数 |
| Task Runtime | 创建、查询、运行、暂停、恢复、重试、结果落库 | 已完成 | 后续将挂接执行模型和风控策略 | 真实上传链路接入后补更细运行态 |
| Runtime Evidence | 最近结果、最近探针、状态快照、状态矩阵 API | 已完成 | 无 | 真实联调样本沉淀 |
| 控制台前端 | 登录、授权、任务向导、任务列表详情、状态矩阵/证据 | 已完成 | 异常场景提示可继续增强 | 更产品化视觉与更细交互 |
| 单元测试 | auth、planner、task、provider、workflow、web、UI smoke | 已完成 | 持续补样本 | 真实 provider 契约覆盖继续加深 |
| Provider 真实实现 | `aliyundrive_open` 已接入真实 `ValidateAuth` 骨架 | 部分完成 | 目录链路 `List/Metadata/CreateDir` 正在推进 | 其余 provider 真实实现、上传链路、异常恢复 |
| 真实联调验收 | 模板、流程、文档已具备 | 部分完成 | 首批真实样本沉淀中 | 每个协议族至少一条真实成功样本 |

## 当前已经能直接演示的能力

### 1. 启动一个完整服务

- 可以直接运行：
  - `go run ./cmd/cloudpan-sync`
- 默认打开：
  - `http://127.0.0.1:8080`

### 2. 走完整主流程

- 当前可以完整演示：
  - 登录
  - 查看 provider 列表
  - 创建授权档案
  - 校验授权档案
  - 预览传输计划
  - 创建任务
  - 运行 / 暂停 / 恢复 / 重试任务
  - 查看运行证据与 provider 状态矩阵

### 2.1 可选但优先推荐的执行模式

- 当前不是只有一种执行模式。
- `leaf-first lazy scan` 是：
  - 可选模式
  - 当前默认优先推荐模式
- 它适合：
  - 多顶层目录
  - 大目录
  - 风控敏感 provider
  - 不想一开始就把全量目录和文件元信息全部扫出来的场景

### 3. 直接调试 provider 能力

- 当前已经有下面这些辅助接口，适合开发和联调：
  - `POST /api/providers/{key}/list`
  - `POST /api/providers/{key}/metadata`
  - `POST /api/providers/{key}/create_dir`
  - `POST /api/providers/{key}/fast_check`

## 10 家 Provider 当前状态

| Provider | 协议族 | 当前状态 | 说明 |
| --- | --- | --- | --- |
| `aliyundrive_open` | `aliyun_123_open` | 部分真实 | 已接入真实 `ValidateAuth` 骨架，目录链路正在推进 |
| `123_open` | `aliyun_123_open` | 占位可运行 | 统一接口已接通，真实实现待补 |
| `xunlei` | `xunlei_pikpak` | 占位可运行 | 适合当前内核联调 |
| `pikpak` | `xunlei_pikpak` | 占位可运行 | 适合当前内核联调 |
| `quark` | `quark_uc` | 占位可运行 | 适合当前内核联调 |
| `uc` | `quark_uc` | 占位可运行 | 适合当前内核联调 |
| `baidu_netdisk` | `baidu_netdisk` | 占位可运行 | 适合当前内核联调 |
| `115_open` | `115_open` | 占位可运行 | 适合当前内核联调 |
| `189cloud` | `189cloud` | 占位可运行 | 适合当前内核联调 |
| `guangya` | `guangya` | 占位可运行 | 目前主要承担源端样本角色 |

## 当前已完成的里程碑视角

- 已完成项目骨架和基础文档
- 已完成统一资源模型和 API 主链路
- 已完成 auth / planner / task / evidence 内核
- 已完成控制台前端和 UI smoke
- 已完成 10 家 provider 的统一注册与占位实现
- 已开始进入“逐家 provider 真实落地 + 执行模型补齐 + 风控策略补齐”阶段

## 当前最值得继续做的三件事

1. 把执行模式补成“可选配置 + 默认推荐 + UI/提示联动”
2. 把执行模型继续补成完整的增量 / 补传闭环
3. 把风控策略做成独立可配置层

## 不要误解的地方

- “支持 10 家 provider” 当前表示：
  - 统一抽象、统一注册、统一 API、统一测试入口已经具备
  - 不表示 10 家都已经真实联网打通
- “有 deepest-first 测试” 当前表示：
  - 已有顺序语义基础
  - 不表示全部执行模式都已经完整落地
- “有 leaf-first lazy scan” 当前表示：
  - 已经支持按顶层目录顺序逐棵子树推进，并按需列目录
  - 不表示目录状态持久化、补传树、完整 UI 提示已经全部完成
- “有 rate_limited 场景” 当前表示：
  - 已有测试语义和占位运行结果
  - 不表示风控策略层已经做成可配置能力
- “可以运行任务” 当前表示：
  - 任务内核、状态机、结果写库、证据聚合都已经可用
  - 不表示所有 provider 上传都已经是真实外部平台传输
- “有前端控制台” 当前表示：
  - 当前已经适合开发联调、演示和 smoke
  - 不表示已经达到最终产品化 UI 形态
