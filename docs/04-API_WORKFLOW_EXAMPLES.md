# CloudPan Sync Go API 工作流示例

## 适用范围

- 本文档基于当前 Go 重构版实际 API 编写。
- 示例以本地默认地址 `http://127.0.0.1:8080` 为准。
- 示例优先使用 PowerShell，方便 Windows 环境直接联调。
- 当前项目是互传模型，不是定向到单个固定目标端。
- 当前返回结构统一为：

```json
{
  "ok": true,
  "data": {},
  "error": null
}
```

## 启动服务

```powershell
go run ./cmd/cloudpan-sync
```

## 0. 准备基础变量

```powershell
$base = "http://127.0.0.1:8080"
```

## 1. 登录控制台会话

```powershell
$login = Invoke-RestMethod `
  -Method Post `
  -Uri "$base/api/session/login" `
  -ContentType "application/json" `
  -Body '{"password":"admin"}'

$login | ConvertTo-Json -Depth 8
```

## 2. 查看 Provider 列表

```powershell
$providers = Invoke-RestMethod `
  -Method Get `
  -Uri "$base/api/providers"

$providers.data.items | Select-Object meta
```

## 3. 查看单个 Provider 能力

```powershell
$capability = Invoke-RestMethod `
  -Method Get `
  -Uri "$base/api/providers/123_open/capabilities"

$capability | ConvertTo-Json -Depth 8
```

## 4. 创建授权档案

下面示例使用 `123_open` 的 `manual_token` 模式，便于快速跑通主流程。

```powershell
$profilePayload = @{
  providerKey = "123_open"
  authMode    = "manual_token"
  displayName = "123 Open Demo"
  token       = "token-demo"
  extra       = @{}
} | ConvertTo-Json -Depth 8

$profile = Invoke-RestMethod `
  -Method Post `
  -Uri "$base/api/auth/profiles" `
  -ContentType "application/json" `
  -Body $profilePayload

$profileId = $profile.data.id
$profile | ConvertTo-Json -Depth 8
```

## 5. 校验授权档案

```powershell
$validation = Invoke-RestMethod `
  -Method Post `
  -Uri "$base/api/auth/profiles/$profileId/validate"

$validation | ConvertTo-Json -Depth 8
```

预期可以看到类似：

- `ok: true`
- `data.status: verified`

## 6. 预览传输计划

```powershell
$previewPayload = @{
  sourceProvider = "guangya"
  targetProvider = "123_open"
  thresholdMB    = 10
  riskMode       = "balanced"
  executionMode  = "leaf_first_lazy"
  conflictPolicy = "auto_rename_new"
  selectedRoots  = @("/demo")
  entries        = @(
    @{
      path = "/demo/a.bin"
      size = 2048
      md5  = "md5-a"
    },
    @{
      path = "/demo/large.bin"
      size = 20971520
    }
  )
} | ConvertTo-Json -Depth 8

$preview = Invoke-RestMethod `
  -Method Post `
  -Uri "$base/api/plans/preview" `
  -ContentType "application/json" `
  -Body $previewPayload

$preview | ConvertTo-Json -Depth 10
```

常见预期：

- 小文件且有指纹时，会看到 `fast_upload`
- 超过阈值或条件不足时，会看到 `download_upload` 或 `pending_manual`

补充说明：

- 预览计划既可用于“已知 entries 的预分析”，也会返回当前执行模式和推荐模式提示。
- 当前返回的 `metadata` 里建议重点看：
  - `executionMode`
  - `recommendedExecutionMode`
  - `recommendedExecutionModeReason`
  - `executionOrder`
- `riskProfile`
- `riskProfileResolution`
  - 可直接看到：
    - `base`
    - `calibrated`
    - `applied`
    - `calibrationReasons`
    - `overrideFields`
- `riskProfile.maxConcurrent`
- `riskProfile.autoRetryStartHour`
- `riskProfile.autoRetryEndHour`
- 当目录较大或 provider 风控敏感时，通常会优先推荐 `leaf_first_lazy`。

## 7. 创建任务

```powershell
$taskPayload = @{
  sourceProvider  = "guangya"
  targetProvider  = "123_open"
  targetProfileId = $profileId
  thresholdMB     = 10
  conflictPolicy  = "auto_rename_new"
  selectedRoots   = @("/demo")
  entries         = @(
    @{
      path = "/demo/a.bin"
      size = 2048
      md5  = "md5-a"
    }
  )
} | ConvertTo-Json -Depth 8

$taskDetail = Invoke-RestMethod `
  -Method Post `
  -Uri "$base/api/tasks" `
  -ContentType "application/json" `
  -Body $taskPayload

$taskId = $taskDetail.data.task.id
$taskDetail | ConvertTo-Json -Depth 12
```

