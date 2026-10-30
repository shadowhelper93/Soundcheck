package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveFallbackToConfig(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Clean(filepath.Join(tmpDir, "soundcheck.yaml"))

	err := SaveFallbackToConfig(cfgPath, "claude-sonnet-5-5", "opencode")
	if err != nil {
		t.Fatalf("unexpected error saving config: %v", err)
	}

	data, err := os.ReadFile(cfgPath) //nolint:gosec
	if err != nil {
		t.Fatalf("could not read written config: %v", err)
	}

	if len(data) == 0 {
		t.Fatal("expected non-empty config file")
	}

	_ = SaveFallbackToConfig(cfgPath, "claude-sonnet-5-5", "opencode")
}
