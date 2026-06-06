package watcher

import (
	"os"
	"path/filepath"
	"testing"

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
