# Claude Traffic Light Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a portable single-exe Windows desktop widget that shows Claude Code's status as a transparent floating traffic light, driven by reading Claude Code's transcript files — zero configuration required.

**Architecture:** Go handles state detection (transcript watcher), config, and Win32 window behavior; WebView2 (system-installed Edge) renders the liquid glass UI from embedded `ui/glass.html`. State changes are pushed via `webview.Eval("setState('running')")`. Drag events flow back from JS to Go via WebView2 messaging.

**Tech Stack:** Go 1.21+, `github.com/jchv/go-webview2` (WebView2 embedding), `golang.org/x/sys/windows` (Win32 for tray/topmost/passthrough), 250ms polling watcher, embedded `ui/glass.html`

---

## File Map

```
claude-traffic-light/        ← workspace root
├── main.go                  ← entry point: wires watcher → window
├── go.mod / go.sum
├── app.manifest             ← DPI-aware manifest (embedded via rsrc.syso)
├── rsrc.syso                ← compiled manifest (auto-embedded by go build)
├── config/
│   ├── config.go            ← Config struct, Load, Save, Default
│   └── config_test.go
├── state/
│   ├── state.go             ← State type (Grey/Green/Yellow/Red), Highest()
│   └── state_test.go
├── watcher/
│   ├── parser.go            ← ParseLastState: JSONL lines → State
│   ├── parser_test.go
│   └── watcher.go           ← polls ~/.claude/projects/, calls onChange
├── ui/
│   ├── glass.html           ← validated liquid glass UI (embedded in binary)
│   ├── window.go            ← WebView2 window: create, topmost, tray, passthrough
│   └── win32.go             ← Win32 constants for tray + window style only
└── docs/
    └── superpowers/
        └── specs/2026-06-03-claude-traffic-light-design.md
```

---

## Task 1: Project Setup

**Files:**
- Create: `go.mod`

- [ ] **Step 1: Initialize module**

```
cd "D:\vs code projects\claude code light"
go mod init claude-traffic-light
go get golang.org/x/sys/windows
go get github.com/jchv/go-webview2
```

Expected: `go.mod` contains `module claude-traffic-light` and both `require` entries

- [ ] **Step 2: Create directories**

```
mkdir config state watcher ui
```

- [ ] **Step 3: Commit**

```
git init
git add go.mod go.sum
git commit -m "feat: initialize Go module"
```

---

## Task 2: State Machine

**Files:**
- Create: `state/state.go`
- Create: `state/state_test.go`

- [ ] **Step 1: Write failing test**

`state/state_test.go`:
```go
package state

import "testing"

func TestHighest(t *testing.T) {
    cases := []struct {
        in   []State
        want State
    }{
        {nil, Grey},
        {[]State{Grey}, Grey},
        {[]State{Green, Grey}, Green},
        {[]State{Yellow, Green}, Yellow},
        {[]State{Red, Yellow, Green}, Red},
    }
    for _, c := range cases {
        if got := Highest(c.in); got != c.want {
            t.Errorf("Highest(%v) = %v, want %v", c.in, got, c.want)
        }
    }
}

func TestString(t *testing.T) {
    if Grey.String() != "grey"     { t.Fail() }
    if Green.String() != "green"   { t.Fail() }
    if Yellow.String() != "yellow" { t.Fail() }
    if Red.String() != "red"       { t.Fail() }
}
```

- [ ] **Step 2: Run — confirm FAIL**

```
go test ./state/...
```
Expected: `undefined: State`

- [ ] **Step 3: Implement**

`state/state.go`:
```go
package state

type State int

const (
    Grey   State = iota
    Green
    Yellow
    Red
)

func (s State) String() string {
    return [...]string{"grey", "green", "yellow", "red"}[s]
}

// Highest returns the highest-priority state. Priority: Red > Yellow > Green > Grey.
func Highest(states []State) State {
    best := Grey
    for _, s := range states {
        if s > best {
            best = s
        }
    }
    return best
}
```

- [ ] **Step 4: Run — confirm PASS**

```
go test ./state/... -v
```

- [ ] **Step 5: Commit**

```
git add state/
git commit -m "feat: state machine with priority aggregation"
```

---

## Task 3: Config

**Files:**
- Create: `config/config.go`
- Create: `config/config_test.go`

- [ ] **Step 1: Write failing test**

`config/config_test.go`:
```go
package config

import (
    "path/filepath"
    "testing"
)

func TestRoundtrip(t *testing.T) {
    path := filepath.Join(t.TempDir(), "config.json")
    in := Config{X: 42, Y: 99, ClickThrough: true, Visible: false}
    if err := Save(path, in); err != nil {
        t.Fatal(err)
    }
    out, err := Load(path)
    if err != nil {
        t.Fatal(err)
    }
    if out != in {
        t.Errorf("got %+v, want %+v", out, in)
    }
}

func TestLoadMissing(t *testing.T) {
    cfg, err := Load("/no/such/file.json")
    if err != nil {
        t.Fatal("missing file should return defaults, not error")
    }
    if cfg != Default() {
        t.Errorf("got %+v, want %+v", cfg, Default())
    }
}
```

