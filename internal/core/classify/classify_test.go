package classify_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/soundcheck/soundcheck/internal/core/classify"
	"github.com/soundcheck/soundcheck/internal/core/domain"
)

type goldenFixture struct {
	ID                string                `json:"id"`
	AgentID           string                `json:"agentId"`
	Name              string                `json:"name"`
	DocIssue          string                `json:"docIssue"`
	Category          string                `json:"category"`
	Description       string                `json:"description"`
	SimulatedExitCode int                   `json:"simulatedExitCode"`
	SimulatedStdout   string                `json:"simulatedStdout"`
	SimulatedStderr   string                `json:"simulatedStderr"`
	NaiveOutcome      string                `json:"naiveOutcome"`
	SoundcheckOutcome string                `json:"soundcheckOutcome"`
	ExpectedClass     domain.ErrorClass     `json:"expectedClass"`
	Explanation       string                `json:"explanation"`
	Classification    domain.Classification `json:"classification"`
}

func findRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd failed: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not find go.mod from %s", dir)
		}
		dir = parent
	}
}

func TestClassifyGoldenFixturesParity(t *testing.T) {
	root := findRepoRoot(t)
	goldenPath := filepath.Join(root, "testdata", "golden", "classify.json")

	data, err := os.ReadFile(filepath.Clean(goldenPath)) // #nosec G304
	if err != nil {
		t.Fatalf("failed reading golden fixtures at %s: %v", goldenPath, err)
	}

	var fixtures []goldenFixture
	if err := json.Unmarshal(data, &fixtures); err != nil {
		t.Fatalf("failed unmarshaling golden fixtures: %v", err)
	}

	if len(fixtures) == 0 {
		t.Fatalf("expected non-empty golden fixtures")
	}

	quirks := classify.DefaultQuirks()

	for _, f := range fixtures {
		f := f
		t.Run(f.ID, func(t *testing.T) {
			raw := domain.RawOutput{
				AgentID:  f.AgentID,
				ExitCode: f.SimulatedExitCode,
				Stdout:   f.SimulatedStdout,
				Stderr:   f.SimulatedStderr,
			}

			got := classify.Classify(raw, quirks)

			if got.ErrorClass != f.ExpectedClass {
				t.Errorf("ErrorClass = %q, want expectedClass %q", got.ErrorClass, f.ExpectedClass)
			}
			if got.ErrorClass != f.Classification.ErrorClass {
				t.Errorf("ErrorClass = %q, want classification.errorClass %q", got.ErrorClass, f.Classification.ErrorClass)
			}
			if got.Reason != f.Classification.Reason {
				t.Errorf("Reason mismatch:\n got:  %q\n want: %q", got.Reason, f.Classification.Reason)
			}
			if got.CaughtLie != f.Classification.CaughtLie {
				t.Errorf("CaughtLie = %v, want %v", got.CaughtLie, f.Classification.CaughtLie)
			}
		})
	}
}

func TestClassifyQuirkRewrites(t *testing.T) {
	quirks := classify.DefaultQuirks()

	t.Run("claude-79500-rewrites-to-429", func(t *testing.T) {
		raw := domain.RawOutput{
			AgentID:  "claude",
			ExitCode: 0,
			Stdout:   `{"type":"result","subtype":"success","is_error":true,"error":{"type":"rate_limit_error","message":"quota exceeded"}}`,
		}
		got := classify.Classify(raw, quirks)
		if got.Rewritten != 429 {
			t.Errorf("expected Rewritten=429, got %d", got.Rewritten)
		}
		if got.BackoffMin != 30 {
			t.Errorf("expected BackoffMin=30, got %d", got.BackoffMin)
		}
		if !got.CaughtLie {
			t.Errorf("expected CaughtLie=true")
		}
	})

	t.Run("codex-doctor-rewrites-to-401", func(t *testing.T) {
		raw := domain.RawOutput{
			AgentID:  "codex",
			ExitCode: 0,
			Stdout:   `{"version":"0.9.4","doctor_status":"FAILED"}`,
		}
		got := classify.Classify(raw, quirks)
		if got.Rewritten != 401 {
			t.Errorf("expected Rewritten=401, got %d", got.Rewritten)
		}
		if got.ErrorClass != domain.ErrorClassUnauthenticated {
			t.Errorf("expected unauthenticated, got %s", got.ErrorClass)
		}
	})

	t.Run("gemini-exit41-rewrites-to-429-with-cooldown", func(t *testing.T) {
		raw := domain.RawOutput{
			AgentID:  "gemini",
			ExitCode: 41,
			Stderr:   "FatalAuthenticationError: exit 41",
		}
		got := classify.Classify(raw, quirks)
		if got.Rewritten != 429 {
			t.Errorf("expected Rewritten=429, got %d", got.Rewritten)
		}
		if got.BackoffMin != 15 {
			t.Errorf("expected BackoffMin=15, got %d", got.BackoffMin)
		}
	})
}

