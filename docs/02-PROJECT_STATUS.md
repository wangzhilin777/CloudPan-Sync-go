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
- 当前仓库已经不是“只有骨架”，而是已经具备完整主流程闭环；provider 真实联网实现已从 `aliyundrive_open` 扩展到 `123_open / xunlei / pikpak / baidu_netdisk / 115_open / quark / uc / 189cloud / guangya`，剩余缺口主要集中在更多真实样本验收、断点恢复深化和后台执行策略补齐。

## 当前一句话状态

- 当前已经完成“统一内核 + 控制台 + 测试回归”的主体建设。
- 当前开发重心已经从“搭框架”切换到“逐家 provider 落真实链路”。
- 同时也要补回原始项目中不可丢失的两类通用能力：执行模型与风控策略。
- 当前 task runtime 已具备第一版后台自动恢复：
  - 应用启动后会立即做一次恢复扫描，避免冷却任务必须等到第一个 tick
  - 冷却到期的 `blocked` 任务可自动重建并继续执行
  - `completed_with_errors` 中“全队列均带 upload checkpoint / providerData 的 upload-session 恢复型失败”可自动补跑
  - 普通 `remote_error` 之类的泛化失败不会被后台盲目自动重试
  - 手动触发后台补传时，已经支持按 `mode / providerKey / retryClass / limit` 精准筛选放行
  - 当前也已支持按 `blockedAction` 做历史阻塞原因口径的精准筛选与手动放行
  - 当前混合失败队列的后台补传候选也已进入统一优先级排序：会综合 `mode / primaryRetryClass / primaryBlockedAction / nextRetryAt` 做稳定拆批
- 当前状态页候选池已可直接按 `protocolGroup / recoverState / primaryRetryClass / primaryBlockedAction / lane(mode + class + action)` 一键聚焦或放行
- 当前真实样本矩阵除了区分 `accepted / in_progress / pending`，还会单独标记协议组是否已经具备“真实上传成功样本”，避免把 `browse_only` 成功和真实上传成功混成一类
  - 当前 `accepted` 已收口为“真实上传成功样本 + 该协议组已有任务覆盖”
  - 当前还会直接显示每个协议组已沉淀的真实上传成功样本数量，方便看联调进度不是停在 0/1 级别
- 当前 `/api/tasks/recover` 也已支持按 `path/paths + scope` 只放行一棵或多棵子树，便于和叶子目录优先排障结合使用
- 当前 `/api/tasks/recover` 还支持额外带 `taskId`，可把后台补传精准约束到单个任务样本，避免状态页排障时误打到同 provider 的其它任务
- 当前 `/api/tasks/recover` 也支持额外带 `profileId`，可把后台补传进一步精准约束到某个授权档案
- 当前 `/api/tasks/recover` 也支持额外带 `protocolGroup`，可把后台补传进一步压缩到某个协议族，便于协议族级排障或联调分批
- 当前 `/api/tasks/recover` 还支持 `dryRun=true` 预演本轮会放行多少、会被哪些预算或等待态挡住，便于先试算再真正执行
- 当前 `/api/tasks/recover` 的返回里还会附带 `decisions` 决策明细，直接列出样本任务 / provider / profile / path / recoverState / outcome / message，方便联调时定位到底是哪一层预算或等待条件拦住了它
- 当前 `/api/tasks/recover` 还支持额外带 `limitPerProtocolGroup / limitPerProvider / limitPerProfile`，可把同一轮放行预算进一步压到协议族级、provider 级或账号级
  - 当前同优先级档位下的后台补传候选会先按协议族轮转，再在协议族内部按 provider 轮转，最后在同 provider 内按授权档案轮转，避免单一协议族或单一账号长时间独占当前批次
  - 当前如果任务本身已经满足自动补传条件，但不在 `autoRetryStartHour / autoRetryEndHour` 时间窗内，也会被显式标记为 `wait_for_retry_window`，并继续留在候选池里等待下一个允许时间点

## 先用业务语言理解当前规则