- [ ] **Step 2: Run — confirm FAIL**

```
go test ./config/...
```

- [ ] **Step 3: Implement**

`config/config.go`:
```go
package config

import (
    "encoding/json"
    "errors"
    "os"
)

type Config struct {
    X            int  `json:"x"`
    Y            int  `json:"y"`
    ClickThrough bool `json:"click_through"`
    Visible      bool `json:"visible"`
}

// Default returns the out-of-box config. X=-1 means "center screen at runtime".
func Default() Config {
    return Config{X: -1, Y: 16, ClickThrough: false, Visible: true}
}

func Load(path string) (Config, error) {
    data, err := os.ReadFile(path)
    if errors.Is(err, os.ErrNotExist) {
        return Default(), nil
    }
    if err != nil {
        return Default(), err
    }
    var cfg Config
    if err := json.Unmarshal(data, &cfg); err != nil {
        return Default(), nil // corrupt → safe fallback
    }
    return cfg, nil
}

func Save(path string, cfg Config) error {
    data, _ := json.MarshalIndent(cfg, "", "  ")
    return os.WriteFile(path, data, 0644)
}
```

- [ ] **Step 4: Run — confirm PASS**

```
go test ./config/... -v
```

- [ ] **Step 5: Commit**

```
git add config/
git commit -m "feat: config load/save with defaults"
```

---

## Task 4: Transcript Parser

**Files:**
- Create: `watcher/parser.go`
- Create: `watcher/parser_test.go`

- [ ] **Step 1: Write failing tests**

`watcher/parser_test.go`:
```go
package watcher

import (
    "testing"
    "claude-traffic-light/state"
)

func TestParseLastState(t *testing.T) {
    cases := []struct {
        name  string
        lines []string
        want  state.State
    }{
        {"empty", nil, state.Green},
        {
            "user message → idle",
            []string{`{"type":"user","message":{"role":"user","content":"hi"}}`},
            state.Green,
        },
        {
            "assistant text → thinking",
            []string{
                `{"type":"user","message":{"role":"user","content":"hi"}}`,
                `{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"sure"}]}}`,
            },
            state.Yellow,
        },
        {
            "tool_use → running",
            []string{
                `{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"1","name":"Bash","input":{}}]}}`,
            },
            state.Red,
        },
        {
            "tool_result after tool_use → idle",
            []string{
                `{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"1","name":"Bash","input":{}}]}}`,
                `{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"1","content":"ok"}]}}`,
            },
            state.Green,
        },
        {
            "malformed json → safe default green",
            []string{`not json`},
            state.Green,
        },
    }
    for _, c := range cases {
        t.Run(c.name, func(t *testing.T) {
            got := ParseLastState(c.lines)
            if got != c.want {
                t.Errorf("got %v, want %v", got, c.want)
            }
        })
    }
}
```

- [ ] **Step 2: Run — confirm FAIL**

```
go test ./watcher/... -run TestParseLastState
```

- [ ] **Step 3: Implement**

`watcher/parser.go`:
```go
package watcher

import (
    "encoding/json"
    "claude-traffic-light/state"
)

type transcriptLine struct {
    Type    string   `json:"type"`
    Message *message `json:"message,omitempty"`
}

type message struct {
    Content json.RawMessage `json:"content"`
}

type contentItem struct {
    Type string `json:"type"`
}

// ParseLastState infers state from the last meaningful line of a transcript.
func ParseLastState(lines []string) state.State {
    for i := len(lines) - 1; i >= 0; i-- {
        if lines[i] == "" {
            continue
        }
        var line transcriptLine
        if err := json.Unmarshal([]byte(lines[i]), &line); err != nil {
            continue
        }
        switch line.Type {
        case "user":
            return state.Green
        case "assistant":
            return assistantState(line.Message)
        }
    }
    return state.Green // empty or all-unknown → idle
}

func assistantState(msg *message) state.State {
    if msg == nil {
        return state.Yellow
    }
    var items []contentItem
    if err := json.Unmarshal(msg.Content, &items); err != nil {
        return state.Yellow
    }
    for _, item := range items {
        if item.Type == "tool_use" {
            return state.Red
        }
    }
    return state.Yellow
}
```

- [ ] **Step 4: Run — confirm PASS**

```
go test ./watcher/... -run TestParseLastState -v
```

- [ ] **Step 5: Commit**

```
git add watcher/parser.go watcher/parser_test.go
git commit -m "feat: JSONL transcript parser"
```

---

## Task 5: File Watcher

**Files:**
- Create: `watcher/watcher.go`
- Create: `watcher/watcher_test.go`

- [ ] **Step 1: Write failing integration test**

`watcher/watcher_test.go`:
```go
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
```

- [ ] **Step 2: Run — confirm FAIL**

