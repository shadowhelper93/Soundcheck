import React, { useEffect, useState } from 'react';
import { CheckCircle2, AlertCircle, FileText, Split, DollarSign, Clock, ShieldAlert } from 'lucide-react';

interface Check {
  role: string;
  candidate: string;
  verified: boolean;
  reason?: string;
}

const ROLES = ['primary', 'subagent', 'auxiliary', 'reviewer'];

interface RoleConfigEditorProps {
  accentColor: string;
  onConfigSaved?: (savedCfg: any) => void;
}

export const RoleConfigEditor: React.FC<RoleConfigEditorProps> = ({ accentColor, onConfigSaved }) => {
  const [cfg, setCfg] = useState<any>(null);
  const [checks, setChecks] = useState<Check[]>([]);
  const [configPath, setConfigPath] = useState<string>('agent-doctor.yaml');
  const [msg, setMsg] = useState<{ text: string; error: boolean } | null>(null);
  const [saving, setSaving] = useState<boolean>(false);

  const validate = async (c: any) => {
    try {
      const r = await fetch('/api/v1/config/validate', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(c)
      });
      if (r.ok) {
        setChecks(await r.json());
      }
    } catch {
      // ignore
    }
  };

  useEffect(() => {
    fetch('/api/v1/config')
      .then(async (r) => {
        const pathHdr = r.headers.get('X-Config-Path');
        if (pathHdr) {
          setConfigPath(pathHdr);
        }
        const c = await r.json();
        setCfg(c);
        validate(c);
      })
      .catch(() => setMsg({ text: 'Cannot load config from server', error: true }));
  }, []);

  if (!cfg) {
    return <div className="text-xs font-mono text-[#8B95A7]">{msg?.text || 'Loading configuration...'}</div>;
  }

  const setRole = (role: string, text: string) => {
    setCfg({
      ...cfg,
      roles: {
        ...cfg.roles,
        [role]: text.split('\n').map((s) => s.trim()).filter(Boolean)
      }
    });
  };

  const toggleVendorDistinct = (enabled: boolean) => {
    setCfg({
      ...cfg,
      policy: {
        ...cfg.policy,
        require_distinct_vendor: enabled ? ['primary', 'reviewer'] : ['primary', 'reviewer'] // must specify at least 2 roles per validator
      }
    });
  };

  const setMaxCost = (val: number) => {
    setCfg({
      ...cfg,
      policy: {
        ...cfg.policy,
        max_cost_usd_per_run: val >= 0 ? val : 0
      }
    });
  };

  const setTTL = (val: number) => {
    setCfg({
      ...cfg,
      policy: {
        ...cfg.policy,
        probe: {
          ...cfg.policy?.probe,
          ttl_minutes: val > 0 ? val : 30
        }
      }
    });
  };

  const save = async () => {
    setSaving(true);
    setMsg(null);
    try {
      const r = await fetch('/api/v1/config', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(cfg)
      });
      if (!r.ok) {
        const body = await r.json().catch(() => null);
        const errMsg = body?.error || r.statusText;
        setMsg({ text: `Save rejected: ${errMsg}`, error: true });
        return;
      }
      const saved = await r.json();
      setMsg({ text: `Configuration successfully saved to ${configPath}`, error: false });
      validate(saved);
      if (onConfigSaved) onConfigSaved(saved);
    } catch (err: any) {
      setMsg({ text: `Save error: ${err?.message || err}`, error: true });
    } finally {
      setSaving(false);
    }
  };

  const distinctVendorActive = (cfg.policy?.require_distinct_vendor?.length || 0) >= 2;

  return (
    <div className="space-y-4 text-xs font-mono">
      {/* File Path and Header */}
      <div className="flex items-center justify-between pb-2 border-b border-[#1E2638] text-[11px] text-[#8B95A7]">
        <div className="flex items-center gap-1.5">
          <FileText className="w-3.5 h-3.5 text-[#06B6D4]" />
          <span>Config file:</span>
          <code className="text-[#E6EAF2] bg-[#0A0D12] px-1.5 py-0.5 rounded border border-[#1E2638]">{configPath}</code>
        </div>
        <span>Schema Version: {cfg.version || 1}</span>
      </div>

      {/* Editable Role Candidates */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
        {ROLES.map((role) => (
          <div key={role} className="p-3 rounded-lg bg-[#121721] border border-[#1E2638] space-y-2">
            <div className="flex items-center justify-between">
              <span className="font-bold uppercase tracking-wider" style={{ color: accentColor }}>
                {role}
              </span>
              <span className="text-[10px] text-[#8B95A7]">One candidate per line (agent:model)</span>
            </div>
            <textarea
              rows={4}
              value={(cfg.roles?.[role] || []).join('\n')}
              onChange={(e) => setRole(role, e.target.value)}
              className="w-full bg-[#0A0D12] border border-[#1E2638] focus:border-[#E6EAF2] rounded p-2 text-[#E6EAF2] text-xs font-mono focus:outline-none"
              aria-label={`${role} candidates`}
              placeholder="e.g. claude:claude-3-5-sonnet"
            />
            {/* Live Verification Badges */}
            <div className="space-y-1 pt-1">
              {checks
                .filter((c) => c.role === role)
                .map((c, i) => (
                  <div key={i} className="flex items-start gap-1.5 text-[11px] leading-tight">
                    {c.verified ? (
                      <CheckCircle2 className="w-3.5 h-3.5 text-[#10B981] shrink-0 mt-0.5" aria-label="verified" />
                    ) : (
                      <AlertCircle className="w-3.5 h-3.5 text-[#F87171] shrink-0 mt-0.5" aria-label="not verified" />
                    )}
                    <span className="text-[#E6EAF2] font-semibold">{c.candidate}</span>
                    {!c.verified && <span className="text-[#F87171]">— {c.reason}</span>}
                  </div>
                ))}
            </div>
          </div>
        ))}
      </div>

      {/* Editable Policy Controls */}
      <div className="p-4 rounded-lg bg-[#121721] border border-[#1E2638] space-y-3">
        <span className="text-xs font-bold uppercase tracking-wider text-[#E6EAF2] block">
          Deterministic Policy Configuration
        </span>

        <div className="grid grid-cols-1 md:grid-cols-3 gap-3 text-xs">
          {/* Vendor Diversity */}
          <div className="p-3 rounded bg-[#0A0D12] border border-[#1E2638] space-y-1.5">
            <div className="flex items-center justify-between">
              <span className="text-[#E6EAF2] font-semibold flex items-center gap-1.5 text-[11px]">
                <Split className="w-3.5 h-3.5 text-[#06B6D4]" />
                require_distinct_vendor
              </span>
              <input
                type="checkbox"
                checked={distinctVendorActive}
                onChange={(e) => toggleVendorDistinct(e.target.checked)}
                className="rounded bg-[#0A0D12] border-[#1E2638] cursor-pointer"
                style={{ accentColor }}
              />
            </div>
            <p className="text-[#8B95A7] text-[10px] leading-relaxed">
              [primary, reviewer] must come from different vendors.
            </p>
          </div>

          {/* Max Cost Budget */}
          <div className="p-3 rounded bg-[#0A0D12] border border-[#1E2638] space-y-1.5">
            <div className="flex items-center justify-between">
              <span className="text-[#E6EAF2] font-semibold flex items-center gap-1.5 text-[11px]">
                <DollarSign className="w-3.5 h-3.5" style={{ color: accentColor }} />
                max_cost_usd_per_run
              </span>
            </div>
            <div className="flex items-center gap-1.5">
              <span className="text-[#8B95A7]">$</span>
              <input
                type="number"
                step="0.5"
                min="0"
                value={cfg.policy?.max_cost_usd_per_run ?? 5.0}
                onChange={(e) => setMaxCost(parseFloat(e.target.value) || 0)}
                className="w-full bg-[#121721] border border-[#1E2638] rounded px-2 py-1 text-xs text-[#E6EAF2] font-mono"
              />
            </div>
            <p className="text-[#8B95A7] text-[10px] leading-relaxed">
              Rejects candidates exceeding stage token cost threshold.
            </p>
          </div>

          {/* Probe TTL */}
          <div className="p-3 rounded bg-[#0A0D12] border border-[#1E2638] space-y-1.5">
            <div className="flex items-center justify-between">
              <span className="text-[#E6EAF2] font-semibold flex items-center gap-1.5 text-[11px]">
                <Clock className="w-3.5 h-3.5 text-[#F59E0B]" />
                probe.ttl_minutes
              </span>
            </div>
            <input
              type="number"
              step="5"
              min="1"
              value={cfg.policy?.probe?.ttl_minutes ?? 30}
              onChange={(e) => setTTL(parseFloat(e.target.value) || 30)}
              className="w-full bg-[#121721] border border-[#1E2638] rounded px-2 py-1 text-xs text-[#E6EAF2] font-mono"
            />
            <p className="text-[#8B95A7] text-[10px] leading-relaxed">
              Cache duration for healthy probe executions.
            </p>
          </div>
        </div>
      </div>

      {/* Action Toolbar & Feedback */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 pt-2">
        <div className="flex items-center gap-2">
          <button
            onClick={() => validate(cfg)}
            className="px-3.5 py-1.5 bg-[#0A0D12] hover:bg-[#1E2638] border border-[#1E2638] hover:border-[#8B95A7] rounded-lg text-[#E6EAF2] transition-colors cursor-pointer font-semibold"
          >
            Verify Candidates
          </button>
          <button
            onClick={save}
            disabled={saving}
            style={{ backgroundColor: accentColor }}
            className="px-4 py-1.5 rounded-lg text-[#0A0D12] hover:brightness-110 disabled:opacity-50 font-bold transition-all cursor-pointer flex items-center gap-1.5"
          >
            {saving ? 'Saving...' : 'Save Configuration'}
          </button>
        </div>

        {msg && (
          <div
            className={`px-3 py-1.5 rounded-lg text-xs font-mono border ${
              msg.error
                ? 'bg-[#EF4444]/10 border-[#EF4444]/30 text-[#F87171]'
                : 'bg-[#10B981]/10 border-[#10B981]/30 text-[#10B981]'
            }`}
          >
            {msg.text}
          </div>
        )}
      </div>
    </div>
  );
};
