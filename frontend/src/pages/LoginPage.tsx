import { FormEvent, useState } from 'react'
import { AlertCircle, ArrowRight, Database, LockKeyhole } from 'lucide-react'
import { APIError, api } from '../api/client'
import type { AuthSession } from '../types/auth'

export function LoginPage({ onAuthenticated }: { onAuthenticated: (session: AuthSession) => void }) {
  const [username, setUsername] = useState('admin')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  async function submit(event: FormEvent) { event.preventDefault(); setBusy(true); setError(''); try { const session = await api.post<AuthSession>('/auth/login', { username: username.trim(), password }); setPassword(''); onAuthenticated(session) } catch (reason) { setError(reason instanceof APIError ? reason.message : 'Unable to sign in') } finally { setBusy(false) } }
  return <main className="auth-screen"><div className="auth-shell"><header className="auth-brand"><span className="brand-mark"><Database /></span><div><strong>PGSentinel</strong><small>Database Intelligence</small></div></header><form className="auth-card" onSubmit={submit}><div className="auth-heading"><div><h1>Sign in</h1><p>Enter your credentials for this deployment.</p></div><span className="auth-lock"><LockKeyhole /></span></div><div className="auth-fields"><label>Username<input autoFocus autoComplete="username" value={username} onChange={event => setUsername(event.target.value)} required /></label><label>Password<input autoComplete="current-password" type="password" value={password} onChange={event => setPassword(event.target.value)} required /></label></div>{error && <div className="auth-error" role="alert"><AlertCircle />{error}</div>}<button className="button primary auth-submit" type="submit" disabled={busy || username.trim().length === 0 || password.length === 0}>{busy ? <><span className="button-spinner" />Signing in…</> : <>Sign in <ArrowRight /></>}</button><p className="auth-security"><LockKeyhole />Credentials are sent only to this instance.</p></form><footer>PostgreSQL monitoring and health analysis</footer></div></main>
}
