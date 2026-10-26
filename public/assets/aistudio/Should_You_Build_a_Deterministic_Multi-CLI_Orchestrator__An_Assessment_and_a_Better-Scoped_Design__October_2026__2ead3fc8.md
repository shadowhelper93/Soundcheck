# Should You Build a Deterministic Multi-CLI Orchestrator? An Honest Assessment and a Better-Scoped Design (October 2026)

Don't build the general orchestrator you described. By late 2026 the "run Claude Code, Codex, OpenCode and friends from one place, with auto-detection" space is crowded. What is still missing is a trustworthy, deterministic preflight-and-capability layer: something that can say which agents, models and auth modes will actually complete a headless turn right now, and resolve your role preferences against that. Build that as a small, sharp tool (a "doctor + resolver") that emits a machine-readable capability manifest. Add a thin declarative workflow runner only if you still need one after trying the existing tools.

## TL;DR

- **Most of your idea already exists.** Vibe Kanban, Agent of Empires, Parallel Code, Crewplane, multicli, claude-code-router, Ruflo, and Claude Code's own subagents, agent teams and dynamic workflows already cover multi-CLI launching, auto-detection of installed CLIs, model routing with failover, and declarative multi-stage workflows. A generic orchestrator would be roughly the 30th entrant, with no moat.
- **The real gap is honest preflight.** The CLIs are inconsistent and sometimes misleading about health:
  - Claude Code's headless mode has a documented bug where a rate-limit error returns `subtype:"success"` with exit code 0.\[1\]
  - Codex has no headless way to read quota.\[2\]
  - Antigravity CLI (which replaced Gemini CLI for individual accounts on June 18, 2026)\[3\] has no auth-status command at all.
  
  A deterministic tool that normalizes this into "available / authenticated / functional / quota-limited", per CLI and per model, is useful and not well served today.
- **Recommendation: build `agent-doctor` first.** Make it a Go (or Rust) single binary with:
  - a per-CLI adapter interface;
  - tiered probes (presence → auth → live smoke test), cached with a TTL;
  - one canonical YAML config validated by JSON Schema (JSON accepted for free);
  - a deterministic rule-based role resolver with fallback chains.
  
  Expose the manifest so existing runners (Crewplane, Vibe Kanban, your own scripts) can consume it. Only then consider a minimal workflow runner, and keep the planner non-AI.

## Key Findings

### 1. The landscape: what already exists and what each does or doesn't do

