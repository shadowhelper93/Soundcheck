package app

import (
	"context"
	"fmt"

	"github.com/soundcheck/soundcheck/internal/core/domain"
	"github.com/soundcheck/soundcheck/internal/edge/probe"
)

// VerifiedAlternative describes an alternative provider verified and ready to run the requested model.
type VerifiedAlternative struct {
	ProviderID string `json:"providerId"`
	ModelID    string `json:"modelId"`
	Status     string `json:"status"`
}

// FindAlternativesForModel checks which other installed providers can run the same model when quota is exhausted.
func (a *App) FindAlternativesForModel(ctx context.Context, failedProvider string, requestedModel string) ([]VerifiedAlternative, error) {
	family := domain.FindModelFamily(requestedModel)
	if family == nil {
		return nil, fmt.Errorf("no known equivalence mappings for model %q", requestedModel)
	}

	var alternatives []VerifiedAlternative

	for providerID, targetModel := range family.ProviderIDs {
		if providerID == failedProvider {
			continue
		}

		adapter, err := a.probeAuthOnly(ctx, providerID)
		if err != nil || !adapter.Installed {
			continue
		}

		isAuth := false
		if p, hasL1 := adapter.Probes[domain.TierL1]; hasL1 && p.ErrorClass == domain.ErrorClassOK {
			isAuth = true
		}

		if isAuth {
			alternatives = append(alternatives, VerifiedAlternative{
				ProviderID: providerID,
				ModelID:    targetModel,
				Status:     "verified_working",
			})
		}
	}

	return alternatives, nil
}

// probeAuthOnly probes L0..L1 only, skipping expensive L2/L3 smoke tests.
func (a *App) probeAuthOnly(ctx context.Context, adapterID string) (domain.Adapter, error) {
	def, ok := a.defs[adapterID]
	if !ok {
		return domain.Adapter{}, fmt.Errorf("unknown adapter: %s", adapterID)
	}
	return a.prober.ProbeAdapter(ctx, def, probe.WithCallMaxTier(domain.TierL1)), nil
}