```
go test ./watcher/... -run TestWatcherDetects -timeout 10s
```

- [ ] **Step 3: Implement**

`watcher/watcher.go`:
```go
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

    // Remove sessions whose files no longer exist
    for path := range w.sessions {
        found := false
        for _, p := range paths {
            if p == path { found = true; break }
        }
        if !found { delete(w.sessions, path) }
    }

    changed := false
    for _, path := range paths {
        info, err := os.Stat(path)
        if err != nil { continue }
        mod := info.ModTime()

        prev, exists := w.sessions[path]
        if exists && prev.modTime == mod { continue }

        s := parseFile(path)
        w.sessions[path] = sessionInfo{state: s, modTime: mod}
        changed = true
    }

    if changed || len(paths) == 0 {
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
    if err != nil { return state.Green }
    defer f.Close()
    var lines []string
    sc := bufio.NewScanner(f)
    for sc.Scan() { lines = append(lines, sc.Text()) }
    return ParseLastState(lines)
}

// ClaudeProjectsPath returns the path to ~/.claude/projects/
func ClaudeProjectsPath() string {
    home, _ := os.UserHomeDir()
    return filepath.Join(home, ".claude", "projects")
}
```

- [ ] **Step 4: Run — confirm PASS**

```
go test ./watcher/... -v -timeout 15s
```

- [ ] **Step 5: Commit**

```
git add watcher/watcher.go watcher/watcher_test.go
git commit -m "feat: polling file watcher with session aggregation"
```

---

## Task 6: Embed glass.html + Win32 Helpers

**Files:**
- Already exists: `ui/glass.html` (validated in browser session)
- Create: `ui/win32.go`

Note: GUI code has no automated tests — verify by running the app.

- [ ] **Step 1: Create ui/win32.go**

