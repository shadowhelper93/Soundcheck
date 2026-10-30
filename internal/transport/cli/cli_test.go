package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/soundcheck/soundcheck/internal/adapters"
	"github.com/soundcheck/soundcheck/internal/app"
	"github.com/soundcheck/soundcheck/internal/core/domain"
	edgeExec "github.com/soundcheck/soundcheck/internal/edge/exec"
	"github.com/soundcheck/soundcheck/internal/transport/cli"
)

var (
	fakecliOnce sync.Once
	fakecliPath string
	fakecliErr  error
)

func getFakeCLI(t testing.TB) string {
	t.Helper()
	fakecliOnce.Do(func() {
		tmpDir, err := os.MkdirTemp("", "fakecli-cli-*")
		if err != nil {
			fakecliErr = err
			return
		}
		bin := filepath.Join(tmpDir, "fakecli")
		root := findModuleRoot(t)
		// #nosec G204
		cmd := exec.Command("go", "build", "-o", bin, filepath.Join(root, "testdata", "fakecli"))
		if out, err := cmd.CombinedOutput(); err != nil {
			fakecliErr = fmt.Errorf("build fakecli: %w: %s", err, string(out))
			return
		}
		fakecliPath = bin
	})
	if fakecliErr != nil {
		t.Fatalf("failed to build fakecli: %v", fakecliErr)
	}
	return fakecliPath
}

func findModuleRoot(t testing.TB) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not locate go.mod")
		}
		dir = parent
	}
}

func testDefinitions(bin string) []adapters.Definition {
	agentIDs := []string{"claude", "codex", "opencode", "agy", "gemini", "copilot"}
	defs := make([]adapters.Definition, len(agentIDs))

	for i, id := range agentIDs {
		orig, _ := adapters.Get(id)
		defs[i] = adapters.Definition{
			Adapter: domain.Adapter{
				ID:              id,
				Name:            orig.Name,
				Vendor:          orig.Vendor,
				Binary:          bin,
				Category:        orig.Category,
				Models:          orig.Models,
				HeadlessCommand: bin,
			},
			ProbeSpecs: map[domain.Tier]domain.Probe{
				domain.TierL0: {Level: domain.TierL0, Command: bin + " --version"},
				domain.TierL1: {Level: domain.TierL1, Command: bin},
				domain.TierL2: {Level: domain.TierL2, Command: bin},
				domain.TierL3: {Level: domain.TierL3, Command: bin},
			},
		}
	}
	return defs
}

func createTestApp(t testing.TB) *app.App {
	t.Helper()
	bin := getFakeCLI(t)
	defs := testDefinitions(bin)
	a, err := app.New(
		app.WithRunner(edgeExec.NewPipeRunner()),
		app.WithDefinitions(defs),
	)
	if err != nil {
		t.Fatalf("failed to create test app: %v", err)
	}
	return a
}

func executeCommand(a *app.App, args ...string) (int, string, string) {
	return executeCommandWithIn(a, nil, args...)
}

func executeCommandWithIn(a *app.App, in io.Reader, args ...string) (int, string, string) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := cli.ExecuteCustom(context.Background(), cli.ExecutionConfig{
		Args: args,
		In:   in,
		Out:  &stdout,
		Err:  &stderr,
		App:  a,
	})
	return code, stdout.String(), stderr.String()
}

func TestCLIVersion(t *testing.T) {
	a := createTestApp(t)

	t.Run("plain version", func(t *testing.T) {
		code, out, _ := executeCommand(a, "version")
		if code != cli.ExitOK {
			t.Fatalf("expected ExitOK (0), got %d", code)
		}
		if !strings.Contains(out, cli.Version) {
			t.Fatalf("expected output to contain %s, got: %s", cli.Version, out)
		}
	})

	t.Run("json version", func(t *testing.T) {
		code, out, _ := executeCommand(a, "version", "--json")
		if code != cli.ExitOK {
			t.Fatalf("expected ExitOK (0), got %d", code)
		}
		var parsed map[string]string
		if err := json.Unmarshal([]byte(out), &parsed); err != nil {
			t.Fatalf("failed to parse JSON: %v", err)
		}
		if parsed["version"] != cli.Version {
			t.Fatalf("expected version %s, got %s", cli.Version, parsed["version"])
		}
	})
}

