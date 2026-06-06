package watcher

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"claude-traffic-light/state"
)

// writeState 在 dir 写一个会话状态文件 agent-light-state-<sid>，返回其路径。
func writeState(t *testing.T, dir, sid, word string) string {
	t.Helper()
	p := filepath.Join(dir, statePrefix+sid)
	if err := os.WriteFile(p, []byte(word), 0644); err != nil {
		t.Fatal(err)
	}
	return p
}

// TestReadAggregates 验证多会话聚合：任一忙就忙、全 idle 才绿、无文件灰。
// 核心用例是 running+idle → 红：一个 agent 结束不再把全局拉绿。
func TestReadAggregates(t *testing.T) {
	cases := []struct {
		name  string
		files map[string]string // sid → 状态词
		want  state.State
	}{
		{"无文件=从未运行", nil, state.Grey},
		{"单 running", map[string]string{"a": "running"}, state.Red},
		{"单 thinking", map[string]string{"a": "thinking"}, state.Yellow},
		{"单 idle", map[string]string{"a": "idle"}, state.Green},
		{"running+idle 一个结束不拉绿", map[string]string{"a": "running", "b": "idle"}, state.Red},
		{"thinking+idle", map[string]string{"a": "thinking", "b": "idle"}, state.Yellow},
		{"全 idle 才绿", map[string]string{"a": "idle", "b": "idle", "c": "idle"}, state.Green},
		{"running+thinking 取最高", map[string]string{"a": "running", "b": "thinking"}, state.Red},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			for sid, word := range c.files {
				writeState(t, dir, sid, word)
			}
			w := &Watcher{stateDir: dir}
			if got := w.read(); got != c.want {
				t.Errorf("got %v, want %v", got, c.want)
			}
		})
	}
}

// TestReadStaleDemotes 验证陈旧 running/thinking 文件被降级为不忙：
// 一个陈旧 running + 一个新鲜 thinking → 黄（陈旧的不计入，否则会是红）。
func TestReadStaleDemotes(t *testing.T) {
	dir := t.TempDir()
	stale := writeState(t, dir, "old", "running")
	old := time.Now().Add(-freshWindow - time.Minute)
	if err := os.Chtimes(stale, old, old); err != nil {
		t.Fatal(err)
	}
	writeState(t, dir, "new", "thinking")

	w := &Watcher{stateDir: dir}
	if got := w.read(); got != state.Yellow {
		t.Errorf("got %v, want Yellow（陈旧 running 应降级）", got)
	}
}

// TestReadStaleAloneIsGreen 单独一个陈旧忙态文件 → 绿，对应「开机/空闲残留不卡黄」。
func TestReadStaleAloneIsGreen(t *testing.T) {
	dir := t.TempDir()
	stale := writeState(t, dir, "old", "thinking")
	old := time.Now().Add(-freshWindow - time.Minute)
	if err := os.Chtimes(stale, old, old); err != nil {
		t.Fatal(err)
	}
	w := &Watcher{stateDir: dir}
	if got := w.read(); got != state.Green {
		t.Errorf("got %v, want Green（陈旧残留应降级，不卡黄）", got)
	}
}
