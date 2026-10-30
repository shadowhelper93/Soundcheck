package probe

import (
	"testing"

	"github.com/soundcheck/soundcheck/internal/adapters"
	"github.com/soundcheck/soundcheck/internal/core/domain"
)

func TestParseLiveModels(t *testing.T) {
	def := adapters.Definition{Adapter: domain.Adapter{Vendor: "OpenAI", Models: []domain.ModelInfo{{ID: "known", InputCostPerM: 1.5}}}}
	got, ok := parseLiveModels(def, []byte(`{"models":[{"slug":"known","display_name":"K"},{"slug":"new-x","display_name":"New X"}]}`))
	if !ok || len(got) != 2 {
		t.Fatalf("ok=%v got=%v", ok, got)
	}
	if got[0].InputCostPerM != 1.5 || got[1].InputCostPerM != 0 || got[1].DisplayName != "New X (live)" {
		t.Errorf("pricing must come only from the catalog: %+v", got)
	}
	if _, ok := parseLiveModels(def, []byte(`not json`)); ok {
		t.Error("garbage must fall back to the static catalog")
	}
}
