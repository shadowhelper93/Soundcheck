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

	// claude failed for claude-3-7-sonnet
	alts, err := a.FindAlternativesForModel(context.Background(), "claude", "claude-3-7-sonnet")
	if err != nil {
		t.Fatalf("unexpected error finding alternatives: %v", err)
	}

	// If opencode or another provider is authenticated on this machine, it should be discovered
	t.Logf("Found %d verified alternatives for claude-3-7-sonnet", len(alts))
	for _, alt := range alts {
		t.Logf("  Alternative: %s (model: %s)", alt.ProviderID, alt.ModelID)
	}
}
