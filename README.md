# CloudPan-Sync-go

CloudPan Sync Go 是一个多网盘互传控制台，用来在浏览器里管理网盘授权、预览同步计划、创建同步任务、查看运行证据，并处理补传、重试和恢复。

如果你只是想使用，不需要自己编译。推荐直接到 GitHub Releases 下载对应平台的桌面包、服务端包或 Docker 包。

## 主要功能

- 多网盘授权管理：为不同 provider 创建独立授权档案，方便源端和目标端分开管理。
- 同步任务向导：选择源 provider、目标 provider、目录范围、执行模式和冲突策略后，先预览再创建任务。
- 同步运行控制：支持任务启动、暂停、恢复、重试，并记录最近运行结果。
- 风险与冲突提示：根据 provider 能力、风险档位和任务参数展示推荐执行方式。
- 证据与状态矩阵：在控制台查看运行证据、provider 状态、异常原因、待补传项和 smoke 记录。
- 后台补传与恢复：对失败、缺会话、待人工处理、上传 checkpoint 等情况提供恢复入口。
- Docker 部署：可以本地构建镜像，也可以下载 Release 里的 Docker 镜像归档导入运行。
- 图形桌面入口：桌面模式会自动启动本地服务并弹出独立窗口，减少手动打开浏览器的步骤。
- GitHub 自动打包：推送版本标签后，GitHub Actions 会自动生成多平台服务端包、桌面包和 Docker 包。

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

- `cloudpan-sync-go-desktop-windows-amd64.zip`：Windows 桌面客户端。
- `cloudpan-sync-go-desktop-linux-amd64.tar.gz`：Linux x64 桌面客户端。
- `cloudpan-sync-go-desktop-linux-arm64.tar.gz`：Linux ARM64 桌面客户端。
- `cloudpan-sync-go-desktop-darwin-amd64.tar.gz`：macOS Intel 桌面客户端。
- `cloudpan-sync-go-desktop-darwin-arm64.tar.gz`：macOS Apple Silicon 桌面客户端。
- `cloudpan-sync-go-windows-amd64.zip`：Windows 服务端。
- `cloudpan-sync-go-linux-amd64.tar.gz`：Linux x64 服务端。
- `cloudpan-sync-go-linux-arm64.tar.gz`：Linux ARM64 服务端。
- `cloudpan-sync-go-darwin-amd64.tar.gz`：macOS Intel 服务端。
- `cloudpan-sync-go-darwin-arm64.tar.gz`：macOS Apple Silicon 服务端。
- `cloudpan-sync-go-docker-image.tar.gz`：Docker 镜像归档，可用 `docker load` 导入。
- `SHA256SUMS.txt`：文件校验值。

不知道下哪个时：

- Windows、macOS、Linux 普通电脑优先下载对应的 `desktop` 桌面包。
- Windows 普通用户下载 `cloudpan-sync-go-windows-amd64.zip`。
- Linux 服务器常规 x64 下载 `cloudpan-sync-go-linux-amd64.tar.gz`。
- NAS、树莓派、ARM 服务器优先看系统架构，通常可试 `linux-arm64`。
- 只想用 Docker 时下载 `cloudpan-sync-go-docker-image.tar.gz`，或者直接本地构建镜像。

怎么区分桌面包和服务端包：

- `desktop`：更像普通软件，启动后会自动打开独立窗口，适合个人电脑日常使用。
- `服务端包`：更像后台服务，启动后你再自己用浏览器打开控制台，适合服务器、远程主机、NAS 或想自己配反向代理的人。
- `Docker 包`：适合容器化部署、NAS Docker 套件和长期后台运行。

## 桌面模式

桌面模式是三期正在推进的图形化客户端入口。当前实现会先启动本地服务，再优先用 Chrome / Edge 的独立 app 窗口打开控制台；如果没有找到兼容浏览器，再退回系统默认浏览器。

启动后你会看到这些提示：

- 会先输出当前本地控制台地址，方便窗口没有自动弹出时手动打开。
- 如果命中了 Chrome / Edge 独立窗口模式，会明确提示“关闭窗口后会自动退出本地服务”。
- 如果退回系统默认浏览器，也会明确提示“关闭浏览器标签页不会自动退出服务，需要关闭终端或按 `Ctrl+C`”。
- 如果既没弹出独立窗口，也没自动打开浏览器，可以直接复制终端里显示的本地地址访问。

