// Package adapters provides declarative embedded CLI adapter definitions and configurations.
package adapters

import (
	"embed"
	"fmt"
	"sync"

	"gopkg.in/yaml.v3"

	"github.com/soundcheck/soundcheck/internal/core/domain"
)

// FS contains the embedded adapter YAML definition files.
//
//go:embed *.yaml
var FS embed.FS

var (
	loadOnce   sync.Once
	cachedDefs []Definition
	defsByID   map[string]Definition
)

var defaultOrder = []string{
	"claude",
	"codex",
	"opencode",
	"agy",
	"gemini",
	"copilot",
}

// Definition defines the declarative metadata and tier probe commands for an agent CLI.
type Definition struct {
	domain.Adapter `yaml:",inline"`
	ProbeSpecs     map[domain.Tier]domain.Probe `json:"probeSpecs,omitempty" yaml:"probeSpecs,omitempty"`
}

// ProbeSpec returns the execution specification for a tier probe.
func (d Definition) ProbeSpec(tier domain.Tier) (domain.Probe, bool) {
	if d.ProbeSpecs == nil {
		return domain.Probe{}, false
	}
	spec, ok := d.ProbeSpecs[tier]
	return spec, ok
}

// ToAdapter converts a Definition into a domain.Adapter with initialized pending probes.
func (d Definition) ToAdapter() domain.Adapter {
	a := d.Adapter
	if a.Probes == nil {
		a.Probes = make(map[domain.Tier]domain.ProbeResult, 4)
	}
	for _, tier := range []domain.Tier{domain.TierL0, domain.TierL1, domain.TierL2, domain.TierL3} {
		if _, ok := a.Probes[tier]; !ok {
			a.Probes[tier] = domain.ProbeResult{
				Level:      tier,
				Status:     "pending",
				ErrorClass: domain.ErrorClassOK,
				Message:    "Not run yet",
			}
		}
	}
	return a
}

// Parse decodes raw YAML bytes into an adapter Definition.
func Parse(data []byte) (Definition, error) {
	var def Definition
	if err := yaml.Unmarshal(data, &def); err != nil {
		return Definition{}, fmt.Errorf("unmarshal adapter YAML: %w", err)
	}
	return def, nil
}

// LoadAll loads all embedded adapter definitions in canonical order.
func LoadAll() ([]Definition, error) {
	out := make([]Definition, 0, len(defaultOrder))
	for _, id := range defaultOrder {
		data, err := FS.ReadFile(id + ".yaml")
		if err != nil {
			return nil, fmt.Errorf("read embedded %s.yaml: %w", id, err)
		}
		def, err := Parse(data)
		if err != nil {
			return nil, err
		}
		out = append(out, def)
	}
	return out, nil
}

func initCache() {
	loadOnce.Do(func() {
		defs, err := LoadAll()
		if err != nil {
			panic(fmt.Sprintf("failed loading embedded adapters: %v", err))
		}
		cachedDefs = defs
		defsByID = make(map[string]Definition, len(defs))
		for _, d := range defs {
			defsByID[d.ID] = d
		}
	})
}

// All returns all embedded adapter definitions in canonical order.
func All() []Definition {
	initCache()
	res := make([]Definition, len(cachedDefs))
	copy(res, cachedDefs)
	return res
}

// Get returns the adapter definition for the given ID, if found.
func Get(id string) (Definition, bool) {
	initCache()
	d, ok := defsByID[id]
	return d, ok
}

// DefaultAdapters returns domain adapters for all embedded definitions.
func DefaultAdapters() []domain.Adapter {
	defs := All()
	adapters := make([]domain.Adapter, len(defs))
	for i, d := range defs {
		adapters[i] = d.ToAdapter()
	}
	return adapters
}
