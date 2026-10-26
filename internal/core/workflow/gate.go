package workflow

import (
	"fmt"
	"strings"
	"time"

	"github.com/soundcheck/soundcheck/internal/core/domain"
)

// EvaluateGate evaluates verification conditions against stage execution results.
func EvaluateGate(gate *Gate, input GateInput) GateEvaluation {
	if gate == nil {
		return GateEvaluation{
			Passed:     true,
			DiffPassed: true,
			TestPassed: true,
			LoopPassed: true,
			Message:    "No verification gate configured",
		}
	}

	diffPassed, diffErr := evalDiffGate(gate, input)
	testPassed, testErr := evalTestGate(gate, input)
	loopPassed, loopErr := evalLoopGate(gate, input)

	failures := collectErrors(diffErr, testErr, loopErr)
	passed := diffPassed && testPassed && loopPassed

	msg := "Stage verification gate passed."
	if !passed {
		msg = "Verification gate failed: " + strings.Join(failures, "; ")
	}

	return GateEvaluation{
		Passed:     passed,
		DiffPassed: diffPassed,
		TestPassed: testPassed,
		LoopPassed: loopPassed,
		Failures:   failures,
		Message:    msg,
	}
}

func evalDiffGate(gate *Gate, input GateInput) (bool, string) {
	if !gate.DiffNonEmpty {
		return true, ""
	}
	if input.HasDiff || input.DiffSize > 0 {
		return true, ""
	}
	return false, "empty diff: expected modified changes"
}

func evalTestGate(gate *Gate, input GateInput) (bool, string) {
	if gate.Tests == "" {
		return true, ""
	}
	if input.TestPassed && input.TestExitCode == 0 {
		return true, ""
	}
	if input.TestExitCode != 0 {
		return false, fmt.Sprintf("tests failed with exit code %d", input.TestExitCode)
	}
	return false, "tests failed"
}

func evalLoopGate(gate *Gate, input GateInput) (bool, string) {
	if gate.MaxLoops <= 0 {
		return true, ""
	}
	if input.CurrentLoop <= gate.MaxLoops {
		return true, ""
	}
	return false, fmt.Sprintf("max loops exceeded: current %d > max %d", input.CurrentLoop, gate.MaxLoops)
}

func collectErrors(errs ...string) []string {
	var res []string
	for _, e := range errs {
		if e != "" {
			res = append(res, e)
		}
	}
	return res
}

// EvaluatePreflight checks if an agent satisfies L0 presence, L1 auth, and cooldown requirements.
func EvaluatePreflight(adapter domain.Adapter, now time.Time) PreflightEvaluation {
	var failures []string
	l0, hasL0 := adapter.Probes[domain.TierL0]
	if !adapter.Installed || !hasL0 || l0.Status != "passed" {
		failures = append(failures, "L0 presence failed: binary not installed or missing in PATH")
	}

	l1, hasL1 := adapter.Probes[domain.TierL1]
	if !hasL1 || l1.Status != "passed" {
		failures = append(failures, "L1 auth failed: agent credentials invalid or unauthenticated")
	}

	if adapter.CooldownUntil != nil && *adapter.CooldownUntil > now.UnixMilli() {
		failures = append(failures, "agent rate-limit cooldown is currently active")
	}

	passed := len(failures) == 0
	msg := "Preflight check passed: agent ready for execution."
	if !passed {
		msg = "Preflight check failed: " + strings.Join(failures, "; ")
	}

	return PreflightEvaluation{
		Passed:   passed,
		Failures: failures,
		Message:  msg,
	}
}

// EvaluateDependencies checks whether all upstream dependencies have completed successfully.
func EvaluateDependencies(stage Stage, completedStages map[string]bool) (bool, []string) {
	var unmet []string
	for _, dep := range stage.Needs {
		if completedStages == nil || !completedStages[dep] {
			unmet = append(unmet, dep)
		}
	}
	return len(unmet) == 0, unmet
}

// CanExecuteStage evaluates whether all stage dependencies and agent preflight checks are satisfied.
func CanExecuteStage(
	stage Stage,
	completed map[string]bool,
	adapter domain.Adapter,
	now time.Time,
) (bool, []string) {
	var blockers []string
	depsPassed, unmet := EvaluateDependencies(stage, completed)
	if !depsPassed {
		blockers = append(blockers, fmt.Sprintf("unmet stage dependencies: %s", strings.Join(unmet, ", ")))
	}

	preflight := EvaluatePreflight(adapter, now)
	if !preflight.Passed {
		blockers = append(blockers, preflight.Failures...)
	}

	return len(blockers) == 0, blockers
}
