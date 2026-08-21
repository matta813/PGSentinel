import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, expect, test, vi } from 'vitest'
import { ChangePasswordPage } from './ChangePasswordPage'

afterEach(() => { cleanup(); vi.restoreAllMocks() })

test('changes the bootstrap password before continuing', async () => {
  const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(JSON.stringify({ authenticated: true, username: 'admin', mustChangePassword: false }), { status: 200, headers: { 'Content-Type': 'application/json' } }))
  const changed = vi.fn()
  render(<ChangePasswordPage username="admin" onChanged={changed} onLogout={() => undefined} />)
  fireEvent.change(screen.getByLabelText('Current password'), { target: { value: 'bootstrap-password' } })
  fireEvent.change(screen.getByLabelText(/New password/), { target: { value: 'replacement-password' } })
  fireEvent.change(screen.getByLabelText('Confirm new password'), { target: { value: 'replacement-password' } })
  fireEvent.click(screen.getByRole('button', { name: 'Continue to PGSentinel' }))
  await waitFor(() => expect(changed).toHaveBeenCalledOnce())
  expect(fetchMock).toHaveBeenCalledWith('/api/v1/auth/password', expect.objectContaining({ method: 'PUT', body: JSON.stringify({ currentPassword: 'bootstrap-password', newPassword: 'replacement-password' }) }))
})

test('rejects a mismatched confirmation locally', () => {
  const fetchMock = vi.spyOn(globalThis, 'fetch')
  render(<ChangePasswordPage username="admin" onChanged={() => undefined} onLogout={() => undefined} />)
  fireEvent.change(screen.getByLabelText('Current password'), { target: { value: 'bootstrap-password' } })
  fireEvent.change(screen.getByLabelText(/New password/), { target: { value: 'replacement-password' } })
  fireEvent.change(screen.getByLabelText('Confirm new password'), { target: { value: 'different-password' } })
  fireEvent.click(screen.getByRole('button', { name: 'Continue to PGSentinel' }))
  expect(screen.getByRole('alert')).toHaveTextContent('do not match')
  expect(fetchMock).not.toHaveBeenCalled()
})
