package ui

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// 本文件是原生渲染管线的 COM 绑定层：一个通用的 vtable 调用器 comCall，
// 加上 D3D11 / DXGI / DirectComposition 的方法序号、IID、描述结构体与初始化
// 辅助函数。所有方法序号均由 spike A/B/C 实证（见 _spike_capture/）。

const ptrSize = unsafe.Sizeof(uintptr(0))

// comCall 通过对象的 vtable 调用第 idx 个方法（IUnknown 占 0/1/2）。
// this 是 COM 对象指针（其首字段即 vtable 指针）。返回 HRESULT/原始返回值。
func comCall(this, idx uintptr, args ...uintptr) uintptr {
	vtbl := *(*uintptr)(unsafe.Pointer(this))
	fn := *(*uintptr)(unsafe.Pointer(vtbl + idx*ptrSize))
	r1, _, _ := syscall.SyscallN(fn, append([]uintptr{this}, args...)...)
	return r1
}

// comRelease 调用 IUnknown::Release。
func comRelease(this uintptr) {
	if this != 0 {
		comCall(this, vtRelease)
	}
}

func failed(hr uintptr) bool { return int32(hr) < 0 }

// vtable 方法序号
const (
	// IUnknown
	vtQueryInterface = 0
	vtRelease        = 2

	// ID3D11Device
	vtDevCreateBuffer  = 3
	vtDevCreateSRV     = 7
	vtDevCreateRTV     = 9
	vtDevCreateVS      = 12
	vtDevCreatePS      = 15
	vtDevCreateSampler = 23

	// ID3D11DeviceContext
	vtCtxPSSetShaderResources   = 8
	vtCtxPSSetShader            = 9
	vtCtxPSSetSamplers          = 10
	vtCtxVSSetShader            = 11
	vtCtxDraw                   = 13
	vtCtxPSSetConstantBuffers   = 16
	vtCtxIASetPrimitiveTopology = 24
	vtCtxOMSetRenderTargets     = 33
	vtCtxRSSetViewports         = 44
	vtCtxCopyResource           = 47
	vtCtxUpdateSubresource      = 48

	// IDXGIDevice / IDXGIObject / IDXGIFactory2
	vtDxgiDeviceGetAdapter   = 7
	vtDxgiObjectGetParent    = 6
	vtFactory2CreateSwapComp = 24

	// IDXGISwapChain
	vtSwapPresent   = 8
	vtSwapGetBuffer = 9

	// IDCompositionDevice
	vtDCompCommit           = 3
	vtDCompCreateTargetHwnd = 6
	vtDCompCreateVisual     = 7

	// IDCompositionTarget / IDCompositionVisual
	vtDCompTargetSetRoot    = 3
	vtDCompVisualSetContent = 15
)

// DXGI / D3D11 枚举常量
const (
	dxgiFormatR8G8B8A8    = 28
	dxgiUsageRTOut        = 0x20 // DXGI_USAGE_RENDER_TARGET_OUTPUT
	dxgiSwapEffectFlipSeq = 3    // DXGI_SWAP_EFFECT_FLIP_SEQUENTIAL
	dxgiAlphaPremult      = 1    // DXGI_ALPHA_MODE_PREMULTIPLIED

	d3d11UsageDefault = 0
	d3d11BindSRV      = 0x8  // D3D11_BIND_SHADER_RESOURCE
	d3d11BindCBuf     = 0x4  // D3D11_BIND_CONSTANT_BUFFER
	d3d11BindRTV      = 0x20 // D3D11_BIND_RENDER_TARGET

	d3d11TopologyTriList   = 4    // D3D11_PRIMITIVE_TOPOLOGY_TRIANGLELIST
	d3d11FilterLinear      = 0x15 // D3D11_FILTER_MIN_MAG_MIP_LINEAR
	d3d11AddressClamp      = 3    // D3D11_TEXTURE_ADDRESS_CLAMP
	d3d11ComparisonNever   = 1    // D3D11_COMPARISON_NEVER
	floatMax               = 3.402823e+38
)

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

// 描述结构体（字段顺序与 C ABI 一致，已由 spike 实证）
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
	dcompDLL                     = syscall.NewLazyDLL("dcomp.dll")
	procDCompositionCreateDevice = dcompDLL.NewProc("DCompositionCreateDevice")
)

