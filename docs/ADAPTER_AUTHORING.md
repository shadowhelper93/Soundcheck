# Adapter Authoring Guide

Soundcheck uses declarative YAML manifests under `internal/adapters/*.yaml` to define CLI adapters. All adapters are embedded directly into the single binary at compile time via `embed.FS`.

---

## 1. Adapter Specification Schema

Each adapter is authored as a YAML file named `<adapter-id>.yaml` (e.g. `claude.yaml`, `codex.yaml`, `agy.yaml`).

```yaml
id: "myagent"
name: "MyAgent CLI"
binary: "myagent"
versionArgs: ["--version"]
headlessCommand: "myagent -p"
modelsCommand: "myagent debug models" # Optional live discovery command

roles:
  - "primary"
  - "subagent"

models:
  - id: "myagent-model-v1"
    displayName: "MyAgent Model v1"
    vendor: "MyAgent Inc"
    tier: "frontier"
    contextWindow: 200000
    inputCostPerM: 2.0
    outputCostPerM: 10.0
    verified: true

probe:
  l0:
    command: "myagent --version"
    timeoutMs: 3000
    expectedExit: 0
  l1:
    command: "myagent auth status"
    timeoutMs: 5000
    expectedExit: 0
  l2:
    command: "myagent test-smoke"
    timeoutMs: 8000
    expectedExit: 0
  l3:
    command: "myagent -p 'ping'"
    timeoutMs: 15000
    expectedExit: 0
```

---

## 2. Probe Tiers (L0 – L3)

| Tier | Name | Purpose | Performance Budget |
|---|---|---|---|
| **L0** | Presence | Verifies binary existence in `$PATH` via `lookPath` and version flag. | ≤ 150ms across all adapters |
| **L1** | Auth Status | Verifies authentication tokens, credentials, or session status. | ≤ 5000ms |
| **L2** | Local Smoke | Fast, non-destructive execution without burning model tokens. | ≤ 8000ms |
| **L3** | Live Model | End-to-end model invocation verifying live token generation. | ≤ 15000ms |

---

## 3. Headless Command Requirements

The `headlessCommand` property is used by `soundcheck exec` to pass prompts non-interactively.

- Must accept a single prompt string as trailing argument or flag.
- Must execute non-interactively without prompting for TTY input.
- Must not invoke interactive plugin hooks or spinners.
- Example: `claude -p --bare` or `agy -p`.

---

## 4. Live Model Discovery (`modelsCommand`)

Adapters can optionally declare `modelsCommand`:
- Command executed via shell runner under 10s timeout during probe.
- Expected to return JSON containing `.models[]` with `slug` (or `id`) and `display_name`.
- When present and successful, live models replace or augment the static catalog for the logged-in account.
- When omitted or when the command returns non-zero, Soundcheck falls back safely to the curated static `models:` list in the YAML file.

---

## 5. Adding an Adapter Step-by-Step

Per `AGENTS.md` Lane C:
1. Create `internal/adapters/<id>.yaml` adhering to the schema above.
2. Define adapter metadata, supported roles, optional `modelsCommand`, and probe configurations (L0-L3).
3. If the adapter encounters deceptive exit codes (e.g. exit 0 on failure), define quirks under `quirks:` with exact regex patterns and error classifications.
4. Add corresponding test fixtures or fake-CLI scripts in `testdata/fakecli/<id>`.
5. Run tests:
   ```bash
   go test -v ./internal/adapters/...
   go test -v ./internal/edge/probe/...
   ```
6. Run `make check` and `bash scripts/parity.sh` to ensure all quality gates and arch tests pass.

---

## 6. Test Fixtures

When adding or modifying an adapter:
1. Create mock or fake-CLI scripts under `testdata/fakecli/<adapter-id>`.
2. Add corresponding golden output fixtures in `testdata/golden/`.
3. Validate classification using `go test ./internal/core/classify/...`.
4. Validate probe execution using `go test ./internal/edge/probe/...`.