## 7.1 可选：按需扫描模式创建任务

当你不想预先把整个目录树的 `entries` 都收集出来时，可以只提交 `selectedRoots`，并额外提供 `sourceProfileId`。

```powershell
$sourceProfileId = "source-profile-id"

$lazyTaskPayload = @{
  sourceProvider  = "guangya"
  sourceProfileId = $sourceProfileId
  targetProvider  = "123_open"
  targetProfileId = $profileId
  thresholdMB     = 10
  conflictPolicy  = "auto_rename_new"
  riskMode        = "balanced"
  executionMode   = "leaf_first_lazy"
  selectedRoots   = @("/1", "/2", "/3")
  entries         = @()
} | ConvertTo-Json -Depth 8

$lazyTask = Invoke-RestMethod `
  -Method Post `
  -Uri "$base/api/tasks" `
  -ContentType "application/json" `
  -Body $lazyTaskPayload

$lazyTaskId = $lazyTask.data.task.id
$lazyTask | ConvertTo-Json -Depth 12
```

这个模式的含义是：

- 任务创建时不预先扫完整棵目录树
- 运行时按顶层目录顺序和叶子优先顺序按需列目录
- 当前它是可选模式，但应作为大目录场景的默认优先推荐模式

如果你希望显式使用预扫描模式，可以把 `executionMode` 改成：

```powershell
executionMode = "pre_scan_flat"
```

它更适合：

- 目录较小
- 希望先拿到完整扫描结果
- 对“先扫再跑”的可见性更敏感的联调场景

## 8. 查询任务列表与详情

```powershell
$tasks = Invoke-RestMethod `
  -Method Get `
  -Uri "$base/api/tasks"

$tasks | ConvertTo-Json -Depth 12
```

```powershell
$task = Invoke-RestMethod `
  -Method Get `
  -Uri "$base/api/tasks/$taskId"

$task | ConvertTo-Json -Depth 12
```

## 9. 暂停 / 恢复 / 运行 / 重试任务

### 暂停

```powershell
Invoke-RestMethod `
  -Method Post `
  -Uri "$base/api/tasks/$taskId/pause" | ConvertTo-Json -Depth 12
```

### 恢复

```powershell
Invoke-RestMethod `
  -Method Post `
  -Uri "$base/api/tasks/$taskId/resume" | ConvertTo-Json -Depth 12
```

### 运行

```powershell
$runResult = Invoke-RestMethod `
  -Method Post `
  -Uri "$base/api/tasks/$taskId/run"

$runResult | ConvertTo-Json -Depth 12
```

### 重试

```powershell
Invoke-RestMethod `
  -Method Post `
  -Uri "$base/api/tasks/$taskId/retry" | ConvertTo-Json -Depth 12
```

当前补充语义：

- 如果任务里存在 `pending_manual` 形成的待补传项，`retry` 不再是整任务原样重跑。
- 当前会优先把任务缩小成“待补传子集”：
  - 只保留待补传文件对应的 `plan/items`
  - 只保留待补传文件对应的 `entries`
  - `metadata.retryPendingOnly` 会标记为 `true`
  - 如果失败项携带上传恢复线索，还会把它们保存在 `metadata.retryUploadCheckpoints`
- 当前如果失败项是 `rate_limited`，还会检查冷却时间：
  - 未到 `eligibleAt` 时，`retry` 会返回 `retry_cooldown_active`
- 当前如果失败项已经被 `retryLimit` 耗尽，或仍属于硬阻塞项：
  - `retry` 会返回 `retry_blocked`
- 当前如果任务执行后只剩阻塞型重试项，任务状态会直接落为 `blocked`：
  - `runtime.blockedReason` 用于说明阻塞原因
  - `runtime.blockedAction` 用于说明建议动作
  - `runtime.blockedAdvice` 用于给出处理提示
  - `runtime.nextRetryAt` 用于说明最早自动恢复时间
  - `runtime.uploadCheckpoint` 用于说明当前最近一次可恢复上传的位置
- 如果当前没有待补传项，`retry` 仍按普通整任务重置语义处理。

按路径子集重试示例：

```powershell
Invoke-RestMethod `
  -Method Post `
  -Uri "$base/api/tasks/$taskId/retry" `
  -Body (@{
    paths = @("/demo/pending.bin", "/demo/subtree")
    scope = "selected_pending_subset"
  } | ConvertTo-Json) `
  -ContentType "application/json" | ConvertTo-Json -Depth 12
```

