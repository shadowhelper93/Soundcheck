# Soundcheck for Orca

An [Orca](https://www.onorca.dev) plugin that runs Soundcheck's preflight check
(`soundcheck doctor --json`) on the coding agent CLIs installed on your machine
and shows the result in a sidebar panel.

## What it adds

- **Soundcheck** sidebar panel with a *Run preflight check* button, diagnostic dashboard (adapters, health tiers, and zero-token role assignments), and collapsible raw JSON output.
- **Soundcheck: Run preflight check** command in the command palette (`doctor.run`).

## Requirements

The `soundcheck` binary must be on your `PATH`, or configured via the `SOUNDCHECK_PATH` or `SOUNDCHECK_BIN` environment variable. See the [Soundcheck installation steps](../../README.md#installation).

## Permissions

This plugin uses the `trusted-worker` execution profile: its worker runs on your
machine with your permissions. It runs one fixed command, `soundcheck doctor --json`,
with no shell and no network access of its own. It requests no capabilities.

## Files

| Path | Purpose |
| --- | --- |
| `orca-plugin.json` | Plugin manifest (format 2) |
| `worker/main.mjs` | Runs `soundcheck doctor --json`, keeps the latest report, answers the panel |
| `worker/main.test.mjs` | Automated unit tests for worker logic |
| `panel/index.html`, `panel/app.js` | Panel UI; renders diagnostic health dashboard and raw report |

## Testing & Validation

Validate the plugin manifest:
```bash
orca plugin validate editors/orca
```

Run plugin tests with Orca or Node:
```bash
orca plugin test editors/orca
# or
node --test editors/orca/worker/main.test.mjs
```

## Publishing

Orca lists a plugin from its own GitHub repository root. This plugin lives in a
subdirectory of the Soundcheck repository, so to list it in the Orca directory it
must be published as its own repository (copy this folder to the root of a new
repo, add the `orca-plugin` topic, and push a `vX.Y.Z` tag), or added to a catalog
with `"path": "editors/orca"`.
