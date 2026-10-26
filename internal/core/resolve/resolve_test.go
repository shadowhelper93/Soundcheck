package resolve_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/soundcheck/soundcheck/internal/core/domain"
	"github.com/soundcheck/soundcheck/internal/core/resolve"
)

const fixedTimestampMs int64 = 1730000000000

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

func loadGoldenResolutions(t *testing.T) map[domain.Role]domain.DetailedResolution {
	t.Helper()
	root := findRepoRoot(t)
	data, err := os.ReadFile(filepath.Clean(filepath.Join(root, "testdata", "golden", "resolve.json")))
	if err != nil {
		t.Fatalf("failed reading resolve.json: %v", err)
	}
	var golden map[domain.Role]domain.DetailedResolution
	if err := json.Unmarshal(data, &golden); err != nil {
		t.Fatalf("failed unmarshaling resolve.json: %v", err)
	}
	return golden
}

func TestResolveGoldenParity(t *testing.T) {
	adapters := loadAdapters(t)
	cfg := resolve.DefaultConfig()
	fixedNow := time.UnixMilli(fixedTimestampMs)

	actual, err := resolve.ResolveRoles(cfg, adapters, fixedNow)
	if err != nil {
		t.Fatalf("unexpected ResolveRoles error: %v", err)
	}

	golden := loadGoldenResolutions(t)
	for _, role := range []domain.Role{domain.RolePrimary, domain.RoleSubagent, domain.RoleAuxiliary, domain.RoleReviewer} {
		expectedRes, ok := golden[role]
		if !ok {
			t.Fatalf("missing expected role in golden: %s", role)
		}
		actualRes, ok := actual[role]
		if !ok {
			t.Fatalf("missing actual role in resolution: %s", role)
		}

		if actualRes.Role != expectedRes.Role {
			t.Errorf("role mismatch: got %q, want %q", actualRes.Role, expectedRes.Role)
		}
		if actualRes.SelectedAgentID != expectedRes.SelectedAgentID {
			t.Errorf("role %s selectedAgentId mismatch: got %q, want %q", role, actualRes.SelectedAgentID, expectedRes.SelectedAgentID)
		}
		if actualRes.SelectedModel != expectedRes.SelectedModel {
			t.Errorf("role %s selectedModel mismatch: got %q, want %q", role, actualRes.SelectedModel, expectedRes.SelectedModel)
		}
		if actualRes.Vendor != expectedRes.Vendor {
			t.Errorf("role %s vendor mismatch: got %q, want %q", role, actualRes.Vendor, expectedRes.Vendor)
		}
		if actualRes.Reasoning != expectedRes.Reasoning {
			t.Errorf("role %s reasoning mismatch: got %q, want %q", role, actualRes.Reasoning, expectedRes.Reasoning)
		}
		if actualRes.EstimatedCostPerM != expectedRes.EstimatedCostPerM {
			t.Errorf("role %s cost mismatch: got %f, want %f", role, actualRes.EstimatedCostPerM, expectedRes.EstimatedCostPerM)
		}
		if actualRes.Healthy != expectedRes.Healthy {
			t.Errorf("role %s healthy mismatch: got %v, want %v", role, actualRes.Healthy, expectedRes.Healthy)
		}
		if !reflect.DeepEqual(actualRes.FallbackChain, expectedRes.FallbackChain) {
			t.Errorf("role %s fallbackChain mismatch: got %+v, want %+v", role, actualRes.FallbackChain, expectedRes.FallbackChain)
		}
		if !reflect.DeepEqual(actualRes.Steps, expectedRes.Steps) {
			t.Errorf("role %s steps mismatch:\ngot  %+v\nwant %+v", role, actualRes.Steps, expectedRes.Steps)
		}
	}
}

