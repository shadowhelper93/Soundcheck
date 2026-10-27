import React, { useState, useRef, useEffect, useCallback } from 'react';
import { Terminal as TerminalIcon, Send, RotateCcw, Copy, Check, Wifi, WifiOff } from 'lucide-react';
import { Terminal } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';

interface InteractiveTerminalProps {
  accentColor: string;
}

export const InteractiveTerminal: React.FC<InteractiveTerminalProps> = ({ accentColor }) => {
  const [command, setCommand] = useState<string>('soundcheck doctor');
  const [connected, setConnected] = useState<boolean>(false);
  const [copied, setCopied] = useState<boolean>(false);

  const terminalContainerRef = useRef<HTMLDivElement>(null);
  const xtermRef = useRef<Terminal | null>(null);
  const fitAddonRef = useRef<FitAddon | null>(null);
  const wsRef = useRef<WebSocket | null>(null);

  const quickCommands = [
    'soundcheck doctor',
    'soundcheck resolve',
    'soundcheck manifest',
    'git status'
  ];

  const connectWebSocket = useCallback(() => {
    if (wsRef.current && (wsRef.current.readyState === WebSocket.OPEN || wsRef.current.readyState === WebSocket.CONNECTING)) {
      return;
    }

    const term = xtermRef.current;
    if (!term) return;

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const host = window.location.host || '127.0.0.1:3000';
    const cols = term.cols || 80;
    const rows = term.rows || 24;
    const wsUrl = `${protocol}//${host}/api/v1/terminal/ws?cols=${cols}&rows=${rows}`;

    term.write(`\r\n\x1b[90mConnecting to ${wsUrl}...\x1b[0m\r\n`);

    try {
      const ws = new WebSocket(wsUrl);
      wsRef.current = ws;

      ws.onopen = () => {
        setConnected(true);
        term.write(`\x1b[32m✔ Connected to Soundcheck PTY terminal session\x1b[0m\r\n\r\n`);
      };

      ws.onmessage = (event) => {
        try {
          const msg = JSON.parse(event.data);
          if (msg.type === 'terminal_chunk' && typeof msg.data === 'string') {
            term.write(msg.data);
          } else if (msg.type === 'exit') {
            term.write(`\r\n\x1b[90m[Process exited with code ${msg.exitCode}]\x1b[0m\r\n`);
          } else if (typeof event.data === 'string') {
            term.write(event.data);
          }
        } catch {
          // If message is raw string
          term.write(event.data);
        }
      };

      ws.onerror = () => {
        setConnected(false);
        term.write(`\r\n\x1b[31m[WebSocket connection error to /api/v1/terminal/ws]\x1b[0m\r\n`);
      };

      ws.onclose = () => {
        setConnected(false);
        term.write(`\r\n\x1b[90m[WebSocket connection closed]\x1b[0m\r\n`);
      };
    } catch (err: any) {
      setConnected(false);
      term.write(`\r\n\x1b[31mFailed to connect WebSocket: ${err?.message || err}\x1b[0m\r\n`);
    }
  }, []);

  useEffect(() => {
    if (!terminalContainerRef.current) return;

    // Initialize xterm
    const term = new Terminal({
      cursorBlink: true,
      fontFamily: "'Fira Code', monospace",
      fontSize: 13,
      lineHeight: 1.25,
      theme: {
        background: '#0A0D12',
        foreground: '#E6EAF2',
        cursor: accentColor,
        selectionBackground: 'rgba(16, 185, 129, 0.3)',
        black: '#0A0D12',
        red: '#EF4444',
        green: '#10B981',
        yellow: '#F59E0B',
        blue: '#3B82F6',
        magenta: '#EC4899',
        cyan: '#06B6D4',
        white: '#E6EAF2',
        brightBlack: '#4A5468',
        brightRed: '#F87171',
        brightGreen: '#34D399',
        brightYellow: '#FBBF24',
        brightBlue: '#60A5FA',
        brightMagenta: '#F472B6',
        brightCyan: '#22D3EE',
        brightWhite: '#FFFFFF',
      },
    });

    const fitAddon = new FitAddon();
    term.loadAddon(fitAddon);

    term.open(terminalContainerRef.current);
    fitAddon.fit();

    xtermRef.current = term;
    fitAddonRef.current = fitAddon;

    term.writeln(`\x1b[1;36mSoundcheck Real PTY Terminal\x1b[0m [Fira Code Ligatures Enabled]`);
    term.writeln(`\x1b[90mIntegrated with Go WebSocket Server (/api/v1/terminal/ws)\x1b[0m`);

    // Handle user input to WebSocket
    term.onData((data) => {
      const ws = wsRef.current;
      if (ws && ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: 'input', data }));
      }
    });

    // Handle resize
    term.onResize(({ cols, rows }) => {
      const ws = wsRef.current;
      if (ws && ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: 'resize', cols, rows }));
      }
    });

    const handleWindowResize = () => {
      try {
        fitAddon.fit();
      } catch {
        // ignore
      }
    };
    window.addEventListener('resize', handleWindowResize);

    // Initial connection
    connectWebSocket();

    return () => {
      window.removeEventListener('resize', handleWindowResize);
      if (wsRef.current) {
        wsRef.current.close();
      }
      term.dispose();
    };
  }, [accentColor, connectWebSocket]);

  const sendCommand = (cmdToSend: string) => {
    const cleanCmd = cmdToSend.trim();
    if (!cleanCmd) return;
    const ws = wsRef.current;
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify({ type: 'input', data: `${cleanCmd}\r` }));
    } else {
      xtermRef.current?.write(`\r\n\x1b[33mNot connected. Reconnecting...\x1b[0m\r\n`);
      connectWebSocket();
    }
  };

  const handleRunCommand = (e: React.FormEvent) => {
    e.preventDefault();
    sendCommand(command);
  };

  const handleClear = () => {
    xtermRef.current?.clear();
  };

  const handleCopy = () => {
    const term = xtermRef.current;
    if (!term) return;
    term.selectAll();
    const selection = term.getSelection();
    term.clearSelection();
    if (selection) {
      navigator.clipboard.writeText(selection);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
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
            /api/v1/terminal/ws
          </span>
          <span
            className={`inline-flex items-center gap-1 text-[10px] px-2 py-0.5 rounded-full border ${
              connected
                ? 'bg-emerald-950/60 text-emerald-400 border-emerald-800/80'
                : 'bg-rose-950/60 text-rose-400 border-rose-800/80'
            }`}
          >
            {connected ? (
              <>
                <Wifi className="w-2.5 h-2.5" />
                Connected
              </>
            ) : (
              <>
                <WifiOff className="w-2.5 h-2.5" />
                Offline
              </>
            )}
          </span>
        </div>

        <div className="flex items-center gap-2">
          {!connected && (
            <button
              onClick={connectWebSocket}
              className="text-xs text-[#06B6D4] hover:underline cursor-pointer"
            >
              Reconnect
            </button>
          )}
          <button
            onClick={handleCopy}
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
            onClick={handleClear}
            className="flex items-center gap-1 text-xs text-[#8B95A7] hover:text-[#E6EAF2] transition-colors cursor-pointer"
          >
            <RotateCcw className="w-3 h-3" />
            <span>Clear</span>
          </button>
        </div>
      </div>

      {/* Quick Launch Buttons */}
      <div className="flex flex-wrap items-center gap-1.5">
        <span className="text-[11px] text-[#8B95A7] mr-1">Quick:</span>
        {quickCommands.map((qc) => (
          <button
            key={qc}
            onClick={() => {
              setCommand(qc);
              sendCommand(qc);
            }}
            className="text-[11px] bg-[#0A0D12] border border-[#1E2638] hover:border-[#10B981] hover:text-[#10B981] text-[#E6EAF2] px-2 py-0.5 rounded transition-colors cursor-pointer"
          >
            {qc}
          </button>
        ))}
      </div>

      {/* xterm Container */}
      <div
        ref={terminalContainerRef}
        className="h-72 w-full rounded-lg bg-[#0A0D12] border border-[#1E2638] overflow-hidden p-2 text-left"
        style={{ fontFeatureSettings: '"calt" 1, "liga" 1' }}
      />

      {/* Command Input Bar */}
      <form onSubmit={handleRunCommand} className="flex gap-2">
        <div className="relative flex-1">
          <span className="absolute left-3 top-2.5 text-xs text-[#8B95A7]">$</span>
          <input
            type="text"
            value={command}
            onChange={(e) => setCommand(e.target.value)}
            placeholder="Type command or shell input..."
            className="w-full bg-[#0A0D12] border border-[#1E2638] rounded-lg pl-7 pr-4 py-2 text-xs font-mono text-[#E6EAF2] focus:outline-none focus:border-[#10B981] transition-colors"
          />
        </div>
        <button
          type="submit"
          className="px-4 py-2 bg-[#161D2B] hover:bg-[#1E2638] border border-[#2A364F] text-xs font-medium text-[#E6EAF2] rounded-lg flex items-center gap-1.5 transition-colors cursor-pointer"
        >
          <Send className="w-3.5 h-3.5" style={{ color: accentColor }} />
          <span>Send</span>
        </button>
      </form>
    </div>
  );
};
