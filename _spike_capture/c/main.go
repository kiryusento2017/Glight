// Spike C（多线程版）: 验证 GPU 链 + 拖动时实时更新。
// 主线程只跑窗口消息循环；渲染在独立 goroutine，拖动的模态循环挡不住它。
// 用完即删。
package main

import (
	"fmt"
	"os"
	"runtime"
	"syscall"
	"time"
	"unsafe"

	"github.com/kirides/go-d3d/d3d11"
	"github.com/kirides/go-d3d/dxgi"
	"github.com/kirides/go-d3d/win"
	"golang.org/x/sys/windows"
)

var (
	user32      = syscall.NewLazyDLL("user32.dll")
	kernel32    = syscall.NewLazyDLL("kernel32.dll")
	dcompdll    = syscall.NewLazyDLL("dcomp.dll")
	d3dcompiler = syscall.NewLazyDLL("d3dcompiler_47.dll")

	pRegisterClassExW         = user32.NewProc("RegisterClassExW")
	pCreateWindowExW          = user32.NewProc("CreateWindowExW")
	pDefWindowProcW           = user32.NewProc("DefWindowProcW")
	pShowWindow               = user32.NewProc("ShowWindow")
	pTranslateMessage         = user32.NewProc("TranslateMessage")
	pDispatchMessageW         = user32.NewProc("DispatchMessageW")
	pSetWindowDisplayAffinity = user32.NewProc("SetWindowDisplayAffinity")
	pGetWindowRect            = user32.NewProc("GetWindowRect")
	pGetMessageW              = user32.NewProc("GetMessageW")
	pPostMessageW             = user32.NewProc("PostMessageW")
	pPostQuitMessage          = user32.NewProc("PostQuitMessage")
	pGetModuleHandleW         = kernel32.NewProc("GetModuleHandleW")
	pDCompositionCreateDevice = dcompdll.NewProc("DCompositionCreateDevice")
	pD3DCompile               = d3dcompiler.NewProc("D3DCompile")
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
	wmNcHitTest             = 0x0084
	htCaption               = 2
	wmClose                 = 0x0010
	wmDestroy               = 0x0002

	fmtR8G8B8A8  = 28
	bindSRV      = 0x8
	bindCBuf     = 0x4
	usageDefault = 0
)

const ptrSize = unsafe.Sizeof(uintptr(0))

func comCall(this, idx uintptr, args ...uintptr) uintptr {
	vtbl := *(*uintptr)(unsafe.Pointer(this))
	fn := *(*uintptr)(unsafe.Pointer(vtbl + idx*ptrSize))
	r1, _, _ := syscall.SyscallN(fn, append([]uintptr{this}, args...)...)
	return r1
}

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
	Width, Height                     uint32
	Format                            uint32
	Stereo                            int32
	SampleDesc                        dxgiSampleDesc
	BufferUsage, BufferCount, Scaling uint32
	SwapEffect, AlphaMode, Flags      uint32
}
type samplerDesc struct {
	Filter         uint32
	AddressU       uint32
	AddressV       uint32
	AddressW       uint32
	MipLODBias     float32
	MaxAnisotropy  uint32
	ComparisonFunc uint32
	BorderColor    [4]float32
	MinLOD         float32
	MaxLOD         float32
}
type bufferDesc struct {
	ByteWidth, Usage, BindFlags, CPUAccessFlags, MiscFlags, StructureByteStride uint32
}
type subresourceData struct {
	pSysMem    uintptr
	rowPitch   uint32
	depthPitch uint32
}
type viewport struct{ TopLeftX, TopLeftY, Width, Height, MinDepth, MaxDepth float32 }

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

func wndProc(hwnd, message, wp, lp uintptr) uintptr {
	switch message {
	case wmNcHitTest:
		return htCaption // 整窗可拖
	case wmDestroy:
		pPostQuitMessage.Call(0)
		return 0
	}
	r, _, _ := pDefWindowProcW.Call(hwnd, message, wp, lp)
	return r
}
func fail(msg string, a ...any) { fmt.Printf("❌ "+msg+"\n", a...); os.Exit(2) }
func step(name string, hr uintptr) {
	if int32(hr) < 0 {
		fail("%s hr=0x%X", name, uint32(hr))
	}
	fmt.Printf("✓ %s\n", name)
}

