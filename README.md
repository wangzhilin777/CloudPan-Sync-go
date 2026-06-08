# CloudPan-Sync-go

CloudPan Sync Go 是一个多网盘互传控制台，用来在浏览器里管理网盘授权、预览同步计划、创建同步任务、查看运行证据，并处理补传、重试和恢复。

如果你只是想使用，不需要自己编译。推荐直接到 GitHub Releases 下载对应平台的服务端包，启动后用浏览器打开控制台即可。

## 主要功能

- 多网盘授权管理：为不同 provider 创建独立授权档案，方便源端和目标端分开管理。
- 同步任务向导：选择源 provider、目标 provider、目录范围、执行模式和冲突策略后，先预览再创建任务。
- 同步运行控制：支持任务启动、暂停、恢复、重试，并记录最近运行结果。
- 风险与冲突提示：根据 provider 能力、风险档位和任务参数展示推荐执行方式。
- 证据与状态矩阵：在控制台查看运行证据、provider 状态、异常原因、待补传项和 smoke 记录。
- 后台补传与恢复：对失败、缺会话、待人工处理、上传 checkpoint 等情况提供恢复入口。
- Docker 部署：可以本地构建镜像，也可以下载 Release 里的 Docker 镜像归档导入运行。
- GitHub 自动打包：推送版本标签后，GitHub Actions 会自动生成多平台服务端包、Docker 包和 Windows 桌面客户端包。

## 支持的 Provider

当前控制台内置这些 provider 标识：

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

不同 provider 的登录方式、风控限制和可用能力不完全相同。创建任务前，建议先在控制台的 Provider 页面查看能力声明和默认风控模板，再用小目录试跑。

## 下载

Release 页面：

<https://github.com/wangzhilin777/CloudPan-Sync-go/releases>

常见下载项说明：

- `cloudpan-sync-go-windows-amd64.zip`：Windows 服务端。
- `cloudpan-sync-go-linux-amd64.tar.gz`：Linux x64 服务端。
- `cloudpan-sync-go-linux-arm64.tar.gz`：Linux ARM64 服务端。
- `cloudpan-sync-go-darwin-amd64.tar.gz`：macOS Intel 服务端。
- `cloudpan-sync-go-darwin-arm64.tar.gz`：macOS Apple Silicon 服务端。
- `cloudpan-sync-go-docker-image.tar.gz`：Docker 镜像归档，可用 `docker load` 导入。
- `cloud-clipboard-desktop-windows-amd64.zip`：Windows 桌面客户端包。
- `SHA256SUMS.txt`：文件校验值。

不知道下哪个时：

- Windows 普通用户下载 `cloudpan-sync-go-windows-amd64.zip`。
- Linux 服务器常规 x64 下载 `cloudpan-sync-go-linux-amd64.tar.gz`。
- NAS、树莓派、ARM 服务器优先看系统架构，通常可试 `linux-arm64`。
- 只想用 Docker 时下载 `cloudpan-sync-go-docker-image.tar.gz`，或者直接本地构建镜像。

## 最快启动

### Windows 服务端

1. 下载 `cloudpan-sync-go-windows-amd64.zip`。
2. 解压到一个固定目录，例如 `D:\Apps\cloudpan-sync-go`。
3. 在目录里打开 PowerShell。
4. 启动服务：

```powershell
.\cloudpan-sync.exe
```

5. 浏览器打开：

```text
http://127.0.0.1:8080/
```

默认管理员密码是 `admin`。正式使用前建议改掉：

```powershell
$env:CLOUDPAN_ADMIN_PASSWORD="换成你的强密码"
.\cloudpan-sync.exe
```

### Linux / macOS 服务端

下载对应平台的 `.tar.gz` 包后：

```bash
tar -xzf cloudpan-sync-go-linux-amd64.tar.gz
chmod +x ./cloudpan-sync
CLOUDPAN_ADMIN_PASSWORD="换成你的强密码" ./cloudpan-sync
```

