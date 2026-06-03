// Spike A: 验证 Desktop Duplication 可用 + 不透明置顶窗 WDA_EXCLUDEFROMCAPTURE
// 能否从抓图中消失。用完即删，不进主项目。
package main

import (
	"errors"
	"fmt"
	"image"
	"image/png"
	"os"
	"runtime"
	"syscall"
	"time"
	"unsafe"

	"github.com/kirides/go-d3d/d3d11"
	"github.com/kirides/go-d3d/outputduplication"
	"github.com/kirides/go-d3d/win"
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	pRegisterClassExW         = user32.NewProc("RegisterClassExW")
	pCreateWindowExW          = user32.NewProc("CreateWindowExW")
	pDefWindowProcW           = user32.NewProc("DefWindowProcW")
	pShowWindow               = user32.NewProc("ShowWindow")
	pPeekMessageW             = user32.NewProc("PeekMessageW")
	pTranslateMessage         = user32.NewProc("TranslateMessage")
	pDispatchMessageW         = user32.NewProc("DispatchMessageW")
	pSetWindowDisplayAffinity = user32.NewProc("SetWindowDisplayAffinity")
	pGetWindowRect            = user32.NewProc("GetWindowRect")
	pDestroyWindow            = user32.NewProc("DestroyWindow")
	pCreateSolidBrush         = gdi32.NewProc("CreateSolidBrush")
	pGetModuleHandleW         = kernel32.NewProc("GetModuleHandleW")
)

const (
	wsPopup            = 0x80000000
	wsVisible          = 0x10000000
	wsExTopmost        = 0x00000008
	wsExToolwindow     = 0x00000080
	wsExNoactivate     = 0x08000000
	wdaExcludeCapture  = 0x00000011
	swShowNoActivate   = 4
	pmRemove           = 0x0001
	colorMagenta       = 0x00FF00FF // COLORREF 0x00BBGGRR → R=255,G=0,B=255
)

type wndClassExW struct {
	cbSize        uint32
	style         uint32
	lpfnWndProc   uintptr
	cbClsExtra    int32
	cbWndExtra    int32
	hInstance     uintptr
	hIcon         uintptr
	hCursor       uintptr
	hbrBackground uintptr
	lpszMenuName  *uint16
	lpszClassName *uint16
	hIconSm       uintptr
}

type msgT struct {
	hwnd    uintptr
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	pt      struct{ x, y int32 }
}

type rectT struct{ left, top, right, bottom int32 }

func wndProc(hwnd, message, wp, lp uintptr) uintptr {
	r, _, _ := pDefWindowProcW.Call(hwnd, message, wp, lp)
	return r
}

func pump() {
	var m msgT
	for {
		r, _, _ := pPeekMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0, pmRemove)
		if r == 0 {
			break
		}
		pTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		pDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}
}

