package ui

import (
	"runtime"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"

	d3d11 "github.com/kirides/go-d3d/d3d11"
	"golang.org/x/sys/windows"

	"claude-traffic-light/config"
	"claude-traffic-light/state"
)

const (
	winW = 250
	winH = 88
)

// theWindow 是当前唯一的挂件窗口（单实例由 main.go 的互斥保证）。
// wndProc 是包级回调，通过它访问实例。
var theWindow *Window

// Window 管理浮动挂件窗口：主线程跑消息循环，渲染在独立 goroutine，
// 拖动的系统模态循环挡不住渲染。
type Window struct {
	hwnd     windows.HWND
	hInst    windows.Handle
	cfg      config.Config
	cfgPath  string
	curState atomic.Int32 // state.State
	closing  atomic.Bool
}

// New 创建挂件窗口并启动渲染线程。必须在将要跑消息循环的线程上调用。
func New(cfgPath string, cfg config.Config) *Window {
	runtime.LockOSThread()
	setThreadDPIAware()

	w := &Window{cfg: cfg, cfgPath: cfgPath}
	w.curState.Store(int32(state.Grey))
	theWindow = w

	var hInst windows.Handle
	windows.GetModuleHandleEx(0, nil, &hInst)
	w.hInst = hInst

	className := u16("ClaudeTrafficLightWnd")
	wc := wndClassExW{
		cbSize:        uint32(unsafe.Sizeof(wndClassExW{})),
		style:         csHRedraw | csVRedraw,
		lpfnWndProc:   syscall.NewCallback(wndProc),
		hInstance:     hInst,
		lpszClassName: className,
	}
	procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))

	// 位置：水平居中、贴屏幕顶部（Y=16），或使用保存的位置
	x, _ := screenCenter(winW, winH)
	y := 16
	if cfg.X >= 0 {
		x, y = cfg.X, cfg.Y
	}
	w.cfg.X, w.cfg.Y = x, y

	hwnd, _, _ := procCreateWindowExW.Call(
		wsExNoredirectionbitmap|wsExTopmost|wsExToolwindow|wsExNoactivate,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(u16("Claude Traffic Light"))),
		wsPopup,
		uintptr(x), uintptr(y), winW, winH,
		0, 0, uintptr(hInst), 0,
	)
	w.hwnd = windows.HWND(hwnd)

	// 命门：把窗口从屏幕捕获中排除，断开「自己折射自己」反馈
	procSetWindowDisplayAffinity.Call(hwnd, wdaExcludeFromCapture)

	if cfg.ClickThrough {
		setPassthrough(w.hwnd, true)
	}

	go w.renderThread()

	if cfg.Visible {
		// 一次 SetWindowPos 同时「升到 topmost + 显示」，避免先显示再置顶的中间帧。
		// （WS_EX_TOPMOST 对 NOACTIVATE 窗口创建时不一定真正升到 topmost z-order。）
		procSetWindowPos.Call(hwnd, hwndTopmost, 0, 0, 0, 0,
			swpNoMove|swpNoSize|swpNoActivate|swpShowWindow)
	}
	// 周期性重新提顶：DComp 内容由渲染线程异步呈现，单次提顶可能被时序打破；
	// 定时器在主线程（含拖动模态循环）周期恢复 topmost。
	procSetTimer.Call(hwnd, 1, 300, 0)

	w.addTrayIcon()

	return w
}

// SetState 线程安全地更新红绿灯状态（渲染线程读取）。
func (w *Window) SetState(s state.State) {
	w.curState.Store(int32(s))
}

// Run 跑主线程消息循环，阻塞至窗口销毁。
func (w *Window) Run() {
	defer w.removeTrayIcon()
	var m MSG
	for {
		r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}
	w.closing.Store(true)
}

// wndProc 处理窗口消息。整窗作为拖动区（HTCAPTION）。
func wndProc(hwnd, message, wParam, lParam uintptr) uintptr {
	switch message {
	case wmNcHitTest:
		if theWindow.cfg.ClickThrough {
			return htTransparent // 穿透模式：鼠标落到下层窗口
		}
		return htCaption
	case wmDestroy:
		procPostQuitMessage.Call(0)
		return 0
	case wmTimer:
		procSetWindowPos.Call(hwnd, hwndTopmost, 0, 0, 0, 0, swpNoMove|swpNoSize|swpNoActivate)
		return 0
	case wmTray:
		if lParam == wmRButtonUp || lParam == wmLButtonUp {
			theWindow.showContextMenu()
		}
		return 0
	case wmNcRButtonUp:
		theWindow.showContextMenu()
		return 0
	}
	r, _, _ := procDefWindowProcW.Call(hwnd, message, wParam, lParam)
	return r
}

