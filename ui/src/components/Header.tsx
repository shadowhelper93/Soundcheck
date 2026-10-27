import React from 'react';
import { ShieldCheck, Cpu, GitFork, Bug, FileCode2, Play, Palette } from 'lucide-react';
import { SoundcheckMark } from './SoundcheckLogo';

export type ActiveTab = 'doctor' | 'resolver' | 'workflow' | 'quirks' | 'manifest' | 'design_system';

interface HeaderProps {
  activeTab: ActiveTab;
  setActiveTab: (tab: ActiveTab) => void;
  onRunDiagnostics: () => void;
  isRunning: boolean;
  lieCount: number;
  accentColor: string;
  setAccentColor: (color: string) => void;
  activeResolvedAgent?: string;
}

export const Header: React.FC<HeaderProps> = ({
  activeTab,
  setActiveTab,
  onRunDiagnostics,
  isRunning,
  lieCount,
  accentColor,
  setAccentColor,
  activeResolvedAgent = 'claude-code'
}) => {
  const accentOptions = ['#10B981', '#06B6D4', '#F59E0B'];

  return (
    <header className="sticky top-0 z-40 bg-[#0A0D12]/95 backdrop-blur-md border-b border-[#1E2638] px-4 md:px-6 py-2.5">
      <div className="max-w-7xl mx-auto flex items-center justify-between gap-4">
        {/* Zone 1: Exact Brand Lockup from Marks.dc.html Section 06 */}
        <div className="flex items-center gap-2.5 shrink-0">
          <div className="bg-[#0A0D12] border border-[#1E2638] rounded-lg px-3 py-2 flex items-center gap-2.5">
            <SoundcheckMark size={22} accentColor={accentColor} />
            <span className="font-mono font-bold text-base tracking-[-0.02em] text-[#E6EAF2]">
              sound<span style={{ color: accentColor }}>check</span>
            </span>
            <span className="font-mono text-[11px] text-[#8B95A7] border border-[#1E2638] rounded px-1.5 py-0.5">
              v1.4.0-oct2026
            </span>
          </div>

          {/* In-bar Telemetry Badge from Marks.dc.html Section 06 */}
          <div className="hidden xl:flex items-center gap-2 bg-[#0A0D12] border border-[#1E2638] rounded-lg px-3 py-2 font-mono text-xs text-[#8B95A7]">
            <span className="text-[#E6EAF2]">{activeResolvedAgent}</span>
            <span>·</span>
            <span className="text-[#10B981] font-semibold flex items-center gap-1">
              <span className="w-1.5 h-1.5 rounded-full bg-[#10B981] animate-pulse" />
              L3 ok
            </span>
          </div>
        </div>

        {/* Zone 2: Navigation Links with Exact Design Component Styling */}
        <nav className="hidden lg:flex items-center gap-1 bg-[#121721] p-1 rounded-lg border border-[#1E2638]">
          <button
            onClick={() => setActiveTab('doctor')}
            className={`flex items-center gap-1.5 px-3 py-1.5 text-xs font-mono font-medium rounded-md transition-all whitespace-nowrap shrink-0 cursor-pointer ${
              activeTab === 'doctor'
                ? 'bg-[#0A0D12] text-[#E6EAF2] border border-[#1E2638]'
                : 'text-[#8B95A7] hover:text-[#E6EAF2]'
            }`}
            style={activeTab === 'doctor' ? { borderColor: `${accentColor}80` } : {}}
          >
            <ShieldCheck className="w-3.5 h-3.5" style={activeTab === 'doctor' ? { color: accentColor } : {}} />
            <span>01 · Doctor</span>
          </button>

          <button
            onClick={() => setActiveTab('resolver')}
            className={`flex items-center gap-1.5 px-3 py-1.5 text-xs font-mono font-medium rounded-md transition-all whitespace-nowrap shrink-0 cursor-pointer ${
              activeTab === 'resolver'
                ? 'bg-[#0A0D12] text-[#E6EAF2] border border-[#1E2638]'
                : 'text-[#8B95A7] hover:text-[#E6EAF2]'
            }`}
            style={activeTab === 'resolver' ? { borderColor: `${accentColor}80` } : {}}
          >
            <Cpu className="w-3.5 h-3.5" style={activeTab === 'resolver' ? { color: accentColor } : {}} />
            <span>02 · Resolver</span>
          </button>

          <button
            onClick={() => setActiveTab('workflow')}
            className={`flex items-center gap-1.5 px-3 py-1.5 text-xs font-mono font-medium rounded-md transition-all whitespace-nowrap shrink-0 cursor-pointer ${
              activeTab === 'workflow'
                ? 'bg-[#0A0D12] text-[#E6EAF2] border border-[#1E2638]'
                : 'text-[#8B95A7] hover:text-[#E6EAF2]'
            }`}
            style={activeTab === 'workflow' ? { borderColor: `${accentColor}80` } : {}}
          >
            <GitFork className="w-3.5 h-3.5" style={activeTab === 'workflow' ? { color: accentColor } : {}} />
            <span>03 · DAG Studio</span>
          </button>

          <button
            onClick={() => setActiveTab('quirks')}
            className={`flex items-center gap-1.5 px-3 py-1.5 text-xs font-mono font-medium rounded-md transition-all whitespace-nowrap shrink-0 cursor-pointer ${
              activeTab === 'quirks'
                ? 'bg-[#0A0D12] text-[#E6EAF2] border border-[#1E2638]'
                : 'text-[#8B95A7] hover:text-[#E6EAF2]'
            }`}
            style={activeTab === 'quirks' ? { borderColor: `${accentColor}80` } : {}}
          >
            <Bug className="w-3.5 h-3.5" style={activeTab === 'quirks' ? { color: accentColor } : {}} />
            <span>04 · Quirk Lab</span>
            {lieCount > 0 && (
              <span className="text-[10px] font-mono px-1 rounded bg-[#EF4444]/20 text-[#F87171] border border-[#EF4444]/40">
                {lieCount}
              </span>
            )}
          </button>

          <button
            onClick={() => setActiveTab('manifest')}
            className={`flex items-center gap-1.5 px-3 py-1.5 text-xs font-mono font-medium rounded-md transition-all whitespace-nowrap shrink-0 cursor-pointer ${
              activeTab === 'manifest'
                ? 'bg-[#0A0D12] text-[#E6EAF2] border border-[#1E2638]'
                : 'text-[#8B95A7] hover:text-[#E6EAF2]'
            }`}
            style={activeTab === 'manifest' ? { borderColor: `${accentColor}80` } : {}}
          >
            <FileCode2 className="w-3.5 h-3.5" style={activeTab === 'manifest' ? { color: accentColor } : {}} />
            <span>05 · Manifest</span>
          </button>

          <button
            onClick={() => setActiveTab('design_system')}
            className={`flex items-center gap-1.5 px-3 py-1.5 text-xs font-mono font-medium rounded-md transition-all whitespace-nowrap shrink-0 cursor-pointer ${
              activeTab === 'design_system'
                ? 'bg-[#0A0D12] text-[#E6EAF2] border border-[#1E2638]'
                : 'text-[#8B95A7] hover:text-[#E6EAF2]'
            }`}
            style={activeTab === 'design_system' ? { borderColor: `${accentColor}80` } : {}}
          >
            <Palette className="w-3.5 h-3.5" style={activeTab === 'design_system' ? { color: accentColor } : {}} />
            <span>06 · Brand System</span>
          </button>
        </nav>

        {/* Zone 3: Accent Color Switcher & Primary Action CTA */}
        <div className="flex items-center gap-3 shrink-0">
          {/* Accent Color Palette Switcher from Marks.dc.html data-props */}
          <div className="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg bg-[#121721] border border-[#1E2638]">
            <span className="text-[10px] font-mono text-[#8B95A7] hidden sm:inline mr-1">ACCENT</span>
            {accentOptions.map((color) => (
              <button
                key={color}
                onClick={() => setAccentColor(color)}
                style={{ backgroundColor: color }}
                className={`w-3.5 h-3.5 rounded-full transition-transform cursor-pointer ${
                  accentColor === color ? 'ring-2 ring-[#E6EAF2] scale-110' : 'opacity-60 hover:opacity-100'
                }`}
                title={`Switch accent to ${color}`}
              />
            ))}
          </div>

          <button
            onClick={onRunDiagnostics}
            disabled={isRunning}
            style={{ backgroundColor: accentColor }}
            className="flex items-center gap-2 px-4 py-2 text-xs font-bold text-[#0A0D12] hover:brightness-110 active:brightness-95 disabled:opacity-50 rounded-lg shadow-sm transition-all whitespace-nowrap shrink-0 font-mono tracking-tight cursor-pointer"
          >
            {isRunning ? (
              <>
                <span className="w-3.5 h-3.5 border-2 border-[#0A0D12] border-t-transparent rounded-full animate-spin" />
                <span>SOUNDCHECKING...</span>
              </>
            ) : (
              <>
                <Play className="w-3.5 h-3.5 fill-[#0A0D12]" />
                <span>RUN SOUNDCHECK</span>
              </>
            )}
          </button>
        </div>
      </div>
    </header>
  );
};
