// Package workflow defines pure declarative workflow stage DAGs, default patterns, and gate evaluation.
package workflow

import (
	"github.com/soundcheck/soundcheck/internal/core/domain"
)

// Stage aliases domain.WorkflowStage for stage DAG nodes.
type Stage = domain.WorkflowStage

// Pattern aliases domain.WorkflowPattern for pipeline DAG definitions.
type Pattern = domain.WorkflowPattern

// Gate aliases domain.Gate for deterministic stage gates.
type Gate = domain.Gate

// GateInput aliases domain.GateInput for runtime stage gate evaluation inputs.
type GateInput = domain.GateInput

// GateEvaluation aliases domain.GateEvaluation for stage gate evaluation outcomes.
type GateEvaluation = domain.GateEvaluation

// PreflightEvaluation aliases domain.PreflightEvaluation for agent readiness checks.
type PreflightEvaluation = domain.PreflightEvaluation