func TestDefaultQuirksCount(t *testing.T) {
	quirks := classify.DefaultQuirks()
	if len(quirks) != 10 {
		t.Fatalf("expected 10 quirks in catalog, got %d", len(quirks))
	}
}

func TestGenericFallbacks(t *testing.T) {
	tests := []struct {
		name      string
		raw       domain.RawOutput
		wantClass domain.ErrorClass
	}{
		{
			name:      "generic success",
			raw:       domain.RawOutput{ExitCode: 0},
			wantClass: domain.ErrorClassOK,
		},
		{
			name:      "generic unauthorized",
			raw:       domain.RawOutput{ExitCode: 401},
			wantClass: domain.ErrorClassUnauthenticated,
		},
		{
			name:      "generic rate limited",
			raw:       domain.RawOutput{ExitCode: 1, Stderr: "HTTP 429 Too Many Requests"},
			wantClass: domain.ErrorClassRateLimited,
		},
		{
			name:      "generic corrupted",
			raw:       domain.RawOutput{ExitCode: 137},
			wantClass: domain.ErrorClassCorruptedState,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classify.Classify(tt.raw, nil)
			if got.ErrorClass != tt.wantClass {
				t.Errorf("got %s, want %s", got.ErrorClass, tt.wantClass)
			}
		})
	}
}

func TestAgentFallbacks(t *testing.T) {
	tests := []struct {
		name      string
		raw       domain.RawOutput
		wantClass domain.ErrorClass
	}{
		{
			name:      "codex doctor failed in checks array",
			raw:       domain.RawOutput{AgentID: "codex", ExitCode: 0, Stdout: `{"checks":[{"name":"authentication","status":"FAILED"}]}`},
			wantClass: domain.ErrorClassUnauthenticated,
		},
		{
			name:      "codex clean exit 0",
			raw:       domain.RawOutput{AgentID: "codex", ExitCode: 0, Stdout: `{"doctor_status":"OK"}`},
			wantClass: domain.ErrorClassOK,
		},
		{
			name:      "codex error fallback exit 1",
			raw:       domain.RawOutput{AgentID: "codex", ExitCode: 1, Stderr: "connection reset"},
			wantClass: domain.ErrorClassUnauthenticated,
		},
		{
			name:      "agy clean exit 0",
			raw:       domain.RawOutput{AgentID: "agy", ExitCode: 0, Stdout: `{"status":"SUCCESS"}`},
			wantClass: domain.ErrorClassOK,
		},
		{
			name:      "agy fallback exit 1",
			raw:       domain.RawOutput{AgentID: "agy", ExitCode: 1, Stderr: "unknown failure"},
			wantClass: domain.ErrorClassUnauthenticated,
		},
		{
			name:      "gemini clean exit 0",
			raw:       domain.RawOutput{AgentID: "gemini", ExitCode: 0, Stdout: "ready"},
			wantClass: domain.ErrorClassOK,
		},
		{
			name:      "gemini generic exit 1",
			raw:       domain.RawOutput{AgentID: "gemini", ExitCode: 1, Stderr: "unauthorized"},
			wantClass: domain.ErrorClassUnauthenticated,
		},
		{
			name:      "claude exit 0 with non-json",
			raw:       domain.RawOutput{AgentID: "claude", ExitCode: 0, Stdout: "Task complete"},
			wantClass: domain.ErrorClassOK,
		},
		{
			name:      "claude exit 1 generic fallback",
			raw:       domain.RawOutput{AgentID: "claude", ExitCode: 1, Stderr: "fatal error"},
			wantClass: domain.ErrorClassUnauthenticated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classify.Classify(tt.raw, nil)
			if got.ErrorClass != tt.wantClass {
				t.Errorf("got %s, want %s", got.ErrorClass, tt.wantClass)
			}
		})
	}
}
