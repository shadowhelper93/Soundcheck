package ui_test

import (
	"io/fs"
	"testing"

	"github.com/soundcheck/soundcheck/ui"
)

func TestAssets(t *testing.T) {
	assets, err := ui.Assets()
	if err != nil {
		t.Fatalf("ui.Assets() failed: %v", err)
	}

	data, err := fs.ReadFile(assets, "index.html")
	if err != nil {
		t.Fatalf("failed to read index.html from assets: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("embedded index.html is empty")
	}
}