`ui/win32.go`:
```go
package ui

import (
    "unsafe"
    "golang.org/x/sys/windows"
)

var (
    user32  = windows.NewLazySystemDLL("user32.dll")
    gdi32   = windows.NewLazySystemDLL("gdi32.dll")
    dwmapi  = windows.NewLazySystemDLL("dwmapi.dll")
    shell32 = windows.NewLazySystemDLL("shell32.dll")
    gdipDLL = windows.NewLazySystemDLL("gdiplus.dll")

    procCreateWindowExW              = user32.NewProc("CreateWindowExW")
    procRegisterClassExW             = user32.NewProc("RegisterClassExW")
    procDefWindowProcW               = user32.NewProc("DefWindowProcW")
    procGetMessageW                  = user32.NewProc("GetMessageW")
    procTranslateMessage             = user32.NewProc("TranslateMessage")
    procDispatchMessageW             = user32.NewProc("DispatchMessageW")
    procPostQuitMessage              = user32.NewProc("PostQuitMessage")
    procShowWindow                   = user32.NewProc("ShowWindow")
    procUpdateWindow                 = user32.NewProc("UpdateWindow")
    procGetWindowRect                = user32.NewProc("GetWindowRect")
    procGetSystemMetrics             = user32.NewProc("GetSystemMetrics")
    procGetCursorPos                 = user32.NewProc("GetCursorPos")
    procSetForegroundWindow          = user32.NewProc("SetForegroundWindow")
    procBeginPaint                   = user32.NewProc("BeginPaint")
    procEndPaint                     = user32.NewProc("EndPaint")
    procInvalidateRect               = user32.NewProc("InvalidateRect")
    procSetWindowLongPtrW            = user32.NewProc("SetWindowLongPtrW")
    procGetWindowLongPtrW            = user32.NewProc("GetWindowLongPtrW")
    procSetTimer                     = user32.NewProc("SetTimer")
    procCreatePopupMenu              = user32.NewProc("CreatePopupMenu")
    procAppendMenuW                  = user32.NewProc("AppendMenuW")
    procTrackPopupMenu               = user32.NewProc("TrackPopupMenu")
    procDestroyMenu                  = user32.NewProc("DestroyMenu")
    procPostMessageW                 = user32.NewProc("PostMessageW")
    procLoadIconW                    = user32.NewProc("LoadIconW")

    procCreateRoundRectRgn           = gdi32.NewProc("CreateRoundRectRgn")
    procSetWindowRgn                 = user32.NewProc("SetWindowRgn")

    procDwmSetWindowAttribute        = dwmapi.NewProc("DwmSetWindowAttribute")
    procDwmExtendFrameIntoClientArea = dwmapi.NewProc("DwmExtendFrameIntoClientArea")

    procShellNotifyIconW             = shell32.NewProc("Shell_NotifyIconW")

    procGdiplusStartup               = gdipDLL.NewProc("GdiplusStartup")
    procGdiplusShutdown              = gdipDLL.NewProc("GdiplusShutdown")
    procGdipCreateFromHDC            = gdipDLL.NewProc("GdipCreateFromHDC")
    procGdipDeleteGraphics           = gdipDLL.NewProc("GdipDeleteGraphics")
    procGdipSetSmoothingMode         = gdipDLL.NewProc("GdipSetSmoothingMode")
    procGdipCreateSolidFill          = gdipDLL.NewProc("GdipCreateSolidFill")
    procGdipDeleteBrush              = gdipDLL.NewProc("GdipDeleteBrush")
    procGdipFillEllipseI             = gdipDLL.NewProc("GdipFillEllipseI")
)

const (
    WS_POPUP          = 0x80000000
    WS_EX_TOPMOST     = 0x00000008
    WS_EX_TOOLWINDOW  = 0x00000080
    WS_EX_NOACTIVATE  = 0x08000000
    WS_EX_TRANSPARENT = 0x00000020
    GWL_EXSTYLE       = -20
    SW_SHOW           = 5
    SW_HIDE           = 0

    WM_DESTROY      = 0x0002
    WM_PAINT        = 0x000F
    WM_TIMER        = 0x0113
    WM_EXITSIZEMOVE = 0x0232
    WM_NCHITTEST    = 0x0084
    WM_USER         = 0x0400
    WM_TRAY         = WM_USER + 1
    WM_STATE_CHANGE  = WM_USER + 2
    HTCAPTION       = 2

    TIMER_BLINK = 1

    DWMWA_SYSTEMBACKDROP_TYPE      = 38
    DWMWA_WINDOW_CORNER_PREFERENCE = 33
    DWMSBT_TRANSIENTWINDOW         = 3
    DWMWCP_ROUND                   = 2

    NIM_ADD     = 0
    NIM_MODIFY  = 1
    NIM_DELETE  = 2
    NIF_MESSAGE = 0x01
    NIF_ICON    = 0x02
    NIF_TIP     = 0x04

    MF_STRING    = 0x0000
    MF_CHECKED   = 0x0008
    MF_SEPARATOR = 0x0800
    TPM_RETURNCMD   = 0x0100
    TPM_RIGHTALIGN  = 0x0008
    TPM_BOTTOMALIGN = 0x0020

    MENU_TOGGLE_VISIBLE      = 1001
    MENU_TOGGLE_PASSTHROUGH  = 1002
    MENU_EXIT                = 1003

    SM_CXSCREEN = 0
)

type POINT struct{ X, Y int32 }
type RECT  struct{ Left, Top, Right, Bottom int32 }
type MARGINS struct{ Left, Right, Top, Bottom int32 }

type MSG struct {
    Hwnd    windows.HWND
    Message uint32
    WParam  uintptr
    LParam  uintptr
    Time    uint32
    Pt      POINT
}

type PAINTSTRUCT struct {
    Hdc         windows.Handle
    FErase      int32
    RcPaint     RECT
    FRestore    int32
    FIncUpdate  int32
    Reserved    [32]byte
}

type WNDCLASSEXW struct {
    Size       uint32
    Style      uint32
    WndProc    uintptr
    ClsExtra   int32
    WndExtra   int32
    Instance   windows.Handle
    Icon       windows.Handle
    Cursor     windows.Handle
    Background windows.Handle
    MenuName   *uint16
    ClassName  *uint16
    IconSm     windows.Handle
}

type NOTIFYICONDATAW struct {
    CbSize           uint32
    HWnd             windows.HWND
    UID              uint32
    UFlags           uint32
    UCallbackMessage uint32
    HIcon            windows.Handle
    SzTip            [128]uint16
    DwState          uint32
    DwStateMask      uint32
    SzInfo           [256]uint16
    UVersion         uint32
    SzInfoTitle      [64]uint16
    DwInfoFlags      uint32
    GuidItem         [16]byte
    HBalloonIcon     windows.Handle
}

type gdipInput struct {
    Version                    uint32
    DebugEventCallback         uintptr
    SuppressBackgroundThread   int32
    SuppressExternalCodecs     int32
}

func u16(s string) *uint16 { p, _ := windows.UTF16PtrFromString(s); return p }
func loword(l uintptr) uint16 { return uint16(l) }
func sysMetric(n int) int { r, _, _ := procGetSystemMetrics.Call(uintptr(n)); return int(r) }

// gdipToken holds the GDI+ startup token.
var gdipToken uintptr

func InitGDIPlus() {
    in := gdipInput{Version: 1}
    procGdiplusStartup.Call(
        uintptr(unsafe.Pointer(&gdipToken)),
        uintptr(unsafe.Pointer(&in)),
        0,
    )
}

func ShutdownGDIPlus() { procGdiplusShutdown.Call(gdipToken) }
```

- [ ] **Step 2: Commit**

```
git add ui/win32.go
git commit -m "feat: Win32 constants, types, GDI+ init helpers"
```

---

## Task 7: WebView2 Window + Tray + Win32 Behaviors

**Files:**
- Create: `ui/window.go`
- Create: `main.go` (minimal — will be extended in Task 10)

- [ ] **Step 1: Create ui/window.go**

