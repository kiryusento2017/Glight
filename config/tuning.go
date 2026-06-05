package config

import (
	"encoding/json"
	"errors"
	"os"
)

// Tuning 是液态玻璃的实时可调视觉参数，存于 exe 同目录 glass-tuning.json。
// 渲染线程每 ~500ms 热重载：保存文件即生效，无需重编/重启。
// 不含 pill/画布尺寸（涉及窗口重建，无法热重载）。
type Tuning struct {
	CornerR     float32 `json:"cornerR"`     // 圆角半径（像素）
	CornerN     float32 `json:"cornerN"`     // 角部连续曲率指数（>2 苹果味 G2）
	RefractBand float32 `json:"refractBand"` // 折射带深度（像素）：仅距边缘此深度内折射
	EdgeSqueeze float32 `json:"edgeSqueeze"` // 边缘最强收缩系数（0=边缘采样中心，最强）
	Contrast    float32 `json:"contrast"`    // 对比度
	Brightness  float32 `json:"brightness"`  // 亮度
	Saturate    float32 `json:"saturate"`    // 饱和度
	LampR       float32 `json:"lampR"`       // 灯半径（像素）
	LampGap     float32 `json:"lampGap"`     // 灯间距（像素，红↔黄、黄↔绿）
	Glow        float32 `json:"glow"`        // 点亮外发光强度

	// 弹簧形变物理（仅 Go 端用，不进 shader cbuffer）
	SpringK        float32 `json:"springK"`        // 弹簧刚度（越大回复越快）
	SpringC        float32 `json:"springC"`        // 弹簧阻尼（越小回弹越明显）
	SteadyX        float32 `json:"steadyX"`        // 稳态水平缩放（<1=静止时稍窄）
	SteadyY        float32 `json:"steadyY"`        // 稳态垂直缩放（>1=静止时稍长）
	PressX         float32 `json:"pressX"`         // 按下时水平缩放（>1=按扁变宽）
	PressY         float32 `json:"pressY"`         // 按下时垂直缩放（<1=按扁变矮）
	DragK          float32 `json:"dragK"`          // 拖动速度→形变力度（越大拖得越快越窄）
	DragMin        float32 `json:"dragMin"`        // 拖动形变下限（保护，0.5=最多缩窄到 50%）
	ReleaseImpulse float32 `json:"releaseImpulse"` // 松手速度倍率（>1 强化过冲回弹）
}

// DefaultTuning 返回与 shader 原写死值一致的默认参数。
func DefaultTuning() Tuning {
	return Tuning{
		CornerR:     48,
		CornerN:     2.1,
		RefractBand: 3,
		EdgeSqueeze: 0.25,
		Contrast:    1.2,
		Brightness:  0.9,
		Saturate:    1.5,
		LampR:       19,
		LampGap:     64,
		Glow:        0,
		SpringK:    120,
		SpringC:    8,
		SteadyX:    0.91,
		SteadyY:    1.11,
		PressX:     0.82,
		PressY:     1.22,
		DragK:      0.02,
		DragMin:    0.5,
		ReleaseImpulse: 1.5,
	}
}

// LoadTuning 读 glass-tuning.json；文件不存在或损坏均回退到默认（永远拿到可用值）。
func LoadTuning(path string) (Tuning, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return DefaultTuning(), nil
	}
	if err != nil {
		return DefaultTuning(), err
	}
	t := DefaultTuning() // 缺字段保留默认值
	if err := json.Unmarshal(data, &t); err != nil {
		return DefaultTuning(), nil // corrupt → safe fallback
	}
	return t, nil
}

// SaveTuning 写出 glass-tuning.json（首次运行生成默认文件供用户编辑）。
func SaveTuning(path string, t Tuning) error {
	data, _ := json.MarshalIndent(t, "", "  ")
	return os.WriteFile(path, data, 0644)
}
