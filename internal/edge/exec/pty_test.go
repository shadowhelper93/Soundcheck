package exec_test

import (
	"bytes"
	"context"
	"runtime"
	"strings"
	"testing"
	"time"

	"go.uber.org/goleak"

	edgeexec "github.com/soundcheck/soundcheck/internal/edge/exec"
)

func waitForOutput(events <-chan edgeexec.Event, expected string, timeout time.Duration) bool {
	deadline := time.After(timeout)
	for {
		select {
		case ev, ok := <-events:
			if !ok {
				return false
			}
			if strings.Contains(string(ev.Data), expected) {
				return true
			}
		case <-deadline:
			return false
		}
	}
}

func TestPTYRunner_InteractivePrompt(t *testing.T) {
	defer goleak.VerifyNone(t)

	bin := buildFakeCLI(t)
	runner := edgeexec.NewPTYRunner()

	cmd := edgeexec.Cmd{
		Binary: bin,
		Args:   []string{"--quirk=interactive"},
	}
	sz := edgeexec.Winsize{Cols: 80, Rows: 24}
	session, err := runner.Start(context.Background(), cmd, sz)
	if err != nil {
		t.Fatalf("failed to start pty: %v", err)
	}
	defer func() { _ = session.Close() }()

	if session.ID() == "" {
		t.Fatalf("expected non-empty session ID")
	}

	if !waitForOutput(session.Events(), "Proceed? [y/N]:", 5*time.Second) {
		t.Fatalf("timed out waiting for interactive prompt")
	}

	if _, err := session.Write([]byte("y\n")); err != nil {
		t.Fatalf("failed to write to pty: %v", err)
	}

	if !waitForOutput(session.Events(), "CONFIRMED", 5*time.Second) {
		t.Fatalf("timed out waiting for confirmation")
	}

	exitCode, err := session.Wait()
	if err != nil {
		t.Fatalf("unexpected wait error: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}

	history := session.History()
	if !bytes.Contains(history, []byte("CONFIRMED")) {
		t.Fatalf("expected history to contain 'CONFIRMED', got %q", string(history))
	}
}

func TestPTYRunner_Resize(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("pty resize signals not supported on windows")
	}
	defer goleak.VerifyNone(t)

	bin := buildFakeCLI(t)
	runner := edgeexec.NewPTYRunner()

	cmd := edgeexec.Cmd{
		Binary: bin,
		Args:   []string{"--quirk=interactive_resize"},
	}
	sz := edgeexec.Winsize{Cols: 80, Rows: 24}
	session, err := runner.Start(context.Background(), cmd, sz)
	if err != nil {
		t.Fatalf("failed to start pty: %v", err)
	}
	defer func() { _ = session.Close() }()

	if !waitForOutput(session.Events(), "ready for resize", 5*time.Second) {
		t.Fatalf("timed out waiting for resize readiness")
	}

	if err := session.Resize(120, 40); err != nil {
		t.Fatalf("failed to resize pty: %v", err)
	}

	if !waitForOutput(session.Events(), "RESIZED", 5*time.Second) {
		t.Fatalf("timed out waiting for resize confirmation")
	}

	if _, err := session.Write([]byte("y\n")); err != nil {
		t.Fatalf("failed to write to pty: %v", err)
	}

	exitCode, err := session.Wait()
	if err != nil {
		t.Fatalf("unexpected wait error: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}
}

func TestPTYRunner_CleanTermination(t *testing.T) {
	defer goleak.VerifyNone(t)

	bin := buildFakeCLI(t)
	runner := edgeexec.NewPTYRunner()

	cmd := edgeexec.Cmd{
		Binary: bin,
		Args:   []string{"--exit=42", "--stdout=process complete\n"},
	}
	session, err := runner.Start(context.Background(), cmd, edgeexec.Winsize{})
	if err != nil {
		t.Fatalf("failed to start pty: %v", err)
	}
	defer func() { _ = session.Close() }()

	var lastExitEvent edgeexec.Event
	for ev := range session.Events() {
		if ev.Type == edgeexec.EventExit {
			lastExitEvent = ev
		}
	}

	exitCode, err := session.Wait()
	if exitCode != 42 {
		t.Fatalf("expected exit code 42, got %d (err: %v)", exitCode, err)
	}
	if lastExitEvent.ExitCode != 42 {
		t.Fatalf("expected EventExit to have exit code 42, got %d", lastExitEvent.ExitCode)
	}
}

func TestPTYRunner_CloseKillsHangingProcess(t *testing.T) {
	defer goleak.VerifyNone(t)

	bin := buildFakeCLI(t)
	runner := edgeexec.NewPTYRunner()

	cmd := edgeexec.Cmd{
		Binary: bin,
		Args:   []string{"--quirk=hang"},
	}
	session, err := runner.Start(context.Background(), cmd, edgeexec.Winsize{Cols: 80, Rows: 24})
	if err != nil {
		t.Fatalf("failed to start pty: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	if err := session.Close(); err != nil {
		t.Fatalf("failed to close session: %v", err)
	}

	done := make(chan struct{})
	go func() {
		_, _ = session.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatalf("timed out waiting for session.Wait() after Close()")
	}
}

func TestPTYRunner_RingBufferCap(t *testing.T) {
	defer goleak.VerifyNone(t)

	bin := buildFakeCLI(t)
	const bufCap = 64
	runner := edgeexec.NewPTYRunner(edgeexec.WithPTYBufferSize(bufCap))

	longOutput := strings.Repeat("0123456789", 20) // 200 bytes
	cmd := edgeexec.Cmd{
		Binary: bin,
		Args:   []string{"--stdout=" + longOutput},
	}
	session, err := runner.Start(context.Background(), cmd, edgeexec.Winsize{Cols: 80, Rows: 24})
	if err != nil {
		t.Fatalf("failed to start pty: %v", err)
	}
	defer func() { _ = session.Close() }()

	_, _ = session.Wait()

	history := session.History()
	if len(history) != bufCap {
		t.Fatalf("expected history length %d, got %d", bufCap, len(history))
	}
}

func TestPTYRunner_Validation(t *testing.T) {
	runner := edgeexec.NewPTYRunner()
	_, err := runner.Start(context.Background(), edgeexec.Cmd{}, edgeexec.Winsize{})
	if err == nil {
		t.Fatalf("expected error for empty binary")
	}
}

func TestPTYRunner_ConcurrentResizeOnExit(t *testing.T) {
	defer goleak.VerifyNone(t)

	bin := buildFakeCLI(t)
	runner := edgeexec.NewPTYRunner()

	cmd := edgeexec.Cmd{
		Binary: bin,
		Args:   []string{"--stdout=hello"},
	}
	session, err := runner.Start(context.Background(), cmd, edgeexec.Winsize{Cols: 80, Rows: 24})
	if err != nil {
		t.Fatalf("failed to start pty: %v", err)
	}
	defer func() { _ = session.Close() }()

	done := make(chan struct{})
	for i := uint16(0); i < 8; i++ {
		go func(id uint16) {
			for {
				select {
				case <-done:
					return
				default:
					_ = session.Resize(80+id, 24+id)
				}
			}
		}(i)
	}

	for range session.Events() {
	}

	exitCode, err := session.Wait()
	close(done)
	if err != nil {
		t.Fatalf("unexpected wait error: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}
}

func TestPTYRunner_BackgroundChildDoesNotBlockWait(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping background process test on windows")
	}
	defer goleak.VerifyNone(t)

	runner := edgeexec.NewPTYRunner()
	cmd := edgeexec.Cmd{
		Binary: "/bin/sh",
		Args:   []string{"-c", "(sleep 5 &); exit 0"},
	}
	session, err := runner.Start(context.Background(), cmd, edgeexec.Winsize{Cols: 80, Rows: 24})
	if err != nil {
		t.Fatalf("failed to start pty: %v", err)
	}
	defer func() { _ = session.Close() }()

	waitDone := make(chan struct{})
	go func() {
		_, _ = session.Wait()
		close(waitDone)
	}()

	select {
	case <-waitDone:
	case <-time.After(2 * time.Second):
		t.Fatalf("session.Wait() hung on background child process")
	}
}
