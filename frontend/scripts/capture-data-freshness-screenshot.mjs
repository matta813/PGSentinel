import { chromium } from '@playwright/test'
import { spawn } from 'node:child_process'
import { fileURLToPath } from 'node:url'
import path from 'node:path'

const frontendDir = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const output = path.resolve(frontendDir, '..', 'docs', 'assets', 'pgsentinel-data-freshness.png')
const baseURL = 'http://127.0.0.1:4173'
const vite = spawn('npm', ['run', 'dev', '--', '--host', '127.0.0.1', '--port', '4173'], { cwd: frontendDir, stdio: 'ignore' })
async function waitForVite() { for (let attempt = 0; attempt < 60; attempt += 1) { try { if ((await fetch(baseURL)).ok) return } catch { /* starting */ } await new Promise(resolve => setTimeout(resolve, 250)) } throw new Error('Vite did not start') }
try {
  await waitForVite()
  const browser = await chromium.launch({ headless: true })
  const page = await browser.newPage({ viewport: { width: 1440, height: 900 }, deviceScaleFactor: 1 })
  await page.addInitScript(() => localStorage.setItem('theme', 'dark'))
  await page.route('**/api/v1/**', async route => {
    const pathname = new URL(route.request().url()).pathname
    let body
    if (pathname === '/api/v1/auth/session') body = { authenticated: true, username: 'admin', mustChangePassword: false }
    else if (pathname === '/api/v1/version') body = { version: '0.7.0', commit: 'demo' }
    else if (pathname === '/api/v1/servers') body = [{ id: 'primary', name: 'Production primary', status: 'degraded', tags: ['production'] }]
    else if (pathname === '/api/v1/servers/primary/queries') body = [{ QueryID: '81273', Database: 'payments', Query: 'SELECT status, count(*) FROM payment_events WHERE created_at > $1 GROUP BY status', Calls: 18420, MeanExecMS: 42.8, TotalExecMS: 788376, ImpactScore: 86.4 }]
    else if (pathname === '/api/v1/servers/primary/freshness') body = [{ serverId: 'primary', resource: 'queries', state: 'unavailable', lastSuccessfulCollection: '2026-08-25T10:14:00Z', ageSeconds: 742, expectedIntervalSeconds: 30, consecutiveFailures: 3, errorSummary: 'Collection failed; the last successful evidence is preserved.' }]
    await route.fulfill({ status: body ? 200 : 404, contentType: 'application/json', body: JSON.stringify(body ?? { error: 'Not found' }) })
  })
  await page.goto(`${baseURL}/queries`, { waitUntil: 'networkidle' })
  await page.getByText('Current evidence is unavailable').waitFor()
  await page.screenshot({ path: output, fullPage: true })
  await browser.close()
} finally { vite.kill('SIGTERM') }
