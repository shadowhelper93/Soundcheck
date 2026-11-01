# Soundcheck VS Code Extension

Official Visual Studio Code & Antigravity IDE extension for **Soundcheck**: deterministic preflight health checks, role resolution, quirk protection, and auto-failover for AI coding agents.

---

## Features

- **Activity Bar View**:
  - **CLI Adapters**: Real-time status for Claude Code, OpenAI Codex, OpenCode, Antigravity CLI, Gemini CLI, and Copilot CLI with tier meters `[L0:✓ L1:✓ L2:✓ L3:-]`.
  - **Resolved Roles**: Live mapping of `Primary`, `Subagent`, `Auxiliary`, and `Reviewer` roles with distinct-vendor enforcement.
  - **Quick Actions**: One-click actions for Doctor checks, role resolution, and headless prompt runs.
- **Status Bar Integration**:
  - Shows current system health (`$(shield) Soundcheck: 6/6 OK` or `$(alert) Soundcheck: DEGRADED`).
  - Clicking opens a quick menu of actions.
- **Headless Prompt Execution (`soundcheck.exec`)**:
  - Runs prompts through the resolved healthy agent.
  - Detects rate limits or false exit 0 lies via deterministic quirk regexes.
  - Automatically cascades down fallback candidates until success.
- **Embedded Web Dashboard (`soundcheck.openDashboard`)**:
  - Launches and embeds the full Soundcheck React dashboard inside an editor tab.
- **Interactive PTY Passthrough (`soundcheck.openTerminal`)**:
  - Launches raw interactive agent sessions inside the VS Code terminal.

---

## Commands

The extension detects the Soundcheck CLI on startup. If it is missing, choose **Install CLI**, or run **Soundcheck: Install Soundcheck CLI** from the Command Palette. Select the operating system of the VS Code extension host; the installer downloads the latest published release and verifies its SHA-256 checksum before installation. A release must exist for that platform.

| Command | Title |
|---|---|
| `soundcheck.doctor` | Run Doctor Preflight Check |
| `soundcheck.resolve` | Resolve Agent Roles |
| `soundcheck.exec` | Execute Prompt with Auto-Failover |
| `soundcheck.fix` | Remediate Broken Adapter |
| `soundcheck.check` | Discover Alternative Fallbacks |
| `soundcheck.openDashboard` | Open Visual Dashboard |
| `soundcheck.openTerminal` | Open Interactive Adapter Terminal |
| `soundcheck.refresh` | Refresh Health Status |

---

## Configuration

| Setting | Default | Description |
|---|---|---|
| `soundcheck.binaryPath` | `""` | Path to custom `soundcheck` binary. Falls back to workspace `bin/soundcheck` or `$PATH`. |
| `soundcheck.autoDoctorOnStartup` | `true` | Automatically run doctor preflight on startup. |
| `soundcheck.serverPort` | `3000` | Local port for the web dashboard. |
