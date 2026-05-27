# CloudPan Sync Go 项目现状与功能清单

## 文档目的

- 本文档补充 `docs/01-GO_REBUILD_PLAN.md` 的实施基线信息。
- 面向项目维护者、协作者和新接手同学，说明“这个项目要做什么、已经做到哪里、接下来还差什么”。
- 如果把当前仓库单独分享出去，建议先看本文档，再看计划文档和代码目录。

## 建议和哪些文档一起看

- 如果你想先看“做到了哪一步”，配合阅读：`docs/07-FEATURE_MATRIX.md`
- 如果你想马上启动项目，配合阅读：`docs/03-DEVELOPER_GUIDE.md`
- 如果你想按 API 实际调试，配合阅读：`docs/04-API_WORKFLOW_EXAMPLES.md`

## 项目定位

- 这是 CloudPan Sync 的 Go 重构版，不再沿用 Python 单体结构。
- 当前版本目标是先完成一个可运行、可测试、可演进的 Go 基础工程：
  - 提供统一 API
  - 提供统一 provider 抽象
  - 提供 auth / planner / task / runtime evidence 主链路
  - 提供一个轻量控制台用于联调和演示
- 当前产品语义是“多 provider 互传”，不是“固定源到固定目标”的定向工具。
- 当前仓库已经不是“只有骨架”，而是已经具备完整主流程闭环；但 provider 真实联网实现仍以协议占位和行为模拟为主。

## 当前一句话状态

- 当前已经完成“统一内核 + 控制台 + 测试回归”的主体建设。
- 当前开发重心已经从“搭框架”切换到“逐家 provider 落真实链路”。
- 同时也要补回原始项目中不可丢失的两类通用能力：执行模型与风控策略。

## 先用业务语言理解当前规则

- 当前项目不是“扫完再统一上传”的单一模型，而是允许任务级选择执行模式。
- `leaf_first_lazy` 是当前默认优先推荐模式，但不是强制模式。
- 它的真实含义是：
  - 多个顶层目录按勾选顺序逐棵推进
  - 每棵子树内部优先下探更深目录
  - 只拉当前下一步需要传的目录列表，不预扫整个树
- `pre_scan_flat` 仍然保留，适合目录较小、需要先拿完整扫描结果的场景。
- 当前增量同步语义也已经固定下来：
  - 目标不存在：创建上传
  - 目标存在但指纹变化：覆盖上传或按冲突策略降级
  - 目标已同步且指纹一致：跳过
  - 源端删除：默认只记录，不默认删除目标端
- 当前风控语义也不再只有档位名：
  - 可以按任务覆盖节流参数
  - 可以按任务覆盖风险关键词
  - 运行证据里会回写命中的风险状态
- 也就是说，当前 Go 版保留的是“互传内核 + 可选模式 + 推荐提示”，不是把所有任务都硬绑到一种执行方式。

## 当前功能范围

### 1. 后端应用基础

- 已完成：
  - Go module、目录结构、配置加载、日志、HTTP 服务
  - SQLite 持久化与启动 migration
  - 统一错误响应、统一 JSON 响应结构
  - 嵌入式静态前端资源服务
- 当前状态：
  - 可直接启动服务
  - 可通过 API 和控制台完成主流程演示

### 2. Provider 抽象与注册表

- 已完成：
  - `Provider` 元信息模型
  - `CapabilitySet` 能力声明
  - `Adapter` 统一接口：
    - `ValidateAuth`
    - `List`
    - `Metadata`
    - `CreateDir`
    - `FastUploadCheck`
    - `Upload`
  - provider registry 与 10 家 provider 注册
- 已接入的 10 家 provider：
  - `guangya`
  - `aliyundrive_open`
  - `115_open`
  - `quark`
  - `189cloud`
  - `baidu_netdisk`
  - `uc`
  - `xunlei`
  - `pikpak`
  - `123_open`
- 当前状态：
  - 协议族和能力面已经统一
  - 各 provider 具备可执行占位行为，能参与 API、planner、task、evidence 全链路
  - `aliyundrive_open` 已开始接入真实 `ValidateAuth` 远程校验骨架
  - 真实外部平台 SDK / HTTP 联网细节仍未全面接入

