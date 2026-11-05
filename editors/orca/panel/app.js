// Soundcheck panel for Orca. Talks to the worker only through window.orca.plugin.
(function () {
  const runButton = document.getElementById('run')
  const statusEl = document.getElementById('status')
  const errorEl = document.getElementById('error')
  const reportEl = document.getElementById('report')

  // Doctor output is third-party text: always set it with textContent, never innerHTML.
  function render(result) {
    errorEl.textContent = ''
    if (!result) {
      statusEl.textContent = 'No run yet.'
      reportEl.textContent = ''
      return
    }
    statusEl.textContent = `Last run: ${result.finishedAt ?? result.startedAt ?? 'unknown'}`
    if (result.error) errorEl.textContent = result.error
    const body = result.report !== undefined ? result.report : result.raw ?? ''
    reportEl.textContent = typeof body === 'string' ? body : JSON.stringify(body, null, 2)
  }

  async function run() {
    runButton.disabled = true
    statusEl.textContent = 'Running soundcheck doctor…'
    try {
      render(await window.orca.plugin.request('doctor.run', {}, { timeoutMs: 130_000 }))
    } catch (error) {
      errorEl.textContent = `Run failed: ${error && error.message ? error.message : error}`
      statusEl.textContent = ''
    } finally {
      runButton.disabled = false
    }
  }

  runButton.addEventListener('click', run)

  // Show the previous result when the panel opens.
  window.orca.plugin
    .request('doctor.latest', {}, { timeoutMs: 10_000 })
    .then(render)
    .catch(() => render(null))
})()
