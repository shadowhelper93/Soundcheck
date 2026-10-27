//go:build !windows

package platform

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func lookPath(file string) (string, error) {
	if filepath.IsAbs(file) || strings.ContainsRune(file, filepath.Separator) {
		return exec.LookPath(file)
	}
	if path, err := exec.LookPath(file); err == nil {
		return path, nil
	}
	for _, dir := range candidateDirs() {
		full := filepath.Join(dir, file)
		if fi, err := os.Stat(full); err == nil && !fi.IsDir() && (fi.Mode()&0111 != 0) {
			return full, nil
		}
	}
	return "", exec.ErrNotFound
}

func nvmNodeDirs(home string) []string {
	var dirs []string
	nvmDir := filepath.Join(home, ".nvm", "versions", "node")
	entries, err := os.ReadDir(nvmDir)
	if err != nil {
		return nil
	}
	for i := len(entries) - 1; i >= 0; i-- {
		if entries[i].IsDir() {
			dirs = append(dirs, filepath.Join(nvmDir, entries[i].Name(), "bin"))
		}
	}
	return dirs
}

func candidateDirs() []string {
	var dirs []string
	home, _ := os.UserHomeDir()
	if home != "" {
		dirs = append(dirs,
			filepath.Join(home, ".local", "bin"),
			filepath.Join(home, ".cargo", "bin"),
			filepath.Join(home, "go", "bin"),
			filepath.Join(home, ".yarn", "bin"),
			filepath.Join(home, ".config", "yarn", "global", "node_modules", ".bin"),
		)
		dirs = append(dirs, nvmNodeDirs(home)...)
	}
	return append(dirs,
		"/opt/homebrew/bin",
		"/opt/homebrew/sbin",
		"/usr/local/bin",
		"/usr/local/go/bin",
	)
}

func enrichEnv(env []string) []string {
	pathIdx := -1
	currentPath := ""
	for i, entry := range env {
		if strings.HasPrefix(entry, "PATH=") {
			pathIdx = i
			currentPath = entry[5:]
			break
		}
	}
	dirs := candidateDirs()
	newPath := appendExistingDirs(currentPath, dirs)
	if newPath == currentPath {
		return env
	}
	res := make([]string, len(env))
	copy(res, env)
	if pathIdx >= 0 {
		res[pathIdx] = "PATH=" + newPath
		return res
	}
	return append(res, "PATH="+newPath)
}

func appendExistingDirs(currentPath string, dirs []string) string {
	parts := strings.Split(currentPath, string(os.PathListSeparator))
	existing := make(map[string]bool, len(parts))
	for _, p := range parts {
		if p != "" {
			existing[p] = true
		}
	}
	var added []string
	for _, d := range dirs {
		if !existing[d] {
			if fi, err := os.Stat(d); err == nil && fi.IsDir() {
				added = append(added, d)
				existing[d] = true
			}
		}
	}
	if len(added) == 0 {
		return currentPath
	}
	if len(currentPath) > 0 {
		return currentPath + string(os.PathListSeparator) + strings.Join(added, string(os.PathListSeparator))
	}
	return strings.Join(added, string(os.PathListSeparator))
}

func configDir() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "soundcheck"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "soundcheck"), nil
}

func cacheDir() (string, error) {
	if xdg := os.Getenv("XDG_CACHE_HOME"); xdg != "" {
		return filepath.Join(xdg, "soundcheck"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".cache", "soundcheck"), nil
}
