# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
