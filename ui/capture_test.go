package ui

import (
	"errors"
	"fmt"
	"testing"

	"github.com/kirides/go-d3d/outputduplication"
)

// TestShouldRebuild 覆盖「这个 GetImage 错误该不该触发 duplication 重建」的决策。
// ErrNoImageYet 是正常的桌面静止/超时，绝不能重建；其余错误（会话切换失效）才重建。
func TestShouldRebuild(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil 无错误", nil, false},
		{"ErrNoImageYet 桌面静止", outputduplication.ErrNoImageYet, false},
		{"包装的 ErrNoImageYet", fmt.Errorf("snapshot: %w", outputduplication.ErrNoImageYet), false},
		{"ACCESS_LOST 会话切换失效", errors.New("failed to AcquireNextFrame. DXGI_ERROR_ACCESS_LOST"), true},
	}
	for _, tt := range tests {
		if got := shouldRebuild(tt.err); got != tt.want {
			t.Errorf("%s: shouldRebuild=%v, want %v", tt.name, got, tt.want)
		}
	}
}