| Category | Tool | What it does | Auto-detect / health / fallback |
|---|---|---|---|
| Native multi-agent | Claude Code subagents, agent teams, dynamic workflows | Subagents report back to the parent. Agent teams coordinate multiple Claude Code instances through a shared task list and messaging; they are experimental, disabled by default, and don't isolate teammates in worktrees. Dynamic workflows move the plan into a script that the runtime executes, with up to 16 concurrent agents by default.\[4\]\[5\] | Claude-only. No cross-vendor health checks. |
| Session managers | Agent of Empires (Rust, tmux) | TUI and web dashboard. Each agent gets a tmux session, optional git worktree and optional Docker sandbox. Supports 16 agents, including Claude Code, OpenCode, Codex, Gemini, Antigravity, Cursor, Copilot, Pi and Kiro.\[6\] | "AoE auto-detects which are installed." Reports status (running/waiting/idle/error), not functional health.\[6\] |
| Session managers | Vibe Kanban (Rust) | Kanban board. Each task runs in its own worktree with a terminal and dev server. Supports 10+ agents, including Claude Code, Codex, Gemini CLI, Copilot, Amp, Cursor, OpenCode, Droid, CCR and Qwen Code.\[7\] | Uses an executor abstraction per agent. Runs agents with permission-skipping flags by default.\[8\]\[9\] |
| Session managers | Parallel Code, Crystal, CliDeck, Claude Squad, CCManager, etc. | Parallel worktrees plus diff review.\[10\]\[11\] | Mostly presence-based. |
| Declarative workflow runners | Crewplane (Python, Apache-2.0) | Markdown workflows: YAML frontmatter defines a DAG and Markdown sections hold the prompts. Supports parallel fan-out across providers and executor/reviewer loops. Handoffs are persisted on disk, and a preflight step validates the composed DAG. Works with Claude Code, Codex, Gemini, Copilot CLI or any CLI.\[12\] | Validates the DAG, not provider health. Small project (about 41 stars).\[13\] |
| Declarative workflow runners | oh-my-claudecode, Claude Octopus, Maestro-Orchestrate | Claude-hosted plugins that spawn Codex, Gemini or Antigravity workers or consensus reviewers.\[14\]\[15\]\[16\] | Expect the CLIs to be "installed/authenticated."\[15\] |
| Cross-CLI bridges | multicli (MCP) | Lets Claude, Antigravity, Codex and OpenCode call each other as tools.\[17\] | "Detects which AI CLIs you have installed and configures Multi-CLI for them automatically."\[17\] |
| Swarm frameworks | Ruflo (formerly claude-flow) | Hive-mind orchestration, memory, and multi-provider routing (Claude, GPT, Gemini, Cohere, Ollama) with failover. | Heavy. Gerald Chen's May 11, 2026 plugin comparison says Ruflo "underdelivers in execution" and concludes it "is not recommended". |
| Model gateways | claude-code-router (musistudio) | A local gateway giving Claude Code, Codex, OpenCode, Kimi, Grok CLI and others "one stable local endpoint," with routing, failover and observability.\[18\] | Provider-level failover. |
| Model gateways | LiteLLM proxy | Fallbacks, retries, cooldowns (`allowed_fails`, `cooldown_time`), and background health-check-driven routing (`health_check_interval` default 300s). | Mature health and fallback, but at the API level, not the CLI level.\[19\]\[20\] |
| Model gateways | OpenRouter | A `models` array tried in priority order on downtime, rate limits or moderation refusals. | API-level only.\[21\] |
| Standards | ACP (Agent Client Protocol) | JSON-RPC over stdio between a client and an agent. 42 agents in the official ACP registry as of October 2026, per Maksim Danilchenko's ACP guide (Claude via Zed's adapter, Codex via adapter, Gemini CLI, OpenCode, Goose, Cline, Copilot preview, etc.). Registry launched January 28, 2026 by JetBrains and Zed, per both companies' announcement posts. | `initialize` returns `agentCapabilities` and `authMethods`. This is the closest thing to standardized capability discovery. |
| Standards | MCP, AGENTS.md, A2A | MCP covers tools and data. AGENTS.md is, per its official site, "a README for agents" "used by over 60k open-source projects." A2A is agent-to-agent. | None of them address install or auth health. |

**What this means:** launching multiple CLIs, isolating them in worktrees, detecting which ones are installed, and routing models with failover are all solved problems. Multiple tools do it, several with large followings: Vibe Kanban, for example, has about 28.4k GitHub stars, though its README now says it is sunsetting after Bloop shut down on April 10, 2026, and it continues as a community-maintained open-source project. Your feature list as described ("receive a pattern, validate, build a strategy from what works, use preferences or auto-detect") overlaps heavily with Crewplane plus Agent of Empires plus claude-code-router combined.

### 2. Where the real gaps are

The gaps all sit below the orchestrators. They assume that "installed" means "works," and it often doesn't.

**a) Exit codes and status fields are unreliable. This is the big one.**
- Claude Code headless:
  - Issue #79500: API errors such as rate limits return `"subtype":"success"` with `"is_error":true` and exit code 0. In plain text mode the error goes to stdout.\[1\]
  - Issue #74761: `-p` can exit 0 mid-task.\[22\]
  - Issue #103253 (v2.1.302, October 2026) is the reverse: a late `rate_limit_event` makes a successful run exit 1 with empty stdout in json/text mode, while `stream-json` is unaffected.\[23\]
- Codex: `codex doctor --json` emits a redacted machine-readable report,\[24\] but bug reports show it exiting 0, so you must gate on the JSON contents rather than the exit code.\[25\]\[26\]\[27\]
- Gemini CLI uses specific exit codes (41 = `FatalAuthenticationError`).\[28\] It also has a known bug where setting `GOOGLE_GEMINI_BASE_URL` triggers exit 41 even with a valid API key.\[29\]

**b) Auth-state detection differs per CLI, and some CLIs have none.**

