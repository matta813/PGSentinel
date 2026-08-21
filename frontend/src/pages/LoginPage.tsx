import { FormEvent, useState } from 'react'
import { AlertCircle, ArrowRight, Database, LockKeyhole } from 'lucide-react'
import { APIError, api } from '../api/client'

export function LoginPage({ onAuthenticated }: { onAuthenticated: () => void }) {
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  async function submit(event: FormEvent) { event.preventDefault(); setBusy(true); setError(''); try { await api.post('/auth/login', { password }); setPassword(''); onAuthenticated() } catch (reason) { setError(reason instanceof APIError ? reason.message : 'Unable to sign in') } finally { setBusy(false) } }
  return <main className="auth-screen"><div className="auth-shell"><header className="auth-brand"><span className="brand-mark"><Database /></span><div><strong>PGSentinel</strong><small>Database Intelligence</small></div></header><form className="auth-card" onSubmit={submit}><div className="auth-heading"><div><h1>Sign in</h1><p>Enter the administrator password for this deployment.</p></div><span className="auth-lock"><LockKeyhole /></span></div><label>Administrator password<input autoFocus autoComplete="current-password" type="password" value={password} onChange={event => setPassword(event.target.value)} required /></label>{error && <div className="auth-error" role="alert"><AlertCircle />{error}</div>}<button className="button primary auth-submit" type="submit" disabled={busy || password.length === 0}>{busy ? <><span className="button-spinner" />Signing in…</> : <>Sign in <ArrowRight /></>}</button><p className="auth-security"><LockKeyhole />Credentials are sent only to this instance.</p></form><footer>PostgreSQL monitoring and health analysis</footer></div></main>
}
