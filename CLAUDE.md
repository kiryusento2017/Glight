# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目概述

**Claude Code Light** — Windows 桌面红绿灯挂件，实时显示 Claude Code 工作状态。单一 `.exe`，存于 U 盘即插即用。

## 构建命令

Go 不在 PATH 中，需完整路径：

```powershell
# 普通构建（带控制台，调试用）
C:\Open Source Projects\go\bin\go.exe build -trimpath -buildvcs=false -o claude-traffic-light.exe .

# 发布构建（无控制台窗口，输出到 dist/；dist 不存在先建）
C:\Open Source Projects\go\bin\go.exe build -trimpath -buildvcs=false -ldflags="-H windowsgui" -o dist\claude-traffic-light.exe .

# 运行测试
C:\Open Source Projects\go\bin\go.exe test ./...
```

工作目录：`D:\vs code projects\claude code light`

### 构建规则（强制，所有构建一律遵守）

**任何 `go build` 都必须带 `-trimpath -buildvcs=false`，调试构建也不例外。** 原因：Go 默认把编译机的绝对路径写进二进制行号表（pclntab），会泄露三类本机信息——GOROOT 路径（`C:\Open Source Projects\go`）、**Windows 用户名**（依赖 cache 路径 `C:\Users\<用户名>\go\pkg\mod\...`）、项目源码路径（`D:\vs code projects\...`）。

- `-trimpath`：把所有嵌入路径换成模块相对路径（`claude-traffic-light/main.go`），盘符/用户名/目录结构全清。**不删符号**，不会加重杀软误报（区别于 `-s -w`，后者反而触发 Wacatac 误报，严禁加）。
- `-buildvcs=false`：去掉嵌入的 git revision / 提交时间 / dirty 标记。
- 代价：调试时 panic 堆栈路径变成模块相对路径，但文件名+行号仍在，定位无碍。
- 验证清零：`grep -a -c "用户名" claude-traffic-light.exe` 应为 0；`go version -m exe` 不应出现绝对路径或 vcs 字段。
- **铁律：发行版 exe 必须输出到 `dist/` 文件夹**（`-o dist\claude-traffic-light.exe`），与源码/调试产物隔离。`dist/` 已 gitignore。go build 不自动建目录，dist 不存在时先 `New-Item -ItemType Directory -Force dist`（或 `mkdir dist`）。调试构建不受此约束（仍可输出到根目录）。

### exe 文件图标（踩坑警示）

窗口/托盘图标走 `main.go` 的 `//go:embed claude-traffic-light.ico`（运行时）；**exe 文件图标**（资源管理器里）是另一套，靠 `rsrc_windows_amd64.syso` 链接时嵌入。

- `.syso` 已 gitignore，缺失时重新生成：`go run github.com/akavel/rsrc@latest -ico claude-traffic-light.ico -o rsrc_windows_amd64.syso`
- **文件名必须是 `rsrc_windows_amd64.syso`**（带平台后缀 go build 才自动挑）。**严禁**再多生成一个 `rsrc.syso`——两个 `.syso` 含资源段 → 链接报 `too many .rsrc sections`，构建必失败。

## 架构

### 模块分层（目标架构，方案2）

```
main.go           — 入口：单实例互斥（含 --restarted 撞锁轮询）、加载配置、开机自启同步、安装 hook、居中、启动 watcher 和窗口；含 hook 子命令模式（从 stdin JSON 取 session_id 写每会话状态文件）+ //go:embed ico
hookinstall.go    — 把状态 hook 安全合并进 ~/.claude/settings.json（幂等：靠当前 exe 真名识别、备份/只增不删）
config/           — config.json（窗口位置/锁定/可见/缩放/开机自启）+ glass-tuning.json（视觉/形变参数）读写
  autostart.go    — HKCU Run 注册表读写删（开机自动启动）+ SyncAutostart 路径自校正 + ToggleAutostart 菜单开关
state/            — 四态枚举（Grey/Green/Yellow/Red）和优先级聚合
watcher/          — 100ms 轮询、聚合每会话状态文件（agent-light-state-<sid>）、忙态只信 hook 内容（无 mtime 超时）、cleanup 定时清理残留文件
  procmon.go      — 每 3s CreateToolhelp32Snapshot 进程检测（claude.exe 没了→灰，残留唯一兜底）
ui/               — 原生渲染与窗口管理（D3D11 + DComp + HLSL）
  window.go       — DComp 透明置顶窗、消息循环、托盘/菜单、自接管鼠标拖动（仅胶囊内响应）、SetState
  win32.go        — Win32 API 绑定（窗口样式、托盘、菜单、comctl32 滑块、WDA syscall）
  com.go          — 通用 comCall + D3D11/DXGI/DComp 绑定（含 swapchain ResizeBuffers）
  capture.go      — Desktop Duplication 桌面纹理获取（支持随缩放 Resize + 会话切换失效后限速重建）
  render.go       — 渲染管线：device/swapchain/shader 编译 + 每帧绘制（动态 viewport）
  glass.hlsl      — 折射 + 红绿灯 shader（移植自 shuding/liquid-glass）
  physics.go      — 二阶弹簧形变物理（Euler 积分驱动按压/拖动形变）
  sizedialog.go   — 调整大小滑块窗（comctl32 trackbar，右键菜单触发，100%~2000% 无极缩放）
```

