package config

import (
	"path/filepath"
	"testing"
)

func TestRoundtrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	in := Config{X: 42, Y: 99, Locked: true, Visible: false, Scale: 1.5}
	if err := Save(path, in); err != nil {
		t.Fatal(err)
	}
	out, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if out != in {
		t.Errorf("got %+v, want %+v", out, in)
	}
}

func TestLoadMissing(t *testing.T) {
	cfg, err := Load("/no/such/file.json")
	if err != nil {
		t.Fatal("missing file should return defaults, not error")
	}
	if cfg != Default() {
		t.Errorf("got %+v, want %+v", cfg, Default())
	}
}
