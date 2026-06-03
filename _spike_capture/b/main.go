// Spike B: 验证 DirectComposition 透明置顶窗（非 WS_EX_LAYERED）+ WDA_EXCLUDEFROMCAPTURE
// 能否同时做到「屏幕上透明可见」且「Desktop Duplication 抓不到」。用完即删。
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
	"golang.org/x/sys/windows"
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	d3d11dll = syscall.NewLazyDLL("d3d11.dll")
	dcompdll = syscall.NewLazyDLL("dcomp.dll")

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
	pGetModuleHandleW         = kernel32.NewProc("GetModuleHandleW")
	pD3D11CreateDevice        = d3d11dll.NewProc("D3D11CreateDevice")
	pDCompositionCreateDevice = dcompdll.NewProc("DCompositionCreateDevice")
)

const (
	wsPopup                 = 0x80000000
	wsVisible               = 0x10000000
	wsExTopmost             = 0x00000008
	wsExToolwindow          = 0x00000080
	wsExNoactivate          = 0x08000000
	wsExNoredirectionbitmap = 0x00200000
	wdaExcludeCapture       = 0x00000011
	swShowNoActivate        = 4
	pmRemove                = 0x0001
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

type dxgiSampleDesc struct{ Count, Quality uint32 }
type dxgiSwapChainDesc1 struct {
	Width       uint32
	Height      uint32
	Format      uint32
	Stereo      int32
	SampleDesc  dxgiSampleDesc
	BufferUsage uint32
	BufferCount uint32
	Scaling     uint32
	SwapEffect  uint32
	AlphaMode   uint32
	Flags       uint32
}

// IID
var (
	iidIDXGIDevice = windows.GUID{Data1: 0x54ec77fa, Data2: 0x1377, Data3: 0x44e6,
		Data4: [8]byte{0x8c, 0x32, 0x88, 0xfd, 0x5f, 0x44, 0xc8, 0x4c}}
	iidIDXGIFactory2 = windows.GUID{Data1: 0x50c83a1c, Data2: 0xe072, Data3: 0x4c48,
		Data4: [8]byte{0x87, 0xb0, 0x36, 0x30, 0xfa, 0x36, 0xa6, 0xd0}}
	iidID3D11Texture2D = windows.GUID{Data1: 0x6f15aaf2, Data2: 0xd208, Data3: 0x4e89,
		Data4: [8]byte{0x9a, 0xb4, 0x48, 0x95, 0x35, 0xd3, 0x4f, 0x9c}}
	iidIDCompositionDevice = windows.GUID{Data1: 0xC37EA93A, Data2: 0xE7AA, Data3: 0x450D,
		Data4: [8]byte{0xB1, 0x6F, 0x97, 0x46, 0xCB, 0x04, 0x07, 0xF3}}
)

const ptrSize = unsafe.Sizeof(uintptr(0))

// comCall: 通过 vtable 调 COM 方法。idx 为方法序号（IUnknown 占 0/1/2）。
func comCall(this, idx uintptr, args ...uintptr) uintptr {
	vtbl := *(*uintptr)(unsafe.Pointer(this))
	fn := *(*uintptr)(unsafe.Pointer(vtbl + idx*ptrSize))
	all := append([]uintptr{this}, args...)
	ret, _, _ := syscall.SyscallN(fn, all...)
	return ret
}

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
func fail(msg string, args ...any) {
	fmt.Printf("❌ "+msg+"\n", args...)
	os.Exit(2)
}

func main() {
	runtime.LockOSThread()
	if win.IsValidDpiAwarenessContext(win.DpiAwarenessContextPerMonitorAwareV2) {
		win.SetThreadDpiAwarenessContext(win.DpiAwarenessContextPerMonitorAwareV2)
	}

	hInst, _, _ := pGetModuleHandleW.Call(0)
	className := syscall.StringToUTF16Ptr("SpikeBClass")
	title := syscall.StringToUTF16Ptr("SpikeB")
	wc := wndClassExW{
		cbSize:        uint32(unsafe.Sizeof(wndClassExW{})),
		lpfnWndProc:   syscall.NewCallback(wndProc),
		hInstance:     hInst,
		lpszClassName: className, // 不设背景刷，内容由 DComp 提供
	}
	if atom, _, err := pRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc))); atom == 0 {
		fail("RegisterClassExW: %v", err)
	}

	const wx, wy, ww, wh = 400, 300, 420, 180
	hwnd, _, err := pCreateWindowExW.Call(
		uintptr(wsExNoredirectionbitmap|wsExTopmost|wsExToolwindow|wsExNoactivate),
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(title)),
		uintptr(wsPopup|wsVisible),
		wx, wy, ww, wh, 0, 0, hInst, 0,
	)
	if hwnd == 0 {
		fail("CreateWindowExW: %v", err)
	}

	// 自建带 BGRA_SUPPORT 的 D3D11 device（专供 DComp swapchain）
	var dev, devCtx uintptr
	hr, _, _ := pD3D11CreateDevice.Call(
		0, 1 /*HARDWARE*/, 0, 0x20, /*BGRA_SUPPORT*/
		0, 0, 7 /*SDK*/, uintptr(unsafe.Pointer(&dev)), 0, uintptr(unsafe.Pointer(&devCtx)),
	)
	if int32(hr) < 0 || dev == 0 {
		fail("D3D11CreateDevice hr=0x%X", uint32(hr))
	}

	// QI device → IDXGIDevice
	var dxgiDevice uintptr
	if r := comCall(dev, 0, uintptr(unsafe.Pointer(&iidIDXGIDevice)), uintptr(unsafe.Pointer(&dxgiDevice))); int32(r) < 0 {
		fail("QI IDXGIDevice hr=0x%X", uint32(r))
	}

	// DCompositionCreateDevice
	var dcompDevice uintptr
	if r, _, _ := pDCompositionCreateDevice.Call(dxgiDevice,
		uintptr(unsafe.Pointer(&iidIDCompositionDevice)), uintptr(unsafe.Pointer(&dcompDevice))); int32(r) < 0 {
		fail("DCompositionCreateDevice hr=0x%X", uint32(r))
	}

	// factory: dxgiDevice.GetAdapter(7) → adapter.GetParent(6, IDXGIFactory2)
	var adapter, factory uintptr
	if r := comCall(dxgiDevice, 7, uintptr(unsafe.Pointer(&adapter))); int32(r) < 0 {
		fail("GetAdapter hr=0x%X", uint32(r))
	}
	if r := comCall(adapter, 6, uintptr(unsafe.Pointer(&iidIDXGIFactory2)), uintptr(unsafe.Pointer(&factory))); int32(r) < 0 {
		fail("GetParent(Factory2) hr=0x%X", uint32(r))
	}

	// CreateSwapChainForComposition (factory idx24)
	desc := dxgiSwapChainDesc1{
		Width: ww, Height: wh, Format: 87, /*B8G8R8A8_UNORM*/
		SampleDesc:  dxgiSampleDesc{Count: 1},
		BufferUsage: 0x20 /*RENDER_TARGET_OUTPUT*/, BufferCount: 2,
		Scaling: 0, SwapEffect: 3 /*FLIP_SEQUENTIAL*/, AlphaMode: 1, /*PREMULTIPLIED*/
	}
	var swapchain uintptr
	if r := comCall(factory, 24, dev, uintptr(unsafe.Pointer(&desc)), 0, uintptr(unsafe.Pointer(&swapchain))); int32(r) < 0 {
		fail("CreateSwapChainForComposition hr=0x%X", uint32(r))
	}

	// GetBuffer(0) → texture (swapchain idx9)
	var tex uintptr
	if r := comCall(swapchain, 9, 0, uintptr(unsafe.Pointer(&iidID3D11Texture2D)), uintptr(unsafe.Pointer(&tex))); int32(r) < 0 {
		fail("GetBuffer hr=0x%X", uint32(r))
	}
	// CreateRenderTargetView (device idx9)
	var rtv uintptr
	if r := comCall(dev, 9, tex, 0, uintptr(unsafe.Pointer(&rtv))); int32(r) < 0 {
		fail("CreateRenderTargetView hr=0x%X", uint32(r))
	}
	// ClearRenderTargetView (context idx50) — 半透明品红，premultiplied
	clearColor := [4]float32{0.7, 0.0, 0.7, 0.7}
	comCall(devCtx, 50, rtv, uintptr(unsafe.Pointer(&clearColor)))
	// Present (swapchain idx8)
	comCall(swapchain, 8, 0, 0)

	// DComp 装配
	var target, visual uintptr
	if r := comCall(dcompDevice, 6, hwnd, 1 /*topmost*/, uintptr(unsafe.Pointer(&target))); int32(r) < 0 {
		fail("CreateTargetForHwnd hr=0x%X", uint32(r))
	}
	if r := comCall(dcompDevice, 7, uintptr(unsafe.Pointer(&visual))); int32(r) < 0 {
		fail("CreateVisual hr=0x%X", uint32(r))
	}
	comCall(visual, 15, swapchain) // SetContent
	comCall(target, 3, visual)     // SetRoot
	comCall(dcompDevice, 3)        // Commit

	// 命门
	ret, _, lerr := pSetWindowDisplayAffinity.Call(hwnd, wdaExcludeCapture)
	fmt.Printf("SetWindowDisplayAffinity 返回=%d (非0=成功), lastErr=%v\n", ret, lerr)
	wdaOK := ret != 0

	pShowWindow.Call(hwnd, swShowNoActivate)
	for i := 0; i < 50; i++ {
		pump()
		time.Sleep(20 * time.Millisecond)
	}

	var wr rectT
	pGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&wr)))
	fmt.Printf("窗口物理矩形: L=%d T=%d R=%d B=%d\n", wr.left, wr.top, wr.right, wr.bottom)

	// 抓屏（go-d3d 独立 device）
	gdev, gctx, derr := d3d11.NewD3D11Device()
	if derr != nil {
		fail("go-d3d NewD3D11Device: %v", derr)
	}
	defer gdev.Release()
	defer gctx.Release()
	ddup, derr := outputduplication.NewIDXGIOutputDuplication(gdev, gctx, 0)
	if derr != nil {
		fail("NewIDXGIOutputDuplication: %v", derr)
	}
	defer ddup.Release()
	bounds, _ := ddup.GetBounds()
	img := image.NewRGBA(bounds)

	var got bool
	for i := 0; i < 120; i++ {
		pump()
		if gerr := ddup.GetImage(img, 200); gerr == nil {
			got = true
			break
		} else if !errors.Is(gerr, outputduplication.ErrNoImageYet) {
			fmt.Printf("GetImage: %v\n", gerr)
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !got {
		fail("无法获取桌面帧")
	}

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
	magenta, total := 0, 0
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			idx := img.PixOffset(bounds.Min.X+x, bounds.Min.Y+y)
			r, g, b := img.Pix[idx], img.Pix[idx+1], img.Pix[idx+2]
			total++
			if r > 150 && g < 80 && b > 150 {
				magenta++
			}
		}
	}
	if f, ferr := os.Create("spikeB_capture.png"); ferr == nil {
		png.Encode(f, img)
		f.Close()
	}

	fmt.Printf("窗口区域像素=%d, 其中品红=%d\n", total, magenta)
	fmt.Println("------------------------------------------------------------")
	switch {
	case !wdaOK:
		fmt.Println("结论: ❌ WDA 调用失败 → 转 Plan B")
	case magenta == 0:
		fmt.Println("结论: ✅ 通过 — DComp 透明置顶窗已从抓图中完全排除（屏幕可见、抓取消失）。命门 100% 坐实，方案2 可行。")
	case magenta < total/50:
		fmt.Println("结论: ⚠️ 几乎排除（边缘微量残留），基本成立。")
	default:
		fmt.Println("结论: ❌ 透明窗仍出现在抓图中 → WDA 对 DComp 窗无效，转 Plan B")
	}
	fmt.Println("已存 spikeB_capture.png（屏幕上有半透明品红块，图里该位置应是纯桌面）")

	for i := 0; i < 150; i++ {
		pump()
		time.Sleep(20 * time.Millisecond)
	}
	pDestroyWindow.Call(hwnd)
}
