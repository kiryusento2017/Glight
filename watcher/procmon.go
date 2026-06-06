package watcher

import (
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	kernel32P                     = windows.NewLazySystemDLL("kernel32.dll")
	procCreateToolhelp32Snapshot  = kernel32P.NewProc("CreateToolhelp32Snapshot")
	procProcess32FirstW           = kernel32P.NewProc("Process32FirstW")
	procProcess32NextW            = kernel32P.NewProc("Process32NextW")
	procCloseHandle               = kernel32P.NewProc("CloseHandle")
	procOpenProcess               = kernel32P.NewProc("OpenProcess")
	procQueryFullProcessImageNameW = kernel32P.NewProc("QueryFullProcessImageNameW")
)

const (
	th32csSnapprocess       = 2
	processQueryLimitedInfo = 0x1000
	imageFileWin32Path      = 1 // Win32 path format (C:\...)
)

type processEntry32W struct {
	dwSize              uint32
	cntUsage            uint32
	th32ProcessID       uint32
	th32DefaultHeapID   uintptr
	th32ModuleID        uint32
	cntThreads          uint32
	th32ParentProcessID uint32
	pcPriClassBase      int32
	dwFlags             uint32
	szExeFile           [260]uint16
}

// isClaudeCodeRunning 枚举所有进程，检查是否有 Claude Code 进程。
// 匹配规则：进程名 claude.exe 且完整路径以 \.local\bin\claude.exe 结尾，
// 区分同名的 Claude Desktop（安装在别处）。
func isClaudeCodeRunning() bool {
	h, _, _ := procCreateToolhelp32Snapshot.Call(th32csSnapprocess, 0)
	if h == 0 || h == ^uintptr(0) {
		return false
	}
	defer syscall.CloseHandle(syscall.Handle(h))

	var pe processEntry32W
	pe.dwSize = uint32(unsafe.Sizeof(pe))
	r, _, _ := procProcess32FirstW.Call(h, uintptr(unsafe.Pointer(&pe)))
	for r != 0 {
		name := strings.ToLower(windows.UTF16ToString(pe.szExeFile[:]))
		if name == "claude.exe" && isClaudeCodeImage(pe.th32ProcessID) {
			return true
		}
		pe.dwSize = uint32(unsafe.Sizeof(pe))
		r, _, _ = procProcess32NextW.Call(h, uintptr(unsafe.Pointer(&pe)))
	}
	return false
}

// isClaudeCodeImage 打开进程获取完整路径，检查是否以 \.local\bin\claude.exe 结尾。
func isClaudeCodeImage(pid uint32) bool {
	hp, _, _ := procOpenProcess.Call(processQueryLimitedInfo, 0, uintptr(pid))
	if hp == 0 {
		return false
	}
	defer syscall.CloseHandle(syscall.Handle(hp))

	var buf [260]uint16
	n := uint32(len(buf))
	r, _, _ := procQueryFullProcessImageNameW.Call(hp, imageFileWin32Path, uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&n)))
	if r == 0 {
		return false
	}
	path := strings.ToLower(windows.UTF16ToString(buf[:n]))
	return strings.HasSuffix(path, `\.local\bin\claude.exe`)
}