- 当前项目不是“扫完再统一上传”的单一模型，而是允许任务级选择执行模式。
- `leaf_first_lazy` 是当前默认优先推荐模式，但不是强制模式。
- 它的真实含义是：
  - 多个顶层目录按勾选顺序逐棵推进
  - 每棵子树内部优先下探更深目录
  - 只拉当前下一步需要传的目录列表，不预扫整个树
- 控制台里会把 `Selected Roots` 和运行后的 `Scan Trace` 一并展示出来，方便直接看出叶子优先的推进顺序。
- `pre_scan_flat` 仍然保留，适合目录较小、需要先拿完整扫描结果的场景。
- 当前增量同步语义也已经固定下来：
  - 目标不存在：创建上传
  - 目标存在但指纹变化：覆盖上传或按冲突策略降级
  - 目标已同步且指纹一致：跳过
  - 源端删除：默认只记录，不默认删除目标端
  - 当前该规则已经升级为显式任务级参数 `sourceDeletePolicy`
    - 首版固定支持 `record_only`
    - 任务向导、计划预览、任务详情、状态矩阵和 evidence 都会回显这个策略
    - 未来如要支持真实删除，必须作为单独策略继续扩展，且默认关闭
  - 源端删除记录会进入预览 metadata、task runtime、provider status 和 evidence/report，便于排查最近源端变更
- 当前风控语义也不再只有档位名：
  - 可以按任务覆盖节流参数
  - 可以按任务覆盖风险关键词
  - 运行证据里会回写命中的风险状态
  - 自动补传时间窗会在 planner 阶段先归一化到合法小时范围，避免预览口径和 runtime 执行口径漂移
  - 后台补传单轮放行也会开始尊重 `riskProfile.maxConcurrent` 的 provider 级批量预算，避免某一类任务一次性吃掉整轮额度
  - 对风控更敏感或并发更大的档位，还会进一步收敛出默认账号级预算，避免同一个授权档案在同轮内吃掉全部 provider 配额
  - 同档位候选在额度之内也会优先做 provider 轮转，再在同 provider 内按原有时间顺序继续推进
