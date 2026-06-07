# 二期进度总览（Phase 2 Progress）

## 说明

- 本文用于跟踪 [08-PHASE2_TODO.md](/E:/Workspace/VSCode/CloudPan-Sync-go/docs/08-PHASE2_TODO.md:1) 的实际落地进度。
- 进度判断以当前仓库实现和本地提交记录为准，不按口头预期估算。
- 当前分支最近相关提交可见：`029c960` 到 `f2924fd`，后续本地提交继续顺延。

## 当前结论

- 二期没有“完全做完”。
- 主线三目前推进最多，主线二次之，主线一完成了模板、矩阵提示和草稿预填，但距离“真实样本全部补齐”还有差距。
- 如果按三条主线整体粗看，当前更接近“已完成基础收口和一批关键增强”，还不适合标记为全部完成。

## 主线一：真实样本矩阵补齐

### 已完成

- 已补 smoke matrix 的异常样本提示与缺口提示。
- 已把大文件、多层目录、重试恢复三类代表性样本缺口显式展示到 smoke matrix，并补齐对应测试。
- 已支持按异常样本缺口预填 smoke 草稿。
- 已支持按代表性样本缺口直接预填 smoke 草稿，便于继续补大文件、多层目录、重试恢复样本。
- 已沉淀固定 smoke 记录模板，便于后续复用。
- 已把固定 smoke 记录模板补上代表性样本维度与自动补传/公平性关注点，便于后续真实样本统一记法。
- 已优化控制台里的样本来源语义，并区分 `accepted / in_progress / pending`。
- 已把真实样本矩阵补成可量化的补齐总览，直接显示各协议组异常样本与代表性样本已完成数量，便于更快判断还差哪些证据。
- 已把真实样本矩阵补成可读 checklist 摘要，直接归并显示 upload / coverage / anomaly / representative 四类样本完成状态。
- 已把真实样本矩阵补成缺口速览，直接压缩显示每个协议组还缺哪些 upload / anomaly / representative 具体样本类型。
- 已把真实样本矩阵补成“下一步补样动作”摘要，直接提示每个协议组此刻最值得优先补的 upload / anomaly / representative 动作。
- 已把真实样本矩阵补成“首要补样项”摘要，只保留当前最应该先补的一项，方便先补最关键验收证据。
- 已把真实样本矩阵补成“补齐完成度单值摘要”，直接给出 `pending / partial / ready` 判断，便于更快评估每个协议组离样本补齐还有多远。
- 已补 provider 级真实样本验收矩阵，覆盖 catalog 中所有 provider，直接展示基础成功样本、上传成功样本、异常样本覆盖、缺口与首要补样动作。
- 已把 provider 级真实样本验收矩阵接入验收报告页摘要，直接显示 `Provider Ready`、`ready / partial / pending` 数量和未 ready provider 的首要补样动作，避免验收时只能翻 Markdown。
- 已把固定记录模板里的 `templateVersion / sampleType / evidenceCompleteness / reuseAdvice / representativeLabels / autoRecoverFocus` 结构化返回到 smoke 记录接口与列表视图，便于不打开 Markdown 也能直接判断样本是否值得复用和继续补齐。
- 已把固定记录模板里的“推荐回归入口”从占位语义收口为自动生成的操作链路；即使未手填 operations，也会按样本类型、备注关键词和失败语义推导出 `ValidateAuth / List / FastUploadCheck / Metadata / Upload / checkpoint / blocked_recovery` 组合，便于后续按样本直接回放。
- 已把固定记录模板里的推荐回归入口继续收口成结构化 `regressionEntry` 字段，并接到 smoke 记录列表展示与筛选，便于直接按回放链路查找可复用样本，而不用再打开 Markdown 逐条翻。
- 已把 smoke 记录列表筛选继续扩展到 `sampleType / evidenceCompleteness / reuseAdvice / regressionEntry / representativeLabels / autoRecoverFocus` 等结构化字段，补样和回归时可以直接按“样本类型、复用建议、代表性标签、自动补传关注点”搜历史样本。
- 已把 smoke 样本的“可复用程度”继续收口成结构化 `reusePriority` 字段，直接区分 `直接回归 / 条件复用 / 参考样本`，并接到列表展示与检索，便于更快挑出最适合优先复跑的样本。
- 已把协议组汇总里的“首选回归样本”继续收口成 `preferredSampleRecordId / preferredSampleTitle / preferredSampleProvider / preferredSamplePriority` 摘要，直接指向当前最值得先复跑的一条真实样本，而不只是显示最近记录。
- 已把协议组汇总继续细分出 `preferredUploadSample*` 与 `preferredAnomalySample*` 摘要，能直接区分“优先复跑成功链路样本”与“优先复跑异常证据样本”，减少人工翻样本记录的时间。
- 已把协议组汇总继续细分出 `preferredRepresentativeSample*` 摘要，能直接把大文件、多层目录、重试恢复这类代表性样本提升到优先回归入口，便于先补关键边界证据。
- 已把 provider 级真实样本验收继续细分出 preferredSample* / preferredUploadSample* / preferredAnomalySample* 摘要，能直接看出每家 provider 此刻最值得先复核的基础样本、上传样本和异常样本。

