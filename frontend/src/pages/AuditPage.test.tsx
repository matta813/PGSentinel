import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, expect, test, vi } from 'vitest'
import { AuditPage } from './AuditPage'

afterEach(() => { cleanup(); vi.restoreAllMocks() })

test('renders safe audit history and filters by action', async () => {
  const fetchMock = vi.spyOn(globalThis, 'fetch').mockImplementation((input) => {
    const path = String(input)
    const events = path.includes('auth.login.failed') ? [{ id: 'failed', occurredAt: '2026-08-25T12:00:00Z', actor: 'anonymous', action: 'auth.login.failed', resourceType: 'session', summary: 'A login attempt failed.' }] : [{ id: 'created', occurredAt: '2026-08-25T11:00:00Z', actor: 'admin', action: 'server.created', resourceType: 'server', resourceId: '12345678-1234-1234-1234-123456789abc', summary: 'A PostgreSQL target was added.' }]
    return Promise.resolve(new Response(JSON.stringify(events), { status: 200, headers: { 'Content-Type': 'application/json' } }))
  })
  render(<MemoryRouter><AuditPage /></MemoryRouter>)
  expect(await screen.findByText('server.created')).toBeInTheDocument()
  expect(screen.getByText('A PostgreSQL target was added.')).toBeInTheDocument()
  fireEvent.change(screen.getByLabelText('Audit action'), { target: { value: 'auth.login.failed' } })
  expect(await screen.findByText('auth.login.failed')).toBeInTheDocument()
  expect(screen.getByText('anonymous')).toBeInTheDocument()
  expect(fetchMock.mock.calls.some(call => String(call[0]).includes('action=auth.login.failed'))).toBe(true)
})
