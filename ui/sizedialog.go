package ui

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"

	"claude-traffic-light/config"
)

// 调整大小滑块窗：屏幕居中的小窗，一条 100%~2000% 的无极滑块 +「重置 100%」+「关闭」。
// 拖滑块实时缩放挂件（保持中心不动），松手存盘；窗口靠按钮/标题栏 × 手动关闭。
// 与挂件同 UI 线程、共用消息循环（modeless），故可直接 SetWindowPos 挂件 hwnd。

const (
	dlgW = 340
	dlgH = 132

	idTrackbar = 3001
	idReset    = 3003
	idClose    = 3004
	idLabel    = 3005

	scaleMinPct = 100
	scaleMaxPct = 2000
)

var sizeDlgClassRegistered bool

func loword(v uintptr) int      { return int(uint16(v)) }
func makeLong(lo, hi int) int32 { return int32(uint16(lo)) | int32(uint16(hi))<<16 }

// openSizeDialog 打开（或前置已开的）调整大小滑块窗。
func (w *Window) openSizeDialog() {
	if w.sizeDlg != 0 {
		procSetForegroundWindow.Call(uintptr(w.sizeDlg))
		return
	}

	icc := initCommonControlsEx{DwSize: uint32(unsafe.Sizeof(initCommonControlsEx{})), DwICC: iccBarClasses}
	procInitCommonControlsEx.Call(uintptr(unsafe.Pointer(&icc)))

	className := u16("ClaudeSizeDlg")
	if !sizeDlgClassRegistered {
		arrow, _, _ := procLoadCursorW.Call(0, idcArrow)
		wc := wndClassExW{
			cbSize:        uint32(unsafe.Sizeof(wndClassExW{})),
			lpfnWndProc:   syscall.NewCallback(sizeDlgProc),
			hInstance:     w.hInst,
			hCursor:       windows.Handle(arrow),
			hbrBackground: windows.Handle(colorBtnface + 1),
			lpszClassName: className,
		}
		procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
		sizeDlgClassRegistered = true
	}

	x, y := screenCenter(dlgW, dlgH)
	hwnd, _, _ := procCreateWindowExW.Call(
		wsExToolwindow|wsExTopmost,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(u16("调整大小"))),
		wsPopup|wsCaption|wsSysmenu,
		uintptr(x), uintptr(y), dlgW, dlgH,
		0, 0, uintptr(w.hInst), 0,
	)
	w.sizeDlg = windows.HWND(hwnd)

	font, _, _ := procGetStockObject.Call(defaultGuiFont)
	curPct := int(w.cfg.Scale*100 + 0.5)

	// 百分比文字
	lbl, _, _ := procCreateWindowExW.Call(0,
		uintptr(unsafe.Pointer(u16("STATIC"))),
		uintptr(unsafe.Pointer(u16(labelText(curPct)))),
		wsChild|wsVisible|ssCenter,
		12, 12, dlgW-24, 22, hwnd, idLabel, uintptr(w.hInst), 0)
	procSendMessageW.Call(lbl, wmSetFont, font, 1)

	// 无极滑块 100~2000
	tb, _, _ := procCreateWindowExW.Call(0,
		uintptr(unsafe.Pointer(u16("msctls_trackbar32"))),
		0,
		wsChild|wsVisible|tbsHorz|tbsNoticks,
		12, 40, dlgW-24, 32, hwnd, idTrackbar, uintptr(w.hInst), 0)
	procSendMessageW.Call(tb, tbmSetRange, 1, uintptr(uint32(makeLong(scaleMinPct, scaleMaxPct))))
	procSendMessageW.Call(tb, tbmSetPageSize, 0, 100)
	procSendMessageW.Call(tb, tbmSetPos, 1, uintptr(curPct))

	// 重置 100% / 关闭
	btnR, _, _ := procCreateWindowExW.Call(0,
		uintptr(unsafe.Pointer(u16("BUTTON"))),
		uintptr(unsafe.Pointer(u16("重置 100%"))),
		wsChild|wsVisible|bsPushbutton,
		12, 86, 150, 30, hwnd, idReset, uintptr(w.hInst), 0)
	procSendMessageW.Call(btnR, wmSetFont, font, 1)

	btnC, _, _ := procCreateWindowExW.Call(0,
		uintptr(unsafe.Pointer(u16("BUTTON"))),
		uintptr(unsafe.Pointer(u16("关闭"))),
		wsChild|wsVisible|bsPushbutton,
		dlgW-12-150, 86, 150, 30, hwnd, idClose, uintptr(w.hInst), 0)
	procSendMessageW.Call(btnC, wmSetFont, font, 1)

	procShowWindow.Call(hwnd, swShow)
}