- 当前 `rate_limited` 重试队列也已经不再只有单一冷却秒数：
  - 会按失败次数进入 `fast / normal / extended` 三档退避
  - 队列项会直接展示 `cooldownTier / cooldownSeconds / eligibleAt`
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
  - 可在控制台保存、查看和下载验收报告历史
  - 可登记真实 provider smoke 记录，并按协议组查看真实样本聚合、分类和 Markdown

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
  - 各 provider 已能参与 API、planner、task、evidence 全链路
  - `internal/provider/catalog_contract_test.go` 现已把 10 家 provider 的目录注册、协议族分组、全量能力声明和缺凭证校验锁成统一门禁，避免后续继续落地真实链路时把 v1 契约面改漂
  - `aliyundrive_open` 已接入真实 `ValidateAuth`，并已落地最小目录链路：
    - `List`
    - `Metadata`
    - `CreateDir`
    - `FastUploadCheck`
    - `Upload`
    - 多分片 `Upload`
  - `123_open` 已接入真实 `ValidateAuth / List / Metadata / CreateDir / FastUploadCheck / Upload`
    - 当前为单分片上传主链路
    - `fast_upload` 已可通过 `file/create` 的 `reuse` 命中完成 provider 侧秒传，未命中时返回 `hash_miss` 交给 fallback
    - `overwrite_existing` 会诚实降级为 `auto_rename_new`
    - 上传完成后会优先按 `fileId` 校验，再回退到父目录按名称确认
  - `xunlei` 已接入真实 `ValidateAuth / List / Metadata / CreateDir / FastUploadCheck`
    - `Upload` 已接入真实建上传请求与校验主链路
    - hash 命中时可直接完成 rapid 路径
    - hash miss 后会继续走内置 `resumable` 二进制 fallback
    - 失败重试时已可复用既有 `resumable` 会话继续上传，避免重复 create upload
    - 整对象 PUT fallback 失败时会按统一口径回填 `uploadId / partCount / failedPartNumber / nextPartNumber / providerData`
    - 当前默认构建已内置 S3-compatible SigV4 PUT 上传能力
  - `pikpak` 已接入真实 `ValidateAuth / List / Metadata / CreateDir / FastUploadCheck`
    - `Upload` 已接入真实建上传请求与校验主链路
    - hash 命中时可直接完成 rapid 路径
    - hash miss 后会继续走内置 `resumable` 二进制 fallback
    - 失败重试时已可复用既有 `resumable` 会话继续上传，避免重复 create upload
    - 整对象 PUT fallback 失败时会按统一口径回填 `uploadId / partCount / failedPartNumber / nextPartNumber / providerData`
    - 当前默认构建已内置 S3-compatible SigV4 PUT 上传能力
  - `quark` 与 `uc` 已接入真实 `ValidateAuth / List / Metadata / CreateDir / FastUploadCheck / Upload`
    - 当前真实链路基于 share token + share detail + download info + drive create folder
    - `Upload` 已接入真实 `upload/pre -> update/hash -> upload/finish` 主链
    - hash 命中时可直接完成 provider 侧快传确认
    - hash miss 时会继续走 `upload/auth + OSS multipart + upload/finish` 二进制 fallback
    - multipart fallback 失败时会回填 `uploadId / uploadedParts / failedPartNumber / nextPartNumber / providerData`
    - 失败重试时已可基于 `providerData` 恢复 upload session，并从失败分片继续 OSS multipart 上传
    - 同名冲突当前会诚实降级为 `auto_rename_new`
  - `baidu_netdisk` 已接入真实 `ValidateAuth / List / Metadata / CreateDir / FastUploadCheck / Upload`
    - 当前真实上传链路为 `precreate -> superfile2 tmpfile -> create -> verify`
    - `overwrite_existing` 会诚实降级为 `auto_rename_new`
    - 上传完成后优先按 `fileId` 校验，再回退到父目录按路径确认
    - 单分片 tmpfile 上传失败/成功会按统一 upload checkpoint 口径回填 `uploadId / partCount / uploadedParts / failedPartNumber / nextPartNumber`
  - `115_open` 已接入真实 `ValidateAuth / List / Metadata / CreateDir / FastUploadCheck`
    - `Upload` 已接入真实 `upload/init` 主链路、`sign_check` follow-up、`get_token` 上传会话获取与 OSS binary fallback
    - hash 命中时可直接完成 rapid 路径并校验
    - hash miss 后当前默认构建已可继续走 OSS 单对象 PUT 上传并做上传后校验
    - 失败重试时已可复用既有 OSS upload session，避免重复跑 `upload/init + get_token`
    - OSS 单对象 PUT 失败/成功会按统一 upload checkpoint 口径回填 `uploadId / partCount / uploadedParts / failedPartNumber / nextPartNumber`
    - 当前已不再停留在“只暴露会话但不上传”的失败态
  - `189cloud` 已接入真实 `ValidateAuth / List / Metadata / CreateDir / FastUploadCheck / Upload`
    - 当前读链路基于 `shareCode / accessCode` 的 share API
    - `CreateDir` 已接入账号级 `AccessToken / Signature / Date` 写鉴权路径
    - `Upload` 已接入真实 `getSessionForPC -> createUploadFile -> fileUploadUrl PUT -> getUploadFileStatus -> fileCommitUrl` 主链
    - hash 命中时可直接走 provider 侧复用并由 commit XML 确认
    - hash miss 时会继续走 binary PUT fallback，并由状态轮询与 commit XML 回包做校验
    - 失败重试时已可复用既有 `uploadFileId + fileUploadUrl + fileCommitUrl`，避免重复 `createUploadFile`
    - binary PUT fallback 失败/成功会按统一 upload checkpoint 口径回填 `uploadId / partCount / uploadedParts / failedPartNumber / nextPartNumber`
  - `guangya` 已接入真实 `ValidateAuth / List / Metadata / CreateDir / FastUploadCheck`
    - 当前目录链路基于 Guangya live HTTP API
    - `FastUploadCheck` 已接入真实库存预检与 GCID follow-up / 任务清理
    - `Upload` 已接入真实上传主链：
      - `fast_upload` 命中时可在 Go 内完成 provider-side 确认与上传后校验
      - 小文件已接入 `upload_token + md5 + upload_info`
      - 大文件已接入 `upload_token + GCID flash-check + OSS multipart + upload_info`
      - 大文件 fallback 失败后已可携带 `uploadId + uploadedParts + failedPartNumber + nextPartNumber` 做分片续传
      - 上传后优先按 `fileId` 校验，再回退到父目录按文件名确认
      - 同名冲突当前会诚实降级为 `auto_rename_new`

