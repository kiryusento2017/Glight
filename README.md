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

## 关于 Claude Code

这个挂件靠 **Claude Code 的 Hooks 机制**实时感知状态，而非轮询 transcript：

- **首次运行**检测 `~/.claude/settings.json`：如果存在（本机已装 Claude Code）则幂等合并 4 条 hook 规则（先备份、只增不删、已存在不重复加）；**如果不存在则静默跳过，不创建任何文件或目录**——不在没装 Claude Code 的机器上写任何东西。
- 每次 Claude Code 触发生命周期事件，hook 直接调挂件自己写状态文件，挂件 100ms 内反映到灯上。
- **自动灭灯**：每 3s 检测 `claude.exe` 进程，关掉 Claude Code 后最多 3s 三灯熄灭。
- hook handler 就是挂件 `.exe` 自身（`claude-traffic-light.exe hook <state>`），**零外部依赖**。

> 如果本机没装 Claude Code，挂件能正常运行，但因检测不到状态会一直保持灰色（三灯全灭）。

## 安装与使用

1. 从 [Releases](../../releases) 下载 `claude-traffic-light.exe`
2. 双击运行（首次启动自动安装 Claude Code hook，并生成默认配置文件）
3. 打开 Claude Code，开始 vibe coding
4. **拖动**：按住可见胶囊拖拽移位（点胶囊外的透明区无反应）
5. **右键菜单**：
   - **调整大小…** — 弹出滑块窗，100%~2000% 无极缩放，松手存盘、手动关闭
   - **开机自动启动** — 写入 `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`（不弹 UAC，取消勾选即删记录）
   - **隐藏 / 固定位置 / 重置大小和位置** — 自明
   - **退出**

**调参**：编辑 exe 同目录的 `glass-tuning.json`（首次运行自动生成），保存即实时生效，无需重启。

## 写入的文件

首次运行及后续使用中，挂件会在以下位置读/写文件。**不修改任何系统文件。**

| 路径 | 内容 | 说明 |
|---|---|---|
| `~/.claude/settings.json` | 4 条 hook 规则 | 首次启动幂等写入（备份 → 合并 → 写回） |
| `~/.claude/settings.json.bak` | 修改前的原文件 | 只在首次 hook 安装时创建一次 |
| `~/.claude/agent-light-state` | 状态词（`idle`/`thinking`/`running`） | 每次 Claude Code hook 触发时覆盖写入 |
| `./config.json` | 位置 + 锁定/可见/缩放/开机自启 | 退出 / 调整大小 / 切换自启时保存，exe 同目录 |
| `./glass-tuning.json` | 全部视觉与形变参数 | 首次运行自动生成，手工编辑热重载 |

> 若开启「开机自动启动」，会在 `HKCU\Software\Microsoft\Windows\CurrentVersion\Run` 写一条注册表记录（用户空间，不弹 UAC）。取消勾选后自动删除记录。

## 架构

```
main.go             入口：单实例互斥 → 加载配置 → 开机自启同步 → 安装 hook → 创建窗口 → 启动监测
hookinstall.go      把状态 hook 幂等合并进 ~/.claude/settings.json
config/             配置读写（config.json 位置/缩放/开机自启 + glass-tuning.json 视觉热重载）
  autostart.go      注册表 HKCU Run 读写删 + 路径自校正（开机自动启动）
state/              四态枚举（灰/绿/黄/红）及优先级
watcher/            每 100ms 读 hook 状态文件 + 每 3s 检测 claude.exe 进程
ui/
  window.go           DComp 透明置顶窗、消息循环、自接管鼠标拖动、弹簧形变状态机、图标加载
  render.go           D3D11 渲染管线：device/swapchain/shader 编译 + 每帧绘制（动态 viewport）
  glass.hlsl          像素 shader：超椭圆 SDF 限定形状、shuding 折射核、三灯叠加
  capture.go          Desktop Duplication 抓取桌面纹理（GPU 常驻，支持随缩放 Resize）
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

Hook 以 exec form 安装（`command`=exe 路径 + `args`，直接 spawn 不经 shell），每个 hook 写状态文件 `~/.claude/agent-light-state`，watcher 每 100ms 读取。

自动灭灯：每 3s 用 `CreateToolhelp32Snapshot` 检测 `claude.exe` 进程，不在则切灰色。

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
# 普通构建（带控制台，调试用）
go build -o claude-traffic-light.exe .

# 发布构建（无控制台窗口）
go build -ldflags="-H windowsgui" -o claude-traffic-light.exe .

# 测试
go test ./...
```

## 技术栈

Go + D3D11 + DirectComposition + HLSL（无 WebView2、无 Electron、无 CGO）

## 限制 / 范围外

- **仅 Windows 11**，不支持 macOS / Linux
- **仅 Claude Code**，不支持其他 AI 工具
- 点击穿透：DComp 透明置顶窗无法穿透到下层，胶囊外透明区会挡住下层窗口点击（已知限制）
- 无声音提示、无多显示器自动定位

## 许可

MIT
