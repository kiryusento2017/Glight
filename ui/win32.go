package ui

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	user32  = windows.NewLazySystemDLL("user32.dll")
	shell32 = windows.NewLazySystemDLL("shell32.dll")
	dwmapi  = windows.NewLazySystemDLL("dwmapi.dll")

	procSetWindowLongPtrW  = user32.NewProc("SetWindowLongPtrW")
	procGetWindowLongPtrW  = user32.NewProc("GetWindowLongPtrW")
	procSetWindowPos       = user32.NewProc("SetWindowPos")
	procGetWindowRect      = user32.NewProc("GetWindowRect")
	procGetSystemMetrics   = user32.NewProc("GetSystemMetrics")
	procCreatePopupMenu    = user32.NewProc("CreatePopupMenu")
	procAppendMenuW        = user32.NewProc("AppendMenuW")
	procTrackPopupMenu     = user32.NewProc("TrackPopupMenu")
	procDestroyMenu        = user32.NewProc("DestroyMenu")
	procSetForegroundWindow = user32.NewProc("SetForegroundWindow")
	procGetCursorPos       = user32.NewProc("GetCursorPos")
	procShellNotifyIconW   = shell32.NewProc("Shell_NotifyIconW")
	procLoadIconW          = user32.NewProc("LoadIconW")
	procPostMessageW       = user32.NewProc("PostMessageW")
	procDefWindowProcW     = user32.NewProc("DefWindowProcW")
	procRegisterClassExW   = user32.NewProc("RegisterClassExW")
	procCreateWindowExW    = user32.NewProc("CreateWindowExW")

	procDwmSetWindowAttribute = dwmapi.NewProc("DwmSetWindowAttribute")
)

// GWL_STYLE and GWL_EXSTYLE are negative; declared as typed vars to avoid uintptr overflow.
var gwlStyle   = -16
var gwlExStyle = -20

const (
	WS_POPUP         = 0x80000000
	WS_VISIBLE       = 0x10000000
	WS_EX_TOPMOST    = 0x00000008
	WS_EX_TOOLWINDOW = 0x00000080
	WS_EX_NOACTIVATE = 0x08000000
	WS_EX_TRANSPARENT = 0x00000020
	WS_EX_LAYERED    = 0x00080000
	WS_CAPTION       = 0x00C00000
	WS_THICKFRAME    = 0x00040000

	HWND_TOPMOST = ^uintptr(0) // (HWND)(-1)
	SWP_NOMOVE   = 0x0002
	SWP_NOSIZE   = 0x0001
	SWP_NOACTIVATE = 0x0010
	SWP_FRAMECHANGED = 0x0020

	SM_CXSCREEN = 0
	SM_CYSCREEN = 1

	MF_STRING    = 0x0000
	MF_CHECKED   = 0x0008
	MF_SEPARATOR = 0x0800
	TPM_RETURNCMD   = 0x0100
	TPM_RIGHTALIGN  = 0x0008
	TPM_BOTTOMALIGN = 0x0020

	NIM_ADD    = 0
	NIM_MODIFY = 1
	NIM_DELETE = 2
	NIF_MESSAGE = 0x01
	NIF_ICON    = 0x02
	NIF_TIP     = 0x04

	WM_USER = 0x0400
	WM_TRAY = WM_USER + 1
	WM_LBUTTONDBLCLK = 0x0203
	WM_RBUTTONUP     = 0x0205

	IDI_APPLICATION = 32512

	MENU_SHOW_HIDE      = 1001
	MENU_PASSTHROUGH    = 1002
	MENU_EXIT           = 1003

	DWMWA_WINDOW_CORNER_PREFERENCE = 33
	DWMWCP_ROUND = 2
)

type POINT struct{ X, Y int32 }
type RECT  struct{ Left, Top, Right, Bottom int32 }

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

func u16(s string) *uint16 { p, _ := windows.UTF16PtrFromString(s); return p }
func sysMetric(n int) int  { r, _, _ := procGetSystemMetrics.Call(uintptr(n)); return int(r) }

// applyWindowStyles sets the window to: frameless, topmost, not in taskbar, no activation.
func applyWindowStyles(hwnd windows.HWND) {
	// Remove caption and thick frame, replace with popup style
	style, _, _ := procGetWindowLongPtrW.Call(uintptr(hwnd), uintptr(gwlStyle))
	style &^= uintptr(WS_CAPTION | WS_THICKFRAME)
	style |= uintptr(WS_POPUP)
	procSetWindowLongPtrW.Call(uintptr(hwnd), uintptr(gwlStyle), style)

	// Add topmost + toolwindow (hidden from taskbar) + no activate
	exStyle, _, _ := procGetWindowLongPtrW.Call(uintptr(hwnd), uintptr(gwlExStyle))
	exStyle |= uintptr(WS_EX_TOPMOST | WS_EX_TOOLWINDOW | WS_EX_NOACTIVATE)
	procSetWindowLongPtrW.Call(uintptr(hwnd), uintptr(gwlExStyle), exStyle)

	// Apply rounded corners (Windows 11)
	corners := uint32(DWMWCP_ROUND)
	procDwmSetWindowAttribute.Call(uintptr(hwnd), DWMWA_WINDOW_CORNER_PREFERENCE, uintptr(unsafe.Pointer(&corners)), 4)

	// Force redraw
	procSetWindowPos.Call(
		uintptr(hwnd), HWND_TOPMOST,
		0, 0, 0, 0,
		SWP_NOMOVE|SWP_NOSIZE|SWP_NOACTIVATE|SWP_FRAMECHANGED,
	)
}

// setPassthrough enables/disables click-through on the window.
func setPassthrough(hwnd windows.HWND, on bool) {
	ex, _, _ := procGetWindowLongPtrW.Call(uintptr(hwnd), uintptr(gwlExStyle))
	if on {
		ex |= uintptr(WS_EX_TRANSPARENT)
	} else {
		ex &^= uintptr(WS_EX_TRANSPARENT)
	}
	procSetWindowLongPtrW.Call(uintptr(hwnd), uintptr(gwlExStyle), ex)
}

// windowCenter returns the x coordinate to center a window of given width on screen.
func windowCenter(width int) int {
	return (sysMetric(SM_CXSCREEN) - width) / 2
}
