<div align="center">

# 🚦 Glight

**Windows 上永远置顶的液态玻璃红绿灯，一眼看出 Claude Code 跑完没 —— vibe coding 余光神器。**

![平台](https://img.shields.io/badge/平台-Windows-0078D6?logo=windows&logoColor=white)
![语言](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go&logoColor=white)
![技术](https://img.shields.io/badge/D3D11_+_DirectComposition_+_HLSL-5C2D91)
![许可](https://img.shields.io/badge/license-MIT-green)
[![下载](https://img.shields.io/badge/⬇_下载-Releases-success)](../../releases)

**简体中文** · [English](docs/readme/README.en.md) · [日本語](docs/readme/README.ja.md) · [한국어](docs/readme/README.ko.md) · [繁體中文](docs/readme/README.zh-TW.md)

<img src="assets/screenshot-wide.jpg" width="600" alt="Glight（Claude Code Light）：浮在 VS Code 上的液态玻璃红绿灯，绿灯亮起" />

<sub>一块真实折射桌面的液态玻璃胶囊，中间三盏灯实时跟着 Claude Code 的状态走。</sub>

</div>

---

## 这是什么

**Glight**（又名 Claude Code Light / Claude Code 红绿灯状态灯）是一个 Windows 桌面挂件：一块**永远悬浮在最上层**的液态玻璃胶囊，透过它能看到真实桌面被实时折射扭曲。胶囊中间嵌着三盏红绿灯，**实时反映 [Claude Code](https://claude.ai/code) 的工作状态**——

> 红灯闪 = 正在执行工具；黄灯闪 = 正在思考；绿灯常亮 = 空闲等你；全灭 = 没在跑。

写代码时不用一直盯着终端，**余光扫一眼挂件就知道 AI 是在忙、在想、还是停了**。单个原生 `.exe`，不依赖任何运行时，扔进 U 盘即插即用。

## ✨ 设计思路

灵感来自 **Apple 的「液态玻璃 (Liquid Glass)」**。和那些只会糊一层模糊背景的方案不同，它是**真·折射**：

- **实时折射真实桌面** —— 自取屏幕像素 + 自写 HLSL 折射 shader，玻璃后面的窗口、代码、壁纸会像隔着真玻璃一样被向心扭曲（中心清晰、边缘强折射），而不是一张死图或一块模糊毛玻璃。
- **可拖拽** —— 按住胶囊随手拖到屏幕任意角落，松手记忆位置。
- **按压会 QQ 弹弹形变** —— 这是灵魂所在 🍮。按下去玻璃会像果冻一样**横向压扁、纵向拉长**；拖得越快被「甩」得越窄；松手后靠**二阶弹簧物理**Q 弹回弹、轻微过冲再归位。整套形变参数（刚度/阻尼/过冲）都能在 `glass-tuning.json` 里热调，存盘即生效。
- **无极缩放** —— 右键「调整大小」100%~2000% 任意放大缩小，大小位置自动记忆。

## 🚦 状态对照表

| 灯 | 含义 | 触发的 Claude Code 事件 |
|---|---|---|
| 🔴 **红灯闪烁** | 正在执行工具 | `PreToolUse` |
| 🟡 **黄灯闪烁** | 正在思考 | `UserPromptSubmit` / `PostToolUse` |
| 🟢 **绿灯常亮** | 空闲等待 | `Stop` |
| ⚫ **三灯全灭** | 未运行 | `claude.exe` 进程不在 |

多状态优先级：**红 > 黄 > 绿 > 灰**。多个 Claude 会话并发时，任一会话在忙就显示忙，不会因为一个结束就误判空闲。

## 💻 系统要求

| 要求 | 说明 |
|---|---|
| **操作系统** | **仅 Windows 10 (2004+) / 11**。**不支持 macOS / Linux**——核心依赖 Windows 独有的 DirectComposition + Desktop Duplication，无跨平台计划。 |
| **Claude Code** | 需本机已安装 [Claude Code](https://claude.ai/code) CLI。这是**专为 Claude Code 设计**的状态指示器，不支持 Cursor / Copilot / 其他 AI 工具。没装也能运行，但三灯恒灰。 |
| **运行时** | **无**。单个原生 `.exe`，不依赖 Node / .NET / Electron / 任何框架。 |

## 📦 安装（3 步）

**1️⃣ 下载**

从 [Releases](../../releases) 下载 `claude-traffic-light.exe`，放到任意目录。

**2️⃣ 首次以管理员身份运行**

> 右键 exe →「以管理员身份运行」。**仅首次需要**，之后正常双击即可。

挂件首次启动会在 exe 同目录释放两个配置文件：`config.json`（位置/缩放）和 `glass-tuning.json`（视觉/形变参数）。

**3️⃣ 给释放出的两个文件足够的读写权限**

如果挂件装在 **C 盘**（尤其 `C:\Program Files\` 这类受保护目录），系统可能不允许它写配置文件。这时需手动给这两个文件**读取 + 写入**权限：右键文件 →「属性」→「安全」→「编辑」，勾上「写入」。

> 💡 **一般只有装在 C 盘才需要这一步。** 装在 D 盘等其他盘时，挂件通常能直接释放配置文件、不需要管理员权限，可跳过此步。

![给 config.json / glass-tuning.json 读写权限](assets/admin-permission.png)

装好后，打开 Claude Code 开始 vibe coding，灯就会跟着动了。

## 🔒 它会动你电脑里的哪些文件？

跑一个陌生的 `.exe`，你最该关心的就是「它到底往我电脑里写了什么」。这里把它碰到的**每一个文件、靠什么运作**全摊开讲清楚——它**不修改任何系统文件，也完全不联网**。

### 靠什么运作：Claude Code 的「钩子 (Hooks)」

Claude Code 自带一套「钩子」机制：在它**开始执行工具、思考、停下**等关键时刻，会自动运行你预先登记好的命令。本挂件就借这个机制工作——首次启动时，往 Claude Code 自己的配置文件 `~/.claude/settings.json` 里**加 4 条钩子**，让 Claude Code 状态一变就顺手「通知」挂件一声，挂件据此切换灯色。这 4 条钩子调用的命令**就是挂件自己**（`claude-traffic-light.exe hook <状态>`），不依赖任何第三方程序（不像有的方案还要装 Node）。

合并钩子时我们很克制：

- **改之前先把整个 `settings.json` 备份**成 `settings.json.bak`；
- **只加自己这 4 条，绝不删改你已有的任何配置**；
- 已经加过就不重复加（幂等）；
- **如果你没装 Claude Code（`~/.claude/` 目录不存在），它什么都不写、连目录都不会建。**

### 它会创建 / 写入的全部文件

| 文件 | 在哪 | 干嘛用的 | 何时写 |
|---|---|---|---|
| `config.json` | **exe 同目录** | 记住挂件的位置 / 大小 / 各项开关 | 拖动、缩放、点菜单时 |
| `glass-tuning.json` | **exe 同目录** | 玻璃外观与形变参数，供你手动调 | 首次启动生成一次 |
| `settings.json`（**加 4 条钩子**） | `~/.claude/` | 让 Claude Code 状态变化时通知挂件 | 首次启动；改前先备份 |
| `settings.json.bak` | `~/.claude/` | 上面那份**改动前的完整备份** | 仅当钩子有变动时 |
| `agent-light-state-<会话id>` | `~/.claude/agent-light/` | 每个 Claude 会话的当前状态（**就一个词**：`running`/`thinking`/`idle`） | 每次状态变化覆盖写 |
| 注册表 `Run` 项 | `HKCU\...\Run`（用户级，**不弹 UAC**） | 实现「开机自动启动」 | **仅当你勾选**开机自启；取消即删 |

> 前两个（`config.json` / `glass-tuning.json`）就在 exe 旁边、一眼可见、随时可删；后面几个都在你自己的 `~/.claude/` 用户目录下。**全程不碰 C 盘系统目录、不碰注册表敏感区。**

### 多个 Claude 一起跑也不会乱

每开一个 Claude 会话，挂件就给它**单独建一个小状态文件**（内容只有一个词）。同时跑多个时，挂件每 0.1 秒把这些文件汇总一次：**只要有任一个会话在忙，灯就显示忙；全都空闲了才变绿。** 这样一个 agent 先结束、另一个还在跑时，灯**不会被误拉成绿**。这些小文件超过 10 分钟没更新会被自动清理，不会越堆越多。

### 想彻底清除？

删掉 exe 同目录的两个 `.json` + `~/.claude/agent-light/` 文件夹，再把 `~/.claude/settings.json` 里那 4 条钩子删掉（或直接用 `.bak` 还原）即可；关掉「开机自动启动」会自动删除那条注册表记录。**卸载干干净净、不留尾巴。**

## 🎮 使用

- **拖动** —— 按住可见胶囊拖拽移位（点胶囊外的透明区无反应，不会误拖）。
- **右键菜单 / 托盘菜单**：
  | 菜单项 | 作用 |
  |---|---|
  | 显示 / 隐藏 | 切换可见，隐藏后可从托盘图标恢复 |
  | 固定位置 | 锁定后不可拖动，防误触移位 |
  | 开机自动启动 | 写入注册表 Run 项（用户空间，不弹 UAC），取消即删 |
  | 调整大小… | 弹滑块窗，100%~2000% 无极缩放 |
  | 重置大小和位置 | 回 100%、屏幕顶部居中 |
  | 重启 | 卡帧/异常时一键重启（新实例接管，绝不"关了没开"） |
  | 退出 | — |
- **系统托盘** —— 右键托盘图标 = 同款菜单；双击托盘图标 = 显示/隐藏。
- **调参** —— 编辑 exe 同目录的 `glass-tuning.json`，**保存即热重载（~0.5s 生效）**，无需重启。

---

<details>
<summary><b>🔧 调参：全部参数与默认值</b>（点击展开）</summary>

编辑 exe 同目录的 `glass-tuning.json`，**保存即热重载，无需重启或重编译**。该文件每台机器各一份、不进版本库；删掉它下次启动会用下表默认值重新生成。默认值的权威来源是 `config/tuning.go` 的 `DefaultTuning()`。

**视觉参数**

| 参数 | 作用 | 默认 | 建议范围 |
|---|---|---|---|
| `cornerR` | 圆角半径(px)，越大越圆 | 48 | 0~115 |
| `cornerN` | 角部曲率指数，2=标准圆，越大越方（苹果味 G2） | 2.1 | 2.0~4.0 |
| `refractBand` | 折射带深度(px)，仅距边缘这么深处折射 | 3 | 1~30 |
| `edgeSqueeze` | 边缘收缩，0=折射最强，1=不折射 | 0.25 | 0~1 |
| `contrast` | 对比度 | 1.2 | 0.5~2.0 |
| `brightness` | 亮度 | 0.9 | 0.5~2.0 |
| `saturate` | 饱和度 | 1.5 | 0~3 |
| `lampR` | 灯半径(px) | 19 | 6~30 |
| `lampGap` | 灯间距(px)，红↔黄↔绿中心距 | 64 | 38~90 |
| `glow` | 点亮外发光强度 | 0 | 0~1 |

**物理形变参数**

| 参数 | 作用 | 默认 | 建议范围 |
|---|---|---|---|
| `springK` | 弹簧刚度，越大回弹越快越硬 | 120 | 30~300 |
| `springC` | 阻尼，越小过冲/弹动越明显 | 8 | 1~20 |
| `steadyX` | 稳态水平缩放，<1 静止偏窄 | 0.91 | 0.8~1.04 |
| `steadyY` | 稳态垂直缩放，>1 静止偏长 | 1.11 | 0.9~1.30 |
| `pressX` | 按下水平缩放，<1 变窄 | 0.82 | 0.5~1.04 |
| `pressY` | 按下垂直缩放，>1 变高 | 1.22 | 0.8~1.30 |
| `dragK` | 拖动形变力度，越大形变越猛 | 0.02 | 0.001~0.05 |
| `dragMin` | 拖动形变下限，0.5=最多缩到 50% | 0.5 | 0.3~1.0 |
| `releaseImpulse` | 松手过冲倍率，>1 强化回弹 | 1.5 | 1.0~3.0 |

> ⚠️ **形变有硬上限**：画布 240×144、玻璃 230×96，所以任意时刻水平缩放 ≤ 1.04、垂直缩放 ≤ 1.50，超了胶囊顶/底会被画布切平。竖向拖动已内置 `maxDragScaleY=1.4` 钳制防撞墙。要更夸张的拉伸得改 `ui/glass.hlsl` 的 `CANVAS`/`PILL`（需重编译）。

</details>

<details>
<summary><b>🏗️ 架构与技术原理</b>（点击展开）</summary>

### 渲染管线（方案核心）

```
Desktop Duplication 抓整屏桌面纹理（GPU 常驻）
  → HLSL 超椭圆 SDF + 折射核（中心清晰、边缘强折射）
  → DXGI 合成 swapchain → DirectComposition 透明置顶窗
窗口设 WDA_EXCLUDEFROMCAPTURE 把自己从抓取中排除，断开「自己折射自己」反馈
```

为什么不用 CSS `backdrop-filter` / WebView2？因为它只能采样 WebView 文档内部，**采不到操作系统桌面** → 旧版只能显示成黑框。Windows 也没有「对窗口背景做折射位移」的系统 API（DWM Acrylic 只有模糊无折射）。唯一出路就是自取桌面像素 + 自写折射 shader。

### 模块分层

```
main.go             入口：单实例互斥 → 加载配置 → 开机自启同步 → 安装 hook → 建窗 + 监测
hookinstall.go      把状态 hook 幂等合并进 ~/.claude/settings.json（先备份、只增不删）
config/             config.json（位置/缩放/自启）+ glass-tuning.json（视觉热重载）
  autostart.go      注册表 HKCU Run 读写删 + 路径自校正
state/              四态枚举（灰/绿/黄/红）及优先级聚合
watcher/            每 100ms 聚合每会话状态文件（任一忙=忙）+ 每 3s 进程检测兜底灭灯
ui/
  window.go           DComp 透明置顶窗、消息循环、自接管鼠标拖动、弹簧形变状态机
  render.go           D3D11 渲染管线：device/swapchain/shader 编译 + 每帧绘制
  glass.hlsl          像素 shader：超椭圆 SDF 限定形状 + 折射核 + 三灯叠加
  capture.go          Desktop Duplication 抓桌面纹理（随缩放 Resize + 失效后限速重建）
  com.go              D3D11/DXGI/DComp COM 绑定
  win32.go            Win32 API 绑定与常量
  physics.go          二阶弹簧物理（每帧 Euler 积分驱动形变）
  sizedialog.go       调整大小滑块窗（comctl32 trackbar）
```

### 状态探测：Claude Code Hooks（实时驱动）

不轮询 transcript，而是靠 Claude Code 的 4 个生命周期 hook **实时推送**状态。挂件首次启动时把这 4 条 hook 幂等合并进 `~/.claude/settings.json`，事件与状态对应如下：

| Claude Code 事件 | 写入状态词 | 灯 |
|---|---|---|
| `UserPromptSubmit` / `PostToolUse` | `thinking` | 🟡 思考中 |
| `PreToolUse` | `running` | 🔴 执行中 |
| `Stop` | `idle` | 🟢 空闲 |

每条 hook 以 exec form 直接调用挂件自己 `claude-traffic-light.exe hook <状态词>`（直接 spawn 不经 shell，规避 Windows 下 shell 不确定性），它只干一件事：把状态词写进一个**每会话独立的状态文件**。

#### 每会话一个状态文件 `agent-light-state-<session_id>`

Claude Code 触发 hook 时会通过 **stdin 传一段 JSON**，里面带着当前会话的 `session_id`。挂件从中读出它，写到：

```
~/.claude/agent-light/agent-light-state-<session_id>
```

文件内容就是一个状态词（`running` / `thinking` / `idle`）。几个要点：

- **每个会话一个文件、互不覆盖**——这是支持多 agent 的基础（见下）。
- session_id 会被过滤成安全文件名（只保留字母/数字/连字符）；读不到时回退为 `default`。
- 读 stdin 带 **500ms 硬超时**，保证 hook 永远毫秒级返回、**绝不拖慢 Claude Code**。
- 统一塞进 `agent-light/` 子目录，不在 `~/.claude/` 根目录摊一堆文件。

#### 多 agent 并发：任一会话忙 → 全局忙

你完全可能**同时跑多个 Claude**（多个终端窗口，或主会话派生出子 agent），它们各写各的状态文件。挂件的 watcher 每 100ms 把 `agent-light/` 目录下所有 `agent-light-state-*` 文件读一遍，按优先级 **红 > 黄 > 绿 > 灰** 聚合成一个全局状态：

```
会话 A: running  🔴 ┐
会话 B: idle     🟢 ├─ 取最高优先级 ─→ 🔴 红（A 还在干活，就显示忙）
会话 C: thinking 🟡 ┘
```

**只要有任一会话在忙，挂件就显示忙。** 这样当一个 agent 先结束（写了 `idle`）、而另一个还在跑时，灯**不会被误拉成绿**；只有全部会话都 `idle` 才显示绿，一个文件都没有才显示灰。

#### 其余兜底机制

- **忙态只信 hook 内容**，不做时间窗口超时降级（长思考无工具调用→无 hook，按超时会被误判空闲）。
- **进程检测兜底灭灯**：每 3s 用 `CreateToolhelp32Snapshot` 检测 `claude.exe`，进程不在则强制灰——崩溃 / 强杀 / 开机残留统一靠这条回灰。
- **定时清理**：每 30s 删除超 10 分钟没更新的残留会话文件（纯磁盘回收，与状态判定解耦）。
- **hook handler 就是挂件自己**，零外部依赖（不像别的方案还要装 node）。

</details>

<details>
<summary><b>🛠️ 从源码构建</b>（点击展开）</summary>

需要 Go 工具链 + Windows：

```powershell
# 调试（带控制台看输出，不产生 exe）
go run .

# 编译 exe（唯一一条命令，四件防护一次带齐）
go build -trimpath -buildvcs=false -ldflags="-H windowsgui" -o dist/claude-traffic-light.exe .

# 测试
go test ./...
```

> **编译铁律**：调试用 `go run .`；产 exe 只用上面那条——`-trimpath`（清本机路径/用户名）、`-buildvcs=false`（清 git 信息）、`-ldflags="-H windowsgui"`（无黑窗）、`-o dist/`（隔离），并自动嵌入 `rsrc_windows_amd64.syso`（图标+版本信息）。**严禁 `-s -w`**（触发杀软误报）、**严禁加壳**。完整流程见 [docs/编译构建发行.md](docs/编译构建发行.md)。

**录屏演示**：正常 exe 对 OBS 等录屏软件隐形（`WDA_EXCLUDEFROMCAPTURE` 设计使然）。带 `--demo` 启动可解除排除让 OBS 录到，但玻璃会折射到自己。想要最干净的演示画面，用手机/相机拍屏幕即可。

</details>

## 🚧 限制 / 范围外

- **仅 Windows**，不支持 macOS / Linux
- **仅 Claude Code**，不支持其他 AI 工具
- **点击穿透**：DComp 透明置顶窗无法穿透到下层，胶囊外透明区会挡住下层窗口点击（已知限制）
- 无声音提示、无多显示器自动定位

## 📄 许可

[MIT](LICENSE)
