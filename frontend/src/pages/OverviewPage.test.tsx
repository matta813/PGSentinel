import { cleanup, render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, expect, test, vi } from 'vitest'
import { OverviewPage } from './OverviewPage'

afterEach(() => {
  cleanup()
  vi.restoreAllMocks()
})

test('counts only healthy servers as online', async () => {
  vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(JSON.stringify({
    servers: [
      { id: 'healthy', name: 'Healthy', host: 'db-1', port: 5432, user: 'monitor', sslMode: 'require', status: 'healthy', tags: [] },
      { id: 'unknown', name: 'Pending', host: 'db-2', port: 5432, user: 'monitor', sslMode: 'require', status: 'unknown', tags: [] },
      { id: 'error', name: 'Failed', host: 'db-3', port: 5432, user: 'monitor', sslMode: 'require', status: 'error', tags: [] },
    ],
    problems: [],
    counts: {},
    score: { overall: 100, categories: {} },
  }), { status: 200, headers: { 'Content-Type': 'application/json' } }))

  render(<MemoryRouter><OverviewPage /></MemoryRouter>)

  expect(await screen.findByText('Servers online')).toBeInTheDocument()
  expect(screen.getByText('of 3')).toBeInTheDocument()
})
