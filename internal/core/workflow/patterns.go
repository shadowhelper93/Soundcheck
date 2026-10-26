package workflow

import (
	"github.com/soundcheck/soundcheck/internal/core/domain"
)

// Standard workflow pattern identifiers.
const (
	PatternIDPlanImplementReview = "plan-implement-review"
	PatternIDSpecTDDAudit        = "spec-tdd-audit"
)

const planImplementReviewYAML = `# Soundcheck Declarative Workflow Pattern
version: 1
pattern: plan-implement-review
description: "Golden tri-stage DAG with deterministic gates and vendor-isolated review"

stages:
  plan:
    role: primary
    permissions: read-only
    output: plan.md

  implement:
    role: subagent
    needs: [plan]
    permissions: workspace-write
    worktree: true
    output: git.patch

  review:
    role: reviewer
    needs: [implement]
    permissions: read-only
    gate:
      tests: "npm run lint"
      diff_nonempty: true
      max_loops: 2`

const specTDDAuditYAML = `# Soundcheck TDD Pattern
version: 1
pattern: spec-tdd-audit
description: "Test-driven autonomous pipeline with verification gates"

stages:
  spec:
    role: primary
    permissions: read-only
    output: spec.yaml

  tests:
    role: auxiliary
    needs: [spec]
    permissions: workspace-write
    output: tests.patch

  implement:
    role: subagent
    needs: [tests]
    permissions: workspace-write
    worktree: true

  audit:
    role: reviewer
    needs: [implement]
    permissions: read-only
    gate:
      tests: "npm test"
      diff_nonempty: true`

// PlanImplementReviewPattern returns the standard tri-stage workflow pattern.
func PlanImplementReviewPattern() Pattern {
	return Pattern{
		ID:          PatternIDPlanImplementReview,
		Name:        "Plan → Implement → Review",
		Description: "The golden tri-stage workflow. Primary agent crafts plan in read-only mode, subagent writes implementation in isolated worktree, independent reviewer inspects diff with test gates.",
		Stages: []Stage{
			{
				ID:             "plan",
				Name:           "Architecture Planning",
				Role:           domain.RolePrimary,
				Permissions:    "read-only",
				Worktree:       false,
				OutputArtifact: "plan.md",
			},
			{
				ID:             "implement",
				Name:           "Isolated Code Synthesis",
				Role:           domain.RoleSubagent,
				Needs:          []string{"plan"},
				Permissions:    "workspace-write",
				Worktree:       true,
				OutputArtifact: "git.patch",
			},
			{
				ID:          "review",
				Name:        "Cross-Vendor Adversarial Review",
				Role:        domain.RoleReviewer,
				Needs:       []string{"implement"},
				Permissions: "read-only",
				Worktree:    false,
				Gate: &Gate{
					Tests:        "npm run lint",
					DiffNonEmpty: true,
					MaxLoops:     2,
				},
			},
		},
		RawYAML: planImplementReviewYAML,
	}
}

// SpecTDDAuditPattern returns the test-driven development workflow pattern.
func SpecTDDAuditPattern() Pattern {
	return Pattern{
		ID:          PatternIDSpecTDDAudit,
		Name:        "Spec → TDD Test Synthesis → Code → Audit",
		Description: "Enforces test-driven development: generates failing test fixtures before writing implementation, then gates on zero test regressions.",
		Stages: []Stage{
			{
				ID:             "spec",
				Name:           "Formal Specification",
				Role:           domain.RolePrimary,
				Permissions:    "read-only",
				OutputArtifact: "spec.yaml",
			},
			{
				ID:             "tests",
				Name:           "Test Harness Generation",
				Role:           domain.RoleAuxiliary,
				Needs:          []string{"spec"},
				Permissions:    "workspace-write",
				OutputArtifact: "tests.patch",
			},
			{
				ID:          "implement",
				Name:        "Implementation Passing Tests",
				Role:        domain.RoleSubagent,
				Needs:       []string{"tests"},
				Permissions: "workspace-write",
				Worktree:    true,
			},
			{
				ID:          "audit",
				Name:        "Security & Token Audit",
				Role:        domain.RoleReviewer,
				Needs:       []string{"implement"},
				Permissions: "read-only",
				Gate: &Gate{
					Tests:        "npm test",
					DiffNonEmpty: true,
				},
			},
		},
		RawYAML: specTDDAuditYAML,
	}
}

// DefaultPatterns returns all built-in standard workflow patterns.
func DefaultPatterns() []Pattern {
	return []Pattern{
		PlanImplementReviewPattern(),
		SpecTDDAuditPattern(),
	}
}

// GetPattern retrieves a default pattern by identifier.
func GetPattern(id string) (Pattern, bool) {
	for _, p := range DefaultPatterns() {
		if p.ID == id {
			return p, true
		}
	}
	return Pattern{}, false
}
