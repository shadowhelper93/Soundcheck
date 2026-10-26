package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/soundcheck/soundcheck/internal/core/domain"
	"github.com/soundcheck/soundcheck/internal/edge/config"
)

func TestLoadCanonicalAgentDoctorYAML(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	// Locate agent-doctor.yaml at workspace root
	root := filepath.Clean(filepath.Join(wd, "..", "..", ".."))
	configPath := filepath.Join(root, "agent-doctor.yaml")

	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("Load(agent-doctor.yaml) failed: %v", err)
	}

	if cfg.Version != 1 {
		t.Errorf("expected version 1, got %d", cfg.Version)
	}
	if len(cfg.Roles) != 4 {
		t.Errorf("expected 4 roles, got %d", len(cfg.Roles))
	}
	if len(cfg.Roles[domain.RolePrimary]) == 0 {
		t.Errorf("expected primary candidates, got none")
	}
	if cfg.Policy.MaxCostUSDPerRun != 5.0 {
		t.Errorf("expected max cost 5.0, got %f", cfg.Policy.MaxCostUSDPerRun)
	}
	if cfg.Policy.Probe.TTLMinutes != 30 {
		t.Errorf("expected ttl 30, got %f", cfg.Policy.Probe.TTLMinutes)
	}
	if len(cfg.Policy.RequireDistinctVendor) != 2 {
		t.Errorf("expected 2 distinct vendor roles, got %d", len(cfg.Policy.RequireDistinctVendor))
	}
}

func TestLoadNonexistentFile(t *testing.T) {
	_, err := config.Load("/nonexistent/agent-doctor.yaml")
	if err == nil {
		t.Fatal("expected error for nonexistent file, got nil")
	}
}

func TestParseEmptyData(t *testing.T) {
	_, err := config.Parse([]byte("   \n\t  "))
	if err == nil {
		t.Fatal("expected error for empty config, got nil")
	}
}

func TestStrictUnknownFieldsWithLineNumbers(t *testing.T) {
	tests := []struct {
		name       string
		yaml       string
		errSubstr  string
		lineSubstr string
	}{
		{
			name:       "unknown top level field",
			yaml:       "version: 1\nextra_field: true\nroles:\n  primary: ['a']\n",
			errSubstr:  "extra_field",
			lineSubstr: "line 2",
		},
		{
			name:       "unknown policy field",
			yaml:       "version: 1\nroles:\n  primary: ['a']\npolicy:\n  bad_policy: 123\n  require_distinct_vendor: [primary, reviewer]\n",
			errSubstr:  "bad_policy",
			lineSubstr: "line 5",
		},
		{
			name:       "unknown probe field",
			yaml:       "version: 1\nroles:\n  primary: ['a']\npolicy:\n  require_distinct_vendor: [primary, reviewer]\n  probe:\n    deep: false\n    unknown_probe: true\n",
			errSubstr:  "unknown_probe",
			lineSubstr: "line 8",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := config.Parse([]byte(tc.yaml))
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.errSubstr)
			}
			errStr := err.Error()
			if !strings.Contains(errStr, tc.errSubstr) {
				t.Errorf("expected error to contain %q, got %q", tc.errSubstr, errStr)
			}
			if !strings.Contains(errStr, tc.lineSubstr) {
				t.Errorf("expected error to contain %q, got %q", tc.lineSubstr, errStr)
			}
		})
	}
}