func TestCLIDoctor(t *testing.T) {
	a := createTestApp(t)

	t.Run("doctor text and exit code", func(t *testing.T) {
		code, out, _ := executeCommand(a, "doctor", "--no-color")
		if code != cli.ExitOK {
			t.Fatalf("expected ExitOK (0), got %d", code)
		}
		if !strings.Contains(out, "SOUNDCHECK PREFLIGHT REPORT") {
			t.Fatalf("expected report header in output, got: %s", out)
		}
		if !strings.Contains(out, "ADAPTERS") {
			t.Fatalf("expected ADAPTERS in output, got: %s", out)
		}
		if !strings.Contains(out, "RESOLVED ROLES") {
			t.Fatalf("expected RESOLVED ROLES in output, got: %s", out)
		}
		if !strings.Contains(out, "claude") {
			t.Fatalf("expected claude in output, got: %s", out)
		}
	})

	t.Run("doctor json", func(t *testing.T) {
		code, out, _ := executeCommand(a, "doctor", "--json")
		if code != cli.ExitOK {
			t.Fatalf("expected ExitOK (0), got %d", code)
		}
		var report app.DoctorReport
		if err := json.Unmarshal([]byte(out), &report); err != nil {
			t.Fatalf("failed to unmarshal JSON: %v", err)
		}
		if report.Summary.TotalAgents != 6 {
			t.Fatalf("expected 6 total agents, got %d", report.Summary.TotalAgents)
		}
		if report.Summary.FunctionalAgents != 6 {
			t.Fatalf("expected 6 functional agents, got %d", report.Summary.FunctionalAgents)
		}
	})
}

func TestCLIDoctorNDJSON(t *testing.T) {
	a := createTestApp(t)
	code, out, _ := executeCommand(a, "doctor", "--ndjson")
	if code != cli.ExitOK {
		t.Fatalf("expected ExitOK (0), got %d", code)
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) < 4 {
		t.Fatalf("expected at least 4 NDJSON lines, got %d", len(lines))
	}
	var resolutionRoles []string
	for _, line := range lines {
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err == nil && m["type"] == "resolution" {
			if r, ok := m["role"].(string); ok {
				resolutionRoles = append(resolutionRoles, r)
			}
		}
	}
	expectedRoles := []string{"primary", "subagent", "auxiliary", "reviewer"}
	if len(resolutionRoles) != len(expectedRoles) {
		t.Fatalf("expected %d resolution roles, got %d: %v", len(expectedRoles), len(resolutionRoles), resolutionRoles)
	}
	for i, role := range expectedRoles {
		if resolutionRoles[i] != role {
			t.Fatalf("expected role %s at index %d, got %s", role, i, resolutionRoles[i])
		}
	}
}

func TestCLIProbe(t *testing.T) {
	a := createTestApp(t)

	t.Run("probe single adapter", func(t *testing.T) {
		code, out, _ := executeCommand(a, "probe", "claude", "--no-color")
		if code != cli.ExitOK {
			t.Fatalf("expected ExitOK (0), got %d", code)
		}
		if !strings.Contains(out, "claude") {
			t.Fatalf("expected claude in output, got: %s", out)
		}
		if !strings.Contains(out, "Tier Meter") {
			t.Fatalf("expected Tier Meter in output, got: %s", out)
		}
	})

	t.Run("probe single adapter json", func(t *testing.T) {
		code, out, _ := executeCommand(a, "probe", "claude", "--json")
		if code != cli.ExitOK {
			t.Fatalf("expected ExitOK (0), got %d", code)
		}
		var ad domain.Adapter
		if err := json.Unmarshal([]byte(out), &ad); err != nil {
			t.Fatalf("failed to unmarshal adapter: %v", err)
		}
		if ad.ID != "claude" {
			t.Fatalf("expected claude, got %s", ad.ID)
		}
	})

	t.Run("probe all adapters", func(t *testing.T) {
		code, out, _ := executeCommand(a, "probe", "--no-color")
		if code != cli.ExitOK {
			t.Fatalf("expected ExitOK (0), got %d", code)
		}
		if !strings.Contains(out, "claude") || !strings.Contains(out, "codex") {
			t.Fatalf("expected all adapters in output, got: %s", out)
		}
	})

	t.Run("probe unknown adapter returns usage error", func(t *testing.T) {
		code, _, _ := executeCommand(a, "probe", "nonexistent-adapter")
		if code != cli.ExitUsage {
			t.Fatalf("expected ExitUsage (3), got %d", code)
		}
	})
}

func TestCLIResolve(t *testing.T) {
	a := createTestApp(t)

	t.Run("resolve all roles", func(t *testing.T) {
		code, out, _ := executeCommand(a, "resolve", "--no-color")
		if code != cli.ExitOK {
			t.Fatalf("expected ExitOK (0), got %d", code)
		}
		if !strings.Contains(out, "primary") || !strings.Contains(out, "HEALTHY") {
			t.Fatalf("expected primary role in output, got: %s", out)
		}
	})

	t.Run("resolve specific role json", func(t *testing.T) {
		code, out, _ := executeCommand(a, "resolve", "primary", "--json")
		if code != cli.ExitOK {
			t.Fatalf("expected ExitOK (0), got %d", code)
		}
		var res map[domain.Role]domain.DetailedResolution
		if err := json.Unmarshal([]byte(out), &res); err != nil {
			t.Fatalf("failed to unmarshal JSON: %v", err)
		}
		if _, ok := res[domain.Role("primary")]; !ok {
			t.Fatalf("expected primary role in resolution")
		}
	})

	t.Run("resolve unknown role", func(t *testing.T) {
		code, _, _ := executeCommand(a, "resolve", "nonexistent-role")
		if code != cli.ExitUsage {
			t.Fatalf("expected ExitUsage (3), got %d", code)
		}
	})
}

