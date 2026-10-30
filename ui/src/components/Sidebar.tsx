import React from 'react';
import {
  ShieldCheck,
  Cpu,
  GitFork,
  Bug,
  FileCode2,
  Settings,
  Play,
  Activity,
  Layers,
  ChevronRight
} from 'lucide-react';
import { SoundcheckMark } from './SoundcheckLogo';

export type ActiveTab = 'doctor' | 'resolver' | 'workflow' | 'quirks' | 'manifest' | 'settings';

interface SidebarProps {
  activeTab: ActiveTab;
  setActiveTab: (tab: ActiveTab) => void;
  onRunDiagnostics: () => void;
  isRunning: boolean;
  accentColor: string;
  activeResolvedAgent?: string;
  functionalAgentsCount?: number;
  totalAgentsCount?: number;
}

export const Sidebar: React.FC<SidebarProps> = ({
  activeTab,
  setActiveTab,
  onRunDiagnostics,
  isRunning,
  accentColor,
  activeResolvedAgent = 'claude-code',
  functionalAgentsCount = 5,
  totalAgentsCount = 6
}) => {
  const navItems: { id: ActiveTab; label: string; number: string; icon: React.ReactNode; badge?: string }[] = [
    {
      id: 'doctor',
      number: '01',
      label: 'Doctor',
      icon: <ShieldCheck className="w-4 h-4" />,
      badge: `${functionalAgentsCount}/${totalAgentsCount}`
    },
    {
      id: 'resolver',
      number: '02',
      label: 'Resolver',
      icon: <Cpu className="w-4 h-4" />
    },
    {
      id: 'workflow',
      number: '03',
      label: 'DAG Studio',
      icon: <GitFork className="w-4 h-4" />
    },
    {
      id: 'quirks',
      number: '04',
      label: 'Quirk Lab',
      icon: <Bug className="w-4 h-4" />,
      badge: '10'
    },
    {
      id: 'manifest',
      number: '05',
      label: 'Manifest',
      icon: <FileCode2 className="w-4 h-4" />
    },
    {
      id: 'settings',
      number: '06',
      label: 'Settings',
      icon: <Settings className="w-4 h-4" />
    }
  ];

  return (
    <aside className="w-64 bg-[#0A0D12] border-r border-[#1E2638] flex flex-col shrink-0 min-h-screen sticky top-0 h-screen overflow-y-auto">
      {/* Brand Header */}
      <div className="p-4 border-b border-[#1E2638] flex items-center justify-between">
        <div className="flex items-center gap-2.5">
          <SoundcheckMark size={22} accentColor={accentColor} />
          <div>
            <span className="font-mono font-bold text-base tracking-[-0.02em] text-[#E6EAF2] block leading-none">
              sound<span style={{ color: accentColor }}>check</span>
            </span>
            <span className="font-mono text-[10px] text-[#8B95A7] block mt-1">
              v1.4.0-oct2026
            </span>
          </div>
        </div>
      </div>

      {/* Primary Action Button */}
      <div className="p-3">
        <button
          onClick={onRunDiagnostics}
          disabled={isRunning}
          style={{ backgroundColor: accentColor }}
          className="w-full flex items-center justify-center gap-2 px-3.5 py-2.5 text-xs font-mono font-bold text-[#0A0D12] hover:brightness-110 disabled:opacity-50 rounded-lg shadow-sm transition-all cursor-pointer"
        >
          {isRunning ? (
            <>
              <span className="w-3.5 h-3.5 border-2 border-[#0A0D12] border-t-transparent rounded-full animate-spin" />
              <span>SCANNING HOST...</span>
            </>
          ) : (
            <>
              <Play className="w-3.5 h-3.5 fill-[#0A0D12]" />
              <span>RUN SOUNDCHECK</span>
            </>
          )}
        </button>
      </div>

      {/* Host Status Indicator */}
      <div className="mx-3 mb-2 p-2.5 rounded-lg bg-[#121721] border border-[#1E2638] text-xs font-mono">
        <div className="flex items-center justify-between text-[#8B95A7] text-[11px] mb-1">
          <span>ACTIVE AGENT</span>
          <span className="text-[#10B981] font-semibold flex items-center gap-1">
            <span className="w-1.5 h-1.5 rounded-full bg-[#10B981] animate-pulse" />
            L3 ok
          </span>
        </div>
        <div className="text-[#E6EAF2] font-bold truncate">
          {activeResolvedAgent}
        </div>
      </div>

      {/* Navigation Section */}
      <div className="px-3 py-2 flex-1 space-y-1">
        <div className="text-[10px] font-mono uppercase tracking-wider text-[#4A5468] px-2 mb-2 font-semibold">
          NAVIGATION
        </div>

        {navItems.map((item) => {
          const isActive = activeTab === item.id;
          return (
            <button
              key={item.id}
              onClick={() => setActiveTab(item.id)}
              className={`w-full flex items-center justify-between px-3 py-2 text-xs font-mono rounded-lg transition-all cursor-pointer ${
                isActive
                  ? 'bg-[#121721] text-[#E6EAF2] border border-[#1E2638] font-bold shadow-sm'
                  : 'text-[#8B95A7] hover:text-[#E6EAF2] hover:bg-[#121721]/50 border border-transparent'
              }`}
              style={isActive ? { borderColor: `${accentColor}60` } : {}}
            >
              <div className="flex items-center gap-2.5">
                <span style={isActive ? { color: accentColor } : { color: '#8B95A7' }}>
                  {item.icon}
                </span>
                <span>
                  <span className="text-[#4A5468] mr-1.5">{item.number}</span>
                  {item.label}
                </span>
              </div>

              {item.badge && (
                <span
                  className={`text-[10px] px-1.5 py-0.5 rounded border ${
                    isActive
                      ? 'bg-[#0A0D12] text-[#E6EAF2] border-[#1E2638]'
                      : 'bg-[#121721] text-[#8B95A7] border-[#1E2638]'
                  }`}
                >
                  {item.badge}
                </span>
              )}
            </button>
          );
        })}
      </div>

      {/* Footer telemetry */}
      <div className="p-3 border-t border-[#1E2638] text-[11px] font-mono text-[#4A5468] space-y-1">
        <div className="flex items-center justify-between">
          <span>PORT: 3000</span>
          <span>WEBSOCKET: OK</span>
        </div>
        <div>IMMUNE DEFENSES: 10 QUIRKS</div>
      </div>
    </aside>
  );
};
