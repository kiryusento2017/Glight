package ui

import (
	"golang.org/x/sys/windows"
)

var (
	user32   = windows.NewLazySystemDLL("user32.dll")
	shell32  = windows.NewLazySystemDLL("shell32.dll")
	dwmapi   = windows.NewLazySystemDLL("dwmapi.dll")
	kernel32 = windows.NewLazySystemDLL("kernel32.dll")
	comctl32 = windows.NewLazySystemDLL("comctl32.dll")
	gdi32    = windows.NewLazySystemDLL("gdi32.dll")

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
	procSetCapture     = user32.NewProc("SetCapture")
	procReleaseCapture = user32.NewProc("ReleaseCapture")
	procSetCursor      = user32.NewProc("SetCursor")
	procLoadCursorW    = user32.NewProc("LoadCursorW")

	procSetWindowPos              = user32.NewProc("SetWindowPos")
	procGetWindowRect             = user32.NewProc("GetWindowRect")
	procGetSystemMetrics          = user32.NewProc("GetSystemMetrics")
	procSetWindowDisplayAffinity  = user32.NewProc("SetWindowDisplayAffinity")
	procSetThreadDpiAwarenessCtx  = user32.NewProc("SetThreadDpiAwarenessContext")
	procSetProcessDpiAwarenessCtx = user32.NewProc("SetProcessDpiAwarenessContext")

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

	// 调整大小滑块窗：trackbar + button + static
	procSendMessageW         = user32.NewProc("SendMessageW")
	procGetDlgItem           = user32.NewProc("GetDlgItem")
	procSetWindowTextW       = user32.NewProc("SetWindowTextW")
	procGetSysColorBrush     = user32.NewProc("GetSysColorBrush")
	procInitCommonControlsEx = comctl32.NewProc("InitCommonControlsEx")
	procGetStockObject       = gdi32.NewProc("GetStockObject")
	procSetBkMode            = gdi32.NewProc("SetBkMode")

	// 运行时图标嵌入（纯内存，不用外部文件）
	procCreateIconFromResourceEx        = user32.NewProc("CreateIconFromResourceEx")
	procLookupIconIdFromDirectoryEx     = user32.NewProc("LookupIconIdFromDirectoryEx")
)

// INITCOMMONCONTROLSEX：注册 comctl32 控件类（trackbar 在 ICC_BAR_CLASSES）。
type initCommonControlsEx struct{ DwSize, DwICC uint32 }

// DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2 == (HANDLE)-4
var dpiPerMonitorV2 = ^uintptr(3)

const (
	wsPopup   = 0x80000000
	wsVisible = 0x10000000

	wsExTopmost             = 0x00000008
	wsExToolwindow          = 0x00000080
	wsExNoactivate          = 0x08000000
	wsExNoredirectionbitmap = 0x00200000

	wdaExcludeFromCapture = 0x00000011

	// 窗口消息
	wmDestroy     = 0x0002
	wmClose       = 0x0010
	wmCommand     = 0x0111
	wmMouseMove   = 0x0200
	wmLButtonDown = 0x0201
	wmLButtonUp   = 0x0202
	wmSetCursor   = 0x0020
	wmRButtonUp   = 0x0205
	wmNcHitTest   = 0x0084
	wmNcRButtonUp = 0x00A5
	wmLButtonDblclk = 0x0203
	wmTray        = 0x0400 + 1 // WM_APP-ish 自定义托盘回调
	wmTimer       = 0x0113
	htCaption = 2
	htClient  = 1 // HTCLIENT：客户区，不可拖动（固定位置用）
	idcArrow  = 32512

	swHide           = 0
	swShow           = 5
	swShowNoActivate = 4

	// 滑块窗：窗口/控件样式、消息、trackbar 常量
	wsChild     = 0x40000000
	wsCaption   = 0x00C00000
	wsSysmenu   = 0x00080000
	ssCenter    = 0x00000001
	bsPushbutton = 0x00000000
	tbsHorz     = 0x00000000
	tbsNoticks  = 0x00000010

	wmSetIcon       = 0x0080
	iconBig         = 1
	iconSmall       = 0

	wmHScroll       = 0x0114
	wmSetFont       = 0x0030
	wmCtlColorStatic = 0x0138

	tbmGetPos      = 0x0400 // WM_USER
	tbmSetPos      = 0x0405 // WM_USER+5
	tbmSetRange    = 0x0406 // WM_USER+6
	tbmSetPageSize = 0x0415 // WM_USER+21
	tbThumbTrack   = 5       // WM_HSCROLL 通知码：拖动中（连续）
	tbEndTrack     = 8       // WM_HSCROLL 通知码：结束拖动

	iccBarClasses  = 0x00000004
	defaultGuiFont = 17
	colorBtnface   = 15
	bkTransparent  = 1

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

	menuShowHide = 1001
	menuLock     = 1002
	menuStartup  = 1006
	menuReset    = 1004
	menuResize   = 1005
	menuExit     = 1003
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

// SetProcessDPIAware 让整个进程 PerMonitorV2 DPI 感知（须在创建任何窗口前调用）。
// 用代码替代 manifest，无需 rsrc/windres 外部工具，契合单 exe 零依赖。
func SetProcessDPIAware() { procSetProcessDpiAwarenessCtx.Call(dpiPerMonitorV2) }

// screenCenter 返回让 width×height 的窗口居中所需的左上角坐标。
func screenCenter(width, height int) (x, y int) {
	return (sysMetric(smCXScreen) - width) / 2, (sysMetric(smCYScreen) - height) / 2
}
