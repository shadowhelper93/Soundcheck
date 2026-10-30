package probe

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/soundcheck/soundcheck/internal/adapters"
	"github.com/soundcheck/soundcheck/internal/core/domain"
	"github.com/soundcheck/soundcheck/internal/edge/exec"
)

// discoverModels runs the adapter's modelsCommand and returns the account's live models.
// ok is false when no command is declared or it fails; callers then keep the static catalog.
func discoverModels(ctx context.Context, runner exec.Runner, def adapters.Definition, binPath string) ([]domain.ModelInfo, bool) {
	if def.ModelsCommand == "" {
		return nil, false
	}
	res, err := runner.Run(ctx, buildCmd(binPath, def.ModelsCommand, nil, 10*time.Second))
	if err != nil || res.ExitCode != 0 {
		return nil, false
	}
	return parseLiveModels(def, res.Stdout)
}

func parseLiveModels(def adapters.Definition, stdout []byte) ([]domain.ModelInfo, bool) {
	var payload struct {
		Models []struct {
			Slug        string `json:"slug"`
			DisplayName string `json:"display_name"`
		} `json:"models"`
	}
	if json.Unmarshal(stdout, &payload) != nil || len(payload.Models) == 0 {
		return nil, false
	}
	known := make(map[string]domain.ModelInfo, len(def.Models))
	for _, m := range def.Models {
		known[m.ID] = m
	}
	out := make([]domain.ModelInfo, 0, len(payload.Models))
	for _, m := range payload.Models {
		out = append(out, liveModel(def, known, m.Slug, m.DisplayName))
	}
	return out, true
}

// liveModel keeps vendor-doc pricing for IDs we know; unknown IDs carry no price (never invented).
func liveModel(def adapters.Definition, known map[string]domain.ModelInfo, id, name string) domain.ModelInfo {
	if m, ok := known[id]; ok {
		m.Verified = true
		return m
	}
	if name == "" {
		name = id
	}
	return domain.ModelInfo{ID: id, DisplayName: fmt.Sprintf("%s (live)", name), Vendor: def.Vendor, Verified: true}
}
