package app

import (
	"context"
	"fmt"

	"github.com/soundcheck/soundcheck/internal/core/domain"
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

		adapter, err := a.ProbeOne(ctx, providerID)
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
