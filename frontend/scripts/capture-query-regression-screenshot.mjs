import { chromium } from '@playwright/test'
import { spawn } from 'node:child_process'
import { fileURLToPath } from 'node:url'
import path from 'node:path'

const frontendDir = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const output = path.resolve(frontendDir, '..', 'docs', 'assets', 'pgsentinel-query-regression.png')
const baseURL = 'http://127.0.0.1:4173'
const vite = spawn('npm', ['run', 'dev', '--', '--host', '127.0.0.1', '--port', '4173'], { cwd: frontendDir, stdio: 'ignore' })
async function waitForVite() { for (let i = 0; i < 60; i++) { try { if ((await fetch(baseURL)).ok) return } catch { /* starting */ } await new Promise(resolve => setTimeout(resolve, 250)) } throw new Error('Vite did not start') }
try {
  await waitForVite()
  const browser = await chromium.launch({ headless: true })
  const page = await browser.newPage({ viewport: { width: 1440, height: 1100 } })
  await page.addInitScript(() => localStorage.setItem('theme', 'dark'))
  await page.route('**/api/v1/**', async route => {
    const pathname = new URL(route.request().url()).pathname
    let body
    if (pathname === '/api/v1/auth/session') body = { authenticated: true, username: 'admin', mustChangePassword: false }
    else if (pathname === '/api/v1/version') body = { version: '0.7.0', commit: 'demo' }
    else if (pathname === '/api/v1/problems') body = [{ id: 'regression', serverId: 'primary', database: 'payments', resource: '81273', severity: 'HIGH', category: 'Queries', title: 'Persistent query latency regression detected', summary: 'Query 81273 exceeded its reset-aware baseline in two consecutive observation intervals with sufficient calls and runtime.', cause: '', impact: 'A persistent increase in execution latency can increase application response time and database load. The overlap is evidence of a meaningful change, not proof of a specific cause.', confidence: 'HIGH', status: 'active', startedAt: '2026-08-25T10:08:00Z', updatedAt: '2026-08-25T10:10:00Z', evidence: [{ label: 'Current window', value: '2026-08-25T10:09:00Z to 2026-08-25T10:10:00Z' }, { label: 'Baseline window', value: '2026-08-25T10:00:00Z to 2026-08-25T10:07:00Z' }, { label: 'Baseline samples', value: '7 intervals' }, { label: 'Previous interval mean', value: '30.00 ms' }, { label: 'Current interval mean', value: '32.00 ms' }, { label: 'Baseline median', value: '10.00 ms' }, { label: 'Absolute difference', value: '+22.00 ms' }, { label: 'Relative difference', value: '+220.0%' }, { label: 'Current calls', value: '85' }, { label: 'Current total runtime', value: '2720 ms' }, { label: 'Current rows', value: '255' }, { label: 'Current shared reads', value: '43 blocks' }, { label: 'Significance', value: 'Above median + 3 MAD, at least 50% and 5 ms slower, persistent for 2 intervals' }], suggestions: [{ title: 'Correlate the regression window', detail: 'Compare deployments, parameter changes, data growth, cache pressure, call volume and row volume.' }, { title: 'Inspect a safe plan separately', detail: 'PGSentinel never runs EXPLAIN ANALYZE automatically.' }] }]
    await route.fulfill({ status: body ? 200 : 404, contentType: 'application/json', body: JSON.stringify(body ?? { error: 'Not found' }) })
  })
  await page.goto(`${baseURL}/problems?id=regression`, { waitUntil: 'networkidle' })
  await page.locator('#finding-regression').waitFor()
  await page.screenshot({ path: output, fullPage: true })
  await browser.close()
} finally { vite.kill('SIGTERM') }
