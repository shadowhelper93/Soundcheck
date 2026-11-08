# Soundcheck for Orca

An [Orca](https://www.onorca.dev) plugin that runs Soundcheck's preflight check
(`soundcheck doctor --json`) on the coding agent CLIs installed on your machine
and shows the result in a sidebar panel.

## What it adds

- **Soundcheck** sidebar panel with a *Run preflight check* button, diagnostic dashboard (adapters, health tiers, and zero-token role assignments), and collapsible raw JSON output.
- **Soundcheck: Run preflight check** command in the command palette (`doctor.run`).

## Installation

### Method 1: Install from Git
In Orca, go to **Settings → Plugins → Install from Git...** and enter:
```text
https://github.com/shadowhelper93/Soundcheck
```

### Method 2: Install from Folder
In Orca, go to **Settings → Plugins → Install from Folder...** and select either the repository root directory or `editors/orca`.

### Method 3: Development Mode
From the Soundcheck repository root:
```bash
orca plugin dev watch .
```

## Requirements

The `soundcheck` binary must be on your `PATH`, or configured via the `SOUNDCHECK_PATH` or `SOUNDCHECK_BIN` environment variable. See the [Soundcheck installation steps](../../README.md#installation).

## Permissions

This plugin uses the `trusted-worker` execution profile: its worker runs on your
machine with your permissions. It runs one fixed command, `soundcheck doctor --json`,
with no shell and no network access of its own. It requests no capabilities.

## Files

| Path | Purpose |
| --- | --- |
| `orca-plugin.json` (root) | Root plugin manifest pointing to `editors/orca` |
| `editors/orca/orca-plugin.json` | Subdirectory plugin manifest |
| `editors/orca/worker/main.mjs` | Runs `soundcheck doctor --json`, keeps the latest report, answers the panel |
| `editors/orca/worker/main.test.mjs` | Automated unit tests for worker logic |
| `editors/orca/panel/index.html`, `panel/app.js` | Panel UI; renders diagnostic health dashboard and raw report |

## Testing & Validation

Validate the plugin manifest:
```bash
orca plugin validate .
orca plugin validate editors/orca
```

Run plugin tests with Orca or Node:
```bash
orca plugin test .
orca plugin test editors/orca
# or
node --test editors/orca/worker/main.test.mjs
```
