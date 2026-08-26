import { cleanup, render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, expect, test, vi } from 'vitest'
import { App } from './App'

afterEach(() => { cleanup(); vi.restoreAllMocks() })

function mockSession(authenticated: boolean, role = 'administrator') {
  vi.spyOn(globalThis, 'fetch').mockImplementation((input) => {
    if (String(input) === '/api/v1/auth/session') {
      return Promise.resolve(new Response(JSON.stringify({ authenticated, username: 'admin', role, mustChangePassword: false }), { status: 200, headers: { 'Content-Type': 'application/json' } }))
    }
    return Promise.resolve(new Response(JSON.stringify({ error: 'not found' }), { status: 404, headers: { 'Content-Type': 'application/json' } }))
  })
}

beforeEach(() => {
  const store = new Map<string, string>()
  vi.stubGlobal('localStorage', {
    getItem: (k: string) => store.get(k) ?? null,
    setItem: (k: string, v: string) => { store.set(k, v) },
    removeItem: (k: string) => { store.delete(k) },
    clear: () => { store.clear() },
  })
})

test('does not expose administrator navigation to a viewer', async () => {
  mockSession(true, 'viewer')
  render(<MemoryRouter><App /></MemoryRouter>)
  expect(await screen.findByText('viewer')).toBeInTheDocument()
  expect(screen.queryByRole('link', { name: 'Settings' })).not.toBeInTheDocument()
  expect(screen.queryByRole('link', { name: 'Users' })).not.toBeInTheDocument()
  expect(screen.getByRole('link', { name: 'Problems' })).toBeInTheDocument()
})

test('stays anonymous when the session endpoint reports authenticated:false', async () => {
  mockSession(false)
  render(<MemoryRouter><App /></MemoryRouter>)
  expect(await screen.findByRole('button', { name: 'Sign in' })).toBeInTheDocument()
})

test('renders the authenticated app when the session endpoint reports authenticated:true', async () => {
  mockSession(true)
  render(<MemoryRouter><App /></MemoryRouter>)
  await waitFor(() => expect(screen.queryByRole('button', { name: 'Sign in' })).not.toBeInTheDocument())
  expect(await screen.findByRole('navigation', { name: 'Primary navigation' })).toBeInTheDocument()
  expect(screen.getByRole('link', { name: 'Overview' })).toHaveAttribute('aria-current', 'page')
})
