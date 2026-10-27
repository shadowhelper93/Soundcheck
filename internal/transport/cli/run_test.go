package cli_test

import (
	"strings"
	"testing"

	"github.com/soundcheck/soundcheck/internal/transport/cli"
)

func TestCLIRun_Help(t *testing.T) {
	a := createTestApp(t)
	code, out, _ := executeCommand(a, "run", "--help")
	if code != cli.ExitOK {
		t.Fatalf("expected ExitOK (0), got %d", code)
	}
	if !strings.Contains(out, "run <adapter> [-- <args...>]") {
		t.Fatalf("expected usage in help output, got: %s", out)
	}
}

func TestCLIRun_MissingAdapter(t *testing.T) {
	a := createTestApp(t)
	code, _, _ := executeCommand(a, "run")
	if code != cli.ExitUsage {
		t.Fatalf("expected ExitUsage (3), got %d", code)
	}
}

func TestCLIRun_Success(t *testing.T) {
	a := createTestApp(t)
	fakecli := getFakeCLI(t)
	code, out, _ := executeCommand(a, "run", fakecli, "--", "--stdout=hello from pty\n")
	if code != cli.ExitOK {
		t.Fatalf("expected ExitOK (0), got %d", code)
	}
	if !strings.Contains(out, "hello from pty") {
		t.Fatalf("expected 'hello from pty' in stdout, got: %q", out)
	}
}

func TestCLIRun_ExitCode(t *testing.T) {
	a := createTestApp(t)
	fakecli := getFakeCLI(t)
	code, _, _ := executeCommand(a, "run", fakecli, "--", "--exit=42")
	if code != 42 {
		t.Fatalf("expected exit code 42, got %d", code)
	}
}

func TestCLIRun_InteractivePrompt(t *testing.T) {
	a := createTestApp(t)
	fakecli := getFakeCLI(t)
	in := strings.NewReader("y\n")
	code, out, _ := executeCommandWithIn(a, in, "run", fakecli, "--", "--quirk=interactive")
	if code != cli.ExitOK {
		t.Fatalf("expected ExitOK (0), got %d, out: %s", code, out)
	}
	if !strings.Contains(out, "CONFIRMED") {
		t.Fatalf("expected output to contain 'CONFIRMED', got: %s", out)
	}
}

func TestCLIRun_NonexistentBinary(t *testing.T) {
	a := createTestApp(t)
	code, _, _ := executeCommand(a, "run", "/path/to/nonexistent/binary/12345")
	if code != cli.ExitFailed {
		t.Fatalf("expected ExitFailed (2), got %d", code)
	}
}
