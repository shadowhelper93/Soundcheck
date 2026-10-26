#!/usr/bin/env node
/**
 * Capture deterministic golden fixtures from legacy TypeScript Soundcheck logic.
 * Produces:
 *   - testdata/golden/classify.json
 *   - testdata/golden/resolve.json
 *   - testdata/golden/manifest.json
 *   - testdata/golden/host_probe.json
 */

import fs from 'node:fs';
import path from 'node:path';
import os from 'node:os';
import { fileURLToPath } from 'node:url';
import { exec, spawnSync } from 'node:child_process';
import { promisify } from 'node:util';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const REPO_ROOT = path.resolve(__dirname, '..');
const execAsync = promisify(exec);

// If run under raw Node without tsx, re-exec via tsx
if (!process.env.__TSX_BOOTSTRAPPED__ && !process.argv.includes('--tsx-internal')) {
  try {
    // Check if TS extension resolution works
    await import(path.join(REPO_ROOT, 'legacy/src/core/adapters/adapterRegistry.ts'));
  } catch {
    const result = spawnSync('npx', ['tsx', __filename, '--tsx-internal', ...process.argv.slice(2)], {
      stdio: 'inherit',
      env: { ...process.env, __TSX_BOOTSTRAPPED__: '1' },
      cwd: REPO_ROOT
    });
    process.exit(result.status ?? 0);
  }
}

// Freeze Date.now for deterministic fixture values
const FIXED_TIMESTAMP_MS = 1730000000000;
const FIXED_ISO_STRING = '2026-10-26T00:00:00.000Z';
Date.now = () => FIXED_TIMESTAMP_MS;

// Dynamic imports of legacy TS modules
const { classifyError, INITIAL_ADAPTERS } = await import(
  path.join(REPO_ROOT, 'legacy/src/core/adapters/adapterRegistry.ts')
);
const { KNOWN_QUIRKS_FIXTURES } = await import(
  path.join(REPO_ROOT, 'legacy/src/core/fixtures/knownQuirksFixtures.ts')
);
const { resolveRoles, DEFAULT_CONFIG } = await import(
  path.join(REPO_ROOT, 'legacy/src/core/engine/resolver.ts')
);
const { generateCapabilityManifest } = await import(
  path.join(REPO_ROOT, 'legacy/src/core/engine/manifestGenerator.ts')
);

