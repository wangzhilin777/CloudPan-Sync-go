# CloudPan-Sync-go

CloudPan Sync 多网盘互传控制台。

这是一个面向常用网盘之间互传的服务端控制台，重点提供：授权管理、任务预览、任务运行、运行证据、状态矩阵、后台补传，以及浏览器控制台操作入口。

GitHub Releases 会提供现成安装包；如果你不想自己编译，直接下载对应平台的压缩包即可。

## 能做什么

- 管理多 provider 授权档案
- 预览同步任务并查看执行模式、风险参数和冲突策略
- 运行、暂停、恢复、重试同步任务
- 查看运行证据、最近结果、最近 probe、状态矩阵
- 记录源端删除，不默认删除目标端文件
- 通过 Docker、本地可执行文件或 `go run` 启动服务端
- 通过 GitHub Releases 下载服务端包和桌面客户端包

## 支持的 provider

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

## 下载方式

推荐直接从 GitHub Releases 下载已打包好的文件：

- Releases 页面：<https://github.com/wangzhilin777/CloudPan-Sync-go/releases>
- Actions 页面：<https://github.com/wangzhilin777/CloudPan-Sync-go/actions>

当前 Release 计划包含这些资产：

- 服务端 Windows 包：`cloudpan-sync-go-windows-amd64.zip`
- 服务端 Linux 包：`cloudpan-sync-go-linux-amd64.tar.gz`
- 服务端 Linux ARM64 包：`cloudpan-sync-go-linux-arm64.tar.gz`
- 服务端 macOS Intel 包：`cloudpan-sync-go-darwin-amd64.tar.gz`
- 服务端 macOS Apple Silicon 包：`cloudpan-sync-go-darwin-arm64.tar.gz`
- Docker 镜像归档：`cloudpan-sync-go-docker-image.tar.gz`
- 桌面客户端 Windows 包：`cloud-clipboard-desktop-windows-amd64.zip`
- 校验文件：`SHA256SUMS.txt`

如果你不想自己编译，优先下载 Release 包使用。

## 最快启动

### 方式 1：直接运行本地 Go 服务

```powershell
go run ./cmd/cloudpan-sync
```

默认地址：`http://127.0.0.1:8080`

默认管理员密码：`admin`

启动后浏览器打开：`http://127.0.0.1:8080/`

### 方式 2：运行打包后的可执行文件

Windows 下下载并解压 `cloudpan-sync-go-windows-amd64.zip` 后，直接运行：

```powershell
.\cloudpan-sync.exe
```

Linux / macOS 下下载对应平台的 `.tar.gz` 包，解压后运行：

```bash
./cloudpan-sync
```

如果端口被占用，可以先改监听端口再启动：

```powershell
$env:CLOUDPAN_ADDR=":18080"
.\cloudpan-sync.exe
```

### 方式 3：Docker 启动

如果你本机已经克隆了仓库，可以直接本地构建镜像：

```powershell
docker build -t cloudpan-sync-go .
```

然后运行容器：

```powershell
docker run --rm -p 8080:8080 -v ${PWD}/.cloudpan-sync-go:/data -e CLOUDPAN_ADMIN_PASSWORD=admin cloudpan-sync-go
```

说明：

- 容器默认监听 `8080`
- 数据目录默认是 `/data`
- SQLite 文件默认是 `/data/cloudpan-sync.db`
- 浏览器访问 `http://127.0.0.1:8080/`
- 停止容器后，数据仍保存在你挂载的本地目录 `${PWD}/.cloudpan-sync-go`

如果你不想本地构建，也可以直接下载 Release 里的 `cloudpan-sync-go-docker-image.tar.gz`，再导入并运行：

```powershell
docker load -i cloudpan-sync-go-docker-image.tar.gz
docker run --rm -p 8080:8080 -v ${PWD}/.cloudpan-sync-go:/data -e CLOUDPAN_ADMIN_PASSWORD=admin cloudpan-sync-go:release
```

## Cloudflare 暴露（CF）

如果你想把本机服务临时暴露到公网，最简单的是用 `cloudflared` Quick Tunnel。这个方式适合自己异地访问，或者临时给别人演示。

### 第一步：先在本机启动服务

```powershell
go run ./cmd/cloudpan-sync
```

或者用 Docker：

```powershell
docker run --rm -p 8080:8080 -v ${PWD}/.cloudpan-sync-go:/data -e CLOUDPAN_ADMIN_PASSWORD=admin cloudpan-sync-go
```

### 第二步：安装 `cloudflared`

安装完成后执行：

