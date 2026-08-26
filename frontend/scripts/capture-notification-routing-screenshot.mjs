import { chromium } from '@playwright/test'
import { spawn } from 'node:child_process'
import { fileURLToPath } from 'node:url'
import path from 'node:path'

const frontendDir = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const output = path.resolve(frontendDir, '..', 'docs', 'assets', 'pgsentinel-notification-routing.png')
const baseURL = 'http://127.0.0.1:4173'
const vite = spawn('npm', ['run', 'dev', '--', '--host', '127.0.0.1', '--port', '4173'], { cwd: frontendDir, stdio: 'ignore' })
const now = new Date().toISOString()
async function waitForVite() { for (let attempt = 0; attempt < 60; attempt += 1) { try { if ((await fetch(baseURL)).ok) return } catch { /* starting */ } await new Promise(resolve => setTimeout(resolve, 250)) } throw new Error('Vite did not start') }
try {
  await waitForVite()
  const browser = await chromium.launch({ headless: true })
  const page = await browser.newPage({ viewport: { width: 1440, height: 1100 }, deviceScaleFactor: 1 })
  await page.addInitScript(() => localStorage.setItem('theme', 'dark'))
  await page.route('**/api/v1/**', async intercepted => {
    const pathname = new URL(intercepted.request().url()).pathname
    const bodies = {
      '/api/v1/auth/session': { authenticated: true, username: 'admin', role: 'administrator', mustChangePassword: false },
      '/api/v1/version': { version: '0.7.0', commit: 'demo' },
      '/api/v1/notifications': [{ id: 'pager', provider: 'webhook', name: 'Pager webhook', enabled: true, createdAt: now, updatedAt: now }, { id: 'dba', provider: 'ntfy', name: 'DBA operations', enabled: true, createdAt: now, updatedAt: now }],
      '/api/v1/notification-routes': [{ id: 'route', name: 'Critical production findings', enabled: true, priority: 10, severities: ['CRITICAL'], categories: [], serverIds: [], serverTags: ['production'], transitions: ['new', 'severity_increased', 'reopened'], destinationIds: ['pager', 'dba'], cooldownSeconds: 300, createdAt: now, updatedAt: now }],
      '/api/v1/notification-deliveries': [{ eventId: 'event', destinationId: 'pager', destinationName: 'Pager webhook', eventType: 'severity_increased', findingId: 'finding', findingTitle: 'Replica replay lag is increasing', serverId: 'primary', serverName: 'Production primary', severity: 'CRITICAL', category: 'Replication', status: 'delivered', attempts: 1, createdAt: now, deliveredAt: now }],
      '/api/v1/servers': [{ id: 'primary', name: 'Production primary', host: 'postgres.internal', port: 5432, user: 'pgsentinel', sslMode: 'verify-full', status: 'healthy', tags: ['production'] }],
    }
    const body = bodies[pathname]
    await intercepted.fulfill({ status: body ? 200 : 404, contentType: 'application/json', body: JSON.stringify(body ?? { error: 'Not found' }) })
  })
  await page.goto(`${baseURL}/settings`, { waitUntil: 'networkidle' })
  await page.locator('#routing').scrollIntoViewIfNeeded()
  await page.screenshot({ path: output, fullPage: true })
  await browser.close()
} finally { vite.kill('SIGTERM') }
