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
