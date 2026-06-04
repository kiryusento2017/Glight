package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTuningRoundtrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "glass-tuning.json")
	in := Tuning{
		CornerR: 28, CornerN: 4, RefractBand: 30, EdgeSqueeze: 0.1,
		Contrast: 1.3, Brightness: 1.1, Saturate: 1.2,
		LampR: 14, LampGap: 60, Glow: 0.7,
	}
	if err := SaveTuning(path, in); err != nil {
		t.Fatal(err)
	}
	out, err := LoadTuning(path)
	if err != nil {
		t.Fatal(err)
	}
	if out != in {
		t.Errorf("got %+v, want %+v", out, in)
	}
}

func TestLoadTuningMissing(t *testing.T) {
	tn, err := LoadTuning(filepath.Join(t.TempDir(), "nope.json"))
	if err != nil {
		t.Fatal("missing file should return defaults, not error")
	}
	if tn != DefaultTuning() {
		t.Errorf("got %+v, want %+v", tn, DefaultTuning())
	}
}

// 缺字段（部分 JSON）应保留默认值，不被零值覆盖。
func TestLoadTuningPartialKeepsDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "glass-tuning.json")
	if err := os.WriteFile(path, []byte(`{"cornerR": 20}`), 0644); err != nil {
		t.Fatal(err)
	}
	tn, err := LoadTuning(path)
	if err != nil {
		t.Fatal(err)
	}
	if tn.CornerR != 20 {
		t.Errorf("cornerR: got %v, want 20", tn.CornerR)
	}
	if tn.CornerN != DefaultTuning().CornerN {
		t.Errorf("cornerN should keep default %v, got %v", DefaultTuning().CornerN, tn.CornerN)
	}
}
