import { chromium } from '@playwright/test'
import { spawn } from 'node:child_process'
import { fileURLToPath } from 'node:url'
import path from 'node:path'

const frontendDir = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const output = path.resolve(frontendDir, '..', 'docs', 'assets', 'pgsentinel-audit-log.png')
const baseURL = 'http://127.0.0.1:4173'
const events = [
  ['12:38:12', 'admin', 'finding.acknowledged', 'finding', 'a1b2c3d4e5f6a7b8c9d0e1f2', 'A finding was acknowledged; its evidence and health impact remain available.'],
  ['12:31:44', 'admin', 'threshold_override.created', 'threshold_override', '47339eef-6e35-4cb5-829d-3ee8fc077f01', 'A scoped analyzer threshold override was created.'],
  ['12:16:03', 'admin', 'notification_route.updated', 'notification_route', '144de3dc-1745-486d-8229-7b9b0d384650', 'A notification routing rule was edited.'],
  ['11:58:29', 'admin', 'server.credentials_rotated', 'server', '7cbde04b-5f6c-4635-8890-c42ad4fc988a', 'PostgreSQL target credentials were rotated.'],
  ['11:58:29', 'admin', 'server.updated', 'server', '7cbde04b-5f6c-4635-8890-c42ad4fc988a', 'A PostgreSQL target was edited.'],
  ['11:42:17', 'anonymous', 'auth.login.failed', 'session', '', 'A login attempt failed.'],
  ['09:05:08', 'admin', 'maintenance_window.created', 'maintenance_window', '9d807f47-337d-4c42-b826-861954e5a0c0', 'A maintenance window was created.'],
].map(([time, actor, action, resourceType, resourceId, summary], index) => ({ id: `event-${index}`, occurredAt: `2026-08-25T${time}Z`, actor, action, resourceType, resourceId, summary }))
const vite = spawn('npm', ['run', 'dev', '--', '--host', '127.0.0.1', '--port', '4173'], { cwd: frontendDir, stdio: 'ignore' })
async function ready() { for (let i = 0; i < 60; i++) { try { if ((await fetch(baseURL)).ok) return } catch { /* starting */ } await new Promise(resolve => setTimeout(resolve, 250)) } throw new Error('Vite did not start') }
try {
  await ready()
  const browser = await chromium.launch({ headless: true })
  const page = await browser.newPage({ viewport: { width: 1440, height: 900 }, deviceScaleFactor: 1 })
  await page.addInitScript(() => localStorage.setItem('theme', 'dark'))
  await page.route('**/api/v1/**', async route => {
    const pathname = new URL(route.request().url()).pathname
    let body
    if (pathname === '/api/v1/auth/session') body = { authenticated: true, username: 'admin', role: 'administrator', mustChangePassword: false }
    else if (pathname === '/api/v1/version') body = { version: '0.6.0', commit: 'demo' }
    else if (pathname === '/api/v1/audit-events') body = events
    await route.fulfill({ status: body ? 200 : 404, contentType: 'application/json', body: JSON.stringify(body ?? { error: 'Not found' }) })
  })
  await page.goto(`${baseURL}/audit`, { waitUntil: 'networkidle' })
  await page.screenshot({ path: output, fullPage: true })
  await browser.close()
} finally { vite.kill('SIGTERM') }
