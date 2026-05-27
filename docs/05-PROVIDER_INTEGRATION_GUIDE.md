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
  - `aliyundrive_open` 还要求：
    - `extra.domainId`
    - `extra.driveId`
- 当前快传指纹：
  - `md5`
  - `size`

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

### 3. Hash Family

- 文件：
  - `internal/provider/hash_family.go`
- 当前成员：
  - `xunlei`
  - `pikpak`
- 当前认证特点：
  - 依赖 `token`
- 当前快传指纹：
  - `gcid`
  - `size`

### 4. 独立链路

- `baidu_netdisk`
  - 文件：`internal/provider/baidu_family.go`
  - 认证：`token` 或 `cookie`
  - 快传指纹：`md5` + `size`
- `115_open`
  - 文件：`internal/provider/pan115_family.go`
  - 认证：`token` 或 `cookie`
  - 快传指纹：`sha1` + `size`
- `189cloud`
  - 文件：`internal/provider/cloud189_family.go`
  - 认证：`cookie`
  - 快传指纹：`md5` + `size`
- `guangya`
  - 文件：`internal/provider/guangya_family.go`
  - 认证：`token`
  - 快传指纹：`md5` + `size` + `name`

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
  - Open / Share / Baidu / 189Cloud：`md5`
  - 115：`sha1`
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
  - 超大文件或特殊场景是否暂时返回 `pending_manual_requires_confirmation`

## 推荐的实现方式

### 1. 先替换 `ValidateAuth`

- 最先接真实认证校验，收益最大：
  - 能验证 auth profile 字段设计是否够用
  - 能尽快形成真实联调样本
  - 能及时发现 cookie/token/extra 的缺口

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
  - `123_open`
  - `guangya`
  - `baidu_netdisk`
- 原因：
  - 认证模型相对清晰
  - 能较快形成真实样本
  - 对现有 planner / task / UI 主链路验证价值高
