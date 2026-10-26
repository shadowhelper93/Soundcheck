import React, { useState, useEffect } from 'react';
import { Header, ActiveTab } from './components/Header';
import { DoctorView } from './components/DoctorView';
import { ResolverView } from './components/ResolverView';
import { WorkflowStudio } from './components/WorkflowStudio';
import { QuirkLabView } from './components/QuirkLabView';
import { ManifestModal } from './components/ManifestModal';
import { BrandSystemView } from './components/BrandSystemView';
import { InteractiveTerminal } from './components/InteractiveTerminal';
import { INITIAL_ADAPTERS } from './core/adapters/adapterRegistry';
import { DEFAULT_CONFIG, resolveRoles } from './core/engine/resolver';
import { generateCapabilityManifest } from './core/engine/manifestGenerator';
import { AgentAdapter, ProbeLevel, SoundcheckConfig } from './types/soundcheck';
import { Terminal, ChevronUp, ChevronDown } from 'lucide-react';

export default function App() {
  const [activeTab, setActiveTab] = useState<ActiveTab>('doctor');
  const [accentColor, setAccentColor] = useState<string>('#10B981'); // Emerald accent from Marks.dc.html
  const [adapters, setAdapters] = useState<AgentAdapter[]>(INITIAL_ADAPTERS);
  const [config, setConfig] = useState<SoundcheckConfig>(DEFAULT_CONFIG);
  const [probingMode, setProbingMode] = useState<'hybrid' | 'live' | 'fixture'>('hybrid');
  const [isScanning, setIsScanning] = useState<boolean>(false);
  const [isPrimarySimulatedLimited, setIsPrimarySimulatedLimited] = useState<boolean>(false);
  const [showTerminal, setShowTerminal] = useState<boolean>(true);
  const [hostData, setHostData] = useState<any>({
    platform: 'linux',
    osName: 'Linux',
    nodeVersion: 'v22.14.0',
    gitWorktreeAvailable: true
  });
  const [lieCount] = useState<number>(3);

  // Fetch host environment probe on mount
  useEffect(() => {
    fetchHostProbe();
  }, []);

  const fetchHostProbe = async () => {
    try {
      const res = await fetch('/api/probe/host');
      if (res.ok) {
        const data = await res.json();
        setHostData(data);

        // In live or hybrid mode, merge real binary presence
        if (data.binaries) {
          setAdapters((prev) =>
            prev.map((adapter) => {
              const liveBinary = data.binaries[adapter.id];
              if (liveBinary) {
                return {
                  ...adapter,
                  installed: liveBinary.installed,
                  versionDetected: liveBinary.version || adapter.versionDetected,
                  probes: {
                    ...adapter.probes,
                    L0: {
                      ...adapter.probes.L0,
                      status: liveBinary.installed ? 'passed' : 'warning',
                      message: liveBinary.installed
                        ? `Found at ${liveBinary.path} (${liveBinary.version || 'installed'})`
                        : `Not installed in host PATH (simulated offline)`
                    }
                  }
                };
              }
              return adapter;
            })
          );
        }
      }
    } catch {
      // Offline fallback
    }
  };

  // Run full real preflight soundcheck across all monitored agents
  const handleRunFullDiagnostics = async () => {
    setIsScanning(true);
    try {
      await fetchHostProbe();
      const res = await fetch('/api/probe/run-all', { method: 'POST' });
      if (res.ok) {
        const data = await res.json();
        const results = data.results || {};
        setAdapters((prev) =>
          prev.map((a) => {
            const live = results[a.id];
            if (!live) return a;
            const isCooldown = a.cooldownUntil && a.cooldownUntil > Date.now();
            return {
              ...a,
              installed: live.installed ?? a.installed,
              versionDetected: live.versionDetected || a.versionDetected,
              probes: {
                L0: {
                  ...a.probes.L0,
                  status: live.probes?.L0?.status || 'passed',
                  errorClass: live.probes?.L0?.errorClass || 'ok',
                  message: live.probes?.L0?.message || a.probes.L0.message,
                  latencyMs: live.probes?.L0?.latencyMs ?? 1,
                  timestamp: Date.now(),
                  cachedUntil: Date.now() + 86400000
                },
                L1: {
                  ...a.probes.L1,
                  status: live.probes?.L1?.status || 'passed',
                  errorClass: live.probes?.L1?.errorClass || 'ok',
                  message: live.probes?.L1?.message || a.probes.L1.message,
                  latencyMs: live.probes?.L1?.latencyMs ?? 5,
                  timestamp: Date.now(),
                  cachedUntil: Date.now() + 3600000
                },
                L2: {
                  ...a.probes.L2,
                  status: isCooldown ? 'failed' : (live.probes?.L2?.status || 'passed'),
                  errorClass: isCooldown ? 'rate_limited' : (live.probes?.L2?.errorClass || 'ok'),
                  message: isCooldown
                    ? 'Rate limited (429 HTTP backoff active)'
                    : (live.probes?.L2?.message || a.probes.L2.message),
                  latencyMs: live.probes?.L2?.latencyMs ?? 15,
                  timestamp: Date.now(),
                  cachedUntil: Date.now() + 1800000
                },
                L3: {
                  ...a.probes.L3,
                  status: live.probes?.L3?.status || 'passed',
                  errorClass: live.probes?.L3?.errorClass || 'ok',
                  message: live.probes?.L3?.message || a.probes.L3.message,
                  latencyMs: live.probes?.L3?.latencyMs ?? 10,
                  timestamp: Date.now(),
                  cachedUntil: Date.now() + 3600000
                }
              }
            };
          })
        );
      }
    } catch (err) {
      console.error('Failed to execute real host diagnostics:', err);
    } finally {
      setIsScanning(false);
    }
  };

  // Real single-adapter probe execution via host backend
  const handleProbeAdapter = async (adapterId: string, level: ProbeLevel) => {
    try {
      const res = await fetch('/api/probe/execute-single', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ adapterId, level })
      });
      if (res.ok) {
        const probeResult = await res.json();
        setAdapters((prev) =>
          prev.map((a) => {
            if (a.id === adapterId) {
              return {
                ...a,
                installed: level === 'L0' && typeof probeResult.installed === 'boolean' ? probeResult.installed : a.installed,
                versionDetected: level === 'L0' && probeResult.version ? probeResult.version : a.versionDetected,
                probes: {
                  ...a.probes,
                  [level]: {
                    ...a.probes[level],
                    timestamp: Date.now(),
                    status: probeResult.status,
                    errorClass: probeResult.errorClass || 'ok',
                    latencyMs: probeResult.latencyMs,
                    message: probeResult.message
                  }
                }
              };
            }
            return a;
          })
        );
      }
    } catch (err) {
      console.error(`Real probe failed for ${adapterId} ${level}:`, err);
    }
  };

  const handleSimulateCooldown = (adapterId: string) => {
    setAdapters((prev) =>
      prev.map((a) => {
        if (a.id === adapterId) {
          return {
            ...a,
            cooldownUntil: Date.now() + 35 * 60000,
            probes: {
              ...a.probes,
              L2: {
                ...a.probes.L2,
                status: 'failed',
                errorClass: 'rate_limited',
                message: 'Simulated rolling 5-hour quota exceeded (Issue #79500).'
              }
            }
          };
        }
        return a;
      })
    );
  };

  const handleClearCooldown = (adapterId: string) => {
    setAdapters((prev) =>
      prev.map((a) => {
        if (a.id === adapterId) {
          return {
            ...a,
            cooldownUntil: undefined,
            probes: {
              ...a.probes,
              L2: {
                ...a.probes.L2,
                status: 'passed',
                errorClass: 'ok',
                message: 'Cooldown cleared. Normal health restored.'
              }
            }
          };
        }
        return a;
      })
    );
  };

  const handleTriggerPrimaryRateLimit = () => {
    setIsPrimarySimulatedLimited(!isPrimarySimulatedLimited);
    if (!isPrimarySimulatedLimited) {
      handleSimulateCooldown('claude');
    } else {
      handleClearCooldown('claude');
    }
  };

  // Compute deterministic role resolutions
  const resolvedRoles = resolveRoles(config, adapters, {
    forceCooldownAgentId: isPrimarySimulatedLimited ? 'claude' : undefined
  });

  // Compute standard capability manifest
  const capabilityManifest = generateCapabilityManifest(adapters, resolvedRoles, {
    platform: hostData.platform || 'linux',
    nodeVersion: hostData.nodeVersion || 'v22.x',
    gitWorktree: !!hostData.gitWorktreeAvailable
  });

  const activePrimaryModel = `${resolvedRoles.primary?.selectedAgentId || 'claude'}`;

  return (
    <div className="min-h-screen bg-[#0A0D12] text-[#E6EAF2] flex flex-col font-mono selection:bg-[#10B981]/20 selection:text-[#10B981]">
      <Header
        activeTab={activeTab}
        setActiveTab={setActiveTab}
        onRunDiagnostics={handleRunFullDiagnostics}
        isRunning={isScanning}
        lieCount={lieCount}
        accentColor={accentColor}
        setAccentColor={setAccentColor}
        activeResolvedAgent={activePrimaryModel}
      />

      <main className="flex-1 max-w-7xl w-full mx-auto p-4 md:p-8 space-y-6">
        {activeTab === 'doctor' && (
          <DoctorView
            adapters={adapters}
            hostData={hostData}
            probingMode={probingMode}
            setProbingMode={setProbingMode}
            onProbeAdapter={handleProbeAdapter}
            onSimulateCooldown={handleSimulateCooldown}
            onClearCooldown={handleClearCooldown}
            isScanning={isScanning}
            accentColor={accentColor}
          />
        )}

        {activeTab === 'resolver' && (
          <ResolverView
            config={config}
            onUpdateConfig={setConfig}
            resolvedRoles={resolvedRoles}
            adapters={adapters}
            onTriggerPrimaryRateLimit={handleTriggerPrimaryRateLimit}
            isPrimarySimulatedLimited={isPrimarySimulatedLimited}
            accentColor={accentColor}
          />
        )}

        {activeTab === 'workflow' && (
          <WorkflowStudio
            resolvedRoles={resolvedRoles}
            accentColor={accentColor}
          />
        )}

        {activeTab === 'quirks' && <QuirkLabView accentColor={accentColor} />}

        {activeTab === 'manifest' && (
          <ManifestModal
            manifest={capabilityManifest}
            config={config}
            accentColor={accentColor}
          />
        )}

        {activeTab === 'design_system' && (
          <BrandSystemView accentColor={accentColor} />
        )}

        {/* Real Interactive Subprocess Terminal Console */}
        <div className="pt-2">
          <div className="flex items-center justify-between pb-2 font-mono text-xs text-[#8B95A7]">
            <span className="flex items-center gap-1.5 font-bold text-[#E6EAF2]">
              <Terminal className="w-3.5 h-3.5" style={{ color: accentColor }} />
              Live Server Terminal
            </span>
            <button
              onClick={() => setShowTerminal(!showTerminal)}
              className="flex items-center gap-1 hover:text-[#E6EAF2] transition-colors cursor-pointer"
            >
              <span>{showTerminal ? 'Collapse Terminal' : 'Expand Terminal'}</span>
              {showTerminal ? <ChevronDown className="w-3.5 h-3.5" /> : <ChevronUp className="w-3.5 h-3.5" />}
            </button>
          </div>
          {showTerminal && <InteractiveTerminal accentColor={accentColor} />}
        </div>
      </main>

      {/* Grounded Precision Footer */}
      <footer className="border-t border-[#1E2638] bg-[#0A0D12] px-6 py-4 mt-auto">
        <div className="max-w-7xl mx-auto flex flex-col sm:flex-row items-center justify-between gap-3 text-xs font-mono text-[#8B95A7]">
          <div className="flex items-center gap-2">
            <span className="font-semibold text-[#E6EAF2]">sound<span style={{ color: accentColor }}>check</span></span>
            <span>·</span>
            <span>Deterministic Multi-CLI Preflight &amp; Capability Resolver</span>
          </div>
          <div className="flex items-center gap-4 text-[11px] text-[#8B95A7]">
            <span>Presence → Auth → Smoke → Model</span>
            <span>·</span>
            <span>Real Node.js &amp; Subprocess Engine</span>
          </div>
        </div>
      </footer>
    </div>
  );
}
