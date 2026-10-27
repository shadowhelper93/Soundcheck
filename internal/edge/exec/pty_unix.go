//go:build !windows

package exec

import (
	"errors"
	"os"
	"os/exec"

	"github.com/creack/pty"
)

func startPTY(cmd *exec.Cmd, sz Winsize) (ptyHandle, error) {
	ws := &pty.Winsize{
		Rows: sz.Rows,
		Cols: sz.Cols,
		X:    sz.X,
		Y:    sz.Y,
	}
	return pty.StartWithSize(cmd, ws)
}

func setPTYSize(h ptyHandle, sz Winsize) error {
	f, ok := h.(*os.File)
	if !ok {
		return errors.New("exec: handle is not an os.File")
	}
	ws := &pty.Winsize{
		Rows: sz.Rows,
		Cols: sz.Cols,
		X:    sz.X,
		Y:    sz.Y,
	}
	return pty.Setsize(f, ws)
}
