import React, { useState, useEffect } from 'react';
import { CapabilityManifest, SoundcheckConfig } from '../types/soundcheck';
import { generateCrewplaneHook, generateCIPreflightBash } from '../core/engine/manifestGenerator';
import {
  FileCode2,
  Copy,
  Check,
  Download,
  RefreshCw
} from 'lucide-react';

interface ManifestModalProps {
  manifest: CapabilityManifest;
  config: SoundcheckConfig;
  accentColor: string;
}

export const ManifestModal: React.FC<ManifestModalProps> = ({ manifest, config, accentColor }) => {
  const [selectedFormat, setSelectedFormat] = useState<'manifest' | 'yaml' | 'crewplane' | 'ci'>('manifest');
  const [copied, setCopied] = useState<boolean>(false);
  const [liveManifest, setLiveManifest] = useState<CapabilityManifest | null>(null);
  const [isLoading, setIsLoading] = useState<boolean>(false);

  const fetchLiveManifest = async () => {
    setIsLoading(true);
    try {
      const res = await fetch('/api/v1/manifest');
      if (res.ok) {
        const data = await res.json();
        setLiveManifest(data);
      }
    } catch {
      // fallback to generated manifest
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    fetchLiveManifest();
  }, []);

  const effectiveManifest = liveManifest || manifest;

  const getContent = () => {
    switch (selectedFormat) {
      case 'manifest':
        return JSON.stringify(effectiveManifest, null, 2);
      case 'yaml':
        return `# Soundcheck Canonical Configuration (agent-doctor.yaml)
# Validated against JSON Schema 2026.1
version: ${config.version}

roles:
  primary:
${config.roles.primary.map((c) => `    - "${c}"`).join('\n')}
  subagent:
${config.roles.subagent.map((c) => `    - "${c}"`).join('\n')}
  auxiliary:
${config.roles.auxiliary.map((c) => `    - "${c}"`).join('\n')}
  reviewer:
${config.roles.reviewer.map((c) => `    - "${c}"`).join('\n')}

policy:
  require_distinct_vendor: [${config.policy.require_distinct_vendor.join(', ')}]
  max_cost_usd_per_run: ${config.policy.max_cost_usd_per_run}
  probe:
    deep: ${config.policy.probe.deep}
    ttl_minutes: ${config.policy.probe.ttl_minutes}
`;
      case 'crewplane':
        return generateCrewplaneHook(effectiveManifest);
      case 'ci':
        return generateCIPreflightBash();
    }
  };

  const currentContent = getContent();

  const handleCopy = () => {
    navigator.clipboard.writeText(currentContent);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const handleDownload = () => {
    const filenameMap = {
      manifest: 'soundcheck-manifest.json',
      yaml: 'agent-doctor.yaml',
      crewplane: 'soundcheck-provider-hook.ts',
      ci: 'preflight.sh'
    };
    const mimeMap = {
      manifest: 'application/json',
      yaml: 'text/yaml',
      crewplane: 'text/typescript',
      ci: 'application/x-sh'
    };

    const blob = new Blob([currentContent], { type: mimeMap[selectedFormat] });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = filenameMap[selectedFormat];
    a.click();
    URL.revokeObjectURL(url);
  };

  return (
    <div className="space-y-6">
      {/* Header and Controls */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 p-5 rounded-xl bg-[#121721] border border-[#1E2638]">
        <div>
          <div className="flex items-center gap-2">
            <span className="font-mono text-xs text-[#8B95A7] uppercase tracking-[0.08em]">
              Output Formats &amp; Interop
            </span>
            <span className="text-[10px] text-[#06B6D4] border border-[#06B6D4]/30 rounded px-1.5 py-0.2">
              /api/v1/manifest
            </span>
            {liveManifest && (
              <span className="text-[10px] text-emerald-400 bg-emerald-950/60 border border-emerald-800 rounded px-1.5 py-0.2">
                Live Server Synced
              </span>
            )}
          </div>
          <h2 className="text-xl font-bold font-mono text-[#E6EAF2] mt-1">
            Deterministic Capability Manifest &amp; Tool Contracts
          </h2>
          <p className="text-xs text-[#8B95A7] mt-1">
            Machine-readable JSON schema 2026.1 and config artifacts ready for agent orchestration engines.
          </p>
        </div>

        <div className="flex items-center gap-2">
          <button
            onClick={fetchLiveManifest}
            disabled={isLoading}
            className="px-3 py-1.5 bg-[#0A0D12] hover:bg-[#161D2B] border border-[#1E2638] text-xs font-mono text-[#8B95A7] hover:text-[#E6EAF2] rounded-lg flex items-center gap-1.5 transition-colors cursor-pointer disabled:opacity-50"
            title="Refetch live manifest from /api/v1/manifest"
          >
            <RefreshCw className={`w-3.5 h-3.5 ${isLoading ? 'animate-spin' : ''}`} />
            <span>Sync</span>
          </button>
          <button
            onClick={handleCopy}
            className="px-3 py-1.5 bg-[#0A0D12] hover:bg-[#161D2B] border border-[#1E2638] text-xs font-mono text-[#E6EAF2] rounded-lg flex items-center gap-1.5 transition-colors cursor-pointer"
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
            onClick={handleDownload}
            className="px-3 py-1.5 bg-[#161D2B] hover:bg-[#1E2638] border border-[#2A364F] text-xs font-mono font-medium text-[#E6EAF2] rounded-lg flex items-center gap-1.5 transition-colors cursor-pointer"
          >
            <Download className="w-3.5 h-3.5" style={{ color: accentColor }} />
            <span>Download</span>
          </button>
        </div>
      </div>

      {/* Format Selector Tabs */}
      <div className="flex items-center gap-2 border-b border-[#1E2638] pb-2">
        <button
          onClick={() => setSelectedFormat('manifest')}
          className={`px-3 py-1.5 text-xs font-mono rounded-lg transition-colors cursor-pointer ${
            selectedFormat === 'manifest'
              ? 'bg-[#121721] text-[#E6EAF2] border border-[#1E2638]'
              : 'text-[#8B95A7] hover:text-[#E6EAF2]'
          }`}
          style={selectedFormat === 'manifest' ? { borderColor: `${accentColor}80` } : {}}
        >
          doctor --json (Manifest)
        </button>

        <button
          onClick={() => setSelectedFormat('yaml')}
          className={`px-3 py-1.5 text-xs font-mono rounded-lg transition-colors cursor-pointer ${
            selectedFormat === 'yaml'
              ? 'bg-[#121721] text-[#E6EAF2] border border-[#1E2638]'
              : 'text-[#8B95A7] hover:text-[#E6EAF2]'
          }`}
          style={selectedFormat === 'yaml' ? { borderColor: `${accentColor}80` } : {}}
        >
          agent-doctor.yaml
        </button>

        <button
          onClick={() => setSelectedFormat('crewplane')}
          className={`px-3 py-1.5 text-xs font-mono rounded-lg transition-colors cursor-pointer ${
            selectedFormat === 'crewplane'
              ? 'bg-[#121721] text-[#E6EAF2] border border-[#1E2638]'
              : 'text-[#8B95A7] hover:text-[#E6EAF2]'
          }`}
          style={selectedFormat === 'crewplane' ? { borderColor: `${accentColor}80` } : {}}
        >
          Crewplane Provider Hook
        </button>

        <button
          onClick={() => setSelectedFormat('ci')}
          className={`px-3 py-1.5 text-xs font-mono rounded-lg transition-colors cursor-pointer ${
            selectedFormat === 'ci'
              ? 'bg-[#121721] text-[#E6EAF2] border border-[#1E2638]'
              : 'text-[#8B95A7] hover:text-[#E6EAF2]'
          }`}
          style={selectedFormat === 'ci' ? { borderColor: `${accentColor}80` } : {}}
        >
          GitHub Actions CI Preflight
        </button>
      </div>

      {/* Code Viewer */}
      <div className="relative rounded-xl bg-[#0A0D12] border border-[#1E2638] p-4 font-mono text-xs overflow-x-auto max-h-[560px]">
        <div className="absolute right-4 top-4 flex items-center gap-2">
          <span className="text-[10px] text-[#8B95A7] uppercase">
            {selectedFormat === 'manifest' && 'JSON (Schema 2026.1)'}
            {selectedFormat === 'yaml' && 'YAML Specification'}
            {selectedFormat === 'crewplane' && 'TypeScript / Node'}
            {selectedFormat === 'ci' && 'Bash / Linux'}
          </span>
        </div>

        <pre className="text-[#E6EAF2] leading-relaxed selection:bg-[#10B981]/30">
          <code>{currentContent}</code>
        </pre>
      </div>
    </div>
  );
};
