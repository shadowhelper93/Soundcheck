// Package upgrade provides self-update inspection, download, and atomic replacement.
package upgrade

import (
	"strings"
)

// PackageManager identifies an external package manager managing the binary.
type PackageManager string

const (
	// PackageManagerNone indicates the binary is not managed by an external package manager.
	PackageManagerNone PackageManager = ""
	// PackageManagerBrew indicates the binary is managed by Homebrew.
	PackageManagerBrew PackageManager = "brew"
	// PackageManagerScoop indicates the binary is managed by Scoop.
	PackageManagerScoop PackageManager = "scoop"
)

// DetectPackageManager checks if the binary path is managed by Homebrew or Scoop.
func DetectPackageManager(execPath string) PackageManager {
	lower := strings.ToLower(execPath)
	if strings.Contains(lower, "cellar") || strings.Contains(lower, "/homebrew/") || strings.Contains(lower, `\homebrew\`) {
		return PackageManagerBrew
	}
	if strings.Contains(lower, "scoop/apps") || strings.Contains(lower, `scoop\apps`) {
		return PackageManagerScoop
	}
	return PackageManagerNone
}

// IsNewerVersion returns true if remote tag is strictly newer than current version.
func IsNewerVersion(current, remote string) bool {
	c := strings.TrimPrefix(strings.TrimSpace(current), "v")
	r := strings.TrimPrefix(strings.TrimSpace(remote), "v")
	if c == "" || c == "dev" {
		return false
	}
	return c != r
}
