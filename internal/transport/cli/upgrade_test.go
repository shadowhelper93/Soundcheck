package cli_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/soundcheck/soundcheck/internal/transport/cli"
)

func TestCLIUpgrade_Help(t *testing.T) {
	var stdout, stderr bytes.Buffer
	cfg := cli.ExecutionConfig{
		Args: []string{"upgrade", "--help"},
		Out:  &stdout,
		Err:  &stderr,
	}

	exitCode := cli.ExecuteCustom(context.Background(), cfg)
	if exitCode != cli.ExitOK {
		t.Fatalf("expected exit code %d, got %d", cli.ExitOK, exitCode)
	}

	output := stdout.String()
	if !strings.Contains(output, "upgrade [flags]") && !strings.Contains(output, "Check for updates") {
		t.Errorf("unexpected help output: %s", output)
	}
	if !strings.Contains(output, "--check") {
		t.Errorf("help missing --check flag: %s", output)
	}
}
