package workflow_test

import (
	"strings"
	"testing"
	"time"

	"github.com/soundcheck/soundcheck/internal/core/domain"
	"github.com/soundcheck/soundcheck/internal/core/workflow"
)

func TestStandardPatterns(t *testing.T) {
	patterns := workflow.DefaultPatterns()
	if len(patterns) != 2 {
		t.Fatalf("expected 2 default patterns, got %d", len(patterns))
	}

	p1, ok1 := workflow.GetPattern(workflow.PatternIDPlanImplementReview)
	if !ok1 || p1.ID != workflow.PatternIDPlanImplementReview {
		t.Fatalf("failed retrieving plan-implement-review pattern")
	}
	if err := workflow.ValidateDAG(p1); err != nil {
		t.Errorf("plan-implement-review DAG validation failed: %v", err)
	}

	order1, err := workflow.TopologicalSort(p1)
	if err != nil {
		t.Fatalf("topological sort failed: %v", err)
	}
	expected1 := []string{"plan", "implement", "review"}
	for i, id := range expected1 {
		if order1[i] != id {
			t.Errorf("stage order [%d] = %q, want %q", i, order1[i], id)
		}
	}

	p2, ok2 := workflow.GetPattern(workflow.PatternIDSpecTDDAudit)
	if !ok2 || p2.ID != workflow.PatternIDSpecTDDAudit {
		t.Fatalf("failed retrieving spec-tdd-audit pattern")
	}
	if err := workflow.ValidateDAG(p2); err != nil {
		t.Errorf("spec-tdd-audit DAG validation failed: %v", err)
	}

	order2, err := workflow.TopologicalSort(p2)
	if err != nil {
		t.Fatalf("topological sort failed: %v", err)
	}
	expected2 := []string{"spec", "tests", "implement", "audit"}
	for i, id := range expected2 {
		if order2[i] != id {
			t.Errorf("stage order [%d] = %q, want %q", i, order2[i], id)
		}
	}

	if _, notFound := workflow.GetPattern("unknown-pattern"); notFound {
		t.Errorf("expected false for unknown pattern")
	}
}

func assertDAGError(t *testing.T, pattern workflow.Pattern, wantErr string) {
	t.Helper()
	err := workflow.ValidateDAG(pattern)
	if err == nil {
		t.Fatalf("expected error containing %q, got nil", wantErr)
	}
	if !strings.Contains(err.Error(), wantErr) {
		t.Errorf("error = %q, want containing %q", err.Error(), wantErr)
	}
	if _, sortErr := workflow.TopologicalSort(pattern); sortErr == nil {
		t.Errorf("TopologicalSort expected error, got nil")
	}
}

func TestDAGValidationBasicErrors(t *testing.T) {
	tests := []struct {
		name    string
		pattern workflow.Pattern
		wantErr string
	}{
		{
			name:    "empty stages",
			pattern: workflow.Pattern{ID: "empty"},
			wantErr: "contains no stages",
		},
		{
			name: "empty stage id",
			pattern: workflow.Pattern{
				ID:     "empty-id",
				Stages: []workflow.Stage{{ID: "", Role: domain.RolePrimary}},
			},
			wantErr: "stage ID cannot be empty",
		},
		{
			name: "missing role",
			pattern: workflow.Pattern{
				ID:     "missing-role",
				Stages: []workflow.Stage{{ID: "s1", Role: ""}},
			},
			wantErr: "has no role specified",
		},
		{
			name: "duplicate stage id",
			pattern: workflow.Pattern{
				ID: "duplicate-id",
				Stages: []workflow.Stage{
					{ID: "s1", Role: domain.RolePrimary},
					{ID: "s1", Role: domain.RoleSubagent},
				},
			},
			wantErr: "duplicate stage ID",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertDAGError(t, tt.pattern, tt.wantErr)
		})
	}
}

