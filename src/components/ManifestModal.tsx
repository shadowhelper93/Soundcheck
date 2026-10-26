import React, { useState } from 'react';
import { CapabilityManifest, SoundcheckConfig } from '../types/soundcheck';
import { generateCrewplaneHook, generateCIPreflightBash } from '../core/engine/manifestGenerator';
import {
  FileCode2,
  Copy,
  Check,
  Download
} from 'lucide-react';

interface ManifestModalProps {
  manifest: CapabilityManifest;
  config: SoundcheckConfig;
  accentColor: string;
}

export const ManifestModal: React.FC<ManifestModalProps> = ({ manifest, config, accentColor }) => {
  const [selectedFormat, setSelectedFormat] = useState<'manifest' | 'yaml' | 'crewplane' | 'ci'>('manifest');
  const [copied, setCopied] = useState<boolean>(false);

  const getContent = () => {
    switch (selectedFormat) {
      case 'manifest':
        return JSON.stringify(manifest, null, 2);
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
        return generateCrewplaneHook(manifest);
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
    const filename =
      selectedFormat === 'manifest'
        ? 'doctor-manifest.json'
        : selectedFormat === 'yaml'
        ? 'agent-doctor.yaml'
        : selectedFormat === 'crewplane'
        ? 'crewplane_provider_hook.py'
        : 'preflight-check.sh';

    const blob = new Blob([currentContent], { type: 'text/plain' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = filename;
    a.click();
    URL.revokeObjectURL(url);
  };

  return (
    <div className="space-y-6">
      {/* Title & Description */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4 p-5 rounded-xl bg-[#121721] border border-[#1E2638]">
        <div>
          <div className="flex items-center gap-2">
            <FileCode2 className="w-5 h-5" style={{ color: accentColor }} />
            <h2 className="text-base font-bold text-[#E6EAF2] font-mono tracking-tight">
              MACHINE-READABLE CAPABILITY MANIFEST &amp; INTEGRATIONS
            </h2>
          </div>
          <p className="text-xs text-[#8B95A7] mt-1">
            Standard capability manifest emitted by <code className="text-[#E6EAF2]">doctor --json</code>. Read by Crewplane, Vibe Kanban, and CI/CD runners.
          </p>
        </div>

        <div className="flex items-center gap-2">
          <button
            onClick={handleCopy}
            className="flex items-center gap-1.5 px-3.5 py-1.5 text-xs font-mono font-medium text-[#E6EAF2] bg-[#0A0D12] hover:bg-[#1E2638] border border-[#1E2638] rounded-lg transition-colors cursor-pointer"
          >
            {copied ? (
              <>
                <Check className="w-3.5 h-3.5 text-[#10B981]" />
                <span className="text-[#10B981]">Copied</span>
              </>
            ) : (
              <>
                <Copy className="w-3.5 h-3.5 text-[#8B95A7]" />
                <span>Copy Output</span>
              </>
            )}
          </button>

          <button
            onClick={handleDownload}
            style={{ backgroundColor: accentColor }}
            className="flex items-center gap-1.5 px-3.5 py-1.5 text-xs font-mono font-bold text-[#0A0D12] hover:brightness-110 rounded-lg transition-all cursor-pointer"
          >
            <Download className="w-3.5 h-3.5" />
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
          CI Preflight Gate (Bash)
        </button>
      </div>

      {/* Code Viewer */}
      <div className="p-5 rounded-xl bg-[#121721] border border-[#1E2638] space-y-2">
        <div className="flex items-center justify-between text-[11px] font-mono text-[#8B95A7] pb-2 border-b border-[#1E2638]">
          <span>Schema Version: 2026.1 · Machine-readable Output</span>
          <span>{currentContent.split('\n').length} lines</span>
        </div>
        <pre className="p-3 bg-[#0A0D12] rounded-lg border border-[#1E2638] font-mono text-xs text-[#E6EAF2] overflow-x-auto max-h-[500px] leading-relaxed scrollbar-thin">
          {currentContent}
        </pre>
      </div>
    </div>
  );
};
