# CLAUDE.md — Soundcheck Architecture & Development Guide

This document provides conventions, architecture rules, and command references for contributors and AI agents working on Soundcheck.

---

## 1. Project Overview

Soundcheck is a deterministic preflight prober, role resolver, and DAG workflow studio for autonomous coding agents (`claude`, `codex`, `opencode`, `agy`, `gemini`, `copilot`).

Key capabilities:
- **Catches False Exit-0 Quirks**: Normalizes deceptive CLI exit codes into true status classes (`rate_limited`, `unauthenticated`, `corrupted_state`, `permission_denied`).
- **Zero-Token Role Resolution**: Resolves candidate chains for agent roles (`primary`, `subagent`, `auxiliary`, `reviewer`) in ≤ 50µs without burning LLM tokens.
- **Pure Hexagonal Architecture**: Strict separation between the pure domain core and the imperative execution shell.
- **Embedded Webview / SPA**: Single static binary embedding React UI, live PTY streaming over WebSocket, and a Model Context Protocol (stdio MCP) server.

---

## 2. Architectural Boundaries & Lanes

Soundcheck strictly follows hexagonal architecture:

```
                  ┌───────────────────────────────────────────────┐
                  │              IMPERATIVE SHELL                 │
                  │  CLI (Cobra)  ·  HTTP (net/http)  ·  MCP/WS   │
                  │  (internal/transport/*, cmd/*)                │
                  └───────────────────────┬───────────────────────┘
                                          │
                                          ▼
                  ┌───────────────────────────────────────────────┐
                  │                 EDGE LAYER                    │
                  │  Subprocess Exec  ·  PTY  ·  Config File IO   │
                  │  (internal/edge/*, internal/adapters/*)       │
                  └───────────────────────┬───────────────────────┘
                                          │
                                          ▼
                  ┌───────────────────────────────────────────────┐
                  │             PURE FUNCTIONAL CORE              │
                  │  Classifier · Resolver · Manifest · Workflow  │
                  │  (internal/core/*)                            │
                  │  * Zero IO, Zero Time, Zero Syscalls, Stdlib  │
                  └───────────────────────────────────────────────┘
```

### Lane Ownership

| Lane | Role | Scope | Constraints |
|---|---|---|---|
| **A Core** | Core Domain & Engine | `internal/core/**`, `testdata/golden/**` | Pure Go standard library only. Zero IO, zero time, zero goroutines, zero environment variables. Purity enforced via `internal/arch_test.go`. |
| **B Edge** | System Shell & Process Exec | `internal/edge/**`, `testdata/fakecli/**` | Process spawning, PTY runners, file store, TTL cache, platform checks. |
| **C Adapters/Quirks** | CLI Adapters & Quirks Catalog | `internal/adapters/**` | Declarative YAML definitions for agents and quirk signatures. |
| **D CLI** | Command Line Interface | `internal/transport/cli/**`, `cmd/**` | Cobra commands, flags, terminal outputs, diagnostic exits. |
| **E PTY** | Pseudoterminal & Streaming | `internal/edge/exec/pty*`, `internal/transport/ws/**` | Terminal multiplexing, WebSocket terminal bridge. |
| **F Server** | HTTP Server & MCP API | `internal/transport/http/**`, `internal/transport/mcp/**`, `schemas/**` | REST endpoints, SSE streams, JSON-RPC 2.0 stdio MCP server. |
| **G UI** | Web Interface & Assets | `ui/**` | React 19, TypeScript, Tailwind CSS, Vite. UI builds into `ui/dist/` and embeds into Go. |
| **H Release** | Packaging & CI Automation | `.goreleaser.yaml`, `scripts/**`, `.github/**` | Cross-compilation, releases, installers. |
| **I QA/Perf** | Performance & Test Suites | Benchmarks, parity gates, fuzzing | Parity validation across all golden files. |

---

## 3. Development Commands

### Common Tasks
```bash
make check            # Format check, linters (golangci-lint), unit tests, and go vet (< 30s)
make test             # Run all unit tests
make ci               # Run full test suite with race detector (go test -v -race ./...)
make build            # Build UI and compile single static binary into bin/soundcheck
make ui-build         # Build React UI assets into ui/dist
make cross-build      # Cross-compile release binaries for 6 OS/arch targets
make release-check    # Verify compiled binary adheres to ≤ 20MB budget
bash scripts/parity.sh # Run 3-gate parity verification against golden fixtures
```

### Running Locally
```bash
# Run CLI health check
./bin/soundcheck doctor

# Run deep health check (executes actual L2/L3 model turns)
./bin/soundcheck doctor --deep

# Start HTTP server and Web UI
./bin/soundcheck serve --port 3000

# Start MCP stdio server
./bin/soundcheck mcp
```

---

## 4. Coding & Quality Standards

1. **Functions**: Max 30 lines, max 4 parameters, cyclomatic complexity ≤ 10 (enforced via `.golangci.yml`).
2. **Naming**: No `utils`, `common`, `helpers`, or `manager` packages. Name packages strictly for domain intent.
3. **Purity**: Never add filesystem, time, or network access to `internal/core/*`. Pass timestamps and inputs as arguments.
4. **Commits**: Conventional Commits format required (`feat:`, `fix:`, `chore:`, `refactor:`, `test:`, `docs:`).
5. **Real Data**: Never simulate outcomes, fabricate telemetry, or invent fake model IDs/prices. Keep pricing and capabilities grounded in live verification or official vendor specifications.