> `config/`、`state/`、`watcher/` 有单元测试；`watcher/` 已改为 hook 状态文件驱动（见「状态探测」）；`ui/` 为方案2 原生渲染，已实现。

### 当前进度（2026-06-06）

- **方案2 已实现跑通**：DComp 透明置顶窗 + Desktop Duplication + 折射 shader + 四态红绿灯，实时折射真桌面、拖动顺滑、常亮。
- **状态检测已切到 Claude Code Hooks**（见「状态探测」），替代旧 transcript 轮询。
- **整体缩放**：右键菜单「调整大小…」弹 comctl32 滑块窗，100%~2000% 无极缩放（顶边固定、向下扩展），窗口/swapchain/桌面捕获动态 resize，`scale` 存 config.json 记忆。
- **开机自动启动**：右键菜单「开机自动启动」写 `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`（ToggleAutostart），启动时路径自校正（SyncAutostart）；`startup` 存 config.json。
- **运行时图标**：claude-traffic-light.ico（256/32/16）通过 `//go:embed` 嵌入 exe，`loadIconFromICO` 解析 ico，`CreateIconFromResourceEx` 设窗口 + 托盘图标（纯 Go 标准库，零外部工具）。
- **点击穿透暂不可行**：DComp/`NOREDIRECTIONBITMAP` 窗无法穿透（已坐实），折射优先、待收尾后重构。
- **右键菜单「重启」**：启动带 `--restarted` 标记的新实例，轮询等旧实例释放单实例锁后接管，用于卡帧等异常时一键恢复。
- **hook 识别幂等**：改用当前 exe 真名（`filepath.Base(exe)`）识别 settings.json 中自己加的 hook，杜绝 debug 版/正式版互不相认导致重复写入。

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

不轮询 transcript（旧方案已废弃）。靠 Claude Code 的 4 个生命周期 hook 实时推送：挂件启动时 `installHooks` 把 hook 合并进 `~/.claude/settings.json`（幂等：靠当前 exe 真名识别，杜绝 debug/正式版互不相认），每个 hook 调挂件自己 `claude-traffic-light.exe hook <state>`（**exec form**：`command`=exe 路径 + `args`，直接 spawn 不经 shell，避开 Windows「Git Bash or PowerShell」不确定性）。

**每会话独立状态文件**：hook 从 stdin JSON 读取 `session_id`，写 `~/.claude/agent-light/agent-light-state-<sid>` —— 每会话一个文件，统一放在 `agent-light/` 子目录下（避免在 `~/.claude/` 根目录摊一堆文件）。watcher 聚合该子目录下所有此前缀文件：**任一会话忙 → 全局忙**（防止多 agent 并发时一个结束误拉绿）；全 idle → 绿；无文件 → 灰。