const shaderSrc = `
cbuffer C : register(b0) { float2 screen; float2 gOrigin; float2 gSize; };
Texture2D tex : register(t0);
SamplerState smp : register(s0);
struct VSOut { float4 pos : SV_Position; float2 uv : TEXCOORD0; };
VSOut VS(uint id : SV_VertexID) {
    VSOut o;
    float2 t = float2((id << 1) & 2, id & 2);
    o.uv = t;
    o.pos = float4(t * float2(2,-2) + float2(-1,1), 0, 1);
    return o;
}
float4 PS(VSOut i) : SV_Target {
    float2 screenPx = gOrigin + i.uv * gSize;
    float2 deskUV = screenPx / screen;
    float2 c = (gOrigin + gSize * 0.5) / screen;
    deskUV = c + (deskUV - c) * 0.7;          // 放大镜 1.4x
    float3 col = tex.Sample(smp, deskUV).rgb;
    col *= float3(0.85, 0.95, 1.15);          // 冷色调
    return float4(col, 1.0);
}`

func compile(entry, target string) (uintptr, uintptr) {
	src := []byte(shaderSrc)
	e := append([]byte(entry), 0)
	t := append([]byte(target), 0)
	var blob, errBlob uintptr
	hr, _, _ := pD3DCompile.Call(
		uintptr(unsafe.Pointer(&src[0])), uintptr(len(src)),
		0, 0, 0,
		uintptr(unsafe.Pointer(&e[0])), uintptr(unsafe.Pointer(&t[0])),
		0, 0,
		uintptr(unsafe.Pointer(&blob)), uintptr(unsafe.Pointer(&errBlob)),
	)
	if int32(hr) < 0 {
		if errBlob != 0 {
			p := comCall(errBlob, 3)
			n := comCall(errBlob, 4)
			fmt.Printf("shader 编译错误: %s\n", string(unsafe.Slice((*byte)(unsafe.Pointer(p)), int(n))))
		}
		fail("D3DCompile(%s) hr=0x%X", entry, uint32(hr))
	}
	return comCall(blob, 3), comCall(blob, 4)
}

func main() {
	runtime.LockOSThread()
	if win.IsValidDpiAwarenessContext(win.DpiAwarenessContextPerMonitorAwareV2) {
		win.SetThreadDpiAwarenessContext(win.DpiAwarenessContextPerMonitorAwareV2)
	}

	hInst, _, _ := pGetModuleHandleW.Call(0)
	className := syscall.StringToUTF16Ptr("SpikeCClass")
	wc := wndClassExW{cbSize: uint32(unsafe.Sizeof(wndClassExW{})),
		lpfnWndProc: syscall.NewCallback(wndProc), hInstance: hInst, lpszClassName: className}
	if a, _, e := pRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc))); a == 0 {
		fail("RegisterClassExW: %v", e)
	}
	const ww, wh = 420, 200
	hwnd, _, _ := pCreateWindowExW.Call(
		uintptr(wsExNoredirectionbitmap|wsExTopmost|wsExToolwindow|wsExNoactivate),
		uintptr(unsafe.Pointer(className)), uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("SpikeC"))),
		uintptr(wsPopup|wsVisible), 500, 350, ww, wh, 0, 0, hInst, 0)
	if hwnd == 0 {
		fail("CreateWindowExW")
	}

	go renderThread(hwnd, ww, wh)

	// 主线程：标准消息循环。拖动的模态循环也在这里跑，挡不住渲染线程。
	var m msgT
	for {
		r, _, _ := pGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		pTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		pDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}
}

