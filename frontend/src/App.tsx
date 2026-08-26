import { useEffect, useState } from 'react'
import { Route, Routes } from 'react-router-dom'
import { api } from './api/client'
import { AppLayout } from './layouts/AppLayout'
import { LoginPage } from './pages/LoginPage'
import { OverviewPage } from './pages/OverviewPage'
import { IncidentsPage } from './pages/IncidentsPage'
import { AuditPage } from './pages/AuditPage'
import { ProblemsPage } from './pages/ProblemsPage'
import { ResourcePage } from './pages/ResourcePage'
import { ServersPage } from './pages/ServersPage'
import { SettingsPage } from './pages/SettingsPage'
import { ChangePasswordPage } from './pages/ChangePasswordPage'
import { UsersPage } from './pages/UsersPage'
import type { AuthSession } from './types/auth'

export function App() {
  const [authentication, setAuthentication] = useState<'loading' | 'anonymous' | AuthSession>('loading')
  useEffect(() => {
    api.get<AuthSession>('/auth/session').then((session) => setAuthentication(session.authenticated ? session : 'anonymous')).catch(() => setAuthentication('anonymous'))
    const expired = () => setAuthentication('anonymous')
    window.addEventListener('pgsentinel:unauthorized', expired)
    return () => window.removeEventListener('pgsentinel:unauthorized', expired)
  }, [])
  if (authentication === 'loading') return <div className="auth-screen"><div className="auth-loading">Loading PGSentinel…</div></div>
  if (authentication === 'anonymous') return <LoginPage onAuthenticated={setAuthentication} />
  if (authentication.mustChangePassword) return <ChangePasswordPage username={authentication.username} onChanged={() => setAuthentication({ ...authentication, mustChangePassword: false })} onLogout={() => setAuthentication('anonymous')} />
  return <Routes><Route element={<AppLayout username={authentication.username} role={authentication.role} onLogout={() => setAuthentication('anonymous')} />}><Route index element={<OverviewPage />} /><Route path="problems" element={<ProblemsPage />} /><Route path="incidents" element={<IncidentsPage />} /><Route path="servers" element={<ServersPage />} />{authentication.role === 'administrator' && <><Route path="audit" element={<AuditPage />} /><Route path="users" element={<UsersPage currentUser={authentication.username} />} /><Route path="settings" element={<SettingsPage />} /></>}<Route path=":resource" element={<ResourcePage />} /></Route></Routes>
}
