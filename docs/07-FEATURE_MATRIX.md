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
  - provider 真实联网样本沉淀与边界场景验收
  - 同步执行模型
  - 风控与频率策略

## 功能矩阵

| 模块 | 当前可用内容 | 已完成状态 | 进行中 | 待完成 |
| --- | --- | --- | --- | --- |
| 项目骨架 | Go 服务、配置、日志、路由、静态资源、SQLite migration | 已完成 | 无 | 无 |
| 统一 API | 登录、provider 列表、能力查询、auth profile、plans、tasks、evidence、status | 已完成 | 无 | 后续仅按真实联调补细节 |
| Provider 注册表 | 10 家 provider 已注册，能力声明统一 | 已完成 | 无 | 后续逐家替换真实实现 |
| Provider 调试接口 | `list`、`metadata`、`create_dir`、`fast_check`、`upload` 已暴露为 API | 已完成 | 联调口径补强 | 后续可继续补更细上传观测字段 |
| Auth Profile | CRUD、校验、脱敏持久化、按 provider 映射 | 已完成 | 真实 provider 字段口径补充 | 更多真实鉴权模式 |
| Planner | 预览计划、策略判定、阈值判断、冲突降级、执行模式推荐、风控档位元数据、源端删除记录汇总 | 已完成 | 结合真实 provider 与增量规则继续校准 | 更细推荐规则与真实联调校准 |
| 同步执行模型 | 已支持 `leaf_first_lazy`、`pre_scan_flat`、按需扫描骨架、fallback、目录状态持久化、断点继续当前子树、目标端 metadata 预检查、`create / overwrite / skip` 判定闭环、待补传树聚合、待补传子集重试、失败重试队列分类、`blocked` 运行态、单机 worker 自动补传调度、启动即恢复扫描、`retryLimit` 次数耗尽阻断、`fast_upload` 预检后直回退、upload-session 级自动续跑、协作式运行中暂停、`pause_requested -> paused` 可见中间态、暂停请求撤销、自动恢复证据留痕、`retrySummary` 队列拆分统计、后台补传候选模式判定、后台补传候选池聚合、候选池可执行/等待态拆分、worker 优先级排序、按路径子集重建 retry 范围、手动触发后台补传、`dryRun` 预演后台补传、预演/执行 `decisions` 明细、批次上限控制、按 `retryClass` 精准筛选后台补传、按 `blockedAction` 精准筛选后台补传、按 `protocolGroup` 精准筛选后台补传、多路径子树后台补传、按 `taskId` 精准约束后台补传、按 `profileId` 精准约束后台补传、按目录树当前筛选重建执行子集、手动覆盖 `limitPerMode / limitPerLane / limitPerProtocolGroup / limitPerProvider / limitPerProfile` | 部分完成 | 任务级执行模式、runtime checkpoint、待补传树与目录树展示已接入 planner / task / evidence / UI，`retry` 已支持 pending-only 重建、`selected_pending_subset / selected_retry_subset / selected_directory_subset` 子集重建，并识别 rate-limit cooldown、retry limit、冷却自动恢复、upload checkpoint 自动续跑，以及 `retry_window_waiting_auto_retry / cooldown_elapsed_auto_retry / upload_checkpoint_auto_resume / retry_queue_auto_retry` 四类后台补传模式；候选池会按 `upload_checkpoint_auto_resume -> retry_queue_auto_retry -> retry_window_waiting_auto_retry -> cooldown_elapsed_auto_retry` 排序，并支持按 `mode / taskId / protocolGroup / providerKey / profileId / retryClass / blockedAction / recoverState / path / paths / scope / limit / limitPerMode / limitPerLane / limitPerProtocolGroup / limitPerProvider / limitPerProfile / dryRun` 手动放行或预演一批候选；同档位候选会先做 provider 轮转，再在同 provider 内按授权档案轮转，并继续受 mode / lane / protocolGroup / provider / profile 五级预算约束；状态页还能按 `protocolGroup / primaryRetryClass / primaryBlockedAction / lane(mode + class + action)` 一键聚焦、预演或执行，并查看最近一次预演/执行的 `decisions` 明细，任务树节点也可直接放行当前子树、当前筛选出的多棵子树，或最近样本任务的当前筛选结果；目录树与待补传树现已支持节点级子树折叠、父级回退、子树路径复制、仅问题节点视角和更完整的筛选摘要 | 更复杂后台补传编排、更真实样本校准 |
| 风控与频率策略 | 已支持 `safe / balanced / fast / custom` 基线、provider 校准默认模板、任务级 `riskOverride`、风控命中证据、runtime 基础节流和 `throttle` 证据、`riskProfileResolution` 解释链、`maxConcurrent`、自动补传时间窗、`rate_limited` 重试队列分层退避 | 部分完成 | 已接入 planner / task metadata / runtime evidence / UI，运行时会按 `requestIntervalMs / directoryIntervalMs` 控制 item 间隔，控制台已提供表单化参数配置，并能直接显示 `base / calibrated / applied / calibrationReasons / overrideFields`；单机自动补传会尊重 `autoRetryStartHour / autoRetryEndHour`，且 planner 会先把时间窗归一化到合法小时范围；后台补传单轮放行会尊重 `maxConcurrent` 的 provider 级批量预算，并在同 provider 内按授权档案轮转；自动 tick 还会附带 mode / lane / protocolGroup / provider / profile 五级默认公平预算；状态页还可手动覆写 provider / profile 两级预算；重试队列会按失败次数展示 `fast / normal / extended` 冷却档位、实际 `cooldownSeconds` 与 `eligibleAt` | 真实样本反推的账号级模板、更细 provider 级限流 |
| Task Runtime | 创建、查询、运行、暂停、恢复、重试、结果落库，结果证据包含 `fastCheck / fallbackUsed / fallbackFrom / upload / uploadedParts / failedPartNumber / nextPartNumber`，并能区分 `completionKind`；runtime / retry queue / status 已可展示 `uploadCheckpoint`，`Retry` 会保留 `retryUploadCheckpoints`，并支持 `providerData` 级恢复线索透传 | 已完成 | 后续将挂接执行模型和风控策略 | 真实上传链路接入后补更细运行态 |
| Runtime Evidence | 最近结果、最近探针、状态快照、状态矩阵 API、协议族覆盖矩阵、验收报告生成与历史保存、真实 provider smoke 记录与协议组聚合矩阵、真实样本矩阵、accepted/pending 验收判定、acceptanceMissing 缺失原因、acceptanceAdvice 补齐建议、accepted/in_progress/pending 统计、smoke Markdown 导出与分类、验收矩阵状态筛选与样本/任务直达、源端删除记录聚合 | 已完成 | 无 | 真实联调样本沉淀 |
| 控制台前端 | 登录、授权、任务向导、任务列表详情、状态矩阵/证据、执行模式可视化、目录状态展示、目录树/待补传树筛选与叶子视角、目录树按 root 分组收起/展开并持久化、目录树节点级子树收起/展开、仅问题节点视角、父级回退、子树路径复制、更完整筛选摘要、重试队列分类视图、上传恢复检查点展示、后台补传候选模式展示、队列拆分摘要展示、协议族覆盖展示、验收报告保存/历史查看/下载、真实 provider smoke 记录登记/查看/协议组聚合/真实样本矩阵/Markdown 下载/分类选择、验收矩阵按 accepted/in_progress/pending 筛选并可直接打开 smoke 样本或样本任务、按协议组预填 smoke 表单、目录树/待补传树一键清空筛选、root/path 级“只看当前路径 / 同步另一棵树”、当前筛选路径一键复制、表单化风控覆盖入口、按当前筛选结果重建 retry 子集、状态页手动触发后台补传、状态页先预演当前筛选、lane 级一键预演建议预算、树节点级“按路径重建向导 / 重试当前路径”、按当前筛选结果一键后台补传、状态页最近样本按当前筛选直接重试/后台补传、按当前筛选结果重建向导、状态页补传预算输入框、mode/lane/provider/profile 四级预算输入 | 已完成 | 异常场景提示可继续增强 | 更产品化视觉 |
| 单元测试 | auth、planner、task、provider、workflow、web、UI smoke | 已完成 | 持续补样本 | 真实 provider 契约覆盖继续加深 |
| Provider 真实实现 | 10 家 provider 已全部接入真实 `ValidateAuth / List / Metadata / CreateDir / FastUploadCheck / Upload` 主链路；`aliyundrive_open / quark / uc / guangya` 已具备 multipart 或分片级恢复证据，`xunlei / pikpak / 115_open / baidu_netdisk / 189cloud` 已具备整对象上传 checkpoint 证据 | 部分完成 | 继续沉淀真实账号样本、限流/风控样本和 provider 边界异常 | 更细粒度断点续传、登录态续期、更多真实验收样本 |
| 真实联调验收 | 模板、流程、文档已具备，已补 accepted/pending 验收判定层与矩阵视图（真实 smoke 成功 + 该协议组已有任务覆盖） | 部分完成 | 首批真实样本沉淀中 | 每个协议族至少一条真实成功样本 |

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
- 当前控制台会把 `Selected Roots` 和运行后的 `Scan Trace` 明确展示出来，而且路径按钮可直接定位目录树，任务页和状态页都能用，任务摘要里的路径芯片也能直接点，方便解释“叶子目录优先 + 按需扫描”的真实顺序。
- 验收报告里的代表任务样本也会同步带上 `Selected Roots` 和 `Scan Trace`，方便单独分享和离线回看。
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
  - 按目标 provider 校准默认 `requestIntervalMs / pageSize / directoryIntervalMs / cooldownSeconds / retryLimit`
  - 在 runtime / result / probe / snapshot 中记录 `riskHit`
  - runtime 按 `requestIntervalMs` 和 `directoryIntervalMs` 做基础节流
  - 每条被节流的 result payload 会记录 `throttle.waitMs / previousPath / currentPath`
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
  - `cooldownTier`
  - `cooldownSeconds`
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
  - 协议族级 `protocolCoverage`
