# Claude Traffic Light — 设计规格

**日期**：2026-06-03  
**状态**：已确认，待实现

---

## 1. 项目概述

一个运行在 Windows 上的虚拟红绿灯桌面挂件，实时显示 Claude Code 的工作状态。灵感来自 DIY 硬件红绿灯项目，但以纯软件方式实现，不需要任何硬件设备。

**核心约束**：
- 单一 `.exe` 文件，可存于 U 盘，插入任何 Windows 电脑直接运行
- 不修改目标电脑的任何配置文件
- 低功耗，低内存，快速响应

---

## 2. 视觉设计

### 2.1 窗口形态

- **形状**：灵动岛式胶囊形（`border-radius: 999px`），长宽比约 3:1
- **材质**：iOS 液态玻璃（Liquid Glass）风格
  - 玻璃边缘折射：SVG `feDisplacementMap` + Canvas 生成斯涅尔定律位移图，仅 bezel 环形区域（边缘 10px）有折射，中心透明
  - `backdrop-filter: url(#lip-filter)` 将折射作用于元素背后的真实桌面内容
  - 顶部镜面高光：`::before` 线性渐变层（白色 22%→0%），模拟玻璃顶面反光
  - 边框：顶部高光线 `rgba(255,255,255,0.28)`，底部收暗
  - 轻微磨砂：`feGaussianBlur stdDeviation=1.5`（可调）
  - 目标效果：透过胶囊能看清背景文字，边缘有可见的光学折射扭曲
- **尺寸**：中号，灯直径 24px，内边距 13px × 24px，灯间距 17px
- **限制**：`backdrop-filter: url(#svg-filter)` 仅 Chrome/Edge 支持，WebView2（系统内置 Edge）满足此要求

### 2.2 四种状态

| 状态 | 视觉 | 触发条件 |
|------|------|---------|
| 未运行 | 三灯全灰 | `~/.claude/projects/` 目录无活跃 session |
| 空闲 | 绿灯常亮，红/黄灰 | Claude Code 启动或本轮回复结束 |
| 思考中 | 黄灯闪烁（0.85s 周期），红/绿灰 | 用户提交 prompt 后，工具调用前 |
| 执行中 | 红灯闪烁（0.85s 周期），黄/绿灰 | 正在调用工具（Bash、Write、Edit 等） |

### 2.3 灯罩质感

每盏灯用径向渐变模拟玻璃灯罩：
- 顶部高光点（白色椭圆，80% 不透明）
- 底部柔和折射（白色椭圆，18% 不透明）
- 点亮时有外发光 halo（`box-shadow` 双层扩散）
- 熄灭时深色渐变，几乎不反光

### 2.4 交互动效

按下与拖动均有 Spring 物理动画（Euler 积分）：

| 事件 | 效果 | Spring 参数 |
|---|---|---|
| 鼠标按下 | 等比放大至 1.08x | stiffness:340, damping:20 |
| 释放 | 弹回 1.0x | 同上 |
| 快速拖动 | scaleY 压缩（最低 0.7x）、scaleX 对等拉伸 | stiffness:340, damping:30 |
| 按下 | 外阴影加深、双向 inset 阴影出现（"压入玻璃"感） | stiffness:220, damping:24 |

液态拉伸公式（与 Apple 官方实现一致）：
- `scaleY = u × max(0.7, 1 − |velocityX| / 5000)`
- `scaleX = u + (u − scaleY)` — 保证体积守恒

---

## 3. 交互设计

### 3.1 窗口行为

- **默认位置**：屏幕顶部居中，距顶边 16px
- **始终置顶**：高于普通窗口，不覆盖系统级模态弹窗（Windows z-order 自动保证）
- **可拖动**：鼠标按住窗口任意位置拖动
- **位置记忆**：拖动后坐标写入 `config.json`（exe 同级目录），下次启动恢复

### 3.2 窗口穿透

- **穿透模式开启**：鼠标点击穿透窗口，直接操作背景内容；窗口仍可见
- **穿透模式关闭**：正常模式，可拖动窗口
- 通过系统托盘右键菜单切换，状态持久化到 `config.json`

### 3.3 系统托盘

- 最小化时收入右下角系统托盘，显示一个小图标（颜色随当前状态变化）
- **右键菜单**：
  - 显示 / 隐藏窗口
  - 开关窗口穿透（含当前状态勾选）
  - 退出

---

## 4. 技术架构

### 4.1 语言与运行时

**Go 语言** + **WebView2**（系统内置 Edge 渲染引擎）。

