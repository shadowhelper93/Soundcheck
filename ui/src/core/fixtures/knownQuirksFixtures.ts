export interface QuirkFixture {
  id: string;
  agentId: string;
  name: string;
  docIssue: string;
  category: 'false_exit_code' | 'silent_auth' | 'quota_lie' | 'protocol_mismatch';
  description: string;
  simulatedExitCode: number;
  simulatedStdout: string;
  simulatedStderr: string;
  naiveOutcome: string;
  soundcheckOutcome: string;
  expectedClass: string;
  explanation: string;
}

export const KNOWN_QUIRKS_FIXTURES: QuirkFixture[] = [
  {
    id: 'claude-79500',
    agentId: 'claude',
    name: 'Claude Code Rate Limit False Exit 0',
    docIssue: 'Issue #79500 (anthropics/claude-code)',
    category: 'false_exit_code',
    description: 'API rate limits return subtype:"success" with is_error:true and exit code 0! Naive runners think the turn completed successfully.',
    simulatedExitCode: 0,
    simulatedStdout: JSON.stringify({
      type: "result",
      subtype: "success",
      is_error: true,
      error: {
        type: "rate_limit_error",
        message: "You have exceeded your rolling 5-hour quota allowance. Retry in 42 minutes.",
        resets_at: Math.floor(Date.now() / 1000) + 2520
      },
      total_cost_usd: 0.00
    }, null, 2),
    simulatedStderr: '',
    naiveOutcome: 'EXIT 0 marked SUCCESS. Worktree merged with broken/empty files.',
    soundcheckOutcome: 'FLAGGED AS RATE_LIMITED. Enters LiteLLM-style cooldown (42m) and shifts role to backup.',
    expectedClass: 'rate_limited',
    explanation: 'Soundcheck looks into parsed JSON `is_error` and inspects `error.type` rather than trusting the exit code 0.'
  },
  {
    id: 'claude-74761',
    agentId: 'claude',
    name: 'Claude Code Mid-Task Exit 0',
    docIssue: 'Issue #74761 (anthropics/claude-code)',
    category: 'false_exit_code',
    description: 'Headless claude -p exits 0 while loop is mid-task (last assistant message has stop_reason: tool_use).',
    simulatedExitCode: 0,
    simulatedStdout: JSON.stringify({
      type: "assistant",
      stop_reason: "tool_use",
      content: [
        { type: "text", text: "I will now edit src/index.ts..." },
        { type: "tool_use", id: "call_abc123", name: "replace_file_content", input: {} }
      ]
    }, null, 2),
    simulatedStderr: '',
    naiveOutcome: 'EXIT 0 marked SUCCESS. Task abandoned mid-execution with partial state.',
    soundcheckOutcome: 'FLAGGED AS CORRUPTED_STATE. Detected unfinished tool_use stop_reason with missing result turn.',
    expectedClass: 'corrupted_state',
    explanation: 'Soundcheck validates that headless runs terminate with stop_reason end_turn or a verified diff.'
  },
  {
    id: 'codex-doctor-exit0',
    agentId: 'codex',
    name: 'Codex Doctor Exit 0 On Auth Failure',
    docIssue: 'OpenAI Codex Developer Bug #54831 / #22336',
    category: 'silent_auth',
    description: 'codex doctor --json outputs a failed report with redacted token details but exits with code 0.',
    simulatedExitCode: 0,
    simulatedStdout: JSON.stringify({
      version: "0.9.4",
      doctor_status: "FAILED",
      authentication: {
        status: "invalid_credentials",
        token_redacted: "sk-proj-...X9qA",
        expired: true,
        message: "OAuth token revoked or expired"
      },
      sandbox: { write_access: false }
    }, null, 2),
    simulatedStderr: '',
    naiveOutcome: 'EXIT 0 marked AUTH OK. Next execution crashes immediately with unhandled reject.',
    soundcheckOutcome: 'FLAGGED AS UNAUTHENTICATED. Gates on doctor_status and authentication.status, not exit code.',
    expectedClass: 'unauthenticated',
    explanation: 'Soundcheck explicitly inspects internal JSON structure of doctor reports.'
  },
  {
    id: 'antigravity-no-auth-cmd',
    agentId: 'agy',
    name: 'Antigravity CLI Missing Auth Command & Silent Hang',
    docIssue: 'Google Antigravity CLI June 2026 Transition',
    category: 'silent_auth',
    description: 'Antigravity has no auth-status command. Headless runs without cached credentials exit with authentication required.',
    simulatedExitCode: 1,
    simulatedStdout: '',
    simulatedStderr: '[antigravity] error: authentication required. User must run agy auth login in interactive terminal.',
    naiveOutcome: 'Generic failure exit 1. Runner repeatedly loops retrying with no resolution.',
    soundcheckOutcome: 'FLAGGED AS UNAUTHENTICATED. Distinguishes auth error from runtime error, directs user to login.',
    expectedClass: 'unauthenticated',
    explanation: 'Soundcheck recognizes agy auth error signatures in stderr and prevents infinite polling.'
  },
  {
    id: 'gemini-base-url-exit41',
    agentId: 'gemini',
    name: 'Gemini CLI Base URL Exit 41 False Auth Failure',
    docIssue: 'GitHub Pull Request #8916 / Issue #8022',
    category: 'protocol_mismatch',
    description: 'Setting GOOGLE_GEMINI_BASE_URL erroneously triggers exit 41 (FatalAuthenticationError) even with a valid key.',
    simulatedExitCode: 41,
    simulatedStdout: '',
    simulatedStderr: 'FatalAuthenticationError: exit 41 - custom endpoint refused handshake on GOOGLE_GEMINI_BASE_URL',
    naiveOutcome: 'User told their API key is invalid and token is revoked.',
    soundcheckOutcome: 'FLAGGED AS PROTOCOL_MISMATCH. Suggests clearing GOOGLE_GEMINI_BASE_URL or migrating to Antigravity.',
    expectedClass: 'unauthenticated',
    explanation: 'Soundcheck isolates the base URL bug from legitimate API key expiration.'
  },
  {
    id: 'claude-clean-success',
    agentId: 'claude',
    name: 'Claude Code Legitimate Clean Headless Turn',
    docIssue: 'Anthropic Reference v2.1.302',
    category: 'false_exit_code',
    description: 'Legitimate turn completion with exit code 0, is_error: false, and zero permission denials.',
    simulatedExitCode: 0,
    simulatedStdout: JSON.stringify({
      type: "result",
      subtype: "success",
      is_error: false,
      total_cost_usd: 0.038,
      duration_ms: 4120,
      permission_denials: [],
      files_modified: ["src/index.ts"]
    }, null, 2),
    simulatedStderr: '',
    naiveOutcome: 'EXIT 0 marked SUCCESS.',
    soundcheckOutcome: 'VERIFIED HEALTHY. Cost recorded ($0.038), zero permission denials validated.',
    expectedClass: 'ok',
    explanation: 'All verification checks passed: is_error is false, permission_denials is empty, and output contains artifacts.'
  }
];
