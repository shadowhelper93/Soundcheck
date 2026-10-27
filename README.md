# Soundcheck

> **Deterministic Preflight Prober & Role Resolver for AI Coding Harnesses**

Soundcheck tests your installed coding agent CLIs (`claude`, `codex`, `opencode`, `agy`, `gemini`, `copilot`) before you hand them tasks. It verifies availability, catches fake exit-zero errors, normalizes quirks, and deterministically assigns roles without burning tokens.

```
       _____   ____  _    _ _   _ _____   _____ _    _ ______ _____ _  __
      / ____| / __ \| |  | | \ | |  __ \ / ____| |  | |  ____/ ____| |/ /
     | (___  | |  | | |  | |  \| | |  | | |    | |__| | |__ | |    | ' /
      \___ \ | |  | | |  | | . ` | |  | | |    |  __  |  __|| |    |  <
      ____) || |__| | |__| | |\  | |__| | |____| |  | | |___| |____| . \
     |_____/  \____/ \____/|_| \_|_____/ \_____|_|  |_|______\_____|_|\_\
```

---

## Quickstart

### Build & Run
```bash
make build
./bin/soundcheck doctor
```

### Start Web UI
```bash
./bin/soundcheck serve --port 8080
```
Open `http://localhost:8080` to view the interactive dashboard.

---

## What It Solves

1. **Deceptive Exit Codes**: Many CLIs exit with status `0` even when failing (e.g. hitting rate limits, missing auth). Soundcheck parses output to catch these silent failures.
2. **Zero-Token Role Resolution**: Resolves which agent should handle primary, subagent, reviewer, or auxiliary tasks deterministically — without asking an LLM and wasting tokens.
3. **Four-Tier Probing**:
   - **L0 Presence**: Checks if the CLI binary exists in `$PATH`.
   - **L1 Auth**: Verifies valid credentials/tokens.
   - **L2 Smoke**: Runs a fast, non-destructive test execution.
   - **L3 Model**: Validates model invocation and output.

---

## CLI Commands

| Command | Description |
|---|---|
| `soundcheck doctor` | Run comprehensive health check on all installed adapters |
| `soundcheck doctor --deep` | Include L2/L3 smoke and execution tests |
| `soundcheck probe <adapter>` | Probe a single adapter (e.g. `soundcheck probe claude`) |
| `soundcheck resolve` | Output deterministic role assignments (primary, reviewer, etc.) |
| `soundcheck manifest` | Export a machine-readable JSON capability manifest |
| `soundcheck serve` | Launch the web dashboard on port `8080` |

---

## Contributing

Soundcheck is written in standard Go (1.24+).

### Development Commands
```bash
make check      # Run linters, unit tests, and vet
make test       # Run tests
make build      # Build binary to bin/soundcheck
```

### Adding an Adapter
Adapters live as declarative YAML specs in `internal/adapters/embedded/`. Add a YAML file defining the binary, probe arguments, and regex matchers.
