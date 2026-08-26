import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, expect, test, vi } from 'vitest'
import { UsersPage } from './UsersPage'

afterEach(() => { cleanup(); vi.restoreAllMocks() })

test('creates a viewer without retaining the initial password in the UI', async () => {
  const fetchMock = vi.spyOn(globalThis, 'fetch').mockImplementation((_input, init) => {
    if (init?.method === 'POST') return Promise.resolve(new Response(JSON.stringify({ id: 'new', username: 'readonly', role: 'viewer', mustChangePassword: true }), { status: 201, headers: { 'Content-Type': 'application/json' } }))
    return Promise.resolve(new Response(JSON.stringify([{ id: 'admin', username: 'admin', role: 'administrator', mustChangePassword: false, createdAt: '2026-08-26T00:00:00Z', updatedAt: '2026-08-26T00:00:00Z' }]), { status: 200, headers: { 'Content-Type': 'application/json' } }))
  })
  render(<MemoryRouter><UsersPage currentUser="admin" /></MemoryRouter>)
  expect(await screen.findByText('Current account')).toBeInTheDocument()
  fireEvent.change(screen.getByLabelText('Username'), { target: { value: 'readonly' } })
  fireEvent.change(screen.getByLabelText('Initial password'), { target: { value: 'temporary-safe-password' } })
  fireEvent.click(screen.getByRole('button', { name: 'Add user' }))
  expect(await screen.findByText(/must change the initial password/i)).toBeInTheDocument()
  expect(fetchMock.mock.calls.some(([, init]) => init?.method === 'POST' && String(init.body).includes('temporary-safe-password'))).toBe(true)
  expect(screen.queryByDisplayValue('temporary-safe-password')).not.toBeInTheDocument()
})
