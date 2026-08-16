import { useEffect, useState } from 'react'
import { Route, Routes } from 'react-router-dom'
import { api } from './api/client'
import { AppLayout } from './layouts/AppLayout'
import { LoginPage } from './pages/LoginPage'
import { OverviewPage } from './pages/OverviewPage'
import { ProblemsPage } from './pages/ProblemsPage'
import { ResourcePage } from './pages/ResourcePage'
import { ServersPage } from './pages/ServersPage'
import { SettingsPage } from './pages/SettingsPage'

export function App() {
  const [authentication, setAuthentication] = useState<'loading' | 'authenticated' | 'anonymous'>('loading')
  useEffect(() => {
    api.get<{ authenticated: boolean }>('/auth/session').then(() => setAuthentication('authenticated')).catch(() => setAuthentication('anonymous'))
    const expired = () => setAuthentication('anonymous')
    window.addEventListener('pgsentinel:unauthorized', expired)
    return () => window.removeEventListener('pgsentinel:unauthorized', expired)
  }, [])
  if (authentication === 'loading') return <div className="auth-screen"><div className="auth-loading">Loading PGSentinel…</div></div>
  if (authentication === 'anonymous') return <LoginPage onAuthenticated={() => setAuthentication('authenticated')} />
  return <Routes><Route element={<AppLayout onLogout={() => setAuthentication('anonymous')} />}><Route index element={<OverviewPage />} /><Route path="problems" element={<ProblemsPage />} /><Route path="servers" element={<ServersPage />} /><Route path=":resource" element={<ResourcePage />} /><Route path="settings" element={<SettingsPage />} /></Route></Routes>
}
