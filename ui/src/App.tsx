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

  const updateAdapterFromLive = (prev: AgentAdapter[], live: any): AgentAdapter[] => {
    return prev.map((a) => {
      if (a.id !== live.id && a.binary !== live.binary) return a;
      return {
        ...a,
        installed: typeof live.installed === 'boolean' ? live.installed : a.installed,
        versionDetected: live.versionDetected || live.version || a.versionDetected,
        cooldownUntil: live.cooldownUntil ?? a.cooldownUntil,
        probes: {
          L0: live.probes?.L0 ? { ...a.probes.L0, ...live.probes.L0 } : a.probes.L0,
          L1: live.probes?.L1 ? { ...a.probes.L1, ...live.probes.L1 } : a.probes.L1,
          L2: live.probes?.L2 ? { ...a.probes.L2, ...live.probes.L2 } : a.probes.L2,
          L3: live.probes?.L3 ? { ...a.probes.L3, ...live.probes.L3 } : a.probes.L3,
        }
      };
    });
  };

  const fetchHostProbe = async () => {
    try {
      const res = await fetch('/api/v1/doctor');
      if (res.ok) {
        const data = await res.json();
        if (data.host) {
          setHostData({
            platform: data.host.platform || data.host.os || 'darwin',
            osName: data.host.os || 'macOS',
            nodeVersion: data.host.nodeVersion || (data.host.node ? 'v22.x' : 'n/a'),
            gitWorktreeAvailable: data.host.gitWorktree ?? data.host.git ?? true
          });
        }
        if (Array.isArray(data.adapters)) {
          setAdapters((prev) => {
            let updated = [...prev];
            for (const live of data.adapters) {
              updated = updateAdapterFromLive(updated, live);
            }
            return updated;
          });
        }
      }
    } catch {
      // Offline fallback
    }
  };

  // Run full real preflight soundcheck across all monitored agents
  const handleRunFullDiagnostics = () => {
    setIsScanning(true);
    try {
      if (typeof EventSource !== 'undefined') {
        const es = new EventSource('/api/v1/doctor');
        es.addEventListener('adapter', (e) => {
          try {
            const live = JSON.parse(e.data);
            setAdapters((prev) => updateAdapterFromLive(prev, live));
          } catch {
            // ignore
          }
        });
        es.addEventListener('report', (e) => {
          try {
            const report = JSON.parse(e.data);
            if (report.host) {
              setHostData({
                platform: report.host.platform || report.host.os || 'darwin',
                osName: report.host.os || 'macOS',
                nodeVersion: report.host.nodeVersion || 'v22.x',
                gitWorktreeAvailable: report.host.gitWorktree ?? true
              });
            }
            if (Array.isArray(report.adapters)) {
              setAdapters((prev) => {
                let updated = [...prev];
                for (const live of report.adapters) {
                  updated = updateAdapterFromLive(updated, live);
                }
                return updated;
              });
            }
          } catch {
            // ignore
          }
        });
        es.addEventListener('done', () => {
          es.close();
          setIsScanning(false);
        });
        es.onerror = () => {
          es.close();
          fetchHostProbe().finally(() => setIsScanning(false));
        };
        return;
      }
    } catch {
      // EventSource failed to initialize
    }

    fetchHostProbe().finally(() => setIsScanning(false));
  };

  // Real single-adapter probe execution via Go backend
  const handleProbeAdapter = async (adapterId: string, level: ProbeLevel) => {
    try {
      const res = await fetch(`/api/v1/probe/${adapterId}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ level })
      });
      if (res.ok) {
        const live = await res.json();
        setAdapters((prev) => updateAdapterFromLive(prev, live));
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
