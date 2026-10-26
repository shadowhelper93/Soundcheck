import { WorkflowPattern } from '../../types/soundcheck';

export const DEFAULT_PATTERNS: WorkflowPattern[] = [
  {
    id: 'plan-implement-review',
    name: 'Plan → Implement → Review',
    description: 'The golden tri-stage workflow. Primary agent crafts plan in read-only mode, subagent writes implementation in isolated worktree, independent reviewer inspects diff with test gates.',
    stages: [
      {
        id: 'plan',
        name: 'Architecture Planning',
        role: 'primary',
        permissions: 'read-only',
        worktree: false,
        outputArtifact: 'plan.md'
      },
      {
        id: 'implement',
        name: 'Isolated Code Synthesis',
        role: 'subagent',
        needs: ['plan'],
        permissions: 'workspace-write',
        worktree: true,
        outputArtifact: 'git.patch'
      },
      {
        id: 'review',
        name: 'Cross-Vendor Adversarial Review',
        role: 'reviewer',
        needs: ['implement'],
        permissions: 'read-only',
        worktree: false,
        gate: {
          tests: 'npm run lint',
          diff_nonempty: true,
          max_loops: 2
        }
      }
    ],
    rawYaml: `# Soundcheck Declarative Workflow Pattern
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
  },
  {
    id: 'spec-tdd-audit',
    name: 'Spec → TDD Test Synthesis → Code → Audit',
    description: 'Enforces test-driven development: generates failing test fixtures before writing implementation, then gates on zero test regressions.',
    stages: [
      {
        id: 'spec',
        name: 'Formal Specification',
        role: 'primary',
        permissions: 'read-only',
        outputArtifact: 'spec.yaml'
      },
      {
        id: 'tests',
        name: 'Test Harness Generation',
        role: 'auxiliary',
        needs: ['spec'],
        permissions: 'workspace-write',
        outputArtifact: 'tests.patch'
      },
      {
        id: 'implement',
        name: 'Implementation Passing Tests',
        role: 'subagent',
        needs: ['tests'],
        permissions: 'workspace-write',
        worktree: true
      },
      {
        id: 'audit',
        name: 'Security & Token Audit',
        role: 'reviewer',
        needs: ['implement'],
        permissions: 'read-only',
        gate: {
          tests: 'npm test',
          diff_nonempty: true
        }
      }
    ],
    rawYaml: `# Soundcheck TDD Pattern
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
  }
];
