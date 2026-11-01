package app_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	"github.com/soundcheck/soundcheck/internal/adapters"
	"github.com/soundcheck/soundcheck/internal/app"
	"github.com/soundcheck/soundcheck/internal/core/domain"
	edgeExec "github.com/soundcheck/soundcheck/internal/edge/exec"
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
		tmpDir, err := os.MkdirTemp("", "fakecli-app-*")
		if err != nil {
			fakecliErr = err
			return
		}
		bin := filepath.Join(tmpDir, "fakecli")
		if runtime.GOOS == "windows" {
			bin += ".exe"
		}
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
				ID:       id,
				Name:     orig.Name,
				Vendor:   orig.Vendor,
				Binary:   bin,
				Category: orig.Category,
				Models:   orig.Models,
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

func TestAppDoctor(t *testing.T) {
	bin := getFakeCLI(t)
	defs := testDefinitions(bin)
	runner := edgeExec.NewPipeRunner()
	cache := store.NewTTLCache(nil)

	application, err := app.New(
		app.WithRunner(runner),
		app.WithCache(cache),
		app.WithDefinitions(defs),
	)
	if err != nil {
		t.Fatalf("app.New failed: %v", err)
	}

	report, err := application.Doctor(context.Background(), app.WithDoctorDeep(true))
	if err != nil {
		t.Fatalf("Doctor failed: %v", err)
	}

	if len(report.Adapters) != 6 {
		t.Errorf("expected 6 adapters, got %d", len(report.Adapters))
	}
	if len(report.Resolutions) == 0 {
		t.Errorf("expected non-empty resolutions")
	}
	if report.Summary.TotalAgents != 6 {
		t.Errorf("expected 6 total agents in summary, got %d", report.Summary.TotalAgents)
	}
	if report.Summary.FunctionalAgents != 6 {
		t.Errorf("expected 6 functional agents, got %d", report.Summary.FunctionalAgents)
	}
}

func TestAppProbeOne(t *testing.T) {
	bin := getFakeCLI(t)
	defs := testDefinitions(bin)
	runner := edgeExec.NewPipeRunner()

	application, err := app.New(
		app.WithRunner(runner),
		app.WithDefinitions(defs),
	)
	if err != nil {
		t.Fatalf("app.New failed: %v", err)
	}

	ad, err := application.ProbeOne(context.Background(), "claude")
	if err != nil {
		t.Fatalf("ProbeOne failed: %v", err)
	}

	if ad.ID != "claude" {
		t.Errorf("expected claude, got %s", ad.ID)
	}
	if !ad.Installed {
		t.Errorf("expected installed true, got false")
	}
	for _, tier := range []domain.Tier{domain.TierL0, domain.TierL1, domain.TierL2, domain.TierL3} {
		if ad.Probes[tier].Status != "passed" {
			t.Errorf("expected %s passed, got %s", tier, ad.Probes[tier].Status)
		}
	}
}

func TestAppProbeOneUnknown(t *testing.T) {
	application, err := app.New()
	if err != nil {
		t.Fatalf("app.New failed: %v", err)
	}

	_, err = application.ProbeOne(context.Background(), "unknown-adapter")
	if err == nil {
		t.Errorf("expected error for unknown adapter, got nil")
	}
}

func TestAppResolve(t *testing.T) {
	bin := getFakeCLI(t)
	defs := testDefinitions(bin)

	application, err := app.New(app.WithDefinitions(defs))
	if err != nil {
		t.Fatalf("app.New failed: %v", err)
	}

	resolutions, err := application.Resolve(context.Background(), "")
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if primary, ok := resolutions[domain.RolePrimary]; !ok || primary.SelectedAgentID == "" {
		t.Errorf("expected primary role resolution, got %+v", primary)
	}
}

func TestAppManifest(t *testing.T) {
	bin := getFakeCLI(t)
	defs := testDefinitions(bin)

	application, err := app.New(app.WithDefinitions(defs))
	if err != nil {
		t.Fatalf("app.New failed: %v", err)
	}

	m, err := application.Manifest(context.Background())
	if err != nil {
		t.Fatalf("Manifest failed: %v", err)
	}

	if m.SchemaVersion != "2026.1" {
		t.Errorf("expected schemaVersion 2026.1, got %s", m.SchemaVersion)
	}
	if len(m.Agents) != 6 {
		t.Errorf("expected 6 agents in manifest, got %d", len(m.Agents))
	}
	if len(m.ResolvedRoles) == 0 {
		t.Errorf("expected resolved roles in manifest")
	}
}
