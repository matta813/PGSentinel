import { cleanup, render, screen } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { afterEach, expect, test, vi } from 'vitest'
import { ResourcePage } from './ResourcePage'

afterEach(() => { cleanup(); vi.restoreAllMocks() })

test('warns before rendering cached evidence as current', async () => {
  vi.spyOn(globalThis, 'fetch').mockImplementation(async input => {
    const url = String(input)
    if (url.endsWith('/servers')) return new Response(JSON.stringify([{ id: 'server-1', name: 'Primary', status: 'degraded', tags: [] }]), { status: 200 })
    if (url.endsWith('/queries')) return new Response(JSON.stringify([{ QueryID: '1', Query: 'select 1', Database: 'app', Calls: 20, MeanExecMS: 3, TotalExecMS: 60, ImpactScore: 1 }]), { status: 200 })
    return new Response(JSON.stringify([{ serverId: 'server-1', resource: 'queries', state: 'unavailable', lastSuccessfulCollection: '2026-08-25T10:00:00Z', expectedIntervalSeconds: 30, consecutiveFailures: 2, errorSummary: 'Collection failed; the last successful evidence is preserved.' }]), { status: 200 })
  })
  render(<MemoryRouter initialEntries={['/queries']}><Routes><Route path="/:resource" element={<ResourcePage />} /></Routes></MemoryRouter>)
  expect(await screen.findByText('select 1')).toBeInTheDocument()
  expect(screen.getByText('Current evidence is unavailable')).toBeInTheDocument()
  expect(screen.getByText(/last successful evidence is preserved/i)).toBeInTheDocument()
})
