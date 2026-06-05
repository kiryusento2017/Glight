package main

import (
	_ "embed"
	"os"
	"path/filepath"
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
	kernel32              = windows.NewLazySystemDLL("kernel32.dll")
	procCreateMutexW      = kernel32.NewProc("CreateMutexW")
)

func main() {
	// hook 子命令模式：被 Claude Code 的 hook 调用，写状态文件后立即退出。
	// 用法：claude-traffic-light.exe hook <running|thinking|idle>
	if len(os.Args) >= 3 && os.Args[1] == "hook" {
		writeHookState(os.Args[2])
		return
	}

	// 单实例
	mutexName, _ := windows.UTF16PtrFromString("Local\\ClaudeTrafficLight_SingleInstance")
	_, _, errCode := procCreateMutexW.Call(0, 0, uintptr(unsafe.Pointer(mutexName)))
	// syscall.Call 第三个返回值为 GetLastError，在函数返回前已捕获，不会被覆盖
	if errCode == windows.ERROR_ALREADY_EXISTS {
		os.Exit(0)
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

	w := watcher.New(hookStatePath(), func(s state.State) { win.SetState(s) })
	go w.Watch()
	defer w.Stop()

	win.Run()
}

// hookStatePath 返回状态文件路径（hook 子命令写、挂件主进程读）。
func hookStatePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude", "agent-light-state")
}

// writeHookState 把状态词写入状态文件（hook 子命令用，极简快速、立即退出）。
func writeHookState(s string) {
	os.WriteFile(hookStatePath(), []byte(s), 0644)
}