### 2.1 Provider 辅助调试接口

- 已完成：
  - `POST /api/providers/{key}/list`
  - `POST /api/providers/{key}/metadata`
  - `POST /api/providers/{key}/create_dir`
  - `POST /api/providers/{key}/fast_check`
  - `POST /api/providers/{key}/upload`
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
  - `fast_upload` 先走 `FastUploadCheck` 的 runtime 预检
  - 预检未命中时优先在 runtime 层直接回退到 `download_upload`
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
  - 源端删除记录数量与样本
  - 最近待补传树
  - 当前重试是否已缩小为待补传子集
  - 当前任务结果证据已可直接看到：
    - `fastCheck`
    - `fallbackUsed`
    - `fallbackFrom`
    - `uploadCheckpoint`
  - 但原始项目要求的这些通用能力还未完整落地：
    - 更完整的后台补传调度与队列策略
    - upload-session 恢复之外的更完整后台自动重试编排
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
  - runtime 已按 `requestIntervalMs` 和 `directoryIntervalMs` 执行基础节流，并在 result payload 写入 `throttle` 证据
- 当前状态：
  - 已形成第一版统一风控档位模型，并进入 planner / task metadata
  - 已支持覆盖：
    - 请求间隔
    - 页面大小
    - 目录间隔
    - 冷却时间
    - 重试次数
    - 最大并发
    - 自动补传时间窗
    - 风险关键词
  - 已支持在结果与状态证据里看到：
    - `riskHit`
    - `riskHitCount`
    - `lastRiskStatus`
  - 默认模板已从单一档位扩展为“档位基线 + provider 校准”：
    - 对 `baidu_netdisk / quark / uc / 189cloud / 115_open / guangya` 等风控更敏感链路自动降低 pageSize、加大请求/目录间隔和冷却时间
    - `custom` 模式仍保留给调用方完全手动覆盖
  - planner 现在还会保留完整 `riskProfileResolution`：
    - `base`
    - `calibrated`
    - `applied`
    - `calibrationReasons`
    - `overrideFields`
  - 当前控制台任务向导已提供表单化任务级覆盖入口，可直接填写请求间隔、目录间隔、分页大小、冷却时间、重试次数和风险关键词，并同步生成 `riskOverride` JSON
  - 任务预览和任务详情会同步展示“风险模板解释”，方便直接看出最终节流值来自 provider 校准还是任务 override
  - 单机自动补传调度现在还会尊重 `autoRetryStartHour / autoRetryEndHour`
    - 不在时间窗内时，worker 不会自动接管
    - 但用户仍可手动 Retry 或手动触发后台补传
  - 目前仍缺少：
    - 更多真实样本反推后的细分账号级模板
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
  - task result 中透传 provider 上传证据：
    - `upload`
    - `fileId`
    - `uploadId`
    - `partCount`
    - `rapidUpload`
    - `uploadedParts`
    - `failedPartNumber`
    - `nextPartNumber`
  - runtime / retry queue / provider probe / provider status 中已挂接上传恢复检查点：
    - `uploadCheckpoint.itemPath`
    - `uploadCheckpoint.uploadId`
    - `uploadCheckpoint.uploadedPartCount`
    - `uploadCheckpoint.failedPartNumber`
    - `uploadCheckpoint.nextPartNumber`
    - `uploadCheckpoint.uploadedParts`
    - `uploadCheckpoint.providerData`
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
  - 当前 `completionKind` 已不再一律写成 `probe_only`
  - 当前会按结果语义区分：
    - `real_transfer`
    - `candidate_only`
    - `probe_only`
  - 当前已支持把 `pending_manual` 类结果聚合成待补传树，并回写到 runtime / probe / provider status
  - 待补传树的根节点顺序会优先遵循 `selectedRoots`，避免结果返回顺序打乱用户勾选顺序
  - 当前已支持在 `Retry` 时自动缩小到待补传子集，并按新 plan 继续执行
  - 当前也支持通过 `paths + scope` 把任务显式缩成用户选定的 retry 子集
    - `selected_pending_subset`
    - `selected_retry_subset`
  - 当前 `Retry` 新任务已保留 `retryUploadCheckpoints` 元数据，并把首个恢复检查点带回 runtime 视图
  - 当前 `Retry` 运行时已会把恢复检查点继续传给 provider upload request，供真实 provider 复用 `uploadId / fileId / nextPartNumber`
  - 当前已支持把失败结果分类成重试队列，并区分：
    - `pending_manual`
    - `rate_limited`
    - `auth_expired`
    - `local_file_missing`
    - `retry_limit_exhausted`
  - 当前重试队列项已可直接携带上传恢复线索，便于后续补传和排查时不用重新翻 task result 原始 payload
  - `aliyundrive_open` 当前已接入 `POST /v2/file/get_upload_url`，可基于已有 `uploadId` 刷新剩余分片上传地址并继续上传
  - 当前 `rate_limited` 会按冷却时间阻断过早重试
  - 当前任务在“只有冷却 / 人工确认 / 授权失效 / 本地文件缺失”时会进入 `blocked`
  - 当前 runtime / probe / status 已回写 `blockedReason` 与 `nextRetryAt`
  - 当前已接入单机 worker 版后台自动补传调度，启动时会先恢复一次，随后按 `CLOUDPAN_AUTO_RETRY_TICK` 自动检查候选任务
  - 当前自动补传还新增了批次上限控制：
    - 默认按 `CLOUDPAN_AUTO_RETRY_BATCH_LIMIT` 控制单次 tick 最多接管多少条候选任务
    - 自动 tick 默认还会附带 `CLOUDPAN_AUTO_RETRY_LIMIT_PER_MODE / _PER_LANE / _PER_PROTOCOL_GROUP / _PER_PROVIDER / _PER_PROFILE` 五级公平预算
    - 避免高失败期一次性把整池候选全部打满，也避免单一模式、协议族、provider 或账号连续吃满整轮
  - 当前 `metadata.retrySummary` 已细化为：
    - `retryableNowCount / cooldownCount / pendingManualCount / authExpiredCount / localMissingCount / exhaustedCount`
    - `uploadCheckpointEligible`
    - `autoRecoverEligible / autoRecoverMode / autoRecoverAdvice`
    - `windowBlocked`
  - 当前会明确区分四类后台补传语义：
    - `cooldown_elapsed_auto_retry`
    - `upload_checkpoint_auto_resume`
    - `retry_queue_auto_retry`
    - `retry_window_waiting_auto_retry`
  - 当前 worker 的优先级顺序也已经固定：
    - 先处理 `upload_checkpoint_auto_resume`
    - 再处理 `retry_queue_auto_retry`
    - 再处理 `retry_window_waiting_auto_retry`
    - 最后才处理 `cooldown_elapsed_auto_retry`
