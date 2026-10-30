package app_test

import (
	"context"
	"testing"

	"github.com/soundcheck/soundcheck/internal/app"
	edgeExec "github.com/soundcheck/soundcheck/internal/edge/exec"
	"github.com/soundcheck/soundcheck/internal/edge/store"
)

func TestAppRunPrompt_FallbackToWorkingAgent(t *testing.T) {
	bin := getFakeCLI(t)
	defs := testDefinitions(bin)
	for i := range defs {
		defs[i].HeadlessCommand = bin
	}
	runner := edgeExec.NewPipeRunner()
	cache := store.NewTTLCache(nil)

	a, err := app.New(
		app.WithRunner(runner),
		app.WithCache(cache),
		app.WithDefinitions(defs),
	)
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
	if !res.Success {
		t.Fatal("expected prompt run to succeed with fakecli")
	}

	t.Run("explicit unknown target returns error", func(t *testing.T) {
		_, err := a.RunPrompt(context.Background(), "hello", "definitely-not-an-agent")
		if err == nil {
			t.Fatal("expected error for unknown target, got nil")
		}
	})

	t.Run("explicit role targets role candidates", func(t *testing.T) {
		cfg := a.Config()
		cfg.Roles["custom_worker"] = []string{"claude:model-a"}
		a.SetConfig(cfg)

		res, err := a.RunPrompt(context.Background(), "hello", "custom_worker")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.AgentUsed != "claude" {
			t.Fatalf("expected agent claude, got %s", res.AgentUsed)
		}
	})
}