// 1. Parse agent-doctor.yaml
function loadAgentDoctorConfig() {
  const yamlPath = path.join(REPO_ROOT, 'agent-doctor.yaml');
  if (!fs.existsSync(yamlPath)) {
    return DEFAULT_CONFIG;
  }
  const content = fs.readFileSync(yamlPath, 'utf8');
  const config = {
    version: 1,
    roles: { primary: [], subagent: [], auxiliary: [], reviewer: [] },
    policy: {
      require_distinct_vendor: ['primary', 'reviewer'],
      max_cost_usd_per_run: 5.0,
      probe: { deep: false, ttl_minutes: 30 }
    }
  };

  let currentRole = null;
  for (const rawLine of content.split('\n')) {
    const line = rawLine.trim();
    if (!line || line.startsWith('#')) continue;
    if (line.startsWith('primary:')) currentRole = 'primary';
    else if (line.startsWith('subagent:')) currentRole = 'subagent';
    else if (line.startsWith('auxiliary:')) currentRole = 'auxiliary';
    else if (line.startsWith('reviewer:')) currentRole = 'reviewer';
    else if (line.startsWith('- ') && currentRole) {
      const val = line.slice(2).trim().replace(/^["']|["']$/g, '');
      config.roles[currentRole].push(val);
    } else if (line.startsWith('require_distinct_vendor:')) {
      const match = line.match(/\[(.*?)\]/);
      if (match) {
        config.policy.require_distinct_vendor = match[1].split(',').map((s) => s.trim().replace(/^["']|["']$/g, ''));
      }
    } else if (line.startsWith('max_cost_usd_per_run:')) {
      config.policy.max_cost_usd_per_run = parseFloat(line.split(':')[1].trim());
    } else if (line.startsWith('deep:')) {
      config.policy.probe.deep = line.split(':')[1].trim() === 'true';
    } else if (line.startsWith('ttl_minutes:')) {
      config.policy.probe.ttl_minutes = parseInt(line.split(':')[1].trim(), 10);
    }
  }
  return config;
}

// 2. Classify Goldens
function captureClassifyGoldens() {
  return KNOWN_QUIRKS_FIXTURES.map((fixture) => {
    let parsed = null;
    try {
      const trimmed = fixture.simulatedStdout.trim();
      if (trimmed.startsWith('{') || trimmed.startsWith('[')) {
        parsed = JSON.parse(fixture.simulatedStdout);
      }
    } catch {
      parsed = null;
    }

    const classification = classifyError(
      fixture.agentId,
      fixture.simulatedStdout,
      fixture.simulatedStderr,
      fixture.simulatedExitCode,
      parsed
    );

    return {
      id: fixture.id,
      agentId: fixture.agentId,
      name: fixture.name,
      docIssue: fixture.docIssue,
      category: fixture.category,
      description: fixture.description,
      simulatedExitCode: fixture.simulatedExitCode,
      simulatedStdout: fixture.simulatedStdout,
      simulatedStderr: fixture.simulatedStderr,
      naiveOutcome: fixture.naiveOutcome,
      soundcheckOutcome: fixture.soundcheckOutcome,
      expectedClass: fixture.expectedClass,
      explanation: fixture.explanation,
      classification: {
        errorClass: classification.errorClass,
        reason: classification.reason,
        caughtLie: classification.caughtLie
      }
    };
  });
}

// 3. Resolve Goldens
function captureResolveGoldens(config) {
  return resolveRoles(config, INITIAL_ADAPTERS);
}

// 4. Manifest Goldens
function captureManifestGoldens(resolvedRoles) {
  const manifest = generateCapabilityManifest(INITIAL_ADAPTERS, resolvedRoles, {
    platform: 'darwin',
    nodeVersion: 'v22.0.0',
    gitWorktree: true
  });
  manifest.generatedAt = FIXED_ISO_STRING;
  return manifest;
}

// 5. Host Probe Goldens
const KNOWN_BINARIES = [
  { id: 'claude', name: 'Claude Code', cmd: 'claude' },
  { id: 'codex', name: 'OpenAI Codex', cmd: 'codex' },
  { id: 'opencode', name: 'OpenCode', cmd: 'opencode' },
  { id: 'agy', name: 'Antigravity CLI', cmd: 'agy' },
  { id: 'gemini', name: 'Gemini CLI (Legacy)', cmd: 'gemini' },
  { id: 'copilot', name: 'GitHub Copilot CLI', cmd: 'copilot' },
  { id: 'cursor', name: 'Cursor Agent', cmd: 'cursor-agent' },
  { id: 'aider', name: 'Aider', cmd: 'aider' }
];

async function findBinaryPath(cmdName) {
  const isWindows = process.platform === 'win32';
  try {
    if (isWindows) {
      const { stdout } = await execAsync(`where.exe ${cmdName} 2>nul || where.exe ${cmdName}.cmd 2>nul || true`);
      return stdout.trim().split(/\r?\n/)[0] || '';
    }
    const { stdout } = await execAsync(`which ${cmdName} 2>/dev/null || true`);
    return stdout.trim();
  } catch {
    return '';
  }
}

async function captureHostProbeGoldens() {
  const isWindows = process.platform === 'win32';
  const isMac = process.platform === 'darwin';
  const isLinux = process.platform === 'linux';

  const binaryResults = {};
  for (const b of KNOWN_BINARIES) {
    try {
      const binPath = await findBinaryPath(b.cmd);
      if (binPath) {
        try {
          const { stdout: verOut } = await execAsync(`"${binPath}" --version`, { timeout: 3000 });
          binaryResults[b.id] = { installed: true, path: binPath, version: verOut.trim().split('\n')[0] || 'Detected' };
        } catch {
          binaryResults[b.id] = { installed: true, path: binPath, version: 'Installed (semver check skipped)' };
        }
      } else {
        binaryResults[b.id] = { installed: false };
      }
    } catch (err) {
      binaryResults[b.id] = { installed: false, error: err.message };
    }
  }

  const envPresence = {
    ANTHROPIC_API_KEY: !!process.env.ANTHROPIC_API_KEY,
    OPENAI_API_KEY: !!process.env.OPENAI_API_KEY,
    CODEX_API_KEY: !!process.env.CODEX_API_KEY,
    GEMINI_API_KEY: !!process.env.GEMINI_API_KEY,
    OPENROUTER_API_KEY: !!process.env.OPENROUTER_API_KEY,
    GITHUB_TOKEN: !!process.env.GITHUB_TOKEN,
    GOOGLE_GEMINI_BASE_URL: !!process.env.GOOGLE_GEMINI_BASE_URL
  };

  const homeDir = os.homedir();
  const appData = process.env.APPDATA || path.join(homeDir, 'AppData', 'Roaming');
  const localAppData = process.env.LOCALAPPDATA || path.join(homeDir, 'AppData', 'Local');

  const checkAnyPathExists = (paths) => paths.some((p) => {
    try {
      return fs.existsSync(p);
    } catch {
      return false;
    }
  });

  const configDirChecks = {
    claude: checkAnyPathExists([
      path.join(homeDir, '.claude'),
      path.join(homeDir, '.config', 'claude')
    ]),
    codex: checkAnyPathExists([
      path.join(homeDir, '.codex'),
      path.join(homeDir, 'Library', 'Application Support', 'OpenAI'),
      path.join(appData, 'OpenAI'),
      path.join(homeDir, '.config', 'codex')
    ]),
    opencode: checkAnyPathExists([
      path.join(homeDir, '.config', 'opencode'),
      path.join(homeDir, '.opencode'),
      path.join(appData, 'opencode')
    ]),
    antigravity: checkAnyPathExists([
      path.join(homeDir, '.gemini', 'antigravity-cli'),
      path.join(homeDir, '.antigravity'),
      path.join(homeDir, 'Library', 'Application Support', 'Google', 'Antigravity'),
      path.join(localAppData, 'Google', 'Antigravity')
    ])
  };

  let gitWorktreeAvailable = false;
  try {
    const { stdout } = await execAsync('git worktree list');
    gitWorktreeAvailable = !!stdout;
  } catch {
    gitWorktreeAvailable = false;
  }

  return {
    platform: process.platform,
    osName: isMac ? 'macOS (Darwin)' : isWindows ? 'Windows' : isLinux ? 'Linux' : process.platform,
    arch: os.arch(),
    homeDir: homeDir,
    nodeVersion: process.version,
    binaries: binaryResults,
    envPresence,
    configDirChecks,
    gitWorktreeAvailable,
    timestamp: FIXED_ISO_STRING
  };
}

async function main() {
  const goldenDir = path.join(REPO_ROOT, 'testdata', 'golden');
  fs.mkdirSync(goldenDir, { recursive: true });

  console.log('Capturing legacy golden fixtures...');

  // 1. Classify
  const classifyGoldens = captureClassifyGoldens();
  const classifyPath = path.join(goldenDir, 'classify.json');
  fs.writeFileSync(classifyPath, JSON.stringify(classifyGoldens, null, 2) + '\n', 'utf8');
  console.log(`✓ Wrote ${classifyPath} (${classifyGoldens.length} fixtures)`);

  // 2. Resolve
  const config = loadAgentDoctorConfig();
  const resolveGoldens = captureResolveGoldens(config);
  const resolvePath = path.join(goldenDir, 'resolve.json');
  fs.writeFileSync(resolvePath, JSON.stringify(resolveGoldens, null, 2) + '\n', 'utf8');
  console.log(`✓ Wrote ${resolvePath} (${Object.keys(resolveGoldens).length} roles)`);

  // 3. Manifest
  const manifestGoldens = captureManifestGoldens(resolveGoldens);
  const manifestPath = path.join(goldenDir, 'manifest.json');
  fs.writeFileSync(manifestPath, JSON.stringify(manifestGoldens, null, 2) + '\n', 'utf8');
  console.log(`✓ Wrote ${manifestPath}`);

  // 4. Host Probe
  const hostProbeGoldens = await captureHostProbeGoldens();
  const hostProbePath = path.join(goldenDir, 'host_probe.json');
  fs.writeFileSync(hostProbePath, JSON.stringify(hostProbeGoldens, null, 2) + '\n', 'utf8');
  console.log(`✓ Wrote ${hostProbePath}`);

  // Validate JSON integrity
  for (const p of [classifyPath, resolvePath, manifestPath, hostProbePath]) {
    const parsed = JSON.parse(fs.readFileSync(p, 'utf8'));
    if (!parsed) throw new Error(`Empty golden JSON: ${p}`);
  }
  console.log('All golden fixtures captured and validated successfully.');
}

main().catch((err) => {
  console.error('Failed capturing goldens:', err);
  process.exit(1);
});