- 当前 `/api/evidence/runtime` 与 `/api/status/providers` 已新增 `autoRecoverTasks / autoRecoverPool`
- 当前 `/api/evidence/runtime` 与 provider 状态快照还会额外聚合后台补传候选池的全局等待态拆分：
  - `autoRecoverRunnableTasks`
  - `autoRecoverWaitingCooldownTasks`
  - `autoRecoverWaitingRetryWindowTasks`
  - `autoRecoverWaitingOtherTasks`
- 当前 `autoRecoverPool` 的 lane 内部也已经细分出更明确的等待态计数：
  - `waiting_auth_refresh`
  - `waiting_local_restore`
  - `waiting_manual_confirmation`
  - `waiting_retry_limit`
  - 其中 `waiting_other` 现在只表示剩余未细分等待态，不再把上述几类硬阻塞都混在一起
- 当前 `/api/evidence/runtime` 还会直接返回自动 worker 的默认调度策略摘要，能看到 `tick / batchLimit / limitPerMode / limitPerLane / limitPerProtocolGroup / limitPerProvider / limitPerProfile`
    - 可直接看出哪些任务已经进入后台补传候选池
    - 也能看出每种模式的任务数、provider 数、queue 大小、冷却量和 checkpoint 量
  - 当前还新增 `POST /api/tasks/recover`
    - 可按 `mode`
    - 可按 `providerKey`
    - 可按本轮 `limit`
    - 便于联调时手动放行一小批后台补传候选，而不是等下一次自动 tick
  - 当前自动恢复会在 runtime / result / provider probe / provider status 中留下 `autoRecovered / autoRecoverReason / autoRecoverCount / autoRecoveredAt / autoRecoverState` 证据，便于判断任务是用户手动重试还是 worker 自动续跑
  - 当前 `retryLimit` 已真正接入重试队列，支持累计次数、剩余次数与 exhausted 阻断
  - 当前 `blockedReason` 已补充统一的 `blockedAction / blockedAdvice`，便于状态矩阵直接给出处理建议
  - 当前 `/api/evidence/runtime` 与 `/api/status/providers` 已补充 `blockedTasks / blockedActions` 聚合摘要，便于快速定位最需要人工处理的动作
  - 当前任务详情页已经支持把“当前筛选结果”直接变成 retry 范围：
    - 重试当前待补传树筛选结果
    - 重试当前重试队列筛选结果
  - 当前任务元数据也会直接保留：
    - `retryScope`
    - `retrySelectedPaths`

