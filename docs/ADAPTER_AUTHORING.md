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

roles:
  - "primary"
  - "subagent"

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

## 4. Test Fixtures

When adding or modifying an adapter:
1. Create mock or fake-CLI scripts under `testdata/fakecli/<adapter-id>`.
2. Add corresponding golden output fixtures in `testdata/golden/`.
3. Validate classification using `go test ./internal/core/classify/...`.
4. Validate probe execution using `go test ./internal/edge/probe/...`.
