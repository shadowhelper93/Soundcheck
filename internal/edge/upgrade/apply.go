package upgrade

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// FindMatchingAsset locates the archive and checksums assets for target OS/arch.
func FindMatchingAsset(assets []ReleaseAsset, goos, goarch string) (ReleaseAsset, ReleaseAsset, error) {
	var archive, checksums ReleaseAsset
	targetSuffix := fmt.Sprintf("_%s_%s", goos, goarch)

	for _, a := range assets {
		if a.Name == "checksums.txt" {
			checksums = a
		}
		if strings.Contains(a.Name, targetSuffix) {
			archive = a
		}
	}
	if archive.DownloadURL == "" {
		return archive, checksums, fmt.Errorf("no matching release asset found for %s/%s", goos, goarch)
	}
	return archive, checksums, nil
}

// DownloadFile downloads a remote file to a local destination.
func DownloadFile(ctx context.Context, url, dest string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned http %d", resp.StatusCode)
	}
	out, err := os.OpenFile(filepath.Clean(dest), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()

	_, err = io.Copy(out, resp.Body)
	return err
}

// VerifyChecksum verifies that the file matches the expected hash in checksums.txt.
func VerifyChecksum(filePath, checksumsPath string) error {
	csBytes, err := os.ReadFile(filepath.Clean(checksumsPath))
	if err != nil {
		return fmt.Errorf("read checksums: %w", err)
	}
	targetName := filepath.Base(filePath)
	var expectedHash string
	for _, line := range strings.Split(string(csBytes), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && filepath.Base(fields[1]) == targetName {
			expectedHash = strings.ToLower(fields[0])
			break
		}
	}
	if expectedHash == "" {
		return fmt.Errorf("no checksum entry found for %s", targetName)
	}
	fileBytes, err := os.ReadFile(filepath.Clean(filePath))
	if err != nil {
		return fmt.Errorf("read target file: %w", err)
	}
	sum := sha256.Sum256(fileBytes)
	actualHash := hex.EncodeToString(sum[:])
	if actualHash != expectedHash {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedHash, actualHash)
	}
	return nil
}

// ExtractTarGz extracts the named binary from a .tar.gz archive.
func ExtractTarGz(archivePath, binaryName, destPath string) error {
	f, err := os.Open(filepath.Clean(archivePath))
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer func() { _ = gzr.Close() }()

	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		if filepath.Base(header.Name) == binaryName {
			return writeExtractedFile(destPath, tr, 0o755)
		}
	}
	return fmt.Errorf("binary %s not found in archive", binaryName)
}

// ExtractZip extracts the named binary from a .zip archive.
func ExtractZip(archivePath, binaryName, destPath string) error {
	r, err := zip.OpenReader(filepath.Clean(archivePath))
	if err != nil {
		return err
	}
	defer func() { _ = r.Close() }()

	for _, f := range r.File {
		if filepath.Base(f.Name) == binaryName {
			rc, err := f.Open()
			if err != nil {
				return err
			}
			defer func() { _ = rc.Close() }()
			return writeExtractedFile(destPath, rc, 0o755)
		}
	}
	return fmt.Errorf("binary %s not found in zip", binaryName)
}

func writeExtractedFile(destPath string, src io.Reader, mode os.FileMode) error {
	out, err := os.OpenFile(filepath.Clean(destPath), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()
	_, err = io.Copy(out, src)
	return err
}

// AtomicReplace safely replaces the target executable with the new binary.
func AtomicReplace(newBinaryPath, currentExecPath string) error {
	dir := filepath.Dir(currentExecPath)
	tempTarget := filepath.Join(dir, fmt.Sprintf(".soundcheck-update-%d", os.Getpid()))

	if err := os.Rename(newBinaryPath, tempTarget); err != nil {
		// Fallback across filesystem boundary
		if err := copyExecutable(newBinaryPath, tempTarget); err != nil {
			return fmt.Errorf("stage new executable: %w", err)
		}
	}
	defer func() { _ = os.Remove(tempTarget) }()

	if err := os.Chmod(tempTarget, 0o755); err != nil { //nolint:gosec // executable requires 0o755
		return fmt.Errorf("chmod executable: %w", err)
	}
	if err := os.Rename(tempTarget, currentExecPath); err != nil {
		return fmt.Errorf("atomic rename failed: %w", err)
	}
	return nil
}

func copyExecutable(src, dst string) error {
	in, err := os.Open(filepath.Clean(src))
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()

	out, err := os.OpenFile(filepath.Clean(dst), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755) //nolint:gosec // executable requires 0o755
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()

	_, err = io.Copy(out, in)
	return err
}
