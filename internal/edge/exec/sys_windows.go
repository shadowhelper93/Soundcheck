//go:build windows

package exec

import (
	"os/exec"
	"strconv"
)

func prepareCmd(cmd *exec.Cmd) {
}

func killProcessGroup(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil || cmd.Process.Pid <= 0 {
		return nil
	}
	pid := cmd.Process.Pid
	// #nosec G204
	killCmd := exec.Command("taskkill", "/F", "/T", "/PID", strconv.Itoa(pid))
	if err := killCmd.Run(); err == nil {
		return nil
	}
	return cmd.Process.Kill()
}
