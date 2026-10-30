import React, { useState, useEffect } from 'react';
import { Type, Terminal, Palette, Server, ShieldCheck, Check, RefreshCw } from 'lucide-react';
import { SoundcheckMark } from './SoundcheckLogo';

interface SettingsViewProps {
  appFont: string;
  setAppFont: (font: string) => void;
  terminalFont: string;
  setTerminalFont: (font: string) => void;
  terminalFontSize: number;
  setTerminalFontSize: (size: number) => void;
  accentColor: string;
  setAccentColor: (color: string) => void;
  hostData: any;
}

interface FontCandidate {
  id: string;
  name: string;
  family: string;
  category: 'monospace' | 'sans';
  available: boolean;
}

const COMMON_FONTS: Omit<FontCandidate, 'available'>[] = [
  { id: 'fira-code', name: 'Fira Code', family: "'Fira Code', monospace", category: 'monospace' },
  { id: 'jetbrains-mono', name: 'JetBrains Mono', family: "'JetBrains Mono', monospace", category: 'monospace' },
  { id: 'sf-mono', name: 'SF Mono', family: "'SF Mono', monospace", category: 'monospace' },
  { id: 'cascadia-code', name: 'Cascadia Code', family: "'Cascadia Code', monospace", category: 'monospace' },
  { id: 'menlo', name: 'Menlo', family: "Menlo, monospace", category: 'monospace' },
  { id: 'monaco', name: 'Monaco', family: "Monaco, monospace", category: 'monospace' },
  { id: 'consolas', name: 'Consolas', family: "Consolas, monospace", category: 'monospace' },
  { id: 'source-code-pro', name: 'Source Code Pro', family: "'Source Code Pro', monospace", category: 'monospace' },
  { id: 'system-mono', name: 'System Monospace', family: "ui-monospace, monospace", category: 'monospace' },
  { id: 'inter', name: 'Inter', family: "Inter, sans-serif", category: 'sans' },
  { id: 'system-ui', name: 'System UI', family: "system-ui, -apple-system, sans-serif", category: 'sans' },
];