- 事件→状态：`UserPromptSubmit`/`PostToolUse`→thinking、`PreToolUse`→running、`Stop`→idle
- **忙态只信 hook 内容（无时间窗口）**：`running`→红、`thinking`→黄、`idle`→绿，不做 mtime 超时降级。曾用 `freshWindow=120s` 陈旧降级，但「hook 静默」既可能是残留、也可能是 Claude 长思考（无工具调用→无 hook），任何固定阈值都两全不了（长思考会被误降绿），故彻底移除——残留/崩溃统一交给进程检测兜底
- **进程检测兜底（残留的唯一防线）**：每 3s 用 `CreateToolhelp32Snapshot` 检测 `claude.exe` 进程，不在则强制灰（无论文件内容）。崩溃/强杀/开机残留→claude.exe 不在→灰；正常结束/Ctrl+C 中断→`Stop` hook 写 idle→绿
- **定时清理（cleanupWindow=10min）**：每 30s 删除超 10min 未更新的残留文件，纯磁盘回收，与状态判定解耦
- **单 exe**：hook handler 就是挂件自己，零外部依赖（不像 agent-light 要 node）
- **安全合并**：幂等（已存在不重复加、路径变则更新）、先备份 `settings.json.bak`、只增不删别人的配置

### 窗口架构（DComp，方案2）

- **非分层窗**：`WS_POPUP | WS_EX_NOREDIRECTIONBITMAP | WS_EX_TOPMOST | WS_EX_TOOLWINDOW | WS_EX_NOACTIVATE`
  - **不用 `WS_EX_LAYERED`**——它与 `WDA_EXCLUDEFROMCAPTURE` 冲突（spike 验证：layered 窗设 WDA 失败）
- DirectComposition 承载透明：`DCompositionCreateDevice → CreateTargetForHwnd → CreateVisual → SetContent(swapchain) → Commit`，纯 GPU 每像素 alpha
- DXGI 合成 swapchain：`CreateSwapChainForComposition`，`DXGI_ALPHA_MODE_PREMULTIPLIED`
- `SetWindowDisplayAffinity(hwnd, WDA_EXCLUDEFROMCAPTURE=0x11)` 排除自身捕获
- 闪烁：renderThread 按 0.85s 周期算 blink phase（sin），shader 调制红/黄灯亮度
- 系统托盘 `NOTIFYICONDATAW` + `WM_TRAY` 消息（需 `NIF_MESSAGE` + 回调消息）

### 单实例

`CreateMutexW("Local\\ClaudeTrafficLight_SingleInstance")`，检测到 `ERROR_ALREADY_EXISTS`：
- 普通双开 → 直接退出
- 带 `--restarted` 标记（右键重启）→ 轮询等旧实例退出（100ms×50=5s 超时），拿到锁后接管；超时则放弃

### WebView2（已退役）

旧版用 `go-webview2` 渲染 `glass.html`，靠 `backdrop-filter` —— 因采不到桌面而显示为黑框，方案2 已弃用。透明背景 unsafe hack 见 memory `[[reference-webview2-transparency-hack]]`（仅留作历史）。

## 边界情况

| 情况 | 处理 |
|------|------|
| `settings.json` 不存在 | installHooks 静默跳过、不创建（未装 Claude Code，不污染 ~/.claude/） |
| `settings.json` 已有 hook | 幂等：路径变则更新、未变则不写；别人的 hook 原样保留 |
| HiDPI 125%/150% | 代码 `SetProcessDpiAwarenessContext` PerMonitorV2（替代 manifest，契合单 exe） |
| 窗口拖动 | `WM_NCHITTEST` 恒返回 `HTCLIENT`，自接管鼠标拖动；仅点在可见胶囊内才响应（胶囊外透明区不误拖） |
| 整体缩放 | 滑块窗设 `scale` → 渲染线程动态 resize swapchain/capture + 动态 viewport；shader CANVAS=240×144（含形变余量），pill 230×96 |
| 点击穿透 | DComp/`NOREDIRECTIONBITMAP` 窗无法穿透，已知限制，待重构 |
| 状态残留 | 崩溃/强杀未走 Stop → 状态文件卡 running/thinking → 进程检测发现 claude.exe 没了 → 强制灰，不卡黄/红 |
| 多 agent 并发 | 每会话独立文件 + 聚合取最高优先级，一个 agent 结束不误拉绿 |
| Desktop Duplication 失效 | 锁屏/解锁/UAC 会话切换后 GetImage 返回错误 → 限速 500ms 重建接口 + 重读 bounds，不停留在失效前最后一帧 |
| 竖向拖动峰值 | maxDragScaleY=1.4 钳制，避免 pill 拉长超出画布被削平 |

## 范围外

- macOS / Linux
- Claude Code 以外的 AI 工具
- 声音提示
- 多显示器自动定位
