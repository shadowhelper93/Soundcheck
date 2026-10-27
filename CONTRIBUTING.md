# Contributing to Soundcheck

Thank you for contributing to Soundcheck! Please review the guidelines below before opening pull requests.

---

## 1. Core Principles

1. **Pure Functional Core**:
   - `internal/core/*` packages must be strictly pure: zero IO, zero time, zero goroutines, zero environment variables, zero global state.
   - Standard library imports only (plus intra-core imports).
   - Automatically verified by `go test -v ./internal -run TestCorePackageImportsStdlibOnly`.
2. **Imperative Shell**:
   - Filesystem access, process execution, PTY streaming, network, and HTTP operations belong exclusively in `internal/edge/*` or `internal/transport/*`.
3. **Quality Gates**:
   - All PRs must pass `make check` cleanly (0 lints, 0 format discrepancies, all tests green).
   - Functions ≤ 30 lines, ≤ 4 parameters, cyclomatic complexity ≤ 10.
   - Single binary size budget: ≤ 20MB (`make release-check`).
4. **Conventional Commits**:
   - Commit messages must follow the Conventional Commits format (`feat:`, `fix:`, `chore:`, `refactor:`, `test:`, `docs:`).

---

## 2. Lane Ownership

| Lane | Role | Owned Paths |
|---|---|---|
| **A Core** | Core Domain & Engine | `internal/core/**`, `testdata/golden/**` |
| **B Edge** | System Shell & Process Exec | `internal/edge/**`, `testdata/fakecli/**` |
| **C Adapters/Quirks** | CLI Adapters & Quirks Catalog | `internal/adapters/**`, `internal/quirks/**` |
| **D CLI** | Command Line Interface | `internal/transport/cli/**`, `cmd/**` |
| **E PTY** | Pseudoterminal & Streaming | `internal/edge/exec/pty*`, `internal/transport/ws/**` |
| **F Server** | HTTP Server & MCP API | `internal/transport/http/**`, `internal/transport/mcp/**`, `schemas/**` |
| **G UI** | Web Interface & Assets | `ui/**` |
| **H Release** | Packaging & CI Automation | `.goreleaser.yaml`, `scripts/**`, `.github/**` |
| **I QA/Perf** | Performance & Test Suites | Benchmarks, fuzz tests, integration/e2e |

---

## 3. Standard Commands

```bash
make check          # Run format check, linters, unit tests, and go vet (< 30s)
make test           # Run all unit tests
make ci             # Run full test suite with race detector
make ui-build       # Build UI assets into ui/dist
make cross-build    # Cross-compile for all 6 release targets
make release-check  # Verify binary size against 20MB budget
```
