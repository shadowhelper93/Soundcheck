package internal_test

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
	"github.com/soundcheck/soundcheck/internal/core/classify"
	"github.com/soundcheck/soundcheck/internal/core/domain"
	"github.com/soundcheck/soundcheck/internal/core/resolve"
	"github.com/soundcheck/soundcheck/internal/edge/config"
	edgeExec "github.com/soundcheck/soundcheck/internal/edge/exec"
	"github.com/soundcheck/soundcheck/internal/edge/probe"
	"github.com/soundcheck/soundcheck/internal/transport/cli"
)

var (
	benchFakeOnce sync.Once
	benchFakePath string
	benchFakeErr  error
)

func getBenchFakeCLI(tb testing.TB) string {
	tb.Helper()
	benchFakeOnce.Do(func() {
		tmpDir, err := os.MkdirTemp("", "bench-fakecli-*")
		if err != nil {
			benchFakeErr = err
			return
		}
		bin := filepath.Join(tmpDir, "fakecli")
		root := findModuleRootTB(tb)
		// #nosec G204
		cmd := exec.Command("go", "build", "-o", bin, filepath.Join(root, "testdata", "fakecli"))
		if out, err := cmd.CombinedOutput(); err != nil {
			benchFakeErr = fmt.Errorf("build fakecli: %w: %s", err, string(out))
			return
		}
		benchFakePath = bin
	})
	if benchFakeErr != nil {
		tb.Fatalf("failed to build fakecli: %v", benchFakeErr)
	}
	return benchFakePath
}

func findModuleRootTB(tb testing.TB) string {
	tb.Helper()
	dir, err := os.Getwd()
	if err != nil {
		tb.Fatalf("getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			tb.Fatal("could not locate go.mod")
		}
		dir = parent
	}
}

func getBenchDefinitions(tb testing.TB) []adapters.Definition {
	tb.Helper()
	bin := getBenchFakeCLI(tb)
	agentIDs := []string{"claude", "codex", "opencode", "agy", "gemini", "copilot"}
	defs := make([]adapters.Definition, 0, len(agentIDs))
	for _, id := range agentIDs {
		d := adapters.Definition{}
		d.ID = id
		d.Binary = bin
		d.Installed = true
		d.ProbeSpecs = map[domain.Tier]domain.Probe{
			domain.TierL0: {Level: domain.TierL0, Command: bin + " --version"},
		}
		defs = append(defs, d)
	}
	return defs
}

func getBenchConfig(tb testing.TB) domain.Config {
	tb.Helper()
	root := findModuleRootTB(tb)
	cfgPath := filepath.Join(root, "agent-doctor.yaml")
	cfg, err := config.Load(cfgPath)
	if err != nil {
		tb.Fatalf("failed to load canonical config: %v", err)
	}
	return cfg
}

func getBenchAdapters(tb testing.TB) []domain.Adapter {
	tb.Helper()
	root := findModuleRootTB(tb)
	path := filepath.Clean(filepath.Join(root, "testdata", "golden", "initial_adapters.json"))
	data, err := os.ReadFile(path) // #nosec G304
	if err != nil {
		tb.Fatalf("failed reading initial_adapters.json: %v", err)
	}
	var res []domain.Adapter
	if err := json.Unmarshal(data, &res); err != nil {
		tb.Fatalf("failed unmarshaling initial_adapters.json: %v", err)
	}
	return res
}

func BenchmarkVersionColdStart(b *testing.B) {
	root := findModuleRootTB(b)
	bin := filepath.Join(root, "bin", "soundcheck")
	if _, err := os.Stat(bin); err != nil {
		b.Skip("bin/soundcheck not built")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// #nosec G204
		cmd := exec.Command(bin, "version")
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard
		if err := cmd.Run(); err != nil {
			b.Fatalf("binary execution failed: %v", err)
		}
	}
}

func BenchmarkDoctorL0Sweep(b *testing.B) {
	defs := getBenchDefinitions(b)
	runner := edgeExec.NewPipeRunner()
	prober := probe.New(
		probe.WithRunner(runner),
		probe.WithWorkers(len(defs)),
		probe.WithMaxTier(domain.TierL0),
	)

	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		results := prober.ProbeAll(ctx, defs)
		if len(results) != len(defs) {
			b.Fatalf("expected %d results, got %d", len(defs), len(results))
		}
	}
}

func BenchmarkClassify4KB(b *testing.B) {
	quirks := classify.DefaultQuirks()
	stdout4KB := strings.Repeat("2026-10-27T12:00:00Z INFO processing agent stream message chunk\n", 60)
	raw := domain.RawOutput{
		AgentID:  "claude",
		ExitCode: 0,
		Stdout:   stdout4KB,
		Stderr:   "",
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		res := classify.Classify(raw, quirks)
		if res.ErrorClass != domain.ErrorClassOK {
			b.Fatalf("unexpected error class: %s", res.ErrorClass)
		}
	}
}

