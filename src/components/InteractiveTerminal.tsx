import React, { useState, useRef, useEffect } from 'react';
import { Terminal as TerminalIcon, Send, Play, RotateCcw, Copy, Check } from 'lucide-react';

interface TerminalLine {
  id: string;
  type: 'input' | 'output' | 'error' | 'system';
  text: string;
  exitCode?: number;
  durationMs?: number;
}

interface InteractiveTerminalProps {
  accentColor: string;
}

export const InteractiveTerminal: React.FC<InteractiveTerminalProps> = ({ accentColor }) => {
  const [command, setCommand] = useState<string>('soundcheck doctor');
  const [history, setHistory] = useState<TerminalLine[]>([
    {
      id: '1',
      type: 'system',
      text: 'Soundcheck Real Subprocess Terminal Console [v1.4.0-oct2026]\nType any permitted command or click a quick shortcut below to run on the live server.'
    }
  ]);
  const [isRunning, setIsRunning] = useState<boolean>(false);
  const [copied, setCopied] = useState<boolean>(false);
  const terminalEndRef = useRef<HTMLDivElement>(null);

  const quickCommands = [
    'soundcheck doctor',
    'soundcheck resolve',
    'git status',
    'npm run lint',
    'tsc --noEmit'
  ];

  useEffect(() => {
    terminalEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [history]);

  const executeCommand = async (cmdToRun: string) => {
    if (!cmdToRun.trim() || isRunning) return;
    const cleanCmd = cmdToRun.trim();

    // Append input line
    setHistory((prev) => [
      ...prev,
      {
        id: String(Date.now()),
        type: 'input',
        text: cleanCmd
      }
    ]);

    setIsRunning(true);

    try {
      const res = await fetch('/api/terminal/exec', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ command: cleanCmd })
      });

      const data = await res.json();

      if (data.stdout) {
        setHistory((prev) => [
          ...prev,
          {
            id: String(Date.now() + 1),
            type: 'output',
            text: data.stdout,
            exitCode: data.exitCode,
            durationMs: data.durationMs
          }
        ]);
      }

      if (data.stderr) {
        setHistory((prev) => [
          ...prev,
          {
            id: String(Date.now() + 2),
            type: 'error',
            text: data.stderr,
            exitCode: data.exitCode,
            durationMs: data.durationMs
          }
        ]);
      }

      if (!data.stdout && !data.stderr) {
        setHistory((prev) => [
          ...prev,
          {
            id: String(Date.now() + 1),
            type: 'system',
            text: `(Command completed with exit code ${data.exitCode} in ${data.durationMs}ms with zero output)`,
            exitCode: data.exitCode,
            durationMs: data.durationMs
          }
        ]);
      }
    } catch (err: any) {
      setHistory((prev) => [
        ...prev,
        {
          id: String(Date.now() + 1),
          type: 'error',
          text: `Fetch failure: ${err.message}`
        }
      ]);
    } finally {
      setIsRunning(false);
      setCommand('');
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') {
      executeCommand(command);
    }
  };

  const clearTerminal = () => {
    setHistory([
      {
        id: String(Date.now()),
        type: 'system',
        text: 'Terminal cleared.'
      }
    ]);
  };

  const copyTerminalOutput = () => {
    const fullText = history.map((h) => `${h.type === 'input' ? '$ ' : ''}${h.text}`).join('\n');
    navigator.clipboard.writeText(fullText);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="p-5 rounded-xl bg-[#121721] border border-[#1E2638] space-y-3 font-mono">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-2 pb-3 border-b border-[#1E2638]">
        <div className="flex items-center gap-2">
          <TerminalIcon className="w-4 h-4" style={{ color: accentColor }} />
          <span className="text-xs font-bold text-[#E6EAF2] uppercase">
            Live Subprocess Terminal &amp; CLI Runner
          </span>
          <span className="text-[10px] text-[#8B95A7] border border-[#1E2638] rounded px-1.5 py-0.5">
            Real Server Bash
          </span>
        </div>

        <div className="flex items-center gap-2">
          <button
            onClick={copyTerminalOutput}
            className="flex items-center gap-1 text-xs text-[#8B95A7] hover:text-[#E6EAF2] transition-colors cursor-pointer"
          >
            {copied ? (
              <>
                <Check className="w-3.5 h-3.5 text-[#10B981]" />
                <span className="text-[#10B981]">Copied</span>
              </>
            ) : (
              <>
                <Copy className="w-3.5 h-3.5" />
                <span>Copy</span>
              </>
            )}
          </button>
          <button
            onClick={clearTerminal}
            className="flex items-center gap-1 text-xs text-[#8B95A7] hover:text-[#E6EAF2] transition-colors cursor-pointer"
          >
            <RotateCcw className="w-3 h-3" />
            <span>Clear</span>
          </button>
        </div>
      </div>

      {/* Quick Execution Shortcuts */}
      <div className="flex items-center gap-1.5 flex-wrap text-xs">
        <span className="text-[11px] text-[#8B95A7]">Shortcuts:</span>
        {quickCommands.map((qc) => (
          <button
            key={qc}
            onClick={() => executeCommand(qc)}
            disabled={isRunning}
            className="px-2 py-0.5 rounded bg-[#0A0D12] hover:bg-[#1E2638] border border-[#1E2638] text-[11px] text-[#E6EAF2] hover:text-white transition-colors cursor-pointer disabled:opacity-50"
          >
            {qc}
          </button>
        ))}
      </div>

      {/* Terminal Viewport */}
      <div className="p-3 bg-[#0A0D12] rounded-lg border border-[#1E2638] h-64 overflow-y-auto space-y-2 text-xs leading-relaxed">
        {history.map((line) => (
          <div key={line.id} className="space-y-0.5">
            {line.type === 'input' && (
              <div className="flex items-center gap-2 font-bold" style={{ color: accentColor }}>
                <span className="text-[#8B95A7]">$</span>
                <span>{line.text}</span>
              </div>
            )}
            {line.type === 'output' && (
              <pre className="text-[#E6EAF2] whitespace-pre-wrap pl-3 border-l border-[#1E2638] font-mono">
                {line.text}
              </pre>
            )}
            {line.type === 'error' && (
              <pre className="text-[#F87171] whitespace-pre-wrap pl-3 border-l border-[#EF4444]/40 font-mono">
                {line.text}
              </pre>
            )}
            {line.type === 'system' && (
              <div className="text-[#8B95A7] text-[11px] italic">
                {line.text}
              </div>
            )}
            {line.exitCode !== undefined && (
              <div className="text-[10px] text-[#8B95A7] flex items-center gap-2 pl-3">
                <span className={line.exitCode === 0 ? 'text-[#10B981]' : 'text-[#F87171]'}>
                  exit {line.exitCode}
                </span>
                {line.durationMs !== undefined && <span>· {line.durationMs}ms</span>}
              </div>
            )}
          </div>
        ))}
        {isRunning && (
          <div className="flex items-center gap-2 text-xs text-[#8B95A7] animate-pulse">
            <span className="w-2 h-2 rounded-full bg-[#10B981]" />
            <span>Executing subprocess command on server...</span>
          </div>
        )}
        <div ref={terminalEndRef} />
      </div>

      {/* Input row */}
      <div className="flex items-center gap-2 pt-1">
        <div className="relative flex-1">
          <span className="absolute left-3 top-2.5 text-xs text-[#8B95A7]">$</span>
          <input
            type="text"
            value={command}
            onChange={(e) => setCommand(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="Enter command (e.g. soundcheck doctor, git status, npm run lint)..."
            disabled={isRunning}
            className="w-full bg-[#0A0D12] border border-[#1E2638] rounded-lg pl-7 pr-3 py-2 text-xs text-[#E6EAF2] focus:outline-none focus:border-[#E6EAF2] font-mono"
          />
        </div>
        <button
          onClick={() => executeCommand(command)}
          disabled={!command.trim() || isRunning}
          style={{ backgroundColor: accentColor }}
          className="px-4 py-2 rounded-lg text-xs font-bold text-[#0A0D12] hover:brightness-110 disabled:opacity-50 transition-all flex items-center gap-1.5 cursor-pointer shrink-0"
        >
          <Send className="w-3.5 h-3.5 fill-[#0A0D12]" />
          <span>EXEC</span>
        </button>
      </div>
    </div>
  );
};
