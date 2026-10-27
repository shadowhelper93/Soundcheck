//go:build windows

package exec

import (
	"io"
	"os/exec"
)

type windowsPty struct {
	stdin  io.WriteCloser
	stdout io.ReadCloser
}

func (w *windowsPty) Read(p []byte) (int, error) {
	return w.stdout.Read(p)
}

func (w *windowsPty) Write(p []byte) (int, error) {
	return w.stdin.Write(p)
}

func (w *windowsPty) Close() error {
	err1 := w.stdin.Close()
	err2 := w.stdout.Close()
	if err1 != nil {
		return err1
	}
	return err2
}

func startPTY(cmd *exec.Cmd, _ Winsize) (ptyHandle, error) {
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return nil, err
	}
	cmd.Stderr = cmd.Stdout
	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		_ = stdout.Close()
		return nil, err
	}
	return &windowsPty{stdin: stdin, stdout: stdout}, nil
}

func setPTYSize(_ ptyHandle, _ Winsize) error {
	return nil
}
