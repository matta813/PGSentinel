import { FormEvent, useState } from 'react'
import { AlertCircle, Check, Database, KeyRound, LogOut } from 'lucide-react'
import { APIError, api } from '../api/client'
import type { AuthSession } from '../types/auth'

export function ChangePasswordPage({ username, onChanged, onLogout }: { username: string; onChanged: () => void; onLogout: () => void }) {
  const [currentPassword, setCurrentPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [confirmation, setConfirmation] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  async function submit(event: FormEvent) {
    event.preventDefault()
    setError('')
    if (newPassword.length < 12) { setError('New password must contain at least 12 characters.'); return }
    if (newPassword !== confirmation) { setError('New password and confirmation do not match.'); return }
    setBusy(true)
    try { await api.put<AuthSession>('/auth/password', { currentPassword, newPassword }); onChanged() }
    catch (reason) { setError(reason instanceof APIError ? reason.message : 'Unable to change password') }
    finally { setBusy(false) }
  }
  async function logout() { await api.post('/auth/logout'); onLogout() }
  return <main className="auth-screen"><div className="auth-shell password-change-shell"><header className="auth-brand"><span className="brand-mark"><Database /></span><div><strong>PGSentinel</strong><small>Database Intelligence</small></div></header><form className="auth-card" onSubmit={submit}><div className="auth-heading"><div><h1>Set a new password</h1><p>The initial password must be replaced before you can access this deployment.</p></div><span className="auth-lock"><KeyRound /></span></div><div className="first-login-user"><span>Signed in as</span><strong>{username}</strong></div><div className="auth-fields"><label>Current password<input autoFocus autoComplete="current-password" type="password" value={currentPassword} onChange={event => setCurrentPassword(event.target.value)} required /></label><label>New password <small>At least 12 characters</small><input autoComplete="new-password" type="password" minLength={12} value={newPassword} onChange={event => setNewPassword(event.target.value)} required /></label><label>Confirm new password<input autoComplete="new-password" type="password" minLength={12} value={confirmation} onChange={event => setConfirmation(event.target.value)} required /></label></div>{error && <div className="auth-error" role="alert"><AlertCircle />{error}</div>}<button className="button primary auth-submit" type="submit" disabled={busy || !currentPassword || !newPassword || !confirmation}>{busy ? <><span className="button-spinner" />Updating password…</> : <><Check />Continue to PGSentinel</>}</button><button className="auth-logout" type="button" onClick={() => void logout()}><LogOut />Sign out</button></form></div></main>
}
