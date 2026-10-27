package app

import (
	"context"
	"testing"
)

func TestAppRunPrompt_FallbackToWorkingAgent(t *testing.T) {
	a, err := New()
	if err != nil {
		t.Fatalf("failed to init app: %v", err)
	}

	res, err := a.RunPrompt(context.Background(), "hello world")
	if err != nil {
		t.Fatalf("unexpected error running prompt: %v", err)
	}

	if res.AgentUsed == "" {
		t.Fatal("expected an agent to be used, got empty")
	}
	if len(res.Attempts) == 0 {
		t.Fatal("expected at least one attempt recorded")
	}
}
