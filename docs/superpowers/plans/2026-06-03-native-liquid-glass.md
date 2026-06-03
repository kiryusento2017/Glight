# 原生液态玻璃挂件 实现计划

> **For agentic workers:** 用 superpowers:subagent-driven-development 或 superpowers:executing-plans 逐任务实现。步骤用 `- [ ]` 勾选。
> **务实取舍（经用户确认）：** 图形/COM 部分无法单元测试，验证用「构建+运行+抓屏/肉眼实证」；COM 调用复用已验证的 `_spike_capture/b/main.go` 模式，计划指向它而非重复代码。state/watcher/config 已有的单元测试照常保留。

**Goal:** 把挂件从 WebView2 重写为原生 D3D11 + DirectComposition 管线，实时折射真实桌面，复刻 shuding/liquid-glass 极简玻璃，永远置顶、单实例、可拖动/穿透/显隐/退出。

**Architecture:** Desktop Duplication 抓整屏桌面纹理（GPU 常驻）→ HLSL 折射 shader（移植自 shuding `liquid-glass.js`，叠加红绿灯）→ DXGI 合成 swapchain → DirectComposition 透明置顶窗。窗口设 `WDA_EXCLUDEFROMCAPTURE` 把自己从抓取中排除，断开「自己折射自己」反馈。全程像素留在 GPU。

**Tech Stack:** Go 1.26、Win32 syscall、D3D11、DXGI Desktop Duplication（github.com/kirides/go-d3d）、DirectComposition、HLSL。

**已验证（spike）：** WDA 对 DComp 透明置顶窗排除有效（Spike A/B）；Desktop Duplication 本机可用 60fps；COM 调用模式跑通。详见 `_spike_capture/`。

---

## 命名约定

- **极简玻璃** = shuding/liquid-glass 风格（纯折射 + 轻微 blur/contrast/brightness/saturate，无高光无阴影）。**本项目唯一目标视觉。**
- 厚玻璃（archisvaze）= 不做。

## 渲染原理（折射核，来自 shuding）

逐像素逻辑，移植自 `_liquid-glass-ref/liquid-glass.js:23-27, 265-278`：

```
ix = uv.x - 0.5; iy = uv.y - 0.5
d  = roundedRectSDF(ix, iy, halfW, halfH, radius)
disp   = smoothstep(0.8, 0.0, d - 0.15)
scaled = smoothstep(0.0, 1.0, disp)
sampleUV = (ix*scaled + 0.5, iy*scaled + 0.5)   // 向心收缩
color = desktop.Sample(sampleUV)
color = saturate(brightness/contrast/saturate 微调)   // 1.05 / 1.2 / 1.1
```

背景源从「DOM」换成「Desktop Duplication 桌面纹理」，其余照搬。

---

## 文件结构

| 文件 | 职责 | 动作 |
|---|---|---|
| `main.go` | 入口：单实例、居中、启动 watcher + 窗口 | 微调（居中逻辑） |
| `config/`、`state/`、`watcher/` | 配置、四态、轮询 | **保留不动**（已工作、有测试） |
| `ui/window.go` | DComp 透明置顶窗 + 消息循环 + 交互（拖动/穿透/托盘/菜单/显隐）+ SetState | **重写**（WebView2→DComp） |
| `ui/win32.go` | Win32 窗口/样式/托盘/菜单 syscall | 扩展（WDA、ShowWindow、显隐修复） |
| `ui/com.go` | 通用 comCall + D3D11/DXGI/DComp 绑定与初始化 | **新增**（提炼自 spike B） |
| `ui/capture.go` | Desktop Duplication 桌面纹理获取 | **新增** |
| `ui/render.go` | 渲染管线：device/swapchain/shader 编译 + 每帧绘制 | **新增** |
| `ui/glass.hlsl` | 折射 + 红绿灯 shader（embed） | **新增** |
| `ui/embed.go`、`ui/glass.html` | WebView2 SVG 玻璃 | **删除** |

**接口契约（main.go 不变）：** `ui.New(cfgPath string, cfg config.Config) *Window`、`(*Window).SetState(state.State)`、`(*Window).Run()`。

---

## Task 1: 引入依赖，退役 WebView2

**Files:** Modify `go.mod`；Delete `ui/embed.go`, `ui/glass.html`；临时桩 `ui/window.go`