### 6. Runtime Evidence / 状态矩阵

- 已完成：
  - 最近任务结果聚合
  - 最近 provider probe 聚合
  - provider 状态快照聚合
  - `/api/evidence/runtime`
  - `/api/status/providers`
  - `protocolCoverage` 协议族覆盖矩阵
  - `providerSmokeSummary` 真实样本聚合
  - `providerSmokeMatrix` 真实样本矩阵
  - `accepted / pending` 真实联调验收判定
  - `acceptanceMissing` 缺失原因提示，能直接看出还差真实 smoke、任务覆盖或两者都缺
  - `acceptanceAdvice` 补齐建议，能直接指导下一步补样本
  - 状态页证据摘要里会直接显示 accepted / in_progress / pending 协议组计数
  - 验收矩阵支持按 accepted / in_progress / pending 快速筛选，并可直接打开对应 smoke 样本或样本任务
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
  - `sourceDeletionCount`
  - `sourceDeletionRecords`
  - `pendingTree`
  - `riskHitCount`
  - `lastRiskStatus`
  - `autoRecovered`
  - `autoRecoverReason`
  - `autoRecoverCount`
  - `autoRecoveredAt`
  - `autoRecoverState`
  - `currentRoot`
  - `currentDirectory`
  - `lastCompletedPath`
  - `protocolCoverage`
- 当前状态：
  - 已具备联调、排错、演示所需的最小证据链
  - 现在还能直接看出每个协议族是否已经至少沉淀出一条真实成功样本
  - `accepted` 表示“真实上传成功样本 + 该协议组已有任务覆盖”，`pending` 表示还缺其中一项或两项

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
    - 重试队列分类展示
    - 目录状态树展示
    - 当前根目录 / 当前目录 / 上次完成路径展示
    - 运行中的任务现在会先进入 `pause_requested`，等当前 item 收尾后再真正落为 `paused`
    - 如果在当前 item 结束前点了恢复，会直接撤销本次暂停请求并继续跑后续 item
  - 待补传树展示
  - 目录树 / 待补传树按 root 分组收起 / 展开
  - 目录树 / 待补传树分组状态本地持久化
  - 按路径 / 状态 / reason 的树筛选
  - 按 retry class / retry state 的重试队列筛选
  - 运行中的任务可在当前 item 完成后协作式暂停，并在恢复后继续后续 item
  - 任务预览和任务详情都能直接看见 `Selected Roots`，运行后还能查看 `Scan Trace`
  - 任务摘要里的 `Selected Roots` / `Scan Trace` 也做成了可点击路径芯片，方便直接跳到目录树并沿叶子优先顺序排查
  - `Selected Roots` 和 `Scan Trace` 里的路径按钮可直接定位目录树，任务页和状态页都能用，方便按顺序排查执行轨迹
  - 验收报告里的代表任务样本也会带上 `Selected Roots` 和 `Scan Trace`，方便离线分享
  - 真实 provider smoke 记录的协议组聚合与真实样本矩阵展示
  - 真实联调验收判定层，能直接看出哪些协议组已经 accepted、哪些仍在 pending
  - pending 行会带上 `acceptanceMissing` 缺失原因，便于直接补样本
  - pending / in_progress 行还会带上 `acceptanceAdvice`，提示下一步应补真实 smoke 还是任务覆盖
  - 首页证据摘要会同步显示 accepted / in_progress / pending 的数量，便于一眼判断收口进度
  - 验收矩阵可以按状态筛选，并直接打开对应 smoke 样本或样本任务，方便继续补齐真实联调材料
  - 验收矩阵现在还能一键按协议组预填 smoke 表单，并把下方 smoke 记录收敛到当前协议组，补样本时不用再手动回填一遍字段
  - 任务详情、运行检查点和状态快照都已同步展示 `后台补传候选 / 队列拆分 / 自动补传提示`，便于直接判断当前失败队列更适合冷却后自动重试、等待自动补传时间窗、upload checkpoint 续跑，还是人工处理
