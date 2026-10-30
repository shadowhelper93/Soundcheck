import React, { useState } from 'react';
import { AgentAdapter, ProbeLevel, KnownQuirk, ProbeResult } from '../types/soundcheck';
import { SoundcheckMark, SoundcheckTierMeter } from './SoundcheckLogo';
import {
  ShieldAlert,
  Terminal,
  RefreshCw,
  ExternalLink,
  Flame,
  Info,
  Layers,
  Server,
  Zap,
  Code2,
  ChevronRight
} from 'lucide-react';

interface DoctorViewProps {
  adapters: AgentAdapter[];
  hostData: any;
  onProbeAdapter: (adapterId: string, level: ProbeLevel) => void;
  onClearCooldown: (adapterId: string) => void;
  isScanning: boolean;
  accentColor: string;
}

export const DoctorView: React.FC<DoctorViewProps> = ({
  adapters,
  hostData,
  onProbeAdapter,
  onClearCooldown,
  isScanning,
  accentColor
}) => {
  const [selectedQuirk, setSelectedQuirk] = useState<KnownQuirk | null>(null);
  const [selectedProbeLog, setSelectedProbeLog] = useState<{ adapter: AgentAdapter; level: ProbeLevel; probe: ProbeResult } | null>(null);

  const totalMonitored = adapters.length;
  const installedCount = adapters.filter((a) => a.installed).length;
  const activeCooldowns = adapters.filter(
    (a) => a.cooldownUntil && a.cooldownUntil > Date.now()
  ).length;
  const totalQuirksProtected = adapters.reduce((sum, a) => sum + a.quirks.length, 0);

  return (
    <div className="space-y-6">
      {/* Hero Banner with the Tier Meter & Tagline from Ladder.dc.html */}
      <div className="p-6 md:p-8 rounded-2xl bg-[#121721] border border-[#1E2638] glow-card flex flex-col lg:flex-row items-center justify-between gap-8 relative overflow-hidden">
        <div className="flex flex-col sm:flex-row items-center gap-6 z-10 text-center sm:text-left">
          {/* Level Meter SVG from Ladder.dc.html */}
          <div className="flex flex-col items-center gap-2 p-3 bg-[#0A0D12] rounded-xl border border-[#1E2638] shrink-0 shadow-inner">
            <SoundcheckTierMeter size={68} accentColor={accentColor} activeTier="all" />
            <div className="grid grid-cols-4 w-18 text-center font-mono text-[10px] text-[#8B95A7]">
              <span>L0</span>
              <span>L1</span>
              <span>L2</span>
              <span style={{ color: accentColor }} className="font-bold">L3</span>
            </div>
          </div>

          <div className="space-y-1.5">
            <div className="font-mono text-xs text-[#8B95A7] uppercase tracking-[0.08em] flex items-center justify-center sm:justify-start gap-2">
              <span>Soundcheck / Capability Layer</span>
              <span>·</span>
              <span style={{ color: accentColor }} className="font-bold">Tier Meter Active</span>
            </div>
            <h1 className="font-display text-2xl sm:text-3xl font-bold tracking-tight text-[#E6EAF2]">
              Presence → Auth → Smoke → Model.
            </h1>
            <p className="text-sm text-[#8B95A7] max-w-xl leading-relaxed">
              Every tier lit before a single token burns. A mixing-desk preflight prober where the check only lands once L3 passes.
            </p>
          </div>
        </div>

        {/* Live Status Glyphs from Marks.dc.html Section 04 */}
        <div className="flex items-center gap-4 bg-[#0A0D12] p-3.5 rounded-xl border border-[#1E2638] shrink-0 z-10 shadow-inner">
          <div className="flex flex-col items-center gap-1.5 px-3 border-r border-[#1E2638]">
            <SoundcheckMark size={28} status="ok" accentColor={accentColor} />
            <span className="font-mono text-[11px] text-[#10B981] font-semibold">ok</span>
          </div>
          <div className="flex flex-col items-center gap-1.5 px-3 border-r border-[#1E2638]">
            <SoundcheckMark size={28} status="cooldown" accentColor={accentColor} />
            <span className="font-mono text-[11px] text-[#F59E0B] font-semibold">cooldown</span>
          </div>
          <div className="flex flex-col items-center gap-1.5 px-3">
            <SoundcheckMark size={28} status="fake_0" accentColor={accentColor} />
            <span className="font-mono text-[11px] text-[#F87171] font-semibold">fake_0</span>
          </div>
        </div>
      </div>

      {/* Master Telemetry Strip */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-3">
        <div className="p-4 rounded-xl bg-[#121721] border border-[#1E2638] transition-all hover:border-[#2A364F]">
          <div className="text-xs text-[#8B95A7] font-mono flex items-center justify-between">
            <span>MONITORED CLIS</span>
            <Server className="w-3.5 h-3.5 text-[#8B95A7]" />
          </div>
          <div className="mt-2 text-2xl font-bold font-mono text-[#E6EAF2] tabular-nums">
            {totalMonitored} <span className="text-xs font-normal text-[#8B95A7]">/ {installedCount} in PATH</span>
          </div>
          <div className="text-[11px] text-[#8B95A7] mt-1">
            Adapters: Claude, Codex, OpenCode, agy, Copilot, ACP
          </div>
        </div>

        <div className="p-4 rounded-xl bg-[#121721] border border-[#1E2638] transition-all hover:border-[#2A364F]">
          <div className="text-xs font-mono flex items-center justify-between" style={{ color: accentColor }}>
            <span>PROBING STATUS</span>
            <Zap className="w-3.5 h-3.5" style={{ color: accentColor }} />
          </div>
          <div className="mt-2 text-2xl font-bold font-mono tabular-nums" style={{ color: accentColor }}>
            {adapters.filter((a) => a.probes.L1.status === 'passed').length}
            <span className="text-xs font-normal text-[#8B95A7]"> VERIFIED</span>
          </div>
          <div className="text-[11px] text-[#8B95A7] mt-1">
            Non-interactive L1 &amp; L2 turns verified
          </div>
        </div>

        <div className="p-4 rounded-xl bg-[#121721] border border-[#1E2638] transition-all hover:border-[#2A364F]">
          <div className="text-xs text-[#F59E0B] font-mono flex items-center justify-between">
            <span>COOLDOWNS ACTIVE</span>
            <Flame className="w-3.5 h-3.5 text-[#F59E0B]" />
          </div>
          <div className="mt-2 text-2xl font-bold font-mono text-[#F59E0B] tabular-nums">
            {activeCooldowns}
            <span className="text-xs font-normal text-[#8B95A7]"> THROTTLED</span>
          </div>
          <div className="text-[11px] text-[#8B95A7] mt-1">
            LiteLLM-style reactive failover buffer
          </div>
        </div>

        <div className="p-4 rounded-xl bg-[#121721] border border-[#1E2638] transition-all hover:border-[#2A364F]">
          <div className="text-xs text-[#06B6D4] font-mono flex items-center justify-between">
            <span>QUIRK DEFENSES</span>
            <Layers className="w-3.5 h-3.5 text-[#06B6D4]" />
          </div>
          <div className="mt-2 text-2xl font-bold font-mono text-[#06B6D4] tabular-nums">
            {totalQuirksProtected}
            <span className="text-xs font-normal text-[#8B95A7]"> ACTIVE</span>
          </div>
          <div className="text-[11px] text-[#8B95A7] mt-1">
            Guarding against fake 0 exits &amp; silent auth fails
          </div>
        </div>
      </div>

      {/* Mode Switch & Host Details */}
      <div className="flex flex-col md:flex-row items-start md:items-center justify-between gap-4 p-4 rounded-xl bg-[#121721]/80 border border-[#1E2638]">
        <div className="flex items-center gap-3">
          <div
            className="w-2.5 h-2.5 rounded-full animate-pulse shadow-sm"
            style={{ backgroundColor: accentColor }}
          />
          <div>
            <div className="text-xs font-mono font-semibold text-[#E6EAF2]">
              Host Environment: {hostData ? `${hostData.platform || 'unknown'} · Node ${hostData.nodeVersion || 'n/a'}` : 'Scanning host...'}
            </div>
            <div className="text-[11px] text-[#8B95A7]">
              Worktree Sandboxing: {hostData ? (hostData.gitWorktreeAvailable ? 'Enabled' : 'Disabled') : 'Detecting'} · TTL Cache: 24h (L0), 1h (L1), 30m (L2)
            </div>
          </div>
        </div>
      </div>

      {/* Cross-Platform Execution Console for Real Mac, Linux, and Windows */}
      <div className="p-4 rounded-xl bg-[#0A0D12] border border-[#1E2638] flex flex-col md:flex-row items-start md:items-center justify-between gap-4 font-mono text-xs">
        <div className="flex items-center gap-2.5">
          <Terminal className="w-4 h-4 text-[#10B981] shrink-0" />
          <div>
            <span className="text-[#E6EAF2] font-semibold">Real Host Terminal Execution: </span>
            <span className="text-[#8B95A7]">Run Soundcheck locally on macOS, Linux, or Windows (Command Prompt / PowerShell).</span>
          </div>
        </div>

        <div className="flex items-center gap-2 self-stretch md:self-auto">
          <div className="bg-[#121721] border border-[#1E2638] rounded-md px-3 py-1.5 text-[11px] text-[#E6EAF2] flex items-center gap-2">
            <span className="text-[#8B95A7]">$</span>
            <code className="text-[#10B981]">soundcheck doctor</code>
          </div>
          <button
            onClick={() => {
              navigator.clipboard.writeText('soundcheck doctor');
            }}
            className="px-2.5 py-1.5 rounded-md bg-[#121721] hover:bg-[#1E2638] border border-[#1E2638] text-[#8B95A7] hover:text-[#E6EAF2] text-[11px] transition-colors cursor-pointer"
            title="Copy command to clipboard"
          >
            Copy
          </button>
        </div>
      </div>

      {/* Anti-Slop Architectural Note */}
      <div className="p-4 rounded-xl bg-[#121721] border border-[#1E2638] flex items-start gap-3.5">
        <Info className="w-5 h-5 text-[#8B95A7] shrink-0 mt-0.5" />
        <div className="text-xs space-y-1 text-[#8B95A7] leading-relaxed">
          <span className="font-semibold text-[#E6EAF2] font-mono">
            GROUND TRUTH PREFLIGHT: WHY --version IS NOT A HEALTH CHECK
          </span>
          <p>
            Traditional agent runners check <code className="text-[#E6EAF2] font-mono">exec.LookPath</code> and assume an installed agent will succeed. In reality:
            Claude Code returns exit code 0 on rate-limit errors with <code className="text-[#E6EAF2] font-mono">subtype: "success"</code> (Issue #79500),
            OpenAI Codex doctor exits 0 despite auth failure, and Google Antigravity (<code className="text-[#E6EAF2] font-mono">agy</code>) lacks an auth-status command.
            Soundcheck classifies raw JSON streams to guarantee true headless readiness.
          </p>
        </div>
      </div>

      {/* Channel Grid */}
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <h2 className="text-sm font-semibold text-[#E6EAF2] font-mono tracking-tight flex items-center gap-2">
            <span>PER-CLI ADAPTER TELEMETRY &amp; TIER METER</span>
            <span className="text-[10px] font-normal text-[#8B95A7]">
              (Click any tier card to inspect raw stdout / cache details)
            </span>
          </h2>
          <span className="text-xs font-mono text-[#8B95A7]">
            {isScanning ? 'Probing channels...' : 'Cache validated'}
          </span>
        </div>

        <div className="grid grid-cols-1 gap-4">
          {adapters.map((adapter) => {
            const isCooldown = adapter.cooldownUntil && adapter.cooldownUntil > Date.now();
            const remainingCooldownMin = adapter.cooldownUntil
              ? Math.max(1, Math.ceil((adapter.cooldownUntil - Date.now()) / 60000))
              : 0;

            const currentStatus: 'ok' | 'cooldown' | 'fake_0' = isCooldown
              ? 'cooldown'
              : !adapter.installed
              ? 'fake_0'
              : 'ok';

            return (
              <div
                key={adapter.id}
                className={`p-5 rounded-xl border transition-all ${
                  isCooldown
                    ? 'bg-[#121721] border-[#F59E0B]/50'
                    : 'bg-[#121721] border-[#1E2638] hover:border-[#2A364F]'
                }`}
              >
                {/* Header row */}
                <div className="flex flex-col lg:flex-row lg:items-center justify-between gap-3 pb-4 border-b border-[#1E2638]">
                  <div className="flex items-start lg:items-center gap-3">
                    <div className="w-10 h-10 rounded-xl bg-[#0A0D12] border border-[#1E2638] flex items-center justify-center shrink-0">
                      <SoundcheckMark size={24} status={currentStatus} accentColor={accentColor} />
                    </div>

                    <div>
                      <div className="flex items-center gap-2 flex-wrap">
                        <span className="text-sm font-bold text-[#E6EAF2] font-mono">
                          {adapter.name}
                        </span>
                        <span className="text-[11px] text-[#8B95A7]">·</span>
                        <span className="text-xs text-[#8B95A7] font-mono">
                          {adapter.vendor}
                        </span>
                        <span className="text-[11px] text-[#8B95A7]">·</span>
                        <span className="text-[11px] text-[#8B95A7] font-mono">
                          {adapter.versionDetected || adapter.versionSupported}
                        </span>
                        {isCooldown && (
                          <span className="text-[10px] font-mono px-2 py-0.5 rounded bg-[#F59E0B]/20 text-[#F59E0B] border border-[#F59E0B]/40 flex items-center gap-1 font-semibold">
                            <Flame className="w-3 h-3 text-[#F59E0B]" />
                            COOLDOWN: {remainingCooldownMin}m left
                          </span>
                        )}
                      </div>
                      <div className="text-xs text-[#8B95A7] font-mono mt-0.5 flex items-center gap-2">
                        <Terminal className="w-3 h-3 text-[#8B95A7]" />
                        <code>{adapter.headlessCommand}</code>
                      </div>
                    </div>
                  </div>

                  {/* Actions for this adapter */}
                  <div className="flex items-center gap-2 self-end lg:self-auto">
                    {isCooldown && (
                      <button
                        onClick={() => onClearCooldown(adapter.id)}
                        className="px-2.5 py-1 text-xs font-mono text-[#F59E0B] hover:text-white bg-[#F59E0B]/10 hover:bg-[#F59E0B]/20 border border-[#F59E0B]/30 rounded-md transition-colors cursor-pointer"
                      >
                        Reset Cooldown
                      </button>
                    )}

                    <button
                      onClick={() => onProbeAdapter(adapter.id, 'L2')}
                      className="px-3 py-1 text-xs font-mono font-medium text-[#E6EAF2] bg-[#0A0D12] hover:bg-[#1E2638] border border-[#1E2638] rounded-md transition-colors flex items-center gap-1.5 cursor-pointer"
                    >
                      <RefreshCw className="w-3 h-3 text-[#8B95A7]" />
                      Probe L2
                    </button>
                  </div>
                </div>

                {/* 4-Tier Interactive Mixing Desk Meter Cards */}
                <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3 mt-4">
                  {/* L0 Presence */}
                  <div
                    onClick={() => setSelectedProbeLog({ adapter, level: 'L0', probe: adapter.probes.L0 })}
                    className="p-3 rounded-lg bg-[#0A0D12] border border-[#1E2638] hover:border-[#8B95A7]/60 cursor-pointer transition-all"
                  >
                    <div className="flex items-center justify-between text-xs font-mono">
                      <span className="text-[#8B95A7]">L0 PRESENCE</span>
                      <span className={`text-[11px] font-mono font-semibold ${adapter.probes.L0.status === 'passed' ? 'text-[#10B981]' : 'text-[#F59E0B]'}`}>
                        {adapter.probes.L0.status.toUpperCase()}
                      </span>
                    </div>
                    <div className="text-xs text-[#E6EAF2] mt-1 line-clamp-2" title={adapter.probes.L0.message}>
                      {adapter.probes.L0.message}
                    </div>
                    <div className="text-[10px] text-[#8B95A7] font-mono mt-1 tabular-nums flex items-center justify-between">
                      <span>Latency: {adapter.probes.L0.latencyMs}ms</span>
                      <ChevronRight className="w-3 h-3 text-[#4A5468]" />
                    </div>
                  </div>

                  {/* L1 Auth */}
                  <div
                    onClick={() => setSelectedProbeLog({ adapter, level: 'L1', probe: adapter.probes.L1 })}
                    className="p-3 rounded-lg bg-[#0A0D12] border border-[#1E2638] hover:border-[#8B95A7]/60 cursor-pointer transition-all"
                  >
                    <div className="flex items-center justify-between text-xs font-mono">
                      <span className="text-[#8B95A7]">L1 AUTHENTICATION</span>
                      <span className={`text-[11px] font-mono font-semibold ${adapter.probes.L1.status === 'passed' ? 'text-[#10B981]' : 'text-[#F87171]'}`}>
                        {adapter.probes.L1.status.toUpperCase()}
                      </span>
                    </div>
                    <div className="text-xs text-[#E6EAF2] mt-1 line-clamp-2" title={adapter.probes.L1.message}>
                      {adapter.probes.L1.message}
                    </div>
                    <div className="text-[10px] text-[#8B95A7] font-mono mt-1 tabular-nums flex items-center justify-between">
                      <span>TTL 1h · {adapter.probes.L1.latencyMs}ms</span>
                      <ChevronRight className="w-3 h-3 text-[#4A5468]" />
                    </div>
                  </div>

                  {/* L2 Functional Smoke Turn */}
                  <div
                    onClick={() => setSelectedProbeLog({ adapter, level: 'L2', probe: adapter.probes.L2 })}
                    className="p-3 rounded-lg bg-[#0A0D12] border border-[#1E2638] hover:border-[#8B95A7]/60 cursor-pointer transition-all"
                  >
                    <div className="flex items-center justify-between text-xs font-mono">
                      <span className="text-[#8B95A7]">L2 SMOKE TURN</span>
                      <span className={`text-[11px] font-mono font-semibold ${adapter.probes.L2.status === 'passed' ? 'text-[#10B981]' : 'text-[#F87171]'}`}>
                        {adapter.probes.L2.status.toUpperCase()}
                      </span>
                    </div>
                    <div className="text-xs text-[#E6EAF2] mt-1 line-clamp-2" title={adapter.probes.L2.message}>
                      {adapter.probes.L2.message}
                    </div>
                    <div className="text-[10px] text-[#8B95A7] font-mono mt-1 tabular-nums flex items-center justify-between">
                      <span>JSON · {adapter.probes.L2.latencyMs}ms</span>
                      <ChevronRight className="w-3 h-3 text-[#4A5468]" />
                    </div>
                  </div>

                  {/* L3 Model Check */}
                  <div
                    onClick={() => setSelectedProbeLog({ adapter, level: 'L3', probe: adapter.probes.L3 })}
                    className="p-3 rounded-lg bg-[#0A0D12] border border-[#1E2638] hover:border-[#8B95A7]/60 cursor-pointer transition-all"
                  >
                    <div className="flex items-center justify-between text-xs font-mono">
                      <span className="text-[#8B95A7]">L3 MODEL CHECK</span>
                      <span className={`text-[11px] font-mono font-semibold ${adapter.probes.L3.status === 'passed' ? 'text-[#10B981]' : 'text-[#8B95A7]'}`}>
                        {adapter.probes.L3.status.toUpperCase()}
                      </span>
                    </div>
                    <div className="text-xs text-[#E6EAF2] mt-1 line-clamp-2" title={adapter.probes.L3.message}>
                      {adapter.probes.L3.message}
                    </div>
                    <div className="text-[10px] text-[#8B95A7] font-mono mt-1 tabular-nums flex items-center justify-between">
                      <span>{adapter.models.length} declared models</span>
                      <ChevronRight className="w-3 h-3 text-[#4A5468]" />
                    </div>
                  </div>
                </div>

                {/* Quirk badges */}
                {adapter.quirks.length > 0 && (
                  <div className="mt-4 pt-3 border-t border-[#1E2638] flex items-center gap-2 flex-wrap text-xs">
                    <span className="text-[11px] font-mono text-[#8B95A7]">
                      Documented 2026 Quirks:
                    </span>
                    {adapter.quirks.map((q) => (
                      <button
                        key={q.id}
                        onClick={() => setSelectedQuirk(q)}
                        className="inline-flex items-center gap-1.5 px-2 py-0.5 rounded text-[11px] font-mono bg-[#EF4444]/10 hover:bg-[#EF4444]/20 text-[#F87171] border border-[#EF4444]/30 transition-colors cursor-pointer"
                      >
                        <ShieldAlert className="w-3 h-3 text-[#F87171]" />
                        <span>{q.title}</span>
                      </button>
                    ))}
                  </div>
                )}
              </div>
            );
          })}
        </div>
      </div>

      {/* Interactive Probe Log Inspector Modal */}
      {selectedProbeLog && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/80 backdrop-blur-sm animate-in fade-in duration-150">
          <div className="w-full max-w-lg p-6 rounded-2xl bg-[#121721] border border-[#1E2638] shadow-2xl space-y-4">
            <div className="flex items-start justify-between gap-4">
              <div className="flex items-center gap-2 font-mono text-xs font-bold uppercase" style={{ color: accentColor }}>
                <Code2 className="w-4 h-4" />
                <span>Probe Telemetry: {selectedProbeLog.adapter.name} ({selectedProbeLog.level})</span>
              </div>
              <button
                onClick={() => setSelectedProbeLog(null)}
                className="text-[#8B95A7] hover:text-[#E6EAF2] font-mono text-sm cursor-pointer"
              >
                ✕
              </button>
            </div>

            <div className="space-y-1">
              <h3 className="text-base font-bold font-mono text-[#E6EAF2]">
                {selectedProbeLog.probe.message}
              </h3>
              <p className="text-xs text-[#8B95A7]">
                Status: <span className="font-semibold text-[#10B981]">{selectedProbeLog.probe.status.toUpperCase()}</span> · Latency: <span className="text-[#E6EAF2] font-mono">{selectedProbeLog.probe.latencyMs}ms</span>
              </p>
            </div>

            <div className="p-3 bg-[#0A0D12] rounded-xl border border-[#1E2638] font-mono text-xs text-[#E6EAF2] space-y-2">
              <div className="text-[11px] text-[#8B95A7]">Executed Subprocess Probe:</div>
              <code className="text-[#10B981] block">
                {selectedProbeLog.level === 'L0' && `${selectedProbeLog.adapter.binary} --version`}
                {selectedProbeLog.level === 'L1' && `claude auth status / codex login status`}
                {selectedProbeLog.level === 'L2' && `${selectedProbeLog.adapter.headlessCommand} 'reply {"ok": true}'`}
                {selectedProbeLog.level === 'L3' && `${selectedProbeLog.adapter.binary} --model ${selectedProbeLog.adapter.models[0]?.id || 'default'}`}
              </code>
              <div className="text-[11px] text-[#8B95A7] pt-2 border-t border-[#1E2638]">
                Normalized Error Class: <span className="text-[#E6EAF2]">{selectedProbeLog.probe.errorClass}</span>
              </div>
            </div>

            <div className="flex justify-end">
              <button
                onClick={() => setSelectedProbeLog(null)}
                className="px-4 py-1.5 text-xs font-mono font-medium text-[#0A0D12] bg-[#E6EAF2] hover:bg-white rounded-lg transition-colors cursor-pointer"
              >
                Close
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Quirk Detail Modal */}
      {selectedQuirk && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/80 backdrop-blur-sm animate-in fade-in duration-150">
          <div className="w-full max-w-lg p-6 rounded-2xl bg-[#121721] border border-[#1E2638] shadow-2xl space-y-4">
            <div className="flex items-start justify-between gap-4">
              <div className="flex items-center gap-2 text-[#F87171] font-mono text-xs font-bold uppercase">
                <ShieldAlert className="w-4 h-4 text-[#F87171]" />
                <span>Documented CLI Quirk / Bug Report</span>
              </div>
              <button
                onClick={() => setSelectedQuirk(null)}
                className="text-[#8B95A7] hover:text-[#E6EAF2] font-mono text-sm cursor-pointer"
              >
                ✕
              </button>
            </div>

            <div>
              <h3 className="text-base font-bold text-[#E6EAF2] font-mono">
                {selectedQuirk.title}
              </h3>
              <p className="text-xs text-[#8B95A7] mt-2 leading-relaxed">
                {selectedQuirk.description}
              </p>
            </div>

            <div className="p-3 rounded-lg bg-[#0A0D12] border border-[#1E2638] space-y-2 text-xs font-mono">
              <div className="text-[#F87171]">
                <span className="text-[#8B95A7]">Pipeline Impact:</span> {selectedQuirk.impact}
              </div>
              <div className="text-[#10B981]">
                <span className="text-[#8B95A7]">Soundcheck Fix:</span> {selectedQuirk.resolution}
              </div>
              <div className="text-[#8B95A7] text-[11px] pt-1 border-t border-[#1E2638]">
                Discovered: {selectedQuirk.discoveredDate}
              </div>
            </div>

            <div className="flex items-center justify-between pt-2">
              <a
                href={selectedQuirk.referenceUrl}
                target="_blank"
                rel="noreferrer"
                className="inline-flex items-center gap-1.5 text-xs font-mono hover:underline"
                style={{ color: accentColor }}
              >
                <span>View Source Reference</span>
                <ExternalLink className="w-3.5 h-3.5" />
              </a>
              <button
                onClick={() => setSelectedQuirk(null)}
                className="px-4 py-1.5 text-xs font-mono font-medium text-[#0A0D12] bg-[#E6EAF2] hover:bg-white rounded-lg transition-colors cursor-pointer"
              >
                Close
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