| CLI | Non-interactive auth check | Machine-readable? | Exit code reflects login? | Gotchas |
|---|---|---|---|---|
| Claude Code | `claude auth status` (JSON by default; `--text` for humans)\[30\] | Yes: `loggedIn`, `authMethod`, `configDirectory`\[31\]\[32\]\[33\]\[34\] | 0 if logged in, 1 if not. But 1 is also returned for ordinary CLI errors.\[30\]\[35\] | A reported bug: it could start an OAuth refresh and exit without saving it, causing later "token revoked" errors. The changelog says this is fixed.\[32\]\[36\] Don't poll it in a loop. |
| Codex | `codex login status` and `codex doctor --json` | Login status: text only. Doctor: JSON. | Login status: 0 when credentials are present.\[37\] Doctor: not reliably. | A one-off `CODEX_API_KEY` used with `codex exec` doesn't count as a cached login. There is no headless quota view (issue #10233).\[2\]\[38\] |
| OpenCode | `opencode auth list --format json` (v2; `providers` in 1.x)\[39\]\[40\] | Yes | Not documented, so parse the output. | The command was renamed between v1 and v2.\[40\] |
| Antigravity (`agy`) | None documented | n/a | A headless run without cached credentials "exits with an `authentication required` error instead of hanging"\[41\] | Only a live probe works. Check `.status == "SUCCESS"`; WAITING is not a failure.\[3\] Older builds lack `--output-format`.\[42\] |
| Gemini CLI (legacy, paid/enterprise) | None. Probe and check for exit 41. | JSON `error` object\[43\] | Yes (41)\[28\] | Replaced for free and individual tiers.\[44\] |

**c) Model enumeration is inconsistent.** Only OpenCode has a first-class `opencode models` command ("every model your credentials can reach").\[40\] I found no model-listing command for Claude Code, Codex or Antigravity. That means your "preferred Claude primary/sub-agent/auxiliary model" must be validated by a live probe with `--model`, not by looking it up in a list.

**d) Headless invocation differs per CLI, which is why adapters are mandatory.**

| CLI | Headless | Machine output | Permission / sandbox control | Resume |
|---|---|---|---|---|
| Claude Code | `claude -p` | `--output-format json` / `stream-json` (has `is_error`, `total_cost_usd`, `permission_denials`)\[45\]\[46\] | `--permission-mode`, `--allowedTools`; `--bare` skips repo hooks and `.mcp.json` and ignores the subscription login\[46\] | `--continue`, `--resume <id>` |
| Codex | `codex exec` | `--json` (JSONL), `--output-schema`, `-o`\[24\]\[47\] | Read-only sandbox by default; `--sandbox workspace-write`\[46\]\[48\] | `codex exec resume --last`\[47\] |
| OpenCode | `opencode run` | `--format json` (NDJSON: step_start, text, tool_use, step_finish with tokens and cost)\[49\] | `permission` in `opencode.json`, `--auto`\[46\] | `-c`, `-s <id>` |
| Antigravity / Gemini | `agy -p` / `gemini -p` | `--output-format json` / `stream-json`; agy adds `--json-schema`\[3\]\[41\]\[46\] | Gemini: `--approval-mode` plus a policy engine; in headless mode, a rule that would ask "is treated as deny"\[46\] | `--resume latest` |
| Copilot CLI | `copilot -p` | `--output-format json` (JSONL), `-s`\[46\] | `--allow-tool`, `--deny-tool`\[46\] | `--continue`, `--resume` |

**e) Quota awareness is mostly absent.** Claude subscriptions use rolling 5-hour and weekly windows.\[50\]\[51\]\[52\] Claude Code reads rate-limit headers internally and, as of v2.1.283, uses a small grace allowance to wrap up the current step,\[53\] but none of this is exposed as a stable preflight API. Codex exposes no quota headlessly.\[2\] The community response is a set of statusline and menu-bar tools that scrape usage.\[54\]\[55\]\[56\] So the honest best you can offer is reactive detection plus a cooldown, which is exactly the LiteLLM model (`allowed_fails` + `cooldown_time`) applied to CLIs.\[57\]\[58\]

**f) Verification of results.** Headless runs can "succeed" on an empty diff when tool calls are silently denied.\[46\] A runner should verify outcomes deterministically (diff non-empty, tests pass, `permission_denials` empty), not trust the agent's final message.

### 3. Is the deterministic (non-AI) backend the right call?

Yes. This is the strongest part of your idea. Choosing which installed agent fills which role is a constraint-satisfaction problem: preferences, availability, capabilities, and cooldown state. It is not a reasoning problem. Anthropic's own docs draw the same line: with subagents and agent teams "Claude is the orchestrator," whereas "a workflow moves the plan into code."\[5\] Deterministic resolution gives you reproducibility, debuggability ("why did it pick Codex for review?") and zero token cost. Keep LLMs inside the stages, not in the scheduler.

