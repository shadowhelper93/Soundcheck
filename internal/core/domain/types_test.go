package domain_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/soundcheck/soundcheck/internal/core/domain"
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

func TestManifestGoldenUnmarshal(t *testing.T) {
	root := findRepoRoot(t)
	path := filepath.Clean(filepath.Join(root, "testdata", "golden", "manifest.json"))
	data, err := os.ReadFile(path) // #nosec G304
	if err != nil {
		t.Fatalf("failed reading manifest.json: %v", err)
	}

	var m domain.Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("failed unmarshaling Manifest: %v", err)
	}

	if m.SchemaVersion != "2026.1" {
		t.Errorf("schemaVersion = %q, want 2026.1", m.SchemaVersion)
	}
	if m.Summary.TotalAgents != 7 {
		t.Errorf("summary.totalAgents = %d, want 7", m.Summary.TotalAgents)
	}
	if m.Summary.QuirksProtected != 10 {
		t.Errorf("summary.quirksProtected = %d, want 10", m.Summary.QuirksProtected)
	}
	if len(m.Agents) != 7 {
		t.Errorf("agents count = %d, want 7", len(m.Agents))
	}
	if primary, ok := m.ResolvedRoles[domain.RolePrimary]; !ok || primary.Agent != "claude" {
		t.Errorf("primary role agent = %q, want claude", primary.Agent)
	}
}

func TestResolveGoldenUnmarshal(t *testing.T) {
	root := findRepoRoot(t)
	path := filepath.Clean(filepath.Join(root, "testdata", "golden", "resolve.json"))
	data, err := os.ReadFile(path) // #nosec G304
	if err != nil {
		t.Fatalf("failed reading resolve.json: %v", err)
	}

	var assignments map[domain.Role]domain.Assignment
	if err := json.Unmarshal(data, &assignments); err != nil {
		t.Fatalf("failed unmarshaling assignments: %v", err)
	}

	for _, role := range []domain.Role{domain.RolePrimary, domain.RoleSubagent, domain.RoleAuxiliary, domain.RoleReviewer} {
		asgn, ok := assignments[role]
		if !ok {
			t.Errorf("missing assignment for role %s", role)
			continue
		}
		if asgn.Role != role {
			t.Errorf("assignment role mismatch: got %s, want %s", asgn.Role, role)
		}
		if !asgn.Healthy {
			t.Errorf("role %s not healthy", role)
		}
	}
}