func TestDAGValidationDependencyErrors(t *testing.T) {
	tests := []struct {
		name    string
		pattern workflow.Pattern
		wantErr string
	}{
		{
			name: "self dependency",
			pattern: workflow.Pattern{
				ID:     "self-dep",
				Stages: []workflow.Stage{{ID: "s1", Role: domain.RolePrimary, Needs: []string{"s1"}}},
			},
			wantErr: "cannot depend on itself",
		},
		{
			name: "unknown dependency",
			pattern: workflow.Pattern{
				ID:     "unknown-dep",
				Stages: []workflow.Stage{{ID: "s1", Role: domain.RolePrimary, Needs: []string{"nonexistent"}}},
			},
			wantErr: "unknown dependency",
		},
		{
			name: "two node cycle",
			pattern: workflow.Pattern{
				ID: "two-cycle",
				Stages: []workflow.Stage{
					{ID: "a", Role: domain.RolePrimary, Needs: []string{"b"}},
					{ID: "b", Role: domain.RoleSubagent, Needs: []string{"a"}},
				},
			},
			wantErr: "cycle detected",
		},
		{
			name: "three node cycle",
			pattern: workflow.Pattern{
				ID: "three-cycle",
				Stages: []workflow.Stage{
					{ID: "a", Role: domain.RolePrimary, Needs: []string{"c"}},
					{ID: "b", Role: domain.RoleSubagent, Needs: []string{"a"}},
					{ID: "c", Role: domain.RoleReviewer, Needs: []string{"b"}},
				},
			},
			wantErr: "cycle detected",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertDAGError(t, tt.pattern, tt.wantErr)
		})
	}
}

func TestGateEvaluation(t *testing.T) {
	// 1. Nil gate
	nilEval := workflow.EvaluateGate(nil, workflow.GateInput{})
	if !nilEval.Passed || len(nilEval.Failures) != 0 {
		t.Errorf("nil gate should pass, got %+v", nilEval)
	}

	// 2. Diff gate
	diffGate := &workflow.Gate{DiffNonEmpty: true}
	failDiff := workflow.EvaluateGate(diffGate, workflow.GateInput{HasDiff: false, DiffSize: 0})
	if failDiff.Passed || failDiff.DiffPassed {
		t.Errorf("empty diff should fail, got %+v", failDiff)
	}
	passDiff := workflow.EvaluateGate(diffGate, workflow.GateInput{HasDiff: true})
	if !passDiff.Passed || !passDiff.DiffPassed {
		t.Errorf("hasDiff=true should pass, got %+v", passDiff)
	}
	passDiffSize := workflow.EvaluateGate(diffGate, workflow.GateInput{DiffSize: 120})
	if !passDiffSize.Passed || !passDiffSize.DiffPassed {
		t.Errorf("diffSize>0 should pass, got %+v", passDiffSize)
	}

	// 3. Tests gate
	testGate := &workflow.Gate{Tests: "npm test"}
	passTest := workflow.EvaluateGate(testGate, workflow.GateInput{TestPassed: true, TestExitCode: 0})
	if !passTest.Passed || !passTest.TestPassed {
		t.Errorf("test passed should pass gate, got %+v", passTest)
	}
	failTestExit := workflow.EvaluateGate(testGate, workflow.GateInput{TestPassed: false, TestExitCode: 2})
	if failTestExit.Passed || failTestExit.TestPassed {
		t.Errorf("exit code 2 should fail test gate, got %+v", failTestExit)
	}
	failTestBool := workflow.EvaluateGate(testGate, workflow.GateInput{TestPassed: false, TestExitCode: 0})
	if failTestBool.Passed || failTestBool.TestPassed {
		t.Errorf("testPassed=false should fail test gate, got %+v", failTestBool)
	}

	// 4. MaxLoops gate
	loopGate := &workflow.Gate{MaxLoops: 2}
	passLoop := workflow.EvaluateGate(loopGate, workflow.GateInput{CurrentLoop: 2})
	if !passLoop.Passed || !passLoop.LoopPassed {
		t.Errorf("currentLoop=2 with max=2 should pass, got %+v", passLoop)
	}
	failLoop := workflow.EvaluateGate(loopGate, workflow.GateInput{CurrentLoop: 3})
	if failLoop.Passed || failLoop.LoopPassed {
		t.Errorf("currentLoop=3 with max=2 should fail, got %+v", failLoop)
	}

	// 5. Compound gate failures
	compoundGate := &workflow.Gate{
		DiffNonEmpty: true,
		Tests:        "cargo test",
		MaxLoops:     1,
	}
	compoundFail := workflow.EvaluateGate(compoundGate, workflow.GateInput{
		HasDiff:      false,
		TestExitCode: 1,
		CurrentLoop:  2,
	})
	if compoundFail.Passed || len(compoundFail.Failures) != 3 {
		t.Errorf("expected 3 failures, got %+v", compoundFail)
	}
}

