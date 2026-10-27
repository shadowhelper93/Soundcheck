import React, { useState } from 'react';
import { KNOWN_QUIRKS_FIXTURES, QuirkFixture } from '../core/fixtures/knownQuirksFixtures';
import { classifyError } from '../core/adapters/adapterRegistry';
import { SoundcheckMark } from './SoundcheckLogo';
import {
  Bug,
  ShieldAlert,
  Play,
  RotateCcw,
  Code2
} from 'lucide-react';

interface QuirkLabViewProps {
  accentColor: string;
}

export const QuirkLabView: React.FC<QuirkLabViewProps> = ({ accentColor }) => {
  const [selectedFixture, setSelectedFixture] = useState<QuirkFixture>(KNOWN_QUIRKS_FIXTURES[0]);
  const [customExitCode, setCustomExitCode] = useState<number>(selectedFixture.simulatedExitCode);
  const [customStdout, setCustomStdout] = useState<string>(selectedFixture.simulatedStdout);
  const [customStderr, setCustomStderr] = useState<string>(selectedFixture.simulatedStderr);
  const [testResult, setTestResult] = useState<any>(null);

  const handleSelectFixture = (f: QuirkFixture) => {
    setSelectedFixture(f);
    setCustomExitCode(f.simulatedExitCode);
    setCustomStdout(f.simulatedStdout);
    setCustomStderr(f.simulatedStderr);
    setTestResult(null);
  };

  const runClassifierTest = () => {
    let parsed: any = null;
    try {
      if (customStdout.trim().startsWith('{') || customStdout.trim().startsWith('[')) {
        parsed = JSON.parse(customStdout);
      }
    } catch {
      parsed = null;
    }

    const result = classifyError(
      selectedFixture.agentId,
      customStdout,
      customStderr,
      customExitCode,
      parsed
    );

    setTestResult(result);
  };

  return (
    <div className="space-y-6">
      {/* Title & Introduction */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4 p-5 rounded-xl bg-[#121721] border border-[#1E2638]">
        <div>
          <div className="flex items-center gap-2.5">
            <SoundcheckMark size={24} status="fake_0" accentColor={accentColor} />
            <h2 className="text-base font-bold text-[#E6EAF2] font-mono tracking-tight">
              2026 CLI QUIRK LABORATORY &amp; LIE DETECTOR
            </h2>
          </div>
          <p className="text-xs text-[#8B95A7] mt-1">
            "A plain slash = an ordinary zero = unverified." Inspect real-world false exit 0s and silent auth failures.
          </p>
        </div>

        <div className="flex items-center gap-2">
          <span className="text-xs font-mono text-[#10B981] px-3 py-1.5 rounded-lg bg-[#0A0D12] border border-[#1E2638]">
            ✓ 6 Real 2026 Bug Fixtures Active
          </span>
        </div>
      </div>

      {/* Fixture Selector Tabs */}
      <div className="flex items-center gap-2 overflow-x-auto pb-2 scrollbar-none">
        {KNOWN_QUIRKS_FIXTURES.map((fixture) => {
          const isSelected = selectedFixture.id === fixture.id;
          return (
            <button
              key={fixture.id}
              onClick={() => handleSelectFixture(fixture)}
              className={`px-3.5 py-2 text-xs font-mono rounded-lg border whitespace-nowrap shrink-0 transition-all cursor-pointer ${
                isSelected
                  ? 'bg-[#121721] text-[#E6EAF2] border-[#E6EAF2] shadow-sm'
                  : 'bg-[#121721] text-[#8B95A7] border-[#1E2638] hover:text-[#E6EAF2]'
              }`}
            >
              {fixture.name}
            </button>
          );
        })}
      </div>

      {/* Selected Bug Overview Banner */}
      <div className="p-5 rounded-xl bg-[#121721] border border-[#1E2638] space-y-3">
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-2">
          <div className="flex items-center gap-2">
            <span className="text-xs font-bold font-mono px-2 py-0.5 rounded bg-[#EF4444]/20 text-[#F87171] border border-[#EF4444]/30">
              {selectedFixture.docIssue}
            </span>
            <span className="text-xs font-mono text-[#8B95A7]">
              Category: {selectedFixture.category.replace('_', ' ')}
            </span>
          </div>

          <span className="text-xs font-mono text-[#8B95A7]">
            Target Adapter: <code className="text-[#E6EAF2]">{selectedFixture.agentId}</code>
          </span>
        </div>

        <p className="text-xs text-[#E6EAF2] leading-relaxed font-mono">
          {selectedFixture.description}
        </p>

        {/* Contrast Comparison Grid */}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4 pt-2">
          {/* Naive Orchestrator Verdict */}
          <div className="p-4 rounded-lg bg-[#0A0D12] border border-[#EF4444]/40 space-y-1.5 font-mono text-xs">
            <div className="flex items-center gap-2 text-[#F87171] font-bold">
              <SoundcheckMark size={20} status="fake_0" accentColor={accentColor} />
              <span>NAIVE ORCHESTRATOR OUTCOME (SLOP RUNNER)</span>
            </div>
            <div className="text-[#F87171] text-[11px] leading-relaxed">
              {selectedFixture.naiveOutcome}
            </div>
          </div>

          {/* Soundcheck Verdict */}
          <div className="p-4 rounded-lg bg-[#0A0D12] border border-[#10B981]/40 space-y-1.5 font-mono text-xs">
            <div className="flex items-center gap-2 text-[#10B981] font-bold">
              <SoundcheckMark size={20} status="ok" accentColor={accentColor} />
              <span>SOUNDCHECK ADAPTER OUTCOME (GROUND TRUTH)</span>
            </div>
            <div className="text-[#10B981] text-[11px] leading-relaxed">
              {selectedFixture.soundcheckOutcome}
            </div>
          </div>
        </div>
      </div>

      {/* Interactive Sandbox: Edit and Test classifyError */}
      <div className="p-6 rounded-xl bg-[#121721] border border-[#1E2638] space-y-4">
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 pb-3 border-b border-[#1E2638]">
          <div>
            <h3 className="text-xs font-bold font-mono text-[#E6EAF2] uppercase flex items-center gap-1.5">
              <Code2 className="w-4 h-4" style={{ color: accentColor }} />
              Interactive Adapter Sandbox (classifyError)
            </h3>
            <p className="text-[11px] text-[#8B95A7] mt-0.5">
              Modify the raw subprocess exit code or stdout JSON and click "Run Classifier Test" to test error normalization.
            </p>
          </div>

          <div className="flex items-center gap-2">
            <button
              onClick={() => handleSelectFixture(selectedFixture)}
              className="px-3 py-1.5 text-xs font-mono text-[#8B95A7] hover:text-[#E6EAF2] bg-[#0A0D12] border border-[#1E2638] rounded-lg transition-colors flex items-center gap-1 cursor-pointer"
            >
              <RotateCcw className="w-3 h-3" />
              <span>Reset</span>
            </button>

            <button
              onClick={runClassifierTest}
              style={{ backgroundColor: accentColor }}
              className="flex items-center gap-1.5 px-4 py-1.5 text-xs font-mono font-bold text-[#0A0D12] hover:brightness-110 rounded-lg shadow-sm transition-all cursor-pointer"
            >
              <Play className="w-3.5 h-3.5 fill-[#0A0D12]" />
              <span>RUN CLASSIFIER TEST</span>
            </button>
          </div>
        </div>

        {/* Inputs */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <div>
            <label className="block text-[11px] font-mono text-[#8B95A7] mb-1">
              Process Exit Code
            </label>
            <input
              type="number"
              value={customExitCode}
              onChange={(e) => setCustomExitCode(parseInt(e.target.value, 10) || 0)}
              className="w-full px-3 py-2 text-xs font-mono bg-[#0A0D12] border border-[#1E2638] rounded-lg text-[#E6EAF2] focus:outline-none focus:border-[#E6EAF2]"
            />
            <span className="text-[10px] text-[#8B95A7] mt-1 block">
              Notice: Exit code 0 is often a false success on rate limits.
            </span>
          </div>

          <div className="md:col-span-2">
            <label className="block text-[11px] font-mono text-[#8B95A7] mb-1">
              Captured stdout / NDJSON Stream
            </label>
            <textarea
              rows={6}
              value={customStdout}
              onChange={(e) => setCustomStdout(e.target.value)}
              className="w-full p-2.5 text-xs font-mono bg-[#0A0D12] border border-[#1E2638] rounded-lg text-[#E6EAF2] focus:outline-none focus:border-[#E6EAF2] leading-relaxed resize-none"
            />
          </div>
        </div>

        {/* Classifier Output Box */}
        {testResult && (
          <div className="p-4 rounded-xl bg-[#0A0D12] border border-[#1E2638] space-y-2 font-mono text-xs animate-in fade-in duration-150">
            <div className="flex items-center justify-between">
              <span className="text-[#8B95A7] font-bold uppercase">
                classifyError() Execution Output
              </span>
              {testResult.caughtLie ? (
                <span className="px-2 py-0.5 rounded bg-[#EF4444]/20 text-[#F87171] border border-[#EF4444]/30 text-[10px] font-bold flex items-center gap-1.5">
                  <SoundcheckMark size={14} status="fake_0" accentColor={accentColor} />
                  ⚡ CLI LIE CAUGHT: FALSE EXIT 0 DETECTED!
                </span>
              ) : (
                <span className="px-2 py-0.5 rounded bg-[#10B981]/20 text-[#10B981] border border-[#10B981]/30 text-[10px] font-bold flex items-center gap-1.5">
                  <SoundcheckMark size={14} status="ok" accentColor={accentColor} />
                  ✓ ACCURATE REPORTING
                </span>
              )}
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 pt-1">
              <div>
                <span className="text-[#8B95A7] text-[11px]">Normalized Error Class:</span>
                <div className="text-base font-bold text-[#E6EAF2]">
                  {testResult.errorClass.toUpperCase()}
                </div>
              </div>

              <div>
                <span className="text-[#8B95A7] text-[11px]">Analysis &amp; Reason:</span>
                <div className="text-xs text-[#8B95A7] leading-relaxed mt-0.5">
                  {testResult.reason}
                </div>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
};
