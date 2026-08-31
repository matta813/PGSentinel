import { cleanup, render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { afterEach, expect, test, vi } from 'vitest'
import { AppLayout } from './AppLayout'

afterEach(() => { cleanup(); vi.restoreAllMocks() })

test('loads build metadata without using a stale cached version', async () => {
  const store = new Map<string, string>()
  vi.stubGlobal('localStorage', {
    get theme() { return store.get('theme') },
    set theme(value: string | undefined) { if (value !== undefined) store.set('theme', value) },
  })
  const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(JSON.stringify({ version: '0.7.0', commit: 'aa3e50f3' }), {
    status: 200,
    headers: { 'Content-Type': 'application/json' },
  }))

  render(<MemoryRouter><Routes><Route element={<AppLayout username="admin" role="administrator" onLogout={() => undefined} />}><Route index element={<p>Overview</p>} /></Route></Routes></MemoryRouter>)

  expect(await screen.findByText('v0.7.0')).toBeInTheDocument()
  await waitFor(() => expect(fetchMock).toHaveBeenCalledWith('/api/v1/version', expect.objectContaining({
    cache: 'no-store',
    headers: expect.objectContaining({ 'Cache-Control': 'no-cache' }),
  })))
})
