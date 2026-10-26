//go:build windows

package exec_test

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

func isProcessAlive(pid int) bool {
	// #nosec G204
	cmd := exec.Command("tasklist", "/FI", fmt.Sprintf("PID eq %d", pid))
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), strconv.Itoa(pid))
}
