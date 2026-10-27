package exec

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/soundcheck/soundcheck/internal/edge/platform"
)

// DefaultMaxOutputBytes is the default buffer cap per stream (1MB).
const DefaultMaxOutputBytes = 1024 * 1024

type eventWriter struct {
	ctx       context.Context
	eventType EventType
	ch        chan<- Event
}

func (w *eventWriter) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	chunk := make([]byte, len(p))
	copy(chunk, p)
	select {
	case w.ch <- Event{Type: w.eventType, Data: chunk}:
		return len(p), nil
	case <-w.ctx.Done():
		return 0, w.ctx.Err()
	}
}

// PipeRunner executes commands using OS pipes with process group isolation and buffer caps.
type PipeRunner struct {
	maxOutputBytes int
}

// NewPipeRunner creates a PipeRunner with an optional output buffer cap.
func NewPipeRunner(maxBytes ...int) *PipeRunner {
	limit := DefaultMaxOutputBytes
	if len(maxBytes) > 0 && maxBytes[0] > 0 {
		limit = maxBytes[0]
	}
	return &PipeRunner{maxOutputBytes: limit}
}

func (p *PipeRunner) buildCmd(ctx context.Context, cmd Cmd, ch chan<- Event) *exec.Cmd {
	bin := cmd.Binary
	if resolved, err := platform.LookPath(cmd.Binary); err == nil {
		bin = resolved
	}
	// #nosec G204
	osCmd := exec.CommandContext(ctx, bin, cmd.Args...)
	if cmd.Dir != "" {
		osCmd.Dir = cmd.Dir
	}
	env := os.Environ()
	if len(cmd.Env) > 0 {
		env = append(env, cmd.Env...)
	}
	osCmd.Env = platform.EnrichEnv(env)
	if cmd.Stdin != nil {
		osCmd.Stdin = cmd.Stdin
	} else {
		osCmd.Stdin = strings.NewReader("")
	}
	osCmd.Stdout = &eventWriter{ctx: ctx, eventType: EventStdout, ch: ch}
	osCmd.Stderr = &eventWriter{ctx: ctx, eventType: EventStderr, ch: ch}
	osCmd.WaitDelay = 2 * time.Second
	prepareCmd(osCmd)
	osCmd.Cancel = func() error {
		return killProcessGroup(osCmd)
	}
	return osCmd
}

func streamProcess(execCtx context.Context, osCmd *exec.Cmd, cancel context.CancelFunc, ch chan Event) {
	defer cancel()
	defer close(ch)

	waitErr := osCmd.Wait()
	timedOut := errors.Is(execCtx.Err(), context.DeadlineExceeded)
	killed := execCtx.Err() != nil
	exitCode := 0
	if exitErr, ok := waitErr.(*exec.ExitError); ok {
		exitCode = exitErr.ExitCode()
	} else if waitErr != nil {
		exitCode = -1
	}
	ev := Event{
		Type:     EventExit,
		ExitCode: exitCode,
		Error:    waitErr,
		TimedOut: timedOut,
		Killed:   killed,
	}
	select {
	case ch <- ev:
	default:
		select {
		case ch <- ev:
		case <-execCtx.Done():
		}
	}
}

// Stream begins executing a command and returns a channel of live streaming events.
func (p *PipeRunner) Stream(ctx context.Context, cmd Cmd) (<-chan Event, error) {
	if cmd.Binary == "" {
		return nil, errors.New("exec: binary is required")
	}
	execCtx, cancel := context.WithCancel(ctx)
	if cmd.Timeout > 0 {
		execCtx, cancel = context.WithTimeout(ctx, cmd.Timeout)
	}

	ch := make(chan Event, 64)
	osCmd := p.buildCmd(execCtx, cmd, ch)
	if err := osCmd.Start(); err != nil {
		cancel()
		return nil, err
	}

	go streamProcess(execCtx, osCmd, cancel, ch)
	return ch, nil
}

// Run executes a command to completion and returns the aggregate result.
func (p *PipeRunner) Run(ctx context.Context, cmd Cmd) (Result, error) {
	start := time.Now()
	events, err := p.Stream(ctx, cmd)
	if err != nil {
		return Result{ExitCode: -1, Duration: time.Since(start)}, err
	}

	stdoutRing := newRingBuffer(p.maxOutputBytes)
	stderrRing := newRingBuffer(p.maxOutputBytes)
	res := Result{ExitCode: 0}

	for ev := range events {
		switch ev.Type {
		case EventStdout:
			_, _ = stdoutRing.Write(ev.Data)
		case EventStderr:
			_, _ = stderrRing.Write(ev.Data)
		case EventExit:
			res.ExitCode = ev.ExitCode
			res.TimedOut = ev.TimedOut
			res.Killed = ev.Killed
		}
	}

	res.Stdout = stdoutRing.Bytes()
	res.Stderr = stderrRing.Bytes()
	res.Duration = time.Since(start)
	return res, nil
}