- 运行证据摘要页还新增了“自动补传候选池”，可以按模式直接看 worker 当前会优先接管哪一类任务，或哪些任务虽然已满足自动补传条件但仍在等待允许时间窗
- 候选池现在还会额外拆出“可立即执行 / 等冷却 / 等时间窗 / 其它等待”四类任务数，避免看到候选后却不知道为什么此刻没有真正开始跑
- 自动补传候选池现在还能直接按主重试类型、主阻塞动作或 lane 级口径一键聚焦与执行，减少手动回填筛选条件
- 状态页当前还新增了“预演当前筛选”和 lane 级“预演该 lane”入口，可直接复用当前筛选或建议预算先做 dry-run 试算
- 状态页现在还会把最近一次预演或执行的 `decisions` 逐条列出来，便于直接看见是哪些任务已放行，哪些任务被 `limit / providerBudget / profileBudget / retryWindow / cooldown` 等口径挡住
- 状态页手动放行还可直接按 `recoverState` 先收敛到“只放行当前能跑的 / 只看等冷却 / 只看等时间窗 / 只看其它等待”，也可继续填写 `limit / limitPerMode / limitPerLane / limitPerProtocolGroup / limitPerProvider / limitPerProfile`
  - 当前 `recoverState` 还支持更细的硬阻塞筛选：
    - `waiting_auth_refresh`
    - `waiting_local_restore`
    - `waiting_manual_confirmation`
    - `waiting_retry_limit`
  - 便于把状态页操作直接收敛到“等授权刷新 / 等补回本地文件 / 等人工确认 / 等重置重试策略”这四类任务
  - 当 `path / paths` 为空时，后台补传 API 不再误把空筛选当成根路径筛选
  - 对当前命中但暂时不能执行的候选，状态页放行结果也会显式区分 cooldownWait / retryWindowWait / blocked，不再把这类跳过静默吞掉
    - 便于把补传节奏同时控制在“小批次 + 模式分流 + lane 分流 + 多账号轮转”
  - 任务目录树和待补传树节点现在也能直接触发“后台补传当前路径 / 当前 root”，方便只放行当前子树
  - 当前任务详情页的“重试队列”和“待补传树”还新增了“后台补传当前筛选”入口
    - 会把当前筛选可见路径直接整理成 `paths + scope`
    - 可一次放行多棵子树，而不是只能单路径逐个点
  - 当前目录树也能直接基于“当前筛选”做两类动作：
    - `重试当前筛选`
      - 以 `selected_directory_subset` 把当前目录子树直接缩成独立执行子集
    - `按当前筛选重建向导`
      - 直接把当前目录树或待补传树筛选结果预填回任务向导
  - 当前状态页里的“最近重试队列”和“最近待补传树”也已接上“重试当前筛选 / 后台补传当前筛选”
    - 默认作用于最近 probe / snapshot 对应的样本任务
    - 配合 `taskId` 精准约束，可直接从状态页按当前样本继续排障
  - 目录树筛选现在也支持“仅叶子节点”视角，便于只看当前最末层目录推进顺序
  - 从重试队列一键定位待补传树或收敛到同类失败项
    - 从目录树节点直接“按当前路径重建向导”
    - 从待补传树节点直接“重试当前路径”
    - 从 blocked 聚合摘要一键跳到样本任务或当前阻塞动作
    - 从任务详情 blocked 引导卡片一键收敛当前任务问题子集
    - 待补传树“仅叶子节点”视角
    - 目录树 / 待补传树筛选支持一键清空，方便从路径定位后快速回到全量视图
    - 每个 root 分组和节点还支持“只看当前路径 / 同步另一棵树”，可以在目录树和待补传树之间快速来回对照
    - 当前筛选结果还能一键复制成路径清单，方便做联调记录、补传排查和外部样本整理
    - 节点级子树“收起 / 展开”与状态持久化，不再只限 root 分组
    - 目录树筛选新增“仅问题节点”视角，可快速聚焦 `pending / running / blocked / failed / 未处理完` 的目录
    - 节点级“复制当前子树”与“只看父级”，方便把某棵子树路径清单导出或逐层回退排查
    - 筛选摘要会同步显示 root 数、leaf 数、最大深度、最深路径和问题节点数，便于判断当前视图规模
  - 后续仍需继续补：
    - 真正异步 worker 下更完整的暂停 / 恢复编排体验

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
  - `aliyundrive_open` 真实 `ValidateAuth + List + Metadata + CreateDir` 最小目录链路

