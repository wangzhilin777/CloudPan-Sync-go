# Completed Milestones

## v0.4.0 - 三期计划完整实现 (2026-06-13)

### 六条主线全部达标

1. **控制台体验收口** ✅
   - Provider 卡默认压缩，高级信息折叠
   - 统一按钮胶囊样式，修复窄屏变形
   - 响应式布局覆盖 960px / 640px 两级断点

2. **完整双语支持** ✅
   - 建立统一 i18n 机制
   - 默认 zh-CN，支持切换并持久化
   - 所有界面元素走翻译函数

3. **授权引导与 OpenList / Alist 接入** ✅
   - OpenList / Alist 连接配置区
   - 存储发现与自动匹配 provider
   - 授权失败返回中文可理解错误

4. **源目录到目标目录可视化选择** ✅
   - 任务向导支持源/目标目录选择器
   - 目录浏览器支持浏览、创建、面包屑导航
   - 运行时上传使用映射后的目标路径

5. **图形化桌面客户端** ✅
   - Chrome/Edge --app 独立窗口作为正式方案
   - 桌面端退出时自动清理服务
   - 启动提示支持双语（CLOUDPAN_LANG）

6. **发布流程与 README 补强** ✅
   - Release 清晰区分服务端包、桌面包、Docker 包
   - SHA256SUMS.txt 覆盖所有平台
   - 发布产物白名单校验

### 技术决策

- 完成 WebView 技术调研，发现 2026 年 webview 生态已重构为纯 C/C++ 库
- 决策保持 Chrome/Edge --app 方案作为长期桌面解决方案
- 详见 `docs/WEBVIEW_INTEGRATION_PLAN.md`

### 测试验证

- ✅ `node --check web/static/app.js`
- ✅ `go test ./... -count=1` 全量通过
- ✅ UI smoke 测试通过
- ✅ 桌面双语测试通过

---

## v0.3.0 - 三期核心功能验证 (2026-06-13)

六条主线核心功能已完成，桌面客户端使用 Chrome/Edge --app 方案。

详细的二期和早期里程碑记录已归档至 `docs/archive/`。

