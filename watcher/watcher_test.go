package watcher

import (
	"os"
	"path/filepath"
	"testing"
	"time"
	"claude-traffic-light/state"
)

func TestWatcherDetectsChange(t *testing.T) {
	root := t.TempDir()
	proj := filepath.Join(root, "abc123")
	os.MkdirAll(proj, 0755)

	got := make(chan state.State, 5)
	w, _ := New(root, 5*time.Second, func(s state.State) { got <- s })
	go w.Watch()
	defer w.Stop()

	time.Sleep(100 * time.Millisecond) // let watcher start

	line := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"1","name":"Bash","input":{}}]}}` + "\n"
	os.WriteFile(filepath.Join(proj, "transcript.jsonl"), []byte(line), 0644)

	select {
	case s := <-got:
		if s != state.Red {
			t.Errorf("got %v, want Red", s)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for state change")
	}
}