- 当前后台自动恢复证据已支持：
  - runtime 级 `autoRecovered / autoRecoverReason / autoRecoverCount / autoRecoveredAt / autoRecoverState`
  - result 级 `autoRecovery`
  - provider probe / status 级自动恢复摘要
- 当前 `retrySummary` 还会明确给出：
  - `retryableNowCount / cooldownCount / pendingManualCount / authExpiredCount / localMissingCount / exhaustedCount`
  - `uploadCheckpointEligible`
  - `autoRecoverEligible / autoRecoverMode / autoRecoverAdvice`
- 当前运行证据摘要和 provider 状态快照还会给出：
  - `autoRecoverTasks`
  - `autoRecoverPool`
- 当前任务详情、运行检查点和状态快照会直接显示：
  - `后台补传候选`
  - `队列拆分`
  - `自动补传提示`
- 当前状态页还新增了：
  - “自动补传候选池”聚合视图
  - provider 级 `autoRecoverCount`
- 候选池行级“只看主重试类型 / 只看主阻塞动作 / 只看该 lane / 预演该 lane / 执行该 lane”快捷入口
  - 任务目录树 / 待补传树节点级“后台补传当前路径 / 当前 root”快捷入口
  - 目录树与待补传树对齐的“仅叶子节点”筛选视角