- [ ] **Step 1:** `go get github.com/kirides/go-d3d@v1.0.1`
- [ ] **Step 2:** 删除 `ui/embed.go`、`ui/glass.html`
- [ ] **Step 3:** 把 `ui/window.go` 暂时改为最小桩：`New` 返回空 `*Window`，`SetState`/`Run` 空实现（仅为编译通过，后续 Task 重写）
- [ ] **Step 4:** `go.mod` 移除 `go-webview2`（`go mod tidy`）
- [ ] **Step 5 验证:** `go build -o claude-traffic-light.exe .` 通过；运行不崩（无窗口）
- [ ] **Step 6:** commit `chore: 退役 WebView2，引入 go-d3d`

## Task 2: COM 绑定层 `ui/com.go`

**Files:** Create `ui/com.go`

- [ ] **Step 1:** 提炼 spike B 的 `comCall(this, idx, args...)`、四个 IID、`dxgiSwapChainDesc1` 结构（见 `_spike_capture/b/main.go:140-180`）
- [ ] **Step 2:** 写 `createD3D11Device() (dev, ctx uintptr, err error)`（BGRA_SUPPORT，见 spike B `D3D11CreateDevice` 调用）
- [ ] **Step 3:** 写 `createCompositionSwapchain(factory, dev, w, h uint32) (swapchain uintptr)` 和 `createDCompForHwnd(dxgiDevice, hwnd) (dcompDevice, target, visual uintptr)`（封装 spike B 的装配序列）
- [ ] **Step 4 验证:** `go build` 通过
- [ ] **Step 5:** commit `feat: COM/D3D/DComp 绑定层`

## Task 3: DComp 透明置顶窗骨架（window.go 重写第一阶段）

**Files:** Rewrite `ui/window.go`；Modify `ui/win32.go`

- [ ] **Step 1:** `win32.go` 增 `procSetWindowDisplayAffinity`、`procShowWindow`、`procDestroyWindow`、常量 `WDA_EXCLUDEFROMCAPTURE=0x11`、`WS_EX_NOREDIRECTIONBITMAP=0x00200000`
- [ ] **Step 2:** `window.go` 重写：注册类、`CreateWindowEx`（NOREDIRECTIONBITMAP|TOPMOST|TOOLWINDOW|NOACTIVATE, WS_POPUP）、建 DComp（Task 2 函数）、先用纯色 swapchain 内容、设 WDA、自建消息循环 `Run()`
- [ ] **Step 3:** 居中：`main.go` 启动位置用 `windowCenter(winW)`，仅当 `cfg.X>=0` 用保存位置（修掉当前无条件覆盖）
- [ ] **Step 4:** 接回交互：`WM_NCHITTEST→HTCAPTION` 拖动（替代 WebView 的 JS 拖动）、右键 `WM_CONTEXTMENU` 弹菜单、托盘
- [ ] **Step 5:** 修复 **显隐 bug**：菜单/托盘切 `Visible` 时真正 `ShowWindow(hwnd, SW_HIDE/SW_SHOWNOACTIVATE)`，并存 config
- [ ] **Step 6:** 修复 **托盘无回调**：`addTrayIcon` 加 `NIF_MESSAGE`+`UCallbackMessage`，`WM_TRAY` 里右键弹同一菜单
- [ ] **Step 7 验证（实证）:** 运行 → 透明置顶纯色块居中出现；可拖动；托盘右键菜单的退出/穿透/显隐**全部真生效**；跑 `_spike_capture/spikeB.exe` 思路抓屏确认窗口被排除
- [ ] **Step 8:** commit `feat: DComp 透明置顶窗 + 交互修复`

## Task 4: Desktop Duplication 桌面纹理 `ui/capture.go`

**Files:** Create `ui/capture.go`

- [ ] **Step 1:** 封装 `go-d3d`：`newCapture(dev, ctx) (*Capture, error)`，内部 `outputduplication.NewIDXGIOutputDuplication`，DPI 设 PerMonitorV2（见 spike）
- [ ] **Step 2:** `(*Capture) AcquireTexture() (texSRV uintptr, ok bool)` —— 拿当前桌面帧作为 shader 可采样纹理（ShaderResourceView）。注意复用 go-d3d 的 staged texture 或直接对 desktop2d 建 SRV
- [ ] **Step 3 验证:** 临时把桌面纹理直接 present 到窗口（不折射），确认窗口里显示的是「窗口背后那块桌面」且实时刷新
- [ ] **Step 4:** commit `feat: Desktop Duplication 桌面纹理`

## Task 5: 折射 shader + 渲染管线 `ui/glass.hlsl` + `ui/render.go`

**Files:** Create `ui/glass.hlsl`, `ui/render.go`

