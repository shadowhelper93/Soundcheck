// Package config implements strict YAML configuration loading and semantic validation for Soundcheck.
package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/soundcheck/soundcheck/internal/core/domain"
)

const (
	defaultMaxCostUSD = 5.0
	defaultTTLMinutes = 30.0
)

type rawConfig struct {
	Version int                      `yaml:"version"`
	Roles   map[domain.Role][]string `yaml:"roles"`
	Policy  rawPolicy                `yaml:"policy"`
}

type rawPolicy struct {
	RequireDistinctVendor []domain.Role   `yaml:"require_distinct_vendor"`
	MaxCostUSDPerRun      *float64        `yaml:"max_cost_usd_per_run"`
	Probe                 *rawProbePolicy `yaml:"probe"`
}

type rawProbePolicy struct {
	Deep       *bool    `yaml:"deep"`
	TTLMinutes *float64 `yaml:"ttl_minutes"`
}

// Load reads and parses a Soundcheck configuration file from path.
func Load(path string) (domain.Config, error) {
	// #nosec G304
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return domain.Config{}, fmt.Errorf("read config file: %w", err)
	}
	return Parse(data)
}

// Parse strictly parses and validates Soundcheck YAML configuration bytes.
func Parse(data []byte) (domain.Config, error) {
	raw, err := decodeYAML(data)
	if err != nil {
		return domain.Config{}, err
	}

	applyDefaults(&raw)
	if err := validateSemantics(raw); err != nil {
		return domain.Config{}, err
	}

	return toDomain(raw), nil
}

func decodeYAML(data []byte) (rawConfig, error) {
	if len(bytes.TrimSpace(data)) == 0 {
		return rawConfig{}, errors.New("configuration data is empty")
	}
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)

	var raw rawConfig
	if err := dec.Decode(&raw); err != nil {
		return rawConfig{}, fmt.Errorf("decode YAML: %w", err)
	}
	return raw, nil
}

func applyDefaults(raw *rawConfig) {
	if raw.Policy.MaxCostUSDPerRun == nil {
		cost := defaultMaxCostUSD
		raw.Policy.MaxCostUSDPerRun = &cost
	}
	if raw.Policy.Probe == nil {
		deep := false
		ttl := defaultTTLMinutes
		raw.Policy.Probe = &rawProbePolicy{
			Deep:       &deep,
			TTLMinutes: &ttl,
		}
		return
	}
	if raw.Policy.Probe.TTLMinutes == nil {
		ttl := defaultTTLMinutes
		raw.Policy.Probe.TTLMinutes = &ttl
	}
	if raw.Policy.Probe.Deep == nil {
		deep := false
		raw.Policy.Probe.Deep = &deep
	}
}

func validateSemantics(raw rawConfig) error {
	if raw.Version < 1 {
		return fmt.Errorf("version must be >= 1, got %d", raw.Version)
	}
	if len(raw.Roles) == 0 {
		return errors.New("roles must not be empty")
	}
	for role, candidates := range raw.Roles {
		if len(candidates) == 0 {
			return fmt.Errorf("role %q must specify at least one candidate", role)
		}
	}
	if err := validatePolicy(raw); err != nil {
		return err
	}
	return nil
}

func validatePolicy(raw rawConfig) error {
	if len(raw.Policy.RequireDistinctVendor) < 2 {
		return errors.New("policy.require_distinct_vendor must specify at least 2 roles")
	}
	seen := make(map[domain.Role]bool, len(raw.Policy.RequireDistinctVendor))
	for _, r := range raw.Policy.RequireDistinctVendor {
		if seen[r] {
			return fmt.Errorf("policy.require_distinct_vendor role %q must not be duplicated", r)
		}
		seen[r] = true
		if _, ok := raw.Roles[r]; !ok {
			return fmt.Errorf("policy.require_distinct_vendor role %q does not exist in roles", r)
		}
	}
	if raw.Policy.Probe != nil && raw.Policy.Probe.TTLMinutes != nil && *raw.Policy.Probe.TTLMinutes <= 0 {
		return fmt.Errorf("policy.probe.ttl_minutes must be > 0, got %v", *raw.Policy.Probe.TTLMinutes)
	}
	if raw.Policy.MaxCostUSDPerRun != nil && *raw.Policy.MaxCostUSDPerRun < 0 {
		return fmt.Errorf("policy.max_cost_usd_per_run must be >= 0, got %v", *raw.Policy.MaxCostUSDPerRun)
	}
	return nil
}

func toDomain(raw rawConfig) domain.Config {
	return domain.Config{
		Version: raw.Version,
		Roles:   raw.Roles,
		Policy: domain.Policy{
			RequireDistinctVendor: raw.Policy.RequireDistinctVendor,
			MaxCostUSDPerRun:      *raw.Policy.MaxCostUSDPerRun,
			Probe: domain.ProbePolicy{
				Deep:       *raw.Policy.Probe.Deep,
				TTLMinutes: *raw.Policy.Probe.TTLMinutes,
			},
		},
	}
}
