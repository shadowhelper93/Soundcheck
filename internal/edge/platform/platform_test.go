package platform_test

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/soundcheck/soundcheck/internal/edge/platform"
)

func TestLookPathFound(t *testing.T) {
	binary := "go"
	if runtime.GOOS == "windows" {
		binary = "cmd"
	}
	path, err := platform.LookPath(binary)
	if err != nil {
		t.Fatalf("LookPath(%q) unexpected error: %v", binary, err)
	}
	if path == "" {
		t.Fatalf("LookPath(%q) returned empty path", binary)
	}
}

func TestLookPathNotFound(t *testing.T) {
	_, err := platform.LookPath("non_existent_binary_xyz_12345")
	if err == nil {
		t.Fatal("expected error for non-existent binary, got nil")
	}
}

func TestConfigDirDefault(t *testing.T) {
	if runtime.GOOS == "windows" {
		dir, err := platform.ConfigDir()
		if err != nil {
			t.Fatalf("ConfigDir() error: %v", err)
		}
		if !strings.HasSuffix(dir, "soundcheck") {
			t.Fatalf("expected path ending in soundcheck, got %q", dir)
		}
		return
	}
	t.Setenv("XDG_CONFIG_HOME", "")
	dir, err := platform.ConfigDir()
	if err != nil {
		t.Fatalf("ConfigDir() error: %v", err)
	}
	if !strings.HasSuffix(dir, filepath.Join(".config", "soundcheck")) {
		t.Fatalf("expected path ending in .config/soundcheck, got %q", dir)
	}
}

func TestCacheDirDefault(t *testing.T) {
	if runtime.GOOS == "windows" {
		dir, err := platform.CacheDir()
		if err != nil {
			t.Fatalf("CacheDir() error: %v", err)
		}
		if !strings.HasSuffix(dir, "soundcheck") {
			t.Fatalf("expected path ending in soundcheck, got %q", dir)
		}
		return
	}
	t.Setenv("XDG_CACHE_HOME", "")
	dir, err := platform.CacheDir()
	if err != nil {
		t.Fatalf("CacheDir() error: %v", err)
	}
	if !strings.HasSuffix(dir, filepath.Join(".cache", "soundcheck")) {
		t.Fatalf("expected path ending in .cache/soundcheck, got %q", dir)
	}
}

func TestConfigDirEnvOverride(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Setenv("APPDATA", `C:\MockAppData`)
		dir, err := platform.ConfigDir()
		if err != nil {
			t.Fatalf("ConfigDir() error: %v", err)
		}
		expected := filepath.Join(`C:\MockAppData`, "soundcheck")
		if dir != expected {
			t.Fatalf("expected %q, got %q", expected, dir)
		}
	} else {
		t.Setenv("XDG_CONFIG_HOME", "/custom/xdg/config")
		dir, err := platform.ConfigDir()
		if err != nil {
			t.Fatalf("ConfigDir() error: %v", err)
		}
		expected := filepath.Join("/custom/xdg/config", "soundcheck")
		if dir != expected {
			t.Fatalf("expected %q, got %q", expected, dir)
		}
	}
}

func TestCacheDirEnvOverride(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Setenv("LOCALAPPDATA", `C:\MockLocalAppData`)
		dir, err := platform.CacheDir()
		if err != nil {
			t.Fatalf("CacheDir() error: %v", err)
		}
		expected := filepath.Join(`C:\MockLocalAppData`, "soundcheck")
		if dir != expected {
			t.Fatalf("expected %q, got %q", expected, dir)
		}
	} else {
		t.Setenv("XDG_CACHE_HOME", "/custom/xdg/cache")
		dir, err := platform.CacheDir()
		if err != nil {
			t.Fatalf("CacheDir() error: %v", err)
		}
		expected := filepath.Join("/custom/xdg/cache", "soundcheck")
		if dir != expected {
			t.Fatalf("expected %q, got %q", expected, dir)
		}
	}
}
