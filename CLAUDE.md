# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目概述

**Claude Code Light** — Windows 桌面红绿灯挂件，实时显示 Claude Code 工作状态。单一 `.exe`，存于 U 盘即插即用。

## 构建命令

Go 不在 PATH 中，需完整路径：

```powershell
# 普通构建（带控制台，调试用）
C:\Open Source Projects\go\bin\go.exe build -o claude-traffic-light.exe .

# 发布构建（无控制台窗口）
C:\Open Source Projects\go\bin\go.exe build -ldflags="-H windowsgui" -o claude-traffic-light.exe .

# 运行测试
C:\Open Source Projects\go\bin\go.exe test ./...
```

工作目录：`D:\vs code projects\claude code light`

### exe 文件图标（踩坑警示）

窗口/托盘图标走 `main.go` 的 `//go:embed claude-traffic-light.ico`（运行时）；**exe 文件图标**（资源管理器里）是另一套，靠 `rsrc_windows_amd64.syso` 链接时嵌入。

- `.syso` 已 gitignore，缺失时重新生成：`go run github.com/akavel/rsrc@latest -ico claude-traffic-light.ico -o rsrc_windows_amd64.syso`
- **文件名必须是 `rsrc_windows_amd64.syso`**（带平台后缀 go build 才自动挑）。**严禁**再多生成一个 `rsrc.syso`——两个 `.syso` 含资源段 → 链接报 `too many .rsrc sections`，构建必失败。

## 架构

### 模块分层（目标架构，方案2）

```
main.go           — 入口：单实例互斥、加载配置、开机自启同步、安装 hook、居中、启动 watcher 和窗口；含 hook 子命令模式 + //go:embed ico
hookinstall.go    — 把状态 hook 安全合并进 ~/.claude/settings.json（幂等/备份/只增不删）
config/           — config.json（窗口位置/锁定/可见/缩放/开机自启）+ glass-tuning.json（视觉/形变参数）读写
  autostart.go    — HKCU Run 注册表读写删（开机自动启动）+ SyncAutostart 路径自校正 + ToggleAutostart 菜单开关
state/            — 四态枚举（Grey/Green/Yellow/Red）和优先级聚合
watcher/          — 100ms 轮询 hook 写的状态文件，映射四态
ui/               — 原生渲染与窗口管理（D3D11 + DComp + HLSL）
  window.go       — DComp 透明置顶窗、消息循环、托盘/菜单、自接管鼠标拖动（仅胶囊内响应）、SetState
  win32.go        — Win32 API 绑定（窗口样式、托盘、菜单、comctl32 滑块、WDA syscall）
  com.go          — 通用 comCall + D3D11/DXGI/DComp 绑定（含 swapchain ResizeBuffers）
  capture.go      — Desktop Duplication 桌面纹理获取（支持随缩放 Resize）
  render.go       — 渲染管线：device/swapchain/shader 编译 + 每帧绘制（动态 viewport）
  glass.hlsl      — 折射 + 红绿灯 shader（移植自 shuding/liquid-glass）
  physics.go      — 二阶弹簧形变物理（Euler 积分驱动按压/拖动形变）
  sizedialog.go   — 调整大小滑块窗（comctl32 trackbar，右键菜单触发，100%~2000% 无极缩放）
```

> `config/`、`state/`、`watcher/` 有单元测试；`watcher/` 已改为 hook 状态文件驱动（见「状态探测」）；`ui/` 为方案2 原生渲染，已实现。

### 当前进度（2026-06-05）

- **方案2 已实现跑通**：DComp 透明置顶窗 + Desktop Duplication + 折射 shader + 四态红绿灯，实时折射真桌面、拖动顺滑、常亮。
- **状态检测已切到 Claude Code Hooks**（见「状态探测」），替代旧 transcript 轮询。
- **整体缩放**：右键菜单「调整大小…」弹 comctl32 滑块窗，100%~2000% 无极缩放（顶边固定、向下扩展），窗口/swapchain/桌面捕获动态 resize，`scale` 存 config.json 记忆。
- **开机自动启动**：右键菜单「开机自动启动」写 `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`（ToggleAutostart），启动时路径自校正（SyncAutostart）；`startup` 存 config.json。
- **运行时图标**：claude-traffic-light.ico（256/32/16）通过 `//go:embed` 嵌入 exe，`loadIconFromICO` 解析 ico，`CreateIconFromResourceEx` 设窗口 + 托盘图标（纯 Go 标准库，零外部工具）。
- **点击穿透暂不可行**：DComp/`NOREDIRECTIONBITMAP` 窗无法穿透（已坐实），折射优先、待收尾后重构。

### 渲染方案（方案2，已锁定）

**目标**：实时折射真实桌面的液态玻璃，永远置顶，仅靠软件渲染（非截图死图、非 backdrop-filter 黑框）。

```
Desktop Duplication 抓整屏桌面纹理(GPU常驻)
   → HLSL 折射 shader（叠红绿灯）
   → DXGI 合成 swapchain → DirectComposition 透明置顶窗
窗口设 WDA_EXCLUDEFROMCAPTURE 把自己从抓取中排除，断开「自己折射自己」反馈
```