### 2.1 Provider 辅助调试接口

- 已完成：
  - `POST /api/providers/{key}/list`
  - `POST /api/providers/{key}/metadata`
  - `POST /api/providers/{key}/create_dir`
  - `POST /api/providers/{key}/fast_check`
- 当前状态：
  - 这些接口已经可以直接拿来验证 provider 适配器行为
  - 对联调和排查字段口径很有帮助
  - 其中真实联网深度仍取决于具体 provider 的落地进度

### 3. Auth Profile 授权档案

- 已完成：
  - 授权档案 CRUD
  - 授权验证
  - 脱敏存储与读取
  - 面向 provider 的档案映射
- 当前状态：
  - 已可在控制台创建、查看、删除、验证授权档案
  - 适合做后续真实登录态接入的统一承载层

### 4. Planner 规划器

- 已完成：
  - `TransferPlan` 和 `TaskItem` 规划生成
  - 基于阈值、指纹条件和 provider 能力的策略判定
  - 冲突策略降级
  - deep-first / 顺序控制相关测试覆盖
- 当前支持的策略语义：
  - `fast_upload`
  - `download_upload`
  - `pending_manual`
- 当前状态：
  - 已能稳定输出预览计划
  - 可作为任务创建前的标准入口
  - 但目前更偏“单次任务策略选择”，还没有完整升级为“叶子目录优先 + 增量 / 补传”执行模型

### 4.1 同步执行模型

- 已完成：
  - `deepest-first` 顺序相关测试基础
  - `fast_upload / download_upload / pending_manual` 运行语义
  - fallback、状态机、结果写库等运行时基础
  - `sourceProfileId + selectedRoots` 的按需扫描骨架
  - 顶层目录按顺序、子树内部叶子优先的懒展开执行样例测试
  - `executionMode` 任务级配置能力
  - `leaf_first_lazy` 与 `pre_scan_flat` 两种模式的 planner / task 联动
  - 推荐模式与推荐原因返回
  - 目标端 metadata 预检查
  - `create / overwrite / skip` 运行时判定闭环
  - 目录状态与运行时恢复点持久化
  - 带部分结果的任务继续当前子树而不是整任务重跑
- 当前状态：
  - 已不再只是“平铺排序语义”，而是开始具备真正的按需扫描执行骨架
  - 当前 `leaf-first lazy scan` 应理解为：
    - 任务级可选模式
    - 当前默认优先推荐模式
    - 先按顶层目录顺序处理
    - 每棵子树内部先下探再上传
    - 不预先拉完整棵目录树
  - 当前 API / metadata / probe 已可直接表达：
    - 当前任务执行模式
    - 推荐执行模式
    - 推荐原因
    - 当前同步判定
  - 当前扫描方式
  - 当前根目录 / 当前目录 / 上次完成路径
  - 目录级状态、完成数、跳过数、失败数
  - 最近待补传树
  - 当前重试是否已缩小为待补传子集
  - 但原始项目要求的这些通用能力还未完整落地：
    - 更完整的后台补传调度与队列策略
    - 真正异步 worker 下的运行中暂停 / 恢复
  - 这部分已经明确纳入 Go 版后续主线，不再视为可选增强

### 4.3 当前增量判定口径

- 当前 runtime 在真正调用上传前，会先看目标端 `Metadata`。
- 首版为了避免占位 provider 误报“已存在”，当前采用保守存在判定：
  - `MetadataResult.Status == "exists"`
  - 或 `MetadataResult.Entry.exists == true`
- 如果只是返回通用 `ok`，不会直接视为目标已存在。
- 当前这样设计的原因是：
  - 仓库里仍有不少 provider 是占位实现
  - 这些实现常常能返回一个“占位 metadata”
  - 但并不代表真实目标端已经有这个文件
- 所以当前宁可保守走 `create`，也不允许误把全部文件跳过。

### 4.2 风控与频率策略

- 已完成：
  - `rate_limited` 运行场景测试
  - provider 能力描述中已有部分 fallback / 冲突行为信息
  - `safe / balanced / fast / custom` 风控档位基线
  - provider 默认风险关键词与默认参数模板
  - 任务级 `riskOverride`
  - 风控命中证据回写到 runtime / result / probe / snapshot
