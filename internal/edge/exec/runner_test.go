package exec_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	edgeexec "github.com/soundcheck/soundcheck/internal/edge/exec"
)

var (
	fakecliOnce sync.Once
	fakecliPath string
	fakecliErr  error
)

func findRepoRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		return "."
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "."
		}
		dir = parent
	}
}

func buildFakeCLI(t *testing.T) string {
	t.Helper()
	fakecliOnce.Do(func() {
		tmpDir, err := os.MkdirTemp("", "fakecli-bin-*")
		if err != nil {
			fakecliErr = err
			return
		}
		bin := filepath.Join(tmpDir, "fakecli")
		if runtime.GOOS == "windows" {
			bin += ".exe"
		}
		root := findRepoRoot()
		// #nosec G204
		cmd := exec.Command("go", "build", "-o", bin, "./testdata/fakecli")
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			fakecliErr = fmt.Errorf("build fakecli failed: %v: %s", err, string(out))
			return
		}
		fakecliPath = bin
	})
	if fakecliErr != nil {
		t.Fatalf("failed to build fakecli: %v", fakecliErr)
	}
	return fakecliPath
}

func TestPipeRunner_Success(t *testing.T) {
	bin := buildFakeCLI(t)
	runner := edgeexec.NewPipeRunner()

	cmd := edgeexec.Cmd{
		Binary: bin,
		Args:   []string{"--stdout=hello world"},
	}
	res, err := runner.Run(context.Background(), cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.ExitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", res.ExitCode)
	}
	if string(res.Stdout) != "hello world" {
		t.Fatalf("expected 'hello world', got %q", string(res.Stdout))
	}
	if res.TimedOut || res.Killed {
		t.Fatalf("expected TimedOut/Killed false, got TimedOut=%v Killed=%v", res.TimedOut, res.Killed)
	}
	if res.Duration <= 0 {
		t.Fatalf("expected non-zero duration")
	}
}

