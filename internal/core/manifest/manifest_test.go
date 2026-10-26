package manifest_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/soundcheck/soundcheck/internal/core/domain"
	"github.com/soundcheck/soundcheck/internal/core/manifest"
)

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

func loadAdapters(t *testing.T) []domain.Adapter {
	t.Helper()
	root := findRepoRoot(t)
	data, err := os.ReadFile(filepath.Clean(filepath.Join(root, "testdata", "golden", "initial_adapters.json")))
	if err != nil {
		t.Fatalf("failed reading initial_adapters.json: %v", err)
	}
	var adapters []domain.Adapter
	if err := json.Unmarshal(data, &adapters); err != nil {
		t.Fatalf("failed unmarshaling initial_adapters.json: %v", err)
	}
	return adapters
}

func loadGoldenResolutions(t *testing.T) map[domain.Role]domain.RoleResolution {
	t.Helper()
	root := findRepoRoot(t)
	data, err := os.ReadFile(filepath.Clean(filepath.Join(root, "testdata", "golden", "resolve.json")))
	if err != nil {
		t.Fatalf("failed reading resolve.json: %v", err)
	}
	var golden map[domain.Role]domain.RoleResolution
	if err := json.Unmarshal(data, &golden); err != nil {
		t.Fatalf("failed unmarshaling resolve.json: %v", err)
	}
	return golden
}

func loadGoldenManifest(t *testing.T) domain.Manifest {
	t.Helper()
	root := findRepoRoot(t)
	data, err := os.ReadFile(filepath.Clean(filepath.Join(root, "testdata", "golden", "manifest.json")))
	if err != nil {
		t.Fatalf("failed reading manifest.json: %v", err)
	}
	var golden domain.Manifest
	if err := json.Unmarshal(data, &golden); err != nil {
		t.Fatalf("failed unmarshaling manifest.json: %v", err)
	}
	return golden
}

func TestManifestGoldenParity(t *testing.T) {
	adapters := loadAdapters(t)
	resolutions := loadGoldenResolutions(t)
	golden := loadGoldenManifest(t)

	now, err := time.Parse("2006-01-02T15:04:05.000Z", "2026-10-26T00:00:00.000Z")
	if err != nil {
		t.Fatalf("failed parsing time: %v", err)
	}

	host := domain.HostInfo{
		Platform:    "darwin",
		NodeVersion: "v22.0.0",
		GitWorktree: true,
	}

	actual := manifest.BuildManifest(adapters, resolutions, host, now)

	if !reflect.DeepEqual(actual, golden) {
		actJSON, _ := json.MarshalIndent(actual, "", "  ")
		goldJSON, _ := json.MarshalIndent(golden, "", "  ")
		t.Fatalf("Manifest mismatch.\nGOT:\n%s\nWANT:\n%s", string(actJSON), string(goldJSON))
	}
}

func TestManifestUnconfiguredAndCooldown(t *testing.T) {
	now := time.UnixMilli(1730000000000)
	futureCooldown := now.UnixMilli() + 60000

	adapters := []domain.Adapter{
		{
			ID:        "cooldown-agent",
			Installed: true,
			Probes: map[domain.Tier]domain.ProbeResult{
				domain.TierL1: {Status: "passed"},
				domain.TierL2: {Status: "failed", ErrorClass: domain.ErrorClassRateLimited},
			},
			CooldownUntil:   &futureCooldown,
			PermissionModes: nil,
			Models:          nil,
		},
		{
			ID:        "unauth-agent",
			Installed: true,
			Probes: map[domain.Tier]domain.ProbeResult{
				domain.TierL1: {Status: "failed"},
			},
		},
	}

	// Empty role resolutions
	resolutions := make(map[domain.Role]domain.RoleResolution)
	host := domain.HostInfo{Platform: "linux", GitWorktree: false}

	m := manifest.GenerateCapabilityManifest(adapters, resolutions, host, now)

	if m.Summary.TotalAgents != 2 {
		t.Errorf("total agents = %d, want 2", m.Summary.TotalAgents)
	}
	if m.Summary.FunctionalAgents != 0 {
		t.Errorf("functional agents = %d, want 0", m.Summary.FunctionalAgents)
	}
	if m.Summary.CooldownsActive != 1 {
		t.Errorf("cooldowns active = %d, want 1", m.Summary.CooldownsActive)
	}

	agentCool := m.Agents["cooldown-agent"]
	if !agentCool.CooldownActive || agentCool.Functional {
		t.Errorf("cooldown agent expected cooldownActive=true, functional=false; got %+v", agentCool)
	}
	if agentCool.ErrorClass != domain.ErrorClassRateLimited {
		t.Errorf("errorClass = %q, want %q", agentCool.ErrorClass, domain.ErrorClassRateLimited)
	}

	// Standard roles should fall back to defaults
	primary := m.ResolvedRoles[domain.RolePrimary]
	if primary.Agent != "none" || primary.Model != "none" || primary.Vendor != "none" || primary.Reason != "No role assignment configured" {
		t.Errorf("unconfigured primary role = %+v", primary)
	}
}

func TestManifestCustomRoleAndPartialResolution(t *testing.T) {
	now := time.UnixMilli(1730000000000)
	adapters := []domain.Adapter{
		{
			ID:        "agent-1",
			Installed: true,
			Probes: map[domain.Tier]domain.ProbeResult{
				domain.TierL1: {Status: "passed"},
			},
			Models: []domain.ModelInfo{
				{ID: "model-1"},
			},
			PermissionModes: []string{"read-only"},
		},
	}

	customRole := domain.Role("custom-auditor")
	emptyAgentRole := domain.Role("empty-agent")
	resolutions := map[domain.Role]domain.RoleResolution{
		domain.RolePrimary: {
			Role:            domain.RolePrimary,
			SelectedAgentID: "agent-1",
		},
		emptyAgentRole: {
			Role:          emptyAgentRole,
			SelectedModel: "model-x",
		},
		customRole: {
			Role:            customRole,
			SelectedAgentID: "agent-1",
			SelectedModel:   "model-1",
			Vendor:          "VendorX",
			Reasoning:       "Custom chosen",
		},
	}

	m := manifest.BuildManifest(adapters, resolutions, domain.HostInfo{Platform: "darwin"}, now)

	// Primary should have partial fallback
	prim := m.ResolvedRoles[domain.RolePrimary]
	if prim.Agent != "agent-1" || prim.Model != "none" || prim.Vendor != "none" || prim.Reason != "No role assignment configured" {
		t.Errorf("partial primary = %+v", prim)
	}

	// Custom role should be included
	cust, ok := m.ResolvedRoles[customRole]
	if !ok || cust.Agent != "agent-1" || cust.Model != "model-1" || cust.Vendor != "VendorX" || cust.Reason != "Custom chosen" {
		t.Errorf("custom role = %+v, ok=%v", cust, ok)
	}
}