## Recommended Architecture

### Overview

```
┌──────────────────────────────────────────────────────────────┐
│ CLI / library:  doctor │ resolve │ run <pattern>             │
├──────────────────────────────────────────────────────────────┤
│ Config loader (YAML canonical, JSON accepted) + JSON Schema  │
├───────────────┬──────────────────────┬───────────────────────┤
│ Discovery     │ Probe engine         │ Resolver (rules)      │
│ PATH scan,    │ L0 presence/version  │ roles → candidates →  │
│ known config  │ L1 auth (native cmd) │ filter by health/caps │
│ dirs, env     │ L2 live smoke test   │ → fallback chain →    │
│ vars, ACP reg │ L3 model probe       │ explainable decision  │
├───────────────┴──────────────────────┴───────────────────────┤
│ Health cache (TTL, cooldowns, last error class)              │
├──────────────────────────────────────────────────────────────┤
│ Adapters: claude │ codex │ opencode │ agy │ gemini │ copilot │
│           generic-acp │ generic-command                      │
└──────────────────────────────────────────────────────────────┘
        ↓ emits capability manifest (JSON) consumable by others
```

### Adapter interface (one per CLI)

Each adapter should implement:
- `detect()`: binary path, version, config directory.
- `authStatus()`: uses the native command where one exists (Claude, Codex, OpenCode) and returns `unknown` otherwise.
- `listModels()`: native command (OpenCode) or a config-declared list.
- `smokeTest(model, timeout)`: a minimal headless prompt with structured output.
- `invoke(task, role, workdir, permissions) → normalized result`.
- `classifyError(stdout, stderr, exitCode, json)`, which returns one of {`not_installed`, `unauthenticated`, `rate_limited`, `quota_exhausted`, `model_unavailable`, `permission_denied`, `timeout`, `ok`}.

Put all of the per-CLI lies and quirks in `classifyError`. For Claude, trust `is_error` and prefer `stream-json` over the exit code. For agy, check `status`. For Gemini, map exit 41. Ship a fixture test suite of captured real outputs per CLI version. This is the part that rots fastest and the part users will value most.

Also support two generic adapters:
- **ACP:** spawn the agent, call `initialize`, and read `agentCapabilities` and `authMethods`.\[59\]\[60\] This covers capability discovery for the 42 agents in the ACP registry (October 2026 count, per danilchenko.dev) through one code path.
- **Generic command:** user-supplied command template plus a JSON path for the result, as Crewplane does with "any CLI."\[61\]

### Probing strategy: `--version` is not a health check

| Level | What | Cost | Default |
|---|---|---|---|
| L0 Presence | `exec.LookPath`, `--version`, parse semver against an adapter's supported range | ~0 | Always |
| L1 Auth | Native status command (Claude `auth status`, `codex login status`, `opencode auth list --format json`); env-var presence for API keys | ~0 | Always, cached |
| L2 Functional | One tiny headless turn ("reply with the JSON {\"ok\":true}") with a hard timeout (e.g. 30–60s) | Tokens and quota | On cache miss, or `--deep` |
| L3 Model | L2 with each preferred `--model` | Tokens per model | Only for models referenced by roles |

Cache results per (adapter, version, auth identity, model) with a TTL: for example 24h for L0, 1h for L1, 15–30 min for L2. Put any target that returns `rate_limited` into a cooldown, as LiteLLM does.\[19\] Invalidate the cache when the binary version or config file mtime changes. Run probes concurrently with a global timeout so `doctor` finishes in seconds.

### Configuration: pick one canonical format

Don't maintain "JSON or YAML" as two first-class formats in your docs. Make YAML canonical and publish a JSON Schema. Because YAML 1.2 is designed as a superset of JSON, a YAML parser will read a `.json` file anyway, so "both" is free at the parser level without doubling your docs. Use the schema for editor autocomplete and for precise validation errors. Avoid adding TOML even though Codex, Agent of Empires and Gemini policies use it; three formats is pure overhead. Example:

```yaml
# agent-doctor.yaml
version: 1
roles:
  primary:   [claude:opus, codex:gpt-5.x, opencode:anthropic/claude-sonnet]
  subagent:  [claude:sonnet, opencode:*]
  auxiliary: [claude:haiku, agy:gemini-flash]
  reviewer:  [codex:*, claude:*]       # prefer a different vendor than primary
agents:
  codex:    { sandbox: workspace-write, profile: ci }
  opencode: { models: [anthropic/claude-sonnet, openai/gpt-5.x-mini] }
policy:
  require_distinct_vendor: [primary, reviewer]
  max_cost_usd_per_run: 5
  probe: { deep: false, ttl_minutes: 30 }
```

