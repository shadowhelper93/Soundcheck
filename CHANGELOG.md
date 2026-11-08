# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [2.0.1] - 2026-11-05

### Added
- **VS Code Extension (v2.0.1)**:
  - Automatic detection of the `soundcheck` CLI on startup, with an **Install CLI** action.
  - Installer downloads the latest release and verifies its SHA-256 checksum before installing.
  - Extension icons.
  - Marketplace listing: [SinaMashini.soundcheck](https://marketplace.visualstudio.com/items?itemName=SinaMashini.soundcheck).
- **CLI**: ASCII banner on `soundcheck --help` and `soundcheck doctor` for wide terminals.
- **Agent skill**: `skills/soundcheck` for using Soundcheck from coding agents.

### Changed
- Version bumped to 2.0.1 across the CLI, HTTP server, and web UI.

### Fixed
- `soundcheck upgrade` targets the correct repository and handles releases that are not yet published.
- CI: fixed the budget build step, golangci-lint version, and Windows test compatibility.

---

## [2.1.0] - 2026-11-08

### Added
- **Orca Plugin (`orca-plugin.json` & `editors/orca`)**:
  - Root `orca-plugin.json` (format 2, `trusted-worker`) enabling direct installation via Git URL (`https://github.com/shadowhelper93/Soundcheck`) and native folder picker ("Install from Folder...").
  - Sidebar panel with live diagnostic dashboard (online agents, quirks protected, CLI adapter status badges, L0-L3 tier pills, zero-token role assignments) and collapsible raw report details.
  - Command palette entry `Soundcheck: Run preflight check` (`doctor.run`).
  - Configurable binary path resolution via `SOUNDCHECK_BIN` or `SOUNDCHECK_PATH` environment variables.
  - Automated unit test suite (`editors/orca/worker/main.test.mjs`) for doctor output parsing, process exit handling, and worker lifecycle.
  - CI workflow integration for automated Orca plugin testing.
- **README sections** for the VS Code extension and Orca plugin.

### Changed
- Version bumped to 2.1.0 across CLI, HTTP server, Web UI, and Orca plugin manifests.

---

## [2.0.0] - 2026-10-31

### Added
- **Live Model Discovery (`modelsCommand`)**:
  - Declarative discovery command support in adapter specifications (e.g. `codex debug models`).
  - Dynamic extraction of active account model slugs directly from CLI output.
  - Zero price fabrication: newly discovered models leave pricing unverified/unknown instead of inventing numbers.
- **Dynamic Preflight Validation (`POST /api/v1/config/validate`)**:
  - Validates role candidates against real probed host adapters before applying changes.
  - Returns descriptive rejection reasons (binary missing, unauthenticated, model not offered by account).
- **Atomic Configuration Persistence (`PUT /api/v1/config`)**:
  - Atomic persistence to disk via temporary file creation, `0600` permissions, and rename.
  - Rejects invalid schemas and broken rules with clear HTTP 400 JSON payloads.
  - `GET /api/v1/config` exposes active file location in `X-Config-Path` header.
- **Persistent Left Sidebar Navigation & Appearance Settings**:
  - Modern left sidebar layout (`01 Doctor`, `02 Resolver`, `03 DAG Studio`, `04 Quirk Lab`, `05 Manifest`, `06 Settings`).
  - Font detection via `document.fonts.check` for coding typefaces (`Fira Code`, `JetBrains Mono`, `SF Mono`, `Cascadia Code`, `Menlo`, `Monaco`, `Consolas`, `Source Code Pro`).
  - Independent typography and size controls for the live PTY terminal runner.
- **Automated First-Run macOS Smoke Test**:
  - `scripts/smoke-macos.sh` tests build, server lifecycle on dynamic ports, health, doctor, and config validation end-to-end.
- **Multi-Language Documentation**:
  - Localized README translations in Spanish (`docs/i18n/README.es.md`), French (`docs/i18n/README.fr.md`), and Farsi (`docs/i18n/README.fa.md`).
- **Contributor & Agent Architecture Manual (`CLAUDE.md`)**:
  - Complete architecture guidelines, lane boundaries, purity rules, and development command reference.

### Changed
- **Official Model Catalog Alignment**:
  - Removed retired models (`claude-3-5-sonnet`, `claude-3-7-sonnet`, `claude-3-5-haiku`, `claude-opus`, `gemini-2.0-flash`, `gemini-1.5-pro`).
  - Mapped verified current models: `claude-sonnet-5-5`, `claude-haiku-5-5`, `claude-opus-5-5`, `claude-fable-5-1`, `gemini-2.5-pro`, `gemini-2.5-flash`, `gpt-4o`, `gpt-4o-mini`, `o3-mini`.
  - Regenerated test fixtures and golden parity files (`resolve.json`, `manifest.json`, `initial_adapters.json`).
- **Cost-Aware Health Probes**:
  - Standard `soundcheck doctor` caps probing at L1 (presence + authentication) to avoid unexpected token expenditure.
  - Deep probing (`soundcheck doctor --deep`) invokes actual L2/L3 model turns and prints a prominent cost warning to stderr.
  - Failing L2/L3 probes classified as `DEGRADED <reason>` via `domain.EvalHealth` rather than false green `OK`.
- **UI Dist Refresh**:
  - Rebuilt production UI bundle with inline SVG favicon, eliminating 404 console errors on initial load.

### Fixed
- Intercepted fake exit-0 CLI rate limit quirks across headless execution runners.
- Enforced SHA-256 checksum verification during binary self-update (`soundcheck upgrade`).
- Made `scripts/install.sh` fail closed on checksum mismatches.
