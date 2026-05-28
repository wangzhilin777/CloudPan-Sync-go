# CloudPan Sync Go Provider 接入开发指南

## 目的

- 面向后续要把 placeholder adapter 替换成真实 provider 实现的开发者。
- 说明当前 provider 抽象层怎么设计、每类 provider 该落到哪里、接入时必须满足哪些约束。
- 尽量让新增或替换 provider 的工作从“摸索代码”变成“按清单落地”。

## 当前架构位置

- provider 抽象定义：
  - `internal/provider/provider.go`
- provider 注册表：
  - `internal/provider/registry.go`
- 当前协议族 / 独立链路适配器：
  - `internal/provider/open_family.go`
  - `internal/provider/share_family.go`
  - `internal/provider/hash_family.go`
  - `internal/provider/baidu_family.go`
  - `internal/provider/pan115_family.go`
  - `internal/provider/cloud189_family.go`
  - `internal/provider/guangya_family.go`
- 当前 provider 契约级测试：
  - `internal/provider/*_test.go`

## 当前 provider 模型

### 1. `Provider`

- 用于描述 provider 元信息和能力面，关键字段包括：
  - `key`
  - `displayName`
  - `protocolGroup`
  - `authModes`
  - `fastUploadInputs`
  - `fallbackModes`
  - `conflictPolicies`
  - `supportsOverwrite`
  - `supportsAutoRename`
  - `overwriteBehavior`
  - `status`

### 2. `CapabilitySet`

- 用于声明 provider 是否支持：
  - `ValidateAuth`
  - `List`
  - `Metadata`
  - `CreateDir`
  - `FastUploadCheck`
  - `Upload`

### 3. `Adapter`

- 所有 provider 都必须实现统一接口：
  - `Meta()`
  - `Capabilities()`
  - `ValidateAuth(profile AuthProfile)`
  - `List(req ListRequest)`
  - `Metadata(req MetadataRequest)`
  - `CreateDir(req CreateDirRequest)`
  - `FastUploadCheck(req FastUploadCheckRequest)`
  - `Upload(req UploadRequest)`

## 当前已注册 provider

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

## 当前协议族划分

### 1. Open Family

- 文件：
  - `internal/provider/open_family.go`
- 当前成员：
  - `aliyundrive_open`
  - `123_open`
- 当前认证特点：
  - 依赖 `token`
  - `123_open` 额外允许从 `extra.authorization` 读取 `Bearer` 头
  - `aliyundrive_open` 还要求：
    - `extra.domainId`
    - `extra.driveId`
    - 支持通过 `extra.apiEndpoint` 覆盖真实校验目标地址，便于本地 mock 或联调
- 当前快传指纹：
  - `aliyundrive_open`: `sha1 + size`
  - `123_open`: `md5 + size`
- 当前落地进度：
  - `aliyundrive_open` 的 `ValidateAuth` 已走真实远程校验
  - 当前最小目录与上传链路已接通：
    - `List`
    - `Metadata`
    - `CreateDir`
    - `FastUploadCheck`
    - `Upload`
    - 多分片 `Upload`
  - 当前会访问：
    - `POST /v2/user/get`
    - `POST /v2/drive/get_default_drive`
    - `POST /adrive/v1.0/openFile/list`
    - `POST /adrive/v1.0/openFile/get`
    - `POST /adrive/v1.0/openFile/create`
    - `POST /v2/file/complete`
  - `123_open` 已接入真实最小主链路：
    - `GET /api/v2/file/list`
    - `POST /upload/v1/file/mkdir`
    - `POST /upload/v1/oss/file/create`
    - `POST /upload/v1/oss/file/get_upload_url`
    - `PUT presignedURL`
    - `POST /upload/v1/oss/file/upload_complete`
    - `POST /upload/v1/oss/file/upload_async_result`
  - `123_open` 当前约束：
    - 先落单分片上传主链路
    - `fast_upload` 已接入 `file/create` 的 provider 侧复用命中；命中返回 `rapidUpload=true`，未命中返回 `hash_miss` 并交给 runtime fallback
    - `overwrite_existing` 诚实降级为 `auto_rename_new`
    - 上传后优先按 `fileId` 校验，校验失败再回退为父目录按文件名确认

### 2. Share Family

- 文件：
  - `internal/provider/share_family.go`
- 当前成员：
  - `quark`
  - `uc`