```powershell
cloudflared tunnel --url http://127.0.0.1:8080
```

执行后终端会返回一个 `https://xxxxx.trycloudflare.com` 地址，外部浏览器即可访问这个控制台。

常见用法：

- 你自己在外网电脑或手机上访问这个地址
- 让协作者打开这个地址帮你一起看控制台
- 配合桌面客户端，把服务端先临时挂到公网再做远程联调

说明：

- Quick Tunnel 适合临时演示和临时远程访问
- 关闭 `cloudflared` 进程后，临时公网地址会失效
- 如果你有自己的 Cloudflare 域名和正式 Tunnel，也可以把本机 `127.0.0.1:8080` 挂到自定义域名上

## 第一次使用怎么操作

1. 打开控制台首页并登录。
2. 使用默认密码 `admin` 登录，或在启动前用环境变量覆盖管理员密码。
3. 第一件事先去 `Provider / 授权`，分别创建源网盘和目标网盘的授权档案。
4. 确认两个 provider 都授权成功后，再进入 `任务向导`。
5. 选择源 provider、目标 provider、风险模式和执行模式。
6. 填写 `Selected Roots` 和 `Entries`，点击“预览计划”。
7. 先看预览结果，确认同步方向、冲突策略、删除记录提示和推荐执行模式都符合预期。
8. 确认无误后创建任务，到 `任务列表详情` 中手动启动。
9. 任务执行后，到 `运行结果`、`证据`、`状态矩阵` 查看是否成功，以及有没有待补传项。

如果你只是第一次试跑，建议先做一个小目录：

- 源端先准备几个测试文件
- 目标端先建一个空目录
- 先同步小范围内容，确认逻辑符合预期后再扩大范围

## 常用环境变量

- `CLOUDPAN_ADDR`
  - 服务监听地址，默认 `:8080`
- `CLOUDPAN_DATA_DIR`
  - 数据目录，默认 `./.cloudpan-sync-go`
- `CLOUDPAN_DB_PATH`
  - SQLite 文件路径，默认 `./.cloudpan-sync-go/cloudpan-sync.db`
- `CLOUDPAN_ADMIN_PASSWORD`
  - 管理员密码，默认 `admin`
- `CLOUDPAN_LOG_LEVEL`
  - 日志级别，默认 `info`

示例：

```powershell
$env:CLOUDPAN_ADMIN_PASSWORD="your-password"
$env:CLOUDPAN_ADDR=":18080"
go run ./cmd/cloudpan-sync
```

## 桌面客户端包说明

Release 中附带一个 Windows 桌面客户端包：`cloud-clipboard-desktop-windows-amd64.zip`。

它适合和服务端配合使用，主要能力包括：

- 托盘常驻
- 本地控制面板
- 连接服务端同步房间
- 文本剪贴板同步
- 文件上传、下载、拉取最新文件到本地剪贴板
- Windows 右键菜单动作

客户端首次使用时：

1. 先启动本仓库的服务端。
2. 解压客户端包并运行 `cloud-clipboard-desktop.exe`。
3. 首次运行会生成 `config.json`。
4. 在客户端控制面板里填写：
   - `serverBase`
   - `room`
   - `roomPassword`
   - `deviceName`
5. 保存后客户端会自动重连。

`serverBase` 可以这样填写：

- 本机直连时：`http://127.0.0.1:8080`
- 局域网访问时：`http://你的电脑IP:8080`
- Cloudflare 暴露时：`https://xxxxx.trycloudflare.com`

## GitHub 打包与 Releases

仓库支持 GitHub 自动打包与发布，不需要每次都本地手工打包再上传。

- `docker-package`：打 Docker 镜像归档
- `release-package`：打服务端多平台包、桌面客户端包，并发布到 GitHub Releases

你可以通过两种方式触发：

- 推送版本标签，例如 `v0.1.0`
- 在 GitHub Actions 页面手动触发 `release-package`

也就是说，不是只能本地打包上传，GitHub Actions 可以直接完成构建并发布到 Releases。

`release-package` 成功后，Release 中会自动带上：

- 服务端 Windows / Linux / macOS 包
- Docker 镜像归档包
- Windows 桌面客户端包
- `SHA256SUMS.txt` 校验文件

## 常用命令

```powershell
go test ./...
go build ./...
```

## 相关文档

- 开发说明：`docs/03-DEVELOPER_GUIDE.md`
- API 工作流示例：`docs/04-API_WORKFLOW_EXAMPLES.md`
- Provider 接入指南：`docs/05-PROVIDER_INTEGRATION_GUIDE.md`