### 对应提交

- `d82f1cc` 二期：补齐真实样本矩阵异常样本提示
- `c03ddf5` 二期：按异常样本缺口预填 smoke 草稿
- `bd907f5` 二期：沉淀真实样本固定记录模板
- `494cd9a` 二期：细化样本来源推荐语义
- `d8572ac` 二期：区分已验收与待补齐样本推荐
- `f2924fd` 二期：补齐 provider 级真实样本验收矩阵
- 当前轮次已新增 provider 级真实样本验收报告页摘要，用于把 provider 维度缺口从 Markdown 提升到页面可视摘要。
- 当前轮次已新增固定记录模板结构化字段回传与列表展示，用于把样本类型、证据完整度和复用建议从 Markdown 提升到可筛查的日常视图。
- 当前轮次已新增固定记录模板“推荐回归入口”自动推导与测试兜底，用于把未填写 operations 的真实样本也收口成可直接复放的回归清单，而不是继续停留在“待补充操作清单”。
- 当前轮次已新增 `regressionEntry` 结构化字段、前端列表展示与筛选，以及 task/web 测试兜底，用于把“回归入口”从 Markdown 内文进一步提升到日常排障可直接筛查的接口字段。
- 当前轮次已把 smoke 列表查询同步扩展到结构化样本字段，并补 workflow/web 断言兜底，用于直接按复用建议、代表性标签和自动补传关注点检索历史样本。
- 当前轮次已新增 `reusePriority` 结构化字段和 task/app 测试兜底，用于把“这条样本更适合直接回归还是只作参考”从经验判断提升成可直接返回和展示的统一口径。
- 当前轮次已新增协议组级 preferred sample 摘要和 task/app/web 测试兜底，用于把“先拿哪条样本回归”从人工判断提升到汇总视图可直接查看的统一口径。
- 当前轮次已新增 `preferredUploadSample* / preferredAnomalySample* / preferredRepresentativeSample*` 摘要与 task/app/web 测试兜底，用于把协议组样本进一步拆成“首选上传样本”“首选异常样本”和“首选代表性样本”三类优先证据。
- 当前轮次已新增 provider 级 preferredSample* / preferredUploadSample* / preferredAnomalySample* 摘要与 task/app/web 测试兜底，用于把 provider 维度的优先复核样本直接提升到验收报告和页面摘要。

### 仍未完成

- 还不能证明“每个协议族都至少有 1 条真实上传成功样本”。
- 还不能证明“每家 provider 都补齐了基础成功样本和最小异常样本”。
- 大文件、多层目录、重试恢复这三类样本现在已经能在矩阵里明确看出缺哪类，但真实样本沉淀本身还没有形成一份全部补齐的完成清单。

### 当前判断

- 这一主线属于“框架、提示和缺口定位工具已补好，但真实样本内容本身还没补完”。

## 主线二：执行模型与后台补传继续收口

### 已完成

