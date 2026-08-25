import { cleanup, render, screen } from '@testing-library/react'
import { afterEach, expect, test, vi } from 'vitest'
import { SettingsPage } from './SettingsPage'

afterEach(() => { cleanup(); vi.restoreAllMocks() })

test('explains routing fallback and exposes bounded operator controls', async () => {
  vi.spyOn(globalThis, 'fetch').mockImplementation((input) => {
    const path = String(input)
    const payload = path.includes('/maintenance-windows') || path.includes('/suppressions') ? []
      : path.includes('/threshold-overrides') ? { items: [], specs: { 'standby-replay-lag': { label: 'Replica replay lag', min: 10, max: 86400, default: 60, unit: 'seconds' } } }
        : path.includes('/notifications') && !path.includes('deliveries') ? [{ id: 'destination', provider: 'webhook', name: 'Pager', enabled: true, createdAt: '', updatedAt: '' }]
          : path.includes('/notification-routes') ? []
            : path.includes('/notification-deliveries') ? [{ eventId: 'event', destinationId: 'destination', destinationName: 'Pager', eventType: 'severity_increased', findingId: 'finding', findingTitle: 'Replica lag', serverId: 'server', serverName: 'Primary', severity: 'CRITICAL', category: 'Replication', status: 'retry', lastError: 'webhook returned HTTP 503', attempts: 1, createdAt: '2026-08-25T00:00:00Z' }]
              : [{ id: 'server', name: 'Primary', host: 'db', port: 5432, user: 'monitor', sslMode: 'require', status: 'healthy', tags: ['production'] }]
    return Promise.resolve(new Response(JSON.stringify(payload), { status: 200, headers: { 'Content-Type': 'application/json' } }))
  })
  render(<SettingsPage />)
  expect(await screen.findByText('No routing rules')).toBeInTheDocument()
  expect(screen.getByText(/High and Critical lifecycle events/)).toBeInTheDocument()
  expect(screen.getByText('Replica lag')).toBeInTheDocument()
  expect(screen.getByText('retry')).toBeInTheDocument()
  expect(screen.getByText('webhook returned HTTP 503')).toBeInTheDocument()
  expect(screen.getByLabelText('Primary')).toBeInTheDocument()
  expect(screen.getByLabelText('Pager')).toBeInTheDocument()
  expect(await screen.findByText('Maintenance windows')).toBeInTheDocument()
  expect(screen.getByText(/10–86400 seconds; default 60/)).toBeInTheDocument()
})
