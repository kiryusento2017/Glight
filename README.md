# Claude Code Light 🚦

Windows 桌面液态玻璃红绿灯挂件，实时显示 [Claude Code](https://claude.ai/code) 工作状态。**上班 vibe coding 时余光一扫就知道 AI 在干嘛。**

## 系统要求

| 要求 | 说明 |
|---|---|
| **操作系统** | **仅 Windows 11（64 位）**。不支持 macOS / Linux（依赖 DirectComposition + Desktop Duplication，无跨平台计划） |
| **Claude Code** | 需本机已安装 [Claude Code](https://claude.ai/code) CLI。这是一个**专为 Claude Code 设计**的状态指示器，不支持 Cursor / Copilot / 其他 AI 工具 |
| **运行时** | 无。单个原生 `.exe`，不依赖 Node / .NET / 任何框架 |

## 效果

一块悬浮在桌面上的**液态玻璃胶囊**，透过它能看到真实桌面被折射扭曲。中间三盏红绿灯实时反映 Claude Code 状态：

| 灯 | 含义 | Claude Code 状态 |
|---|---|---|
| 🔴 红灯闪烁 | 正在执行工具 | PreToolUse |
| 🟡 黄灯闪烁 | 正在思考 | UserPromptSubmit / PostToolUse |
| 🟢 绿灯常亮 | 空闲等待 | Stop |
| ⚫ 全灭 | 未运行 | Claude Code 进程不在 |

按住玻璃会**果冻形变**（横向变窄、纵向拉长），拖得越快越窄，松手 Q 弹回弹——所有弹簧参数可在 `glass-tuning.json` 里热调。

**整体缩放**：右键「调整大小…」弹出滑块窗，100%~2000% 无极缩放整个挂件（顶边固定、向下扩展），大小和位置自动记忆，下次启动恢复；可一键重置回默认。

## 运行流程

双击 `.exe` 后，挂件按以下顺序启动（**全程不修改系统文件**）：

1. **单实例检查** — 已在运行则直接退出，不会开第二个。
2. **加载配置** — 读 exe 同目录的 `config.json`（位置/缩放/可见/开机自启）和 `glass-tuning.json`（视觉/形变参数）。`glass-tuning.json` 不存在则**自动生成默认文件**（供用户编辑调参）；`config.json` 不存在则**用内存默认值**（不写盘，等用户操作后才首次生成）。
3. **同步开机自启** — 若上次开启过，校正注册表里的 exe 路径。
4. **安装 hook（分支）**：
   - **装了 Claude Code**（`~/.claude/settings.json` 存在）→ 幂等合并 4 条状态 hook（先备份 `.bak`、只增不删、已存在不重复加）。
   - **没装 Claude Code**（`~/.claude/` 不存在）→ **静默跳过，不创建任何文件或目录**。
5. **建窗 + 监测** — 创建透明置顶玻璃窗，watcher 每 100ms 读状态文件、每 3s 检测 `claude.exe` 进程。

**装了 Claude Code**：灯随 Claude 工作实时变色（红=执行、黄=思考、绿=空闲），关掉 Claude 后最多 3s 三灯熄灭。
**没装 Claude Code**：挂件照常显示液态玻璃，但检测不到状态，三灯**恒灰**，且不往 `~/.claude/` 写任何东西。

## 关于 Claude Code

这个挂件靠 **Claude Code 的 Hooks 机制**实时感知状态，而非轮询 transcript：

- **首次运行**检测 `~/.claude/settings.json`：如果存在（本机已装 Claude Code）则幂等合并 4 条 hook 规则（先备份、只增不删、已存在不重复加）；**如果不存在则静默跳过，不创建任何文件或目录**——不在没装 Claude Code 的机器上写任何东西。
- 每次 Claude Code 触发生命周期事件，hook 直接调挂件自己写状态文件，挂件 100ms 内反映到灯上。
- **自动灭灯**：每 3s 检测 `claude.exe` 进程，关掉 Claude Code 后最多 3s 三灯熄灭。
- hook handler 就是挂件 `.exe` 自身（`claude-traffic-light.exe hook <state>`），**零外部依赖**。

> 如果本机没装 Claude Code，挂件能正常运行，但因检测不到状态会一直保持灰色（三灯全灭）。

## 安装与使用

1. 从 [Releases](../../releases) 下载 `claude-traffic-light.exe`
2. 双击运行（首次启动自动生成 `glass-tuning.json` 供调参；若已装 Claude Code 则自动安装 hook）
3. 打开 Claude Code，开始 vibe coding
4. **拖动**：按住可见胶囊拖拽移位（点胶囊外的透明区无反应）
5. **右键菜单**（顺序同代码）：
   - **显示/隐藏** — 切换挂件可见，隐藏后从托盘图标恢复
   - **固定位置** — 锁定后不可拖动，防误触移位
   - **开机自动启动** — 写入 `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`（不弹 UAC，取消勾选即删记录）
   - **调整大小…** — 弹出滑块窗，100%~2000% 无极缩放，松手存盘、手动关闭
   - **重置大小和位置** — 回 100%、屏幕顶部居中
   - **重启** — 卡帧/异常时一键重启（新实例轮询等旧实例释放锁后接管，绝不"关了没开"）
   - **退出**

**调参**：编辑 exe 同目录的 `glass-tuning.json`（首次运行自动生成），保存即实时生效，无需重启。

**系统托盘**：启动后任务栏右下角出现挂件图标。**右键托盘图标**弹出与右键窗口相同的菜单；**双击托盘图标**显示/隐藏窗口。

## 写入的文件

首次运行及后续使用中，挂件会在以下位置读/写文件。**不修改任何系统文件。**

| 路径 | 内容 | 说明 |
|---|---|---|
| `~/.claude/settings.json` | 4 条 hook 规则 | 首次启动幂等写入（备份 → 合并 → 写回） |
| `~/.claude/settings.json.bak` | 修改前的原文件 | hook 配置有变动时创建（exe 首次安装/换盘/改名后） |
| `~/.claude/agent-light/agent-light-state-<session_id>` | 状态词（`idle`/`thinking`/`running`），每会话一个文件 | 每次 Claude Code hook 触发时覆盖写入 |
| `./config.json` | 位置 + 锁定/可见/缩放/开机自启 | 拖动松手 / 菜单操作 / 调整大小时保存，exe 同目录 |
| `./glass-tuning.json` | 全部视觉与形变参数 | 首次运行自动生成，手工编辑热重载 |

> 若开启「开机自动启动」，会在 `HKCU\Software\Microsoft\Windows\CurrentVersion\Run` 写一条注册表记录（用户空间，不弹 UAC）。取消勾选后自动删除记录。

## 调参：默认参数与范围

编辑 exe 同目录的 `glass-tuning.json`，**保存即热重载（~500ms 生效），无需重启或重编译**。该文件每台机器各一份、不进版本库；删掉它下次启动会用下表默认值重新生成。

> 默认值的权威来源是 `config/tuning.go` 的 `DefaultTuning()`（编译进 exe）。新用户没有 json，首次运行即得这套默认。

**视觉参数**

| 参数 | 中文 | 作用 | 默认 | 建议范围 |
|---|---|---|---|---|
| `cornerR` | 圆角半径(px) | 越大越圆 | 48 | 0~48 真圆角矩形；=48 短边已全圆；48~115 趋向跑道形 |
| `cornerN` | 角部曲率指数 | 2=标准圆，越大越方（苹果味 G2） | 2.1 | 2.0~4.0 |
| `refractBand` | 折射带深度(px) | 仅距边缘这么深处折射 | 3 | 1~30（小=只极边缘，大=大面积折射）|
| `edgeSqueeze` | 边缘收缩 | 0=折射最强，1=不折射 | 0.25 | 0~1 |
| `contrast` | 对比度 | 1=原样 | 1.2 | 0.5~2.0 |
| `brightness` | 亮度 | 1=原样，<1 变暗 | 0.9 | 0.5~2.0 |
| `saturate` | 饱和度 | 0=灰，1=原样 | 1.5 | 0~3 |
| `lampR` | 灯半径(px) | 灯大小 | 19 | 6~30（>32 邻灯重叠）|
| `lampGap` | 灯间距(px) | 红↔黄↔绿中心距 | 64 | 38~90（<2×lampR 重叠，>95 出胶囊）|
| `glow` | 点亮外发光 | 亮灯光晕强度 | 0 | 0~1（注：加法叠加，调大且背景偏暗时会洗白成光环，慎用）|

**物理形变参数**

| 参数 | 中文 | 作用 | 默认 | 建议范围 |
|---|---|---|---|---|
| `springK` | 弹簧刚度 | 越大回弹越快越硬 | 120 | 30~300 |
| `springC` | 阻尼 | 越小过冲/弹动越明显 | 8 | 1~20 |
| `steadyX` | 稳态水平缩放 | <1 静止时偏窄 | 0.91 | 0.8~1.04 |
| `steadyY` | 稳态垂直缩放 | >1 静止时偏长 | 1.11 | 0.9~1.30 |
| `pressX` | 按下水平缩放 | <1 按下变窄 | 0.82 | 0.5~1.04 |
| `pressY` | 按下垂直缩放 | >1 按下变高 | 1.22 | 0.8~1.30 |
| `dragK` | 拖动形变力度 | 越大拖时形变越猛 | 0.02 | 0.001~0.05 |
| `dragMin` | 拖动形变下限 | 0.5=最多缩到 50% | 0.5 | 0.3~1.0 |
| `releaseImpulse` | 松手过冲倍率 | >1 强化回弹 | 1.5 | 1.0~3.0 |

> ⚠️ **形变有硬上限，超了会被画布裁切**：画布 240×144、玻璃 230×96，所以任意时刻 **水平缩放 ≤ 240/230 ≈ 1.04、垂直缩放 ≤ 144/96 = 1.50**。`steadyY` 叠加 `pressY` 与松手过冲的峰值一旦超过 1.50，胶囊顶/底会被切平。竖向拖动已内置 `maxDragScaleY=1.4` 钳制防止撞墙。要更夸张的拉伸，得改 `ui/glass.hlsl` 的 `CANVAS`/`PILL`（涉及窗口重建，需重编译，非热重载）。

## 架构

```
main.go             入口：单实例互斥（--restarted 轮询） → 加载配置 → 开机自启同步 → 安装 hook（幂等：exe真名） → 创建窗口 → 启动监测
hookinstall.go      把状态 hook 幂等合并进 ~/.claude/settings.json
config/             配置读写（config.json 位置/缩放/开机自启 + glass-tuning.json 视觉热重载）
  autostart.go      注册表 HKCU Run 读写删 + 路径自校正（开机自动启动）
state/              四态枚举（灰/绿/黄/红）及优先级
watcher/            每 100ms 聚合每会话状态文件（任一会话忙=忙）+ 每 3s 进程检测兜底灭灯（procmon.go）
ui/
  window.go           DComp 透明置顶窗、消息循环、自接管鼠标拖动、弹簧形变状态机、图标加载
  render.go           D3D11 渲染管线：device/swapchain/shader 编译 + 每帧绘制（动态 viewport）
  glass.hlsl          像素 shader：超椭圆 SDF 限定形状、shuding 折射核、三灯叠加
  capture.go          Desktop Duplication 抓取桌面纹理（支持随缩放 Resize + 会话切换失效后限速重建）
  com.go              D3D11/DXGI/DComp COM 绑定（含 swapchain ResizeBuffers）
  win32.go            Win32 API 绑定与常量
  physics.go          二阶弹簧物理（每帧 Euler 积分，驱动形变）
  sizedialog.go       调整大小滑块窗（comctl32 trackbar，100%~2000% 无极缩放）
```

### 状态探测：Claude Code Hooks

不轮询 transcript。靠 4 个生命周期 hook 实时推送：

```
UserPromptSubmit → 黄（思考中）
PostToolUse      → 黄（思考中）
PreToolUse       → 红（执行中）
Stop             → 绿（空闲）
```

Hook 以 exec form 安装（`command`=exe 路径 + `args`，直接 spawn 不经 shell）。每个 hook 从 stdin JSON 读取 `session_id`，写 **每会话独立状态文件** `~/.claude/agent-light/agent-light-state-<sid>`。watcher 每 100ms 聚合所有会话文件：**任一会话忙 → 全局忙**（防多 agent 并发时一个结束误拉绿）；全 idle → 绿；无文件 → 灰。

- **忙态只信 hook 内容**：`running`→红、`thinking`→黄、`idle`→绿，不做时间窗口超时降级（长思考无工具调用→无 hook→若按超时会被误降绿）
- **进程检测兜底灭灯**：每 3s 用 `CreateToolhelp32Snapshot` 检测 `claude.exe`，不在则强制灰——崩溃/强杀/开机残留统一靠这条回灰，根治开机卡黄灯
- **定时清理（10min）**：每 30s 清理超 10min 未更新的残留文件（纯磁盘回收）

### 渲染管线

```
Desktop Duplication 抓整屏桌面纹理（GPU 常驻）
  → HLSL 超椭圆 SDF + shuding 折射核（中心清晰、边缘强折射）
  → DXGI swapchain → DirectComposition 透明置顶窗
WDA_EXCLUDEFROMCAPTURE 排除自身捕获，断开反馈循环
```

### 弹簧形变

二阶弹簧（刚度 K + 阻尼 C）每帧 Euler 积分，主线程鼠标事件设目标，渲染线程每帧推进：

- **按下**：横向窄 + 纵向长（`pressX/pressY`）
- **稳态**：稍窄稍高（`steadyX/steadyY`）
- **拖动**：按速度叠加变窄（`dragK`，下限 `dragMin`）
- **松手**：回到稳态 + 基于速度的过冲冲量（`releaseImpulse`）

所有参数在 `glass-tuning.json` 中可热调。

## 构建

需要 Go 工具链 + Windows：

```powershell
# 调试（带控制台看输出，不产生 exe）
go run .

# 编译 exe（本地试装 / 发行，唯一一条；dist 不存在先建 mkdir dist）
go build -trimpath -buildvcs=false -ldflags="-H windowsgui" -o dist/claude-traffic-light.exe .

# 测试
go test ./...
```

> **编译铁律**：调试用 `go run .`（不产 exe）；凡是产出 exe 就用上面唯一那条命令，四件防护一次带齐——`-trimpath`（清本机路径/用户名）、`-buildvcs=false`（清 git 信息）、`-ldflags="-H windowsgui"`（无黑窗）、`-o dist/`（隔离），并自动嵌入图标+版本信息。**严禁 `-s -w`**（触发 Wacatac 误报）、严禁加壳。完整流程见 [docs/编译构建发行.md](docs/编译构建发行.md)。

### exe 图标 + 版本信息（资源管理器属性）

窗口/托盘图标由 `main.go` 的 `//go:embed claude-traffic-light.ico` 在运行时加载；而 **exe 文件图标 + 资源管理器属性里的版本信息**（产品名/版本/署名）是另一套——靠 `rsrc_windows_amd64.syso` 在**链接时**嵌入。

- 该 syso 由 **goversioninfo** 用 `versioninfo.json`（版本信息源）+ ico **合成一个**生成；`versioninfo.json` 与 syso **均已入库**，克隆后直接 `go build` 即带图标+版本信息。
- **只在改图标/版本号/署名时**重新生成：
  ```powershell
  goversioninfo -icon=claude-traffic-light.ico -o=rsrc_windows_amd64.syso
  ```
  （工具装一次：`go install github.com/josephspurrier/goversioninfo/cmd/goversioninfo@latest`）
- ⚠️ **文件名必须带 `_windows_amd64` 后缀**，go build 才自动按平台挑它。**严禁**再生成第二个含资源段的 `.syso`（如 `rsrc.syso`）——两个都含资源段 → 链接报 `too many .rsrc sections` 构建失败。

## 技术栈

Go + D3D11 + DirectComposition + HLSL（无 WebView2、无 Electron、无 CGO）

## 限制 / 范围外

- **仅 Windows 11**，不支持 macOS / Linux
- **仅 Claude Code**，不支持其他 AI 工具
- 点击穿透：DComp 透明置顶窗无法穿透到下层，胶囊外透明区会挡住下层窗口点击（已知限制）
- 无声音提示、无多显示器自动定位

## 许可

MIT