All preferences are optional. A missing role falls back to the auto-discovery ranking.

### Zero-config auto-discovery

1. Scan PATH for known binaries: `claude`, `codex`, `opencode`, `agy`, `gemini`, `copilot`, `cursor-agent`, `goose`, `aider`, and so on.
2. Check known config dirs (`~/.claude`, `~/.codex`, `~/.config/opencode`, `~/.gemini/antigravity-cli`) and API-key env vars (`ANTHROPIC_API_KEY`, `OPENAI_API_KEY` / `CODEX_API_KEY`, `GEMINI_API_KEY`, `OPENROUTER_API_KEY`).
3. Optionally check the ACP registry for ACP-capable agents.
4. Run L0 and L1 on everything, and L2 only on the top candidate per role.
5. Rank with a static, documented default table (e.g. primary: strongest available frontier agent; auxiliary: cheapest fast model). Print the decision with reasons.

Discovery is cheap. Never silently burn quota: L2 probes in zero-config mode should be opt-in or limited to one per role.

### Patterns: express workflows declaratively, and don't invent a new DSL

Model a pattern as a small DAG of stages, each declaring a role (not a vendor), inputs (artifacts from earlier stages), a permission profile, and deterministic gates. Crewplane's "YAML frontmatter for the graph, Markdown for the prompts" is a good format to copy or simply adopt:\[13\]

```yaml
pattern: plan-implement-review
stages:
  plan:      { role: primary,  permissions: read-only,  output: plan.md }
  implement: { role: subagent, needs: [plan], permissions: workspace-write, worktree: true }
  review:    { role: reviewer, needs: [implement], permissions: read-only,
               gate: { tests: "npm test", diff_nonempty: true, max_loops: 2 } }
```

The resolver binds roles to concrete agent and model pairs at run time from the health manifest, so the same pattern runs on any machine.

### Language choice

- **Go (recommended):** a single static binary, excellent subprocess, process-group and timeout handling, trivial cross-compilation, and fast startup for a `doctor` command. Google rewrote Antigravity CLI in Go for similar reasons.\[3\]\[62\]
- **Rust:** equally good, and the choice of Agent of Empires and Vibe Kanban.\[6\]\[8\] Pick it if you already know it; there's slower iteration on fiddly output parsers.
- **TypeScript:** best if you want to lean on ACP and the Claude Agent SDK. The cost is that users need Node, and process management is clunkier.
- **Python:** fastest to prototype (Crewplane uses it), but distribution is weaker for a tool that should "just run."\[63\]

### Security

- **Least privilege by default.** Default to read-only and escalate per stage. Codex's headless default is a read-only sandbox, and Gemini's headless mode turns "ask" into "deny."\[46\]\[48\] Don't copy Vibe Kanban's default of permission-skipping flags.\[9\]
- **Worktree per writing stage, optional container.** Agent of Empires shows that Docker, Podman or Apple Containers sandboxing with shared auth volumes is practical.\[6\]\[64\]
- **Repo-controlled code runs with your credentials.** Claude Code's `-p` runs repo hooks and `.mcp.json` servers with no trust dialog unless you pass `--bare`. OpenAI warns against job-level `OPENAI_API_KEY` or `CODEX_API_KEY` in workflows that run repo code.\[46\] Pass secrets per invocation through an env allowlist, never log them, and redact probe output (copy Codex doctor's redaction approach).\[65\]
- **Verify outcomes deterministically** (tests, diff, `permission_denials`) before marking a stage green.

## Recommendations

**Verdict:** don't build "yet another orchestrator." Build the missing preflight layer, and make it composable.

1. **Phase 1 MVP: `agent-doctor` (about 2–4 weeks part-time).**
   - Adapters for Claude Code, Codex, OpenCode and Antigravity, plus a generic command adapter.
   - L0–L2 probes, error classification, a TTL cache, and cooldowns.
   - `doctor` (human table) and `doctor --json` (capability manifest).
   - YAML config plus JSON Schema, and `resolve` (role → concrete agent/model with an explanation and fallback chain).
   - Zero-config mode.
   - Success criterion: it correctly flags the cases the CLIs themselves misreport (Claude rate-limit exit 0, agy unauthenticated, wrong model name).
