// Soundcheck worker for Orca.
//
// Runs the locally installed `soundcheck` binary with `doctor --json` and keeps
// the latest report in memory. The panel (panel/index.html) asks for a run or
// reads the last result through the handlers registered below.
//
// Note: this plugin runs as trusted-worker, i.e. full access on the user's
// machine. It executes only `soundcheck doctor --json`, with no shell.
import { execFile } from 'node:child_process'
import { promisify } from 'node:util'

const execFileAsync = promisify(execFile)

const SOUNDCHECK_BIN = 'soundcheck'
const DOCTOR_TIMEOUT_MS = 120_000
const MAX_OUTPUT_BYTES = 4 * 1024 * 1024

let latest = null

// Parses the doctor output. Returns { report } when it is JSON, else { raw }.
export function parseDoctorOutput(stdout) {
  const text = String(stdout ?? '').trim()
  if (text === '') return { raw: '' }
  try {
    return { report: JSON.parse(text) }
  } catch {
    return { raw: text }
  }
}

// Builds the result the panel shows. `failure` is the error thrown by execFile,
// if any: soundcheck may exit non-zero when an adapter is unhealthy but still
// print a report, so the stdout is kept whenever it is present.
export function buildResult({ stdout, failure, startedAt, finishedAt }) {
  const parsed = parseDoctorOutput(stdout ?? failure?.stdout)
  const result = { startedAt, finishedAt, ...parsed }
  if (failure) {
    result.ok = false
    result.exitCode = typeof failure.code === 'number' ? failure.code : null
    result.error = failure.code === 'ENOENT'
      ? `"${SOUNDCHECK_BIN}" was not found on PATH. Install Soundcheck first.`
      : failure.killed
        ? 'soundcheck doctor timed out.'
        : String(failure.message ?? failure)
  } else {
    result.ok = true
    result.exitCode = 0
  }
  return result
}

async function runDoctor(signal) {
  const startedAt = new Date().toISOString()
  try {
    const { stdout } = await execFileAsync(SOUNDCHECK_BIN, ['doctor', '--json'], {
      timeout: DOCTOR_TIMEOUT_MS,
      maxBuffer: MAX_OUTPUT_BYTES,
      signal,
      windowsHide: true
    })
    latest = buildResult({ stdout, startedAt, finishedAt: new Date().toISOString() })
  } catch (failure) {
    latest = buildResult({ failure, startedAt, finishedAt: new Date().toISOString() })
  }
  return latest
}

export default function activate(orca) {
  orca.commands.register('doctor.run', async (_args, { signal }) => runDoctor(signal))

  orca.panels.handle('doctor.run', async (_input, { signal }) => runDoctor(signal))
  orca.panels.handle('doctor.latest', async () => latest)
}

export async function deactivate() {
  latest = null
}