- 已补恢复预算提示与展示。
- 已增强上传检查点恢复线索展示。
- 已补齐续传恢复线索透传。
- 已增强筛选协同动作反馈。
- 已把“等待冷却”“等待时间窗”“等待重建 provider 会话”从恢复态里进一步拆开。
- 已把 provider 会话缺失从人工确认/重试耗尽语义里独立出来。
- 已补路径子集重试下的 upload checkpoint 过滤与续传上下文回归覆盖，可证明大文件/长链路在子树级补传时不会把未选路径一并重跑，也不会丢掉既有 checkpoint。
- 已把自动补传候选池的协议族 / provider / profile 覆盖范围与建议预算写入证据报告，便于直接核对公平性收敛情况。
- 已把自动补传公平性摘要补上 sample provider / protocol group / profile / strategy 上下文，便于直接核对每条 lane 的真实样本落点。
- 已把自动补传公平性摘要补成可读完成判断，直接说明当前候选池是否已出现多 provider / 多账号 / 多协议组分散证据。
- 已把 upload checkpoint 任务数、自动续传任务数和样本路径写入运行证据与报告，便于直接核对续传恢复是否形成稳定证据。
- 已把 upload checkpoint 自动续传样本的 provider / protocol group / task / profile 上下文写入运行证据与报告，便于直接回看具体续传样本落点。
- 已把 upload checkpoint 自动续传样本的 uploadId / nextPart / uploadedParts 进度写入运行证据与报告，便于直接核对是否从既有分片状态继续恢复。
- 已把 upload checkpoint 自动续传样本补成稳定性摘要，直接说明当前是否已经具备“从既有 checkpoint 继续恢复”的关键证据。
- 已把 upload checkpoint 自动续传证据补成 `pending / partial / ready` readiness 判断，报告和状态摘要都能直接看出离“默认恢复能力”还差多远。
- 已把 upload checkpoint 默认恢复 readiness 接入验收报告页摘要，直接展示 `Checkpoint Resume Ready`、自动续传任务数、样本上下文、分片进度和首要恢复动作，便于不用翻 Markdown 就能核对大文件/长链路恢复证据。
- 已补 legacy upload checkpoint 缺少 `itemPath` 时的请求路径兜底，避免历史 checkpoint metadata 恢复时带空路径进入 provider resume 链路。
- 已把运行证据补成“自动补传首要动作”摘要，直接指出当前最该先处理的恢复阻塞或补样方向，便于继续减少人工介入。
- 已把运行证据补成“自动补传恢复完成度”摘要，直接给出 `pending / partial / ready` 单值判断，便于更快评估离默认稳定恢复还有多远。
- 已把自动补传公平性补成“完成度单值摘要”，直接给出 `pending / partial / ready` 判断，便于更快核对多 provider / 多账号候选池是否形成分散证据。
- 已把自动补传公平性补成“首要动作摘要”，直接指出当前应优先补多 provider、多账号或多协议组候选池样本，便于继续收口公平性验证。
- 已把自动补传恢复与公平性摘要接入验收报告页，直接显示 `Recover Ready`、`Fairness Ready`、首要动作、等待态分布和 lane 样本上下文，便于不用翻 Markdown 或状态页也能核对主线二验收情况。
- 已把选择路径重试的路径数量补成结构化 `retrySelectedPathCount`，并透传到任务 metadata、运行结果、recent probe、provider 状态摘要、验收报告和页面展示，便于直接核对路径级/子树级补传范围。
- 已把自动补传公平性验收补成结构化 `autoRecoverFairnessReadiness / autoRecoverFairnessMissing / autoRecoverFairnessPriorityAction`，并透传到运行证据、provider 状态、验收报告和页面展示，便于直接看出还缺多 provider、多账号还是多协议组候选池证据。
- 已把重试链路里的 upload checkpoint 数量补成结构化 `retryUploadCheckpointCount`，并透传到任务 metadata、运行结果、recent probe、provider 状态摘要、验收报告和页面展示，便于直接核对本轮 retry 是否带着 checkpoint 恢复上下文继续执行。
- 已把 upload checkpoint 默认恢复验收补成结构化 `uploadCheckpointResumeReadiness / uploadCheckpointResumePriorityAction`，并接入运行证据、验收报告和页面展示，便于直接判断还缺失败样本、自动续传样本还是 uploadId/分片进度证据。

