//go:build windows

package platform

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func lookPath(file string) (string, error) {
	path, err := exec.LookPath(file)
	if err == nil {
		return path, nil
	}
	pathext := os.Getenv("PATHEXT")
	if pathext == "" {
		pathext = ".COM;.EXE;.BAT;.CMD"
	}
	for _, ext := range strings.Split(pathext, ";") {
		if ext == "" {
			continue
		}
		if p, err := exec.LookPath(file + ext); err == nil {
			return p, nil
		}
	}
	return "", err
}

func configDir() (string, error) {
	if appData := os.Getenv("APPDATA"); appData != "" {
		return filepath.Join(appData, "soundcheck"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "AppData", "Roaming", "soundcheck"), nil
}

func cacheDir() (string, error) {
	if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
		return filepath.Join(localAppData, "soundcheck"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "AppData", "Local", "soundcheck"), nil
}

func enrichEnv(env []string) []string {
	return env
}
