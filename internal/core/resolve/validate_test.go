package resolve

import (
	"strings"
	"testing"

	"github.com/soundcheck/soundcheck/internal/core/domain"
)

func TestValidateConfig(t *testing.T) {
	ok := domain.Adapter{ID: "a", Name: "A", Binary: "a", Installed: true, Models: []domain.ModelInfo{{ID: "m1"}}}
	noAuth := domain.Adapter{ID: "b", Name: "B", Binary: "b", Installed: true,
		Probes: map[domain.Tier]domain.ProbeResult{domain.TierL1: {Status: "failed", ErrorClass: domain.ErrorClassUnauthenticated}}}
	missing := domain.Adapter{ID: "c", Name: "C", Binary: "c"}
	cfg := domain.Config{Roles: map[domain.Role][]string{domain.RolePrimary: {
		"a:m1", "a:*", "a", "a:nope", "b:x", "c:x", "zzz:x",
	}}}
	want := []string{"", "", "", "not in the A catalog", "not logged in", "not installed", "Unknown agent"}
	got := ValidateConfig(cfg, []domain.Adapter{ok, noAuth, missing})
	if len(got) != len(want) {
		t.Fatalf("got %d checks, want %d", len(got), len(want))
	}
	for i, w := range want {
		if (w == "") != got[i].Verified || !strings.Contains(got[i].Reason, w) {
			t.Errorf("%s: verified=%v reason=%q, want reason containing %q", got[i].Candidate, got[i].Verified, got[i].Reason, w)
		}
	}
}
