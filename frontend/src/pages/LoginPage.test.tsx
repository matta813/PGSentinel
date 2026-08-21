import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, expect, test, vi } from 'vitest'
import { LoginPage } from './LoginPage'

afterEach(() => { cleanup(); vi.restoreAllMocks() })

test('authenticates with the administrator password', async () => {
  const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(JSON.stringify({ authenticated: true }), { status: 200, headers: { 'Content-Type': 'application/json' } }))
  const authenticated = vi.fn()
  render(<LoginPage onAuthenticated={authenticated} />)
  fireEvent.change(screen.getByLabelText('Password'), { target: { value: 'secret-password' } })
  fireEvent.click(screen.getByRole('button', { name: 'Sign in' }))
  await waitFor(() => expect(authenticated).toHaveBeenCalledOnce())
  expect(fetchMock).toHaveBeenCalledWith('/api/v1/auth/login', expect.objectContaining({ credentials: 'same-origin' }))
  expect(fetchMock).toHaveBeenCalledWith('/api/v1/auth/login', expect.objectContaining({ body: JSON.stringify({ username: 'admin', password: 'secret-password' }) }))
})

test('shows a safe authentication error', async () => {
  vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(JSON.stringify({ error: 'Invalid credentials' }), { status: 401, headers: { 'Content-Type': 'application/json' } }))
  render(<LoginPage onAuthenticated={() => undefined} />)
  fireEvent.change(screen.getByLabelText('Password'), { target: { value: 'wrong' } })
  fireEvent.click(screen.getByRole('button', { name: 'Sign in' }))
  expect(await screen.findByRole('alert')).toHaveTextContent('Invalid credentials')
})
