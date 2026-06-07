# CloudPan Sync Go 真实 Provider 联调记录模板

## 用途

- 用于记录单个 provider 或单个协议族的真实联调结果。
- 目标不是写“过程随笔”，而是沉淀可复核、可交接、可验收的联调证据。
- 建议每打通一条真实链路，就复制本模板生成一份记录。

## 建议命名

- 单 provider：
  - `docs/records/2026-05-27-123_open-real-smoke.md`
- 协议族：
  - `docs/records/2026-05-27-aliyun_123_open-family-smoke.md`

## 基本信息

- 记录日期：
- 记录人：
- 分支 / commit：
- 环境：
  - 操作系统：
  - Go 版本：
  - 服务地址：
- 目标 provider：
- 所属协议组：
- 本次联调目标：
  - 例如：登录校验
  - 例如：目录查询
  - 例如：快传命中
  - 例如：二进制上传成功

## 认证信息摘要

- `authMode`：
- 使用的认证类型：
  - `token`
  - `cookie`
  - `extra`
- `extra` 键摘要：
  - 只记录 key，不记录敏感值
- 凭据来源：
  - 手工登录导出
  - 官方开放平台
  - 浏览器抓取
  - 其他
- 脱敏结论：
  - 是否已确认不会把明文 token/cookie 写入仓库

## 本次覆盖范围

- `ValidateAuth`
- `List`
- `Metadata`
- `CreateDir`
- `FastUploadCheck`
- `Upload`

建议用下面格式标记：

- `ValidateAuth`: 未测 / 失败 / 成功
- `List`: 未测 / 失败 / 成功
- `Metadata`: 未测 / 失败 / 成功
- `CreateDir`: 未测 / 失败 / 成功
- `FastUploadCheck`: 未测 / 失败 / 成功
- `Upload`: 未测 / 失败 / 成功

## 本次最小补样清单

> 如果这份记录要直接推进二期验收，建议至少补齐下面 1 类真实证据；否则就把它明确标成 `browse_only` 或 `partial_blocked`，避免把“看过流程”误写成“补完样本”。

- 真实上传成功样本
  - 目标：证明当前协议族至少有 1 条能稳定落库的真实上传成功证据。
  - 最低要求：`Upload` 成功，且 `completionKind` 能回到 `done` 或等价完成态。
- 最小异常样本
  - 目标：证明 provider 的异常边界不是口头经验。
  - 建议至少覆盖其中 1 项：
    - `auth_expired`
    - `rate_limited`
    - `local_file_missing`
    - `pending_manual_requires_confirmation`
- 代表性样本
  - 目标：把后续最容易回归的边界样本先沉淀下来。
  - 建议至少覆盖其中 1 项：
    - 大文件
    - 多层目录
    - 重试恢复

## 记录判定建议

- `auth_only`
  - 只打通授权校验，还没有形成真实上传样本。
- `browse_only`
  - 只打通目录 / 元数据 / 建目录，还没有真实上传成功样本。
- `fast_upload_success`
  - 已形成真实上传成功样本，且优先可复用快传链路。
- `binary_upload_success`
  - 已形成真实上传成功样本，且优先可复用普通上传链路。
- `partial_blocked`
  - 已有部分样本，但还没达到二期验收最低集合。
- `failed`
  - 本次未形成可复用样本。

## 联调前准备

- 服务启动命令：

```powershell
go run ./cmd/cloudpan-sync
```

- 回归基线：

```powershell
go test ./...
go build ./...
```

- 使用的参考文档：
  - `docs/04-API_WORKFLOW_EXAMPLES.md`
  - `docs/05-PROVIDER_INTEGRATION_GUIDE.md`
  - 如适用，记录本次参考的官方 API 文档链接

## 实际联调步骤记录

### 1. 创建授权档案

- 请求：
  - `POST /api/auth/profiles`
- 关键字段：
  - `providerKey`
  - `authMode`
  - `token/cookie/extra`
- 结果：
  - 成功 / 失败
- 返回摘要：
  - `profileId`
  - `status`

### 2. 校验授权档案

- 请求：
  - `POST /api/auth/profiles/{id}/validate`
- 结果：
  - 成功 / 失败
- 返回摘要：
  - `ok`
  - `status`
  - `message`