- 当前认证特点：
  - 依赖 `cookie`
  - 当前实现中 `RequirePwdID` 为 `true`
  - 还要求：
    - `extra.pwdId`
- 当前快传指纹：
  - `md5`
  - `size`
- 当前落地进度：
  - `quark` 已接入真实最小目录主链路：
    - `POST /1/clouddrive/share/sharepage/token`
    - `GET /1/clouddrive/share/sharepage/detail`
    - `POST /1/clouddrive/file/download`
    - `POST /1/clouddrive/file` folder create
  - `uc` 已接入同构真实最小目录主链路：
    - `POST /1/clouddrive/share/sharepage/token`
    - `GET /1/clouddrive/share/sharepage/detail`
    - `POST /1/clouddrive/file/download`
    - `POST /1/clouddrive/file` folder create
  - 两家当前都已接入真实：
    - `ValidateAuth`
    - `List`
    - `Metadata`
    - `CreateDir`
    - `FastUploadCheck`
    - `Upload`
  - 当前上传链路：
    - `POST /1/clouddrive/file/upload/pre`
    - `POST /1/clouddrive/file/update/hash`
    - `POST /1/clouddrive/file/upload/auth`
    - `POST /1/clouddrive/file/upload/finish`
    - `PUT OSS part`
    - `POST OSS multipart complete`
  - 当前约束：
    - hash 命中时会直接走 provider 侧快传确认
    - hash miss 时会继续走 `upload/auth + OSS multipart + upload/finish`
    - 当前同名冲突会诚实降级为 `auto_rename_new`
    - 当前 provider 侧真实上传已支持基于既有 `uploadId + uploadedParts + providerData` 的 multipart checkpoint 恢复
    - `providerData` 会保留 `authInfo / bucket / objKey / uploadUrl / callback / taskId / fileId / parentId / resolvedTargetName / md5 / sha1`
    - 恢复上传会跳过已完成分片，从 `failedPartNumber` 或 `nextPartNumber` 继续上传剩余 OSS part
    - task runtime 已可透传 `uploadCheckpoint + providerData` 做自动续跑编排

### 3. Hash Family

- 文件：
  - `internal/provider/hash_family.go`
- 当前成员：
  - `xunlei`
  - `pikpak`
- 当前认证特点：
  - 依赖 `token`
  - `xunlei` 额外允许：
    - `extra.authorization`
    - `extra.deviceId`
    - `extra.captchaToken`
    - `extra.clientId`
    - `extra.apiEndpoint`
- 当前快传指纹：
  - `gcid`
  - `size`
- 当前落地进度：
  - `xunlei` 已接入真实最小目录主链路：
    - `GET /drive/v1/files`
    - `POST /drive/v1/files` folder create
  - `xunlei` 已接入真实上传 create 主链路：
    - `POST /drive/v1/files` file create
    - hash 命中时直接走 rapid 成功
    - hash miss 时会返回 `resumable` 会话并继续走内置 S3-compatible SigV4 PUT fallback
    - task retry 时已可复用失败结果里保留的 `resumable` 会话，不再重复 create upload
    - PUT 失败时会按统一 upload checkpoint 口径回填 `partCount=1 / failedPartNumber=1 / nextPartNumber=1`
    - PUT 成功后会回填 `statusCode / etag / objectSize / uploadedParts`
  - `xunlei` 当前约束：
    - 当前先落地整对象 PUT fallback 与会话复用恢复，尚未扩展到 multipart-resume / 分片级断点续传
  - `pikpak` 已接入真实最小目录主链路：
    - `GET /drive/v1/files`
    - `POST /drive/v1/files` folder create
  - `pikpak` 已接入真实上传 create 主链路：
    - `POST /drive/v1/files` file create
    - hash 命中时直接走 rapid 成功
    - hash miss 时会返回 `resumable` 会话并继续走内置 S3-compatible SigV4 PUT fallback
    - task retry 时已可复用失败结果里保留的 `resumable` 会话，不再重复 create upload
    - PUT 失败时会按统一 upload checkpoint 口径回填 `partCount=1 / failedPartNumber=1 / nextPartNumber=1`
    - PUT 成功后会回填 `statusCode / etag / objectSize / uploadedParts`
  - `pikpak` 当前约束：
    - 当前先落地整对象 PUT fallback 与会话复用恢复，尚未扩展到 multipart-resume / 分片级断点续传

### 4. 独立链路