2. **Phase 2: integrate rather than compete.** Publish the manifest format. Write a Crewplane provider-selection hook or a Vibe Kanban / Agent of Empires preflight script, or upstream adapters to them. Add an ACP client adapter for broad coverage.
3. **Phase 3 (optional): a thin runner.** Only if the integrations don't fit, add `run <pattern>`: a deterministic DAG executor with worktrees, gates and the role resolver. Keep it under roughly 2k lines and resist adding a UI.
4. **Differentiate on correctness, not breadth.** Version-pinned fixture tests of real CLI outputs, a public compatibility matrix, and fast releases when CLIs change are the actual moat.

## Caveats

- **This market moves monthly.**
  - Gemini CLI was replaced by Antigravity CLI for free and individual accounts on June 18, 2026; it remains available through paid API keys and enterprise.\[3\]\[44\]\[46\]\[66\]
  - OpenCode renamed commands between v1 and v2.\[40\]
  - Claude Code ships multiple releases a week.\[67\]
  
  Adapter maintenance is the main ongoing cost. Budget for it or don't start.
- **Live probes cost money and quota.** On subscription plans, a probe consumes part of the same rolling 5-hour window you want for real work.\[68\]\[69\] Make deep probes explicit.
- **Some findings come from secondary sources or issue reports, not official docs.** That includes the Codex doctor exit-code behavior, parts of the agy exit-code semantics, and the Claude headless bugs, some of which may already be fixed. Re-verify against the versions you pin.
- **Star counts and adoption figures** (e.g. Ruflo's 710,854 lifetime npm downloads and 45,956 GitHub stars as of May 7, 2026, per Jared Smith's deep dive on sublimecoding.com, and Crewplane's about 41 stars) are self-reported or point-in-time.
- **The strongest argument against building anything:** if you only run Claude Code, its native dynamic workflows plus subagents may be all you need. If you run several CLIs, Crewplane plus a 100-line preflight script may get you 80% of the way. Try both for a week before writing code.

## Sources

1. [\[BUG\] Headless \`claude -p\`: API errors (e.g. rate limit) return \`subtype:"success"\` + exit code 0 — error text only in \`result\`/stdout · Issue #79500 · anthropics/claude-code · GitHub](https://github.com/anthropics/claude-code/issues/79500)
2. [Non-interactive codex status (JSON/headless replacement for /status) · Issue #10233 · openai/codex · GitHub](https://github.com/openai/codex/issues/10233)
3. [Antigravity CLI: install agy, commands, models, flags](https://continuumcode.ai/guides/antigravity-cli/)
4. [Run agents in parallel - Claude Code Docs](https://code.claude.com/docs/en/agents)
5. [Orchestrate subagents at scale with dynamic workflows - Claude Code Docs](https://code.claude.com/docs/en/workflows)
6. [Agent of Empires - Agent of Empires](https://www.agent-of-empires.com/docs/)
7. [Vibe Kanban: Kanban Board for Coding Agents](https://ai-tldr.dev/tools/vibe-kanban/)
8. [vibe-kanban](https://virtuslab.com/blog/ai/vibe-kanban)
9. [Vibe Kanban Tool Review - by Eleanor Berger](https://elite-ai-assisted-coding.dev/p/vibe-kanban-tool-review)
10. [GitHub - bradAGI/awesome-cli-coding-agents: Curated directory of terminal-native AI coding agents and the harnesses that orchestrate them. Covers open-source tools (Pi, OpenCode, Aider, Goose), platform agents (Claude Code, Codex, Gemini CLI), parallel runners, autonomous loops, and agent infrastructure. · GitHub](https://github.com/bradAGI/awesome-cli-coding-agents)
11. [GitHub - johannesjo/parallel-code: Run Claude Code, Codex, and Gemini side by side — each in its own git worktree · GitHub](https://github.com/johannesjo/parallel-code)
12. [GitHub - andyrewlee/awesome-agent-orchestrators: List of agent orchestrators · GitHub](https://github.com/andyrewlee/awesome-agent-orchestrators)
13. [Crewplane Multi-Agent Orchestration Profile](https://openorchestrators.org/players/crewplane/)
14. [Agentmaxxing: Parallel Multi-CLI Orchestration with Codex CLI, Claude Code and Gemini CLI](https://codex.danielvaughan.com/2026/04/11/agentmaxxing-parallel-multi-cli-orchestration/)
15. [GitHub - Yeachan-Heo/oh-my-claudecode: Teams-first Multi-agent orchestration for Claude Code · GitHub](https://github.com/yeachan-heo/oh-my-claudecode)
16. [GitHub - nyldn/claude-octopus: Run multiple AI models against the same research, design, or coding task. Surface disagreements before you ship. · GitHub](https://github.com/nyldn/claude-octopus)
17. [GitHub - osanoai/multicli: Gemini, Codex, Claude, and OpenCode all in a single MCP tool for use by any coding agent. · GitHub](https://github.com/osanoai/multicli)
18. [GitHub - musistudio/claude-code-router: One local control plane for every AI agent: route across models, fuse new capabilities, orchestrate tools, and stay fully in control. · GitHub](https://github.com/musistudio/claude-code-router)
19. [Fallbacks (Provider Failover)](https://docs.litellm.ai/docs/proxy/reliability)
20. [Health Check Driven Routing](https://docs.litellm.ai/docs/proxy/health_check_routing)
21. [Model Fallbacks - Automatic Failover Between Models](https://openrouter.ai/docs/guides/routing/model-fallbacks)
22. [\[BUG\] claude -p exits 0 ("success") while agent loop is mid-task — last assistant message has stop\_reason tool\_use, tool results delivered, no continuation API call · Issue #74761 · anthropics/claude-code · GitHub](https://github.com/anthropics/claude-code/issues/74761)
23. [\`claude -p\` exits 1 with "No messages returned from query" when a \`rate\_limit\_event\` arrives after the result · Issue #103253 · anthropics/claude-code · GitHub](https://github.com/anthropics/claude-code/issues/103253)
24. [Developer commands](https://developers.openai.com/codex/cli/reference)
25. [WSL: exec\_command returns exit 1 with empty output; execution failure cause unknown · Issue #54831 · openai/codex · GitHub](https://github.com/openai/codex/issues/54831)
26. [Android Codex Remote pairing fails with both QR and manual code · Issue #56600 · openai/codex · GitHub](https://github.com/openai/codex/issues/56600)
27. [Developer commands](https://learn.chatgpt.com/docs/developer-commands)
28. [Troubleshooting guide](https://google-gemini.github.io/gemini-cli/docs/troubleshooting.html)
29. [Pin Gemini CLI auth type so api-proxy runs stop failing with exit 41 by Copilot · Pull Request #8916 · github/gh-aw-firewall · GitHub](https://github.com/github/gh-aw-firewall/pull/8916)
30. [CLI reference - Claude Wiki](https://claude-wiki.com/cli-reference.html)
31. [\[Bug\]: Claude Settings reports authenticated while the launched CLI is logged out · Issue #7690 · pingdotgg/t3code · GitHub](https://github.com/pingdotgg/t3code/issues/7690)
32. [Claude Code changelog - Claude Code Docs](https://code.claude.com/docs/en/changelog)
33. [Claude Code — Delegate coding to Claude Code CLI (features, PRs)](https://hermes-agent.nousresearch.com/docs/user-guide/skills/bundled/autonomous-ai-agents/autonomous-ai-agents-claude-code)
34. [Claude Code CLI: The Complete Guide — Hooks, MCP, Skills](https://blakecrosley.com/guides/claude-code)
35. [Probe claude's login state with claude auth status by lezli01 · Pull Request #460 · lezli01/vincent · GitHub](https://github.com/lezli01/vincent/pull/460)
36. [Short-lived commands (\`claude auth status\`, \`claude --bg …\`) start an OAuth refresh at init and exit without saving it, leaving the profile with a spent refresh token · Issue #95822 · anthropics/claude-code · GitHub](https://github.com/anthropics/claude-code/issues/95822)
37. [Developer commands](https://www.codex-docs.com/en/docs/developer-commands)
38. [How to check Codex login status](https://www.simplified.guide/codex/login-status-check)
39. [Commands](https://opencode.ai/v2/docs/cli/commands/)
40. [OpenCode CLI Cheat Sheet (PDF): OpenCode 2 Commands](https://computingforgeeks.com/opencode-cli-cheat-sheet/)
41. [Headless mode - Antigravity CLI | Google Antigravity Docs](https://antigravity.google/docs/cli/headless/)
42. [Antigravity CLI: A Hands-On Guide to Google's Terminal Coding Agent - DEV Community](https://dev.to/arindam_1729/antigravity-cli-a-hands-on-guide-to-googles-terminal-coding-agent-5bc7)
43. [Structured JSON Output · Issue #8022 · google-gemini/gemini-cli · GitHub](https://github.com/google-gemini/gemini-cli/issues/8022)
44. [Is Gemini CLI Deprecated? Shutdown Date and What to Use Now](https://www.aibuilderclub.com/blog/google-kills-gemini-cli-june-18-2026)
45. [sdk headless](https://docs.anthropic.com/en/docs/claude-code/sdk/sdk-headless)
46. [Headless Mode in 5 Agents: claude -p, codex exec, gemini -p, copilot -p, opencode run](https://dev.to/gabrielanhaia/headless-mode-in-5-agents-claude-p-codex-exec-gemini-p-copilot-p-opencode-run-4816)
47. [Non-interactive mode · Codex Docs](https://docs.onlinetool.cc/codex/docs/exec.html)
48. [Non-interactive mode](https://dev-docs.moodybeard.com/en/codex/noninteractive/)
49. [run-opencode: Invokes opencode CLI as a headless executor for delegated coding tasks with NDJSON output](https://skills.rest/skill/run-opencode-gabesan21)
50. [Claude Code Rate Limit Reached: Retry Setup for Cron Jobs - DEV Community](https://dev.to/aobaworks/claude-code-rate-limit-reached-retry-setup-for-cron-jobs-1ioi)
51. [Anthropic Claude 3.5 logo](<https://techcrunch.com/2025/07/28/anthropic-unveils-new-rate-limits-to-curb-claude-code-power-users).>)
52. [\[BUG\] Claude Code "Usage limit reached" with 0% on 5-hour window and 5% weekly usage · Issue #54177 · anthropics/claude-code · GitHub](https://github.com/anthropics/claude-code/issues/54177)
53. [Samuel Lawrentz](https://samuellawrentz.com/blog/claude-code-usage-limit-wrap-up/)
54. [How to see your Claude Code and Codex usage limits before you hit them (6 open-source tools) - DEV Community](https://dev.to/aaron_w_ef987250d3531be6d/how-to-see-your-claude-code-and-codex-usage-limits-before-you-hit-them-6-open-source-tools-5gl9)
55. [Claude Code Rate Limit Exceeded: What It Means and How to Fix It - Usagebar Blog](https://usagebar.com/blog/claude-code-rate-limit-exceeded)
56. [Show GN: Claude Code Status Bar - 컨텍스트, 비용, rate limit을 항상 보여주는 status line](https://news.hada.io/topic?id=28227)
57. [Health Checks and Reliability](https://deepwiki.com/mikeplavsky/litellm/3.4-health-checks-and-reliability)
58. [LiteLLM Proxy Errors: Causes & Fixes](https://www.swfte.com/errors/litellm-errors)
59. [Initialization and Session Management](https://deepwiki.com/zed-industries/agent-client-protocol/2.1-initialization-and-session-management)
60. [Agent Client Protocol — Grokipedia](https://grokipedia.com/page/Agent_Client_Protocol)
61. [agent-workflow-cli · GitHub Topics · GitHub](https://github.com/topics/agent-workflow-cli)
62. [Antigravity CLI Deep Dive: Google's Go-Based Terminal Agent (May 2026)](https://agentpedia.codes/blog/antigravity-cli-deep-dive)
63. [crewplane · PyPI](https://pypi.org/project/crewplane/0.3.8/)
64. [agent-of-empires/agent-of-empires — Hermes Agent Multi-Agent & Orchestration](https://hermesatlas.com/projects/agent-of-empires/agent-of-empires)
65. [feat(cli): add codex doctor diagnostics by fcoury-oai · Pull Request #22336 · openai/codex · GitHub](https://github.com/openai/codex/pull/22336)
66. [Antigravity CLI Prompting Guide: Google's Gemini CLI Replacement (2026)](https://sureprompts.com/blog/antigravity-cli-prompting-guide)
67. [Claude Code Cheat Sheet 2026: Shortcuts, Commands & Flags](https://blakecrosley.com/guides/claude-code-cheatsheet)
68. [Claude Code Rate Limits: 5-Hour Caps, 429 Errors, and Fixes](https://fast.io/resources/claude-code-rate-limit/)
69. [DEV Community](https://dev.to/portkey/everything-we-know-about-claude-code-limits-3984)
