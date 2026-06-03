package main

import (
	"os"
	"path/filepath"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"

	"claude-traffic-light/config"
	"claude-traffic-light/state"
	"claude-traffic-light/ui"
	"claude-traffic-light/watcher"
)

var (
	kernel32              = windows.NewLazySystemDLL("kernel32.dll")
	procCreateMutexW      = kernel32.NewProc("CreateMutexW")
)

func main() {
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

	win.Run()
}