然后打开：

```text
http://127.0.0.1:8080/
```

### 从源码运行

适合开发者或想本地调试的人：

```powershell
go run ./cmd/cloudpan-sync
```

默认地址：

```text
http://127.0.0.1:8080/
```

## Docker 使用

### 方式一：本地构建镜像

在仓库根目录执行：

```powershell
docker build -t cloudpan-sync-go .
```

运行：

```powershell
docker run --rm -p 8080:8080 -v ${PWD}/.cloudpan-sync-go:/data -e CLOUDPAN_ADMIN_PASSWORD=admin cloudpan-sync-go
```

浏览器打开：

```text
http://127.0.0.1:8080/
```

### 方式二：导入 Release 里的 Docker 镜像

先下载 `cloudpan-sync-go-docker-image.tar.gz`，然后在文件所在目录执行：

```powershell
docker load -i cloudpan-sync-go-docker-image.tar.gz
docker run --rm -p 8080:8080 -v ${PWD}/.cloudpan-sync-go:/data -e CLOUDPAN_ADMIN_PASSWORD=admin cloudpan-sync-go:release
```

### Docker 数据目录

容器内默认数据目录是 `/data`，上面的命令会把它挂载到当前目录的 `.cloudpan-sync-go`：

- SQLite 数据库：`.cloudpan-sync-go/cloudpan-sync.db`
- 停止容器后，数据仍保留在本地挂载目录。
- 如果你换目录运行，挂载出来的数据目录也会变。

长期使用时建议把挂载目录换成固定路径，例如：

```powershell
docker run --rm -p 8080:8080 -v "D:\CloudPanSyncData:/data" -e CLOUDPAN_ADMIN_PASSWORD="换成你的强密码" cloudpan-sync-go:release
```

## 可选公网访问

项目本身没有 Cloudflare Worker 功能，也不需要部署到 Cloudflare Workers。服务端仍然运行在你的本机、服务器或 Docker 容器里。

如果你想在外网临时访问本机控制台，可以额外使用 Cloudflare Quick Tunnel、正式 Cloudflare Tunnel、Nginx、Caddy 或其它反向代理工具。下面只以 Cloudflare Quick Tunnel 举例，它适合临时演示、自己异地访问或短时间远程联调。

### 第一步：启动 CloudPan Sync Go

任选一种启动方式，例如：

```powershell
.\cloudpan-sync.exe
```

或：

```powershell
docker run --rm -p 8080:8080 -v ${PWD}/.cloudpan-sync-go:/data -e CLOUDPAN_ADMIN_PASSWORD=admin cloudpan-sync-go:release
```

确认本机可以打开：

```text
http://127.0.0.1:8080/
```

### 第二步：启动 cloudflared

安装 `cloudflared` 后执行：

```powershell
cloudflared tunnel --url http://127.0.0.1:8080
```

终端会输出一个类似下面的地址：

```text
https://xxxxx.trycloudflare.com
```

外部设备打开这个地址，就能访问你的本机控制台。

注意事项：

- Quick Tunnel 是临时地址，关闭 `cloudflared` 后地址会失效。
- 公网暴露前一定要修改 `CLOUDPAN_ADMIN_PASSWORD`，不要继续使用默认密码。
- 如果你有正式 Cloudflare Tunnel 和自己的域名，也可以把公网域名转发到本机 `127.0.0.1:8080`。
- 这只是外部访问入口，不是项目内置的 CF Worker、边缘函数或托管后端。

## 第一次使用流程

1. 启动服务端，打开 `http://127.0.0.1:8080/`。
2. 使用管理员密码登录。
3. 进入 `Provider / 授权`。
4. 分别创建源网盘和目标网盘的授权档案。
5. 确认授权可用后，进入 `任务向导`。
6. 选择源 provider、目标 provider、源目录、目标目录、风险档位和执行模式。
7. 点击预览计划，先检查同步方向、冲突策略、删除记录提示和推荐执行模式。
8. 确认无误后创建任务。
9. 到任务列表或任务详情里启动任务。
10. 运行后到 `运行结果`、`证据`、`状态矩阵` 查看结果。
11. 如果有待补传、缺会话、风控暂停或上传 checkpoint，按控制台提示处理后再恢复或重试。