### 对应提交

- `70c7d22` 二期：补齐恢复决策预算占用提示
- `4fda64b` 二期：补齐恢复决策预算占用提示展示
- `3df639a` 二期：增强上传检查点恢复线索展示
- `1651c91` 二期：补齐续传恢复线索透传
- `4d5e09c` 二期：增强筛选协同动作反馈
- `5efc7e3` 二期：细化自动补传时间窗等待态
- `d9611bc` 二期：拆分 provider 会话重建等待态
- 当前轮次已新增“selected retry subset 保留 checkpoint context”专项回归测试，用于补强子树级补传与 upload checkpoint 续传组合边界的稳定性证明。
- 当前轮次已新增 upload checkpoint 默认恢复验收报告页摘要，用于把大文件/长链路恢复 readiness、样本上下文和首要动作从 Markdown 提升到页面可视摘要。
- 当前轮次已新增 legacy checkpoint `itemPath` 兜底回归测试，用于证明历史 retryUploadCheckpoints metadata 仍能按请求路径恢复上传会话。
- 当前轮次已新增自动补传恢复与公平性验收报告页摘要，用于把恢复完成度、公平性完成度、首要动作和 lane 样本上下文提升到统一验收视图。
- 当前轮次已新增 `retrySelectedPathCount` 透传与 task/app/web 测试兜底，用于让选择路径重试的范围不再只能靠人工展开 `retrySelectedPaths` 数组确认。
- 当前轮次已新增自动补传公平性结构化缺口字段与 app/web 测试兜底，用于把公平性验收从纯文本摘要推进到接口可直接判断的稳定字段。
- 当前轮次已新增 `retryUploadCheckpointCount` 透传与 task/web 测试兜底，用于让 upload checkpoint 续传上下文数量从内部 metadata map 提升成可直接查看的稳定字段。
- 当前轮次已新增 upload checkpoint 默认恢复结构化 readiness / priority 字段与 task/app/web 测试兜底，用于把大文件/长链路恢复验收从报告文案进一步推进到接口稳定字段。

### 仍未完成

- 还不能证明“大文件失败后的恢复路径已经足够稳定，可当默认能力使用”。
- checkpoint / resume 虽然已经明显增强，但还不等于所有大文件长链路都稳定。
- 多 provider / 多账号下的后台补传公平性，还缺少更明确的完成证据或专项验证记录。

### 当前判断

- 这一主线属于“核心机制已经有了，并且可观测性提升明显，但离完全收口还差最后一段稳定性验证和边界补齐”。
- 相比上一轮，子树级/路径级补传与 checkpoint 续传的组合边界已经多了一条明确回归证明。

## 主线三：风控模板按真实样本校准

### 已完成

- 已补协议族级风控模板校准。
- 已补账号级恢复预算建议展示。
- 已将样本结论继续同步回控制台推荐语义。
- 已区分 smoke matrix 已验收与待补齐样本对推荐语义的影响。
- 已把当前代码中的 provider / protocol group 默认风控模板整理成可核对清单，并补了契约测试锁定关键默认值与默认无时间窗语义。
- 已把 provider 默认模板里的 `auto retry window` 来源语义显式返回到接口与控制台，便于区分“代码默认空窗口”与“账号/任务覆盖注入”。
- 已把账号默认风控的真实样本来源与建议同步展示到 provider 卡片，便于直接核对 smoke matrix 预填来源。
- 已把授权档案列表同步展示账号默认风控来源与建议，便于不打开编辑弹窗也能直接核对账号级默认值依据。
- 已把 provider 默认风控模板补成“校准完成度/缺口”摘要，能直接看出哪些字段已由代码默认覆盖、哪些仍主要依赖账号默认或任务覆盖。
- 已把 provider 默认风控模板补成“首要校准项”摘要，直接指出当前最该优先补的默认字段，便于继续按真实样本收口。
- 已把 provider 默认风控模板补成“校准完成度单值摘要”，直接给出 `pending / partial / ready` 判断，便于更快评估离默认可用模板还有多远。
- 已把 provider 默认风控模板校准摘要接入验收报告页，直接显示 `Calibration Ready`、`priority calibration`、`calibrationMissing` 和 `auto retry window` 来源/建议，便于不用翻 provider 卡片也能核对默认模板缺口。
- 已把 provider 默认风控模板校准覆盖度补成结构化 `calibrationCoveredCount / calibrationTargetCount / calibrationMissingCount`，并接入接口、验收报告和页面展示，便于不用解析 `partial 7/8` 文本也能直接判断校准进度。
- 已把 provider 默认风控模板校准字段清单补成结构化 `calibrationCoveredFields / calibrationTargetFields`，并接入接口、验收报告和页面展示，便于直接核对 request interval、directory interval、retry limit、risk keywords、auto retry window 等字段到底覆盖了哪些。

