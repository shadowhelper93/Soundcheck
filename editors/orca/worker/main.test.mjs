import test from 'node:test'
import assert from 'node:assert/strict'
import activate, { parseDoctorOutput, buildResult, deactivate } from './main.mjs'

test('parseDoctorOutput: empty or whitespace returns empty raw', () => {
  assert.deepEqual(parseDoctorOutput(''), { raw: '' })
  assert.deepEqual(parseDoctorOutput(null), { raw: '' })
  assert.deepEqual(parseDoctorOutput(undefined), { raw: '' })
  assert.deepEqual(parseDoctorOutput('   \n  '), { raw: '' })
})

test('parseDoctorOutput: valid JSON returns parsed report', () => {
  const json = JSON.stringify({ summary: { totalAgents: 6, functionalAgents: 4 } })
  const result = parseDoctorOutput(json)
  assert.deepEqual(result, { report: { summary: { totalAgents: 6, functionalAgents: 4 } } })
})

test('parseDoctorOutput: invalid JSON returns raw string', () => {
  const raw = 'Error: could not run check'
  assert.deepEqual(parseDoctorOutput(raw), { raw })
})

test('buildResult: successful execution with JSON output', () => {
  const startedAt = '2026-11-08T18:00:00.000Z'
  const finishedAt = '2026-11-08T18:00:01.000Z'
  const stdout = JSON.stringify({ summary: { totalAgents: 6 } })

  const result = buildResult({ stdout, startedAt, finishedAt })
  assert.equal(result.ok, true)
  assert.equal(result.exitCode, 0)
  assert.equal(result.startedAt, startedAt)
  assert.equal(result.finishedAt, finishedAt)
  assert.deepEqual(result.report, { summary: { totalAgents: 6 } })
  assert.equal(result.error, undefined)
})

test('buildResult: ENOENT error when binary is missing', () => {
  const startedAt = '2026-11-08T18:00:00.000Z'
  const finishedAt = '2026-11-08T18:00:01.000Z'
  const failure = new Error('spawn soundcheck ENOENT')
  failure.code = 'ENOENT'

  const result = buildResult({ failure, startedAt, finishedAt })
  assert.equal(result.ok, false)
  assert.equal(result.exitCode, null)
  assert.match(result.error, /was not found on PATH\. Install Soundcheck first\./)
})

test('buildResult: timeout failure', () => {
  const startedAt = '2026-11-08T18:00:00.000Z'
  const finishedAt = '2026-11-08T18:00:01.000Z'
  const failure = new Error('timed out')
  failure.killed = true

  const result = buildResult({ failure, startedAt, finishedAt })
  assert.equal(result.ok, false)
  assert.equal(result.error, 'soundcheck doctor timed out.')
})

test('buildResult: degraded check exit code 2 with report in failure.stdout', () => {
  const startedAt = '2026-11-08T18:00:00.000Z'
  const finishedAt = '2026-11-08T18:00:01.000Z'
  const failure = new Error('Command failed: soundcheck doctor --json\nwarning: live smoke probes may use provider quota or incur charges')
  failure.code = 2
  failure.stdout = JSON.stringify({ summary: { functionalAgents: 0 } })

  const result = buildResult({ failure, startedAt, finishedAt })
  assert.equal(result.ok, false)
  assert.equal(result.exitCode, 2)
  assert.deepEqual(result.report, { summary: { functionalAgents: 0 } })
  assert.equal(result.error, undefined)
})

test('buildResult: execution failure without report sets error message', () => {
  const startedAt = '2026-11-08T18:00:00.000Z'
  const finishedAt = '2026-11-08T18:00:01.000Z'
  const failure = new Error('Command failed: soundcheck doctor --json')
  failure.code = 1
  failure.stderr = 'fatal: configuration error'

  const result = buildResult({ failure, startedAt, finishedAt })
  assert.equal(result.ok, false)
  assert.equal(result.exitCode, 1)
  assert.equal(result.report, undefined)
  assert.equal(result.error, 'fatal: configuration error')
})

test('activate and deactivate register handlers and clean up state', async () => {
  const commands = new Map()
  const panels = new Map()

  const fakeOrca = {
    commands: {
      register(name, handler) {
        commands.set(name, handler)
      }
    },
    panels: {
      handle(name, handler) {
        panels.set(name, handler)
      }
    }
  }

  activate(fakeOrca)

  assert.equal(commands.has('doctor.run'), true)
  assert.equal(panels.has('doctor.run'), true)
  assert.equal(panels.has('doctor.latest'), true)

  const latestHandler = panels.get('doctor.latest')
  const initialLatest = await latestHandler()
  assert.equal(initialLatest, null)

  await deactivate()
  assert.equal(await latestHandler(), null)
})