- 当前要避免误解：
  - 这表示“待补传结构已经可见，而且已具备 pending-only retry、`blocked` 运行态、retry-limit 阻断、冷却到期自动恢复、upload-session checkpoint 的最小自动续跑闭环、协作式运行中暂停，以及自动恢复证据留痕”
  - 待补传树的 root 顺序会优先遵循 `selectedRoots`，这样任务详情里看到的补传上下文和用户勾选顺序一致
  - 源端删除记录当前只做记录、展示和证据沉淀，不默认删除目标端真实文件
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
| `aliyundrive_open` | `aliyun_123_open` | 部分真实 | 已接入真实 `ValidateAuth`，并已落地 `List / Metadata / CreateDir / FastUploadCheck / Upload / 多分片上传` 主链路 |
| `123_open` | `aliyun_123_open` | 部分真实 | 已接入真实 `ValidateAuth / List / Metadata / CreateDir / FastUploadCheck / Upload`，当前为单分片上传主链路，`fast_upload` 已可通过 `file/create` 的 `reuse` 命中完成 provider 侧秒传，`overwrite_existing` 会降级为 `auto_rename_new` |
| `xunlei` | `xunlei_pikpak` | 部分真实 | 已接入真实 `ValidateAuth / List / Metadata / CreateDir / FastUploadCheck`，`Upload` 已接入真实 create + verify 主链路，并已内置 hash miss 后的 S3-compatible SigV4 PUT fallback；失败重试时可复用既有 `resumable` 会话，整对象 PUT 失败/成功会按统一 checkpoint 口径回填证据 |
| `pikpak` | `xunlei_pikpak` | 部分真实 | 已接入真实 `ValidateAuth / List / Metadata / CreateDir / FastUploadCheck`，`Upload` 已接入真实 create + verify 主链路，并已内置 hash miss 后的 S3-compatible SigV4 PUT fallback；失败重试时可复用既有 `resumable` 会话，整对象 PUT 失败/成功会按统一 checkpoint 口径回填证据 |
| `quark` | `quark_uc` | 部分真实 | 已接入真实 `ValidateAuth / List / Metadata / CreateDir / FastUploadCheck / Upload`，当前基于 share token + detail + download info + drive create folder；`Upload` 已支持 `upload/pre -> update/hash -> upload/finish`，hash miss 会继续走 `upload/auth + OSS multipart + upload/finish`，失败后可基于 `uploadId + uploadedParts + providerData` 继续分片上传 |
| `uc` | `quark_uc` | 部分真实 | 已接入真实 `ValidateAuth / List / Metadata / CreateDir / FastUploadCheck / Upload`，当前基于 share token + detail + download info + drive create folder；`Upload` 已支持 `upload/pre -> update/hash -> upload/finish`，hash miss 会继续走 `upload/auth + OSS multipart + upload/finish`，失败后可基于 `uploadId + uploadedParts + providerData` 继续分片上传 |
| `baidu_netdisk` | `baidu_netdisk` | 部分真实 | 已接入真实 `ValidateAuth / List / Metadata / CreateDir / FastUploadCheck / Upload`，当前二进制上传链路为 `precreate -> superfile2 tmpfile -> create -> verify`，`overwrite_existing` 会降级为 `auto_rename_new`，tmpfile 失败/成功会按统一 checkpoint 口径回填证据 |
| `115_open` | `115_open` | 部分真实 | 已接入真实 `ValidateAuth / List / Metadata / CreateDir / FastUploadCheck`，`Upload` 已接入真实 `upload/init + sign_check + get_token + OSS PUT fallback` 主链路；hash 命中可直接成功，hash miss 已可继续做二进制上传与上传后校验，失败重试时可复用既有 OSS upload session，整对象 PUT 失败/成功会按统一 checkpoint 口径回填证据 |
| `189cloud` | `189cloud` | 部分真实 | 已接入真实 `ValidateAuth / List / Metadata / CreateDir / FastUploadCheck / Upload`；当前读链路基于 `shareCode / accessCode`，`CreateDir` 已接入 `AccessToken / Signature / Date` 写鉴权，`Upload` 已支持 `getSessionForPC -> createUploadFile -> fileUploadUrl PUT -> getUploadFileStatus -> fileCommitUrl`，hash miss 会继续走 binary PUT fallback，失败重试时可复用既有 upload session，binary PUT 失败/成功会按统一 checkpoint 口径回填证据 |
| `guangya` | `guangya` | 部分真实 | 已接入真实 `ValidateAuth / List / Metadata / CreateDir / FastUploadCheck / Upload`；当前目录链路走 Guangya live HTTP API，fast-check 已接入库存预检与 GCID follow-up，`Upload` 已支持快传命中确认、小文件 `upload_token + md5 + upload_info`、大文件 `upload_token + GCID flash-check + OSS multipart + upload_info`，并支持基于 `uploadId + uploadedParts` 的 multipart checkpoint 续传 |

