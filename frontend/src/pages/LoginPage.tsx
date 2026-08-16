import { FormEvent, useState } from 'react'
import { Database, LockKeyhole } from 'lucide-react'
import { APIError, api } from '../api/client'

export function LoginPage({ onAuthenticated }: { onAuthenticated: () => void }) {
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  async function submit(event: FormEvent) {
    event.preventDefault(); setBusy(true); setError('')
    try { await api.post('/auth/login', { password }); setPassword(''); onAuthenticated() }
    catch (reason) { setError(reason instanceof APIError ? reason.message : 'Unable to sign in') }
    finally { setBusy(false) }
  }
  return <main className="auth-screen"><form className="auth-card" onSubmit={submit}><div className="auth-brand"><span><Database /></span><div><strong>PGSentinel</strong><small>PostgreSQL health analysis</small></div></div><div className="auth-heading"><LockKeyhole /><div><h1>Administrator sign in</h1><p>Enter the password configured for this PGSentinel instance.</p></div></div><label>Administrator password<input autoFocus autoComplete="current-password" type="password" value={password} onChange={event => setPassword(event.target.value)} required /></label>{error && <p className="auth-error" role="alert">{error}</p>}<button type="submit" disabled={busy || password.length === 0}>{busy ? 'Signing in…' : 'Sign in'}</button></form></main>
}
