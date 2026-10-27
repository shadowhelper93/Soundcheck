package exec

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/soundcheck/soundcheck/internal/edge/platform"
)

// DefaultPTYBufferSize is the default ring buffer capacity for terminal history (64KB).
const DefaultPTYBufferSize = 64 * 1024

// Winsize describes terminal window dimensions.
type Winsize struct {
	Cols uint16
	Rows uint16
	X    uint16
	Y    uint16
}

type ptyHandle interface {
	io.ReadWriteCloser
}

// PTYSession represents an active interactive pseudoterminal session.
type PTYSession interface {
	ID() string
	Write(p []byte) (int, error)
	Resize(cols, rows uint16) error
	Close() error
	Wait() (int, error)
	Events() <-chan Event
	History() []byte
}

// PTYRunner manages the lifecycle and creation of PTY sessions.
type PTYRunner interface {
	Start(ctx context.Context, cmd Cmd, sz Winsize) (PTYSession, error)
}

type ptyRunner struct {
	bufSize int
}

// PTYOption configures PTYRunner settings.
type PTYOption func(*ptyRunner)

// WithPTYBufferSize sets the capacity of the terminal history ring buffer.
func WithPTYBufferSize(size int) PTYOption {
	return func(r *ptyRunner) {
		if size > 0 {
			r.bufSize = size
		}
	}
}

// NewPTYRunner creates a new PTYRunner.
func NewPTYRunner(opts ...PTYOption) PTYRunner {
	r := &ptyRunner{bufSize: DefaultPTYBufferSize}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

func (r *ptyRunner) Start(ctx context.Context, cmd Cmd, sz Winsize) (PTYSession, error) {
	if cmd.Binary == "" {
		return nil, errors.New("exec: binary is required")
	}
	if sz.Cols == 0 {
		sz.Cols = 80
	}
	if sz.Rows == 0 {
		sz.Rows = 24
	}
	execCtx, cancel := context.WithCancel(ctx)
	if cmd.Timeout > 0 {
		execCtx, cancel = context.WithTimeout(ctx, cmd.Timeout)
	}
	osCmd := buildOSCmd(execCtx, cmd)
	handle, err := startPTY(osCmd, sz)
	if err != nil {
		cancel()
		return nil, err
	}
	return newPTYSession(execCtx, osCmd, handle, sz, r.bufSize, cancel), nil
}

func buildOSCmd(ctx context.Context, cmd Cmd) *exec.Cmd {
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
	osCmd.Cancel = func() error {
		return killProcessGroup(osCmd)
	}
	return osCmd
}

type ptySession struct {
	id        string
	ctx       context.Context
	cmd       *exec.Cmd
	ptmx      ptyHandle
	size      Winsize
	ringBuf   *ringBuffer
	events    chan Event
	done      chan struct{}
	cancel    context.CancelFunc
	closeOnce sync.Once
	mu        sync.RWMutex
	exitCode  int
	waitErr   error
	closed    bool
}

func newPTYSession(ctx context.Context, cmd *exec.Cmd, h ptyHandle, sz Winsize, bufCap int, cancel context.CancelFunc) *ptySession {
	sess := &ptySession{
		id:      generateSessionID(),
		ctx:     ctx,
		cmd:     cmd,
		ptmx:    h,
		size:    sz,
		ringBuf: newRingBuffer(bufCap),
		events:  make(chan Event, 1024),
		done:    make(chan struct{}),
		cancel:  cancel,
	}
	readDone := make(chan struct{})
	go sess.readLoop(readDone)
	go sess.waitLoop(readDone)
	return sess
}

func (s *ptySession) ID() string {
	return s.id
}

func (s *ptySession) Events() <-chan Event {
	return s.events
}

func (s *ptySession) History() []byte {
	return s.ringBuf.Bytes()
}

func (s *ptySession) Write(p []byte) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.closed {
		return 0, errors.New("pty: session closed")
	}
	return s.ptmx.Write(p)
}

func (s *ptySession) Resize(cols, rows uint16) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return errors.New("pty: session closed")
	}
	s.size = Winsize{Cols: cols, Rows: rows}
	if err := setPTYSize(s.ptmx, s.size); err != nil {
		return err
	}
	s.sendEvent(Event{
		Type: EventResize,
		Data: []byte(fmt.Sprintf("%dx%d", cols, rows)),
	})
	return nil
}

func (s *ptySession) Close() error {
	var err error
	s.closeOnce.Do(func() {
		s.mu.Lock()
		s.closed = true
		if s.ptmx != nil {
			if cerr := s.ptmx.Close(); cerr != nil && !errors.Is(cerr, os.ErrClosed) {
				err = cerr
			}
		}
		s.mu.Unlock()

		if s.cancel != nil {
			s.cancel()
		}
		if s.cmd != nil && s.cmd.Process != nil {
			_ = killProcessGroup(s.cmd)
		}
	})
	return err
}

func (s *ptySession) Wait() (int, error) {
	<-s.done
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.exitCode, s.waitErr
}

func (s *ptySession) readLoop(readDone chan<- struct{}) {
	defer close(readDone)
	buf := make([]byte, 4096)
	for {
		n, err := s.ptmx.Read(buf)
		if n > 0 {
			chunk := make([]byte, n)
			copy(chunk, buf[:n])
			_, _ = s.ringBuf.Write(chunk)
			s.sendEvent(Event{Type: EventOutput, Data: chunk})
		}
		if err != nil {
			break
		}
	}
}

func (s *ptySession) waitLoop(readDone <-chan struct{}) {
	waitErr := s.cmd.Wait()

	s.mu.Lock()
	s.closed = true
	if s.ptmx != nil {
		_ = s.ptmx.Close()
	}
	s.mu.Unlock()

	<-readDone

	code := procExitCode(waitErr)
	s.sendExitEvent(Event{
		Type:     EventExit,
		ExitCode: code,
		Error:    waitErr,
		TimedOut: errors.Is(s.ctx.Err(), context.DeadlineExceeded),
		Killed:   s.ctx.Err() != nil,
	})

	s.mu.Lock()
	s.exitCode = code
	s.waitErr = waitErr
	close(s.events)
	s.mu.Unlock()

	close(s.done)
}

func (s *ptySession) sendEvent(ev Event) {
	select {
	case s.events <- ev:
	default:
	}
}

func (s *ptySession) sendExitEvent(ev Event) {
	select {
	case s.events <- ev:
	case <-time.After(50 * time.Millisecond):
	}
}

func procExitCode(err error) int {
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	return -1
}

func generateSessionID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return fmt.Sprintf("pty-%x", b)
}