- `baidu_netdisk`
  - 文件：`internal/provider/baidu_family.go`
  - 认证：`token` 或 `cookie`
  - 快传指纹：`md5` + `size`
  - 当前落地进度：
    - 已接入真实 `ValidateAuth`
    - 已接入真实 `List / Metadata / CreateDir`
    - `Upload` 已接入真实 `precreate -> superfile2 tmpfile -> create -> verify`
    - `overwrite_existing` 会诚实降级为 `auto_rename_new`
    - tmpfile 上传失败时会按统一 upload checkpoint 口径回填 `partCount=1 / failedPartNumber=1 / nextPartNumber=1`
    - tmpfile 上传成功后会回填 `uploadedPartCount / uploadedParts / md5 / size`
  - 当前约束：
    - 目前先落单分片 tmpfile 主链路
    - 仍未扩展到多分片并行或更复杂断点续传
- `115_open`
  - 文件：`internal/provider/pan115_family.go`
  - 认证：`token` 或 `cookie`
  - 快传指纹：`sha1` + `size`
  - 当前落地进度：
    - 已接入真实 `ValidateAuth / List / Metadata / CreateDir / FastUploadCheck`
    - `Upload` 已接入真实 `upload/init`
    - 当返回 `sign_check` 时，已继续做本地区间 SHA1 follow-up
    - hash miss 后，已接入 `upload/get_token` 并可解析完整 OSS 上传会话
    - 当前默认构建已内置基于 OSS 鉴权头的单对象 PUT fallback
    - task retry 时已可复用失败结果里保留的 OSS upload session，不再重复请求 `upload/init + get_token`
    - hash 命中时可直接返回 rapid success 并做校验
    - OSS PUT 失败时会按统一 upload checkpoint 口径回填 `partCount=1 / failedPartNumber=1 / nextPartNumber=1`
    - OSS PUT 成功后会回填 `uploadedPartCount / uploadedParts / objectSize / responseStatus`
  - 当前约束：
    - 当前先落整对象 PUT 主链路与 upload-session 复用恢复，尚未扩展到 multipart / 分片级断点续传
    - 如果 OSS 会话字段不完整或 provider 侧返回异常，仍会诚实返回 `binary_upload_failed` 并保留真实上传会话
- `189cloud`
  - 文件：`internal/provider/cloud189_family.go`
  - 认证：`cookie`
  - 快传指纹：`md5` + `size`
  - 当前落地进度：
    - 已接入真实 `ValidateAuth / List / Metadata / CreateDir / FastUploadCheck / Upload`
    - 当前读链路基于 `shareCode / accessCode`：
      - `POST /api/open/share/getShareInfoByCodeV2.action`
      - `GET /api/open/share/checkAccessCode.action`
      - `GET /api/open/share/listShareDir.action`
    - `CreateDir` 已接入账号级写鉴权：
      - `POST /api/open/file/createFolder.action`
      - 需要 `AccessToken / Signature / Date`
    - `Upload` 已接入真实主链：
      - `GET /getSessionForPC.action`
      - `POST /createUploadFile.action`
      - `PUT fileUploadUrl`
      - `GET /getUploadFileStatus.action`
      - `POST fileCommitUrl`
    - hash 命中时会直接走 provider 侧复用，并由 commit XML 回包确认
    - hash miss 时会继续走 `fileUploadUrl PUT + getUploadFileStatus + fileCommitUrl`
    - task retry 时已可复用失败结果里保留的 `uploadFileId + fileUploadUrl + fileCommitUrl`，不再重复 `createUploadFile`
    - binary PUT 失败时会按统一 upload checkpoint 口径回填 `partCount=1 / failedPartNumber=1 / nextPartNumber=1`
    - binary PUT 成功后会回填 `uploadedPartCount / uploadedParts / objectSize / status`
  - 当前约束：
    - shareCode/accessCode 当前只提供只读目录链路，不能直接写目录
    - `CreateDir` 仍依赖账号级 `AccessToken / Signature / Date`
    - `Upload` 当前仍主要依赖 `accessToken -> getSessionForPC` 刷新的临时 sessionKey/sessionSecret；已支持 upload-session 级恢复，但尚未扩展到更细粒度断点续传
