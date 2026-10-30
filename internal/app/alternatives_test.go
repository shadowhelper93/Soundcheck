package app

import (
	"context"
	"testing"
)

func TestFindAlternativesForModel(t *testing.T) {
	a, err := New()
	if err != nil {
		t.Fatalf("failed to init app: %v", err)
	}

	// claude failed for claude-sonnet-5-5
	alts, err := a.FindAlternativesForModel(context.Background(), "claude", "claude-sonnet-5-5")
	if err != nil {
		t.Fatalf("unexpected error finding alternatives: %v", err)
	}

	// If opencode or another provider is authenticated on this machine, it should be discovered
	t.Logf("Found %d verified alternatives for claude-sonnet-5-5", len(alts))
	for _, alt := range alts {
		t.Logf("  Alternative: %s (model: %s)", alt.ProviderID, alt.ModelID)
	}
}