## 当前仍未完成的部分

### 1. Provider 真实样本与边界验收仍不完整

- 现在 10 家 provider 已全部进入真实主链路阶段，不再是只有 placeholder。
- 当前仍需继续补的是“可验收样本”和“边界场景证据”，而不是从零接 provider：
  - 每家 provider 的真实账号 smoke 记录
  - 限流、风控、授权过期、文件缺失、覆盖降级等异常样本
  - 大文件、深目录、重试恢复等长链路样本
- 还缺：
  - 真实登录态续期
  - 更细粒度断点续传
  - 更多真实验收记录与边界覆盖

### 2. UI 自动化 smoke 仍可继续增强

- 当前已经补上仓库内可重复执行的浏览器级 smoke 主流程回归
- 后续仍可继续增强：
  - 更多异常场景 UI 提示校验
  - 多 provider 组合样本
  - 截图或录屏级产物沉淀

### 3. “每个协议族至少一条真实成功样本” 还未完成

- 当前统一内核已经搭好，10 家 provider 也已经进入真实链路阶段。
- 真正的切换验收还需要真实 auth profile 和真实链路样本，尤其是跨协议族的成功样本矩阵。
- 现在已经补上 accepted / pending 真实联调验收判定层，但 accepted 协议组数量仍在继续增长中。

### 4. 对外分享说明还可以继续增强

- 当前仓库已经能看懂并启动
- 但后续仍建议继续补：
  - 按模板逐步沉淀真实 provider 样本记录

### 5. 执行模型仍未补成完整主能力

- 当前还缺：
  - 更复杂的后台调度编排
    - 当前已经补上候选池里的“可立即执行 / 等冷却 / 等时间窗 / 其它等待”拆分，但更细的跨 provider 长时调度策略仍需继续收口
- 当前控制台已支持预览后展示推荐执行模式、推荐原因，并可一键采用推荐模式

### 6. 风控策略层仍未独立成型

- 当前还缺：
  - 真实 provider 样本校准后的更细默认模板
  - 更多真实样本反推后的 provider / 账号级限流模板
- 当前控制台已支持表单化任务级风控参数配置，并可同步生成 `riskOverride` JSON

## 当前最清晰的完成度判断

- 如果按“工程能否独立运行、独立测试、独立演示”来判断：
  - 已经完成
- 如果按“10 家 provider 是否都已经真实联网可用”来判断：
  - 还没有完成
- 如果按“是否已经进入真实联调阶段”来判断：
  - 已经进入，而且 `aliyundrive_open / 123_open / xunlei / pikpak / baidu_netdisk / 115_open / quark / uc / 189cloud / guangya` 都已经进入真实链路样本阶段
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
