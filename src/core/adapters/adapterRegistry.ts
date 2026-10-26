import { AgentAdapter, ErrorClass, ProbeResult } from '../../types/soundcheck';

export function createDefaultProbe(level: 'L0' | 'L1' | 'L2' | 'L3', message = 'Not run yet'): ProbeResult {
  return {
    level,
    timestamp: 0,
    cachedUntil: 0,
    latencyMs: 0,
    status: 'pending',
    errorClass: 'ok',
    message
  };
}

export const INITIAL_ADAPTERS: AgentAdapter[] = [
  {
    id: 'claude',
    name: 'Claude Code',
    vendor: 'Anthropic',
    binary: 'claude',
    category: 'platform',
    versionSupported: '>=2.1.280',
    installed: true,
    versionDetected: '2.1.302 (October 2026)',
    authMethodSupported: ['OAuth subscription', 'ANTHROPIC_API_KEY'],
    headlessCommand: 'claude -p --output-format json',
    permissionModes: ['--permission-mode', '--allowedTools', '--bare'],
    allowedFails: 2,
    consecutiveFails: 0,
    cooldownUntil: undefined,
    quirks: [
      {
        id: 'issue-79500',
        title: 'Rate Limit Returns Exit 0 With subtype: "success"',
        description: 'Headless claude -p returns subtype: "success" and exit 0 even when rolling 5-hour quota is exceeded. Error only visible in JSON is_error.',
        referenceUrl: 'https://github.com/anthropics/claude-code/issues/79500',
        discoveredDate: '2026-06-12',
        severity: 'critical',
        impact: 'Silent workflow pipeline corruption, empty files marked as completed',
        resolution: 'Inspect JSON is_error and error.type, ignore shell exit code'
      },
      {
        id: 'issue-74761',
        title: '-p Exits 0 Mid-Task On Tool Invocation',
        description: 'Headless invocation exits 0 prematurely when stop_reason is tool_use rather than waiting for results.',
        referenceUrl: 'https://github.com/anthropics/claude-code/issues/74761',
        discoveredDate: '2026-05-18',
        severity: 'high',
        impact: 'Partial file edits left unstaged or uncommitted',
        resolution: 'Check stream-json end_turn event and diff_nonempty gate'
      },
      {
        id: 'issue-95822',
        title: 'Auth Check Polling Exhausts Refresh Tokens',
        description: 'claude auth status can start OAuth token refresh and exit without saving, invalidating future session tokens.',
        referenceUrl: 'https://github.com/anthropics/claude-code/issues/95822',
        discoveredDate: '2026-08-04',
        severity: 'medium',
        impact: 'Token revoked errors when checking health in tight loop',
        resolution: 'Strict L1 cache TTL (minimum 60 minutes) prevents polling'
      }
    ],
    models: [
      {
        id: 'claude-3-7-sonnet',
        displayName: 'Claude 3.7 Sonnet',
        vendor: 'Anthropic',
        tier: 'frontier',
        contextWindow: 200000,
        inputCostPerM: 3.0,
        outputCostPerM: 15.0,
        verified: true
      },
      {
        id: 'claude-3-5-haiku',
        displayName: 'Claude 3.5 Haiku',
        vendor: 'Anthropic',
        tier: 'fast',
        contextWindow: 200000,
        inputCostPerM: 0.8,
        outputCostPerM: 4.0,
        verified: true
      },
      {
        id: 'claude-opus',
        displayName: 'Claude 3 Opus',
        vendor: 'Anthropic',
        tier: 'reasoning',
        contextWindow: 200000,
        inputCostPerM: 15.0,
        outputCostPerM: 75.0,
        verified: true
      }
    ],
    probes: {
      L0: {
        level: 'L0',
        timestamp: Date.now() - 3600000,
        cachedUntil: Date.now() + 82800000,
        latencyMs: 14,
        status: 'passed',
        errorClass: 'ok',
        message: 'Binary found at /usr/local/bin/claude (v2.1.302)'
      },
      L1: {
        level: 'L1',
        timestamp: Date.now() - 1800000,
        cachedUntil: Date.now() + 1800000,
        latencyMs: 120,
        status: 'passed',
        errorClass: 'ok',
        message: 'OAuth active: Pro subscription, loggedIn: true'
      },
      L2: {
        level: 'L2',
        timestamp: Date.now() - 600000,
        cachedUntil: Date.now() + 1200000,
        latencyMs: 1420,
        status: 'passed',
        errorClass: 'ok',
        message: 'Smoke test passed: {"ok": true} returned in 1.42s'
      },
      L3: {
        level: 'L3',
        timestamp: Date.now() - 600000,
        cachedUntil: Date.now() + 1200000,
        latencyMs: 1650,
        status: 'passed',
        errorClass: 'ok',
        message: 'Model sonnet validated with live turn'
      }
    }
  },
  {
    id: 'codex',
    name: 'OpenAI Codex',
    vendor: 'OpenAI',
    binary: 'codex',
    category: 'platform',
    versionSupported: '>=0.9.0',
    installed: true,
    versionDetected: '0.9.4',
    authMethodSupported: ['OAuth login', 'CODEX_API_KEY', 'OPENAI_API_KEY'],
    headlessCommand: 'codex exec --json --sandbox workspace-write',
    permissionModes: ['--sandbox read-only', '--sandbox workspace-write'],
    allowedFails: 2,
    consecutiveFails: 0,
    cooldownUntil: undefined,
    quirks: [
      {
        id: 'codex-doctor-exit0',
        title: 'codex doctor --json Exits 0 Despite Auth Failure',
        description: 'Diagnostics command returns exit 0 even when credentials are dead or expired. Must parse stdout JSON rather than exit code.',
        referenceUrl: 'https://github.com/openai/codex/issues/54831',
        discoveredDate: '2026-04-11',
        severity: 'critical',
        impact: 'False green status in CI runners followed by hard crash',
        resolution: 'Parse JSON doctor report; reject if status is FAILED'
      },
      {
        id: 'codex-issue-10233',
        title: 'No Headless Quota Inspection API',
        description: 'Issue #10233: OpenAI exposes no non-interactive quota command. Quota state can only be inferred reactively from HTTP 429 errors.',
        referenceUrl: 'https://github.com/openai/codex/issues/10233',
        discoveredDate: '2026-03-02',
        severity: 'high',
        impact: 'Cannot pre-check usage balance before expensive runs',
        resolution: 'Implement LiteLLM cooldown tracker (allowed_fails + cooldown_time)'
      }
    ],
    models: [
      {
        id: 'gpt-5-codex',
        displayName: 'GPT-5.2 Codex',
        vendor: 'OpenAI',
        tier: 'frontier',
        contextWindow: 256000,
        inputCostPerM: 2.5,
        outputCostPerM: 10.0,
        verified: true
      },
      {
        id: 'gpt-5-mini',
        displayName: 'GPT-5 Mini',
        vendor: 'OpenAI',
        tier: 'fast',
        contextWindow: 128000,
        inputCostPerM: 0.3,
        outputCostPerM: 1.2,
        verified: true
      }
    ],
    probes: {
      L0: {
        level: 'L0',
        timestamp: Date.now() - 3600000,
        cachedUntil: Date.now() + 82800000,
        latencyMs: 18,
        status: 'passed',
        errorClass: 'ok',
        message: 'Binary found at /usr/local/bin/codex (v0.9.4)'
      },
      L1: {
        level: 'L1',
        timestamp: Date.now() - 1800000,
        cachedUntil: Date.now() + 1800000,
        latencyMs: 95,
        status: 'passed',
        errorClass: 'ok',
        message: 'codex login status verified (exit 0, valid session)'
      },
      L2: {
        level: 'L2',
        timestamp: Date.now() - 600000,
        cachedUntil: Date.now() + 1200000,
        latencyMs: 1890,
        status: 'passed',
        errorClass: 'ok',
        message: 'Smoke test passed in read-only sandbox mode'
      },
      L3: {
        level: 'L3',
        timestamp: Date.now() - 600000,
        cachedUntil: Date.now() + 1200000,
        latencyMs: 2100,
        status: 'passed',
        errorClass: 'ok',
        message: 'gpt-5-codex verified'
      }
    }
  },
  {
    id: 'opencode',
    name: 'OpenCode',
    vendor: 'OpenCode Project',
    binary: 'opencode',
    category: 'opensource',
    versionSupported: '>=2.0.0',
    installed: true,
    versionDetected: '2.4.1',
    authMethodSupported: ['opencode auth login', 'provider keys'],
    headlessCommand: 'opencode run --format json --auto',
    permissionModes: ['opencode.json permission block', '--auto'],
    allowedFails: 3,
    consecutiveFails: 0,
    cooldownUntil: undefined,
    quirks: [
      {
        id: 'opencode-v1-v2',
        title: 'Auth Command Renamed Between v1 and v2',
        description: 'opencode providers was renamed to opencode auth list --format json in v2.0.',
        referenceUrl: 'https://opencode.ai/v2/docs/cli/commands/',
        discoveredDate: '2026-02-14',
        severity: 'medium',
        impact: 'Subprocess crashes with unknown command on older installs',
        resolution: 'Probe binary version first; dispatch command based on semver'
      }
    ],
    models: [
      {
        id: 'anthropic/claude-sonnet',
        displayName: 'OpenCode Sonnet Gateway',
        vendor: 'Anthropic',
        tier: 'frontier',
        contextWindow: 200000,
        inputCostPerM: 3.0,
        outputCostPerM: 15.0,
        verified: true
      },
      {
        id: 'openai/gpt-5.x-mini',
        displayName: 'OpenCode GPT-5 Mini',
        vendor: 'OpenAI',
        tier: 'fast',
        contextWindow: 128000,
        inputCostPerM: 0.3,
        outputCostPerM: 1.2,
        verified: true
      }
    ],
    probes: {
      L0: {
        level: 'L0',
        timestamp: Date.now() - 3600000,
        cachedUntil: Date.now() + 82800000,
        latencyMs: 12,
        status: 'passed',
        errorClass: 'ok',
        message: 'Binary found at /usr/bin/opencode (v2.4.1)'
      },
      L1: {
        level: 'L1',
        timestamp: Date.now() - 1800000,
        cachedUntil: Date.now() + 1800000,
        latencyMs: 80,
        status: 'passed',
        errorClass: 'ok',
        message: '2 providers active via opencode auth list'
      },
      L2: {
        level: 'L2',
        timestamp: Date.now() - 600000,
        cachedUntil: Date.now() + 1200000,
        latencyMs: 1250,
        status: 'passed',
        errorClass: 'ok',
        message: 'NDJSON stream completed with step_finish'
      },
      L3: {
        level: 'L3',
        timestamp: Date.now() - 600000,
        cachedUntil: Date.now() + 1200000,
        latencyMs: 1300,
        status: 'passed',
        errorClass: 'ok',
        message: 'First-class opencode models check confirmed'
      }
    }
  },
  {
    id: 'agy',
    name: 'Antigravity CLI',
    vendor: 'Google',
    binary: 'agy',
    category: 'platform',
    versionSupported: '>=1.0.0',
    installed: true,
    versionDetected: '1.2.0 (Go runtime)',
    authMethodSupported: ['Google Cloud OAuth', 'GEMINI_API_KEY'],
    headlessCommand: 'agy -p --output-format json',
    permissionModes: ['--json-schema', '--approval-mode deny'],
    allowedFails: 2,
    consecutiveFails: 0,
    cooldownUntil: undefined,
    quirks: [
      {
        id: 'agy-no-auth-cmd',
        title: 'Antigravity Has No Non-Interactive Auth Command',
        description: 'Google replaced Gemini CLI with Antigravity on June 18, 2026. agy provides no auth-status command; unauthenticated headless runs exit with authentication required.',
        referenceUrl: 'https://antigravity.google/docs/cli/headless/',
        discoveredDate: '2026-06-18',
        severity: 'critical',
        impact: 'Silent lockups or unhandled process exits',
        resolution: 'Issue micro headless probe; check for .status == SUCCESS (WAITING is not failure)'
      }
    ],
    models: [
      {
        id: 'gemini-2.5-flash',
        displayName: 'Gemini 2.5 Flash',
        vendor: 'Google',
        tier: 'fast',
        contextWindow: 1000000,
        inputCostPerM: 0.15,
        outputCostPerM: 0.60,
        verified: true
      },
      {
        id: 'gemini-2.5-pro',
        displayName: 'Gemini 2.5 Pro',
        vendor: 'Google',
        tier: 'frontier',
        contextWindow: 2000000,
        inputCostPerM: 1.25,
        outputCostPerM: 5.0,
        verified: true
      }
    ],
    probes: {
      L0: {
        level: 'L0',
        timestamp: Date.now() - 3600000,
        cachedUntil: Date.now() + 82800000,
        latencyMs: 8,
        status: 'passed',
        errorClass: 'ok',
        message: 'Binary found at /usr/local/bin/agy (Go static binary v1.2.0)'
      },
      L1: {
        level: 'L1',
        timestamp: Date.now() - 1800000,
        cachedUntil: Date.now() + 1800000,
        latencyMs: 140,
        status: 'passed',
        errorClass: 'ok',
        message: 'Live probe check returned status: SUCCESS'
      },
      L2: {
        level: 'L2',
        timestamp: Date.now() - 600000,
        cachedUntil: Date.now() + 1200000,
        latencyMs: 980,
        status: 'passed',
        errorClass: 'ok',
        message: 'Micro-turn validated in 980ms'
      },
      L3: {
        level: 'L3',
        timestamp: Date.now() - 600000,
        cachedUntil: Date.now() + 1200000,
        latencyMs: 1100,
        status: 'passed',
        errorClass: 'ok',
        message: 'gemini-2.5-flash verified'
      }
    }
  },
  {
    id: 'gemini',
    name: 'Gemini CLI (Legacy)',
    vendor: 'Google Enterprise',
    binary: 'gemini',
    category: 'platform',
    versionSupported: '<=0.9.x',
    installed: false,
    versionDetected: undefined,
    authMethodSupported: ['Paid API Key', 'Vertex AI Enterprise'],
    headlessCommand: 'gemini -p --approval-mode deny',
    permissionModes: ['--approval-mode', 'policy engine'],
    allowedFails: 2,
    consecutiveFails: 0,
    cooldownUntil: undefined,
    quirks: [
      {
        id: 'gemini-exit-41',
        title: 'Exit Code 41 (FatalAuthenticationError) Bug',
        description: 'Replaced for individual users June 2026. Known bug triggers exit 41 whenever GOOGLE_GEMINI_BASE_URL is defined.',
        referenceUrl: 'https://google-gemini.github.io/gemini-cli/docs/troubleshooting.html',
        discoveredDate: '2026-05-01',
        severity: 'high',
        impact: 'Unusable behind enterprise proxy setups',
        resolution: 'Normalize exit 41 into protocol_mismatch / migrate to Antigravity'
      }
    ],
    models: [
      {
        id: 'gemini-1.5-pro',
        displayName: 'Gemini 1.5 Pro (Legacy)',
        vendor: 'Google',
        tier: 'mid',
        contextWindow: 1000000,
        inputCostPerM: 2.5,
        outputCostPerM: 10.0,
        verified: false
      }
    ],
    probes: {
      L0: {
        level: 'L0',
        timestamp: 0,
        cachedUntil: 0,
        latencyMs: 0,
        status: 'warning',
        errorClass: 'not_installed',
        message: 'Not in PATH (Migrated to Antigravity CLI June 18, 2026)'
      },
      L1: createDefaultProbe('L1', 'Skipped: CLI not installed'),
      L2: createDefaultProbe('L2', 'Skipped'),
      L3: createDefaultProbe('L3', 'Skipped')
    }
  },
  {
    id: 'copilot',
    name: 'GitHub Copilot CLI',
    vendor: 'GitHub / Microsoft',
    binary: 'copilot',
    category: 'platform',
    versionSupported: '>=1.1.0',
    installed: true,
    versionDetected: '1.3.4',
    authMethodSupported: ['gh auth login', 'GITHUB_TOKEN'],
    headlessCommand: 'copilot -p --output-format json -s',
    permissionModes: ['--allow-tool', '--deny-tool'],
    allowedFails: 2,
    consecutiveFails: 0,
    cooldownUntil: undefined,
    quirks: [
      {
        id: 'copilot-tool-denials',
        title: 'Silent Permission Denials Yield Empty Diffs',
        description: 'When tool calls are blocked without explicit --allow-tool flags, Copilot exits 0 with no error message and an empty git diff.',
        referenceUrl: 'https://github.com/github/copilot-cli',
        discoveredDate: '2026-07-20',
        severity: 'high',
        impact: 'Runner assumes task completed, but code remains unchanged',
        resolution: 'Enforce diff_nonempty deterministic gate'
      }
    ],
    models: [
      {
        id: 'gpt-4o-copilot',
        displayName: 'Copilot GPT-4o',
        vendor: 'OpenAI / GitHub',
        tier: 'mid',
        contextWindow: 128000,
        inputCostPerM: 2.5,
        outputCostPerM: 10.0,
        verified: true
      }
    ],
    probes: {
      L0: {
        level: 'L0',
        timestamp: Date.now() - 3600000,
        cachedUntil: Date.now() + 82800000,
        latencyMs: 15,
        status: 'passed',
        errorClass: 'ok',
        message: 'Binary found at /usr/bin/copilot (v1.3.4)'
      },
      L1: {
        level: 'L1',
        timestamp: Date.now() - 1800000,
        cachedUntil: Date.now() + 1800000,
        latencyMs: 110,
        status: 'passed',
        errorClass: 'ok',
        message: 'GitHub OAuth token authenticated'
      },
      L2: {
        level: 'L2',
        timestamp: Date.now() - 600000,
        cachedUntil: Date.now() + 1200000,
        latencyMs: 1520,
        status: 'passed',
        errorClass: 'ok',
        message: 'Headless turn passed'
      },
      L3: {
        level: 'L3',
        timestamp: Date.now() - 600000,
        cachedUntil: Date.now() + 1200000,
        latencyMs: 1600,
        status: 'passed',
        errorClass: 'ok',
        message: 'Default model verified'
      }
    }
  },
  {
    id: 'generic-acp',
    name: 'Generic ACP Agent',
    vendor: 'Agent Client Protocol (JetBrains/Zed)',
    binary: 'acp-runner',
    category: 'protocol',
    versionSupported: 'ACP 2026.1 (42 agents)',
    installed: true,
    versionDetected: 'ACP stdio bridge',
    authMethodSupported: ['ACP initialize handshake', 'authMethods negotiation'],
    headlessCommand: 'acp-client --agent <name> --stdio',
    permissionModes: ['agentCapabilities declaration', 'read-only'],
    allowedFails: 2,
    consecutiveFails: 0,
    cooldownUntil: undefined,
    quirks: [
      {
        id: 'acp-discovery',
        title: 'Single Code-Path Capability Discovery',
        description: 'Standardized JSON-RPC over stdio. initialize returns agentCapabilities and authMethods for all 42 ACP registry agents.',
        referenceUrl: 'https://grokipedia.com/page/Agent_Client_Protocol',
        discoveredDate: '2026-01-28',
        severity: 'medium',
        impact: 'High compatibility without vendor-specific CLI hacks',
        resolution: 'Negotiate schema in initialize payload'
      }
    ],
    models: [
      {
        id: 'acp/dynamic',
        displayName: 'ACP Negotiated Model',
        vendor: 'ACP Dynamic',
        tier: 'frontier',
        contextWindow: 128000,
        inputCostPerM: 1.0,
        outputCostPerM: 3.0,
        verified: true
      }
    ],
    probes: {
      L0: {
        level: 'L0',
        timestamp: Date.now() - 3600000,
        cachedUntil: Date.now() + 82800000,
        latencyMs: 5,
        status: 'passed',
        errorClass: 'ok',
        message: 'ACP stdio bridge ready (42 registry agents)'
      },
      L1: {
        level: 'L1',
        timestamp: Date.now() - 1800000,
        cachedUntil: Date.now() + 1800000,
        latencyMs: 45,
        status: 'passed',
        errorClass: 'ok',
        message: 'Capabilities & authMethods received via initialize'
      },
      L2: {
        level: 'L2',
        timestamp: Date.now() - 600000,
        cachedUntil: Date.now() + 1200000,
        latencyMs: 780,
        status: 'passed',
        errorClass: 'ok',
        message: 'JSON-RPC response validated'
      },
      L3: {
        level: 'L3',
        timestamp: Date.now() - 600000,
        cachedUntil: Date.now() + 1200000,
        latencyMs: 820,
        status: 'passed',
        errorClass: 'ok',
        message: 'ACP capability declared'
      }
    }
  }
];

