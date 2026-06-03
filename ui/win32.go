package ui

import (
	"golang.org/x/sys/windows"
)

var (
	user32   = windows.NewLazySystemDLL("user32.dll")
	shell32  = windows.NewLazySystemDLL("shell32.dll")
	dwmapi   = windows.NewLazySystemDLL("dwmapi.dll")
	kernel32 = windows.NewLazySystemDLL("kernel32.dll")

	// 窗口创建与消息循环
	procRegisterClassExW = user32.NewProc("RegisterClassExW")
	procCreateWindowExW  = user32.NewProc("CreateWindowExW")
	procDefWindowProcW   = user32.NewProc("DefWindowProcW")
	procDestroyWindow    = user32.NewProc("DestroyWindow")
	procShowWindow       = user32.NewProc("ShowWindow")
	procGetMessageW      = user32.NewProc("GetMessageW")
	procTranslateMessage = user32.NewProc("TranslateMessage")
	procDispatchMessageW = user32.NewProc("DispatchMessageW")
	procPostMessageW     = user32.NewProc("PostMessageW")
	procPostQuitMessage  = user32.NewProc("PostQuitMessage")
	procSetTimer         = user32.NewProc("SetTimer")

	// 样式 / 定位 / DPI
	procSetWindowLongPtrW         = user32.NewProc("SetWindowLongPtrW")
	procGetWindowLongPtrW         = user32.NewProc("GetWindowLongPtrW")
	procSetWindowPos              = user32.NewProc("SetWindowPos")
	procGetWindowRect             = user32.NewProc("GetWindowRect")
	procGetSystemMetrics          = user32.NewProc("GetSystemMetrics")
	procSetWindowDisplayAffinity  = user32.NewProc("SetWindowDisplayAffinity")
	procSetThreadDpiAwarenessCtx  = user32.NewProc("SetThreadDpiAwarenessContext")

	// 菜单 / 托盘
	procCreatePopupMenu     = user32.NewProc("CreatePopupMenu")
	procAppendMenuW         = user32.NewProc("AppendMenuW")
	procTrackPopupMenu      = user32.NewProc("TrackPopupMenu")
	procDestroyMenu         = user32.NewProc("DestroyMenu")
	procSetForegroundWindow = user32.NewProc("SetForegroundWindow")
	procGetCursorPos        = user32.NewProc("GetCursorPos")
	procLoadIconW           = user32.NewProc("LoadIconW")
	procShellNotifyIconW    = shell32.NewProc("Shell_NotifyIconW")

	procDwmSetWindowAttribute = dwmapi.NewProc("DwmSetWindowAttribute")
	procGetModuleHandleW      = kernel32.NewProc("GetModuleHandleW")
)

// GWL_STYLE / GWL_EXSTYLE 为负，声明为 typed var 避免 uintptr 溢出。
var gwlExStyle = -20

// DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2 == (HANDLE)-4
var dpiPerMonitorV2 = ^uintptr(3)

const (
	wsPopup   = 0x80000000
	wsVisible = 0x10000000

	wsExTopmost             = 0x00000008
	wsExToolwindow          = 0x00000080
	wsExNoactivate          = 0x08000000
	wsExTransparent         = 0x00000020
	wsExNoredirectionbitmap = 0x00200000

	wdaExcludeFromCapture = 0x00000011

	// 窗口消息
	wmDestroy     = 0x0002
	wmClose       = 0x0010
	wmCommand     = 0x0111
	wmNcHitTest   = 0x0084
	wmRButtonUp   = 0x0205
	wmLButtonUp   = 0x0202
	wmNcRButtonUp = 0x00A5
	wmLButtonDblclk = 0x0203
	wmTray        = 0x0400 + 1 // WM_APP-ish 自定义托盘回调
	wmTimer       = 0x0113
	htCaption     = 2
	htTransparent = ^uintptr(0) // HTTRANSPARENT = -1，鼠标穿透到下层窗口

	swHide           = 0
	swShowNoActivate = 4

	hwndTopmost      = ^uintptr(0) // (HWND)-1
	swpNoMove        = 0x0002
	swpNoSize        = 0x0001
	swpNoZOrder      = 0x0004
	swpNoActivate    = 0x0010
	swpFrameChanged  = 0x0020
	swpShowWindow    = 0x0040

	smCXScreen = 0
	smCYScreen = 1

	csVRedraw = 0x0001
	csHRedraw = 0x0002

	mfString    = 0x0000
	mfChecked   = 0x0008
	mfSeparator = 0x0800
	tpmReturnCmd   = 0x0100
	tpmRightAlign  = 0x0008
	tpmBottomAlign = 0x0020

	nimAdd     = 0
	nimDelete  = 2
	nifIcon    = 0x02
	nifTip     = 0x04
	nifMessage = 0x01

	idiApplication = 32512

	menuShowHide   = 1001
	menuPassthrough = 1002
	menuExit        = 1003
)

type POINT struct{ X, Y int32 }
type RECT struct{ Left, Top, Right, Bottom int32 }

type wndClassExW struct {
	cbSize        uint32
	style         uint32
	lpfnWndProc   uintptr
	cbClsExtra    int32
	cbWndExtra    int32
	hInstance     windows.Handle
	hIcon         windows.Handle
	hCursor       windows.Handle
	hbrBackground windows.Handle
	lpszMenuName  *uint16
	lpszClassName *uint16
	hIconSm       windows.Handle
}

type MSG struct {
	HWnd    windows.HWND
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      POINT
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

func u16(s string) *uint16 { p, _ := windows.UTF16PtrFromString(s); return p }
func sysMetric(n int) int  { r, _, _ := procGetSystemMetrics.Call(uintptr(n)); return int(r) }

// setThreadDPIAware 让当前线程 PerMonitorV2 感知，保证窗口坐标与桌面纹理同在物理像素。
func setThreadDPIAware() { procSetThreadDpiAwarenessCtx.Call(dpiPerMonitorV2) }

// setPassthrough 启用/关闭点击穿透（动态增删 WS_EX_TRANSPARENT）。
func setPassthrough(hwnd windows.HWND, on bool) {
	ex, _, _ := procGetWindowLongPtrW.Call(uintptr(hwnd), uintptr(gwlExStyle))
	if on {
		ex |= wsExTransparent
	} else {
		ex &^= wsExTransparent
	}
	procSetWindowLongPtrW.Call(uintptr(hwnd), uintptr(gwlExStyle), ex)
}

// screenCenter 返回让 width×height 的窗口居中所需的左上角坐标。
func screenCenter(width, height int) (x, y int) {
	return (sysMetric(smCXScreen) - width) / 2, (sysMetric(smCYScreen) - height) / 2
}