// renderThread 拥有 D3D device 与渲染循环，独立于 UI 线程。
// Task 3a：先画半透明纯色，验证透明置顶窗与拖动实时性。
func (w *Window) renderThread() {
	runtime.LockOSThread()
	setThreadDPIAware()

	device, ctx, err := d3d11.NewD3D11Device()
	if err != nil {
		return // 降级：无 GPU/被限制时不渲染（Task 9 完善重试）
	}
	defer device.Release()
	defer ctx.Release()
	dev := uintptr(unsafe.Pointer(device))
	dctx := uintptr(unsafe.Pointer(ctx))

	dxgiDevice, factory, err := queryDXGIFactory(dev)
	if err != nil {
		return
	}
	swapchain, err := createCompositionSwapchain(factory, dev, winW, winH)
	if err != nil {
		return
	}
	rtv, err := backBufferRTV(dev, swapchain)
	if err != nil {
		return
	}
	if _, err := dcompAttach(dxgiDevice, uintptr(w.hwnd), swapchain); err != nil {
		return
	}

	// 半透明青蓝（premultiplied alpha），验证透明合成
	col := [4]float32{0.08, 0.22, 0.40, 0.55}
	first := true
	for !w.closing.Load() {
		comCall(dctx, vtCtxClearRTV, rtv, uintptr(unsafe.Pointer(&col)))
		comCall(swapchain, vtSwapPresent, 0, 0)
		if first {
			// 内容首帧呈现的此刻确保 topmost（DComp 异步呈现晚于 New 的提顶时机）
			procSetWindowPos.Call(uintptr(w.hwnd), hwndTopmost, 0, 0, 0, 0,
				swpNoMove|swpNoSize|swpNoActivate)
			first = false
		}
		time.Sleep(16 * time.Millisecond)
	}
}

// addTrayIcon 注册系统托盘图标，鼠标事件回调为 wmTray 消息。
func (w *Window) addTrayIcon() {
	var tip [128]uint16
	for i, c := range windows.StringToUTF16("Claude Traffic Light") {
		if i >= len(tip) {
			break
		}
		tip[i] = c
	}
	hIcon, _, _ := procLoadIconW.Call(0, idiApplication)
	nid := NOTIFYICONDATAW{
		CbSize:           uint32(unsafe.Sizeof(NOTIFYICONDATAW{})),
		HWnd:             w.hwnd,
		UID:              1,
		UFlags:           nifIcon | nifTip | nifMessage,
		UCallbackMessage: wmTray,
		HIcon:            windows.Handle(hIcon),
		SzTip:            tip,
	}
	procShellNotifyIconW.Call(nimAdd, uintptr(unsafe.Pointer(&nid)))
}

// removeTrayIcon 移除托盘图标。
func (w *Window) removeTrayIcon() {
	nid := NOTIFYICONDATAW{
		CbSize: uint32(unsafe.Sizeof(NOTIFYICONDATAW{})),
		HWnd:   w.hwnd,
		UID:    1,
	}
	procShellNotifyIconW.Call(nimDelete, uintptr(unsafe.Pointer(&nid)))
}

// showContextMenu 弹出右键菜单：显示/隐藏窗口、开启/关闭穿透、退出。
// 穿透开启后窗口本体不接收鼠标，此菜单经托盘图标触发。
func (w *Window) showContextMenu() {
	menu, _, _ := procCreatePopupMenu.Call()
	defer procDestroyMenu.Call(menu)

	visLabel := "隐藏窗口"
	if !w.cfg.Visible {
		visLabel = "显示窗口"
	}
	procAppendMenuW.Call(menu, mfString, menuShowHide, uintptr(unsafe.Pointer(u16(visLabel))))

	ptFlags := uintptr(mfString)
	ptLabel := "开启穿透"
	if w.cfg.ClickThrough {
		ptFlags |= mfChecked
		ptLabel = "关闭穿透"
	}
	procAppendMenuW.Call(menu, ptFlags, menuPassthrough, uintptr(unsafe.Pointer(u16(ptLabel))))

	procAppendMenuW.Call(menu, mfSeparator, 0, 0)
	procAppendMenuW.Call(menu, mfString, menuExit, uintptr(unsafe.Pointer(u16("退出"))))

	var pt POINT
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	procSetForegroundWindow.Call(uintptr(w.hwnd))

	cmd, _, _ := procTrackPopupMenu.Call(menu,
		tpmReturnCmd|tpmRightAlign|tpmBottomAlign,
		uintptr(pt.X), uintptr(pt.Y), 0, uintptr(w.hwnd), 0)

	switch cmd {
	case menuShowHide:
		w.cfg.Visible = !w.cfg.Visible
		if w.cfg.Visible {
			procSetWindowPos.Call(uintptr(w.hwnd), hwndTopmost, 0, 0, 0, 0,
				swpNoMove|swpNoSize|swpNoActivate|swpShowWindow)
		} else {
			procShowWindow.Call(uintptr(w.hwnd), swHide)
		}
		config.Save(w.cfgPath, w.cfg)
	case menuPassthrough:
		w.cfg.ClickThrough = !w.cfg.ClickThrough
		setPassthrough(w.hwnd, w.cfg.ClickThrough)
		config.Save(w.cfgPath, w.cfg)
	case menuExit:
		procDestroyWindow.Call(uintptr(w.hwnd))
	}
}