### Windows 桌面包

1. 下载 `cloudpan-sync-go-desktop-windows-amd64.zip`。
2. 解压到固定目录，例如 `D:\Apps\cloudpan-sync-go-desktop`。
3. 双击 `cloudpan-sync-desktop.exe`，或在 PowerShell 中执行：

```powershell
.\cloudpan-sync-desktop.exe
```

启动后会自动选择可用本地端口，并打开独立窗口或浏览器页面。默认管理员密码仍是 `admin`，正式使用前建议改掉。

退出方式：

- 如果打开的是独立窗口，直接关闭窗口即可，内置服务会一起退出。
- 如果退回到了系统浏览器，请回到启动它的 PowerShell 窗口，按 `Ctrl+C` 停止。

### Linux / macOS 桌面包

下载对应平台的桌面包后：

```bash
tar -xzf cloudpan-sync-go-desktop-linux-amd64.tar.gz
chmod +x ./cloudpan-sync-desktop
./cloudpan-sync-desktop
```

如果当前系统装有 Chrome、Chromium 或 Edge，通常会直接打开独立 app 窗口；否则会回退到系统浏览器。

退出方式同样分两种：

- 独立窗口模式：关闭窗口即可自动退出。
- 系统浏览器兜底模式：关闭浏览器标签页不会自动停服务，需要回到终端按 `Ctrl+C`。

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

如果是在其它电脑、手机或 NAS 旁路设备上访问，请把 `127.0.0.1` 换成运行服务那台机器的局域网 IP。

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

## 访问地址怎么选

服务默认监听 `:8080`，表示监听当前机器或容器的 `8080` 端口，不是只能通过 `127.0.0.1` 访问。

这个地址同时提供浏览器控制台和后端 API，不是只给本机管理面板使用。

- 在运行服务的本机访问：`http://127.0.0.1:8080/`
- 在局域网其它设备访问：`http://运行服务的电脑IP:8080/`
- 在 NAS 上部署后访问：`http://NAS的局域网IP:8080/`
- 通过反向代理或 Tunnel 访问：使用你配置的域名，例如 `https://pan.example.com/`

如果你只想允许本机访问，可以把监听地址显式改成 `127.0.0.1:8080`：

```powershell
$env:CLOUDPAN_ADDR="127.0.0.1:8080"
.\cloudpan-sync.exe
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

在运行 Docker 的本机打开：

```text
http://127.0.0.1:8080/
```

如果 Docker 跑在服务器或 NAS 上，请访问 `http://服务器IP:8080/` 或 `http://NAS的局域网IP:8080/`。

### 方式二：导入 Release 里的 Docker 镜像

先下载 `cloudpan-sync-go-docker-image.tar.gz`，然后在文件所在目录执行：

```powershell
docker load -i cloudpan-sync-go-docker-image.tar.gz
docker run --rm -p 8080:8080 -v ${PWD}/.cloudpan-sync-go:/data -e CLOUDPAN_ADMIN_PASSWORD=admin cloudpan-sync-go:release
```

Docker 标签说明：

- 已发布的 `v0.2.0` Docker 镜像归档使用 `cloudpan-sync-go:release`。
- 后续 GitHub 打包会同时写入 `cloudpan-sync-go:release` 和 `cloudpan-sync-go:latest`。
- `latest` 是 Docker 常用标签名，不是 `last`。
- 如果你导入的是旧包但想使用 `latest`，可以手动补一个别名：

```powershell
docker tag cloudpan-sync-go:release cloudpan-sync-go:latest
```

### 方式三：Docker Compose 启动

仓库提供了 `compose.example.yml` 示例。你可以复制成 `compose.yml` 后修改密码：

```powershell
Copy-Item .\compose.example.yml .\compose.yml
notepad .\compose.yml
```

最小示例：

```yaml
services:
  cloudpan-sync-go:
    image: cloudpan-sync-go:release
    container_name: cloudpan-sync-go
    restart: unless-stopped
    ports:
      - "8080:8080"
    environment:
      CLOUDPAN_ADMIN_PASSWORD: "change-me"
      CLOUDPAN_LOG_LEVEL: "info"
    volumes:
      - ./.cloudpan-sync-go:/data
```

启动：

```powershell
docker compose up -d
```

查看日志：

