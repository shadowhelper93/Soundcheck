export interface ProbeResult {
  level: string;
  status: 'passed' | 'failed' | 'warning' | 'pending';
  errorClass: string;
  message: string;
  latencyMs: number;
}

export interface AdapterModel {
  id: string;
  displayName: string;
  vendor: string;
  tier: string;
}

export interface Adapter {
  id: string;
  name: string;
  vendor: string;
  binary: string;
  installed: boolean;
  versionDetected?: string;
  probes: Record<string, ProbeResult>;
  models?: AdapterModel[];
}

export interface DetailedResolution {
  role: string;
  selectedAgentId: string;
  selectedModel: string;
  vendor: string;
  healthy: boolean;
  reason: string;
}

export interface ManifestSummary {
  totalAgents: number;
  functionalAgents: number;
  cooldownsActive: number;
  quirksProtected: number;
}

export interface DoctorReport {
  timestamp: string;
  adapters: Adapter[];
  resolutions: Record<string, DetailedResolution>;
  summary: ManifestSummary;
  host: {
    platform: string;
    nodeVersion: string;
    gitWorktree: boolean;
  };
}

export interface RunPromptAttempt {
  agentId: string;
  success: boolean;
  errorMsg?: string;
}

export interface RunPromptResult {
  prompt: string;
  agentUsed: string;
  output: string;
  success: boolean;
  failoverOccurred: boolean;
  attempts: RunPromptAttempt[];
  durationMs: number;
}