`ui/window.go`:
```go
package ui

import (
    "unsafe"
    "golang.org/x/sys/windows"
    "claude-traffic-light/config"
    "claude-traffic-light/state"
)

const (
    winWidth  = 120
    winHeight = 44
    winClass  = "ClaudeTrafficLight"
)

type Window struct {
    hwnd    windows.HWND
    cfg     config.Config
    cfgPath string
    cur     state.State
    blink   bool
    trayOK  bool
}

var gw *Window // WndProc needs access to Window; single instance only

func New(cfgPath string, cfg config.Config) *Window {
    w := &Window{cfg: cfg, cfgPath: cfgPath}
    gw = w

    hInst, _ := windows.GetModuleHandle(nil)
    cls := u16(winClass)

    wc := WNDCLASSEXW{
        Size:      uint32(unsafe.Sizeof(WNDCLASSEXW{})),
        WndProc:   windows.NewCallback(wndProc),
        Instance:  hInst,
        ClassName: cls,
        // Background = 0 → DWM paints the background
    }
    procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))

    x := cfg.X
    if x < 0 {
        x = (sysMetric(SM_CXSCREEN) - winWidth) / 2
    }

    exStyle := uintptr(WS_EX_TOPMOST | WS_EX_TOOLWINDOW | WS_EX_NOACTIVATE)
    hwnd, _, _ := procCreateWindowExW.Call(
        exStyle,
        uintptr(unsafe.Pointer(cls)),
        uintptr(unsafe.Pointer(u16("Claude Traffic Light"))),
        uintptr(WS_POPUP),
        uintptr(x), uintptr(cfg.Y),
        uintptr(winWidth), uintptr(winHeight),
        0, 0, uintptr(hInst), 0,
    )
    w.hwnd = windows.HWND(hwnd)

    applyDWM(w.hwnd)
    applyPillShape(w.hwnd)

    if cfg.ClickThrough {
        w.setPassthrough(true)
    }

    if cfg.Visible {
        procShowWindow.Call(hwnd, SW_SHOW)
        procUpdateWindow.Call(hwnd)
    }

    w.addTray()
    procSetTimer.Call(hwnd, TIMER_BLINK, 425, 0)
    return w
}

// Run starts the Win32 message loop. Blocks until the window is destroyed.
func (w *Window) Run() {
    var msg MSG
    for {
        r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
        if r == 0 { break }
        procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
        procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
    }
    w.removeTray()
}

// PostState safely delivers a state change from any goroutine.
func (w *Window) PostState(s state.State) {
    procPostMessageW.Call(uintptr(w.hwnd), WM_STATE_CHANGE, uintptr(s), 0)
}

func (w *Window) setPassthrough(on bool) {
    r, _, _ := procGetWindowLongPtrW.Call(uintptr(w.hwnd), uintptr(GWL_EXSTYLE))
    style := uintptr(r)
    if on { style |= WS_EX_TRANSPARENT } else { style &^= WS_EX_TRANSPARENT }
    procSetWindowLongPtrW.Call(uintptr(w.hwnd), uintptr(GWL_EXSTYLE), style)
}

func applyDWM(hwnd windows.HWND) {
    m := MARGINS{-1, -1, -1, -1}
    procDwmExtendFrameIntoClientArea.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&m)))
    v := uint32(DWMSBT_TRANSIENTWINDOW)
    procDwmSetWindowAttribute.Call(uintptr(hwnd), DWMWA_SYSTEMBACKDROP_TYPE, uintptr(unsafe.Pointer(&v)), 4)
    c := uint32(DWMWCP_ROUND)
    procDwmSetWindowAttribute.Call(uintptr(hwnd), DWMWA_WINDOW_CORNER_PREFERENCE, uintptr(unsafe.Pointer(&c)), 4)
}

func applyPillShape(hwnd windows.HWND) {
    // Corner radius = winHeight gives perfect pill shape
    rgn, _, _ := procCreateRoundRectRgn.Call(
        0, 0, uintptr(winWidth+1), uintptr(winHeight+1),
        uintptr(winHeight), uintptr(winHeight),
    )
    procSetWindowRgn.Call(uintptr(hwnd), rgn, 1)
}

// wndProc is the Win32 window procedure.
func wndProc(hwnd windows.HWND, msg uint32, wParam, lParam uintptr) uintptr {
    w := gw
    switch msg {

    case WM_PAINT:
        var ps PAINTSTRUCT
        procBeginPaint.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&ps)))
        drawLamps(windows.Handle(ps.Hdc), w.cur, w.blink)
        procEndPaint.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&ps)))
        return 0

    case WM_TIMER:
        if wParam == TIMER_BLINK {
            w.blink = !w.blink
            procInvalidateRect.Call(uintptr(hwnd), 0, 1)
        }
        return 0

    case WM_STATE_CHANGE:
        w.cur = state.State(wParam)
        w.updateTray(w.cur)
        procInvalidateRect.Call(uintptr(hwnd), 0, 1)
        return 0

    case WM_NCHITTEST:
        // Make entire client area act as title bar for drag — unless passthrough
        if !w.cfg.ClickThrough {
            return HTCAPTION
        }

    case WM_EXITSIZEMOVE:
        // Drag ended — persist new position
        var r RECT
        procGetWindowRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&r)))
        w.cfg.X = int(r.Left)
        w.cfg.Y = int(r.Top)
        config.Save(w.cfgPath, w.cfg)
        return 0

    case WM_TRAY:
        if loword(lParam) == 0x0205 { // WM_RBUTTONUP
            w.showMenu()
        }
        return 0

    case WM_DESTROY:
        procPostQuitMessage.Call(0)
        return 0
    }

    r, _, _ := procDefWindowProcW.Call(uintptr(hwnd), uintptr(msg), wParam, lParam)
    return r
}

func (w *Window) showMenu() {
    menu, _, _ := procCreatePopupMenu.Call()
    defer procDestroyMenu.Call(menu)

    visLabel := "隐藏窗口"
    if !w.cfg.Visible { visLabel = "显示窗口" }
    procAppendMenuW.Call(menu, MF_STRING, MENU_TOGGLE_VISIBLE,
        uintptr(unsafe.Pointer(u16(visLabel))))

    ptFlags := uintptr(MF_STRING)
    ptLabel := "开启穿透"
    if w.cfg.ClickThrough { ptFlags |= MF_CHECKED; ptLabel = "关闭穿透" }
    procAppendMenuW.Call(menu, ptFlags, MENU_TOGGLE_PASSTHROUGH,
        uintptr(unsafe.Pointer(u16(ptLabel))))

    procAppendMenuW.Call(menu, MF_SEPARATOR, 0, 0)
    procAppendMenuW.Call(menu, MF_STRING, MENU_EXIT,
        uintptr(unsafe.Pointer(u16("退出"))))

    var pt POINT
    procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
    procSetForegroundWindow.Call(uintptr(w.hwnd))

    cmd, _, _ := procTrackPopupMenu.Call(
        menu,
        TPM_RETURNCMD|TPM_RIGHTALIGN|TPM_BOTTOMALIGN,
        uintptr(pt.X), uintptr(pt.Y),
        0, uintptr(w.hwnd), 0,
    )
    switch cmd {
    case MENU_TOGGLE_VISIBLE:
        w.cfg.Visible = !w.cfg.Visible
        if w.cfg.Visible { procShowWindow.Call(uintptr(w.hwnd), SW_SHOW) } else { procShowWindow.Call(uintptr(w.hwnd), SW_HIDE) }
        config.Save(w.cfgPath, w.cfg)
    case MENU_TOGGLE_PASSTHROUGH:
        w.cfg.ClickThrough = !w.cfg.ClickThrough
        w.setPassthrough(w.cfg.ClickThrough)
        config.Save(w.cfgPath, w.cfg)
    case MENU_EXIT:
        procPostQuitMessage.Call(0)
    }
}

// addTray, updateTray, removeTray — in tray.go
```

