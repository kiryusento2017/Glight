package watcher

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"claude-traffic-light/state"
)

// Watcher 轮询 hook 写的会话状态文件，聚合为四态，内容变化时回调。
// 数据源是 Claude Code hook 实时写入的状态词：每个会话（session_id）一个文件
// agent-light-state-<sid>，本 watcher 聚合 stateDir 下所有此前缀文件——
// 任一会话忙就显示忙，解决多 agent 并发时「一个会话结束误把全局拉绿」的问题。
// 每 3s 检查 Claude Code 进程是否还在——不在则切灰色。
type Watcher struct {
	stateDir      string
	onChange      func(state.State)
	stop          chan struct{}
	last          state.State
	tickN         int
	claudeRunning bool // 上次进程检测结果，初始 true（启动时可能已在运行）
}

// New 创建状态监测器。stateDir 是 hook 写、挂件读的状态文件所在目录（~/.claude/agent-light/）。
func New(stateDir string, onChange func(state.State)) *Watcher {
	return &Watcher{
		stateDir:      stateDir,
		onChange:      onChange,
		stop:          make(chan struct{}),
		last:          state.Grey,
		claudeRunning: true, // 保守假设进程在运行，避免启动时错误灭灯
	}
}

func (w *Watcher) Stop() { close(w.stop) }

// Watch 每 100ms 聚合状态文件，内容变化时回调。阻塞 — 在 goroutine 里调。
func (w *Watcher) Watch() {
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()
	w.poll()
	for {
		select {
		case <-w.stop:
			return
		case <-tick.C:
			w.poll()
		}
	}
}

func (w *Watcher) poll() {
	s := w.read()
	w.tickN++
	if w.tickN%30 == 0 {
		w.claudeRunning = isClaudeCodeRunning() // 每 3s 刷新一次进程检测
	}
	if w.tickN%300 == 0 {
		w.cleanupStale() // 每 30s 清理陈旧会话文件，防长期堆积
	}
	if !w.claudeRunning {
		s = state.Grey // 进程不在，每个 tick 都强制灰色
	}
	if s != w.last {
		w.last = s
		w.onChange(s)
	}
}

// statePrefix 是每会话状态文件名前缀：agent-light-state-<session_id>。
const statePrefix = "agent-light-state-"

// cleanupWindow 是会话文件的物理清理阈值：超此时长未更新就删除，回收崩溃/中断
// 未走 Stop 的残留文件，防长期堆积。纯磁盘清理，与状态判定解耦——状态忙/闲只看
// 文件内容（running/thinking/idle），残留兜底交给 poll 的进程检测（claude.exe 没
// 了→灰），不再用时间窗口猜「静默=空闲」（长思考也静默，固定阈值两全不了）。
const cleanupWindow = 10 * time.Minute

// read 聚合 stateDir 下所有会话状态文件为四态：任一会话 running→红、thinking→黄、
// 全 idle→绿、无文件→灰。忙态只信 hook 写的内容，不做 mtime 超时降级；崩溃/强杀/
// 开机残留靠 poll 的进程检测兜底（claude.exe 没了→灰）。
func (w *Watcher) read() state.State {
	files, _ := filepath.Glob(filepath.Join(w.stateDir, statePrefix+"*"))
	if len(files) == 0 {
		return state.Grey // 无任何会话文件 = 从未运行
	}
	states := make([]state.State, 0, len(files))
	for _, f := range files {
		states = append(states, w.readOne(f))
	}
	return state.Highest(states)
}

// readOne 把单个会话文件映射为四态，只看 hook 写入的状态词：running→红、
// thinking→黄、idle/读不到/无法识别→绿（不计入忙），由聚合层决定整体。
func (w *Watcher) readOne(path string) state.State {
	data, err := os.ReadFile(path)
	if err != nil {
		return state.Green // 读不到（刚被删等）→ 不计入忙
	}
	switch strings.TrimSpace(string(data)) {
	case "running":
		return state.Red
	case "thinking":
		return state.Yellow
	case "idle":
		return state.Green
	default:
		return state.Green // 无法识别 → 不计入忙
	}
}

// cleanupStale 删除超过 cleanupWindow 未更新的会话文件（崩溃/中断未走 Stop 的
// 残留、或早已结束的会话）。活跃会话文件 mtime 新鲜，不受影响。
func (w *Watcher) cleanupStale() {
	files, _ := filepath.Glob(filepath.Join(w.stateDir, statePrefix+"*"))
	for _, f := range files {
		fi, err := os.Stat(f)
		if err == nil && time.Since(fi.ModTime()) > cleanupWindow {
			os.Remove(f)
		}
	}
}
