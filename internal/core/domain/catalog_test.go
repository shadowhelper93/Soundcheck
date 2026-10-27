package domain

import "testing"

func TestFindModelFamily(t *testing.T) {
	// Canonical
	fam := FindModelFamily("claude-3-7-sonnet")
	if fam == nil || fam.CanonicalName != "claude-3-7-sonnet" {
		t.Fatalf("expected to find claude-3-7-sonnet canonical")
	}

	// Alias
	fam = FindModelFamily("sonnet-3.7")
	if fam == nil || fam.CanonicalName != "claude-3-7-sonnet" {
		t.Fatalf("expected to resolve alias sonnet-3.7")
	}

	// OpenCode specific path
	fam = FindModelFamily("anthropic/claude-3-7-sonnet")
	if fam == nil || fam.CanonicalName != "claude-3-7-sonnet" {
		t.Fatalf("expected to resolve opencode provider model string")
	}

	// Non-existent
	fam = FindModelFamily("unknown-model-xyz")
	if fam != nil {
		t.Fatalf("expected nil for unknown model")
	}
}
