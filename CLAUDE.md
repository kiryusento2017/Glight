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

## 架构

### 模块分层（目标架构，方案2）

```
main.go           — 入口：单实例互斥、加载配置、居中、启动 watcher 和窗口
config/           — config.json 读写（窗口位置、穿透开关、可见性）
state/            — 四态枚举（Grey/Green/Yellow/Red）和优先级聚合
watcher/          — 250ms 轮询 transcript.jsonl，解析状态变化
ui/               — 原生渲染与窗口管理（D3D11 + DComp + HLSL）
  window.go       — DComp 透明置顶窗、消息循环、托盘/菜单/拖动/穿透/显隐、SetState
  win32.go        — Win32 API 绑定（窗口样式、托盘、菜单、WDA syscall）
  com.go          — 通用 comCall + D3D11/DXGI/DComp 绑定与初始化
  capture.go      — Desktop Duplication 桌面纹理获取
  render.go       — 渲染管线：device/swapchain/shader 编译 + 每帧绘制
  glass.hlsl      — 折射 + 红绿灯 shader（移植自 shuding/liquid-glass）
```

> `config/`、`state/`、`watcher/` 已实现且有单元测试，保持不变；`ui/` 为重写目标。

### 当前进度（2026-06-03）

- **技术路线已 100% 验证、锁定方案2。实现尚未开始**——当前 `.exe` 仍是旧的 WebView2 SVG 玻璃（被「黑框」困扰的那版）。
- **实现计划**：`docs/superpowers/plans/2026-06-03-native-liquid-glass.md`（逐任务、带验证点）。
- **spike 实证**：`_spike_capture/`（最小程序证明命门可行，完工后删除）。

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

| 状态 | 视觉效果 | 触发条件 |
|------|---------|---------|
| 灰 | 三灯全灭 | 无活跃 session（60s 无写入） |
| 绿 | 绿灯常亮 | `user` / `tool_result` 是最新条目 |
| 黄 | 黄灯闪烁 0.85s | `assistant` 文本（非 tool_use） |
| 红 | 红灯闪烁 0.85s | `assistant` content 含 `tool_use` |

### 状态探测：轮询 transcript 文件

Claude Code 自动写入 `~/.claude/projects/<project-hash>/transcript.jsonl`。

**不是 ReadDirectoryChangesW**，而是 `filepath.Glob` + `os.Stat` 每 250ms 轮询。通过 ModTime 判断文件变化，解析最后几行 JSONL 推断状态。

parser.go 只解析必要字段：`type`（顶层）、`message.content[]` 中的 `type`（判断是否含 `tool_use`）。

### 窗口架构（DComp，方案2）

- **非分层窗**：`WS_POPUP | WS_EX_NOREDIRECTIONBITMAP | WS_EX_TOPMOST | WS_EX_TOOLWINDOW | WS_EX_NOACTIVATE`
  - **不用 `WS_EX_LAYERED`**——它与 `WDA_EXCLUDEFROMCAPTURE` 冲突（spike 验证：layered 窗设 WDA 失败）
- DirectComposition 承载透明：`DCompositionCreateDevice → CreateTargetForHwnd → CreateVisual → SetContent(swapchain) → Commit`，纯 GPU 每像素 alpha
- DXGI 合成 swapchain：`CreateSwapChainForComposition`，`DXGI_ALPHA_MODE_PREMULTIPLIED`
- `SetWindowDisplayAffinity(hwnd, WDA_EXCLUDEFROMCAPTURE=0x11)` 排除自身捕获
- 闪烁定时器 425ms（半周期），Go 端控制
- 系统托盘 `NOTIFYICONDATAW` + `WM_TRAY` 消息（需 `NIF_MESSAGE` + 回调消息）

### 单实例

`CreateMutexW("Local\\ClaudeTrafficLight_SingleInstance")`，检测到 `ERROR_ALREADY_EXISTS` 直接退出。

### WebView2（已退役）

旧版用 `go-webview2` 渲染 `glass.html`，靠 `backdrop-filter` —— 因采不到桌面而显示为黑框，方案2 已弃用。透明背景 unsafe hack 见 memory `[[reference-webview2-transparency-hack]]`（仅留作历史）。

## 边界情况

| 情况 | 处理 |
|------|------|
| `~/.claude/` 不存在 | 显示灰色，每 5 秒重试 |
| transcript 解析失败 | 回退为绿（保守估计空闲） |
| HiDPI 125%/150% | manifest 声明 PerMonitorV2 DPI-aware |
| 窗口拖动 | `WM_NCHITTEST` 返回 `HTCAPTION`，系统处理拖动 |
| 穿透模式 | 动态追加/移除 `WS_EX_TRANSPARENT` |
| 睡眠唤醒 | 重新扫描所有 transcript 文件 |

## 范围外

- macOS / Linux
- Claude Code 以外的 AI 工具
- 声音提示
- 多显示器自动定位