// labelText 生成百分比标签文字。
func labelText(pct int) string { return fmt.Sprintf("缩放：%d%%", pct) }

// applyScalePct 把挂件缩放到 pct%（保持中心不动），更新 curW/curH 供渲染线程 resize。
func (w *Window) applyScalePct(pct int) {
	if pct < scaleMinPct {
		pct = scaleMinPct
	}
	w.cfg.Scale = float64(pct) / 100.0
	nw, nh := scaledWindow(w.cfg.Scale)
	var wr RECT
	procGetWindowRect.Call(uintptr(w.hwnd), uintptr(unsafe.Pointer(&wr)))
	cx := (wr.Left + wr.Right) / 2
	nx := cx - nw/2 // 水平中心不变
	ny := wr.Top    // 顶边固定：放大向下扩展，顶边不动
	procSetWindowPos.Call(uintptr(w.hwnd), 0, uintptr(int(nx)), uintptr(int(ny)),
		uintptr(int(nw)), uintptr(int(nh)), swpNoZOrder|swpNoActivate)
	w.curW.Store(nw)
	w.curH.Store(nh)
}

// saveScale 把当前缩放与位置写入 config.json。
func (w *Window) saveScale() {
	var wr RECT
	procGetWindowRect.Call(uintptr(w.hwnd), uintptr(unsafe.Pointer(&wr)))
	w.cfg.X, w.cfg.Y = int(wr.Left), int(wr.Top)
	config.Save(w.cfgPath, w.cfg)
}

// sizeDlgProc 处理滑块窗消息。
func sizeDlgProc(hwnd, message, wParam, lParam uintptr) uintptr {
	switch message {
	case wmHScroll:
		// lParam = trackbar hwnd；读位置实时应用，非连续拖动时落盘
		pos, _, _ := procSendMessageW.Call(lParam, tbmGetPos, 0, 0)
		theWindow.applyScalePct(int(pos))
		setDlgLabel(hwnd, int(pos))
		if loword(wParam) != tbThumbTrack {
			theWindow.saveScale()
		}
		return 0

	case wmCommand:
		switch loword(wParam) {
		case idReset:
			theWindow.applyScalePct(100)
			theWindow.saveScale()
			tb, _, _ := procGetDlgItem.Call(hwnd, idTrackbar)
			procSendMessageW.Call(tb, tbmSetPos, 1, 100)
			setDlgLabel(hwnd, 100)
		case idClose:
			procDestroyWindow.Call(hwnd)
		}
		return 0

	case wmCtlColorStatic:
		// 让静态文字背景与对话框灰底一致
		procSetBkMode.Call(wParam, bkTransparent)
		br, _, _ := procGetSysColorBrush.Call(colorBtnface)
		return br

	case wmClose:
		procDestroyWindow.Call(hwnd)
		return 0

	case wmDestroy:
		theWindow.sizeDlg = 0
		return 0
	}
	r, _, _ := procDefWindowProcW.Call(hwnd, message, wParam, lParam)
	return r
}

// setDlgLabel 更新百分比文字。
func setDlgLabel(dlg uintptr, pct int) {
	lbl, _, _ := procGetDlgItem.Call(dlg, idLabel)
	procSetWindowTextW.Call(lbl, uintptr(unsafe.Pointer(u16(labelText(pct)))))
}
