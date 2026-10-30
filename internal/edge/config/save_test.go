package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/soundcheck/soundcheck/internal/core/domain"
)

func TestSaveRoundTripAndRejectInvalid(t *testing.T) {
	p := filepath.Join(t.TempDir(), "agent-doctor.yaml")
	cfg := domain.Config{
		Version: 1,
		Roles:   map[domain.Role][]string{domain.RolePrimary: {"claude:claude-sonnet-5-5"}, domain.RoleReviewer: {"codex:gpt-4o"}},
		Policy:  domain.Policy{RequireDistinctVendor: []domain.Role{domain.RolePrimary, domain.RoleReviewer}, MaxCostUSDPerRun: 2, Probe: domain.ProbePolicy{TTLMinutes: 10}},
	}
	if err := Save(p, cfg); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := Load(p)
	if err != nil || !reflect.DeepEqual(got.Roles, cfg.Roles) {
		t.Fatalf("reload mismatch: %v %+v", err, got)
	}
	before, _ := os.ReadFile(p) //nolint:gosec
	if err := Save(p, domain.Config{Version: 99}); err == nil {
		t.Fatal("expected invalid config rejected")
	}
	after, _ := os.ReadFile(p) //nolint:gosec
	if string(before) != string(after) {
		t.Fatal("file changed on invalid save")
	}
}