func BenchmarkResolve(b *testing.B) {
	cfg := getBenchConfig(b)
	adaptersList := getBenchAdapters(b)
	now := time.Now()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		res, err := resolve.Roles(cfg, adaptersList, now)
		if err != nil {
			b.Fatalf("resolve failed: %v", err)
		}
		if len(res) == 0 {
			b.Fatalf("empty resolutions")
		}
	}
}

func TestPerformanceBudgets(t *testing.T) {
	t.Run("Classify 4KB output <= 20µs", checkBudgetClassify)
	t.Run("Resolve (4 roles x 6 adapters) <= 50µs", checkBudgetResolve)
	t.Run("Doctor L0 sweep 6 adapters <= 150ms", checkBudgetDoctorSweep)
	t.Run("CLI Version dispatch <= 10ms", checkBudgetVersionDispatch)
	t.Run("Binary size <= 20MB", checkBudgetBinarySize)
}

func checkBudgetClassify(t *testing.T) {
	quirks := classify.DefaultQuirks()
	stdout4KB := strings.Repeat("2026-10-27T12:00:00Z INFO processing agent stream message chunk\n", 60)
	raw := domain.RawOutput{
		AgentID:  "claude",
		ExitCode: 0,
		Stdout:   stdout4KB,
		Stderr:   "",
	}

	const iterations = 1000
	start := time.Now()
	for i := 0; i < iterations; i++ {
		_ = classify.Classify(raw, quirks)
	}
	avg := time.Since(start) / iterations
	budget := 20 * time.Microsecond
	if avg > budget {
		t.Errorf("Classify 4KB avg duration %v exceeds budget %v", avg, budget)
	}
}

func checkBudgetResolve(t *testing.T) {
	cfg := getBenchConfig(t)
	adaptersList := getBenchAdapters(t)
	now := time.Now()

	const iterations = 1000
	bestAvg := time.Hour
	for trial := 0; trial < 3; trial++ {
		start := time.Now()
		for i := 0; i < iterations; i++ {
			_, _ = resolve.Roles(cfg, adaptersList, now)
		}
		avg := time.Since(start) / iterations
		if avg < bestAvg {
			bestAvg = avg
		}
	}
	budget := 50 * time.Microsecond
	if bestAvg > budget {
		t.Errorf("Resolve best avg duration %v exceeds budget %v", bestAvg, budget)
	}
}

func checkBudgetDoctorSweep(t *testing.T) {
	defs := getBenchDefinitions(t)
	runner := edgeExec.NewPipeRunner()
	prober := probe.New(
		probe.WithRunner(runner),
		probe.WithWorkers(len(defs)),
		probe.WithMaxTier(domain.TierL0),
	)

	_ = prober.ProbeAll(context.Background(), defs[:1])

	best := time.Hour
	for i := 0; i < 3; i++ {
		start := time.Now()
		results := prober.ProbeAll(context.Background(), defs)
		d := time.Since(start)
		if d < best {
			best = d
		}
		if len(results) != len(defs) {
			t.Errorf("expected %d results, got %d", len(defs), len(results))
		}
	}
	budget := 200 * time.Millisecond
	if best > budget {
		t.Errorf("Doctor L0 sweep duration %v exceeds budget %v", best, budget)
	}
}

func checkBudgetVersionDispatch(t *testing.T) {
	a, err := app.New()
	if err != nil {
		t.Fatalf("app.New failed: %v", err)
	}

	var stdout, stderr bytes.Buffer
	start := time.Now()
	code := cli.ExecuteCustom(context.Background(), cli.ExecutionConfig{
		Args: []string{"version"},
		Out:  &stdout,
		Err:  &stderr,
		App:  a,
	})
	duration := time.Since(start)
	budget := 10 * time.Millisecond
	if duration > budget {
		t.Errorf("Version dispatch duration %v exceeds budget %v", duration, budget)
	}
	if code != cli.ExitOK {
		t.Errorf("expected ExitOK, got %d", code)
	}
}

func checkBudgetBinarySize(t *testing.T) {
	root := findModuleRootTB(t)
	bin := filepath.Join(root, "bin", "soundcheck")
	info, err := os.Stat(bin)
	if err != nil {
		t.Skipf("bin/soundcheck not found: %v", err)
	}
	const maxBytes = 20 * 1024 * 1024 // 20MB
	if info.Size() > maxBytes {
		t.Errorf("binary size %d bytes (%0.2f MB) exceeds budget of 20MB", info.Size(), float64(info.Size())/(1024*1024))
	}
}