// queryDXGIFactory 从 D3D11 device 取得 IDXGIDevice 以及 IDXGIFactory2。
// 返回的 dxgiDevice 供 DComp 使用，factory 供创建合成 swapchain。
func queryDXGIFactory(device uintptr) (dxgiDevice, factory uintptr, err error) {
	if hr := comCall(device, vtQueryInterface,
		uintptr(unsafe.Pointer(&iidIDXGIDevice)), uintptr(unsafe.Pointer(&dxgiDevice))); failed(hr) {
		return 0, 0, fmt.Errorf("QI IDXGIDevice: 0x%X", uint32(hr))
	}
	var adapter uintptr
	if hr := comCall(dxgiDevice, vtDxgiDeviceGetAdapter, uintptr(unsafe.Pointer(&adapter))); failed(hr) {
		return 0, 0, fmt.Errorf("GetAdapter: 0x%X", uint32(hr))
	}
	defer comRelease(adapter)
	if hr := comCall(adapter, vtDxgiObjectGetParent,
		uintptr(unsafe.Pointer(&iidIDXGIFactory2)), uintptr(unsafe.Pointer(&factory))); failed(hr) {
		return 0, 0, fmt.Errorf("GetParent IDXGIFactory2: 0x%X", uint32(hr))
	}
	return dxgiDevice, factory, nil
}

// createCompositionSwapchain 用 device 创建一个用于 DirectComposition 的
// premultiplied-alpha 合成 swapchain（R8G8B8A8，无需 BGRA_SUPPORT）。
func createCompositionSwapchain(factory, device uintptr, w, h uint32) (uintptr, error) {
	scd := dxgiSwapChainDesc1{
		Width: w, Height: h, Format: dxgiFormatR8G8B8A8,
		SampleDesc:  dxgiSampleDesc{Count: 1},
		BufferUsage: dxgiUsageRTOut, BufferCount: 2,
		SwapEffect:  dxgiSwapEffectFlipSeq, AlphaMode: dxgiAlphaPremult,
	}
	var sc uintptr
	if hr := comCall(factory, vtFactory2CreateSwapComp, device,
		uintptr(unsafe.Pointer(&scd)), 0, uintptr(unsafe.Pointer(&sc))); failed(hr) {
		return 0, fmt.Errorf("CreateSwapChainForComposition: 0x%X", uint32(hr))
	}
	return sc, nil
}

// dcompAttach 为 hwnd 建立 DirectComposition 设备/目标/视觉，把 swapchain
// 作为内容挂上并 Commit。返回的 dcompDevice 由调用方持有（后续可再 Commit）。
func dcompAttach(dxgiDevice, hwnd, swapchain uintptr) (dcompDevice uintptr, err error) {
	if hr, _, _ := procDCompositionCreateDevice.Call(dxgiDevice,
		uintptr(unsafe.Pointer(&iidIDCompositionDevice)), uintptr(unsafe.Pointer(&dcompDevice))); failed(hr) {
		return 0, fmt.Errorf("DCompositionCreateDevice: 0x%X", uint32(hr))
	}
	var target, visual uintptr
	if hr := comCall(dcompDevice, vtDCompCreateTargetHwnd, hwnd, 1, uintptr(unsafe.Pointer(&target))); failed(hr) {
		return 0, fmt.Errorf("CreateTargetForHwnd: 0x%X", uint32(hr))
	}
	if hr := comCall(dcompDevice, vtDCompCreateVisual, uintptr(unsafe.Pointer(&visual))); failed(hr) {
		return 0, fmt.Errorf("CreateVisual: 0x%X", uint32(hr))
	}
	comCall(visual, vtDCompVisualSetContent, swapchain)
	comCall(target, vtDCompTargetSetRoot, visual)
	comCall(dcompDevice, vtDCompCommit)
	return dcompDevice, nil
}

// backBufferRTV 取 swapchain 后台缓冲并创建 RenderTargetView。
func backBufferRTV(device, swapchain uintptr) (rtv uintptr, err error) {
	var backTex uintptr
	if hr := comCall(swapchain, vtSwapGetBuffer, 0,
		uintptr(unsafe.Pointer(&iidID3D11Texture2D)), uintptr(unsafe.Pointer(&backTex))); failed(hr) {
		return 0, fmt.Errorf("swapchain.GetBuffer: 0x%X", uint32(hr))
	}
	defer comRelease(backTex)
	if hr := comCall(device, vtDevCreateRTV, backTex, 0, uintptr(unsafe.Pointer(&rtv))); failed(hr) {
		return 0, fmt.Errorf("CreateRenderTargetView: 0x%X", uint32(hr))
	}
	return rtv, nil
}
