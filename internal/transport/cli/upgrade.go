package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"github.com/soundcheck/soundcheck/internal/edge/upgrade"
)

type upgradeOptions struct {
	check bool
}

func newUpgradeCmd(rootOpts *RootOptions) *cobra.Command {
	uOpts := &upgradeOptions{}
	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Check for updates and upgrade Soundcheck to the latest release",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx, cancel := resolveContext(cmd.Context(), rootOpts.Timeout)
			defer cancel()
			return runUpgrade(ctx, cmd, uOpts)
		},
	}
	cmd.Flags().BoolVar(&uOpts.check, "check", false, "Check for newer version without applying update")
	return cmd
}

func runUpgrade(ctx context.Context, cmd *cobra.Command, opts *upgradeOptions) error {
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate executable: %w", err)
	}
	realPath, err := filepath.EvalSymlinks(execPath)
	if err == nil {
		execPath = realPath
	}

	pm := upgrade.DetectPackageManager(execPath)
	if pm != upgrade.PackageManagerNone {
		return reportPackageManagerAdvice(cmd, pm)
	}

	rel, err := upgrade.FetchLatest(ctx, "shadowhelper93/Soundcheck")
	if err != nil {
		return fmt.Errorf("check latest version: %w", err)
	}
	if !upgrade.IsNewerVersion(Version, rel.TagName) {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Soundcheck is already up to date (%s).\n", Version)
		return nil
	}
	if opts.check {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "New version available: %s (current: %s)\n", rel.TagName, Version)
		return nil
	}
	return applyUpgrade(ctx, cmd, rel, execPath)
}

func reportPackageManagerAdvice(cmd *cobra.Command, pm upgrade.PackageManager) error {
	w := cmd.OutOrStdout()
	switch pm {
	case upgrade.PackageManagerBrew:
		_, _ = fmt.Fprintln(w, "Soundcheck is managed by Homebrew. Please run: brew upgrade soundcheck")
	case upgrade.PackageManagerScoop:
		_, _ = fmt.Fprintln(w, "Soundcheck is managed by Scoop. Please run: scoop update soundcheck")
	}
	return nil
}

func applyUpgrade(ctx context.Context, cmd *cobra.Command, rel *upgrade.ReleaseInfo, execPath string) error {
	out := cmd.OutOrStdout()
	_, _ = fmt.Fprintf(out, "==> Upgrading Soundcheck %s -> %s...\n", Version, rel.TagName)

	archive, checksums, err := upgrade.FindMatchingAsset(rel.Assets, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return err
	}
	if checksums.DownloadURL == "" {
		return fmt.Errorf("release %s is missing checksums.txt asset", rel.TagName)
	}

	tmpDir, err := os.MkdirTemp("", "soundcheck-upgrade-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	archivePath := filepath.Join(tmpDir, archive.Name)
	if err := upgrade.DownloadFile(ctx, archive.DownloadURL, archivePath); err != nil {
		return fmt.Errorf("download archive: %w", err)
	}

	csPath := filepath.Join(tmpDir, "checksums.txt")
	if err := upgrade.DownloadFile(ctx, checksums.DownloadURL, csPath); err != nil {
		return fmt.Errorf("download checksums: %w", err)
	}
	if err := upgrade.VerifyChecksum(archivePath, csPath); err != nil {
		return fmt.Errorf("verify checksum: %w", err)
	}

	return extractAndReplace(archivePath, execPath, rel.TagName, out)
}

func extractAndReplace(archivePath, execPath, newVersion string, w any) error {
	tmpDir := filepath.Dir(archivePath)
	binName := "soundcheck"
	if runtime.GOOS == "windows" {
		binName = "soundcheck.exe"
	}
	extractedBin := filepath.Join(tmpDir, "new-"+binName)

	if strings.HasSuffix(archivePath, ".zip") {
		if err := upgrade.ExtractZip(archivePath, binName, extractedBin); err != nil {
			return err
		}
	} else {
		if err := upgrade.ExtractTarGz(archivePath, binName, extractedBin); err != nil {
			return err
		}
	}

	if err := upgrade.AtomicReplace(extractedBin, execPath); err != nil {
		return fmt.Errorf("atomic replace: %w", err)
	}
	if writer, ok := w.(interface{ Write([]byte) (int, error) }); ok {
		_, _ = fmt.Fprintf(writer, "==> Successfully upgraded Soundcheck to %s!\n", newVersion)
	}
	return nil
}