func TestSemanticValidationRoles(t *testing.T) {
	t.Run("version less than 1", func(t *testing.T) {
		y := "version: 0\nroles:\n  primary: ['a']\n  reviewer: ['b']\npolicy:\n  require_distinct_vendor: [primary, reviewer]\n"
		_, err := config.Parse([]byte(y))
		if err == nil || !strings.Contains(err.Error(), "version must be >= 1") {
			t.Fatalf("expected version error, got %v", err)
		}
	})

	t.Run("empty roles", func(t *testing.T) {
		y := "version: 1\nroles: {}\npolicy:\n  require_distinct_vendor: [primary, reviewer]\n"
		_, err := config.Parse([]byte(y))
		if err == nil || !strings.Contains(err.Error(), "roles must not be empty") {
			t.Fatalf("expected empty roles error, got %v", err)
		}
	})

	t.Run("role with no candidates", func(t *testing.T) {
		y := "version: 1\nroles:\n  primary: []\n  reviewer: ['b']\npolicy:\n  require_distinct_vendor: [primary, reviewer]\n"
		_, err := config.Parse([]byte(y))
		if err == nil || !strings.Contains(err.Error(), "must specify at least one candidate") {
			t.Fatalf("expected candidate error, got %v", err)
		}
	})
}

func TestSemanticValidationPolicy(t *testing.T) {
	baseYAML := "version: 1\nroles:\n  primary: ['a']\n  reviewer: ['b']\npolicy:\n  require_distinct_vendor: [primary, reviewer]\n"

	t.Run("require_distinct_vendor with fewer than 2 roles", func(t *testing.T) {
		y := "version: 1\nroles:\n  primary: ['a']\npolicy:\n  require_distinct_vendor: [primary]\n"
		_, err := config.Parse([]byte(y))
		if err == nil || !strings.Contains(err.Error(), "at least 2 roles") {
			t.Fatalf("expected at least 2 roles error, got %v", err)
		}
	})

	t.Run("require_distinct_vendor with duplicate roles", func(t *testing.T) {
		y := "version: 1\nroles:\n  primary: ['a']\npolicy:\n  require_distinct_vendor: [primary, primary]\n"
		_, err := config.Parse([]byte(y))
		if err == nil || !strings.Contains(err.Error(), "must not be duplicated") {
			t.Fatalf("expected duplicate role error, got %v", err)
		}
	})

	t.Run("require_distinct_vendor role missing from roles", func(t *testing.T) {
		y := "version: 1\nroles:\n  primary: ['a']\n  reviewer: ['b']\npolicy:\n  require_distinct_vendor: [primary, missing]\n"
		_, err := config.Parse([]byte(y))
		if err == nil || !strings.Contains(err.Error(), "does not exist in roles") {
			t.Fatalf("expected missing role error, got %v", err)
		}
	})

	t.Run("ttl_minutes zero or negative", func(t *testing.T) {
		y := baseYAML + "  probe:\n    ttl_minutes: 0\n"
		_, err := config.Parse([]byte(y))
		if err == nil || !strings.Contains(err.Error(), "ttl_minutes must be > 0") {
			t.Fatalf("expected ttl error, got %v", err)
		}
	})

	t.Run("negative max cost", func(t *testing.T) {
		y := baseYAML + "  max_cost_usd_per_run: -1.0\n"
		_, err := config.Parse([]byte(y))
		if err == nil || !strings.Contains(err.Error(), "max_cost_usd_per_run must be >= 0") {
			t.Fatalf("expected max cost error, got %v", err)
		}
	})
}

func TestDefaultValuesApplied(t *testing.T) {
	y := `
version: 1
roles:
  primary: ["claude:1"]
  reviewer: ["codex:1"]
policy:
  require_distinct_vendor: [primary, reviewer]
`
	cfg, err := config.Parse([]byte(y))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Policy.MaxCostUSDPerRun != 5.0 {
		t.Errorf("expected default max cost 5.0, got %f", cfg.Policy.MaxCostUSDPerRun)
	}
	if cfg.Policy.Probe.TTLMinutes != 30.0 {
		t.Errorf("expected default ttl 30.0, got %f", cfg.Policy.Probe.TTLMinutes)
	}
	if cfg.Policy.Probe.Deep != false {
		t.Errorf("expected default deep false, got %v", cfg.Policy.Probe.Deep)
	}
}