func main() {
	runtime.LockOSThread()

	// PerMonitorV2 DPI 感知，保证窗口坐标与抓图都在物理像素
	if win.IsValidDpiAwarenessContext(win.DpiAwarenessContextPerMonitorAwareV2) {
		win.SetThreadDpiAwarenessContext(win.DpiAwarenessContextPerMonitorAwareV2)
	}

	hInst, _, _ := pGetModuleHandleW.Call(0)
	brush, _, _ := pCreateSolidBrush.Call(colorMagenta)
	className := syscall.StringToUTF16Ptr("SpikeAClass")
	title := syscall.StringToUTF16Ptr("SpikeA")

	wc := wndClassExW{
		cbSize:        uint32(unsafe.Sizeof(wndClassExW{})),
		lpfnWndProc:   syscall.NewCallback(wndProc),
		hInstance:     hInst,
		hbrBackground: brush,
		lpszClassName: className,
	}
	atom, _, err := pRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	if atom == 0 {
		fmt.Printf("RegisterClassExW 失败: %v\n", err)
		os.Exit(2)
	}

	const wx, wy, ww, wh = 400, 300, 420, 180
	hwnd, _, err := pCreateWindowExW.Call(
		uintptr(wsExTopmost|wsExToolwindow|wsExNoactivate),
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(title)),
		uintptr(wsPopup|wsVisible),
		wx, wy, ww, wh,
		0, 0, hInst, 0,
	)
	if hwnd == 0 {
		fmt.Printf("CreateWindowExW 失败: %v\n", err)
		os.Exit(2)
	}

	// 命门：把窗口从捕获面排除
	ret, _, err := pSetWindowDisplayAffinity.Call(hwnd, wdaExcludeCapture)
	fmt.Printf("SetWindowDisplayAffinity(WDA_EXCLUDEFROMCAPTURE) 返回=%d (非0=成功), lastErr=%v\n", ret, err)
	wdaOK := ret != 0

	pShowWindow.Call(hwnd, swShowNoActivate)
	// 泵消息 + 等 DWM 合成稳定
	for i := 0; i < 40; i++ {
		pump()
		time.Sleep(20 * time.Millisecond)
	}

	var wr rectT
	pGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&wr)))
	fmt.Printf("窗口物理矩形: L=%d T=%d R=%d B=%d\n", wr.left, wr.top, wr.right, wr.bottom)

	// Desktop Duplication 抓全屏
	device, deviceCtx, derr := d3d11.NewD3D11Device()
	if derr != nil {
		fmt.Printf("NewD3D11Device 失败: %v\n", derr)
		os.Exit(2)
	}
	defer device.Release()
	defer deviceCtx.Release()

	ddup, derr := outputduplication.NewIDXGIOutputDuplication(device, deviceCtx, 0)
	if derr != nil {
		fmt.Printf("NewIDXGIOutputDuplication 失败: %v\n", derr)
		os.Exit(2)
	}
	defer ddup.Release()

	bounds, _ := ddup.GetBounds()
	img := image.NewRGBA(bounds)

	var got bool
	for i := 0; i < 120; i++ {
		pump()
		gerr := ddup.GetImage(img, 200)
		if gerr == nil {
			got = true
			break
		}
		if !errors.Is(gerr, outputduplication.ErrNoImageYet) {
			fmt.Printf("GetImage 错误: %v\n", gerr)
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !got {
		fmt.Println("结论: 无法获取桌面帧 → Desktop Duplication 在本机可能受限（不确定，需排查）")
		pDestroyWindow.Call(hwnd)
		os.Exit(3)
	}

	// 扫描窗口矩形区域内的品红像素
	clamp := func(v, lo, hi int) int {
		if v < lo {
			return lo
		}
		if v > hi {
			return hi
		}
		return v
	}
	x0 := clamp(int(wr.left)-bounds.Min.X, 0, bounds.Dx())
	x1 := clamp(int(wr.right)-bounds.Min.X, 0, bounds.Dx())
	y0 := clamp(int(wr.top)-bounds.Min.Y, 0, bounds.Dy())
	y1 := clamp(int(wr.bottom)-bounds.Min.Y, 0, bounds.Dy())

	magenta := 0
	total := 0
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			idx := img.PixOffset(bounds.Min.X+x, bounds.Min.Y+y)
			r := img.Pix[idx]
			g := img.Pix[idx+1]
			b := img.Pix[idx+2]
			total++
			if r > 200 && g < 60 && b > 200 {
				magenta++
			}
		}
	}

	// 存全屏 PNG 供肉眼核对
	if f, ferr := os.Create("spikeA_capture.png"); ferr == nil {
		png.Encode(f, img)
		f.Close()
	}

	fmt.Printf("窗口区域像素=%d, 其中品红=%d\n", total, magenta)
	fmt.Println("------------------------------------------------------------")
	switch {
	case !wdaOK:
		fmt.Println("结论: ❌ SetWindowDisplayAffinity 调用本身失败 → WDA 在本机不可用，转 Plan B")
	case magenta == 0:
		fmt.Println("结论: ✅ 通过 — 不透明窗已从抓图中完全排除（肉眼可见、抓取消失）。命门基础成立，进 Spike B 测透明窗。")
	case magenta < total/100:
		fmt.Println("结论: ⚠️ 几乎排除（边缘残留少量品红），基本成立，可进 Spike B 复核。")
	default:
		fmt.Println("结论: ❌ 窗口仍出现在抓图中 → WDA 未排除，转 Plan B")
	}
	fmt.Println("已存 spikeA_capture.png（你可以打开看：屏幕上有品红块，但图里该位置应是桌面）")

	// 让窗口停留几秒，便于肉眼确认屏幕上确实可见
	for i := 0; i < 150; i++ {
		pump()
		time.Sleep(20 * time.Millisecond)
	}
	pDestroyWindow.Call(hwnd)
}