子集重试补充语义：

- `scope=selected_pending_subset`
  - 表示这次 retry 来自待补传树当前筛选结果
- `scope=selected_retry_subset`
  - 表示这次 retry 来自重试队列当前筛选结果
- `scope=selected_directory_subset`
  - 表示这次 retry 来自目录树当前筛选结果
  - 适合把某个目录子树直接缩成独立执行子集，而不必先等它进入 pending / retry queue
- 任务元数据会额外保留：
  - `metadata.retryScope`
  - `metadata.retrySelectedPaths`
- 如果传入的 `paths` 没有命中任何可运行 pending / retryable 项：
  - API 会返回 `retry_selection_empty`

后台补传手动触发示例：

```powershell
Invoke-RestMethod `
  -Method Post `
  -Uri "$base/api/tasks/recover" `
  -Body (@{
    mode        = "upload_checkpoint_auto_resume"
    providerKey = "123_open"
    limit       = 2
  } | ConvertTo-Json) `
  -ContentType "application/json" | ConvertTo-Json -Depth 12
```

按当前筛选结果只放行多棵子树示例：

```powershell
Invoke-RestMethod `
  -Method Post `
  -Uri "$base/api/tasks/recover" `
  -Body (@{
    providerKey = "123_open"
    paths       = @("/demo/leaf-a", "/demo/leaf-c")
    scope       = "selected_retry_subset"
    limit       = 1
  } | ConvertTo-Json -Depth 8) `
  -ContentType "application/json" | ConvertTo-Json -Depth 12
```

如果你要明确限制“只作用于某一个任务样本”，现在可以额外带上 `taskId`：

```powershell
Invoke-RestMethod `
  -Method Post `
  -Uri "$base/api/tasks/recover" `
  -Body (@{
    taskId      = "task-sample-id"
    providerKey = "123_open"
    paths       = @("/demo/leaf-b")
    scope       = "selected_retry_subset"
    limit       = 1
  } | ConvertTo-Json -Depth 8) `
  -ContentType "application/json" | ConvertTo-Json -Depth 12
```

返回重点字段：

- `matchedCount`
  - 当前筛选命中的后台补传候选数量
- `recoveredCount`
  - 本轮实际成功接管并重跑的任务数量
- `skippedByLimit`
  - 因为本轮 `limit` 被保留到下一轮的候选数量
- `skippedByProviderBudget`
  - 因为当前 provider 已达到本轮 `riskProfile.maxConcurrent` 批量预算而被保留到下一轮的候选数量
- `mode / providerKey / limit`
  - 便于 UI 和脚本确认本轮到底按什么条件执行
- `taskId / retryClass / blockedAction / path / scope`
  - 便于确认本轮是否只放行了某种失败类型、某个阻塞动作，或某一棵指定子树
- `profileId`
  - 适合把后台补传进一步收敛到某个授权档案，避免同 provider 下的其它账号被一起命中
- `mode=retry_window_waiting_auto_retry`
  - 适合只查看“已经满足自动补传条件，但还不在允许时间窗内”的候选
- `taskId`
  - 适合把后台补传明确约束在某一个任务样本上，避免同 provider 的其它任务被一起命中
- `path + scope=selected_retry_subset`
  - 适合只放行当前路径子树，避免把整批失败项一起重建
- `paths + scope=selected_retry_subset`
  - 适合按当前 retry 队列筛选结果，一次只放行多棵指定子树
- `paths + scope=selected_pending_subset`
  - 适合按当前待补传树筛选结果，一次只放行多棵指定子树

恢复检查点示例：

```json
{
  "runtime": {
    "uploadCheckpoint": {
      "itemPath": "/docs/a.bin",
      "fileId": "file-uploaded",
      "uploadId": "upload-1",
      "partCount": 3,
      "uploadedPartCount": 1,
      "failedPartNumber": 2,
      "nextPartNumber": 2
    }
  },
  "plan": {
    "metadata": {
      "retryUploadCheckpoints": {
        "/docs/a.bin": {
          "fileId": "file-uploaded",
          "uploadId": "upload-1",
          "nextPartNumber": 2
        }
      }
    }
  }
}
```

## 10. 查看运行证据

```powershell
$evidence = Invoke-RestMethod `
  -Method Get `
  -Uri "$base/api/evidence/runtime"

$evidence | ConvertTo-Json -Depth 12
```

重点字段：

- `totalTasks`
- `completedTasks`
- `blockedTasks`
- `doneResultCount`
- `failedResultCount`
- `blockedActions`
- `recentResults`
- `recentProbes`

