export type ErrorClass =
  | 'ok'
  | 'not_installed'
  | 'unauthenticated'
  | 'rate_limited'
  | 'quota_exhausted'
  | 'model_unavailable'
  | 'permission_denied'
  | 'timeout'
  | 'corrupted_state';

export type ProbeLevel = 'L0' | 'L1' | 'L2' | 'L3';

export interface KnownQuirk {
  id: string;
  title: string;
  description: string;
  referenceUrl: string;
  discoveredDate: string;
  severity: 'critical' | 'high' | 'medium';
  impact: string;
  resolution: string;
}

export interface ModelInfo {
  id: string;
  displayName: string;
  vendor: string;
  tier: 'frontier' | 'mid' | 'fast' | 'reasoning';
  contextWindow: number;
  inputCostPerM: number;
  outputCostPerM: number;
  verified: boolean;
}

export interface ProbeResult {
  level: ProbeLevel;
  timestamp: number;
  cachedUntil: number;
  latencyMs: number;
  status: 'passed' | 'failed' | 'warning' | 'pending';
  errorClass: ErrorClass;
  message: string;
  rawDetails?: {
    exitCode?: number;
    stdout?: string;
    stderr?: string;
    parsedJson?: any;
    quirkTriggered?: string;
  };
}

export interface AgentAdapter {
  id: string;
  name: string;
  vendor: string;
  binary: string;
  category: 'platform' | 'opensource' | 'protocol';
  versionSupported: string;
  installed: boolean;
  versionDetected?: string;
  authMethodSupported: string[];
  headlessCommand: string;
  permissionModes: string[];
  quirks: KnownQuirk[];
  models: ModelInfo[];
  probes: {
    L0: ProbeResult; // Presence & semver
    L1: ProbeResult; // Auth check (non-interactive)
    L2: ProbeResult; // Functional smoke test
    L3: ProbeResult; // Model probe
  };
  cooldownUntil?: number;
  allowedFails: number;
  consecutiveFails: number;
}

export type AgentRole = 'primary' | 'subagent' | 'auxiliary' | 'reviewer';

export interface RoleResolution {
  role: AgentRole;
  selectedAgentId: string;
  selectedModel: string;
  fallbackChain: string[];
  reasoning: string;
  vendor: string;
  estimatedCostPerM: number;
  healthy: boolean;
}

export interface SoundcheckConfig {
  version: 1;
  roles: {
    primary: string[];
    subagent: string[];
    auxiliary: string[];
    reviewer: string[];
  };
  policy: {
    require_distinct_vendor: [AgentRole, AgentRole];
    max_cost_usd_per_run: number;
    probe: {
      deep: boolean;
      ttl_minutes: number;
    };
  };
}

export interface WorkflowStage {
  id: string;
  name: string;
  role: AgentRole;
  needs?: string[];
  permissions: 'read-only' | 'workspace-write';
  worktree?: boolean;
  outputArtifact?: string;
  gate?: {
    tests?: string;
    diff_nonempty?: boolean;
    max_loops?: number;
  };
}

export interface WorkflowPattern {
  id: string;
  name: string;
  description: string;
  stages: WorkflowStage[];
  rawYaml: string;
}

export interface CapabilityManifest {
  schemaVersion: '2026.1';
  generatedAt: string;
  host: {
    platform: string;
    nodeVersion: string;
    gitWorktree: boolean;
  };
  summary: {
    totalAgents: number;
    functionalAgents: number;
    cooldownsActive: number;
    quirksProtected: number;
  };
  agents: Record<string, {
    installed: boolean;
    version?: string;
    authValid: boolean;
    functional: boolean;
    cooldownActive: boolean;
    errorClass: ErrorClass;
    models: string[];
    permissions: string[];
  }>;
  resolvedRoles: Record<AgentRole, {
    agent: string;
    model: string;
    vendor: string;
    reason: string;
  }>;
}
