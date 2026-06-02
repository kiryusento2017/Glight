package watcher

import (
	"bufio"
	"os"
	"path/filepath"
	"sync"
	"time"
	"claude-traffic-light/state"
)

type Watcher struct {
	root     string
	timeout  time.Duration
	onChange func(state.State)
	stop     chan struct{}
	mu       sync.Mutex
	sessions map[string]sessionInfo
	last     state.State
}

type sessionInfo struct {
	state   state.State
	modTime time.Time
}

func New(root string, inactivityTimeout time.Duration, onChange func(state.State)) (*Watcher, error) {
	os.MkdirAll(root, 0755)
	return &Watcher{
		root:     root,
		timeout:  inactivityTimeout,
		onChange: onChange,
		stop:     make(chan struct{}),
		sessions: make(map[string]sessionInfo),
		last:     state.Grey,
	}, nil
}

func (w *Watcher) Stop() { close(w.stop) }

// Watch polls every 250ms. Blocks — call in a goroutine.
func (w *Watcher) Watch() {
	tick := time.NewTicker(250 * time.Millisecond)
	prune := time.NewTicker(10 * time.Second)
	defer tick.Stop()
	defer prune.Stop()

	w.scan()

	for {
		select {
		case <-w.stop:
			return
		case <-tick.C:
			w.scan()
		case <-prune.C:
			w.pruneExpired()
			w.notify()
		}
	}
}

func (w *Watcher) scan() {
	pattern := filepath.Join(w.root, "*/transcript.jsonl")
	paths, _ := filepath.Glob(pattern)

	w.mu.Lock()
	defer w.mu.Unlock()

	hadSessions := len(w.sessions) > 0

	// Remove sessions whose files no longer exist
	for path := range w.sessions {
		found := false
		for _, p := range paths {
			if p == path {
				found = true
				break
			}
		}
		if !found {
			delete(w.sessions, path)
		}
	}

	changed := false
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		mod := info.ModTime()

		prev, exists := w.sessions[path]
		if exists && prev.modTime == mod {
			continue
		}

		s := parseFile(path)
		w.sessions[path] = sessionInfo{state: s, modTime: mod}
		changed = true
	}

	// Notify if state changed OR if we just lost all sessions
	if changed || (hadSessions && len(w.sessions) == 0) {
		w.notifyLocked()
	}
}

func (w *Watcher) pruneExpired() {
	w.mu.Lock()
	defer w.mu.Unlock()
	cutoff := time.Now().Add(-w.timeout)
	for path, info := range w.sessions {
		if info.modTime.Before(cutoff) {
			delete(w.sessions, path)
		}
	}
}

func (w *Watcher) notify() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.notifyLocked()
}

func (w *Watcher) notifyLocked() {
	states := make([]state.State, 0, len(w.sessions))
	for _, info := range w.sessions {
		states = append(states, info.state)
	}
	s := state.Highest(states)
	if s != w.last {
		w.last = s
		go w.onChange(s) // don't block under lock
	}
}

func parseFile(path string) state.State {
	f, err := os.Open(path)
	if err != nil {
		return state.Green
	}
	defer f.Close()
	var lines []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	return ParseLastState(lines)
}

// ClaudeProjectsPath returns the path to ~/.claude/projects/
func ClaudeProjectsPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude", "projects")
}
