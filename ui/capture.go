package ui

import (
	"fmt"
	"image"
	"unsafe"

	"github.com/kirides/go-d3d/d3d11"
	"github.com/kirides/go-d3d/outputduplication"
)

// Capture 抓取窗口背后那块桌面，作为 shader 可采样的 GPU 纹理（路 B）。
//
// go-d3d 高层只暴露 CPU 抓图（GetImage 把桌面拷进 STAGING 纹理读到内存），
// 不给可采样的 GPU 桌面纹理。于是：duplication 用独立 device 抓整屏到 CPU，
// 每帧裁出窗口覆盖的 winW×winH 矩形，UpdateSubresource 上传到渲染 device 上
// 自建的 SHADER_RESOURCE 纹理。折射只需「窗口背后那块桌面」，裁小块即够。
//
// 注：窗口设了 WDA_EXCLUDEFROMCAPTURE，Desktop Duplication 抓不到挂件自身，
// 自折射反馈天然断开。
type Capture struct {
	dup    *outputduplication.OutputDuplicator
	gdev   *d3d11.ID3D11Device        // duplication 专用 device（CPU 往返中立）
	gctx   *d3d11.ID3D11DeviceContext
	bounds image.Rectangle            // 桌面坐标范围
	full   *image.RGBA                // 整屏帧缓冲

	rctx uintptr // 渲染 device context（UpdateSubresource）
	rdev uintptr // 渲染 device（Resize 时重建纹理/SRV）
	tex  uintptr // 渲染 device 上的桌面纹理
	srv  uintptr // 供 shader 采样
	buf  []byte  // w×h×4 裁剪缓冲（RGBA）
	w, h int     // 当前桌面纹理尺寸（随窗口缩放变化）
}

// newCapture 在渲染 device(rdev/rctx) 上建桌面 SRV 纹理，并启动 Desktop Duplication。
func newCapture(rdev, rctx uintptr) (*Capture, error) {
	gdev, gctx, err := d3d11.NewD3D11Device()
	if err != nil {
		return nil, fmt.Errorf("capture NewD3D11Device: %w", err)
	}
	dup, err := outputduplication.NewIDXGIOutputDuplication(gdev, gctx, 0)
	if err != nil {
		gctx.Release()
		gdev.Release()
		return nil, fmt.Errorf("NewIDXGIOutputDuplication: %w", err)
	}
	bounds, err := dup.GetBounds()
	if err != nil {
		dup.Release()
		gctx.Release()
		gdev.Release()
		return nil, fmt.Errorf("GetBounds: %w", err)
	}

	// 渲染 device 上建 winW×winH 桌面纹理（DEFAULT + SHADER_RESOURCE，UpdateSubresource 填充）
	desc := texture2DDesc{
		Width: winW, Height: winH, MipLevels: 1, ArraySize: 1,
		Format:     dxgiFormatR8G8B8A8,
		SampleDesc: dxgiSampleDesc{Count: 1},
		Usage:      d3d11UsageDefault,
		BindFlags:  d3d11BindSRV,
	}
	var tex uintptr
	if hr := comCall(rdev, vtDevCreateTexture2D,
		uintptr(unsafe.Pointer(&desc)), 0, uintptr(unsafe.Pointer(&tex))); failed(hr) {
		dup.Release()
		gctx.Release()
		gdev.Release()
		return nil, fmt.Errorf("CreateTexture2D: 0x%X", uint32(hr))
	}
	var srv uintptr
	if hr := comCall(rdev, vtDevCreateSRV, tex, 0, uintptr(unsafe.Pointer(&srv))); failed(hr) {
		comRelease(tex)
		dup.Release()
		gctx.Release()
		gdev.Release()
		return nil, fmt.Errorf("CreateShaderResourceView: 0x%X", uint32(hr))
	}

	c := &Capture{
		dup: dup, gdev: gdev, gctx: gctx,
		bounds: bounds, full: image.NewRGBA(bounds),
		rctx: rctx, rdev: rdev, tex: tex, srv: srv,
		buf: make([]byte, winW*winH*4),
		w:   winW, h: winH,
	}
	// 预热：抓一帧整屏填充缓存，避免首帧全黑（拖动时也从此缓存裁剪）
	for i := 0; i < 10; i++ {
		if c.dup.GetImage(c.full, 100) == nil {
			break
		}
	}
	return c, nil
}