### 对应提交

- `029c960` 二期：补齐协议族级风控模板校准
- `f243023` 二期：展示账号级恢复预算建议
- `494cd9a` 二期：细化样本来源推荐语义
- `d8572ac` 二期：区分已验收与待补齐样本推荐
- 当前轮次已新增 provider 风控模板清单与默认值契约测试，便于核对 `request interval / directory interval / retry limit / risk keywords` 当前到底落成了什么。
- 当前轮次已新增 provider 默认风控校准验收报告页摘要，用于把校准 readiness、缺失字段清单、首要校准项和自动补传时间窗来源从局部卡片提升到统一验收视图。
- 当前轮次已新增 provider 默认风控校准覆盖计数字段与 planner/app/task/web 测试兜底，用于把校准完成度从展示文本推进到接口可直接判断的稳定字段。
- 当前轮次已新增 provider 默认风控校准 covered/target 字段清单与 planner/app/task/web 测试兜底，用于把风控模板校准项从计数进一步推进到可直接核对字段名的稳定结构。

### 仍未完成

- 还不能证明“各 provider 的默认节流建议已经根据真实联调样本完整回收”。
- `request interval / directory interval / retry limit / risk keywords` 已经整理成可核对清单，但 `auto retry window` 仍主要依赖账号默认或任务级覆盖，尚未沉淀成一份真实样本驱动的完整默认表。
- 高风险 provider 的账号级建议虽然已有一部分，但未看到一份“全部补齐完成”的验收结果。

### 当前判断

- 这一主线属于“推荐语义和默认模板已经明显进步，但离‘按真实样本完整校准完毕’还有文档化和补齐工作”。
- 相比上一轮，这条主线已经从“代码里有但难核对”推进到“仓库内可直接核对当前默认模板”，但真实样本驱动的最终校准还没完全闭环。

## 建议的进度理解

- 如果按“支撑能力是否已经具备”看：二期已经做出不少实质进展。
- 如果按 [08-PHASE2_TODO.md](/E:/Workspace/VSCode/CloudPan-Sync-go/docs/08-PHASE2_TODO.md:1) 的原始验收标准看：还不能说全部做完。
- 更准确的说法是：
  - 主线一：部分完成，工具和模板已就绪，真实样本内容仍待补齐。
  - 主线二：中后段，恢复态、补传协同和观察能力已经比较完整，但稳定性收口未结束。
  - 主线三：中后段，推荐语义和模板校准已有明显成果，但还未形成“全部校准完成”的闭环证明。

## 下一步建议

- 先继续补主线一真实样本本体，尤其是“每协议族至少 1 条真实成功样本”的缺口。
- 同时继续主线二的大文件恢复稳定性验证，把 checkpoint / resume 从“可用”推进到“默认稳”。
- 最后把主线三里已经散落在代码里的风控经验，整理成一份更容易核对的完成清单。

## 更新方式

- 后续每完成一个二期里程碑，建议同时更新本文，而不是只看 commit 记录。
- 如果需要，也可以再补一个更短的“打勾版”清单，把每条里程碑拆成 `已完成 / 进行中 / 未开始` 三列。
