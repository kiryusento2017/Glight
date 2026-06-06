# 窗口整体缩放（放大功能）设计

日期：2026-06-05
范围：仅本功能（开机自启功能另开 spec，本次不做）

> ⚠️ **此 spec 为早期设计，已被实现推翻，仅作决策历史留档，勿当作当前实现。**
> 实际实现改为：**右键菜单「调整大小…」弹 comctl32 滑块窗**（非拖拽四角）、画布 **240×144**（非 270:160）、缩放 **100%~2000%**。当前实现以 CLAUDE.md「整体缩放」+ `ui/sizedialog.go` + `ui/glass.hlsl` 为准。

## 目标

让红绿灯挂件可以用鼠标拖拽四角**等比例放大**，记住上次大小与位置，支持一键重置回默认。

## 核心洞察：shader 分辨率无关

当前 `glass.hlsl` 在 0..270 的「设计空间」里计算所有几何（`px = i.uv * CANVAS`），采样桌面用归一化坐标 `suv`。因此放大的实现 **不需要改 shader**：

> 只要把窗口与 swapchain 的实际像素尺寸放大 Z 倍，shader 原样运行，整个挂件（灯/圆角/折射带）等比清晰放大，不糊。

放大的本质 = **改窗口尺寸 + 同步 resize swapchain + 动态 viewport**。

## 缩放交互（自接管，与现有拖动一致）

项目已自接管鼠标拖动（取消系统 caption 拖动），缩放同样自接管，保持架构一致：

- 鼠标移到 pill 包围盒**四角**附近（约 24px 命中半径）→ 光标变对角双向箭头
  - 左上 / 右下角 → `IDC_SIZENWSE`（↖↘）
  - 右上 / 左下角 → `IDC_SIZENESW`（↗↙）
- 在角上 `WM_LBUTTONDOWN` → 进入 **resize 模式**（而非 drag 模式）
- resize 期间 `WM_MOUSEMOVE`：按对角线锚定计算新 scale
  - **对角锚定**：拖某角时，对角保持不动
  - **锁定纵横比** 270:160，缩放是单一标量 Z
  - **最小 Z = 1.0**（当前大小，不可更小）；**最大不限**
- `WM_LBUTTONUP` → 退出 resize，存盘
- 非角区域照旧 = 拖动移位（drag 模式）

### 角判定区位置

判定区放在 **pill 包围盒**（设计空间 230×96 居中，画布坐标四角约 (20,32)/(250,32)/(20,128)/(250,128)）的四角，随 scale 缩放，视觉上贴着可见玻璃，而非窗口透明边角。

## 渲染层 resize

- 缩放是 UI 线程事件，swapchain 由渲染线程拥有 → 线程安全通信：UI 线程写期望尺寸（atomic），渲染线程每帧检测变化
- 渲染线程检测到尺寸变化 → `IDXGISwapChain::ResizeBuffers` + 重建 RTV
- `Frame()` 中写死的 `winW/winH`（viewport）改为读动态尺寸
- 桌面捕获 `AcquireTexture(wr)` 已用 `GetWindowRect`，窗口变大自动捕获更大桌面区域，折射保持正确

## 持久化（config.json）

新增字段：

```go
type Config struct {
    X       int     `json:"x"`
    Y       int     `json:"y"`
    Locked  bool    `json:"locked"`
    Visible bool    `json:"visible"`
    Scale   float64 `json:"scale"` // 新增，默认 1.0
}
```

- 存 `scale` 标量而非宽高（更简洁，宽高 = round(270×scale)×round(160×scale)）
- `Default()` 中 `Scale: 1.0`
- 旧 config.json 无此字段 → JSON 解码后为 0，需在 Load 后做兜底：`if cfg.Scale < 1.0 { cfg.Scale = 1.0 }`
- 缩放结束（`WM_LBUTTONUP`）时存盘，与现有位置存盘同处

## 一键重置

右键菜单新增「重置大小和位置」：

- `scale = 1.0`
- 位置回**首次运行位置**：屏幕水平居中 + 贴顶 `Y = 16`
- 立即 SetWindowPos（同时改位置和尺寸）+ 触发 swapchain resize + 存盘

## 不在本次范围

- 开机自启（另开 spec）
- 滚轮缩放（用户明确不要）
- 缩放上限（用户要无限）

## 验证

1. 拖右下角放大 → pill 等比变大、灯/折射清晰不糊、左上角不动
2. 拖到最小 → 卡在 Z=1.0 不再缩小
3. 退出重开 → 恢复上次大小和位置
4. 右键「重置」→ 回到 100% 居中贴顶
5. 旧 config.json（无 scale 字段）→ 启动按 100% 不崩
6. 缩放后拖动移位 → 正常，角区/中区不串模式