- 当前状态：
  - 已形成第一版统一风控档位模型，并进入 planner / task metadata
  - 已支持覆盖：
    - 请求间隔
    - 页面大小
    - 目录间隔
    - 冷却时间
    - 重试次数
    - 风险关键词
  - 已支持在结果与状态证据里看到：
    - `riskHit`
    - `riskHitCount`
    - `lastRiskStatus`
  - 目前仍缺少：
    - 更完整的表单化任务级覆盖入口
    - 真实 provider 校准后的更细默认模板
  - 这部分已经明确纳入 Go 版主线能力，而不是只在单个 provider 中零散处理

### 5. Task 队列与运行时

- 已完成：
  - `Task / TaskItem / TaskResult` 模型
  - 任务创建、查询、运行、暂停、恢复、重试
  - 任务状态机
  - 运行结果写库
  - runtime evidence 聚合
  - provider status snapshot 聚合
  - 任务运行结果透传执行模式、推荐模式、扫描方式、风险档位等元数据
  - task payload 中持久化 runtime / directory states / resume checkpoint
- 已覆盖的关键运行场景：
  - `fast_upload`
  - `hash_miss -> download_upload fallback`
  - `overwrite_existing -> auto_rename_new downgrade`
  - `target_already_synced -> skip`
  - `target_changed -> overwrite`
  - `pending_manual_requires_confirmation`
  - `auth_expired`
  - `rate_limited`
  - `local_file_missing`
- 当前状态：
  - API 主工作流已完整打通
  - 运行时更偏“可验证内核 + 可扩展占位实现”，而非真实大规模传输引擎
  - 当前已支持把 `pending_manual` 类结果聚合成待补传树，并回写到 runtime / probe / provider status
  - 当前已支持在 `Retry` 时自动缩小到待补传子集，并按新 plan 继续执行
  - 当前已支持把失败结果分类成重试队列，并区分：
    - `pending_manual`
    - `rate_limited`
    - `auth_expired`
    - `local_file_missing`
    - `retry_limit_exhausted`
  - 当前 `rate_limited` 会按冷却时间阻断过早重试
  - 当前任务在“只有冷却 / 人工确认 / 授权失效 / 本地文件缺失”时会进入 `blocked`
  - 当前 runtime / probe / status 已回写 `blockedReason` 与 `nextRetryAt`
  - 当前已接入单机 tick 版后台自动补传调度，可自动恢复冷却到期的 `blocked` 任务
  - 当前 `retryLimit` 已真正接入重试队列，支持累计次数、剩余次数与 exhausted 阻断
  - 当前 `blockedReason` 已补充统一的 `blockedAction / blockedAdvice`，便于状态矩阵直接给出处理建议
  - 当前 `/api/evidence/runtime` 与 `/api/status/providers` 已补充 `blockedTasks / blockedActions` 聚合摘要，便于快速定位最需要人工处理的动作

### 6. Runtime Evidence / 状态矩阵

- 已完成：
  - 最近任务结果聚合
  - 最近 provider probe 聚合
  - provider 状态快照聚合
  - `/api/evidence/runtime`
  - `/api/status/providers`
- 当前可展示的数据包括：
  - `recentResults`
  - `recentProbes`
  - `latestProbe`
  - `lastTaskState`
  - `snapshotSummary`
  - `executionMode`
  - `recommendedExecutionMode`
  - `scanMode`
  - `runtime`
  - `doneCount`
  - `skippedCount`
  - `failedCount`
  - `pendingCount`
  - `pendingTree`
  - `riskHitCount`
  - `lastRiskStatus`
  - `currentRoot`
  - `currentDirectory`
  - `lastCompletedPath`
- 当前状态：
  - 已具备联调、排错、演示所需的最小证据链

### 7. 前端控制台

- 已完成固定页面与交互：
  - 登录
  - Provider / 授权
  - 任务向导
  - 任务列表详情
  - 状态矩阵 / 证据
