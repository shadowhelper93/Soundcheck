# Soundcheck

> **Deterministic Preflight Prober & Role Resolver for AI Coding Harnesses**

[![CI](https://github.com/soundcheck/soundcheck/actions/workflows/ci.yaml/badge.svg)](https://github.com/soundcheck/soundcheck/actions/workflows/ci.yaml)
[![License: Apache-2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/soundcheck/soundcheck)](https://goreportcard.com/report/github.com/soundcheck/soundcheck)
[![Release](https://img.shields.io/github/v/release/soundcheck/soundcheck)](https://github.com/soundcheck/soundcheck/releases)

Soundcheck tests your installed coding agent CLIs (`claude`, `codex`, `opencode`, `agy`, `gemini`, `copilot`) before you hand them tasks. It verifies availability, catches fake exit-zero errors, normalizes quirks, deterministically resolves role assignments without burning LLM tokens, and provides automated fallback cascading.

```
       _____   ____  _    _ _   _ _____   _____ _    _ ______ _____ _  __
      / ____| / __ \| |  | | \ | |  __ \ / ____| |  | |  ____/ ____| |/ /
     | (___  | |  | | |  | |  \| | |  | | |    | |__| | |__ | |    | ' /
      \___ \ | |  | | |  | | . ` | |  | | |    |  __  |  __|| |    |  <
      ____) || |__| | |__| | |\  | |__| | |____| |  | | |___| |____| . \
     |_____/  \____/ \____/|_| \_|_____/ \_____|_|  |_|______\_____|_|\_\
```

---

## Highlights

- **Catch Deceptive Exit Codes**: Many CLI tools exit with status `0` even when failing (e.g. rate limits, expired OAuth sessions). Soundcheck normalizes these into actionable error classes (`429`, `401`, `corrupted`).
- **Zero-Token Role Resolution**: Resolves which agent should handle `primary`, `subagent`, `reviewer`, or `auxiliary` tasks in ≤ 50µs — without asking an LLM and wasting budget.
- **Headless Execution with Auto-Failover**: Run prompts via `soundcheck exec "prompt"`. If the primary adapter is rate-limited or unauthenticated, Soundcheck instantly cascades to the next healthy adapter in the role chain.
- **Four-Tier Probing**:
  - **L0 Presence**: Checks binary existence and version in `$PATH` (≤ 150ms).
  - **L1 Auth**: Inspects credentials and session tokens (with TTL caching to protect OAuth refresh).
  - **L2 Smoke**: Fast, non-destructive execution verification.
  - **L3 Model**: Live model generation test.
- **Single Static Binary**: Full interactive React web UI embedded into a single static executable (~8.3 MB, budget ≤ 20 MB). Zero external runtime dependencies.
- **Model Context Protocol (MCP)**: Native stdio MCP server (`soundcheck mcp`) providing preflight health check, role resolution, and capability inspection directly to Claude Desktop, Cursor, and Antigravity.
- **VS Code Extension**: Official extension in `editors/vscode` featuring live health status bar, agent inspector, headless prompt runner, and embedded web dashboard.

---

## Installation

### macOS & Linux
```bash
curl -fsSL https://raw.githubusercontent.com/soundcheck/soundcheck/main/scripts/install.sh | sh
```

### Windows (PowerShell)
```powershell
irm https://raw.githubusercontent.com/soundcheck/soundcheck/main/scripts/install.ps1 | iex
```

### Homebrew
```bash
brew install soundcheck/tap/soundcheck
```

### Scoop (Windows)
```powershell
scoop bucket add soundcheck https://github.com/soundcheck/scoop-bucket
scoop install soundcheck
```

### Self-Update
```bash
soundcheck upgrade
```

### Build from Source
```bash
git clone https://github.com/soundcheck/soundcheck.git
cd soundcheck
make build
./bin/soundcheck doctor
```

---

## CLI Reference

| Command | Description |
|---|---|
| `soundcheck doctor [--deep] [--json]` | Run comprehensive health check on all installed adapters |
| `soundcheck probe <adapter> [--deep]` | Probe a single adapter (e.g. `soundcheck probe claude`) |
| `soundcheck resolve [role]` | Output deterministic role assignments (e.g. `soundcheck resolve primary`) |
| `soundcheck exec <prompt> [--role <role>] [--adapter <adapter>]` | Run prompt headless with automatic failover cascading |
| `soundcheck run <adapter> [-- <args>]` | Launch interactive PTY session with real-time streaming |
| `soundcheck manifest [--json]` | Export machine-readable agent capability manifest |
| `soundcheck serve [--port 3000] [--token <token>]` | Start HTTP server and embedded web dashboard (default port 3000) |
| `soundcheck mcp` | Launch stdio Model Context Protocol (MCP) server |
| `soundcheck upgrade [--check]` | Check for newer releases and self-update executable |
| `soundcheck version` | Print version and build metadata |

---

## REST & Streaming API Reference

When launched via `soundcheck serve [--port 3000]`, the embedded HTTP server exposes the following endpoints:

| Endpoint | Method | Description |
|---|---|---|
| `/api/health` | `GET` | Health check and server version |
| `/api/v1/doctor` | `GET` | Aggregated preflight report (supports SSE streaming via `Accept: text/event-stream`) |
| `/api/v1/probe` | `GET` | Run preflight probes across all CLI adapters |
| `/api/v1/probe/{id}` | `POST` | Trigger targeted probe for a single adapter |
| `/api/v1/resolve` | `GET`, `POST` | Deterministic role-to-adapter resolution |
| `/api/v1/manifest` | `GET` | Complete capability manifest JSON |
| `/api/v1/config` | `GET` | Read active configuration (returns `X-Config-Path` header) |
| `/api/v1/config` | `PUT` | Atomically validate and persist configuration to YAML file |
| `/api/v1/config/validate` | `POST` | Validate candidate list against live probed host adapters |
| `/api/v1/terminal/ws` | `WS` | WebSocket PTY session for interactive CLI streaming |

---

## Configuration & Web UI Resolver Policy Editor

Soundcheck deterministically routes agent roles (`primary`, `subagent`, `auxiliary`, `reviewer`) based on rules and candidate fallbacks.

### Configuration File Location & `X-Config-Path`

When Soundcheck starts, it resolves configuration files in order:
1. Explicit path passed via `-c / --config <file>`
2. `./soundcheck.yaml` or `./soundcheck.yml`
3. `./agent-doctor.yaml` or `./agent-doctor.yml`
4. `~/.config/soundcheck/config.yaml`
5. Built-in defaults (persisting defaults to `agent-doctor.yaml` if no file was found)

Every request to `/api/v1/config` returns the active filepath in the `X-Config-Path` HTTP response header (and the header is exposed in CORS via `Access-Control-Expose-Headers: X-Config-Path`).

Real request header inspection:
```http
HTTP/1.1 200 OK
Content-Type: application/json
X-Config-Path: agent-doctor.yaml
```

### Web UI Resolver Policy Editor

Run the server to open the web dashboard:
```bash
soundcheck serve --port 3000
```

Navigate to **Resolver** in the web interface and click **Edit Policies** to open the `RoleConfigEditor`:
- Displays active file path from `X-Config-Path` and schema version.
- **Candidate Fallbacks per Role**: Edit candidate lists (e.g., `claude:claude-sonnet-5-5`, `codex:*`). Clicking live discovered account models automatically appends candidates.
- **Policy Controls**:
  - `require_distinct_vendor`: Forces primary and reviewer agents to run on different AI vendor infrastructures (e.g. Anthropic vs OpenAI).
  - `max_cost_usd_per_run`: Numerical ceiling rejecting candidates exceeding token cost budget.
  - `probe.ttl_minutes`: Probe cache duration for healthy adapter runs.
- **Live Verification Badges**: Candidates show a green badge if verified, or a red badge with the failure reason (e.g. uninstalled binary, unauthenticated session, or unknown model ID).

### Verify and Save Flow

1. **Verify (`POST /api/v1/config/validate`)**:
   - The UI evaluates candidates without altering files on disk.
   - Probes live host CLIs and accounts in real time.
   - Returns per-candidate validation records with exact error reasons when unverified.
2. **Save (`PUT /api/v1/config`)**:
   - Sends candidate mappings and policies to `/api/v1/config`.
   - Validates schema, role definitions, and candidates.
   - Atomically persists to disk using a temporary file with `0600` permissions and atomic rename.
   - Returns HTTP `400 Bad Request` with error JSON if validation fails, preventing file corruption.

### Real Example: `POST /api/v1/config/validate`

Request:
```bash
curl -s -X POST http://127.0.0.1:3000/api/v1/config/validate \
  -H "Content-Type: application/json" \
  -d '{"roles":{"primary":["claude:claude-sonnet-5-5","unknown-agent:model-x","codex:bogus-model"]}}'
```

Real response:
```json
[
  {
    "role": "primary",
    "candidate": "claude:claude-sonnet-5-5",
    "verified": true
  },
  {
    "role": "primary",
    "candidate": "unknown-agent:model-x",
    "verified": false,
    "reason": "Unknown agent \"unknown-agent\"."
  },
  {
    "role": "primary",
    "candidate": "codex:bogus-model",
    "verified": false,
    "reason": "Model \"bogus-model\" is not offered by your OpenAI Codex account. Available: gpt-6-astra, gpt-6.1-sol, gpt-6.1-terra, gpt-6-sol, gpt-6-luna, gpt-5.6-sol, gpt-5.6-terra, gpt-5.6-luna, gpt-daybreak-blue-latest, gpt-daybreak-red-latest, gpt-5.5, codex-auto-review."
  }
]
```

---

## Live Model Discovery via `modelsCommand`

Adapters can specify an optional `modelsCommand` in their YAML definition (e.g. `codex debug models` for Codex).

- **With `modelsCommand`**: During adapter probing, Soundcheck executes this command against the CLI runner under a 10-second timeout. If successful (`exit 0`), Soundcheck parses live JSON output for models available to the authenticated account. Known models retain vendor-doc pricing/context metadata, while newly discovered models are surfaced with clean zero-invention metadata and marked as live.
- **Without `modelsCommand` (or when command fails / CLI lacks command)**: For CLIs lacking dynamic discovery commands (such as Claude Code or Gemini CLI), Soundcheck preserves the curated static model catalog declared under `models:` in the adapter's YAML file.

For full authoring details, see [Adapter Authoring Guide](docs/ADAPTER_AUTHORING.md).

---

## Architecture

Soundcheck follows a strict **Hexagonal Architecture** with a **Pure Functional Core** and an **Imperative Shell**:

- **Pure Functional Core** (`internal/core/*`):
  - Strictly pure: data in → data out.
  - Zero IO, zero system time, zero goroutines, zero environment variables, zero global state.
  - Standard library imports only. Verified by `internal/arch_test.go`.
- **Imperative Shell** (`internal/edge/*`, `internal/transport/*`):
  - Subprocess execution (`internal/edge/exec`), PTY streaming, probe runner, config loading, and self-update.
  - Transports: CLI (`cobra`), HTTP (`net/http` + embedded UI), WebSocket PTY, and MCP (`internal/transport/mcp`).

---

## Documentation

- [Adapter Authoring Guide](docs/ADAPTER_AUTHORING.md) — How to define new CLI adapters and probe tiers.
- [Quirk Authoring Guide](docs/QUIRK_AUTHORING.md) — How to catalog deceptive errors and remediation recipes.
- [Contributor Guidelines](CONTRIBUTING.md) — Package lanes, coding rules, and quality gates.

---

## License

Apache-2.0. Copyright (c) 2026 Soundcheck Contributors.