Add to `ui/win32.go`:
```go
var procGetWindowRect = user32.NewProc("GetWindowRect")
```

- [ ] **Step 2: Create ui/tray.go**

`ui/tray.go`:
```go
package ui

import (
    "unsafe"
    "golang.org/x/sys/windows"
    "claude-traffic-light/state"
)

func (w *Window) addTray() {
    nid := w.nid(state.Grey)
    procShellNotifyIconW.Call(NIM_ADD, uintptr(unsafe.Pointer(&nid)))
    w.trayOK = true
}

func (w *Window) updateTray(s state.State) {
    if !w.trayOK { return }
    nid := w.nid(s)
    procShellNotifyIconW.Call(NIM_MODIFY, uintptr(unsafe.Pointer(&nid)))
}

func (w *Window) removeTray() {
    if !w.trayOK { return }
    nid := w.nid(state.Grey)
    procShellNotifyIconW.Call(NIM_DELETE, uintptr(unsafe.Pointer(&nid)))
}

func (w *Window) nid(s state.State) NOTIFYICONDATAW {
    tip := [128]uint16{}
    label := "Claude — " + s.String()
    for i, r := range []rune(label) {
        if i >= 127 { break }
        tip[i] = uint16(r)
    }
    hInst, _ := windows.GetModuleHandle(nil)
    hIcon, _, _ := procLoadIconW.Call(uintptr(hInst), 32512) // IDI_APPLICATION
    return NOTIFYICONDATAW{
        CbSize:           uint32(unsafe.Sizeof(NOTIFYICONDATAW{})),
        HWnd:             w.hwnd,
        UID:              1,
        UFlags:           NIF_MESSAGE | NIF_ICON | NIF_TIP,
        UCallbackMessage: WM_TRAY,
        HIcon:            windows.Handle(hIcon),
        SzTip:            tip,
    }
}
```

- [ ] **Step 3: Create minimal main.go**