- `guangya`
  - 文件：`internal/provider/guangya_family.go`
  - 认证：`token`
  - 快传指纹：`md5` + `size` + `name`
  - 当前落地进度：
    - 已接入真实 `ValidateAuth / List / Metadata / CreateDir / FastUploadCheck`
    - 当前目录链路基于：
      - `POST /nd.bizuserres.s/v1/file/get_file_list`
      - `POST /nd.bizuserres.s/v1/get_res_download_url`
      - `POST /nd.bizuserres.s/v1/file/create_dir`
    - `FastUploadCheck` 已接入真实库存预检：
      - `POST /nd.bizuserres.s/v1/get_res_center_token`
      - `POST /nd.bizuserres.s/v1/check_can_flash_upload`
      - `POST /nd.bizuserres.s/v1/file/delete_upload_task`
    - `Upload` 当前已支持真实快传命中闭环：
      - `get_res_center_token` 直接命中时，会在 Go 内确认成功并回做上传后校验
      - GCID flash hit 会继续调用 `upload_info` 做最终确认
      - 上传后优先按 `fileId` 校验，再回退到父目录按文件名确认
      - 同名冲突当前会诚实降级为 `auto_rename_new`
    - `Upload` 当前也已接入真实二进制上传 runtime：
      - 小文件：`upload_token + md5 + upload_info`
      - 大文件：`upload_token + check_can_flash_upload + OSS multipart + upload_info`
      - OSS multipart 当前使用与 `guangyaclient` 一致的签名和 complete flow
      - multipart 失败时已会把 `uploadId / uploadedParts / failedPartNumber / nextPartNumber` 回填到 evidence，重试时可直接从失败分片继续
  - 当前约束：
    - 当前仍缺真实在线样本验收，尤其是风控、限流和不同账号环境下的稳定性证据
    - 当前 provider 侧已支持基于 checkpoint 的已传分片恢复
    - task runtime 已可自动恢复冷却到期任务，以及仅包含 upload-session checkpoint 的安全续传队列
    - 更完整的后台自动恢复编排仍待继续补齐

## 当前接入状态要怎么理解

- 现在这些 adapter 已经全部接进统一内核。
- 但大部分仍属于：
  - 协议占位
  - 字段口径统一
  - 最小行为模拟
- 这意味着：
  - API、planner、task、evidence、UI 都能跑完整闭环
  - 但真实外部平台请求、真实目录与上传细节，还需要逐步替换进来

## 新增或替换 provider 的标准步骤

### 1. 先判断归属

- 如果只是同协议族下的另一家 provider：
  - 优先复用现有 family adapter
  - 把差异沉到参数或小分支，而不是立刻复制整份代码
- 只有在协议语义明显不同的时候，才新建独立 adapter 文件

### 2. 补 `Provider` 元信息

- 在 `internal/provider/registry.go` 的 `DefaultCatalog()` 中注册新条目
- 必须明确：
  - `key`
  - `protocolGroup`
  - `authModes`
  - `fastUploadInputs`
  - `fallbackModes`
  - `conflictPolicies`
  - `overwriteBehavior`

### 3. 落 adapter 实现

- 至少先把以下方法补成真实实现或真实可扩展骨架：
  - `ValidateAuth`
  - `List`
  - `Metadata`
  - `CreateDir`
  - `FastUploadCheck`
  - `Upload`
- 这 6 个能力是当前系统中 provider 可被完整消费的最小集合

### 4. 对齐认证字段

- `AuthProfile` 目前统一承载：
  - `token`
  - `cookie`
  - `extra`
- provider 自己决定从哪些字段读认证信息
- 推荐原则：
  - 固定字段优先放 `token` / `cookie`
  - 不同平台特有字段放 `extra`
  - `extra` 的 key 命名要稳定，不要一会儿驼峰一会儿下划线

### 5. 对齐快传输入

- `FastUploadCheckRequest` 目前统一提供：
  - `md5`
  - `sha1`
  - `gcid`
  - `size`
  - `name`
- provider 要根据自身协议声明可用指纹
- 当前已有口径：
  - Aliyun Open / 115：`sha1`
  - 123 Open / Share / Baidu / 189Cloud：`md5`
  - Xunlei / PikPak：`gcid`
  - Guangya：`md5 + size + name`

### 6. 对齐冲突策略

- 当前统一冲突策略：
  - `overwrite_existing`
  - `auto_rename_new`
- provider 自己要明确：
  - 是否真的支持 overwrite
  - 是否真的支持 auto rename
  - 如果不支持，如何降级