func TestPipeRunner_NonZeroExit(t *testing.T) {
	bin := buildFakeCLI(t)
	runner := edgeexec.NewPipeRunner()

	cmd := edgeexec.Cmd{
		Binary: bin,
		Args:   []string{"--exit=41", "--stderr=FatalAuthenticationError: invalid key"},
	}
	res, err := runner.Run(context.Background(), cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.ExitCode != 41 {
		t.Fatalf("expected exit code 41, got %d", res.ExitCode)
	}
	if !strings.Contains(string(res.Stderr), "FatalAuthenticationError") {
		t.Fatalf("expected stderr to contain error, got %q", string(res.Stderr))
	}
	if res.TimedOut || res.Killed {
		t.Fatalf("expected TimedOut/Killed false, got TimedOut=%v Killed=%v", res.TimedOut, res.Killed)
	}
}

func TestPipeRunner_OutputCap(t *testing.T) {
	bin := buildFakeCLI(t)
	const capBytes = 16 * 1024 // 16KB
	runner := edgeexec.NewPipeRunner(capBytes)

	cmd := edgeexec.Cmd{
		Binary: bin,
		Args:   []string{"--quirk=noisy"},
	}
	res, err := runner.Run(context.Background(), cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Stdout) != capBytes {
		t.Fatalf("expected output capped at %d bytes, got %d", capBytes, len(res.Stdout))
	}
}

func TestPipeRunner_Streaming(t *testing.T) {
	bin := buildFakeCLI(t)
	runner := edgeexec.NewPipeRunner()

	cmd := edgeexec.Cmd{
		Binary: bin,
		Args:   []string{"--stdout=stream data"},
	}
	events, err := runner.Stream(context.Background(), cmd)
	if err != nil {
		t.Fatalf("stream unexpected error: %v", err)
	}

	var stdoutChunks []string
	var exitReceived bool
	for ev := range events {
		if ev.Type == edgeexec.EventStdout {
			stdoutChunks = append(stdoutChunks, string(ev.Data))
		}
		if ev.Type == edgeexec.EventExit {
			exitReceived = true
			if ev.ExitCode != 0 {
				t.Fatalf("expected exit code 0, got %d", ev.ExitCode)
			}
		}
	}

	if !exitReceived {
		t.Fatal("expected EventExit")
	}
	fullStdout := strings.Join(stdoutChunks, "")
	if fullStdout != "stream data" {
		t.Fatalf("expected 'stream data', got %q", fullStdout)
	}
}

func TestPipeRunner_ContextTimeout(t *testing.T) {
	bin := buildFakeCLI(t)
	runner := edgeexec.NewPipeRunner()

	cmd := edgeexec.Cmd{
		Binary:  bin,
		Args:    []string{"--quirk=hang"},
		Timeout: 100 * time.Millisecond,
	}
	res, err := runner.Run(context.Background(), cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.TimedOut {
		t.Fatal("expected res.TimedOut true")
	}
	if !res.Killed {
		t.Fatal("expected res.Killed true")
	}
	if runtime.GOOS != "windows" && res.ExitCode != -1 {
		t.Fatalf("expected exit code -1, got %d", res.ExitCode)
	}
}

func TestPipeRunner_ProcessGroupLeak(t *testing.T) {
	bin := buildFakeCLI(t)
	runner := edgeexec.NewPipeRunner()

	pidFile := filepath.Join(t.TempDir(), "child.pid")
	cmd := edgeexec.Cmd{
		Binary:  bin,
		Args:    []string{"--quirk=spawn_and_hang", "--pid-file=" + pidFile},
		Timeout: 200 * time.Millisecond,
	}

	res, err := runner.Run(context.Background(), cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.TimedOut || !res.Killed {
		t.Fatalf("expected TimedOut=true Killed=true, got TimedOut=%v Killed=%v", res.TimedOut, res.Killed)
	}

	// Read child PID written by fakecli
	// #nosec G304
	data, err := os.ReadFile(filepath.Clean(pidFile))
	if err != nil {
		t.Fatalf("failed to read pidfile: %v", err)
	}
	childPid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		t.Fatalf("invalid child pid in file: %q", string(data))
	}
	if childPid <= 0 {
		t.Fatalf("child pid should be > 0, got %d", childPid)
	}

	// Verify child process has been killed and no zombie/orphan remains
	dead := false
	for i := 0; i < 30; i++ {
		if !isProcessAlive(childPid) {
			dead = true
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !dead {
		t.Fatalf("orphan process detected: child process %d is still running after timeout", childPid)
	}
}

func TestPipeRunner_BinaryNotFound(t *testing.T) {
	runner := edgeexec.NewPipeRunner()
	cmd := edgeexec.Cmd{
		Binary: "non_existent_binary_xyz_12345",
	}
	_, err := runner.Run(context.Background(), cmd)
	if err == nil {
		t.Fatal("expected error for non-existent binary, got nil")
	}
}

func TestPipeRunner_EmptyBinary(t *testing.T) {
	runner := edgeexec.NewPipeRunner()
	cmd := edgeexec.Cmd{}
	_, err := runner.Run(context.Background(), cmd)
	if err == nil {
		t.Fatal("expected error for empty binary, got nil")
	}
}

func TestPipeRunner_WithEnvAndDir(t *testing.T) {
	bin := buildFakeCLI(t)
	runner := edgeexec.NewPipeRunner()
	tmpDir := t.TempDir()

	cmd := edgeexec.Cmd{
		Binary: bin,
		Args:   []string{"--stdout=env_dir_test"},
		Dir:    tmpDir,
		Env:    []string{"TEST_VAR=123"},
	}
	res, err := runner.Run(context.Background(), cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(res.Stdout) != "env_dir_test" {
		t.Fatalf("expected 'env_dir_test', got %q", string(res.Stdout))
	}
}

func TestPipeRunner_QuirksParity(t *testing.T) {
	bin := buildFakeCLI(t)
	runner := edgeexec.NewPipeRunner()

	tests := []struct {
		name         string
		quirk        string
		wantExit     int
		wantInStdout string
		wantInStderr string
	}{
		{
			name:         "claude-79500 rate limit exit 0",
			quirk:        "claude-79500",
			wantExit:     0,
			wantInStdout: "rate_limit_error",
		},
		{
			name:         "codex-doctor-exit0 unauth exit 0",
			quirk:        "codex-doctor-exit0",
			wantExit:     0,
			wantInStdout: "authentication",
		},
		{
			name:         "gemini-base-url-exit41 fatal auth error",
			quirk:        "gemini-base-url-exit41",
			wantExit:     41,
			wantInStderr: "FatalAuthenticationError",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := edgeexec.Cmd{
				Binary: bin,
				Args:   []string{"--quirk=" + tt.quirk},
			}
			res, err := runner.Run(context.Background(), cmd)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res.ExitCode != tt.wantExit {
				t.Fatalf("expected exit code %d, got %d", tt.wantExit, res.ExitCode)
			}
			if tt.wantInStdout != "" && !strings.Contains(string(res.Stdout), tt.wantInStdout) {
				t.Fatalf("expected stdout to contain %q, got %q", tt.wantInStdout, string(res.Stdout))
			}
			if tt.wantInStderr != "" && !strings.Contains(string(res.Stderr), tt.wantInStderr) {
				t.Fatalf("expected stderr to contain %q, got %q", tt.wantInStderr, string(res.Stderr))
			}
		})
	}
}
