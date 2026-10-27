package probe_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/soundcheck/soundcheck/internal/adapters"
	"github.com/soundcheck/soundcheck/internal/core/domain"
	edgeExec "github.com/soundcheck/soundcheck/internal/edge/exec"
	"github.com/soundcheck/soundcheck/internal/edge/probe"
	"github.com/soundcheck/soundcheck/internal/edge/store"
)

var (
	fakecliOnce sync.Once
	fakecliPath string
	fakecliErr  error
)

func getFakeCLI(t testing.TB) string {
	t.Helper()
	fakecliOnce.Do(func() {
		tmpDir, err := os.MkdirTemp("", "fakecli-probe-*")
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

func TestProbeL0L3Success(t *testing.T) {
	bin := getFakeCLI(t)
	runner := edgeExec.NewPipeRunner()
	prober := probe.New(probe.WithRunner(runner))

	def := adapters.Definition{
		Adapter: domain.Adapter{
			ID:     "test-agent",
			Name:   "Test Agent",
			Binary: bin,
		},
		ProbeSpecs: map[domain.Tier]domain.Probe{
			domain.TierL0: {Level: domain.TierL0, Command: bin + " --version"},
			domain.TierL1: {Level: domain.TierL1, Command: bin},
			domain.TierL2: {Level: domain.TierL2, Command: bin},
			domain.TierL3: {Level: domain.TierL3, Command: bin},
		},
	}

	ad := prober.ProbeAdapter(context.Background(), def)

	if !ad.Installed {
		t.Errorf("expected installed true, got false")
	}
	for _, tier := range []domain.Tier{domain.TierL0, domain.TierL1, domain.TierL2, domain.TierL3} {
		res, ok := ad.Probes[tier]
		if !ok {
			t.Fatalf("missing probe for %s", tier)
		}
		if res.Status != "passed" {
			t.Errorf("expected %s passed, got %s (msg: %s)", tier, res.Status, res.Message)
		}
		if res.ErrorClass != domain.ErrorClassOK {
			t.Errorf("expected %s ok, got %s", tier, res.ErrorClass)
		}
	}
}

func TestProbeStopOnFailL0(t *testing.T) {
	runner := edgeExec.NewPipeRunner()
	prober := probe.New(probe.WithRunner(runner))

	def := adapters.Definition{
		Adapter: domain.Adapter{
			ID:     "missing-agent",
			Name:   "Missing Agent",
			Binary: "nonexistent-binary-12345",
		},
	}

	ad := prober.ProbeAdapter(context.Background(), def)

	if ad.Installed {
		t.Errorf("expected installed false, got true")
	}
	if ad.Probes[domain.TierL0].Status != "warning" {
		t.Errorf("expected L0 warning, got %s", ad.Probes[domain.TierL0].Status)
	}
	if ad.Probes[domain.TierL0].ErrorClass != domain.ErrorClassNotInstalled {
		t.Errorf("expected not_installed, got %s", ad.Probes[domain.TierL0].ErrorClass)
	}

	for _, tier := range []domain.Tier{domain.TierL1, domain.TierL2, domain.TierL3} {
		res := ad.Probes[tier]
		if res.Status != "pending" {
			t.Errorf("expected %s pending, got %s", tier, res.Status)
		}
		if res.Message != "Skipped" {
			t.Errorf("expected %s skipped message, got %s", tier, res.Message)
		}
	}
}

func TestProbeStopOnFailL1QuirkUnauth(t *testing.T) {
	bin := getFakeCLI(t)
	runner := edgeExec.NewPipeRunner()
	prober := probe.New(probe.WithRunner(runner))

	def := adapters.Definition{
		Adapter: domain.Adapter{
			ID:     "codex",
			Name:   "OpenAI Codex",
			Binary: bin,
			Quirks: []domain.Quirk{
				{
					ID:              "codex-doctor-exit0",
					AgentID:         "codex",
					TriggerExitCode: intPtr(0),
					TriggerRegex:    "doctor_status.*FAILED",
					ErrorClass:      domain.ErrorClassUnauthenticated,
					CaughtLie:       true,
				},
			},
		},
		ProbeSpecs: map[domain.Tier]domain.Probe{
			domain.TierL0: {Level: domain.TierL0, Command: bin + " --version"},
			domain.TierL1: {Level: domain.TierL1, Command: bin + " -quirk unauth"},
			domain.TierL2: {Level: domain.TierL2, Command: bin},
			domain.TierL3: {Level: domain.TierL3, Command: bin},
		},
	}

	ad := prober.ProbeAdapter(context.Background(), def)

	if !ad.Installed {
		t.Errorf("expected installed true, got false")
	}
	if ad.Probes[domain.TierL0].Status != "passed" {
		t.Errorf("expected L0 passed, got %s", ad.Probes[domain.TierL0].Status)
	}
	if ad.Probes[domain.TierL1].Status != "failed" {
		t.Errorf("expected L1 failed, got %s", ad.Probes[domain.TierL1].Status)
	}
	if ad.Probes[domain.TierL1].ErrorClass != domain.ErrorClassUnauthenticated {
		t.Errorf("expected L1 unauthenticated, got %s", ad.Probes[domain.TierL1].ErrorClass)
	}
	for _, tier := range []domain.Tier{domain.TierL2, domain.TierL3} {
		if ad.Probes[tier].Status != "pending" {
			t.Errorf("expected %s pending, got %s", tier, ad.Probes[tier].Status)
		}
	}
}

func TestProbeStopOnFailL2QuirkRateLimit(t *testing.T) {
	bin := getFakeCLI(t)
	runner := edgeExec.NewPipeRunner()
	prober := probe.New(probe.WithRunner(runner))

	def := adapters.Definition{
		Adapter: domain.Adapter{
			ID:     "claude",
			Name:   "Claude Code",
			Binary: bin,
			Quirks: []domain.Quirk{
				{
					ID:              "claude-79500",
					AgentID:         "claude",
					TriggerExitCode: intPtr(0),
					TriggerRegex:    "rate_limit_error",
					ErrorClass:      domain.ErrorClassRateLimited,
					CaughtLie:       true,
				},
			},
		},
		ProbeSpecs: map[domain.Tier]domain.Probe{
			domain.TierL0: {Level: domain.TierL0, Command: bin + " --version"},
			domain.TierL1: {Level: domain.TierL1, Command: bin},
			domain.TierL2: {Level: domain.TierL2, Command: bin + " -quirk rate_limit"},
			domain.TierL3: {Level: domain.TierL3, Command: bin},
		},
	}

	ad := prober.ProbeAdapter(context.Background(), def)

	if ad.Probes[domain.TierL0].Status != "passed" || ad.Probes[domain.TierL1].Status != "passed" {
		t.Errorf("expected L0 and L1 passed")
	}
	if ad.Probes[domain.TierL2].Status != "failed" {
		t.Errorf("expected L2 failed, got %s", ad.Probes[domain.TierL2].Status)
	}
	if ad.Probes[domain.TierL2].ErrorClass != domain.ErrorClassRateLimited {
		t.Errorf("expected L2 rate_limited, got %s", ad.Probes[domain.TierL2].ErrorClass)
	}
	if ad.Probes[domain.TierL3].Status != "pending" {
		t.Errorf("expected L3 pending, got %s", ad.Probes[domain.TierL3].Status)
	}
}

func TestProbeTTLCacheHonored(t *testing.T) {
	bin := getFakeCLI(t)
	runner := edgeExec.NewPipeRunner()
	cache := store.NewTTLCache(nil)
	prober := probe.New(probe.WithRunner(runner), probe.WithCache(cache), probe.WithTTL(10*time.Minute))

	def := adapters.Definition{
		Adapter: domain.Adapter{
			ID:     "cached-agent",
			Name:   "Cached Agent",
			Binary: bin,
		},
		ProbeSpecs: map[domain.Tier]domain.Probe{
			domain.TierL0: {Level: domain.TierL0, Command: bin + " --version"},
		},
	}

	ad1 := prober.ProbeAdapter(context.Background(), def)
	if ad1.Probes[domain.TierL0].Status != "passed" {
		t.Fatalf("first probe run failed")
	}

	failingProber := probe.New(
		probe.WithRunner(&failRunner{}),
		probe.WithCache(cache),
		probe.WithTTL(10*time.Minute),
	)

	ad2 := failingProber.ProbeAdapter(context.Background(), def)
	if ad2.Probes[domain.TierL0].Status != "passed" {
		t.Errorf("expected cached probe to return passed even if runner fails")
	}
}

type failRunner struct{}

func (f *failRunner) Run(_ context.Context, _ edgeExec.Cmd) (edgeExec.Result, error) {
	return edgeExec.Result{ExitCode: 1, Stderr: []byte("runner failure")}, nil
}

func (f *failRunner) Stream(_ context.Context, _ edgeExec.Cmd) (<-chan edgeExec.Event, error) {
	ch := make(chan edgeExec.Event)
	close(ch)
	return ch, nil
}

func intPtr(i int) *int {
	return &i
}

func BenchmarkL0Sweep(b *testing.B) {
	bin := getFakeCLI(b)
	runner := edgeExec.NewPipeRunner()
	prober := probe.New(probe.WithRunner(runner), probe.WithMaxTier(domain.TierL0))

	defs := make([]adapters.Definition, 6)
	for i := range 6 {
		defs[i] = adapters.Definition{
			Adapter: domain.Adapter{
				ID:     fmt.Sprintf("bench-agent-%d", i),
				Binary: bin,
			},
			ProbeSpecs: map[domain.Tier]domain.Probe{
				domain.TierL0: {Level: domain.TierL0, Command: bin + " --version"},
			},
		}
	}

	ctx := context.Background()
	b.ResetTimer()

	start := time.Now()
	for i := 0; i < b.N; i++ {
		res := prober.ProbeAll(ctx, defs)
		if len(res) != 6 {
			b.Fatalf("expected 6 results, got %d", len(res))
		}
	}
	elapsed := time.Since(start)

	avgDuration := elapsed / time.Duration(b.N)
	if avgDuration > 150*time.Millisecond {
		b.Errorf("expected L0 sweep <= 150ms, got %v", avgDuration)
	}
}
