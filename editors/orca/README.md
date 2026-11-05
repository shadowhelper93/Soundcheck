# Soundcheck for Orca

An [Orca](https://www.onorca.dev) plugin that runs Soundcheck's preflight check
(`soundcheck doctor --json`) on the coding agent CLIs installed on your machine
and shows the result in a sidebar panel.

## What it adds

- **Soundcheck** sidebar panel with a *Run preflight check* button and the last result.
- **Soundcheck: Run preflight check** command in the command palette.

## Requirements

The `soundcheck` binary must be on your `PATH`. See the
[Soundcheck installation steps](../../README.md#installation).

## Permissions

This plugin uses the `trusted-worker` execution profile: its worker runs on your
machine with your permissions. It runs one fixed command, `soundcheck doctor --json`,
with no shell and no network access of its own. It requests no capabilities.

## Files

| Path | Purpose |
| --- | --- |
| `orca-plugin.json` | Plugin manifest (format 2) |
| `worker/main.mjs` | Runs `soundcheck doctor --json`, keeps the latest report, answers the panel |
| `panel/index.html`, `panel/app.js` | Panel UI; renders output as text only |

## Publishing

Orca lists a plugin from its own GitHub repository root. This plugin lives in a
subdirectory of the Soundcheck repository, so to list it in the Orca directory it
must be published as its own repository (copy this folder to the root of a new
repo, add the `orca-plugin` topic, and push a `vX.Y.Z` tag), or added to a catalog
with `"path": "editors/orca"`.
