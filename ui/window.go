package ui

import (
	"claude-traffic-light/config"
	"claude-traffic-light/state"
)

// Window 管理浮动挂件窗口。
//
// WebView2 实现已退役（见 docs/superpowers/plans/2026-06-03-native-liquid-glass.md）。
// DComp 透明置顶窗 + 折射渲染将在 Task 3-6 实现。当前为编译桩，保持
// main.go 依赖的 New/SetState/Run 接口不变。
type Window struct {
	cfg     config.Config
	cfgPath string
}

// New 创建挂件窗口。
func New(cfgPath string, cfg config.Config) *Window {
	return &Window{cfg: cfg, cfgPath: cfgPath}
}

// SetState 更新红绿灯状态（桩，待渲染管线接入）。
func (w *Window) SetState(s state.State) {}

// Run 启动消息循环并阻塞（桩，待 DComp 窗口实现）。
func (w *Window) Run() { select {} }