func TestCLIManifest(t *testing.T) {
	a := createTestApp(t)

	code, out, _ := executeCommand(a, "manifest")
	if code != cli.ExitOK {
		t.Fatalf("expected ExitOK (0), got %d", code)
	}
	var m domain.Manifest
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatalf("failed to unmarshal JSON manifest: %v", err)
	}
	if m.Summary.TotalAgents != 6 {
		t.Fatalf("expected 6 total agents, got %d", m.Summary.TotalAgents)
	}
}

func TestCLIInvalidConfig(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	tmpFile, err := os.CreateTemp("", "bad-config-*.yaml")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()
	if _, err := tmpFile.WriteString("unknown_field: true\n"); err != nil {
		t.Fatalf("failed to write bad config: %v", err)
	}
	_ = tmpFile.Close()

	code := cli.ExecuteCustom(context.Background(), cli.ExecutionConfig{
		Args: []string{"doctor", "--config", tmpFile.Name()},
		Out:  &stdout,
		Err:  &stderr,
	})
	if code != cli.ExitConfigInvalid {
		t.Fatalf("expected ExitConfigInvalid (4), got %d", code)
	}
	if !strings.Contains(stderr.String(), "config error") {
		t.Fatalf("expected stderr to contain 'config error', got: %s", stderr.String())
	}
}

func TestCLIUsageError(t *testing.T) {
	a := createTestApp(t)

	t.Run("unknown command", func(t *testing.T) {
		code, _, stderr := executeCommand(a, "nonexistent-command")
		if code != cli.ExitUsage {
			t.Fatalf("expected ExitUsage (3), got %d", code)
		}
		if !strings.Contains(stderr, "unknown command") {
			t.Fatalf("expected stderr to contain 'unknown command', got: %s", stderr)
		}
	})

	t.Run("unknown flag", func(t *testing.T) {
		code, _, stderr := executeCommand(a, "version", "--unknown-flag")
		if code != cli.ExitUsage {
			t.Fatalf("expected ExitUsage (3), got %d", code)
		}
		if !strings.Contains(stderr, "unknown flag") {
			t.Fatalf("expected stderr to contain 'unknown flag', got: %s", stderr)
		}
	})
}

func TestCLIDoctorDegradedExitCode(t *testing.T) {
	bin := getFakeCLI(t)
	defs := testDefinitions(bin)
	// Make codex fail TierL1 auth (exit 1)
	defs[1].ProbeSpecs[domain.TierL1] = domain.Probe{
		Level:   domain.TierL1,
		Command: bin + " -exit 1",
	}

	a, err := app.New(
		app.WithRunner(edgeExec.NewPipeRunner()),
		app.WithDefinitions(defs),
	)
	if err != nil {
		t.Fatalf("failed to create test app: %v", err)
	}

	code, _, _ := executeCommand(a, "doctor", "--no-color")
	if code != cli.ExitDegraded {
		t.Fatalf("expected ExitDegraded (1), got %d", code)
	}
}

func TestCLIProbeFailedExitCode(t *testing.T) {
	defs := []adapters.Definition{
		{
			Adapter: domain.Adapter{
				ID:        "broken",
				Name:      "Broken CLI",
				Binary:    "/path/to/nonexistent/binary",
				Installed: false,
			},
			ProbeSpecs: map[domain.Tier]domain.Probe{
				domain.TierL0: {Level: domain.TierL0, Command: "/path/to/nonexistent/binary"},
			},
		},
	}
	a, err := app.New(
		app.WithRunner(edgeExec.NewPipeRunner()),
		app.WithDefinitions(defs),
	)
	if err != nil {
		t.Fatalf("failed to create test app: %v", err)
	}

	code, _, _ := executeCommand(a, "probe", "broken")
	if code != cli.ExitFailed {
		t.Fatalf("expected ExitFailed (2), got %d", code)
	}
}