- **目标视觉 = 极简玻璃**（shuding/liquid-glass 风格）：纯折射 + 轻微 blur/contrast/brightness/saturate，**无高光无阴影**。厚玻璃（archisvaze）不做。
- **已验证（spike A/B）**：`WDA_EXCLUDEFROMCAPTURE` 对 DComp 透明置顶窗有效——肉眼可见、Desktop Duplication 抓不到，反馈循环根除；Desktop Duplication 本机 60fps 可用。
- **为什么不用 backdrop-filter/WebView2**：CSS `backdrop-filter` 只采样 WebView 文档内部，采不到操作系统桌面 → 旧版显示成「黑框」。Windows 也无「对窗口背景做折射位移」的系统 API（DWM Acrylic 只有模糊无折射）。唯一出路是自取桌面像素 + 自写折射 shader。
- **参考实现**：`_liquid-glass-ref/`（shuding 原版，折射核蓝本）、`_liquid-glass-archisvaze/`（archisvaze，含 webgl.html GLSL 参考）。

### 状态机

四种状态，优先级从高到低：

```
红（执行中） > 黄（思考中） > 绿（空闲） > 灰（未运行）
```

| 状态 | 视觉效果 | 触发条件（hook 事件） |
|------|---------|---------|
| 灰 | 三灯全灭 | 状态文件不存在（Claude Code 从未运行过） |
| 绿 | 绿灯常亮 | `Stop`（回合结束、空闲） |
| 黄 | 黄灯闪烁 0.85s | `UserPromptSubmit` / `PostToolUse`（思考中） |
| 红 | 红灯闪烁 0.85s | `PreToolUse`（正在执行工具） |

### 状态探测：Claude Code Hooks（实时驱动）

不轮询 transcript（旧方案已废弃）。靠 Claude Code 的 4 个生命周期 hook 实时推送：挂件启动时 `installHooks` 把 hook 合并进 `~/.claude/settings.json`，每个 hook 调挂件自己 `claude-traffic-light.exe hook <state>`（**exec form**：`command`=exe 路径 + `args`，直接 spawn 不经 shell，避开 Windows「Git Bash or PowerShell」不确定性），写状态文件 `~/.claude/agent-light-state`；`watcher/` 每 100ms 读该文件映射四态。

- 事件→状态：`UserPromptSubmit`/`PostToolUse`→thinking、`PreToolUse`→running、`Stop`→idle
- **自动灭灯**：每 3s 用 `CreateToolhelp32Snapshot` 检测 `claude.exe` 进程，不在则切灰（关闭 Claude Code 后最多 3s 灭灯）
- **单 exe**：hook handler 就是挂件自己，零外部依赖（不像 agent-light 要 node）
- **安全合并**：幂等（已存在不重复加、路径变则更新）、先备份 `settings.json.bak`、只增不删别人的配置；靠 command 的 basename 识别「我加的那条」

### 窗口架构（DComp，方案2）

- **非分层窗**：`WS_POPUP | WS_EX_NOREDIRECTIONBITMAP | WS_EX_TOPMOST | WS_EX_TOOLWINDOW | WS_EX_NOACTIVATE`
  - **不用 `WS_EX_LAYERED`**——它与 `WDA_EXCLUDEFROMCAPTURE` 冲突（spike 验证：layered 窗设 WDA 失败）
- DirectComposition 承载透明：`DCompositionCreateDevice → CreateTargetForHwnd → CreateVisual → SetContent(swapchain) → Commit`，纯 GPU 每像素 alpha
- DXGI 合成 swapchain：`CreateSwapChainForComposition`，`DXGI_ALPHA_MODE_PREMULTIPLIED`
- `SetWindowDisplayAffinity(hwnd, WDA_EXCLUDEFROMCAPTURE=0x11)` 排除自身捕获
- 闪烁：renderThread 按 0.85s 周期算 blink phase（sin），shader 调制红/黄灯亮度
- 系统托盘 `NOTIFYICONDATAW` + `WM_TRAY` 消息（需 `NIF_MESSAGE` + 回调消息）

### 单实例

`CreateMutexW("Local\\ClaudeTrafficLight_SingleInstance")`，检测到 `ERROR_ALREADY_EXISTS` 直接退出。

### WebView2（已退役）

旧版用 `go-webview2` 渲染 `glass.html`，靠 `backdrop-filter` —— 因采不到桌面而显示为黑框，方案2 已弃用。透明背景 unsafe hack 见 memory `[[reference-webview2-transparency-hack]]`（仅留作历史）。

## 边界情况

| 情况 | 处理 |
|------|------|
| `settings.json` 不存在 | installHooks 创建只含 hook 的新文件 |
| `settings.json` 已有 hook | 幂等：路径变则更新、未变则不写；别人的 hook 原样保留 |
| HiDPI 125%/150% | 代码 `SetProcessDpiAwarenessContext` PerMonitorV2（替代 manifest，契合单 exe） |
| 窗口拖动 | `WM_NCHITTEST` 恒返回 `HTCLIENT`，自接管鼠标拖动；仅点在可见胶囊内才响应（胶囊外透明区不误拖） |
| 整体缩放 | 滑块窗设 `scale` → 渲染线程动态 resize swapchain/capture + 动态 viewport；shader CANVAS=240×128（含形变余量），pill 230×96 |
| 点击穿透 | DComp/`NOREDIRECTIONBITMAP` 窗无法穿透，已知限制，待重构 |

## 范围外

- macOS / Linux
- Claude Code 以外的 AI 工具
- 声音提示
- 多显示器自动定位
