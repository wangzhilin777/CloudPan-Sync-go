# WebView 桌面窗口集成技术调研

## 调研结论（2026-06-13）

**结论：暂不实施 WebView 集成，保持 Chrome/Edge --app 方案作为 v0.4.0 稳定版本。**

## 技术障碍

### 问题描述
在尝试集成 Go webview 库时，发现所有主流 webview 库在 2026 年都已经重构：

1. **github.com/webview/webview** (v0.0.0-20260309075125)
   - 已重写为纯 C/C++ 库
   - 不再包含 Go 绑定代码
   - 下载后发现没有任何 `.go` 文件

2. **github.com/zserge/webview** (v0.0.0-20260309075125)
   - 同样已重写为纯 C/C++ 库
   - 不包含 Go 包代码
   - `go mod tidy` 报错：module does not contain package

### 根本原因
webview 生态在 2024-2026 年间经历了重大架构调整，从 Go 绑定方式改为纯 C/C++ API。现有的 Go 项目如果要使用 webview，需要：
- 自己编写 CGO 绑定
- 处理跨平台 C/C++ 依赖（Windows WebView2、macOS WKWebView、Linux WebKitGTK）
- 增加构建复杂度

这超出了本项目的技术范围和工作量预算。

## 当前方案评估

### Chrome/Edge --app 方案的优势
- ✓ 已验证稳定可用（v0.3.0 已发布）
- ✓ 无需额外依赖，用户系统通常已有 Chrome/Edge
- ✓ 构建简单，纯 Go 代码
- ✓ 跨平台支持良好
- ✓ 开发和维护成本低

### Chrome/Edge --app 方案的劣势
- ⚠ 依赖用户系统已安装 Chrome 或 Edge
- ⚠ 不是"真正的"原生窗口体验
- ⚠ 窗口标题栏和图标无法完全自定义

## 决策

### v0.4.0：保持当前方案
- 继续使用 Chrome/Edge --app 作为桌面客户端方案
- 更新文档说明这是稳定的长期方案，不再标注为"过渡"
- 产物命名保持 `cloudpan-sync-go-desktop-*`
- README 说明桌面模式依赖 Chrome/Edge，并提供系统浏览器兜底

### 未来可能的路径（v0.5.0+）

如果未来需要真正的原生窗口，可考虑：

1. **等待 Go webview 生态恢复**
   - 关注 webview/webview 是否重新提供 Go 绑定
   - 或者社区是否出现新的维护良好的 Go webview 库

2. **使用 Electron/Tauri 替代方案**
   - Electron: 成熟但体积大
   - Tauri: 轻量但需要 Rust 工具链

3. **编写 CGO 绑定**
   - 工作量大，需要维护三平台 C/C++ 代码
   - 增加构建复杂度和依赖管理负担

4. **接受当前方案作为长期解决方案**
   - Chrome/Edge --app 已能满足大部分桌面使用场景
   - 用户体验可接受
   - 维护成本最低

## 建议
**v0.4.0 发布时不再将 Chrome/Edge --app 标注为"过渡方案"，而是作为正式的桌面客户端实现方式。**

用户如果需要完全不依赖浏览器的方案，可以使用服务端模式 + 自己选择的浏览器。

---

## 原计划（存档，未实施）

以下是原本计划的 WebView 集成方案，因技术障碍未实施。



### 步骤1：添加依赖（网络恢复后执行）

```go
// go.mod 添加
require (
    github.com/webview/webview v0.0.0-20240831120633-6173450d4dd6
)
```

运行 `go mod tidy` 更新依赖。

### 步骤2：重构 internal/desktop/desktop.go

#### 当前实现（Chrome/Edge --app）
```go
// 查找 Chrome/Edge 可执行文件
// 启动 browser --app=http://127.0.0.1:port --user-data-dir=...
// 等待进程退出
```

#### 新实现（WebView）
```go
import "github.com/webview/webview"

func launchDesktopWithWebView(url string, ctx context.Context) error {
    // 创建 WebView 窗口
    w := webview.New(false) // false = 不开启调试模式
    defer w.Destroy()
    
    // 配置窗口属性
    w.SetTitle("CloudPan Sync")
    w.SetSize(1280, 860, webview.HintNone)
    
    // 加载 localhost URL
    w.Navigate(url)
    
    // 运行窗口（阻塞直到窗口关闭）
    w.Run()
    
    return nil
}
```

#### 关键修改点
1. 删除 Chrome/Edge 查找逻辑（`findChromePath`、`chromeCandidates`）
2. 删除浏览器进程启动逻辑（`exec.Command`）
3. 使用 `webview.New()` 创建窗口
4. 使用 `w.Navigate(url)` 加载 localhost URL
5. 使用 `w.Run()` 阻塞主线程直到窗口关闭