func TestResolveAliasFunction(t *testing.T) {
	adapters := loadAdapters(t)
	cfg := resolve.DefaultConfig()
	fixedNow := time.UnixMilli(fixedTimestampMs)

	res1, err1 := resolve.Roles(cfg, adapters, fixedNow)
	if err1 != nil {
		t.Fatalf("Roles error: %v", err1)
	}
	res2, err2 := resolve.Resolve(cfg, adapters, fixedNow)
	if err2 != nil {
		t.Fatalf("Resolve error: %v", err2)
	}

	if !reflect.DeepEqual(res1, res2) {
		t.Errorf("Roles and Resolve returned differing outputs")
	}
}

func TestResolveAllCandidatesInCooldown(t *testing.T) {
	futureCooldown := fixedTimestampMs + 600000 // 10 minutes from fixedNow
	adapters := []domain.Adapter{
		{
			ID:            "agent-a",
			Installed:     true,
			Vendor:        "VendorA",
			CooldownUntil: &futureCooldown,
		},
		{
			ID:        "agent-b",
			Installed: true,
			Vendor:    "VendorB",
		},
	}

	cfg := domain.Config{
		Version: 1,
		Roles: map[domain.Role][]string{
			domain.RolePrimary: {"agent-a:model-1", "agent-b:model-2"},
		},
	}
	fixedNow := time.UnixMilli(fixedTimestampMs)

	// Agent A is in cooldown via CooldownUntil; force Agent B into cooldown via option.
	res, err := resolve.ResolveRoles(cfg, adapters, fixedNow, resolve.WithForceCooldownAgentID("agent-b"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	primary := res[domain.RolePrimary]
	if primary.Healthy {
		t.Errorf("expected primary to be unhealthy, got healthy")
	}
	if primary.SelectedAgentID != "none" {
		t.Errorf("expected selectedAgentId none, got %q", primary.SelectedAgentID)
	}
	if len(primary.FallbackChain) != 2 {
		t.Fatalf("expected 2 fallbacks, got %d", len(primary.FallbackChain))
	}
	if primary.FallbackChain[0] != "agent-a:model-1 (cooldown)" {
		t.Errorf("fallback 0 mismatch: %q", primary.FallbackChain[0])
	}
	if primary.FallbackChain[1] != "agent-b:model-2 (cooldown)" {
		t.Errorf("fallback 1 mismatch: %q", primary.FallbackChain[1])
	}
	if len(primary.Steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(primary.Steps))
	}
	if primary.Steps[0].Status != domain.StepStatusSkippedCooldown {
		t.Errorf("step 0 status mismatch: %s", primary.Steps[0].Status)
	}
	if primary.Steps[1].Status != domain.StepStatusSkippedCooldown {
		t.Errorf("step 1 status mismatch: %s", primary.Steps[1].Status)
	}
}

func TestResolveUninstalledFallback(t *testing.T) {
	adapters := []domain.Adapter{
		{
			ID:        "installed-broken",
			Installed: true,
			Vendor:    "VendorX",
			Probes: map[domain.Tier]domain.ProbeResult{
				domain.TierL0: {Status: "failed"},
			},
		},
		{
			ID:        "healthy-backup",
			Installed: true,
			Vendor:    "VendorY",
			Models: []domain.ModelInfo{
				{ID: "m1", DisplayName: "Backup Model", InputCostPerM: 1.0, OutputCostPerM: 2.0},
			},
		},
	}

	cfg := domain.Config{
		Version: 1,
		Roles: map[domain.Role][]string{
			domain.RolePrimary: {
				"unknown-agent:m0",
				"installed-broken:m0",
				"healthy-backup:m1",
			},
		},
	}

	res, err := resolve.ResolveRoles(cfg, adapters, time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	primary := res[domain.RolePrimary]
	if !primary.Healthy {
		t.Fatalf("expected healthy resolution for fallback, got unhealthy")
	}
	if primary.SelectedAgentID != "healthy-backup" {
		t.Errorf("selected agent mismatch: got %q, want healthy-backup", primary.SelectedAgentID)
	}
	if primary.SelectedModel != "Backup Model" {
		t.Errorf("selected model mismatch: got %q, want Backup Model", primary.SelectedModel)
	}
	if len(primary.Steps) != 3 {
		t.Fatalf("expected 3 steps, got %d", len(primary.Steps))
	}
	if primary.Steps[0].Status != domain.StepStatusSkippedUninstalled {
		t.Errorf("step 0 status mismatch: %s", primary.Steps[0].Status)
	}
	if primary.Steps[1].Status != domain.StepStatusSkippedUninstalled {
		t.Errorf("step 1 status mismatch: %s", primary.Steps[1].Status)
	}
	if primary.Steps[2].Status != domain.StepStatusSelected {
		t.Errorf("step 2 status mismatch: %s", primary.Steps[2].Status)
	}
}

func TestResolveVendorDiversityConflict(t *testing.T) {
	adapters := []domain.Adapter{
		{
			ID:        "claude-like",
			Vendor:    "Anthropic",
			Installed: true,
			Models: []domain.ModelInfo{
				{ID: "sonnet", DisplayName: "Sonnet", InputCostPerM: 3.0, OutputCostPerM: 15.0},
			},
		},
		{
			ID:        "codex-like",
			Vendor:    "OpenAI",
			Installed: true,
			Models: []domain.ModelInfo{
				{ID: "codex-mod", DisplayName: "Codex", InputCostPerM: 2.5, OutputCostPerM: 12.5},
			},
		},
	}

	cfg := domain.Config{
		Version: 1,
		Roles: map[domain.Role][]string{
			domain.RolePrimary:  {"claude-like:sonnet"},
			domain.RoleReviewer: {"claude-like:*", "codex-like:*"},
		},
		Policy: domain.Policy{
			RequireDistinctVendor: []domain.Role{domain.RolePrimary, domain.RoleReviewer},
		},
	}

	res, err := resolve.ResolveRoles(cfg, adapters, time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	primary := res[domain.RolePrimary]
	if primary.SelectedAgentID != "claude-like" || primary.Vendor != "Anthropic" {
		t.Fatalf("primary mismatch: %+v", primary)
	}

	reviewer := res[domain.RoleReviewer]
	if reviewer.SelectedAgentID != "codex-like" || reviewer.Vendor != "OpenAI" {
		t.Fatalf("reviewer mismatch: %+v", reviewer)
	}
	if len(reviewer.FallbackChain) != 1 || reviewer.FallbackChain[0] != "claude-like:* (vendor conflict: Anthropic)" {
		t.Errorf("reviewer fallback chain mismatch: %+v", reviewer.FallbackChain)
	}
	if len(reviewer.Steps) != 2 {
		t.Fatalf("expected 2 steps for reviewer, got %d", len(reviewer.Steps))
	}
	if reviewer.Steps[0].Status != domain.StepStatusSkippedVendorConflict {
		t.Errorf("step 0 status mismatch: %s", reviewer.Steps[0].Status)
	}
	if reviewer.Steps[1].Status != domain.StepStatusSelected {
		t.Errorf("step 1 status mismatch: %s", reviewer.Steps[1].Status)
	}
}

func TestResolveAuthFailure(t *testing.T) {
	adapters := []domain.Adapter{
		{
			ID:        "agent-unauth",
			Installed: true,
			Vendor:    "VendorA",
			Probes: map[domain.Tier]domain.ProbeResult{
				domain.TierL1: {
					Status:     "failed",
					ErrorClass: domain.ErrorClassUnauthenticated,
				},
			},
		},
	}

	cfg := domain.Config{
		Version: 1,
		Roles: map[domain.Role][]string{
			domain.RolePrimary: {"agent-unauth:model-1"},
		},
	}

	res, err := resolve.ResolveRoles(cfg, adapters, time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	primary := res[domain.RolePrimary]
	if primary.Healthy {
		t.Errorf("expected unhealthy primary, got healthy")
	}
	if len(primary.Steps) != 1 || primary.Steps[0].Status != domain.StepStatusSkippedUnauth {
		t.Errorf("unexpected step: %+v", primary.Steps)
	}
	if len(primary.FallbackChain) != 1 || primary.FallbackChain[0] != "agent-unauth:model-1 (unauthenticated)" {
		t.Errorf("unexpected fallback chain: %+v", primary.FallbackChain)
	}
}

func TestResolveEmptyCandidates(t *testing.T) {
	cfg := domain.Config{
		Version: 1,
		Roles:   map[domain.Role][]string{},
	}

	res, err := resolve.ResolveRoles(cfg, nil, time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, role := range []domain.Role{domain.RolePrimary, domain.RoleSubagent, domain.RoleAuxiliary, domain.RoleReviewer} {
		r, ok := res[role]
		if !ok {
			t.Fatalf("missing role %s", role)
		}
		if r.Healthy {
			t.Errorf("expected role %s to be unhealthy", role)
		}
		if r.SelectedAgentID != "none" {
			t.Errorf("expected selectedAgentId none, got %q", r.SelectedAgentID)
		}
		if r.SelectedModel != "None available" {
			t.Errorf("expected selectedModel 'None available', got %q", r.SelectedModel)
		}
		if r.EstimatedCostPerM != 0 {
			t.Errorf("expected cost 0, got %f", r.EstimatedCostPerM)
		}
		if len(r.FallbackChain) != 0 {
			t.Errorf("expected empty fallback chain, got %+v", r.FallbackChain)
		}
		if len(r.Steps) != 0 {
			t.Errorf("expected empty steps, got %+v", r.Steps)
		}
	}
}

func TestResolveWildcardAndCostDefaults(t *testing.T) {
	adapters := []domain.Adapter{
		{
			ID:        "no-models-agent",
			Installed: true,
			Vendor:    "VendorNone",
		},
		{
			ID:        "zero-cost-agent",
			Installed: true,
			Vendor:    "VendorZero",
			Models: []domain.ModelInfo{
				{ID: "m-zero", DisplayName: "Zero Cost"},
			},
		},
	}

	cfg := domain.Config{
		Version: 1,
		Roles: map[domain.Role][]string{
			domain.RolePrimary:   {"no-models-agent"},
			domain.RoleAuxiliary: {"zero-cost-agent:*"},
		},
	}

	res, err := resolve.ResolveRoles(cfg, adapters, time.Now(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	primary := res[domain.RolePrimary]
	if primary.SelectedModel != "default" {
		t.Errorf("expected default model, got %q", primary.SelectedModel)
	}
	if primary.EstimatedCostPerM != 4.0 { // Default input 1.0 + output 3.0
		t.Errorf("expected default cost 4.0, got %f", primary.EstimatedCostPerM)
	}

	aux := res[domain.RoleAuxiliary]
	if aux.SelectedModel != "Zero Cost" {
		t.Errorf("expected 'Zero Cost', got %q", aux.SelectedModel)
	}
	if aux.EstimatedCostPerM != 4.0 { // 0 in costs fall back to 1.0 + 3.0 = 4.0
		t.Errorf("expected default cost 4.0, got %f", aux.EstimatedCostPerM)
	}
}

func TestResolveCustomRoleOrder(t *testing.T) {
	adapters := []domain.Adapter{
		{
			ID:        "agent-x",
			Installed: true,
			Vendor:    "VendorX",
			Models: []domain.ModelInfo{
				{ID: "mod-x", DisplayName: "Model X"},
			},
		},
	}

	customRole := domain.Role("custom-qa")
	cfg := domain.Config{
		Version: 1,
		Roles: map[domain.Role][]string{
			customRole: {"agent-x:mod-x"},
		},
	}

	res, err := resolve.ResolveRoles(cfg, adapters, time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	qa, ok := res[customRole]
	if !ok || !qa.Healthy || qa.SelectedAgentID != "agent-x" {
		t.Errorf("custom role resolution failed: %+v", qa)
	}
}
