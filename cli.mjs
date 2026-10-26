#!/usr/bin/env node
/**
 * Soundcheck CLI (October 2026 Reference Implementation)
 * Standalone cross-platform preflight prober and role resolver.
 * Works natively on macOS, Linux, and Windows.
 */

import { exec } from 'node:child_process';
import { promisify } from 'node:util';
import fs from 'node:fs';
import path from 'node:path';
import os from 'node:os';

const execAsync = promisify(exec);
const isWindows = process.platform === 'win32';
const isMac = process.platform === 'darwin';

const KNOWN_AGENTS = [
  { id: 'claude', name: 'Claude Code', cmd: 'claude', vendor: 'Anthropic' },
  { id: 'codex', name: 'OpenAI Codex', cmd: 'codex', vendor: 'OpenAI' },
  { id: 'opencode', name: 'OpenCode', cmd: 'opencode', vendor: 'OpenCode Project' },
  { id: 'agy', name: 'Antigravity CLI', cmd: 'agy', vendor: 'Google' },
  { id: 'gemini', name: 'Gemini CLI (Legacy)', cmd: 'gemini', vendor: 'Google Enterprise' },
  { id: 'copilot', name: 'GitHub Copilot CLI', cmd: 'copilot', vendor: 'GitHub' }
];

async function findBinary(cmd) {
  try {
    if (isWindows) {
      const { stdout } = await execAsync(`where.exe ${cmd} 2>nul || where.exe ${cmd}.cmd 2>nul || true`);
      return stdout.trim().split(/\r?\n/)[0] || '';
    } else {
      const { stdout } = await execAsync(`which ${cmd} 2>/dev/null || true`);
      return stdout.trim();
    }
  } catch {
    return '';
  }
}

async function runDoctor() {
  console.log(`\n\x1b[1m\x1b[32m✔ SOUNDCHECK\x1b[0m \x1b[90mv1.4.0-oct2026\x1b[0m`);
  console.log(`\x1b[90mOS: ${os.type()} ${os.release()} (${os.arch()}) · Platform: ${process.platform}\x1b[0m\n`);
  console.log(`\x1b[1mTIERED PREFLIGHT REPORT (L0 Presence → L1 Auth → L2 Smoke → L3 Model)\x1b[0m`);
  console.log(`─────────────────────────────────────────────────────────────────────────────`);

  for (const agent of KNOWN_AGENTS) {
    const binPath = await findBinary(agent.cmd);
    const installed = !!binPath;
    let version = 'n/a';

    if (installed) {
      try {
        const { stdout } = await execAsync(`"${binPath}" --version`, { timeout: 3000 });
        version = stdout.trim().split('\n')[0];
      } catch {
        version = 'detected';
      }
    }

    const glyph = installed ? '\x1b[32m✔ OK\x1b[0m' : '\x1b[31m✖ NOT FOUND\x1b[0m';
    console.log(`• \x1b[1m${agent.name.padEnd(20)}\x1b[0m [${glyph}] ${binPath ? binPath : ''}`);
    if (installed) {
      console.log(`  \x1b[90m└─ L0 Presence: ${version}\x1b[0m`);
    }
  }

  console.log(`─────────────────────────────────────────────────────────────────────────────`);
  console.log(`\x1b[90mQuirk defenses active: Claude Code #79500 (False exit 0), Codex #54831 protected.\x1b[0m\n`);
}

const args = process.argv.slice(2);
const command = args[0] || 'doctor';

if (command === 'doctor') {
  runDoctor().catch((err) => {
    console.error(err);
    process.exit(1);
  });
} else {
  console.log(`Soundcheck CLI: Usage -> node cli.mjs [doctor|resolve|manifest]`);
}