```powershell
docker compose logs -f
```

停止：

```powershell
docker compose down
```

在运行 Docker 的本机打开：

```text
http://127.0.0.1:8080/
```

如果 Docker 跑在服务器或 NAS 上，请访问对应机器的 IP 或域名。

### Docker 数据目录

容器内默认数据目录是 `/data`，上面的命令会把它挂载到当前目录的 `.cloudpan-sync-go`：

- SQLite 数据库：`.cloudpan-sync-go/cloudpan-sync.db`
- 停止容器后，数据仍保留在本地挂载目录。
- 如果你换目录运行，挂载出来的数据目录也会变。

长期使用时建议把挂载目录换成固定路径，例如：

```powershell
docker run --rm -p 8080:8080 -v "D:\CloudPanSyncData:/data" -e CLOUDPAN_ADMIN_PASSWORD="换成你的强密码" cloudpan-sync-go:release
```

## NAS 部署

NAS 上推荐优先使用 Docker Compose 或 NAS 自带的 Container Manager / Container Station / Docker 套件部署。核心原则是：镜像可以重建，数据目录必须持久化挂载。

### NAS 用 Docker Compose

1. 在 NAS 上新建一个目录，例如 `/volume1/docker/cloudpan-sync-go`。
2. 把 `cloudpan-sync-go-docker-image.tar.gz` 上传到这个目录。
3. 通过 SSH 进入该目录。
4. 导入镜像：

```bash
docker load -i cloudpan-sync-go-docker-image.tar.gz
```

5. 新建 `compose.yml`：

```yaml
services:
  cloudpan-sync-go:
    image: cloudpan-sync-go:release
    container_name: cloudpan-sync-go
    restart: unless-stopped
    ports:
      - "8080:8080"
    environment:
      CLOUDPAN_ADMIN_PASSWORD: "change-me"
      CLOUDPAN_LOG_LEVEL: "info"
    volumes:
      - ./data:/data
```

6. 启动：

```bash
docker compose up -d
```

7. 浏览器访问：

```text
http://NAS的局域网IP:8080/
```

### NAS 图形界面部署

如果你的 NAS 不方便 SSH，也可以在图形界面操作：

- 先在镜像页面导入 `cloudpan-sync-go-docker-image.tar.gz`。
- 创建容器时选择镜像 `cloudpan-sync-go:release`。
- 端口映射填写本机 `8080` 到容器 `8080`。
- 新增环境变量 `CLOUDPAN_ADMIN_PASSWORD`，不要继续使用默认密码。
- 新增环境变量 `CLOUDPAN_LOG_LEVEL=info`。
- 把 NAS 上的固定目录挂载到容器 `/data`。
- 启动后访问 `http://NAS的局域网IP:8080/`，不是访问你电脑自己的 `127.0.0.1`。

NAS 注意事项：

- 不要把 `/data` 挂到临时目录，否则容器重建后数据库会丢。
- 如果 NAS 防火墙开启了端口限制，需要放行 `8080` 或你自定义的端口。
- 如果要从公网访问，优先用 VPN、内网穿透、反向代理或 Cloudflare Tunnel，并务必修改管理员密码。
- ARM NAS 请确认镜像架构；当前 Dockerfile 默认构建 `linux/amd64`，不同架构 NAS 可能需要本地重新构建。

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

确认服务端可以访问。本机部署时打开：

```text
http://127.0.0.1:8080/
```

如果服务端部署在 NAS 或服务器上，则在浏览器中打开 `http://NAS的局域网IP:8080/` 或你的反代域名。

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

1. 启动服务端，打开控制台地址；本机访问用 `http://127.0.0.1:8080/`，局域网或 NAS 访问用对应机器 IP。
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

局域网其它设备访问时，把 `127.0.0.1` 换成运行服务那台机器的 IP。

## GitHub 打包与发布

这个仓库支持 GitHub Actions 自动打包，不需要每次都本地编译后手工上传。

主要工作流：

- `release-package`：统一构建服务端多平台包、桌面客户端包、Docker 镜像归档，并发布到 GitHub Releases。

触发方式：

- 推送版本标签，例如 `v0.2.0`。
- 在 GitHub Actions 页面手动运行 `release-package`。

发布成功后，Release 会包含：

- Windows / Linux / macOS 服务端包。
- Docker 镜像归档包。
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
