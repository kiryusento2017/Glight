package watcher

import (
	"os"
	"strings"
	"time"

	"claude-traffic-light/state"
)

// Watcher 轮询 hook 写的状态文件，内容变化时回调。
// 数据源是 Claude Code hook（PreToolUse/PostToolUse/UserPromptSubmit/Stop）
// 实时写入的状态词，比解析 transcript 实时、准确。
// 每 3s 检查 Claude Code 进程是否还在——不在则切灰色。
type Watcher struct {
	statePath     string
	onChange      func(state.State)
	stop          chan struct{}
	last          state.State
	tickN         int
	claudeRunning bool // 上次进程检测结果，初始 true（启动时可能已在运行）
}

// New 创建状态文件监测器。statePath 是 hook 写、挂件读的状态文件路径。
func New(statePath string, onChange func(state.State)) *Watcher {
	return &Watcher{
		statePath:     statePath,
		onChange:      onChange,
		stop:          make(chan struct{}),
		last:          state.Grey,
		claudeRunning: true, // 保守假设进程在运行，避免启动时错误灭灯
	}
}

func (w *Watcher) Stop() { close(w.stop) }

// Watch 每 100ms 读状态文件，内容变化时回调。阻塞 — 在 goroutine 里调。
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
	if !w.claudeRunning {
		s = state.Grey // 进程不在，每个 tick 都强制灰色
	}
	if s != w.last {
		w.last = s
		w.onChange(s)
	}
}

// read 读状态文件并映射为四态。文件不存在=还没任何活动=灰；
// 否则保持最后写入的状态（常亮，不超时变灰）。
func (w *Watcher) read() state.State {
	data, err := os.ReadFile(w.statePath)
	if err != nil {
		return state.Grey
	}
	switch strings.TrimSpace(string(data)) {
	case "running":
		return state.Red
	case "thinking":
		return state.Yellow
	case "idle":
		return state.Green
	default:
		return state.Grey
	}
}