export const SettingsView: React.FC<SettingsViewProps> = ({
  appFont,
  setAppFont,
  terminalFont,
  setTerminalFont,
  terminalFontSize,
  setTerminalFontSize,
  accentColor,
  setAccentColor,
  hostData,
}) => {
  const [fonts, setFonts] = useState<FontCandidate[]>([]);
  const [detecting, setDetecting] = useState<boolean>(false);

  const detectAvailableFonts = () => {
    setDetecting(true);
    const checked = COMMON_FONTS.map((font) => {
      let available = false;
      try {
        if ('fonts' in document && typeof document.fonts.check === 'function') {
          available = document.fonts.check(`12px ${font.family}`);
        }
      } catch {
        available = true;
      }
      return { ...font, available };
    });
    setFonts(checked);
    setDetecting(false);
  };

  useEffect(() => {
    detectAvailableFonts();
  }, []);

  const accentOptions = [
    { color: '#10B981', label: 'Emerald' },
    { color: '#06B6D4', label: 'Cyan' },
    { color: '#F59E0B', label: 'Amber' },
    { color: '#8B5CF6', label: 'Purple' },
    { color: '#EC4899', label: 'Pink' },
  ];

  return (
    <div className="space-y-6 max-w-5xl">
      {/* Banner */}
      <div className="p-5 rounded-xl bg-[#121721] border border-[#1E2638] flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div>
          <div className="flex items-center gap-2.5">
            <SoundcheckMark size={20} accentColor={accentColor} />
            <h2 className="text-base font-bold text-[#E6EAF2] font-mono tracking-tight">
              APPLICATION SETTINGS &amp; WORKSPACE CONFIGURATION
            </h2>
          </div>
          <p className="text-xs text-[#8B95A7] mt-1">
            Configure UI typography, terminal render engine, color themes, and runtime environment.
          </p>
        </div>

        <button
          onClick={detectAvailableFonts}
          disabled={detecting}
          className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-mono text-[#8B95A7] hover:text-[#E6EAF2] bg-[#0A0D12] border border-[#1E2638] rounded-lg transition-colors cursor-pointer self-start md:self-auto shrink-0"
        >
          <RefreshCw className={`w-3.5 h-3.5 ${detecting ? 'animate-spin' : ''}`} />
          <span>Rescan System Fonts</span>
        </button>
      </div>

      {/* Typography Section */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* App Font Settings */}
        <div className="p-5 rounded-xl bg-[#121721] border border-[#1E2638] space-y-4">
          <div className="flex items-center gap-2 pb-3 border-b border-[#1E2638]">
            <Type className="w-4 h-4" style={{ color: accentColor }} />
            <h3 className="text-sm font-bold text-[#E6EAF2] font-mono">APP UI TYPOGRAPHY</h3>
          </div>

          <p className="text-xs text-[#8B95A7] leading-relaxed">
            Choose the typeface for navigation, telemetry metrics, and dashboards across the entire app.
          </p>

          <div className="space-y-2">
            {fonts.map((f) => {
              const isSelected = appFont === f.family;
              return (
                <div
                  key={f.id}
                  onClick={() => setAppFont(f.family)}
                  className={`p-3 rounded-lg border text-xs flex items-center justify-between cursor-pointer transition-all ${
                    isSelected
                      ? 'bg-[#0A0D12] border-[#E6EAF2] ring-1 ring-[#E6EAF2]'
                      : 'bg-[#0A0D12] border-[#1E2638] hover:border-[#2A364F]'
                  }`}
                >
                  <div className="space-y-1">
                    <div className="font-semibold text-[#E6EAF2] flex items-center gap-2">
                      <span style={{ fontFamily: f.family }}>{f.name}</span>
                      {f.available && (
                        <span className="text-[10px] text-[#10B981] bg-[#10B981]/10 px-1.5 py-0.2 rounded border border-[#10B981]/20">
                          Detected
                        </span>
                      )}
                    </div>
                    <div className="text-[11px] text-[#8B95A7]" style={{ fontFamily: f.family }}>
                      The quick brown fox jumps over the lazy dog 0123456789
                    </div>
                  </div>

                  {isSelected && (
                    <div
                      className="w-5 h-5 rounded-full flex items-center justify-center shrink-0"
                      style={{ backgroundColor: accentColor }}
                    >
                      <Check className="w-3 h-3 text-[#0A0D12]" />
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        </div>

        {/* Terminal Font Settings */}
        <div className="p-5 rounded-xl bg-[#121721] border border-[#1E2638] space-y-4">
          <div className="flex items-center gap-2 pb-3 border-b border-[#1E2638]">
            <Terminal className="w-4 h-4" style={{ color: accentColor }} />
            <h3 className="text-sm font-bold text-[#E6EAF2] font-mono">INTERACTIVE TERMINAL (PTY)</h3>
          </div>

          <p className="text-xs text-[#8B95A7] leading-relaxed">
            Select the monospace font family and character size for the live XTerm.js PTY subprocess runner.
          </p>

          <div className="space-y-4">
            <div>
              <label className="text-xs text-[#8B95A7] block mb-2 font-semibold">
                Terminal Font Size ({terminalFontSize}px)
              </label>
              <div className="flex items-center gap-2">
                {[11, 12, 13, 14, 15, 16].map((size) => (
                  <button
                    key={size}
                    onClick={() => setTerminalFontSize(size)}
                    className={`px-3 py-1.5 rounded-lg text-xs font-mono border transition-all cursor-pointer ${
                      terminalFontSize === size
                        ? 'bg-[#0A0D12] text-[#E6EAF2] border-[#E6EAF2]'
                        : 'bg-[#0A0D12] text-[#8B95A7] border-[#1E2638] hover:text-[#E6EAF2]'
                    }`}
                  >
                    {size}px
                  </button>
                ))}
              </div>
            </div>

            <div>
              <label className="text-xs text-[#8B95A7] block mb-2 font-semibold">
                Terminal Font Family
              </label>
              <div className="space-y-2">
                {fonts
                  .filter((f) => f.category === 'monospace')
                  .map((f) => {
                    const isSelected = terminalFont === f.family;
                    return (
                      <div
                        key={f.id}
                        onClick={() => setTerminalFont(f.family)}
                        className={`p-3 rounded-lg border text-xs flex items-center justify-between cursor-pointer transition-all ${
                          isSelected
                            ? 'bg-[#0A0D12] border-[#E6EAF2] ring-1 ring-[#E6EAF2]'
                            : 'bg-[#0A0D12] border-[#1E2638] hover:border-[#2A364F]'
                        }`}
                      >
                        <div className="space-y-1">
                          <div className="font-semibold text-[#E6EAF2] flex items-center gap-2">
                            <span style={{ fontFamily: f.family }}>{f.name}</span>
                          </div>
                          <div
                            className="text-[11px] text-[#10B981] bg-[#0A0D12] px-2 py-1 rounded border border-[#1E2638] inline-block"
                            style={{ fontFamily: f.family, fontSize: `${terminalFontSize}px` }}
                          >
                            $ soundcheck doctor --json
                          </div>
                        </div>

                        {isSelected && (
                          <div
                            className="w-5 h-5 rounded-full flex items-center justify-center shrink-0"
                            style={{ backgroundColor: accentColor }}
                          >
                            <Check className="w-3 h-3 text-[#0A0D12]" />
                          </div>
                        )}
                      </div>
                    );
                  })}
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Theme Accent Color */}
      <div className="p-5 rounded-xl bg-[#121721] border border-[#1E2638] space-y-3">
        <div className="flex items-center gap-2 pb-2 border-b border-[#1E2638]">
          <Palette className="w-4 h-4" style={{ color: accentColor }} />
          <h3 className="text-sm font-bold text-[#E6EAF2] font-mono">PRIMARY ACCENT THEME</h3>
        </div>

        <p className="text-xs text-[#8B95A7]">
          Custom visual brand accent applied across meters, active states, and terminal cursors.
        </p>

        <div className="flex items-center gap-3 pt-1">
          {accentOptions.map((opt) => (
            <button
              key={opt.color}
              onClick={() => setAccentColor(opt.color)}
              className={`flex items-center gap-2 px-3 py-1.5 rounded-lg border text-xs transition-all cursor-pointer ${
                accentColor === opt.color
                  ? 'bg-[#0A0D12] text-[#E6EAF2] border-[#E6EAF2]'
                  : 'bg-[#0A0D12] text-[#8B95A7] border-[#1E2638] hover:text-[#E6EAF2]'
              }`}
            >
              <span className="w-3 h-3 rounded-full" style={{ backgroundColor: opt.color }} />
              <span>{opt.label}</span>
            </button>
          ))}
        </div>
      </div>

      {/* Runtime & Host Diagnostic */}
      <div className="p-5 rounded-xl bg-[#121721] border border-[#1E2638] space-y-3">
        <div className="flex items-center gap-2 pb-2 border-b border-[#1E2638]">
          <Server className="w-4 h-4" style={{ color: accentColor }} />
          <h3 className="text-sm font-bold text-[#E6EAF2] font-mono">HOST ENVIRONMENT &amp; BACKEND</h3>
        </div>

        <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-4 gap-4 text-xs font-mono">
          <div className="p-3 rounded-lg bg-[#0A0D12] border border-[#1E2638]">
            <span className="text-[#8B95A7] block text-[11px]">PLATFORM</span>
            <span className="text-[#E6EAF2] font-bold">{hostData?.platform || 'darwin / local'}</span>
          </div>
          <div className="p-3 rounded-lg bg-[#0A0D12] border border-[#1E2638]">
            <span className="text-[#8B95A7] block text-[11px]">NODE RUNTIME</span>
            <span className="text-[#E6EAF2] font-bold">{hostData?.nodeVersion || 'v25.x'}</span>
          </div>
          <div className="p-3 rounded-lg bg-[#0A0D12] border border-[#1E2638]">
            <span className="text-[#8B95A7] block text-[11px]">GIT WORKTREE</span>
            <span className="text-[#10B981] font-bold">{hostData?.gitWorktree ? 'Active' : 'Standby'}</span>
          </div>
          <div className="p-3 rounded-lg bg-[#0A0D12] border border-[#1E2638]">
            <span className="text-[#8B95A7] block text-[11px]">IMMUNE QUIRKS</span>
            <span className="text-[#E6EAF2] font-bold">10 Defenses</span>
          </div>
        </div>
      </div>
    </div>
  );
};
