package adapters_test

import (
	"testing"

	"github.com/soundcheck/soundcheck/internal/adapters"
	"github.com/soundcheck/soundcheck/internal/core/domain"
)

func TestLoadAllEmbeddedAdapters(t *testing.T) {
	defs, err := adapters.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}

	expectedIDs := []string{"claude", "codex", "opencode", "agy", "gemini", "copilot"}
	if len(defs) != len(expectedIDs) {
		t.Fatalf("expected %d adapters, got %d", len(expectedIDs), len(defs))
	}

	for i, id := range expectedIDs {
		if defs[i].ID != id {
			t.Errorf("expected def[%d] id %s, got %s", i, id, defs[i].ID)
		}
		if defs[i].Name == "" {
			t.Errorf("adapter %s has empty Name", id)
		}
		if defs[i].Binary == "" {
			t.Errorf("adapter %s has empty Binary", id)
		}
		if len(defs[i].Models) == 0 {
			t.Errorf("adapter %s has no models", id)
		}
		for _, tier := range []domain.Tier{domain.TierL0, domain.TierL1, domain.TierL2, domain.TierL3} {
			if _, ok := defs[i].ProbeSpec(tier); !ok {
				t.Errorf("adapter %s missing probe spec for %s", id, tier)
			}
		}
	}
}

func TestGetAndToAdapter(t *testing.T) {
	claude, ok := adapters.Get("claude")
	if !ok {
		t.Fatal("expected claude adapter to be found")
	}

	if claude.Vendor != "Anthropic" {
		t.Errorf("expected vendor Anthropic, got %s", claude.Vendor)
	}

	ad := claude.ToAdapter()
	if len(ad.Probes) != 4 {
		t.Errorf("expected 4 probes in adapter, got %d", len(ad.Probes))
	}
	if ad.Probes[domain.TierL0].Status != "pending" {
		t.Errorf("expected pending status, got %s", ad.Probes[domain.TierL0].Status)
	}
}
