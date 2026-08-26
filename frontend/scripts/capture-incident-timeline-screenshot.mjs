import { chromium } from '@playwright/test'
import { spawn } from 'node:child_process'
import { fileURLToPath } from 'node:url'
import path from 'node:path'

const frontendDir = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const output = path.resolve(frontendDir, '..', 'docs', 'assets', 'pgsentinel-incident-timeline.png')
const baseURL = 'http://127.0.0.1:4173'
const id = 'a1b2c3d4e5f6a7b8c9d0e1f2'
const incident = { id, serverId: 'primary', title: 'Overlapping PostgreSQL operational findings', summary: 'Three findings occurred during the same 15-minute period and match explicit PostgreSQL operational relationships. They may be related; this grouping does not establish causation.', rationale: ['blocked work and query pressure are operationally related', 'connection pressure and lock waits are operationally related'], severity: 'CRITICAL', status: 'active', startedAt: '2026-08-25T08:42:00Z', updatedAt: '2026-08-25T08:51:00Z' }
const findings = [
  { id: 'lock', serverId: 'primary', severity: 'HIGH', category: 'Locks', title: 'Queries are blocked by other sessions', status: 'active', startedAt: '2026-08-25T08:42:00Z', updatedAt: '2026-08-25T08:51:00Z' },
  { id: 'connections', serverId: 'primary', severity: 'CRITICAL', category: 'Connections', title: 'Connection capacity is running low', status: 'active', startedAt: '2026-08-25T08:45:00Z', updatedAt: '2026-08-25T08:51:00Z' },
  { id: 'query', serverId: 'primary', severity: 'HIGH', category: 'Queries', title: 'Query latency regressed against its baseline', status: 'resolved', startedAt: '2026-08-25T08:47:00Z', updatedAt: '2026-08-25T08:50:00Z', resolvedAt: '2026-08-25T08:50:00Z' },
]
const timeline = findings.flatMap(finding => [{ at: finding.startedAt, type: 'finding_started', findingId: finding.id, title: finding.title, detail: 'PGSentinel first observed this finding.', severity: finding.severity }, ...(finding.resolvedAt ? [{ at: finding.resolvedAt, type: 'finding_resolved', findingId: finding.id, title: finding.title, detail: 'The finding evidence no longer crossed its rule threshold.', severity: finding.severity }] : [])])
const vite = spawn('npm', ['run', 'dev', '--', '--host', '127.0.0.1', '--port', '4173'], { cwd: frontendDir, stdio: 'ignore' })

async function ready() { for (let i = 0; i < 60; i++) { try { if ((await fetch(baseURL)).ok) return } catch { /* starting */ } await new Promise(resolve => setTimeout(resolve, 250)) } throw new Error('Vite did not start') }
try {
  await ready()
  const browser = await chromium.launch({ headless: true })
  const page = await browser.newPage({ viewport: { width: 1440, height: 980 }, deviceScaleFactor: 1 })
  await page.addInitScript(() => localStorage.setItem('theme', 'dark'))
  await page.route('**/api/v1/**', async route => {
    const url = new URL(route.request().url())
    let body
    if (url.pathname === '/api/v1/auth/session') body = { authenticated: true, username: 'admin', mustChangePassword: false }
    else if (url.pathname === '/api/v1/version') body = { version: '0.6.0', commit: 'demo' }
    else if (url.pathname === '/api/v1/servers') body = [{ id: 'primary', name: 'Production primary', status: 'healthy', tags: ['production'] }]
    else if (url.pathname === '/api/v1/incidents') body = [incident]
    else if (url.pathname === `/api/v1/incidents/${id}`) body = { ...incident, findings, timeline }
    await route.fulfill({ status: body ? 200 : 404, contentType: 'application/json', body: JSON.stringify(body ?? { error: 'Not found' }) })
  })
  await page.goto(`${baseURL}/incidents?status=active&id=${id}`, { waitUntil: 'networkidle' })
  await page.screenshot({ path: output, fullPage: true })
  await browser.close()
} finally { vite.kill('SIGTERM') }
