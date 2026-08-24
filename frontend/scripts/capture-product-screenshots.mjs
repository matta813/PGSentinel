import { chromium } from '@playwright/test'
import { spawn } from 'node:child_process'
import { mkdir } from 'node:fs/promises'
import { fileURLToPath } from 'node:url'
import path from 'node:path'

const frontendDir = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const outputDir = path.resolve(frontendDir, '..', 'docs', 'assets')
const baseURL = 'http://127.0.0.1:4173'
const vite = spawn('npm', ['run', 'dev', '--', '--host', '127.0.0.1', '--port', '4173'], { cwd: frontendDir, stdio: 'ignore' })

const now = new Date().toISOString()
const server = {
  id: 'prod-primary', name: 'Production primary', host: 'postgres-primary.internal', port: 5432,
  user: 'pgsentinel', sslMode: 'verify-full', version: '16.4', status: 'healthy',
  lastConnectedAt: now, tags: ['production', 'primary'],
}
const findings = [
  {
    id: 'long-transaction', serverId: server.id, database: 'payments', resource: 'PID 18422', severity: 'CRITICAL',
    category: 'transactions', title: 'Long-running transaction is holding back cleanup',
    summary: 'A transaction has remained open for 47 minutes and may prevent vacuum from reclaiming dead tuples.',
    cause: 'The session is idle in transaction after a reporting query completed.',
    impact: 'Old row versions remain visible, increasing table bloat and transaction ID pressure while the transaction stays open.',
    evidence: [{ label: 'Transaction age', value: '47', unit: 'minutes' }, { label: 'Session state', value: 'idle in transaction' }, { label: 'Database', value: 'payments' }],
    suggestions: [{ title: 'Identify the owning workload', detail: 'Inspect PID 18422 and confirm whether the client still needs the transaction.' }, { title: 'Check cleanup pressure', detail: 'Review dead tuples and autovacuum progress on affected tables before intervening.' }],
    confidence: 'HIGH', status: 'active', startedAt: now, updatedAt: now,
  },
  {
    id: 'connection-pressure', serverId: server.id, database: 'payments', severity: 'HIGH', category: 'connections',
    title: 'Connection capacity is close to exhaustion', summary: 'Active and idle sessions are using 91% of available PostgreSQL connections.',
    cause: 'Connection usage has crossed the high-pressure threshold.', impact: 'New clients may be rejected if demand increases or reserved capacity is consumed.',
    evidence: [{ label: 'Connections used', value: '182 / 200' }, { label: 'Utilization', value: '91', unit: '%' }],
    suggestions: [{ title: 'Break down sessions by application', detail: 'Find unexpected pools or clients retaining more connections than intended.' }],
    confidence: 'HIGH', status: 'active', startedAt: now, updatedAt: now,
  },
  {
    id: 'vacuum-pressure', serverId: server.id, database: 'orders', resource: 'public.order_events', severity: 'MEDIUM', category: 'vacuum',
    title: 'Table is accumulating dead tuples', summary: 'Dead tuples are growing faster than the most recent autovacuum cycle is clearing them.',
    cause: 'A write-heavy table is approaching its estimated autovacuum trigger.', impact: 'Continued growth can increase storage use and slow scans.',
    evidence: [{ label: 'Dead tuples', value: '1.8M' }, { label: 'Vacuum progress', value: '86', unit: '%' }],
    suggestions: [{ title: 'Review autovacuum history', detail: 'Confirm workers are reaching the table and completing without cancellation.' }],
    confidence: 'MEDIUM', status: 'active', startedAt: now, updatedAt: now,
  },
]

async function waitForVite() {
  for (let attempt = 0; attempt < 60; attempt += 1) {
    try { if ((await fetch(baseURL)).ok) return } catch { /* Vite is still starting. */ }
    await new Promise(resolve => setTimeout(resolve, 250))
  }
  throw new Error('Vite did not start within 15 seconds')
}

try {
  await waitForVite()
  await mkdir(outputDir, { recursive: true })
  const browser = await chromium.launch({ headless: true })
  const page = await browser.newPage({ viewport: { width: 1440, height: 960 }, deviceScaleFactor: 1 })
  await page.addInitScript(() => localStorage.setItem('theme', 'dark'))
  await page.route('**/api/v1/**', async route => {
    const pathname = new URL(route.request().url()).pathname
    const bodies = {
      '/api/v1/auth/session': { authenticated: true, username: 'admin', mustChangePassword: false },
      '/api/v1/version': { version: '0.5.0', commit: 'demo' },
      '/api/v1/overview': { servers: [server], problems: findings, counts: { CRITICAL: 1, HIGH: 1, MEDIUM: 1 }, score: { overall: 68, categories: { connections: 61, transactions: 44, queries: 87, vacuum: 72, indexes: 93 } } },
      '/api/v1/problems': findings,
    }
    const body = bodies[pathname]
    await route.fulfill({ status: body ? 200 : 404, contentType: 'application/json', body: JSON.stringify(body ?? { error: 'Not found' }) })
  })

  await page.goto(baseURL, { waitUntil: 'networkidle' })
  await page.screenshot({ path: path.join(outputDir, 'pgsentinel-overview.png'), fullPage: true })
  await page.goto(`${baseURL}/problems?id=long-transaction`, { waitUntil: 'networkidle' })
  await page.locator('#finding-long-transaction').waitFor()
  await page.screenshot({ path: path.join(outputDir, 'pgsentinel-problem-detail.png'), fullPage: true })
  await page.setViewportSize({ width: 1280, height: 640 })
  await page.setContent(`<!doctype html><html><head><style>
    * { box-sizing: border-box } body { margin: 0; width: 1280px; height: 640px; overflow: hidden; background: #070c12; color: #f5f7fa; font-family: Inter, ui-sans-serif, system-ui, sans-serif }
    main { position: relative; width: 100%; height: 100%; padding: 92px 100px; background: radial-gradient(circle at 88% 12%, rgba(34,211,173,.16), transparent 31%), linear-gradient(135deg, #070c12 0%, #0c141e 100%) }
    main::after { content: ''; position: absolute; inset: 32px; border: 1px solid #1d3342; border-radius: 24px; pointer-events: none }
    .brand { display: flex; align-items: center; gap: 20px; color: #83f3d7; font-size: 25px; font-weight: 700; letter-spacing: .01em }
    .mark { display: grid; place-items: center; width: 58px; height: 58px; border: 1px solid #1ca88c; border-radius: 14px; background: #09261f; font-size: 28px }
    h1 { width: 900px; margin: 72px 0 24px; font-size: 66px; line-height: 1.08; letter-spacing: -.045em }
    p { width: 820px; margin: 0; color: #9eb1c4; font-size: 27px; line-height: 1.45 }
    strong { color: #55dfbd; font-weight: 600 }
    .signal { position: absolute; right: 100px; bottom: 88px; display: flex; align-items: center; gap: 12px; color: #9eb1c4; font: 17px ui-monospace, monospace }
    .signal i { width: 10px; height: 10px; border-radius: 50%; background: #35d6ad; box-shadow: 0 0 20px #35d6ad }
  </style></head><body><main><div class="brand"><span class="mark">⌕</span>PGSentinel</div><h1>PostgreSQL monitoring that explains problems.</h1><p>See <strong>what is wrong</strong>, why it matters, the evidence behind it, and what to investigate next.</p><div class="signal"><i></i>operations inbox for PostgreSQL</div></main></body></html>`)
  await page.screenshot({ path: path.join(outputDir, 'pgsentinel-social-preview.png') })
  await browser.close()
} finally {
  vite.kill('SIGTERM')
}