### 3. 辅助链路验证

按实际情况填写：

- `List`
  - 请求：
  - 结果：
  - 返回摘要：
- `Metadata`
  - 请求：
  - 结果：
  - 返回摘要：
- `CreateDir`
  - 请求：
  - 结果：
  - 返回摘要：
- `FastUploadCheck`
  - 请求：
  - 结果：
  - 返回摘要：

### 4. 任务预览

- 请求：
  - `POST /api/plans/preview`
- 输入摘要：
  - `thresholdMB`
  - `conflictPolicy`
  - `entries`
- 结果：
  - 成功 / 失败
- 预览摘要：
  - 每个文件的 `strategy`
  - 是否出现 `pending_manual`

### 5. 任务创建与运行

- 请求：
  - `POST /api/tasks`
  - `POST /api/tasks/{id}/run`
- 结果：
  - 成功 / 失败
- 任务摘要：
  - `taskId`
  - `task.state`
  - `completionKind`
- 结果摘要：
  - `done` 数量
  - `failed` 数量

## 运行证据记录

### 1. `/api/evidence/runtime`

至少记录：

- `totalTasks`
- `completedTasks`
- `doneResultCount`
- `failedResultCount`
- `recentResults`
- `recentProbes`

建议记录格式：

```json
{
  "totalTasks": 1,
  "completedTasks": 1,
  "doneResultCount": 1,
  "failedResultCount": 0
}
```

### 2. `/api/status/providers`

至少记录目标 provider 的：

- `providerKey`
- `profileCount`
- `taskCount`
- `completedCount`
- `latestProbe`
- `lastTaskState`
- `snapshotSummary`

## 关键结果断言

### 成功样本至少要满足

- `ValidateAuth` 返回成功
- `POST /api/tasks/{id}/run` 后任务进入：
  - `completed`
  - 或 `completed_with_errors`
- 目标 provider 在 `/api/status/providers` 中可见
- `latestProbe` 不为空
- `lastTaskState` 不为空
- `/api/evidence/runtime` 中存在：
  - `recentResults`
  - `recentProbes`

### 如果要声明“真实成功样本已打通”，还应满足

- 至少有一个任务结果为 `done`
- `recentResults` 中能看到对应结果
- `recentProbes` 中能看到对应 provider 的最新 probe
- 能说明是：
  - 快传成功
  - 或二进制上传成功
  - 或目录链路成功但上传未完成

## 结果分类

本次记录建议明确归类为以下之一：

- `auth_only`
  - 只打通授权校验
- `browse_only`
  - 打通目录 / 元数据 / 建目录
- `fast_upload_success`
  - 打通快传成功样本
- `binary_upload_success`
  - 打通普通上传成功样本
- `partial_blocked`
  - 部分链路成功，但关键环节仍阻塞
- `failed`
  - 本次未形成有效样本

## 异常与阻塞记录

- 失败点：
- 错误状态：
  - 例如 `missing_md5`
  - 例如 `auth_expired`
  - 例如 `rate_limited`
  - 例如 `pending_manual_requires_confirmation`
- 失败原因分析：
- 是否属于当前已知设计缺口：
  - 是 / 否
- 是否需要新增字段、状态或能力声明：
  - 是 / 否

## 对代码的反推结论

每次真实联调后，建议补这一段，避免记录只停留在“跑过一次”。

- 当前 `AuthProfile` 字段是否够用：
- 当前 `Provider` 元信息是否需要新增字段：
- 当前 `FastUploadCheckRequest` 是否缺字段：
- 当前 `UploadRequest` 是否缺字段：
- 当前错误状态名是否够表达真实失败：
- 当前 UI / evidence 是否还缺展示项：

## 验收结论

- 本次是否形成最小真实联调样本：
  - 是 / 否
- 本次是否可以作为协议族样本：
  - 是 / 否
- 本次是否满足切换前验收的一个子项：
  - 是 / 否
- 下一步建议：

## 附件建议

- 脱敏后的请求 / 响应片段
- 控制台页面截图
- `/api/evidence/runtime` 返回片段
- `/api/status/providers` 返回片段
- 如果有必要：
  - provider 返回体字段映射表
  - 错误报文样例
