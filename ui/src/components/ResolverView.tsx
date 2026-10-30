import React, { useState } from 'react';
import { AgentAdapter, AgentRole, SoundcheckConfig } from '../types/soundcheck';
import { DetailedResolution } from '../core/engine/resolver';
import { SoundcheckMark } from './SoundcheckLogo';
import { RoleConfigEditor } from './RoleConfigEditor';
import {
  Cpu,
  Flame,
  CheckCircle2,
  AlertCircle,
  HelpCircle,
  DollarSign,
  Split,
  Settings2
} from 'lucide-react';

interface ResolverViewProps {
  config: SoundcheckConfig;
  onUpdateConfig: (newConfig: SoundcheckConfig) => void;
  resolvedRoles: Record<AgentRole, DetailedResolution>;
  adapters: AgentAdapter[];
  accentColor: string;
}

export const ResolverView: React.FC<ResolverViewProps> = ({
  config,
  onUpdateConfig,
  resolvedRoles,
  accentColor
}) => {
  const [selectedRoleForDetail, setSelectedRoleForDetail] = useState<AgentRole>('primary');
  const [isEditingPolicy, setIsEditingPolicy] = useState(false);
  const [liveResolutions, setLiveResolutions] = useState<any[] | null>(null);
  const [isQueryingApi, setIsQueryingApi] = useState(false);

  const queryLiveResolve = async (roleToQuery: string) => {
    setIsQueryingApi(true);
    setLiveResolutions(null);
    try {
      const res = await fetch(`/api/v1/resolve?role=${roleToQuery}`);
      if (res.ok) {
        const data = await res.json();
        const list = Array.isArray(data)
          ? data
          : (data && typeof data === 'object' && ('role' in data || 'selectedAgentId' in data))
            ? [data]
            : Object.values(data || {});
        setLiveResolutions(list);
      }
    } catch {
      // ignore
    } finally {
      setIsQueryingApi(false);
    }
  };

  React.useEffect(() => {
    queryLiveResolve(selectedRoleForDetail);
  }, [selectedRoleForDetail]);

  const rolesList: AgentRole[] = ['primary', 'subagent', 'auxiliary', 'reviewer'];

  const roleDescriptions: Record<AgentRole, string> = {
    primary: 'Frontier architect & planner. Handles codebase analysis and plan.md synthesis.',
    subagent: 'Worker executor. Executes delegated implementation stages inside git worktrees.',
    auxiliary: 'Fast helper for tests, docstrings, triage, and lint corrections.',
    reviewer: 'Adversarial auditor. Performs cross-vendor review of diffs and test passes.'
  };

  const currentDetail = resolvedRoles[selectedRoleForDetail];

  return (
    <div className="space-y-6">
      {/* Header and Interactive Simulation Toolbar */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4 p-5 rounded-xl bg-[#121721] border border-[#1E2638]">
        <div>
          <div className="flex items-center gap-2">
            <Cpu className="w-5 h-5" style={{ color: accentColor }} />
            <h2 className="text-base font-bold text-[#E6EAF2] font-mono tracking-tight">
              DETERMINISTIC ROLE RESOLVER & FALLBACK ENGINE
            </h2>
          </div>
          <p className="text-xs text-[#8B95A7] mt-1">
            Pure constraint-satisfaction solver: zero token cost, 100% reproducible, explainable decision trail.
          </p>
        </div>

        {/* Policy Editor Action */}
        <div className="flex items-center gap-3">
          <button
            onClick={() => setIsEditingPolicy(!isEditingPolicy)}
            className="flex items-center gap-1.5 px-3 py-2 text-xs font-mono text-[#8B95A7] hover:text-[#E6EAF2] bg-[#0A0D12] border border-[#1E2638] rounded-lg transition-colors cursor-pointer"
          >
            <Settings2 className="w-3.5 h-3.5 text-[#8B95A7]" />
            <span>{isEditingPolicy ? 'Close Policy Editor' : 'Edit Policies'}</span>
          </button>
        </div>
      </div>

      {/* Policy Editor Drawer */}
      {isEditingPolicy && (
        <div className="p-5 rounded-xl bg-[#0A0D12] border border-[#1E2638] space-y-4 animate-in slide-in-from-top-2 duration-150">
          <div className="flex items-center justify-between">
            <span className="text-xs font-bold font-mono uppercase tracking-wider" style={{ color: accentColor }}>
              Soundcheck Canonical Policy Rules (agent-doctor.yaml)
            </span>
            <span className="text-[11px] font-mono text-[#8B95A7]">
              Validated against JSON Schema 2026.1
            </span>
          </div>

          <RoleConfigEditor
            accentColor={accentColor}
            onConfigSaved={(saved) => {
              onUpdateConfig(saved);
              queryLiveResolve(selectedRoleForDetail);
            }}
          />
        </div>
      )}

      {/* 4 Role Resolution Cards Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        {rolesList.map((role) => {
          const res = resolvedRoles[role];
          const isSelected = selectedRoleForDetail === role;
          const candidateList = config.roles[role] || [];

          return (
            <div
              key={role}
              onClick={() => setSelectedRoleForDetail(role)}
              className={`p-4 rounded-xl border cursor-pointer transition-all ${
                isSelected
                  ? 'bg-[#121721] border-[#E6EAF2] shadow-lg'
                  : 'bg-[#121721] border-[#1E2638] hover:border-[#8B95A7]'
              }`}
            >
              {/* Role Header */}
              <div className="flex items-center justify-between pb-2 border-b border-[#1E2638]">
                <span className="text-xs font-bold font-mono uppercase tracking-wider" style={{ color: accentColor }}>
                  {role}
                </span>
                {res?.healthy ? (
                  <span className="inline-flex items-center gap-1 text-[11px] font-mono text-[#10B981]">
                    <SoundcheckMark size={14} status="ok" accentColor={accentColor} />
                    RESOLVED
                  </span>
                ) : (
                  <span className="inline-flex items-center gap-1 text-[11px] font-mono text-[#F87171]">
                    <AlertCircle className="w-3 h-3 text-[#F87171]" />
                    EXHAUSTED
                  </span>
                )}
              </div>

              {/* Resolved Agent & Model */}
              <div className="mt-3">
                <div className="text-sm font-bold text-[#E6EAF2] font-mono truncate">
                  {res?.selectedModel || 'None available'}
                </div>
                <div className="text-xs text-[#8B95A7] font-mono mt-0.5 flex items-center gap-1.5">
                  <span>CLI: {res?.selectedAgentId}</span>
                  <span>·</span>
                  <span>{res?.vendor}</span>
                </div>
              </div>

              {/* Fallback chain preview */}
              <div className="mt-3 pt-3 border-t border-[#1E2638]">
                <div className="text-[10px] font-mono uppercase text-[#8B95A7] mb-1.5">
                  Fallback Chain ({candidateList.length} defined)
                </div>
                <div className="space-y-1">
                  {candidateList.slice(0, 3).map((cand, idx) => {
                    const [candAgent] = cand.split(':');
                    const isCandidateActive = res?.selectedAgentId === candAgent;
                    return (
                      <div
                        key={idx}
                        className={`text-[11px] font-mono px-2 py-0.5 rounded flex items-center justify-between ${
                          isCandidateActive
                            ? 'bg-[#0A0D12] text-[#E6EAF2] border border-[#1E2638] font-medium'
                            : 'bg-[#0A0D12]/40 text-[#8B95A7]'
                        }`}
                      >
                        <span className="truncate">{idx + 1}. {cand}</span>
                        {isCandidateActive && (
                          <span className="text-[9px]" style={{ color: accentColor }}>ACTIVE</span>
                        )}
                      </div>
                    );
                  })}
                </div>
              </div>

              <div className="mt-3 text-[11px] text-[#8B95A7] line-clamp-2">
                {roleDescriptions[role]}
              </div>
            </div>
          );
        })}
      </div>

      {/* Selected Role Decision Inspector */}
      {currentDetail && (
        <div className="p-6 rounded-xl bg-[#121721] border border-[#1E2638] space-y-4">
          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-2 pb-4 border-b border-[#1E2638]">
            <div>
              <div className="flex items-center gap-2">
                <h3 className="text-sm font-bold font-mono text-[#E6EAF2] uppercase">
                  EXPLAINABLE DECISION TRACE: {selectedRoleForDetail.toUpperCase()} ROLE
                </h3>
                <span className="text-xs font-mono" style={{ color: accentColor }}>
                  [{currentDetail.selectedAgentId}:{currentDetail.selectedModel}]
                </span>
              </div>
              <p className="text-xs text-[#8B95A7] mt-0.5">
                Evaluated deterministically in priority order according to Soundcheck rule constraints.
              </p>
            </div>

            <div className="text-xs font-mono text-[#8B95A7] tabular-nums">
              Est. Cost: ${currentDetail.estimatedCostPerM.toFixed(2)}/1M tokens
            </div>
          </div>

          {/* Decision trail steps */}
          <div className="space-y-2.5">
            {currentDetail.steps.map((step, idx) => {
              const isSelected = step.status === 'selected';
              return (
                <div
                  key={idx}
                  className={`p-3 rounded-lg border font-mono text-xs flex items-start gap-3 ${
                    isSelected
                      ? 'bg-[#0A0D12] border-[#10B981]/50 text-[#E6EAF2]'
                      : 'bg-[#0A0D12] border-[#1E2638] text-[#8B95A7]'
                  }`}
                >
                  <div className="mt-0.5">
                    {isSelected ? (
                      <SoundcheckMark size={16} status="ok" accentColor={accentColor} />
                    ) : (
                      <span className="w-4 h-4 rounded-full bg-[#1E2638] text-[10px] text-[#8B95A7] flex items-center justify-center font-bold">
                        {idx + 1}
                      </span>
                    )}
                  </div>

                  <div className="flex-1 space-y-0.5">
                    <div className="flex items-center justify-between">
                      <span className="font-semibold text-[#E6EAF2]">
                        Candidate: {step.candidate}
                      </span>
                      <span className="text-[10px] uppercase font-bold px-1.5 py-0.5 rounded bg-[#121721] text-[#8B95A7] border border-[#1E2638]">
                        {step.status.replace('_', ' ')}
                      </span>
                    </div>
                    <div className="text-[11px] leading-relaxed text-[#8B95A7]">
                      {step.details}
                    </div>
                  </div>
                </div>
              );
            })}
          </div>

          <div className="p-3 rounded-lg bg-[#0A0D12] border border-[#1E2638] text-xs font-mono text-[#8B95A7] flex items-center justify-between">
            <div className="flex items-center gap-2">
              <HelpCircle className="w-4 h-4 text-[#8B95A7] shrink-0" />
              <span>
                If this agent hits an API rate limit or silent exit 0 during execution, Soundcheck immediately rolls back and promotes candidate #2 without restarting the entire pipeline.
              </span>
            </div>
            <button
              onClick={() => queryLiveResolve(selectedRoleForDetail)}
              className="px-2.5 py-1 text-[11px] bg-[#121721] hover:bg-[#1E2638] border border-[#1E2638] text-[#06B6D4] rounded cursor-pointer shrink-0"
            >
              {isQueryingApi ? 'Querying...' : 'Query /api/v1/resolve'}
            </button>
          </div>

          {liveResolutions && liveResolutions.length > 0 && (
            <div className="p-4 rounded-lg bg-[#0A0D12] border border-[#06B6D4]/30 space-y-2">
              <div className="flex items-center justify-between text-xs font-mono">
                <span className="text-[#06B6D4] font-bold">Go API Live Response (/api/v1/resolve)</span>
                <span className="text-[10px] text-[#8B95A7]">Resolved by Go engine</span>
              </div>
              <div className="space-y-1">
                {liveResolutions.map((res: any, idx: number) => (
                  <div key={idx} className="text-xs font-mono flex items-center justify-between text-[#E6EAF2]">
                    <span>
                      {res.role || selectedRoleForDetail}: <span style={{ color: accentColor }}>{res.selectedAgentId || res.adapterId || res.adapter}</span> ({res.selectedModel || res.model || 'default'})
                    </span>
                    <span className="text-[11px] text-[#8B95A7]">{res.reasoning || res.reason || res.vendor}</span>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
};
