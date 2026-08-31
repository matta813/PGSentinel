import { cleanup, render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { afterEach, expect, test, vi } from 'vitest'
import { AppLayout } from './AppLayout'
import { MonitoringProvider } from '../context/MonitoringContext'

afterEach(() => { cleanup(); vi.restoreAllMocks() })

test('loads build metadata without using a stale cached version', async () => {
  const store = new Map<string, string>()
  vi.stubGlobal('localStorage', {
    getItem: (key: string) => store.get(key) ?? null,
    setItem: (key: string, value: string) => store.set(key, value),
    removeItem: (key: string) => store.delete(key),
  })
  const fetchMock = vi.spyOn(globalThis, 'fetch').mockImplementation(async input => new Response(JSON.stringify(String(input).endsWith('/servers') ? [] : { version: '0.7.0', commit: 'aa3e50f3' }), { status: 200, headers: { 'Content-Type': 'application/json' } }))

  render(<MemoryRouter><MonitoringProvider><Routes><Route element={<AppLayout username="admin" role="administrator" onLogout={() => undefined} />}><Route index element={<p>Dashboard</p>} /></Route></Routes></MonitoringProvider></MemoryRouter>)

  expect(await screen.findByText('v0.7.0')).toBeInTheDocument()
  await waitFor(() => expect(fetchMock).toHaveBeenCalledWith('/api/v1/version', expect.objectContaining({
    cache: 'no-store',
    headers: expect.objectContaining({ 'Cache-Control': 'no-cache' }),
  })))
})