`main.go`:
```go
package main

import (
    "os"
    "path/filepath"
    "claude-traffic-light/config"
    "claude-traffic-light/ui"
)

func main() {
    ui.InitGDIPlus()
    defer ui.ShutdownGDIPlus()

    exePath, _ := os.Executable()
    cfgPath := filepath.Join(filepath.Dir(exePath), "config.json")
    cfg, _ := config.Load(cfgPath)

    win := ui.New(cfgPath, cfg)
    win.Run()
}
```

- [ ] **Step 4: Build and verify window appears**

```
go build -ldflags="-H windowsgui" -o claude-traffic-light.exe .
.\claude-traffic-light.exe
```

Expected:
- Pill-shaped window at top-center of screen, frosted glass appearance
- No taskbar entry (tool window)
- Tray icon in system tray
- Right-click tray → menu with 3 items
- "穿透" toggle works (click through / drag)
- Window can be dragged; position saved on release

- [ ] **Step 5: Commit**

```
git add ui/window.go ui/tray.go main.go
git commit -m "feat: Win32 window with DWM Acrylic, tray, drag, click-through"
```

---

## Task 8: Go ↔ WebView2 State Communication

**Files:**
- Create: `ui/draw.go`

- [ ] **Step 1: Create ui/draw.go**

`ui/draw.go`:
```go
package ui

import (
    "unsafe"
    "golang.org/x/sys/windows"
    "claude-traffic-light/state"
)

// argb packs an ARGB color (GDI+ format).
func argb(a, r, g, b uint8) uint32 {
    return uint32(a)<<24 | uint32(r)<<16 | uint32(g)<<8 | uint32(b)
}

const (
    lampDiam    = 22
    lampSpacing = 32 // center-to-center
    lampCY      = winHeight / 2
)

func lampCX(i int) int {
    total := lampSpacing*2 + lampDiam
    start := (winWidth-total)/2 + lampDiam/2
    return start + i*lampSpacing
}

type lampDef struct{ on, off, glow uint32 }

var lamps = [3]lampDef{
    {argb(255,200,0,20),  argb(255,35,28,30),  argb(70,220,0,20)},   // red
    {argb(255,210,140,0), argb(255,35,30,20),  argb(70,210,140,0)},  // yellow
    {argb(255,0,160,60),  argb(255,20,35,25),  argb(70,0,180,60)},   // green
}

// activeMap returns which lamps are lit given state and blink phase.
func activeMap(s state.State, blink bool) [3]bool {
    switch s {
    case state.Red:    return [3]bool{blink, false, false}
    case state.Yellow: return [3]bool{false, blink, false}
    case state.Green:  return [3]bool{false, false, true}
    default:           return [3]bool{false, false, false}
    }
}

// drawLamps renders the three traffic light circles onto hdc.
func drawLamps(hdc windows.Handle, s state.State, blink bool) {
    var g uintptr
    procGdipCreateFromHDC.Call(uintptr(hdc), uintptr(unsafe.Pointer(&g)))
    defer procGdipDeleteGraphics.Call(g)
    procGdipSetSmoothingMode.Call(g, 2) // AntiAlias

    active := activeMap(s, blink)
    for i, def := range lamps {
        cx, cy := lampCX(i), lampCY
        if active[i] {
            // glow ring
            gs := lampDiam + 10
            fillCircle(g, def.glow, cx-gs/2, cy-gs/2, gs)
            fillCircle(g, def.on, cx-lampDiam/2, cy-lampDiam/2, lampDiam)
        } else {
            fillCircle(g, def.off, cx-lampDiam/2, cy-lampDiam/2, lampDiam)
        }
        // highlight dot
        hs := lampDiam / 3
        fillCircle(g, argb(150,255,255,255), cx-lampDiam/4, cy-lampDiam/2+3, hs)
    }
}

func fillCircle(g uintptr, color uint32, x, y, d int) {
    var brush uintptr
    procGdipCreateSolidFill.Call(uintptr(color), uintptr(unsafe.Pointer(&brush)))
    defer procGdipDeleteBrush.Call(brush)
    procGdipFillEllipseI.Call(g, brush, uintptr(x), uintptr(y), uintptr(d), uintptr(d))
}
```

- [ ] **Step 2: Build and verify lamp rendering**

```
go build -ldflags="-H windowsgui" -o claude-traffic-light.exe .
.\claude-traffic-light.exe
```

Expected: Three circles visible on window. Default (no sessions) = all grey. Blink timer fires at 425ms intervals.

- [ ] **Step 3: Manually cycle states to verify all visuals**

Add a temporary test in `main.go` after `ui.New(...)`:

```go
import "time"
// ... after win := ui.New(...)
go func() {
    for _, s := range []state.State{state.Green, state.Yellow, state.Red, state.Grey} {
        time.Sleep(2 * time.Second)
        win.PostState(s)
    }
}()
```

Build, run. Expected: green (solid) → yellow (blink) → red (blink) → grey (all off).

Remove the test goroutine and rebuild.

