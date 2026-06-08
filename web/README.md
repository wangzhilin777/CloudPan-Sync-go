# Web 控制台

`web/` 目录存放 CloudPan Sync Go 的内置浏览器控制台。服务端启动后会通过 `go:embed` 把这里的静态页面直接嵌入到 Go 二进制里，不需要单独安装 Node.js，也不需要另外启动前端开发服务器。

## 目录说明

- `embed.go`：把 `static/` 目录嵌入 Go 程序，并提供给服务端路由使用。
- `static/index.html`：控制台页面结构。
- `static/app.js`：控制台交互逻辑和 API 调用。
- `static/styles.css`：控制台样式。

## 页面功能

当前控制台覆盖这些主要模块：

- 登录：使用服务端管理员密码进入控制台。
- Provider / 授权：查看 provider 能力声明，创建和管理授权档案。
- 任务向导：选择源端、目标端、目录范围、风险档位和执行模式，先预览计划再创建任务。
- 任务列表与详情：启动、暂停、恢复、重试任务，并查看最近结果。
- 状态矩阵 / 证据：查看 provider 状态、运行证据、待补传项、retry 候选和 pending 节点。
- Smoke 记录：记录真实 provider 样本、协议组覆盖情况和后续校准关注点。
- 报告入口：查看运行证据报告和 provider smoke 摘要。

## 本地打开

从仓库根目录启动服务端：

```powershell
go run ./cmd/cloudpan-sync
```

或者运行已打包的服务端：

```powershell
.\cloudpan-sync.exe
```

然后浏览器打开：

```text
http://127.0.0.1:8080/
```

如果你修改了监听端口，例如：

```powershell
$env:CLOUDPAN_ADDR=":18080"
go run ./cmd/cloudpan-sync
```

则访问：

```text
http://127.0.0.1:18080/
```

## 开发注意

- 修改 `static/index.html`、`static/app.js` 或 `static/styles.css` 后，重新启动 Go 服务即可看到新页面。
- 页面通过同源 API 访问后端，例如 `/api/providers`、`/api/tasks`、`/api/status/providers`。
- 静态资源由后端挂到 `/assets/`，首页由 `/` 返回。
- 这里是轻量内置前端，不依赖打包工具；如果后续引入前端构建链，需要同步调整 `embed.go` 和发布流程。