- [ ] **Step 1:** `glass.hlsl`：全屏三角顶点 + 像素着色器，实现上文「渲染原理」折射核（`roundedRectSDF` + 双 `smoothstep` + 向心采样桌面 + 色彩微调）。uniform：玻璃在屏幕的矩形、圆角、桌面纹理尺寸
- [ ] **Step 2:** `render.go`：`D3DCompile` 编译 hlsl；建采样器、constant buffer；`(*Renderer) Frame(desktopSRV, glassRect)` 绘制到 swapchain 后台缓冲并 `Present`
- [ ] **Step 3:** render loop：`window.go` 的 `Run()` 消息循环里，每帧 `capture.AcquireTexture()` → `renderer.Frame()` → DComp `Commit`（首帧后内容自更新）
- [ ] **Step 4 验证（实证）:** 运行 → 玻璃**实时折射窗口背后的真实桌面**，拖动时折射内容跟随；**无黑框、无截图死图**
- [ ] **Step 5:** commit `feat: shuding 折射 shader + 渲染管线`

## Task 6: 红绿灯叠加 + 四态 + 闪烁

**Files:** Modify `ui/glass.hlsl`, `ui/render.go`, `ui/window.go`

- [ ] **Step 1:** `glass.hlsl` 叠加：折射结果之上画三个圆灯（圆 SDF + 发光 halo），uniform `uActive`(0灰/1绿/2黄/3红)、`uBlinkPhase`(0~1)
- [ ] **Step 2:** 灯色与发光参数对齐旧 `glass.html`（红 `#e8302a`、黄 `#f0c040`、绿 `#30c040` + glow）
- [ ] **Step 3:** 闪烁：Go 端按 425ms 半周期更新 `uBlinkPhase`，红/黄闪、绿常亮、灰全灭
- [ ] **Step 4:** `SetState(s)`：线程安全设置 `uActive`，触发重绘
- [ ] **Step 5 验证（实证）:** 手动调用 SetState 四态切换，红黄闪烁、绿常亮、灰灭，灯浮在折射玻璃上
- [ ] **Step 6:** commit `feat: 红绿灯叠加 + 四态闪烁`

## Task 7: 状态接入

**Files:** Modify `main.go`（如需）

- [ ] **Step 1:** 确认 `watcher.New(..., func(s){ win.SetState(s) })` 已接通（main.go 现有结构基本可用）
- [ ] **Step 2 验证（实证）:** 真实 Claude Code 会话：执行中→红闪、思考→黄闪、空闲→绿、无会话→灰
- [ ] **Step 3:** commit `feat: 状态接入实时驱动`

## Task 8: 视觉对照，调到 100%

**Files:** 临时对照页（用完删）

- [ ] **Step 1:** 搭对照：一张固定桌面截图作背景，左跑 shuding 原版 `liquid-glass.js`，右是 HLSL 同尺寸/同圆角输出
- [ ] **Step 2:** 逐项对齐：圆角、位移强度（`smoothstep` 阈值 0.8/0.15）、blur 0.25、contrast 1.2、brightness 1.05、saturate 1.1
- [ ] **Step 3 验证:** 肉眼/逐像素无可见差异
- [ ] **Step 4:** commit `fix: 折射参数对齐 shuding 原版`

## Task 9: 发布构建

- [ ] **Step 1:** 确认 DPI manifest（PerMonitorV2）随 release 嵌入
- [ ] **Step 2:** `go build -ldflags="-H windowsgui" -o claude-traffic-light.exe .`
- [ ] **Step 3 验证:** 双击无控制台；125%/150% DPI 下尺寸/位置正确；睡眠唤醒重扫
- [ ] **Step 4:** commit `build: release 构建`

## Task 10: 文档与清理

- [ ] **Step 1:** 更新 `CLAUDE.md`（架构改为原生方案2、文件清单、命名约定、现状）—— **注：本轮已先行更新**
- [ ] **Step 2:** 删除 `_spike_capture/`
- [ ] **Step 3:** commit `docs: 更新 CLAUDE.md，清理 spike`

---

## 自审

- **Spec 覆盖：** 居中(T3)、单实例(已有 mutex)、置顶(T3)、托盘退出/穿透/显隐(T3 含 bug 修复)、实时折射真桌面(T4-T5)、shuding 效果 100%(T8)、四态(T6-T7)、无框无截图(T5) —— 全覆盖。
- **接口一致：** `ui.New/SetState/Run` 全程不变；`comCall`、`AcquireTexture`、`Frame`、`uActive/uBlinkPhase` 命名前后一致。
- **依赖风险：** go-d3d 只做 duplication；DComp/swapchain/shader 自写，模式已由 spike B 实证。
- **未决工程细节（实现期处理，不阻塞）：** 桌面纹理 SRV 的最高效路径（共享纹理 vs staged）、render loop 是否做脏区/变化驱动省电（MVP 先持续渲染）、多显示器选择哪个 output。
