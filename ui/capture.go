package ui

import (
	"errors"
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
	tex  uintptr // 渲染 device 上的桌面纹理
	srv  uintptr // 供 shader 采样
	buf  []byte  // winW×winH×4 裁剪缓冲（RGBA）
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

	return &Capture{
		dup: dup, gdev: gdev, gctx: gctx,
		bounds: bounds, full: image.NewRGBA(bounds),
		rctx: rctx, tex: tex, srv: srv,
		buf: make([]byte, winW*winH*4),
	}, nil
}

// AcquireTexture 抓当前桌面帧，裁出 winRect 覆盖的那块上传到 GPU 纹理，返回 SRV。
// ok=false 表示本帧无新画面（桌面静止时 duplication 不出帧），用上一帧 SRV 即可。
func (c *Capture) AcquireTexture(winRect RECT) (srv uintptr, ok bool) {
	if err := c.dup.GetImage(c.full, 0); err != nil {
		if errors.Is(err, outputduplication.ErrNoImageYet) {
			return c.srv, false
		}
		return c.srv, false // 其它错误也回退上一帧，不打断渲染
	}

	ox := int(winRect.Left) - c.bounds.Min.X
	oy := int(winRect.Top) - c.bounds.Min.Y
	dw, dh := c.bounds.Dx(), c.bounds.Dy()
	stride := c.full.Stride
	const dstRow = winW * 4

	// 裁剪（屏幕坐标→full 坐标），越界像素填黑——拖到屏幕边缘也不 panic
	for y := 0; y < winH; y++ {
		drow := c.buf[y*dstRow : (y+1)*dstRow]
		sy := oy + y
		for x := 0; x < winW; x++ {
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