## 当前已完成的里程碑视角

- 已完成项目骨架和基础文档
- 已完成统一资源模型和 API 主链路
- 已完成 auth / planner / task / evidence 内核
- 已完成控制台前端和 UI smoke
- 已完成 10 家 provider 的统一注册
- `aliyundrive_open`、`123_open`、`xunlei` 与 `pikpak` 已进入真实链路阶段
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
  - 已经支持启动即扫和冷却到期后的最小自动补传恢复
  - 已经支持 `completed_with_errors` 中 upload-session checkpoint 队列的后台自动续跑
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
  - 已经支持任务详情和状态页里直接查看 retryQueue 分类、attempt / retryLimit / remainingCount / eligibleAt
  - 已经支持在任务详情和状态页按路径 / 状态 / reason 筛选目录树与待补传树
  - 已经支持按 retry class / retry state 筛选重试队列项
  - 已经支持从 retry queue item 一键定位待补传树或同类失败项
  - 已经支持从 blocked 聚合摘要一键跳到样本任务或当前阻塞动作
  - 已经支持从任务详情 blocked 引导卡片一键收敛当前任务问题子集
  - 已经支持把待补传树切到“仅叶子节点”视角
  - 已经支持目录树节点级子树折叠、父级回退、子树路径复制和“仅问题节点”视角
  - 不表示已经达到最终产品化 UI 形态
