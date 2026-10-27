package upgrade_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/soundcheck/soundcheck/internal/edge/upgrade"
)

func TestDetectPackageManager(t *testing.T) {
	tests := []struct {
		path     string
		expected upgrade.PackageManager
	}{
		{"/opt/homebrew/Cellar/soundcheck/2.0.0/bin/soundcheck", upgrade.PackageManagerBrew},
		{"/usr/local/Cellar/soundcheck/bin/soundcheck", upgrade.PackageManagerBrew},
		{`C:\Users\User\scoop\apps\soundcheck\current\soundcheck.exe`, upgrade.PackageManagerScoop},
		{"/usr/local/bin/soundcheck", upgrade.PackageManagerNone},
		{"/Users/sinamashini/.local/bin/soundcheck", upgrade.PackageManagerNone},
	}

	for _, tt := range tests {
		got := upgrade.DetectPackageManager(tt.path)
		if got != tt.expected {
			t.Errorf("DetectPackageManager(%q) = %v, want %v", tt.path, got, tt.expected)
		}
	}
}

func TestIsNewerVersion(t *testing.T) {
	if upgrade.IsNewerVersion("v2.0.0", "v2.0.0") {
		t.Errorf("expected false for identical versions")
	}
	if !upgrade.IsNewerVersion("v2.0.0", "v2.1.0") {
		t.Errorf("expected true for newer remote version")
	}
	if upgrade.IsNewerVersion("dev", "v2.1.0") {
		t.Errorf("expected false for dev version")
	}
}

func TestFindMatchingAsset(t *testing.T) {
	assets := []upgrade.ReleaseAsset{
		{Name: "soundcheck_2.0.0_darwin_arm64.tar.gz", DownloadURL: "https://example.com/darwin_arm64.tar.gz"},
		{Name: "soundcheck_2.0.0_linux_amd64.tar.gz", DownloadURL: "https://example.com/linux_amd64.tar.gz"},
		{Name: "checksums.txt", DownloadURL: "https://example.com/checksums.txt"},
	}

	arch, cs, err := upgrade.FindMatchingAsset(assets, "darwin", "arm64")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if arch.Name != "soundcheck_2.0.0_darwin_arm64.tar.gz" {
		t.Errorf("unexpected archive: %s", arch.Name)
	}
	if cs.Name != "checksums.txt" {
		t.Errorf("unexpected checksums: %s", cs.Name)
	}

	_, _, err = upgrade.FindMatchingAsset(assets, "windows", "arm64")
	if err == nil {
		t.Errorf("expected error for missing platform asset")
	}
}

func TestVerifyChecksumAndExtractTarGz(t *testing.T) {
	tempDir := t.TempDir()
	content := []byte("#!/bin/sh\necho test\n")

	// Create test .tar.gz
	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)
	hdr := &tar.Header{
		Name: "soundcheck",
		Mode: 0o755,
		Size: int64(len(content)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gzw.Close(); err != nil {
		t.Fatal(err)
	}

	archivePath := filepath.Join(tempDir, "soundcheck_darwin_arm64.tar.gz")
	if err := os.WriteFile(archivePath, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}

	// Calculate checksum
	sum := sha256.Sum256(buf.Bytes())
	hashHex := hex.EncodeToString(sum[:])
	csPath := filepath.Join(tempDir, "checksums.txt")
	csContent := fmt.Sprintf("%s  soundcheck_darwin_arm64.tar.gz\n", hashHex)
	if err := os.WriteFile(csPath, []byte(csContent), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := upgrade.VerifyChecksum(archivePath, csPath); err != nil {
		t.Fatalf("checksum verification failed: %v", err)
	}

	destBinary := filepath.Join(tempDir, "extracted-soundcheck")
	if err := upgrade.ExtractTarGz(archivePath, "soundcheck", destBinary); err != nil {
		t.Fatalf("tar.gz extraction failed: %v", err)
	}

	extracted, err := os.ReadFile(destBinary) //nolint:gosec // local temp file, path locally constructed
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(extracted, content) {
		t.Errorf("extracted content mismatch")
	}
}

func TestAtomicReplace(t *testing.T) {
	tempDir := t.TempDir()
	oldExec := filepath.Join(tempDir, "app")
	newExec := filepath.Join(tempDir, "app-new")

	if err := os.WriteFile(oldExec, []byte("v1"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(newExec, []byte("v2"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := upgrade.AtomicReplace(newExec, oldExec); err != nil {
		t.Fatalf("AtomicReplace failed: %v", err)
	}

	result, err := os.ReadFile(oldExec) //nolint:gosec // local temp file, path locally constructed
	if err != nil {
		t.Fatal(err)
	}
	if string(result) != "v2" {
		t.Errorf("expected 'v2', got %q", string(result))
	}
}
