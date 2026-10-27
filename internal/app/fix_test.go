package app

import (
	"context"
	"testing"
)

func TestAppFixUnknownAdapter(t *testing.T) {
	a, err := New()
	if err != nil {
		t.Fatalf("failed to init app: %v", err)
	}

	_, err = a.Fix(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent adapter, got nil")
	}
}

func TestAppFixKnownAdapter(t *testing.T) {
	a, err := New()
	if err != nil {
		t.Fatalf("failed to init app: %v", err)
	}

	res, err := a.Fix(context.Background(), "claude")
	if err != nil {
		t.Fatalf("unexpected error for claude fix: %v", err)
	}
	if res.AdapterID != "claude" {
		t.Fatalf("expected adapterID claude, got %s", res.AdapterID)
	}
	if res.RemediationCmd == "" {
		t.Fatal("expected non-empty remediation command")
	}
}