- [ ] **Step 4: Commit**

```
git add ui/draw.go
git commit -m "feat: GDI+ lamp rendering with glow and blink"
```

---

## Task 9: Wire Watcher → Window

**Files:**
- Modify: `main.go`

- [ ] **Step 1: Update main.go**

```go
package main

import (
    "os"
    "path/filepath"
    "time"
    "claude-traffic-light/config"
    "claude-traffic-light/state"
    "claude-traffic-light/ui"
    "claude-traffic-light/watcher"
)

func main() {
    exePath, _ := os.Executable()
    cfgPath := filepath.Join(filepath.Dir(exePath), "config.json")
    cfg, _ := config.Load(cfgPath)

    // win.SetState(s) calls webview.Eval("setState('running')")
    win := ui.New(cfgPath, cfg)

    w, err := watcher.New(
        watcher.ClaudeProjectsPath(),
        60*time.Second,
        func(s state.State) { win.SetState(s) },
    )
    if err == nil {
        go w.Watch()
        defer w.Stop()
    }

    win.Run() // blocks on WebView2 message loop
}
```

- [ ] **Step 2: Build**

```
go build -ldflags="-H windowsgui" -o claude-traffic-light.exe .
```

- [ ] **Step 3: End-to-end integration test**

1. Run `.\claude-traffic-light.exe`
2. All lamps grey (no active sessions) OR green if sessions exist
3. Open a Claude Code session in VS Code or terminal
4. Within 250ms: green lamp activates
5. Submit a prompt to Claude Code
6. Within 1–3s: yellow blink (thinking state detected)
7. When Claude Code runs a tool (Bash, Write, etc.): red blink
8. When Claude finishes: green solid
9. Close Claude Code, wait 60s: all grey

- [ ] **Step 4: Commit**

```
git add main.go
git commit -m "feat: wire watcher to WebView2 window via setState JS injection"
```

---

## Task 10: Portable Build

**Files:**
- Create: `app.manifest`
- Create: `rsrc.syso` (generated)

- [ ] **Step 1: Install rsrc tool**

```
go install github.com/akavel/rsrc@latest
```

- [ ] **Step 2: Create app.manifest**

`app.manifest`:
```xml
<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<assembly xmlns="urn:schemas-microsoft-com:asm.v1" manifestVersion="1.0">
  <assemblyIdentity version="1.0.0.0" processorArchitecture="*"
    name="ClaudeTrafficLight" type="win32"/>
  <dependency>
    <dependentAssembly>
      <assemblyIdentity type="win32"
        name="Microsoft.Windows.Common-Controls" version="6.0.0.0"
        processorArchitecture="*" publicKeyToken="6595b64144ccf1df" language="*"/>
    </dependentAssembly>
  </dependency>
  <trustInfo xmlns="urn:schemas-microsoft-com:asm.v3">
    <security><requestedPrivileges>
      <requestedExecutionLevel level="asInvoker" uiAccess="false"/>
    </requestedPrivileges></security>
  </trustInfo>
  <application xmlns="urn:schemas-microsoft-com:asm.v3">
    <windowsSettings>
      <dpiAwareness xmlns="http://schemas.microsoft.com/SMI/2016/WindowsSettings">PerMonitorV2</dpiAwareness>
    </windowsSettings>
  </application>
</assembly>
```

- [ ] **Step 3: Embed manifest**

```
rsrc -manifest app.manifest -o rsrc.syso
```

Expected: `rsrc.syso` created. Go auto-embeds it at build time.

- [ ] **Step 4: Final portable build**

```
go build -ldflags="-H windowsgui -s -w" -o claude-traffic-light.exe .
```

Expected: single `claude-traffic-light.exe`, ~8–12 MB.

- [ ] **Step 5: Portability test**

Copy only `claude-traffic-light.exe` to a clean folder (no `config.json`). Run it.

Expected:
- Window appears at top-center
- `config.json` created automatically next to the exe
- All features work without any other files

- [ ] **Step 6: Commit**

```
git add app.manifest rsrc.syso
git commit -m "feat: DPI-aware manifest — portable release build complete"
```

---

## Quick Reference

```bash
# Run all tests (state, config, watcher — all pure Go, no UI)
go test ./...

# Development build (shows console for debug output)
go build -o claude-traffic-light.exe .

# Release build (no console, stripped)
go build -ldflags="-H windowsgui -s -w" -o claude-traffic-light.exe .
```

## Known Limitations (from spec)

- **Yellow lag**: Transcript isn't written until Claude starts outputting. Yellow appears 1–3s after user submits.
- **WebView2 runtime**: Requires WebView2 runtime (pre-installed on Windows 10 1803+ and all Windows 11). If absent, app shows an install prompt.
- **Liquid glass browser support**: `backdrop-filter: url(#svg-filter)` is Chrome/Edge only — WebView2 uses system Edge, so this works correctly.
- **Poll interval**: State changes are detected within 250ms of transcript update.