/**
 * Universal Error Classifier:
 * Implements the core thesis of the paper:
 * "Put all of the per-CLI lies and quirks in classifyError.
 * For Claude, trust is_error and prefer stream-json over exit code.
 * For agy, check status. For Gemini, map exit 41."
 */
export function classifyError(
  adapterId: string,
  stdout: string,
  stderr: string,
  exitCode: number,
  parsedJson?: any
): { errorClass: ErrorClass; reason: string; caughtLie: boolean } {
  // 1. Claude Code specific classifier
  if (adapterId === 'claude') {
    // Check for Issue #79500: Exit 0 with rate limit error inside JSON
    if (parsedJson && parsedJson.is_error) {
      const errType = parsedJson.error?.type || '';
      const errMsg = parsedJson.error?.message || '';
      if (errType.includes('rate_limit') || errMsg.toLowerCase().includes('quota') || errMsg.includes('5-hour')) {
        return {
          errorClass: 'rate_limited',
          reason: `Detected Issue #79500: Claude returned exit code 0 ("success"), but JSON contains rate_limit_error: "${errMsg}"`,
          caughtLie: exitCode === 0
        };
      }
      return {
        errorClass: 'unauthenticated',
        reason: `Claude returned exit 0 with is_error: true - ${errMsg}`,
        caughtLie: exitCode === 0
      };
    }

    // Check for Issue #74761: exit 0 mid-task with stop_reason: tool_use
    if (parsedJson && parsedJson.stop_reason === 'tool_use' && !parsedJson.result) {
      return {
        errorClass: 'corrupted_state',
        reason: 'Detected Issue #74761: claude -p exited 0 mid-task with stop_reason "tool_use" before tool completion.',
        caughtLie: exitCode === 0
      };
    }

    // Check stdout for text-mode rate limit errors
    if (stdout.includes('You have exceeded your 5-hour quota') || stdout.includes('rate_limit')) {
      return {
        errorClass: 'rate_limited',
        reason: 'Claude stdout contains 5-hour rolling quota limit warning.',
        caughtLie: exitCode === 0
      };
    }

    // Issue #103253: Late rate limit event causes exit 1 with empty stdout
    if (exitCode === 1 && !stdout.trim() && stderr.includes('rate_limit_event')) {
      return {
        errorClass: 'rate_limited',
        reason: 'Detected Issue #103253: Late rate_limit_event caused exit 1 with empty stdout in JSON mode.',
        caughtLie: false
      };
    }

    if (exitCode !== 0) {
      if (stderr.toLowerCase().includes('auth') || stderr.includes('login') || stderr.includes('token')) {
        return { errorClass: 'unauthenticated', reason: 'Claude authentication error or expired OAuth token.', caughtLie: false };
      }
      return { errorClass: 'corrupted_state', reason: `Claude execution failed with exit code ${exitCode}`, caughtLie: false };
    }

    return { errorClass: 'ok', reason: 'Claude turn verified cleanly (is_error: false).', caughtLie: false };
  }

  // 2. OpenAI Codex classifier
  if (adapterId === 'codex') {
    // Check doctor bug: exit 0 with FAILED status
    if (parsedJson && (parsedJson.doctor_status === 'FAILED' || parsedJson.authentication?.status === 'invalid_credentials')) {
      return {
        errorClass: 'unauthenticated',
        reason: 'Detected Codex Doctor Bug: codex doctor exited 0, but internal JSON reports FAILED auth.',
        caughtLie: exitCode === 0
      };
    }

    if (stdout.includes('429 Too Many Requests') || stdout.toLowerCase().includes('rate limit')) {
      return {
        errorClass: 'rate_limited',
        reason: 'OpenAI Codex HTTP 429 rate limit reached (reactive detection).',
        caughtLie: exitCode === 0
      };
    }

    if (exitCode !== 0) {
      return { errorClass: 'unauthenticated', reason: `Codex exited with code ${exitCode}. Check CODEX_API_KEY.`, caughtLie: false };
    }

    return { errorClass: 'ok', reason: 'Codex verified successfully.', caughtLie: false };
  }

  // 3. Antigravity (agy) classifier
  if (adapterId === 'agy') {
    if (stderr.includes('authentication required') || stderr.includes('agy auth login')) {
      return {
        errorClass: 'unauthenticated',
        reason: 'Antigravity CLI requires interactive login (no non-interactive auth command).',
        caughtLie: false
      };
    }

    if (parsedJson && parsedJson.status === 'SUCCESS') {
      return { errorClass: 'ok', reason: 'Antigravity micro-probe passed with status SUCCESS.', caughtLie: false };
    }

    if (parsedJson && parsedJson.status === 'WAITING') {
      return { errorClass: 'ok', reason: 'Antigravity in WAITING status (asynchronous step processing, not failure).', caughtLie: false };
    }

    if (exitCode !== 0) {
      return { errorClass: 'unauthenticated', reason: `Antigravity probe exited with code ${exitCode}`, caughtLie: false };
    }

    return { errorClass: 'ok', reason: 'Antigravity verified.', caughtLie: false };
  }

  // 4. Gemini CLI (Legacy) classifier
  if (adapterId === 'gemini') {
    if (exitCode === 41 || stderr.includes('FatalAuthenticationError')) {
      return {
        errorClass: 'unauthenticated',
        reason: 'Gemini CLI exit 41 (FatalAuthenticationError). Note: Setting GOOGLE_GEMINI_BASE_URL triggers this bug even with a valid key.',
        caughtLie: true
      };
    }
  }

  // Generic fallback classification
  if (exitCode === 0) {
    return { errorClass: 'ok', reason: 'Normal clean execution.', caughtLie: false };
  }
  if (stderr.toLowerCase().includes('auth') || stderr.toLowerCase().includes('unauthorized') || exitCode === 401) {
    return { errorClass: 'unauthenticated', reason: 'Authentication failed.', caughtLie: false };
  }
  if (stderr.includes('429') || stderr.toLowerCase().includes('rate limit')) {
    return { errorClass: 'rate_limited', reason: 'Rate limit / quota exceeded.', caughtLie: false };
  }

  return { errorClass: 'corrupted_state', reason: `Command exited with non-zero code ${exitCode}`, caughtLie: false };
}
