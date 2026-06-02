# CloudPan-Sync-go

CloudPan Sync 多网盘互传控制台。

这是一个面向常用网盘之间互传的服务端控制台，重点提供：授权管理、任务预览、任务运行、运行证据、状态矩阵、后台补传，以及浏览器控制台操作入口。

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

推荐直接从 GitHub Releases 下载已打包好的文件。

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

Windows 下解压 Release 包后，直接运行：

```powershell
.\cloudpan-sync.exe
```

Linux / macOS 下解压后运行：

```bash
./cloudpan-sync
```

### 方式 3：Docker 启动

本地构建镜像：

```powershell
docker build -t cloudpan-sync-go .
```

运行容器：

```powershell
docker run --rm -p 8080:8080 -v ${PWD}/.cloudpan-sync-go:/data -e CLOUDPAN_ADMIN_PASSWORD=admin cloudpan-sync-go
```

说明：

- 容器默认监听 `8080`
- 数据目录默认是 `/data`
- SQLite 文件默认是 `/data/cloudpan-sync.db`
- 浏览器访问 `http://127.0.0.1:8080/`

如果你是用 Release 里的 Docker 镜像归档包，先导入：

```powershell
docker load -i cloudpan-sync-go-docker-image.tar.gz
docker run --rm -p 8080:8080 -v ${PWD}/.cloudpan-sync-go:/data -e CLOUDPAN_ADMIN_PASSWORD=admin cloudpan-sync-go:release
```

## Cloudflare 暴露（CF）

如果你想把本机服务临时暴露到公网，最简单的是用 `cloudflared` Quick Tunnel。

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

说明：

- Quick Tunnel 适合临时演示和临时远程访问
- 关闭 `cloudflared` 进程后，临时公网地址会失效
- 如果你有自己的 Cloudflare 域名和正式 Tunnel，也可以把本机 `127.0.0.1:8080` 挂到自定义域名上

## 第一次使用怎么操作

1. 打开控制台首页并登录。
2. 使用默认密码 `admin` 登录，或在启动前用环境变量覆盖管理员密码。
3. 切到 `Provider / 授权`，先创建目标网盘或源网盘的授权档案。
4. 切到 `任务向导`，选择源 provider、目标 provider、风险模式和执行模式。
5. 填写 `Selected Roots` 和 `Entries`，点击“预览计划”。
6. 确认预览中的同步策略、删除记录提示、推荐执行模式和推荐风控后创建任务。
7. 到 `任务列表详情` 里运行任务，并查看结果、待补传树和运行提示。
8. 到 `状态矩阵 / 证据` 查看最近结果、最近 probe、自动补传候选和状态聚合。

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

## GitHub 打包与 Releases

仓库支持 GitHub 自动打包与发布。

- `docker-package`：打 Docker 镜像归档
- `release-package`：打服务端多平台包、桌面客户端包，并发布到 GitHub Releases

你可以通过两种方式触发：

- 推送版本标签，例如 `v0.1.0`
- 在 GitHub Actions 页面手动触发 `release-package`

## 常用命令

```powershell
go test ./...
go build ./...
```

## 相关文档

- 项目实施计划：`docs/01-GO_REBUILD_PLAN.md`
- 当前功能与进度：`docs/02-PROJECT_STATUS.md`
- 开发与上手说明：`docs/03-DEVELOPER_GUIDE.md`
- API 工作流示例：`docs/04-API_WORKFLOW_EXAMPLES.md`
- Provider 接入指南：`docs/05-PROVIDER_INTEGRATION_GUIDE.md`
- 真实联调记录模板：`docs/06-REAL_PROVIDER_SMOKE_TEMPLATE.md`
- 功能清单与完成度矩阵：`docs/07-FEATURE_MATRIX.md`
