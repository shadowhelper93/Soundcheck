# Quirk Authoring Guide

Quirks capture idiosyncratic CLI behavior, deceptive exit codes (e.g. exit 0 on rate limit or missing auth), and upstream CLI regressions.

---

## 1. Quirk Specification Schema

Quirks are defined in `internal/quirks/catalog.yaml`:

```yaml
quirks:
  - id: "claude-79500"
    adapterId: "claude"
    kind: "rate_limited"
    pattern: "rate_limit_exceeded|hit your limit"
    expectedExitCode: 0
    actualExitCode: 0
    cooldownMinutes: 30
    description: "Claude Code CLI exits 0 on rate limit error instead of non-zero code."
    remediation: "Wait 30 minutes for token window reset, or failover to secondary adapter."
```

---

## 2. Quirk Kinds

| Kind | Description | Fallback Action |
|---|---|---|
| `rate_limited` | API rate limit or credit depletion encountered. | Auto-failover to next adapter in role fallback chain. Cooldown applied. |
| `unauthorized` | Missing API key, expired OAuth session, or invalid credentials. | Skip adapter; suggest login command in remediation. |
| `corrupted` | Unparseable output, broken internal state, or incomplete JSON. | Skip adapter; advise CLI upgrade or config reset. |
| `model_not_found` | Requested model alias or engine does not exist. | Retry with fallback model or alternative CLI. |

---

## 3. Authoring Workflow

1. **Add Quirk Definition**:
   Append the quirk entry to `internal/quirks/catalog.yaml`.
2. **Add Golden Snapshot**:
   Create a test fixture in `testdata/golden/<quirk-id>.json` containing sample stdout, stderr, and exit code.
3. **Run Classification Tests**:
   ```bash
   go test -v ./internal/core/classify/...
   ```
4. **Verify No Allocations in Hot Path**:
   Classification of 4KB output must run within ≤ 20µs with 0 heap allocations.