// AcquireTexture 抓当前桌面帧，裁出 winRect 覆盖的那块上传到 GPU 纹理，返回 SRV。
// ok=false 表示本帧无新画面（桌面静止时 duplication 不出帧），用上一帧 SRV 即可。
func (c *Capture) AcquireTexture(winRect RECT) (srv uintptr, ok bool) {
	// 取新桌面帧；无新帧（桌面静止/拖动中）则沿用整屏缓存 c.full，
	// 这样窗口移动时仍按当前位置裁剪、折射跟随，无需等新 duplication 帧，
	// 拖动期间零整屏拷贝，顺滑。
	_ = c.dup.GetImage(c.full, 10) // 10ms 超时：timeout 0 可能在桌面变化时恰好错过 DWM 帧

	ox := int(winRect.Left) - c.bounds.Min.X
	oy := int(winRect.Top) - c.bounds.Min.Y
	dw, dh := c.bounds.Dx(), c.bounds.Dy()
	stride := c.full.Stride
	dstRow := c.w * 4

	// 裁剪（屏幕坐标→full 坐标），越界像素填黑——拖到屏幕边缘也不 panic
	for y := 0; y < c.h; y++ {
		drow := c.buf[y*dstRow : (y+1)*dstRow]
		sy := oy + y
		for x := 0; x < c.w; x++ {
			di := x * 4
			sx := ox + x
			if sy < 0 || sy >= dh || sx < 0 || sx >= dw {
				drow[di], drow[di+1], drow[di+2], drow[di+3] = 0, 0, 0, 255
				continue
			}
			si := sy*stride + sx*4
			drow[di] = c.full.Pix[si]
			drow[di+1] = c.full.Pix[si+1]
			drow[di+2] = c.full.Pix[si+2]
			drow[di+3] = 255
		}
	}

	comCall(c.rctx, vtCtxUpdateSubresource, c.tex, 0, 0,
		uintptr(unsafe.Pointer(&c.buf[0])), uintptr(dstRow), 0)
	return c.srv, true
}

// Resize 把桌面纹理/SRV/裁剪缓冲重建到 w×h（窗口缩放时调用）。
// 失败保持旧尺寸不变（best-effort，下一帧再试）。
func (c *Capture) Resize(w, h int) error {
	if w == c.w && h == c.h {
		return nil
	}
	if w < 1 || h < 1 {
		return fmt.Errorf("invalid size %dx%d", w, h)
	}
	desc := texture2DDesc{
		Width: uint32(w), Height: uint32(h), MipLevels: 1, ArraySize: 1,
		Format:     dxgiFormatR8G8B8A8,
		SampleDesc: dxgiSampleDesc{Count: 1},
		Usage:      d3d11UsageDefault,
		BindFlags:  d3d11BindSRV,
	}
	var tex uintptr
	if hr := comCall(c.rdev, vtDevCreateTexture2D,
		uintptr(unsafe.Pointer(&desc)), 0, uintptr(unsafe.Pointer(&tex))); failed(hr) {
		return fmt.Errorf("Resize CreateTexture2D: 0x%X", uint32(hr))
	}
	var srv uintptr
	if hr := comCall(c.rdev, vtDevCreateSRV, tex, 0, uintptr(unsafe.Pointer(&srv))); failed(hr) {
		comRelease(tex)
		return fmt.Errorf("Resize CreateShaderResourceView: 0x%X", uint32(hr))
	}
	comRelease(c.srv)
	comRelease(c.tex)
	c.tex, c.srv = tex, srv
	c.buf = make([]byte, w*h*4)
	c.w, c.h = w, h
	return nil
}

// Release 释放 duplication、device 与 GPU 资源。
func (c *Capture) Release() {
	comRelease(c.srv)
	comRelease(c.tex)
	if c.dup != nil {
		c.dup.Release()
	}
	if c.gctx != nil {
		c.gctx.Release()
	}
	if c.gdev != nil {
		c.gdev.Release()
	}
}