## 11. 查看 Provider 状态矩阵

```powershell
$status = Invoke-RestMethod `
  -Method Get `
  -Uri "$base/api/status/providers"

$status | ConvertTo-Json -Depth 12
```

重点字段：

- `providerKey`
- `profileCount`
- `taskCount`
- `completedCount`
- `blockedCount`
- `latestProbe`
- `lastTaskState`
- `snapshotSummary`

## 12. 可选：直接调用 Provider 辅助接口

这些接口适合在真实 provider 接入时做链路排查，不是普通控制台主流程的必经步骤。

### List

```powershell
$listPayload = @{
  profileId = $profileId
  path      = "/"
  parentId  = ""
  pageSize  = 100
} | ConvertTo-Json -Depth 8

Invoke-RestMethod `
  -Method Post `
  -Uri "$base/api/providers/123_open/list" `
  -ContentType "application/json" `
  -Body $listPayload | ConvertTo-Json -Depth 12
```

### Metadata

```powershell
$metadataPayload = @{
  profileId = $profileId
  path      = "/demo/a.bin"
  fileId    = "demo-file-id"
  parentId  = "demo-parent-id"
} | ConvertTo-Json -Depth 8

Invoke-RestMethod `
  -Method Post `
  -Uri "$base/api/providers/123_open/metadata" `
  -ContentType "application/json" `
  -Body $metadataPayload | ConvertTo-Json -Depth 12
```

### Create Dir

```powershell
$dirPayload = @{
  profileId = $profileId
  parentId  = "demo-parent-id"
  dirName   = "new-folder"
} | ConvertTo-Json -Depth 8

Invoke-RestMethod `
  -Method Post `
  -Uri "$base/api/providers/123_open/create_dir" `
  -ContentType "application/json" `
  -Body $dirPayload | ConvertTo-Json -Depth 12
```

### Fast Upload Check

```powershell
$fastCheckPayload = @{
  profileId = $profileId
  path      = "/demo/a.bin"
  parentId  = "demo-parent-id"
  name      = "a.bin"
  size      = 2048
  md5       = "md5-a"
  sha1      = ""
  gcid      = ""
} | ConvertTo-Json -Depth 8

Invoke-RestMethod `
  -Method Post `
  -Uri "$base/api/providers/123_open/fast_check" `
  -ContentType "application/json" `
  -Body $fastCheckPayload | ConvertTo-Json -Depth 12
```

```powershell
$uploadPayload = @{
  profileId      = $profileId
  path           = "/demo/a.bin"
  parentId       = ""
  name           = "a.bin"
  size           = 2048
  localPath      = "C:\\temp\\a.bin"
  conflictPolicy = "auto_rename_new"
  strategy       = "download_upload"
  md5            = "md5-a"
  sha1           = ""
  gcid           = ""
} | ConvertTo-Json -Depth 8

Invoke-RestMethod `
  -Method Post `
  -Uri "$base/api/providers/123_open/upload" `
  -ContentType "application/json" `
  -Body $uploadPayload | ConvertTo-Json -Depth 12
```

## 13. 常见注意点

- 这套 API 不兼容 Python 旧接口。
- `targetProfileId` 只在创建任务时需要；预览计划不依赖它。
- `sourceProfileId` 在按需扫描模式下是运行任务的必要字段。
- `executionMode` 当前支持：
  - `leaf_first_lazy`
  - `pre_scan_flat`
- 不传 `executionMode` 时，默认按 `leaf_first_lazy`。
- `pending_manual_requires_confirmation` 目前仍代表需要后续真实 fallback 运行时补全。
- 当任务详情里出现 `metadata.retryPendingOnly=true` 时，表示这次重试已经缩成待补传子集。
- 当 runtime / probe / snapshot 中出现 `retryQueue` 时，表示当前任务已经具备失败分类后的重试队列证据。
- 当 runtime / retryQueue / probe / snapshot 中出现 `uploadCheckpoint` 时，表示当前任务已经具备上传恢复检查点证据。
- 当 `metadata.retryUploadCheckpoints` 出现时，表示后续 `retry` 会把这批恢复线索继续传给 provider 上传链路。
- 当 `retryQueue` item 出现 `attemptCount / retryLimit / remainingCount / exhausted` 时，表示当前任务已经具备累计重试次数证据。
- 当任务状态为 `blocked` 且 `runtime.nextRetryAt` 已到时，单机 tick 调度器会尝试自动恢复仅受冷却影响的任务。
- 当前很多 provider 仍是协议占位实现，适合联调内核、字段口径和控制台闭环，不代表真实外部平台已经完全打通。
