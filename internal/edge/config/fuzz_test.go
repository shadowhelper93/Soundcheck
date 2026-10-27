package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/soundcheck/soundcheck/internal/edge/config"
)

func FuzzConfigParse(f *testing.F) {
	if data, err := os.ReadFile(filepath.Join("..", "..", "..", "agent-doctor.yaml")); err == nil {
		f.Add(data)
	}
	f.Add([]byte(""))
	f.Add([]byte("version: \"2026.1\"\n"))
	f.Add([]byte("invalid: yaml: [content"))
	f.Add([]byte("roles:\n  primary:\n    candidates: []\n"))

	f.Fuzz(func(_ *testing.T, data []byte) {
		_, _ = config.Parse(data)
	})
}