- 当前常见降级语义：
  - `downgrade_to_auto_rename`
  - `provider_managed`
  - `not_implemented`
  - `readonly_auth_blocked`

### 7. 对齐 fallback 语义

- 当前规划器和 runtime 已经认识：
  - `fast_upload`
  - `download_upload`
  - `pending_manual`
- provider 真实实现时至少要考虑：
  - hash 命中直接快传
  - hash miss 是否允许回落到二进制上传
  - `FastUploadCheck` 是否会被 runtime 先调用做预检
  - 预检未命中时，是否允许 runtime 不再发起一次无意义快传而直接回退
  - 超大文件或特殊场景是否暂时返回 `pending_manual_requires_confirmation`

## 推荐的实现方式

### 1. 先替换 `ValidateAuth`

- 最先接真实认证校验，收益最大：
  - 能验证 auth profile 字段设计是否够用
  - 能尽快形成真实联调样本
  - 能及时发现 cookie/token/extra 的缺口
- 当前已落地的第一条样板：
  - `aliyundrive_open`

### 2. 再替换目录与元数据链路

- 建议顺序：
  - `List`
  - `Metadata`
  - `CreateDir`
- 这一步先打通资源定位和目录组织，比直接上上传链路更稳

### 3. 最后替换上传链路

- 建议顺序：
  - `FastUploadCheck`
  - `Upload`
- 如果平台上传协议复杂，可以先实现：
  - 快传命中
  - 简单文件上传
- 再逐步补：
  - 分片
  - 秒传失败回退
  - 断点续传
  - 限流恢复
- 当前 `aliyundrive_open` 已补到：
  - 快传命中
  - 秒传失败回退
  - 多分片上传
  - 基于 `uploadId + fileId` 的剩余分片 URL 刷新与继续上传

## 当前错误语义约束

- 当前 API 层已经对这些错误语义做了映射：
  - `missing_access_token`
  - `missing_domain_or_drive_id`
  - `missing_cookie`
  - `missing_pwd_id`
  - `missing_access_token_or_cookie`
  - `missing_md5`
  - `missing_sha1`
  - `missing_gcid`
  - `pending_manual_requires_confirmation`
- 新实现建议尽量复用这些状态名，减少前后端和 runtime 额外分支

## 当前测试入口

### 1. 注册表测试

- `internal/provider/registry_test.go`
- 当前验证：
  - 默认目录中存在 10 家 provider

### 2. 协议族 / provider 契约测试

- `internal/provider/open_family_test.go`
- `internal/provider/share_family_test.go`
- `internal/provider/hash_family_test.go`
- `internal/provider/baidu_family_test.go`
- `internal/provider/pan115_family_test.go`
- `internal/provider/cloud189_family_test.go`
- `internal/provider/guangya_family_test.go`

### 3. Runtime / API 联调测试

- `internal/task/service_test.go`
- `internal/app/workflow_test.go`
- `internal/app/ui_smoke_test.go`

## 每次接入后至少要验证什么

### 最小验证

- `go test ./internal/provider/...`
- `go test ./internal/task/...`
- `go test ./internal/app/...`

### 推荐整仓验证

- `go test ./...`
- `go build ./...`

## 建议的新增测试清单

- 新 provider 接入后，至少补这几类测试：
  - 认证缺字段时的失败测试
  - 合法认证时的 `ValidateAuth` 成功测试
  - `FastUploadCheck` 候选判定测试
  - 缺失关键指纹时的 `Upload` 失败测试
  - `pending_manual` 或 fallback 相关语义测试

## 什么时候该新建 family，什么时候不该

- 适合复用 family：
  - 认证模型一致
  - 目录 API 结构相近
  - 快传输入类型一致
  - 上传冲突策略差异不大
- 适合拆独立 adapter：
  - 认证字段结构明显不同
  - 上传协议完全不同
  - 错误语义或回退逻辑不共享
  - 继续硬塞进 family 会让代码分支过多

## 当前最推荐的下一步

- 优先从下面任一方向切一条真实 provider：
  - `115_open`
  - `guangya`
  - `xunlei/pikpak`
- 原因：
  - `quark / uc / 189cloud` 已完成当前阶段最核心的上传主链补齐
  - `115_open / guangya` 继续补断点恢复与真实样本，对现有 planner / task / UI 主链路验证价值高
  - `xunlei/pikpak` 的 resumable fallback 也适合继续深化恢复语义
