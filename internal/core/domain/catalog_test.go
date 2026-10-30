package domain

import "testing"

func TestFindModelFamily(t *testing.T) {
	// Canonical
	fam := FindModelFamily("claude-sonnet-5-5")
	if fam == nil || fam.CanonicalName != "claude-sonnet-5-5" {
		t.Fatalf("expected to find claude-sonnet-5-5 canonical")
	}

	// Alias
	fam = FindModelFamily("sonnet-5.5")
	if fam == nil || fam.CanonicalName != "claude-sonnet-5-5" {
		t.Fatalf("expected to resolve alias sonnet-5.5")
	}

	// OpenCode specific path
	fam = FindModelFamily("anthropic/claude-sonnet-5-5")
	if fam == nil || fam.CanonicalName != "claude-sonnet-5-5" {
		t.Fatalf("expected to resolve opencode provider model string")
	}

	// Non-existent
	fam = FindModelFamily("unknown-model-xyz")
	if fam != nil {
		t.Fatalf("expected nil for unknown model")
	}
}
