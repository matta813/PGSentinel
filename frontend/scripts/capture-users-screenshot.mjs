import { chromium } from '@playwright/test'
import { spawn } from 'node:child_process'
import { fileURLToPath } from 'node:url'
import path from 'node:path'

const frontendDir = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const output = path.resolve(frontendDir, '..', 'docs', 'assets', 'pgsentinel-users.png')
const baseURL = 'http://127.0.0.1:4173'
const users = [
  { id: '1', username: 'admin', role: 'administrator', mustChangePassword: false, createdAt: '2026-08-20T08:00:00Z', updatedAt: '2026-08-25T12:00:00Z' },
  { id: '2', username: 'database-operations', role: 'operator', mustChangePassword: false, createdAt: '2026-08-23T10:00:00Z', updatedAt: '2026-08-23T10:00:00Z' },
  { id: '3', username: 'incident-viewer', role: 'viewer', mustChangePassword: true, createdAt: '2026-08-26T06:00:00Z', updatedAt: '2026-08-26T06:00:00Z' },
]
const vite = spawn('npm', ['run', 'dev', '--', '--host', '127.0.0.1', '--port', '4173'], { cwd: frontendDir, stdio: 'ignore' })
async function ready() { for (let i = 0; i < 60; i++) { try { if ((await fetch(baseURL)).ok) return } catch { /* starting */ } await new Promise(resolve => setTimeout(resolve, 250)) } throw new Error('Vite did not start') }
try {
  await ready()
  const browser = await chromium.launch({ headless: true })
  const page = await browser.newPage({ viewport: { width: 1440, height: 900 } })
  await page.addInitScript(() => localStorage.setItem('theme', 'dark'))
  await page.route('**/api/v1/**', async route => {
    const pathname = new URL(route.request().url()).pathname
    let body
    if (pathname === '/api/v1/auth/session') body = { authenticated: true, username: 'admin', role: 'administrator', mustChangePassword: false }
    else if (pathname === '/api/v1/version') body = { version: '0.7.0', commit: 'demo' }
    else if (pathname === '/api/v1/users') body = users
    await route.fulfill({ status: body ? 200 : 404, contentType: 'application/json', body: JSON.stringify(body ?? { error: 'Not found' }) })
  })
  await page.goto(`${baseURL}/users`, { waitUntil: 'networkidle' })
  await page.screenshot({ path: output, fullPage: true })
  await browser.close()
} finally { vite.kill('SIGTERM') }
