---
name: soundcheck
description: >-
  Preflight health check, role resolution, quirk immunity, and automatic failover for AI coding agents
  (Claude Code, OpenAI Codex, OpenCode, Antigravity CLI, Gemini CLI, Copilot CLI). Use before executing
  prompts or dispatching sub-agents to verify agent health, bypass rate limits and false exit codes,
  discover verified fallback models, or run tasks with automatic cascade failover.
---

# Soundcheck: Agent Immunity & Preflight Execution

Soundcheck is a lightweight, zero-dependency preflight engine and failover runner for AI coding agents. It prevents silent failures, catches false exit code lies (e.g. rate limits exiting with code 0), and ensures tasks run on verified, working models without manual babysitting or harness lock-in.

---

## 1. Quick Decision Matrix: How to Use Soundcheck

| Goal | Command / Action | Why |
|---|---|---|
| **Health Check** | `soundcheck doctor --json` | Probe all installed agents (L0 existence, L1 credentials, L2 latency, L3 model). |
| **Failover Run** | `soundcheck exec "<prompt>"` | Executes prompt with automatic retry down the fallback chain if rate-limited or broken. |
| **Role Failover** | `soundcheck exec --role primary "<prompt>"` | Executes through the resolved primary agent; falls back automatically. |
| **Specific Agent** | `soundcheck exec --adapter codex "<prompt>"` | Runs directly through specified agent with quirk detection and classification. |
| **Role Resolution** | `soundcheck resolve --json` | Find which agent/model is currently healthy and assigned to `primary`, `reviewer`, etc. |
| **Find Fallbacks** | `soundcheck check <provider>` | Discover ready-to-use alternative models on this host for a specific provider. |
| **Fix Adapter** | `soundcheck fix <adapter> --json` | Get deterministic remediation instructions and commands for unauthenticated agents. |

---

## 2. Using in Orca Orchestration (Dispatched Workers & Helpers)

When working as an Orca coordinator or dispatched worker, you do **not** need complex sub-agent harnesses or fragile multi-agent frameworks to execute auxiliary tasks:

### Running Auxiliary or Review Tasks via Soundcheck
Instead of spawning heavy nested worker trees when you just need code generation, review, or alternative checks:
```bash
# Run review through healthy reviewer agent with auto-failover
soundcheck exec --role reviewer "Review the git diff in current branch for security and edge cases"

# Run fast auxiliary query
soundcheck exec --role auxiliary "Generate TypeScript types for the following schema: ..."
```

### Self-Contained Execution (Zero Harness Overhead)
If an agent or model is rate-limited or fails authentication mid-turn, `soundcheck exec` catches the quirk, flags the failure, and automatically reroutes to the next working adapter:
```json
// Example: Output from soundcheck exec --json
{
  "prompt": "Say hello",
  "agentUsed": "codex",
  "success": true,
  "failoverOccurred": true,
  "attempts": [
    { "agentId": "claude", "success": false, "errorMsg": "rate_limited" },
    { "agentId": "codex", "success": true }
  ],
  "durationMs": 1420
}
```

---

## 3. Quirk Protection (Immunity from Agent Lies)

Many agent CLIs exhibit known failure modes that deceive standard process runners:
- **Claude Code (#79500)**: Returns `is_error: true` with exit code `0` on rate limits. Soundcheck intercepts this and rewrites to exit code `429`.
- **Codex Doctor (#54831)**: Exits `0` with `FAILED` auth status. Soundcheck rewrites to exit code `401`.
- **Gemini CLI (#41)**: Exits `41` on custom base URLs. Soundcheck normalizes to `429` with cooldown.
- **Copilot CLI**: Silent permission denials leave empty git diffs while exiting `0`. Soundcheck enforces deterministic gates.

---

## 4. MCP Server Integration

Soundcheck runs as a stdio Model Context Protocol (MCP) server exposing native tools to Claude Desktop, Cursor, Zed, or any MCP client:

```bash
soundcheck mcp
```

### Available MCP Tools:
- `soundcheck_doctor`: Diagnostic health checks across CLI adapters and capabilities (`deep: boolean`).
- `soundcheck_probe`: Probe a specific CLI adapter (`adapter: string`).
- `soundcheck_resolve`: Resolve assigned agent/model for a role or task (`role: string`, `task: string`).
- `soundcheck_manifest`: Get complete JSON capability manifest of available CLI tools and models.
