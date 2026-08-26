import { Activity, AlertTriangle, ChevronRight, Database, FileSearch, Gauge, GitBranch, KeyRound, ListTree, Lock, LogOut, Menu, Moon, ScrollText, Search, Settings, Sun, Table2, Users, Waves, X } from 'lucide-react'
import { NavLink, Outlet, useLocation } from 'react-router-dom'
import { useEffect, useState } from 'react'
import { api } from '../api/client'
import { useApi } from '../hooks/useApi'

const groups = [
  { label: 'Monitor', items: [['/', 'Overview', Gauge], ['/problems', 'Problems', AlertTriangle], ['/incidents', 'Incidents', ListTree], ['/servers', 'Servers', Database]] },
  { label: 'Database', items: [['/queries', 'Queries', Search], ['/tables', 'Tables', Table2], ['/indexes', 'Indexes', KeyRound], ['/vacuum', 'Vacuum', Activity], ['/locks', 'Locks', Lock], ['/replication', 'Replication', GitBranch], ['/wal', 'WAL & archive', Waves]] },
  { label: 'System', items: [['/audit', 'Audit log', ScrollText], ['/users', 'Users', Users], ['/settings', 'Settings', Settings]] },
] as const
const routeNames: Record<string, string> = { '/': 'Overview', '/problems': 'Problems', '/incidents': 'Incidents', '/servers': 'Servers', '/audit': 'Audit log', '/users': 'Users', '/queries': 'Queries', '/tables': 'Tables', '/indexes': 'Indexes', '/vacuum': 'Vacuum', '/locks': 'Locks', '/replication': 'Replication', '/wal': 'WAL & archive', '/settings': 'Settings' }

export function AppLayout({ username, role, onLogout }: { username: string; role: 'administrator' | 'operator' | 'viewer'; onLogout: () => void }) {
  const [open, setOpen] = useState(false)
  const [dark, setDark] = useState(() => localStorage.theme !== 'light')
  const { pathname } = useLocation()
  const { data: build } = useApi(() => api.get<{ version: string; commit: string }>('/version'), [])
  useEffect(() => { document.documentElement.dataset.theme = dark ? 'dark' : 'light'; localStorage.theme = dark ? 'dark' : 'light' }, [dark])
  async function logout() { await api.post('/auth/logout'); onLogout() }
  return <div className="app-shell">
    {open && <button className="nav-scrim" aria-label="Close navigation" onClick={() => setOpen(false)} />}
    <aside className={`sidebar ${open ? 'open' : ''}`}>
      <div className="brand"><span className="brand-mark"><FileSearch /></span><div><strong>PGSentinel</strong><small>Database Intelligence</small></div><button className="icon-button sidebar-close" aria-label="Close navigation" onClick={() => setOpen(false)}><X /></button></div>
      <nav aria-label="Primary navigation">{groups.filter(group => role === 'administrator' || group.label !== 'System').map(group => <div className="nav-group" key={group.label}><p>{group.label}</p>{group.items.map(([to, label, Icon]) => <NavLink key={to} to={to} end={to === '/'} onClick={() => setOpen(false)}><Icon /><span>{label}</span></NavLink>)}</div>)}</nav>
      <div className="sidebar-footer"><div className="monitoring-state"><span className="live-dot" /><strong>Collector online</strong></div><div className="build-meta"><span>v{build?.version ?? 'dev'}</span>{build?.commit !== 'unknown' && build?.commit && <code>{build.commit.slice(0, 8)}</code>}</div></div>
    </aside>
    <main className="app-main">
      <header className="topbar"><div className="topbar-context"><button className="icon-button menu-button" aria-label="Open navigation" onClick={() => setOpen(true)}><Menu /></button><span>PGSentinel</span><ChevronRight /><strong>{routeNames[pathname] ?? 'Database'}</strong></div><div className="topbar-actions"><span className="topbar-status"><i />Live</span><span className="topbar-divider" /><span className="account-label">{username}<small>{role}</small></span><button className="icon-button" aria-label={`Switch to ${dark ? 'light' : 'dark'} theme`} title="Toggle color theme" onClick={() => setDark(value => !value)}>{dark ? <Sun /> : <Moon />}</button><button className="icon-button" aria-label="Sign out" title="Sign out" onClick={() => void logout()}><LogOut /></button></div></header>
      <div className="page"><Outlet /></div>
    </main>
  </div>
}
