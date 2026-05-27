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
| Planner | 预览计划、策略判定、阈值判断、冲突降级、执行模式推荐、风控档位元数据 | 已完成 | 结合真实 provider 与增量规则继续校准 | 更细推荐规则与真实联调校准 |
| 同步执行模型 | 已支持 `leaf_first_lazy`、`pre_scan_flat`、按需扫描骨架、fallback、目录状态持久化、断点继续当前子树、目标端 metadata 预检查、`create / overwrite / skip` 判定闭环、待补传树聚合、待补传子集重试、失败重试队列分类、`blocked` 运行态、最小自动补传调度、`retryLimit` 次数耗尽阻断 | 部分完成 | 任务级执行模式、runtime checkpoint、待补传树与目录树展示已接入 planner / task / evidence / UI，`retry` 已支持 pending-only 重建，并识别 rate-limit cooldown、retry limit 与自动恢复 | 更完整目录树交互、更细后台补传策略 |
| 风控与频率策略 | 已支持 `safe / balanced / fast / custom` 基线、默认风险模板、任务级 `riskOverride`、风控命中证据 | 部分完成 | 已接入 planner / task metadata / runtime evidence / UI | 真实 provider 校准、更易用的表单化配置 |
| Task Runtime | 创建、查询、运行、暂停、恢复、重试、结果落库 | 已完成 | 后续将挂接执行模型和风控策略 | 真实上传链路接入后补更细运行态 |
| Runtime Evidence | 最近结果、最近探针、状态快照、状态矩阵 API | 已完成 | 无 | 真实联调样本沉淀 |
| 控制台前端 | 登录、授权、任务向导、任务列表详情、状态矩阵/证据、执行模式可视化、目录状态展示、目录树/待补传树筛选与叶子视角 | 已完成 | 异常场景提示可继续增强 | 更产品化视觉与更细交互 |
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
- 当前已支持：
  - `leaf_first_lazy`
  - `pre_scan_flat`
- `leaf_first_lazy` 是：
  - 可选模式
  - 当前默认优先推荐模式
- 当前 planner / runtime 会同步返回：
  - 当前执行模式
  - 推荐模式
  - 推荐原因
- 当前 `leaf_first_lazy` 的定位是：
  - 可选配置
  - 默认优先推荐
  - 不是强制模式
- 它适合：
  - 多顶层目录
  - 大目录
  - 风控敏感 provider
  - 不想一开始就把全量目录和文件元信息全部扫出来的场景
- `pre_scan_flat` 适合：
  - 目录较小
  - 需要先拿到完整扫描结果
  - 更重视预扫描可见性的联调场景

### 2.2 当前同步判定语义

- 当前 runtime 在上传前会先看目标端 `Metadata`。
- 当前会输出三种判定：
  - `create`
  - `overwrite`
  - `skip`
- 当前 `skip` 不是单纯按文件名判定，而是要求：
  - 目标端明确存在
  - 且大小一致
  - 且 `md5 / sha1 / gcid` 至少一个有效指纹一致
- 当前为了避免占位 provider 误判，默认只把下面两种情况当成“目标明确存在”：
  - `status == exists`
  - `entry.exists == true`

### 2.3 当前风控配置语义

- 当前风控不再只有档位名。
- 当前已支持：
  - 选择 `riskMode`
  - 按任务传 `riskOverride`
  - 让 planner 产出最终生效的 `riskProfile`
  - 在 runtime / result / probe / snapshot 中记录 `riskHit`
- 当前可覆盖的参数包括：
  - `requestIntervalMs`
  - `pageSize`
  - `directoryIntervalMs`
  - `cooldownSeconds`
  - `retryLimit`
  - `riskKeywords`

### 2.4 当前待补传树语义

- 当前 `pending_manual` 不再只体现在单条结果里。
- 当前已支持把这类结果聚合成待补传树，并写入：
  - `runtime.pendingTree`
  - `runtime.pendingCount`
  - `recent probe payload`
  - `provider status snapshot summary`
- 当前任务详情页和状态页都可以直接展示这棵树。
- 当前 `retry` 已支持优先缩小到待补传子集再执行。
- 当前 `retryQueue` 已支持把失败项分类为：
  - `pending_manual`
  - `rate_limited`
  - `auth_expired`
  - `local_file_missing`
- 当前 `retryQueue` item 已支持：
  - `attemptCount`
  - `retryLimit`
  - `remainingCount`
  - `exhausted`
- 当前 `blocked` 证据已支持：
  - `blockedReason`
  - `blockedAction`
  - `blockedAdvice`
  - `nextRetryAt`
- 当前状态矩阵聚合已支持：
  - evidence 级 `blockedTasks`
  - evidence/provider 级 `blockedActions`
  - provider 级 `blockedCount`
- 当前要避免误解：
  - 这表示“待补传结构已经可见，而且已具备 pending-only retry、`blocked` 运行态、retry-limit 阻断和最小自动补传闭环”
  - 不表示“后台补传队列调度已经全部完成”

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

1. 把目录状态继续补到更细的树结构和可折叠交互
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
  - 已经支持作为正式 `executionMode` 参与 API 和任务运行
  - 已经支持目录状态持久化和带部分结果的继续执行
  - 已经支持配合目标端 metadata 做 `create / overwrite / skip` 运行时判定
  - 已经支持待补传树聚合和目录树展示
  - 已经支持在 `retry` 时缩小到待补传子集继续执行
  - 已经支持冷却到期后的最小自动补传恢复
  - 不表示复杂目录树交互、细粒度后台补传编排、异步运行中暂停已经全部完成
- “有 rate_limited 场景” 当前表示：
  - 已有测试语义、任务级参数覆盖和风险命中证据
  - 不表示风控策略层已经完全做成最终形态
- “可以运行任务” 当前表示：
  - 任务内核、状态机、结果写库、证据聚合都已经可用
  - 不表示所有 provider 上传都已经是真实外部平台传输
- “有前端控制台” 当前表示：
  - 当前已经适合开发联调、演示和 smoke
  - 已经支持任务向导里直接选择 `riskMode` / `executionMode` 并展示推荐理由
  - 已经支持任务详情和状态页里直接查看目录状态与运行检查点
  - 已经支持在任务详情和状态页按路径 / 状态 / reason 筛选目录树与待补传树
  - 已经支持把待补传树切到“仅叶子节点”视角
  - 不表示已经达到最终产品化 UI 形态