- 当前状态：
  - 前端只消费 Go 新 API
  - 已可用于本地联调、主流程 smoke 和功能演示
  - UI 目前偏工程控制台风格，不是最终产品化视觉版本
  - 已补充浏览器级 UI smoke 回归测试，覆盖主流程关键交互
  - 任务向导已支持：
    - `riskMode` 选择
    - `riskOverride(JSON)` 输入
    - `executionMode` 选择
    - 推荐模式与推荐原因展示
  - 任务详情、状态矩阵、最近结果、最近 probe 已可直接看到执行模式证据
  - 任务详情和状态页已支持：
    - 运行检查点展示
    - 目录状态树展示
    - 当前根目录 / 当前目录 / 上次完成路径展示
    - 待补传树展示
  - 后续仍需继续补：
    - 更细粒度的目录树交互
    - 真正运行中的暂停体验

## 已完成里程碑概览

- 已完成：
  - 重构计划基线文档
  - 项目状态、上手说明、API 示例、Provider 接入、真实联调模板等配套文档
  - Go 项目骨架
  - SQLite migration 与仓储基础
  - Provider 抽象与 10 家 provider 注册
  - Auth / Planner / Task 核心主链路
  - Runtime evidence 与状态矩阵聚合
  - 控制台前端骨架与证据页增强
  - API 主工作流联调回归
  - Runtime 关键场景测试覆盖
  - 控制台浏览器级 UI smoke 回归
  - `aliyundrive_open` 真实 `ValidateAuth` 骨架

## 当前仍未完成的部分

### 1. Provider 真实落地仍不完整

- 现在的 provider adapter 已统一接入，但大部分仍属于：
  - 协议占位
  - 字段口径模拟
  - 行为级假实现
- 还缺：
  - 真实接口调用
  - 真实登录态续期
  - 真实目录 / 文件元数据映射
  - 真正的上传链路与异常恢复

### 2. UI 自动化 smoke 仍可继续增强

- 当前已经补上仓库内可重复执行的浏览器级 smoke 主流程回归
- 后续仍可继续增强：
  - 更多异常场景 UI 提示校验
  - 多 provider 组合样本
  - 截图或录屏级产物沉淀

### 3. “每个协议族至少一条真实成功样本” 还未完成

- 当前更像“统一内核已经搭好，provider 真实联调待补”
- 真正的切换验收还需要真实 auth profile 和真实链路样本

### 4. 对外分享说明还可以继续增强

- 当前仓库已经能看懂并启动
- 但后续仍建议继续补：
  - 按模板逐步沉淀真实 provider 样本记录

### 5. 执行模型仍未补成完整主能力

- 当前还缺：
  - 更细补传队列策略与更复杂的后台调度编排
  - UI 中更完整的执行模式配置与推荐提示联动
  - 更细粒度的目录树交互与筛选

### 6. 风控策略层仍未独立成型

- 当前还缺：
  - 更友好的表单化任务级参数配置
  - 真实 provider 样本校准后的更细默认模板

## 当前最清晰的完成度判断

- 如果按“工程能否独立运行、独立测试、独立演示”来判断：
  - 已经完成
- 如果按“10 家 provider 是否都已经真实联网可用”来判断：
  - 还没有完成
- 如果按“是否已经进入真实联调阶段”来判断：
  - 已经进入，而且 `aliyundrive_open` 是当前优先推进样本
- 如果按“原始项目里的执行模型和风控策略有没有完整继承”来判断：
  - 还没有完全继承，但执行模式、任务级风险覆盖和风控命中证据已经接入主链路
- 如果按“叶子目录优先是不是已经从单纯排序升级为按需扫描骨架”来判断：
  - 已经开始接通，但还没完全收口

## 推荐阅读顺序

1. `README.md`
2. `docs/02-PROJECT_STATUS.md`
3. `docs/07-FEATURE_MATRIX.md`
4. `docs/01-GO_REBUILD_PLAN.md`
5. `internal/app`
6. `internal/provider`
7. `internal/auth` / `internal/planner` / `internal/task`
8. `web/static`

## 下一阶段建议

- 优先继续：
  - 执行模型内核补齐
  - 风控策略层落地
  - provider 真实联调样本沉淀
- 如果要让别人快速参与开发，最推荐的切入点：
  - 先读 `docs/07-FEATURE_MATRIX.md` 明确哪些是主线缺口
  - 再补执行模型或风控策略中的一个独立里程碑
  - 最后结合真实 provider 样本推进联调