### 步骤3：更新生命周期管理

#### 服务清理
当前方案：监听浏览器进程退出后取消上下文
新方案：`w.Run()` 返回后取消上下文

```go
// 启动服务
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

go func() {
    app.Start(ctx, listener)
}()

// 等待服务就绪
time.Sleep(500 * time.Millisecond)

// 启动 WebView（阻塞）
launchDesktopWithWebView(url, ctx)

// WebView 关闭后自动清理（通过 defer cancel()）
```

### 步骤4：更新启动提示

#### 中文提示
```go
fmt.Printf("本地控制台地址：%s\n", url)
fmt.Println("正在启动桌面窗口...")
fmt.Println("关闭窗口后会自动退出本地服务")
```

#### 英文提示
```go
fmt.Printf("Local console URL: %s\n", url)
fmt.Println("Launching desktop window...")
fmt.Println("Service will exit automatically when window closes")
```

### 步骤5：删除不再需要的代码

- 删除 `chromeCandidates()` 函数
- 删除 `findChromePath()` 函数
- 删除 Chrome/Edge 特定的命令行参数构建
- 简化 `buildDesktopWindowOpenError()`（不再需要区分找不到浏览器的错误）

### 步骤6：更新测试

#### internal/desktop/desktop_test.go

删除或更新以下测试：
- `TestChromeCandidatesIncludeLocalEdgePathOnWindows` - 不再需要
- 与浏览器查找相关的测试

新增测试：
- WebView 窗口创建和销毁
- 服务清理机制

### 步骤7：更新构建流程

#### 平台特定依赖

**Windows**
- 需要 WebView2 Runtime（通常已预装在 Windows 10/11）
- 如果用户系统没有，需要提示安装 WebView2 Runtime

**macOS**
- 使用系统自带的 WKWebView，无需额外依赖

**Linux**
- 需要 WebKitGTK
- Ubuntu/Debian: `libwebkit2gtk-4.0-dev`
- Fedora/RHEL: `webkit2gtk3-devel`

#### GitHub Actions 更新

`.github/workflows/release-package.yml` 可能需要添加构建时依赖：

```yaml
- name: Install Linux dependencies
  if: matrix.os == 'ubuntu-latest'
  run: |
    sudo apt-get update
    sudo apt-get install -y libwebkit2gtk-4.0-dev
```

但运行时依赖由用户系统提供，不打包到二进制中。

### 步骤8：验证和测试

#### 本地测试
1. Windows: 直接运行，验证 WebView2 窗口
2. Linux: 安装 WebKitGTK 后运行
3. macOS: 直接运行（如有 Mac 环境）

#### 验证项
- ✓ 窗口正常启动并显示控制台
- ✓ localhost URL 正确加载
- ✓ 窗口标题和尺寸符合预期
- ✓ 窗口关闭后服务进程被清理
- ✓ 双语提示正常显示
- ✓ 产物大小合理（对比 v0.3.0）

#### UI smoke 测试
现有 UI smoke 测试应该可以继续使用，因为它们通过 ChromeDP 访问 localhost，与桌面窗口实现无关。

### 步骤9：文档更新

#### README.md
更新桌面模式说明：
- 说明 WebView 方案替代了浏览器依赖
- Windows 用户如果遇到问题，提示安装 WebView2 Runtime
- Linux 用户需要安装 WebKitGTK 运行时库

#### 12-PHASE3_PROGRESS.md
记录 WebView 集成完成，标记主线五里程碑3达标。

## 预期效果

### 优势
- ✓ 不再依赖系统已安装 Chrome/Edge
- ✓ 真正的原生窗口体验
- ✓ 窗口标题、图标更专业
- ✓ 启动速度可能更快（不需要查找浏览器）

### 可能的问题
- Windows 上需要 WebView2 Runtime（但大部分系统已有）
- Linux 上需要用户安装 WebKitGTK（需要在 README 中说明）
- 产物可能略微增大（但 webview 库很轻量）

## 工作量评估
- 代码重构：2-3 小时
- 本地测试和调试：1-2 小时
- 三平台验证：1-2 小时
- 文档更新：0.5 小时
- 总计：约 1 个工作日

## 实施顺序
1. 等待网络恢复，添加 webview 依赖
2. 重构 internal/desktop/desktop.go
3. 更新测试
4. 本地 Windows 验证
5. 提交代码并触发 GitHub Actions 三平台构建
6. 验证三平台产物
7. 更新文档
8. 打标签发布 v0.4.0
