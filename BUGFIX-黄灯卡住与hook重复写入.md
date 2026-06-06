# Bug 修复记录：黄灯卡住 & hook 重复写入

> 日期：2026-06-06
> 涉及文件：`watcher/watcher.go`、`hookinstall.go`
> 状态：✅ 已修复，编译 / 单元测试 / 发布 exe 构建全部通过

本次定位并修复两个相互独立的 bug。下面分别给出**表现 → 成因 → 解决方案**。

---

## Bug A：开机 / 空闲时一直卡黄灯

### 表现

1. **电脑刚启动**，没运行任何 Claude Code，挂件却一直闪黄灯。
2. **打开 VS Code 但没给 Claude 输入任何内容**，按预期应是绿灯（空闲），实际仍卡在黄灯。

### 成因（已确证，非猜测）

证据来自代码机制 + 实际状态文件的值，闭环如下：

- 状态词 `thinking`（黄）/ `running`（红）本质是 **Claude Code hook 触发时的"瞬时事件态"**，但旧代码把它当成"常亮状态"**无限保持**——`watcher.go` 的 `read()` 只要读到 `thinking` 就一直返回黄，不超时、不衰减。
- 而**唯一能把状态写回 `idle`（绿）的只有 `Stop` 这一个 hook**。
- 于是只要上次会话**没干净地走完 `Stop`**（直接关 VS Code、强杀进程、崩溃），状态文件 `~/.claude/agent-light-state` 里就**残留**着上次的 `thinking`/`running` 值，并跨重启保留在磁盘上。

两个现象由此而来：

| 现象 | 根因 |
|------|------|
| 开机一直闪黄 | 状态文件残留上次关机时的 `thinking` 值，开机后挂件**无条件信任**这个陈旧值 → 闪黄 |
| 打开 VS Code 没输入也闪黄 | VS Code 的 Claude Code 扩展一启动就有 `claude.exe` 常驻 → `procmon.go` 的"3 秒灭灯"检测到进程在 → **永不切灰**；但又没有新 hook 去写 `idle` → 灯卡在残留的 `thinking` |

**核心矛盾**：用户的心智模型是"Claude 在运行但空闲 = 绿灯"，但旧代码里绿灯**必须**收到一次 `Stop` 才会亮。进程在、又收不到 `Stop` → 灯永久卡黄/红，靠进程检测也灭不掉。

### 解决方案：状态文件 mtime 新鲜度判断

只改**读取端** `watcher.go`，写入端一行不动。引入新鲜度窗口 `freshWindow = 15s`：

- 进程不在 → 灰（不变）
- 文件是 `thinking`/`running` 且 **mtime 在 15s 内**被刷新（最近真有 hook 活动）→ 黄 / 红
- 文件是 `thinking`/`running` 但 **mtime 已超过 15s**（早就没活动了）→ 判定空闲 → **绿**
- 文件是 `idle` → 绿
- 文件不存在 / 无法识别 → 灰

效果：开机 / 空闲时文件 mtime 是很久以前 → 自动回绿；真在干活时 hook 密集刷新 mtime → 黄 / 红。

```go
// freshWindow 是瞬时事件态（running/thinking）的新鲜度窗口。
const freshWindow = 15 * time.Second

func (w *Watcher) read() state.State {
	data, err := os.ReadFile(w.statePath)
	if err != nil {
		return state.Grey
	}
	switch strings.TrimSpace(string(data)) {
	case "running", "thinking":
		fi, err := os.Stat(w.statePath)
		if err != nil || time.Since(fi.ModTime()) > freshWindow {
			return state.Green // 陈旧 = 早已不在干活 = 空闲
		}
		if strings.TrimSpace(string(data)) == "running" {
			return state.Red
		}
		return state.Yellow
	case "idle":
		return state.Green
	default:
		return state.Grey
	}
}
```

### 阈值 N = 15s 的权衡

| 取值 | 优点 | 代价 |
|------|------|------|
| 5s | 空闲后回绿快、开机灭黄快 | 长命令 / 大文件操作执行时容易闪一下绿再回黄 |
| **15s（采用）** | 折中：空闲约 15s 自动回绿 | 仅当单个工具连续执行超 15s 且中途无 hook 才会短暂闪绿，实际很少见 |
| 30s | 长工具几乎不误判 | 开机残留旧值要等 30s 才回绿，反应偏慢 |

**唯一代价**：单个工具执行超过 15s、期间没有新 hook 时，灯会短暂闪一下绿再回黄。

---

## Bug B：重复安装挂件会重复写入 hook（settings.json 出现两套）

### 表现

`~/.claude/settings.json` 里装了**两套** hook —— `claude-traffic-light-debug.exe` 和 `claude-traffic-light.exe`，4 个事件每个都触发两个进程。（注：这不是黄灯的主因，两者写同一文件、后写覆盖；但是该清的隐患，已手动清理。）

### 成因（真 bug）

旧代码靠一个**写死的常量**识别"settings.json 里哪条 hook 是我加的"：

```go
const hookExeName = "claude-traffic-light.exe"   // 硬编码！
// ...
if strings.EqualFold(filepath.Base(cmd), hookExeName) { ... }
```

但开发期用过 debug 名构建（`-o claude-traffic-light-debug.exe`）。那个 debug 挂件启动时：

1. 它的真实名字是 `claude-traffic-light-debug.exe`；
2. 识别逻辑却去找硬编码的 `claude-traffic-light.exe` → **认不出自己**；
3. 于是认为"我还没装" → **追加一条**。

结果 debug 版和正式版互不相认、各装各的 → 两套并存。

**根因**：识别基准用了**硬编码常量**，而不是"当前运行 exe 自己的名字"。basename 一旦不等于那个写死的值（debug 后缀、改名），幂等就失效，重复追加。

> 注：U 盘换盘符**不受此 bug 影响**——那是路径变、basename 不变，能正确走"更新路径"分支。问题只出在 basename 本身变了。

### 解决方案：识别基准改为当前 exe 真名

把比较基准从硬编码常量换成 `filepath.Base(exe)`（当前运行 exe 自己的 basename），并删掉孤儿常量 `hookExeName`：

```go
// 改前：拿写死的名字去比
if strings.EqualFold(filepath.Base(cmd), hookExeName) {
// 改后：拿当前 exe 自己的 basename 去比
if strings.EqualFold(filepath.Base(cmd), filepath.Base(exe)) {
```

**效果**：每个 exe 都认得自己那条 → 真正幂等，不再重复追加。

### 一个要知道的权衡

这样改之后，debug 版和正式版**仍会各装一条**（因为名字不同，本就是两个独立程序），但好处是**各自幂等、不会无限堆叠**。

**没有**改成"前缀匹配把两者当同一条"——那会导致 debug 与正式交替运行时互相覆盖对方的路径，更乱。开发期的 debug 条手动清一次即可，发布日常只有正式版一条。

---

## 验证

```
go build ./...      → BUILD_OK
go test ./...       → 全部 ok（config / state / ui / watcher）
go build -ldflags="-H windowsgui" -o claude-traffic-light.exe .  → EXE_BUILT
```

> settings.json 里的 debug 残留 hook 已手动清理，4 个事件现各只剩正式版 `claude-traffic-light.exe` 一条。
