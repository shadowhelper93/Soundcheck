// Soundcheck panel for Orca. Talks to the worker only through window.orca.plugin.
(function () {
  const runButton = document.getElementById('run')
  const statusEl = document.getElementById('status')
  const errorEl = document.getElementById('error')
  const reportEl = document.getElementById('report')
  const dashboardEl = document.getElementById('dashboard')
  const summaryAgentsEl = document.getElementById('summary-agents')
  const summaryQuirksEl = document.getElementById('summary-quirks')
  const adaptersEl = document.getElementById('adapters')
  const rolesEl = document.getElementById('roles')

  function clearElement(el) {
    while (el.firstChild) {
      el.removeChild(el.firstChild)
    }
  }

  function createBadge(text, type) {
    const span = document.createElement('span')
    span.className = `badge badge-${type}`
    span.textContent = text
    return span
  }

  function createTierPill(name, tier) {
    const pill = document.createElement('span')
    const status = tier?.status ?? 'pending'
    pill.className = `tier-pill tier-${status}`
    pill.textContent = `${name}:${status}`
    return pill
  }

  function renderDashboard(report) {
    if (!report || typeof report !== 'object' || (!report.adapters && !report.summary)) {
      dashboardEl.style.display = 'none'
      return
    }

    dashboardEl.style.display = 'block'

    // Summary stats
    if (report.summary) {
      const func = report.summary.functionalAgents ?? 0
      const total = report.summary.totalAgents ?? 0
      summaryAgentsEl.textContent = `${func} / ${total}`
      summaryQuirksEl.textContent = String(report.summary.quirksProtected ?? 0)
    }

    // Adapters
    clearElement(adaptersEl)
    if (Array.isArray(report.adapters)) {
      for (const adapter of report.adapters) {
        const card = document.createElement('div')
        card.className = 'card'

        const header = document.createElement('div')
        header.className = 'card-header'

        const title = document.createElement('span')
        title.className = 'card-title'
        title.textContent = adapter.name || adapter.id || 'Adapter'

        const statusType = adapter.status === 'healthy' ? 'healthy' : adapter.status === 'degraded' ? 'degraded' : 'down'
        const badge = createBadge(adapter.status || 'unknown', statusType)

        header.appendChild(title)
        header.appendChild(badge)
        card.appendChild(header)

        if (adapter.tiers) {
          const tiersContainer = document.createElement('div')
          tiersContainer.className = 'tiers'
          for (const tierKey of ['L0', 'L1', 'L2', 'L3']) {
            tiersContainer.appendChild(createTierPill(tierKey, adapter.tiers[tierKey]))
          }
          card.appendChild(tiersContainer)
        }

        adaptersEl.appendChild(card)
      }
    }

    // Roles
    clearElement(rolesEl)
    if (report.resolutions && typeof report.resolutions === 'object') {
      for (const [roleName, resolution] of Object.entries(report.resolutions)) {
        const card = document.createElement('div')
        card.className = 'card'

        const header = document.createElement('div')
        header.className = 'card-header'

        const title = document.createElement('span')
        title.className = 'card-title'
        title.textContent = roleName.charAt(0).toUpperCase() + roleName.slice(1)

        const isHealthy = Boolean(resolution.healthy)
        const badge = createBadge(isHealthy ? 'assigned' : 'unresolved', isHealthy ? 'healthy' : 'down')

        header.appendChild(title)
        header.appendChild(badge)
        card.appendChild(header)

        const details = document.createElement('div')
        details.className = 'role-details'
        const agent = resolution.selectedAgentId ?? 'none'
        const model = resolution.selectedModel ?? 'None'
        details.textContent = `${agent} • ${model}`
        card.appendChild(details)

        rolesEl.appendChild(card)
      }
    }
  }

  // Doctor output is third-party text: always set it with textContent, never innerHTML.
  function render(result) {
    if (!result) {
      statusEl.textContent = 'No run yet.'
      errorEl.style.display = 'none'
      errorEl.textContent = ''
      reportEl.textContent = ''
      dashboardEl.style.display = 'none'
      return
    }

    statusEl.textContent = `Last run: ${result.finishedAt ?? result.startedAt ?? 'unknown'}`
    
    if (result.error) {
      errorEl.textContent = result.error
      errorEl.style.display = 'block'
    } else {
      errorEl.style.display = 'none'
      errorEl.textContent = ''
    }

    const body = result.report !== undefined ? result.report : result.raw ?? ''
    reportEl.textContent = typeof body === 'string' ? body : JSON.stringify(body, null, 2)

    if (result.report && typeof result.report === 'object') {
      renderDashboard(result.report)
    } else {
      dashboardEl.style.display = 'none'
    }
  }

  async function run() {
    runButton.disabled = true
    statusEl.textContent = 'Running soundcheck doctor…'
    try {
      render(await window.orca.plugin.request('doctor.run', {}, { timeoutMs: 120_000 }))
    } catch (error) {
      errorEl.textContent = `Run failed: ${error && error.message ? error.message : error}`
      errorEl.style.display = 'block'
      statusEl.textContent = ''
    } finally {
      runButton.disabled = false
    }
  }

  runButton.addEventListener('click', run)

  // Show previous result when panel opens.
  window.orca.plugin
    .request('doctor.latest', {}, { timeoutMs: 10_000 })
    .then(render)
    .catch(() => render(null))
})()
