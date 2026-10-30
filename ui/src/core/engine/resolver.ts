import { AgentAdapter, AgentRole, RoleResolution, SoundcheckConfig } from '../../types/soundcheck';

export const DEFAULT_CONFIG: SoundcheckConfig = {
  version: 1,
  roles: {
    primary: ['claude:claude-3-7-sonnet', 'codex:gpt-4o', 'opencode:anthropic/claude-sonnet', 'agy:gemini-2.5-pro'],
    subagent: ['claude:claude-3-7-sonnet', 'opencode:*', 'agy:gemini-2.5-flash'],
    auxiliary: ['claude:claude-3-5-haiku', 'agy:gemini-2.5-flash', 'codex:gpt-4o-mini'],
    reviewer: ['codex:*', 'claude:*']
  },
  policy: {
    require_distinct_vendor: ['primary', 'reviewer'],
    max_cost_usd_per_run: 5.0,
    probe: {
      deep: false,
      ttl_minutes: 30
    }
  }
};

export interface ResolverExplanationStep {
  candidate: string;
  agentId: string;
  modelId: string;
  status: 'selected' | 'skipped_cooldown' | 'skipped_uninstalled' | 'skipped_unauth' | 'skipped_vendor_conflict';
  details: string;
}

export interface DetailedResolution extends RoleResolution {
  steps: ResolverExplanationStep[];
}

export function resolveRoles(
  config: SoundcheckConfig,
  adapters: AgentAdapter[]
): Record<AgentRole, DetailedResolution> {
  const adapterMap = new Map<string, AgentAdapter>();
  adapters.forEach((a) => adapterMap.set(a.id, a));

  const results: Partial<Record<AgentRole, DetailedResolution>> = {};
  const assignedVendors: Partial<Record<AgentRole, string>> = {};

  const rolesOrder: AgentRole[] = ['primary', 'subagent', 'auxiliary', 'reviewer'];

  for (const role of rolesOrder) {
    const candidateList = config.roles[role] || [];
    const steps: ResolverExplanationStep[] = [];
    let selected: { adapter: AgentAdapter; modelId: string; reasoning: string } | null = null;
    const fallbackChain: string[] = [];

    for (const cand of candidateList) {
      const [agentId, modelSpec] = cand.split(':');
      const adapter = adapterMap.get(agentId);

      if (!adapter) {
        steps.push({
          candidate: cand,
          agentId,
          modelId: modelSpec || '*',
          status: 'skipped_uninstalled',
          details: `Adapter "${agentId}" is not recognized or installed.`
        });
        continue;
      }

      // Check cooldown
      const isInCooldown = Boolean(adapter.cooldownUntil && adapter.cooldownUntil > Date.now());

      if (isInCooldown) {
        const remainingMinutes = adapter.cooldownUntil ? Math.max(1, Math.ceil((adapter.cooldownUntil - Date.now()) / 60000)) : 15;
        steps.push({
          candidate: cand,
          agentId,
          modelId: modelSpec || '*',
          status: 'skipped_cooldown',
          details: `In cooldown (${remainingMinutes}m remaining) due to rate limit error.`
        });
        fallbackChain.push(`${cand} (cooldown)`);
        continue;
      }

      // Check L0 Presence
      if (!adapter.installed || adapter.probes.L0.status === 'failed') {
        steps.push({
          candidate: cand,
          agentId,
          modelId: modelSpec || '*',
          status: 'skipped_uninstalled',
          details: `L0 Presence check failed: binary not found in PATH.`
        });
        fallbackChain.push(`${cand} (not installed)`);
        continue;
      }

      // Check L1 Auth
      if (adapter.probes.L1.status === 'failed' || adapter.probes.L1.errorClass === 'unauthenticated') {
        steps.push({
          candidate: cand,
          agentId,
          modelId: modelSpec || '*',
          status: 'skipped_unauth',
          details: `L1 Auth check failed: credentials invalid or missing.`
        });
        fallbackChain.push(`${cand} (unauthenticated)`);
        continue;
      }

      // Check distinct vendor policy (e.g. reviewer != primary)
      const [roleA, roleB] = config.policy.require_distinct_vendor;
      if (role === roleB && assignedVendors[roleA] && assignedVendors[roleA] === adapter.vendor) {
        steps.push({
          candidate: cand,
          agentId,
          modelId: modelSpec || '*',
          status: 'skipped_vendor_conflict',
          details: `Policy require_distinct_vendor violated: primary is already from ${adapter.vendor}.`
        });
        fallbackChain.push(`${cand} (vendor conflict: ${adapter.vendor})`);
        continue;
      }

      // Resolve concrete model
      let targetModelId = modelSpec;
      if (!targetModelId || targetModelId === '*') {
        // Pick first verified model or default
        targetModelId = adapter.models[0]?.id || 'default';
      }

      selected = {
        adapter,
        modelId: targetModelId,
        reasoning: `Selected candidate #${candidateList.indexOf(cand) + 1} (${cand}). Passed L0 presence, L1 auth, and vendor diversity constraints.`
      };

      steps.push({
        candidate: cand,
        agentId,
        modelId: targetModelId,
        status: 'selected',
        details: selected.reasoning
      });

      break;
    }

    if (selected) {
      const modelObj = selected.adapter.models.find((m) => m.id === selected.modelId) || selected.adapter.models[0];
      assignedVendors[role] = selected.adapter.vendor;

      results[role] = {
        role,
        selectedAgentId: selected.adapter.id,
        selectedModel: modelObj?.displayName || selected.modelId,
        fallbackChain,
        reasoning: selected.reasoning,
        vendor: selected.adapter.vendor,
        estimatedCostPerM: (modelObj?.inputCostPerM || 1) + (modelObj?.outputCostPerM || 3),
        healthy: true,
        steps
      };
    } else {
      // Degraded fallback
      results[role] = {
        role,
        selectedAgentId: 'none',
        selectedModel: 'None available',
        fallbackChain,
        reasoning: 'All configured candidates exhausted or in cooldown. Manual intervention required.',
        vendor: 'None',
        estimatedCostPerM: 0,
        healthy: false,
        steps
      };
    }
  }

  return results as Record<AgentRole, DetailedResolution>;
}
