package classify_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/soundcheck/soundcheck/internal/core/classify"
	"github.com/soundcheck/soundcheck/internal/core/domain"
)

func findRootForFuzz(tb testing.TB) string {
	tb.Helper()
	dir, err := os.Getwd()
	if err != nil {
		tb.Fatalf("getwd failed: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			tb.Fatalf("could not find go.mod from %s", dir)
		}
		dir = parent
	}
}

func addGoldenSeeds(f *testing.F, root string) {
	goldenPath := filepath.Join(root, "testdata", "golden", "classify.json")
	data, err := os.ReadFile(filepath.Clean(goldenPath))
	if err != nil {
		return
	}
	var fixtures []goldenFixture
	if err := json.Unmarshal(data, &fixtures); err != nil {
		return
	}
	for _, fix := range fixtures {
		f.Add(fix.AgentID, fix.SimulatedExitCode, fix.SimulatedStdout, fix.SimulatedStderr)
	}
}

func addManualSeeds(f *testing.F) {
	f.Add("claude", 0, `{"is_error": true, "error": {"type": "rate_limit_error"}}`, "")
	f.Add("codex", 0, `{"doctor_status": "FAILED", "checks": []}`, "")
	f.Add("gemini", 41, "", "FatalAuthenticationError: invalid credentials")
	f.Add("agy", 0, "", "Model not found")
	f.Add("unknown", 127, "", "command not found")
	f.Add("", 0, "", "")
	f.Add("claude", 1, "{}", "")
	f.Add("codex", 255, string(make([]byte, 4096)), "")
}

func FuzzClassify(f *testing.F) {
	root := findRootForFuzz(f)
	addGoldenSeeds(f, root)
	addManualSeeds(f)

	quirks := classify.DefaultQuirks()

	f.Fuzz(func(t *testing.T, agentID string, exitCode int, stdout string, stderr string) {
		raw := domain.RawOutput{
			AgentID:  agentID,
			ExitCode: exitCode,
			Stdout:   stdout,
			Stderr:   stderr,
		}

		res := classify.Classify(raw, quirks)
		if res.ErrorClass == "" {
			t.Errorf("expected non-empty ErrorClass for agent=%q exitCode=%d", agentID, exitCode)
		}
		if res.Reason == "" {
			t.Errorf("expected non-empty Reason for agent=%q exitCode=%d", agentID, exitCode)
		}
	})
}