func renderThread(hwnd uintptr, ww, wh float32) {
	runtime.LockOSThread()
	if win.IsValidDpiAwarenessContext(win.DpiAwarenessContextPerMonitorAwareV2) {
		win.SetThreadDpiAwarenessContext(win.DpiAwarenessContextPerMonitorAwareV2)
	}

	device, ctx, derr := d3d11.NewD3D11Device()
	if derr != nil {
		fail("NewD3D11Device: %v", derr)
	}
	dev := uintptr(unsafe.Pointer(device))
	dctx := uintptr(unsafe.Pointer(ctx))

	var dxgiDev *dxgi.IDXGIDevice1
	step("QI IDXGIDevice1", uintptr(uint32(device.QueryInterface(dxgi.IID_IDXGIDevice1, &dxgiDev))))
	var pAdapter unsafe.Pointer
	step("GetParent Adapter1", uintptr(uint32(dxgiDev.GetParent(dxgi.IID_IDXGIAdapter1, &pAdapter))))
	adapter := (*dxgi.IDXGIAdapter1)(pAdapter)
	var output *dxgi.IDXGIOutput
	step("EnumOutputs(0)", uintptr(adapter.EnumOutputs(0, &output)))
	var output5 *dxgi.IDXGIOutput5
	step("QI IDXGIOutput5", uintptr(uint32(output.QueryInterface(dxgi.IID_IDXGIOutput5, &output5))))
	var odesc dxgi.DXGI_OUTPUT_DESC
	output5.GetDesc(&odesc)
	screenW := odesc.DesktopCoordinates.Right - odesc.DesktopCoordinates.Left
	screenH := odesc.DesktopCoordinates.Bottom - odesc.DesktopCoordinates.Top
	fmt.Printf("屏幕物理尺寸: %dx%d\n", screenW, screenH)
	var dup *dxgi.IDXGIOutputDuplication
	step("DuplicateOutput1", uintptr(uint32(output5.DuplicateOutput1(dxgiDev, 0,
		[]dxgi.DXGI_FORMAT{fmtR8G8B8A8}, &dup))))

	var srvTex *d3d11.ID3D11Texture2D
	tdesc := d3d11.D3D11_TEXTURE2D_DESC{
		Width: uint32(screenW), Height: uint32(screenH), MipLevels: 1, ArraySize: 1,
		Format: fmtR8G8B8A8, SampleDesc: dxgi.DXGI_SAMPLE_DESC{Count: 1},
		Usage: usageDefault, BindFlags: bindSRV,
	}
	step("CreateTexture2D(SRV)", uintptr(uint32(device.CreateTexture2D(&tdesc, &srvTex))))
	srvTexThis := uintptr(unsafe.Pointer(srvTex))
	var srv uintptr
	step("CreateShaderResourceView", comCall(dev, 7, srvTexThis, 0, uintptr(unsafe.Pointer(&srv))))

	var dxgiDeviceForFactory uintptr
	step("QI IDXGIDevice", comCall(dev, 0, uintptr(unsafe.Pointer(&iidIDXGIDevice)), uintptr(unsafe.Pointer(&dxgiDeviceForFactory))))
	var dcompDevice uintptr
	if r, _, _ := pDCompositionCreateDevice.Call(dxgiDeviceForFactory,
		uintptr(unsafe.Pointer(&iidIDCompositionDevice)), uintptr(unsafe.Pointer(&dcompDevice))); int32(r) < 0 {
		fail("DCompositionCreateDevice hr=0x%X", uint32(r))
	}
	var adp, factory uintptr
	step("dxgiDevice.GetAdapter", comCall(dxgiDeviceForFactory, 7, uintptr(unsafe.Pointer(&adp))))
	step("adapter.GetParent Factory2", comCall(adp, 6, uintptr(unsafe.Pointer(&iidIDXGIFactory2)), uintptr(unsafe.Pointer(&factory))))
	scd := dxgiSwapChainDesc1{Width: uint32(ww), Height: uint32(wh), Format: fmtR8G8B8A8,
		SampleDesc: dxgiSampleDesc{Count: 1}, BufferUsage: 0x20, BufferCount: 2,
		SwapEffect: 3, AlphaMode: 1}
	var swapchain uintptr
	step("CreateSwapChainForComposition", comCall(factory, 24, dev, uintptr(unsafe.Pointer(&scd)), 0, uintptr(unsafe.Pointer(&swapchain))))
	var backTex uintptr
	step("swapchain.GetBuffer", comCall(swapchain, 9, 0, uintptr(unsafe.Pointer(&iidID3D11Texture2D)), uintptr(unsafe.Pointer(&backTex))))
	var rtv uintptr
	step("CreateRenderTargetView", comCall(dev, 9, backTex, 0, uintptr(unsafe.Pointer(&rtv))))

	var target, visual uintptr
	step("CreateTargetForHwnd", comCall(dcompDevice, 6, hwnd, 1, uintptr(unsafe.Pointer(&target))))
	step("CreateVisual", comCall(dcompDevice, 7, uintptr(unsafe.Pointer(&visual))))
	comCall(visual, 15, swapchain)
	comCall(target, 3, visual)
	comCall(dcompDevice, 3)

	vsPtr, vsLen := compile("VS", "vs_5_0")
	psPtr, psLen := compile("PS", "ps_5_0")
	fmt.Println("✓ shader 编译")
	var vs, ps, sampler, cbuf uintptr
	step("CreateVertexShader", comCall(dev, 12, vsPtr, vsLen, 0, uintptr(unsafe.Pointer(&vs))))
	step("CreatePixelShader", comCall(dev, 15, psPtr, psLen, 0, uintptr(unsafe.Pointer(&ps))))
	sd := samplerDesc{Filter: 0x15, AddressU: 3, AddressV: 3, AddressW: 3, ComparisonFunc: 1, MaxLOD: 3.402823e+38}
	step("CreateSamplerState", comCall(dev, 23, uintptr(unsafe.Pointer(&sd)), uintptr(unsafe.Pointer(&sampler))))
	cb := [8]float32{float32(screenW), float32(screenH), 500, 350, ww, wh, 0, 0}
	bd := bufferDesc{ByteWidth: 32, Usage: usageDefault, BindFlags: bindCBuf}
	srd := subresourceData{pSysMem: uintptr(unsafe.Pointer(&cb[0]))}
	step("CreateBuffer(cbuffer)", comCall(dev, 3, uintptr(unsafe.Pointer(&bd)), uintptr(unsafe.Pointer(&srd)), uintptr(unsafe.Pointer(&cbuf))))

	pShowWindow.Call(hwnd, swShowNoActivate)
	pSetWindowDisplayAffinity.Call(hwnd, wdaExcludeCapture)

	fmt.Println("------------------------------------------------------------")
	fmt.Println("渲染中（约 12 秒）：拖动窗口，确认放大的桌面内容【实时跟随、不冻结】。")

	vp := viewport{Width: ww, Height: wh, MaxDepth: 1}
	for frame := 0; frame < 720; frame++ {
		var fi dxgi.DXGI_OUTDUPL_FRAME_INFO
		var deskRes *dxgi.IDXGIResource
		hr := dup.AcquireNextFrame(16, &fi, &deskRes)
		if int32(hr) >= 0 {
			if fi.AccumulatedFrames > 0 && deskRes != nil {
				var deskTex *d3d11.ID3D11Texture2D
				if deskRes.QueryInterface(d3d11.IID_ID3D11Texture2D, &deskTex) >= 0 {
					comCall(dctx, 47, srvTexThis, uintptr(unsafe.Pointer(deskTex)))
					deskTex.Release()
				}
			}
			if deskRes != nil {
				deskRes.Release()
			}
			dup.ReleaseFrame()
		}

		var wr rectT
		pGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&wr)))
		cb[2], cb[3] = float32(wr.left), float32(wr.top)
		comCall(dctx, 48, cbuf, 0, 0, uintptr(unsafe.Pointer(&cb[0])), 0, 0)

		comCall(dctx, 44, 1, uintptr(unsafe.Pointer(&vp)))
		comCall(dctx, 33, 1, uintptr(unsafe.Pointer(&rtv)), 0)
		comCall(dctx, 24, 4)
		comCall(dctx, 11, vs, 0, 0)
		comCall(dctx, 9, ps, 0, 0)
		comCall(dctx, 8, 0, 1, uintptr(unsafe.Pointer(&srv)))
		comCall(dctx, 10, 0, 1, uintptr(unsafe.Pointer(&sampler)))
		comCall(dctx, 16, 0, 1, uintptr(unsafe.Pointer(&cbuf)))
		comCall(dctx, 13, 3, 0)
		comCall(swapchain, 8, 0, 0)

		time.Sleep(16 * time.Millisecond)
	}

	fmt.Println("------------------------------------------------------------")
	fmt.Println("结论: ✅ 渲染线程独立于 UI —— 若拖动时放大内容实时跟随不冻结，流畅度问题清零。")
	pPostMessageW.Call(hwnd, wmClose, 0, 0) // 通知主线程退出
}