| 层 | 技术 | 职责 |
|---|---|---|
| 后端逻辑 | Go + `golang.org/x/sys/windows` | 文件监听、状态机、配置、系统托盘 |
| UI 渲染 | `go-webview2`（`github.com/jchv/go-webview2`）| 嵌入 `ui/glass.html`，渲染液态玻璃 |
| 通信 | `webview.Eval("setState('running')")` | Go → WebView2 注入 JS |
| 窗口行为 | Win32（拖动、置顶、穿透）| WebView2 窗口的 Win32 句柄直接操作 |

**为什么不用纯 Win32 + GDI+**：液态玻璃需要 `backdrop-filter: url(#svg-filter)`，这是浏览器专属 API，GDI+ 无法实现。WebView2 使用系统内置的 Edge 运行时（Windows 10 1803+ 和所有 Windows 11 预装），**不需要单独安装浏览器**。

**为什么不用 Electron**：Electron 打包 Chromium 约 150MB；WebView2 用系统已有的 Edge，exe 本身仅 ~8MB。

### 4.2 状态探测：Transcript 文件监听

**零配置方案**，不修改任何配置文件。

Claude Code 运行时自动在以下路径写入对话记录：

```
~/.claude/projects/<project-hash>/transcript.jsonl
```

所有客户端（CLI、VS Code 扩展、Obsidian 插件）均使用同一路径。

**监听机制**：
1. 使用 Windows API `ReadDirectoryChangesW` 监听 `~/.claude/projects/` 目录树变化
2. 文件新增 → 新 session 启动
3. 文件内容追加 → 解析最后几行 JSONL，判断当前状态
4. 全部 transcript 文件 60 秒无变化 → 灰（未运行）

**JSONL 状态解析规则**：

| transcript 最新条目类型 | 推断状态 |
|---|---|
| `tool_use`（在 assistant content 中，无对应 result） | 红（执行中） |
| `assistant` 文本内容（非工具调用） | 黄（思考中） |
| 用户消息写入后、assistant 首行输出前 | 黄（思考中）— **注：此窗口期 transcript 无写入，黄灯滞后 1–3s，属已知局限** |
| `user` 消息 / `tool_result` | 绿（空闲，等待） |
| 无文件 / 全部文件静止 > 30s | 灰（未运行） |

### 4.3 多会话并发

同时存在多个 session 时（VS Code + Obsidian 同时运行），取所有活跃 session 的最高优先级状态：

```
红（执行中） > 黄（思考中） > 绿（空闲） > 灰（未运行）
```

活跃定义：transcript 文件在过去 **60 秒**内有写入（给用户思考/慢速输入留足余量，避免频繁闪烁到灰色）。

### 4.4 文件结构

```
claude-traffic-light.exe   ← 主程序（单文件）
config.json                ← 自动生成，存窗口位置、穿透开关等
```

`config.json` 示例：
```json
{
  "x": 860,
  "y": 16,
  "click_through": false,
  "visible": true
}
```

### 4.5 GUI 渲染

**双层架构**：WebView2 负责玻璃渲染，Win32 负责窗口行为。

**WebView2 层（`ui/glass.html` 嵌入二进制）**：
- 液态玻璃效果：SVG `feDisplacementMap` + 斯涅尔定律位移图 + Spring 物理动效
- 灯罩：CSS 径向渐变 + 双层高光 + `box-shadow` 外发光
- 状态切换：Go 调用 `webview.Eval("setState('running')")` 注入 JS
- 拖动位移：WebView2 通过 `window.chrome.webview.postMessage({type:'move', dx, dy})` 上报给 Go

**Win32 层（`golang.org/x/sys/windows`）**：
- 无边框、无标题栏、不在任务栏显示（`WS_EX_TOOLWINDOW`）
- 始终置顶（`WS_EX_TOPMOST`）
- 穿透模式：动态切换 `WS_EX_TRANSPARENT`
- 接收 WebView2 拖动消息 → `SetWindowPos` 更新位置

**已验证的 HTML/JS 源文件**：`ui/glass.html`（可直接作为 WebView2 内容加载）

---

## 5. 边界情况处理

| 情况 | 处理方式 |
|------|---------|
| `~/.claude/` 目录不存在 | 显示灰色，每 5 秒重试检测 |
| transcript 格式变化（CC 新版本） | 解析失败时回退为绿（保守估计为空闲） |
| `~/.claude/projects/` 读取权限不足 | 弹出一行提示后继续运行，降级为灰 |
| 电脑从睡眠唤醒 | 重新扫描所有 transcript 文件，重置状态 |
| HiDPI（125%/150% 缩放） | exe manifest 声明 DPI-aware，坐标使用逻辑像素 |

---

## 6. 不在范围内

- macOS / Linux 支持（当前仅 Windows）
- Claude Code 以外的 AI 工具状态（Copilot 等）
- 声音提示
- 多显示器自动定位（用户手动拖到目标屏幕即可）
