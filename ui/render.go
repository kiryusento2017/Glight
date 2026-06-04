package ui

import (
	_ "embed"
	"fmt"
	"unsafe"

	"claude-traffic-light/config"
)

//go:embed glass.hlsl
var glassHLSL []byte

// Renderer 持有折射管线（编译好的 VS/PS + 采样器），每帧把桌面纹理折射
// 绘制到 swapchain 后台缓冲。Present 由调用方（renderThread）负责。
type Renderer struct {
	ctx  uintptr
	vs   uintptr
	ps   uintptr
	samp uintptr
	cbuf uintptr // constant buffer：状态/形变 + glass-tuning 视觉参数（64 字节）
}

// newRenderer 编译 glass.hlsl 并建立折射管线（用渲染 device dev / context ctx）。
func newRenderer(dev, ctx uintptr) (*Renderer, error) {
	vsBC, err := compileHLSL(glassHLSL, "VSMain", "vs_5_0")
	if err != nil {
		return nil, err
	}
	psBC, err := compileHLSL(glassHLSL, "PSMain", "ps_5_0")
	if err != nil {
		return nil, err
	}

	var vs uintptr
	if hr := comCall(dev, vtDevCreateVS,
		uintptr(unsafe.Pointer(&vsBC[0])), uintptr(len(vsBC)), 0,
		uintptr(unsafe.Pointer(&vs))); failed(hr) {
		return nil, fmt.Errorf("CreateVertexShader: 0x%X", uint32(hr))
	}
	var ps uintptr
	if hr := comCall(dev, vtDevCreatePS,
		uintptr(unsafe.Pointer(&psBC[0])), uintptr(len(psBC)), 0,
		uintptr(unsafe.Pointer(&ps))); failed(hr) {
		comRelease(vs)
		return nil, fmt.Errorf("CreatePixelShader: 0x%X", uint32(hr))
	}

	sd := samplerDesc{
		Filter:         d3d11FilterLinear,
		AddressU:       d3d11AddressClamp,
		AddressV:       d3d11AddressClamp,
		AddressW:       d3d11AddressClamp,
		ComparisonFunc: d3d11ComparisonNever,
		MinLOD:         0,
		MaxLOD:         floatMax,
	}
	var samp uintptr
	if hr := comCall(dev, vtDevCreateSampler,
		uintptr(unsafe.Pointer(&sd)), uintptr(unsafe.Pointer(&samp))); failed(hr) {
		comRelease(ps)
		comRelease(vs)
		return nil, fmt.Errorf("CreateSamplerState: 0x%X", uint32(hr))
	}

	// constant buffer：64 字节（状态/形变 4 + tuning 视觉参数 + pad），DEFAULT + UpdateSubresource
	bd := bufferDesc{ByteWidth: 64, Usage: d3d11UsageDefault, BindFlags: d3d11BindCBuf}
	var cbuf uintptr
	if hr := comCall(dev, vtDevCreateBuffer,
		uintptr(unsafe.Pointer(&bd)), 0, uintptr(unsafe.Pointer(&cbuf))); failed(hr) {
		comRelease(samp)
		comRelease(ps)
		comRelease(vs)
		return nil, fmt.Errorf("CreateBuffer(cbuf): 0x%X", uint32(hr))
	}

	return &Renderer{ctx: ctx, vs: vs, ps: ps, samp: samp, cbuf: cbuf}, nil
}

// Frame 把 desktopSRV 折射绘制到 rtv（全屏三角，3 顶点），叠加红绿灯。不 Present。
// active: 0灰/1绿/2黄/3红；blink: 0~1 闪烁亮度；scaleX/scaleY: 形变缩放（1=原尺寸）；
// t: glass-tuning 视觉参数（圆角/折射/调色/三灯）。params 布局须与 glass.hlsl cbuffer 一致。
func (r *Renderer) Frame(rtv, desktopSRV uintptr, active, blink, scaleX, scaleY float32, t config.Tuning) {
	ctx := r.ctx

	params := [16]float32{
		active, blink, scaleX, scaleY,
		t.CornerR, t.CornerN, t.RefractBand, t.EdgeSqueeze,
		t.Contrast, t.Brightness, t.Saturate, t.LampR,
		t.LampGap, t.Glow, 0, 0,
	}
	comCall(ctx, vtCtxUpdateSubresource, r.cbuf, 0, 0,
		uintptr(unsafe.Pointer(&params[0])), 0, 0)

	vp := viewport{Width: winW, Height: winH, MaxDepth: 1}
	comCall(ctx, vtCtxRSSetViewports, 1, uintptr(unsafe.Pointer(&vp)))

	rtvs := [1]uintptr{rtv}
	comCall(ctx, vtCtxOMSetRenderTargets, 1, uintptr(unsafe.Pointer(&rtvs[0])), 0)

	comCall(ctx, vtCtxIASetPrimitiveTopology, d3d11TopologyTriList)
	comCall(ctx, vtCtxVSSetShader, r.vs, 0, 0)
	comCall(ctx, vtCtxPSSetShader, r.ps, 0, 0)

	srvs := [1]uintptr{desktopSRV}
	comCall(ctx, vtCtxPSSetShaderResources, 0, 1, uintptr(unsafe.Pointer(&srvs[0])))
	samps := [1]uintptr{r.samp}
	comCall(ctx, vtCtxPSSetSamplers, 0, 1, uintptr(unsafe.Pointer(&samps[0])))
	cbufs := [1]uintptr{r.cbuf}
	comCall(ctx, vtCtxPSSetConstantBuffers, 0, 1, uintptr(unsafe.Pointer(&cbufs[0])))

	comCall(ctx, vtCtxDraw, 3, 0)
}

// Release 释放管线资源。
func (r *Renderer) Release() {
	comRelease(r.cbuf)
	comRelease(r.samp)
	comRelease(r.ps)
	comRelease(r.vs)
}
