# AGENTS.md — Contributor & Agent Guidelines

## 1. Core Principles & Rules

1. **Read First**: Read `AGENTS.md` and the relevant package `doc.go` before proposing or making changes. Query CodeGraph before grep.
2. **Lane Ownership**: Touch only owned paths according to the ownership table. Cross-lane changes require coordination or a dedicated task.
3. **Pure Functional Core**: `internal/core/*` is strictly pure:
   - Data in → data out. Zero IO, zero time, zero goroutines, zero env, zero global state.
   - Standard library imports only (plus intra-core imports). No external dependencies.
   - Purity is enforced automatically via `internal/arch_test.go`.
4. **Imperative Shell**: All IO, external command execution, process lifecycle, filesystem access, and network operations belong exclusively in `internal/edge/*` or `internal/transport/*`.
5. **Quality Gates**:
   - `make check` must pass cleanly before every commit/handoff.
   - Use Conventional Commits (`feat:`, `fix:`, `chore:`, `refactor:`, `test:`).
   - Functions ≤ 30 lines, ≤ 4 parameters, cyclomatic complexity ≤ 10 (enforced via `.golangci.yml`).
   - No `utils`, `common`, `helpers`, or `manager` packages. Name packages for their domain intent.
   - Never fabricate metrics, simulate outcomes, or hardcode fake numbers.

---

## 2. Package Ownership & Lanes

| Lane | Role | Owned Paths | Active Phases |
|---|---|---|---|
| **A Core** | Core Domain & Engine | `internal/core/**`, `testdata/golden/**` | 0, 1 |
| **B Edge** | System Shell & Process Exec | `internal/edge/**`, `testdata/fakecli/**` | 2, 3 |
| **C Adapters/Quirks** | CLI Adapters & Quirks Catalog | `internal/adapters/**`, `internal/quirks/**` | 3 |
| **D CLI** | Command Line Interface | `internal/transport/cli/**`, `cmd/**` | 4 |
| **E PTY** | Pseudoterminal & Streaming | `internal/edge/exec/pty*`, `internal/transport/ws/**` | 5 |
| **F Server** | HTTP Server & MCP API | `internal/transport/http/**`, `internal/mcp/**`, `schemas/**` | 6 |
| **G UI** | Web Interface & Assets | `ui/**` | 7 |
| **H Release** | Packaging & CI Automation | `.goreleaser.yaml`, `scripts/**`, `.github/**` | 8 |
| **I QA/Perf** | Performance & Test Suites | Benchmarks, fuzz tests, integration/e2e | All |
| **J Review** | Architecture & Security Audit | Read-only | Every gate |

---

## 3. Standard Commands

- `make check`: Run format check, linters, unit tests, and `go vet` (< 30s).
- `make test`: Run all unit and architectural tests (`go test -v ./...`).
- `make lint`: Run `golangci-lint` with project rules.
- `make build`: Build Go binary into `bin/soundcheck`.
- `make ci`: Run full test suite with race detector (`go test -v -race ./...`).

---

## 4. Extension Recipes

### Recipe: Adding an Adapter
1. Define adapter metadata and probe configurations as declarative data under `internal/adapters/`.
2. Do not modify the core engine.
3. Add corresponding test fixtures or fake-CLI scripts in `testdata/fakecli/`.
4. Validate that all probe tiers (L0-L3) parse and classify output correctly.

### Recipe: Adding a Quirk
1. Add the quirk specification (regex/pattern, error classification, remediation message) to `internal/quirks/catalog.yaml` (or Go table).
2. Add positive and negative test cases in `testdata/golden/`.
3. Verify classification using `go test ./internal/core/classify/...`.

### Recipe: Adding an Endpoint or Transport
1. Implement protocol handlers under `internal/transport/http/` or `internal/transport/cli/`.
2. Delegate all domain evaluation to pure core packages.
3. Validate request/response contracts against JSON schemas under `schemas/`.