第一次试跑建议：

- 先准备一个很小的测试目录。
- 源端放几个普通文件，不要一开始就跑大目录。
- 目标端用空目录，方便观察结果。
- 确认方向和冲突策略都正确后，再扩大同步范围。

## 常用环境变量

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `CLOUDPAN_ADDR` | `:8080` | 服务监听地址。 |
| `CLOUDPAN_DATA_DIR` | `./.cloudpan-sync-go` | 数据目录。 |
| `CLOUDPAN_DB_PATH` | `./.cloudpan-sync-go/cloudpan-sync.db` | SQLite 数据库路径。 |
| `CLOUDPAN_ADMIN_PASSWORD` | `admin` | 管理员登录密码。 |
| `CLOUDPAN_LOG_LEVEL` | `info` | 日志级别。 |

示例：

```powershell
$env:CLOUDPAN_ADMIN_PASSWORD="换成你的强密码"
$env:CLOUDPAN_ADDR=":18080"
$env:CLOUDPAN_DATA_DIR="D:\CloudPanSyncData"
.\cloudpan-sync.exe
```

然后访问：

```text
http://127.0.0.1:18080/
```

## 桌面客户端

Release 中附带 `cloud-clipboard-desktop-windows-amd64.zip`，这是配套的 Windows 桌面客户端包。

它主要用于和服务端配合：

- 托盘常驻。
- 本地控制面板。
- 连接服务端房间。
- 文本剪贴板同步。
- 文件上传、下载和拉取。
- Windows 右键菜单动作。

首次使用：

1. 先启动 CloudPan Sync Go 服务端。
2. 解压 `cloud-clipboard-desktop-windows-amd64.zip`。
3. 运行 `cloud-clipboard-desktop.exe`。
4. 首次运行会生成 `config.json`。
5. 在控制面板里填写 `serverBase`、`room`、`roomPassword`、`deviceName`。
6. 保存后等待客户端自动重连。

`serverBase` 填写示例：

- 本机服务端：`http://127.0.0.1:8080`
- 局域网服务端：`http://你的电脑IP:8080`
- Cloudflare 临时地址：`https://xxxxx.trycloudflare.com`

## GitHub 打包与发布

这个仓库支持 GitHub Actions 自动打包，不需要每次都本地编译后手工上传。

主要工作流：

- `docker-package`：构建 Docker 镜像归档。
- `release-package`：构建服务端多平台包、Docker 镜像归档、Windows 桌面客户端包，并发布到 GitHub Releases。

触发方式：

- 推送版本标签，例如 `v0.2.0`。
- 在 GitHub Actions 页面手动运行 `release-package`。

发布成功后，Release 会包含：

- Windows / Linux / macOS 服务端包。
- Docker 镜像归档包。
- Windows 桌面客户端包。
- `SHA256SUMS.txt` 校验文件。

建议版本号策略：

- `v1.0.0`：首个正式稳定版。
- `v1.0.1`：只修复问题，不改主要功能。
- `v1.1.0`：增加功能，但保持兼容。
- `v2.0.0`：存在明显不兼容变更。

## 开发和测试

常用命令：

```powershell
go test ./...
go vet ./...
go build ./cmd/cloudpan-sync
```

更多开发资料：

- `docs/03-DEVELOPER_GUIDE.md`：开发说明。
- `docs/04-API_WORKFLOW_EXAMPLES.md`：API 工作流示例。
- `docs/05-PROVIDER_INTEGRATION_GUIDE.md`：Provider 接入指南。
- `web/README.md`：内置 Web 控制台说明。
