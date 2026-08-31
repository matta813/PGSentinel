import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, expect, test, vi } from 'vitest'
import { MonitoringProvider, useMonitoring } from './MonitoringContext'

afterEach(() => { cleanup(); vi.restoreAllMocks() })

function Harness() {
  const context = useMonitoring()
  return <><span data-testid="selected-server">{context.selectedServer?.name}</span><span>{context.selectedDatabase || 'all'}</span><button onClick={() => context.setSelectedServerId('secondary')}>Secondary</button><button onClick={() => context.setSelectedDatabase('payments')}>Payments</button><button onClick={() => context.setTimeRange('7d')}>Seven days</button></>
}

test('persists monitoring context and reloads databases when the server changes', async () => {
  const values = new Map<string, string>()
  vi.stubGlobal('localStorage', { getItem: (key: string) => values.get(key) ?? null, setItem: (key: string, value: string) => values.set(key, value), removeItem: (key: string) => values.delete(key) })
  const requests: string[] = []
  vi.spyOn(globalThis, 'fetch').mockImplementation(async input => {
    const url = String(input); requests.push(url)
    if (url.endsWith('/servers')) return new Response(JSON.stringify([{ id: 'primary', name: 'Primary', status: 'healthy' }, { id: 'secondary', name: 'Secondary', status: 'healthy' }]), { status: 200, headers: { 'Content-Type': 'application/json' } })
    return new Response(JSON.stringify({ databases: [{ Name: 'payments' }] }), { status: 200, headers: { 'Content-Type': 'application/json' } })
  })
  render(<MonitoringProvider><Harness /></MonitoringProvider>)
  expect(await screen.findByText('Primary')).toBeInTheDocument()
  fireEvent.click(screen.getByRole('button', { name: 'Secondary' }))
  await waitFor(() => expect(screen.getByTestId('selected-server')).toHaveTextContent('Secondary'))
  await waitFor(() => expect(requests.some(url => url.endsWith('/servers/secondary/databases'))).toBe(true))
  fireEvent.click(screen.getByRole('button', { name: 'Payments' }))
  expect(screen.getByText('payments')).toBeInTheDocument()
  fireEvent.click(screen.getByRole('button', { name: 'Seven days' }))
  expect(values.get('monitoring.range')).toBe('7d')
})