func TestCLIGoldenSnapshots(t *testing.T) {
	a := createTestApp(t)

	t.Run("manifest golden schema snapshot", func(t *testing.T) {
		code, out, _ := executeCommand(a, "manifest")
		if code != cli.ExitOK {
			t.Fatalf("expected ExitOK, got %d", code)
		}
		var m domain.Manifest
		if err := json.Unmarshal([]byte(out), &m); err != nil {
			t.Fatalf("manifest json parse: %v", err)
		}
		if m.SchemaVersion != "2026.1" {
			t.Errorf("manifest schemaVersion = %q, want 2026.1", m.SchemaVersion)
		}
		if m.GeneratedAt == "" {
			t.Errorf("manifest generatedAt is empty")
		}
		if m.Summary.TotalAgents != 6 || m.Summary.FunctionalAgents != 6 {
			t.Errorf("manifest summary mismatch: %+v", m.Summary)
		}
	})

	t.Run("doctor golden text snapshot", func(t *testing.T) {
		code, out, _ := executeCommand(a, "doctor", "--no-color")
		if code != cli.ExitOK {
			t.Fatalf("expected ExitOK, got %d", code)
		}
		for _, section := range []string{"SOUNDCHECK PREFLIGHT REPORT", "ADAPTERS", "RESOLVED ROLES", "Summary:"} {
			if !strings.Contains(out, section) {
				t.Errorf("expected output to contain %q, got:\n%s", section, out)
			}
		}
	})
}

func TestBinarySubprocessExecution(t *testing.T) {
	root := findModuleRoot(t)
	bin := filepath.Join(root, "bin", "soundcheck")
	if _, err := os.Stat(bin); err != nil {
		t.Skip("binary not yet built, skipping subprocess test")
	}

	t.Run("binary version", func(t *testing.T) {
		cmd := exec.Command(bin, "version")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("binary version failed: %v: %s", err, string(out))
		}
		if !strings.Contains(string(out), "soundcheck") {
			t.Errorf("expected version output to contain soundcheck, got: %s", string(out))
		}
	})

	t.Run("binary usage error", func(t *testing.T) {
		cmd := exec.Command(bin, "unknown-command-xyz")
		_ = cmd.Run()
		if cmd.ProcessState.ExitCode() != cli.ExitUsage {
			t.Errorf("expected exit code %d, got %d", cli.ExitUsage, cmd.ProcessState.ExitCode())
		}
	})
}

func TestCLIServeAndMCP(t *testing.T) {
	a := createTestApp(t)

	t.Run("serve help", func(t *testing.T) {
		code, out, _ := executeCommand(a, "serve", "--help")
		if code != cli.ExitOK {
			t.Fatalf("expected ExitOK (0), got %d", code)
		}
		if !strings.Contains(out, "Start HTTP server") {
			t.Errorf("expected serve help text, got: %s", out)
		}
	})

	t.Run("mcp ping", func(t *testing.T) {
		var stdin bytes.Buffer
		var stdout bytes.Buffer
		stdin.WriteString(`{"jsonrpc":"2.0","id":1,"method":"ping"}` + "\n")

		code := cli.ExecuteCustom(context.Background(), cli.ExecutionConfig{
			Args: []string{"mcp"},
			In:   &stdin,
			Out:  &stdout,
			App:  a,
		})
		if code != cli.ExitOK {
			t.Fatalf("expected ExitOK (0), got %d", code)
		}
		if !strings.Contains(stdout.String(), `"id":1`) {
			t.Errorf("expected mcp ping response, got: %s", stdout.String())
		}
	})

	t.Run("exec command with fakecli", func(t *testing.T) {
		var stdout bytes.Buffer
		code := cli.ExecuteCustom(context.Background(), cli.ExecutionConfig{
			Args: []string{"exec", "hello world"},
			Out:  &stdout,
			App:  a,
		})
		if code != cli.ExitOK {
			t.Fatalf("expected ExitOK (0), got %d, out=%s", code, stdout.String())
		}
	})

	t.Run("exec command with unknown adapter returns usage error", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := cli.ExecuteCustom(context.Background(), cli.ExecutionConfig{
			Args: []string{"exec", "--adapter", "definitely-not-an-agent", "hello world"},
			Out:  &stdout,
			Err:  &stderr,
			App:  a,
		})
		if code != cli.ExitUsage {
			t.Fatalf("expected ExitUsage (3), got %d, out=%s, err=%s", code, stdout.String(), stderr.String())
		}
	})
}

func BenchmarkVersionColdStart(b *testing.B) {
	start := time.Now()
	var stdout bytes.Buffer
	code := cli.ExecuteCustom(context.Background(), cli.ExecutionConfig{
		Args: []string{"version"},
		Out:  &stdout,
	})
	elapsed := time.Since(start)
	if code != cli.ExitOK {
		b.Fatalf("expected ExitOK (0), got %d", code)
	}
	if elapsed > 10*time.Millisecond {
		b.Logf("warning: cold start took %v (> 10ms)", elapsed)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stdout.Reset()
		_ = cli.ExecuteCustom(context.Background(), cli.ExecutionConfig{
			Args: []string{"version"},
			Out:  &stdout,
		})
	}
}