func TestPreflightEvaluation(t *testing.T) {
	now := time.UnixMilli(1730000000000)
	futureCooldown := now.UnixMilli() + 30000
	pastCooldown := now.UnixMilli() - 30000

	healthy := domain.Adapter{
		ID:        "healthy-agent",
		Installed: true,
		Probes: map[domain.Tier]domain.ProbeResult{
			domain.TierL0: {Status: "passed"},
			domain.TierL1: {Status: "passed"},
		},
		CooldownUntil: &pastCooldown,
	}

	evalPass := workflow.EvaluatePreflight(healthy, now)
	if !evalPass.Passed || len(evalPass.Failures) != 0 {
		t.Errorf("healthy adapter expected preflight pass, got %+v", evalPass)
	}

	// Missing L0 / not installed
	uninstalled := domain.Adapter{
		ID:        "missing-binary",
		Installed: false,
		Probes: map[domain.Tier]domain.ProbeResult{
			domain.TierL1: {Status: "passed"},
		},
	}
	evalUninst := workflow.EvaluatePreflight(uninstalled, now)
	if evalUninst.Passed {
		t.Errorf("uninstalled adapter should fail preflight, got %+v", evalUninst)
	}

	// Missing L1 / unauth
	unauth := domain.Adapter{
		ID:        "unauth-agent",
		Installed: true,
		Probes: map[domain.Tier]domain.ProbeResult{
			domain.TierL0: {Status: "passed"},
			domain.TierL1: {Status: "failed"},
		},
	}
	evalUnauth := workflow.EvaluatePreflight(unauth, now)
	if evalUnauth.Passed {
		t.Errorf("unauth adapter should fail preflight, got %+v", evalUnauth)
	}

	// Cooldown active
	cooldown := domain.Adapter{
		ID:        "cooldown-agent",
		Installed: true,
		Probes: map[domain.Tier]domain.ProbeResult{
			domain.TierL0: {Status: "passed"},
			domain.TierL1: {Status: "passed"},
		},
		CooldownUntil: &futureCooldown,
	}
	evalCool := workflow.EvaluatePreflight(cooldown, now)
	if evalCool.Passed {
		t.Errorf("cooldown adapter should fail preflight, got %+v", evalCool)
	}
}

func TestDependenciesAndCanExecuteStage(t *testing.T) {
	stage := workflow.Stage{
		ID:    "deploy",
		Needs: []string{"build", "test"},
	}

	// Nil / missing map
	pass0, unmet0 := workflow.EvaluateDependencies(stage, nil)
	if pass0 || len(unmet0) != 2 {
		t.Errorf("expected 2 unmet with nil map, got pass=%v, unmet=%v", pass0, unmet0)
	}

	// Partial
	pass1, unmet1 := workflow.EvaluateDependencies(stage, map[string]bool{"build": true})
	if pass1 || len(unmet1) != 1 || unmet1[0] != "test" {
		t.Errorf("expected unmet test, got pass=%v, unmet=%v", pass1, unmet1)
	}

	// Complete
	pass2, unmet2 := workflow.EvaluateDependencies(stage, map[string]bool{"build": true, "test": true})
	if !pass2 || len(unmet2) != 0 {
		t.Errorf("expected all dependencies met, got pass=%v, unmet=%v", pass2, unmet2)
	}

	// CanExecuteStage
	now := time.UnixMilli(1730000000000)
	healthyAdapter := domain.Adapter{
		ID:        "agent",
		Installed: true,
		Probes: map[domain.Tier]domain.ProbeResult{
			domain.TierL0: {Status: "passed"},
			domain.TierL1: {Status: "passed"},
		},
	}

	canExec, blockers := workflow.CanExecuteStage(stage, map[string]bool{"build": true, "test": true}, healthyAdapter, now)
	if !canExec || len(blockers) != 0 {
		t.Errorf("expected CanExecuteStage true, got %v, blockers: %v", canExec, blockers)
	}

	cannotExec, blockers2 := workflow.CanExecuteStage(stage, map[string]bool{"build": true}, healthyAdapter, now)
	if cannotExec || len(blockers2) == 0 {
		t.Errorf("expected CanExecuteStage false due to unmet dependency")
	}

	unauthAdapter := domain.Adapter{ID: "bad"}
	cannotExecPreflight, blockers3 := workflow.CanExecuteStage(stage, map[string]bool{"build": true, "test": true}, unauthAdapter, now)
	if cannotExecPreflight || len(blockers3) == 0 {
		t.Errorf("expected CanExecuteStage false due to preflight failure")
	}
}
