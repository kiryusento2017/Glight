package main

import (
	_ "embed"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"

	"claude-traffic-light/config"
	"claude-traffic-light/state"
	"claude-traffic-light/ui"
	"claude-traffic-light/watcher"
)

//go:embed claude-traffic-light.ico
var icoData []byte // 运行时图标资源（256+32+16 三合一 .ico）

var (
	kernel32         = windows.NewLazySystemDLL("kernel32.dll")
	procCreateMutexW = kernel32.NewProc("CreateMutexW")
	procCloseHandle  = kernel32.NewProc("CloseHandle")
)

func main() {
	// hook 子命令模式：被 Claude Code 的 hook 调用，写状态文件后立即退出。
	// 用法：claude-traffic-light.exe hook <running|thinking|idle>
	if len(os.Args) >= 3 && os.Args[1] == "hook" {
		writeHookState(readSessionID(), os.Args[2])
		return
	}

	// 单实例。重启场景（新进程带 --restarted 标记）撞锁时轮询等旧实例退出再
	// 接管，最多等 5s；普通双开撞锁直接退出，单实例语义不变。
	restarted := len(os.Args) >= 2 && os.Args[1] == "--restarted"
	mutexName, _ := windows.UTF16PtrFromString("Local\\ClaudeTrafficLight_SingleInstance")
	for i := 0; ; i++ {
		h, _, errCode := procCreateMutexW.Call(0, 0, uintptr(unsafe.Pointer(mutexName)))
		// syscall.Call 第三个返回值为 GetLastError，在函数返回前已捕获，不会被覆盖
		if errCode != windows.ERROR_ALREADY_EXISTS {
			break // 拿到锁（旧实例已退出或本就没有）；h 持有至进程结束
		}
		procCloseHandle.Call(h) // 这次没拿到，关掉句柄避免泄露
		if !restarted || i >= 50 {
			os.Exit(0) // 普通双开 → 退；重启等待超 5s（旧实例疑似卡死）→ 放弃
		}
		time.Sleep(100 * time.Millisecond)
	}

	exePath, _ := os.Executable()
	cfgPath := filepath.Join(filepath.Dir(exePath), "config.json")
	cfg, _ := config.Load(cfgPath)

	// 路径自校正：exe 换盘/改名则更新注册表；config 与注册表双向对齐
	config.SyncAutostart(exePath, cfg.Startup)

	// 自动把状态 hook 合并进 ~/.claude/settings.json（幂等、只增不删、先备份）
	installHooks()

	ui.SetProcessDPIAware() // 进程级 DPI 感知（创建窗口前，替代 manifest）
	win := ui.New(cfgPath, cfg, icoData)

	w := watcher.New(hookStateDir(), func(s state.State) { win.SetState(s) })
	go w.Watch()
	defer w.Stop()

	win.Run()
}

// hookStateDir 返回状态文件所在目录（~/.claude/agent-light/）。每个会话写
// agent-light-state-<session_id>，挂件聚合该目录下所有此前缀文件。
// 放在子目录避免在 ~/.claude/ 根目录摊一堆文件。
func hookStateDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude", "agent-light")
}

// sessionFileName 把 session_id 规整为安全文件名；空则用 "default"。
func sessionFileName(sid string) string {
	sid = sanitizeSessionID(sid)
	if sid == "" {
		sid = "default"
	}
	return "agent-light-state-" + sid
}

// sanitizeSessionID 仅保留字母/数字/连字符，防非法文件名。
func sanitizeSessionID(sid string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-':
			return r
		default:
			return -1
		}
	}, sid)
}

// parseSessionID 从 hook 的 stdin JSON 取 session_id 字段（坏/空 JSON 返回空）。
func parseSessionID(data []byte) string {
	var p struct {
		SessionID string `json:"session_id"`
	}
	_ = json.Unmarshal(data, &p)
	return p.SessionID
}

// readSessionID 从 hook 的 stdin 读 JSON 取 session_id，带 500ms 硬超时——
// 拿不到（超时/空/坏 JSON）返回空，调用方回退 "default"。保证 hook 永远
// 毫秒级返回、绝不阻塞 Claude Code。
func readSessionID() string {
	ch := make(chan string, 1)
	go func() {
		data, _ := io.ReadAll(os.Stdin)
		ch <- parseSessionID(data)
	}()
	select {
	case sid := <-ch:
		return sid
	case <-time.After(500 * time.Millisecond):
		return ""
	}
}

// writeHookState 把状态词写入该会话的状态文件（hook 子命令用，极简快速、立即退出）。
// 首次调用时自动创建 agent-light 子目录（如不存在）。
func writeHookState(sid, s string) {
	dir := hookStateDir()
	os.MkdirAll(dir, 0755)
	os.WriteFile(filepath.Join(dir, sessionFileName(sid)), []byte(s), 0644)
}
