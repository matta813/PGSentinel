import { Activity, AlertTriangle, ChevronsUpDown, Database, FileSearch, Gauge, GitBranch, KeyRound, ListTree, Lock, LogOut, Menu, Moon, ScrollText, Search, Settings, Sun, Table2, Users, Waves, X } from 'lucide-react'
import { NavLink, Outlet } from 'react-router-dom'
import { useEffect, useState } from 'react'
import { api } from '../api/client'
import { useMonitoring, type TimeRange } from '../context/MonitoringContext'
import { useApi } from '../hooks/useApi'

const groups = [
  { label: 'Monitoring', items: [['/', 'Dashboard', Gauge], ['/problems', 'Problems', AlertTriangle], ['/incidents', 'Incidents', ListTree]] },
  { label: 'Performance', items: [['/queries', 'Query Performance', Search], ['/tables', 'Tables', Table2], ['/indexes', 'Index Analysis', KeyRound], ['/vacuum', 'Vacuum', Activity], ['/locks', 'Locks', Lock], ['/replication', 'Replication', GitBranch], ['/wal', 'WAL & Archive', Waves]] },
  { label: 'Management', items: [['/servers', 'Servers', Database]] },
  { label: 'System', admin: true, items: [['/audit', 'Audit Log', ScrollText], ['/users', 'Users', Users], ['/settings', 'Settings', Settings]] },
] as const
const ranges: [TimeRange, string][] = [['1h', 'Last 1 hour'], ['6h', 'Last 6 hours'], ['24h', 'Last 24 hours'], ['7d', 'Last 7 days'], ['30d', 'Last 30 days']]

export function AppLayout({ username, role, onLogout }: { username: string; role: 'administrator' | 'operator' | 'viewer'; onLogout: () => void }) {
  const [open, setOpen] = useState(false)
  const [dark, setDark] = useState(() => localStorage.getItem('theme') === 'dark')
  const monitoring = useMonitoring()
  const { data: build } = useApi(() => api.get<{ version: string; commit: string }>('/version', { cache: 'no-store', headers: { 'Cache-Control': 'no-cache' } }), [])
  useEffect(() => { document.documentElement.dataset.theme = dark ? 'dark' : 'light'; localStorage.setItem('theme', dark ? 'dark' : 'light') }, [dark])
  async function logout() { await api.post('/auth/logout'); onLogout() }
  return <div className="app-shell">
    {open && <button className="nav-scrim" aria-label="Close navigation" onClick={() => setOpen(false)} />}
    <aside className={`sidebar ${open ? 'open' : ''}`}>
      <div className="brand"><span className="brand-mark"><FileSearch /></span><div><strong>PGSentinel</strong><small>Database Intelligence</small></div><button className="icon-button sidebar-close" aria-label="Close navigation" onClick={() => setOpen(false)}><X /></button></div>
      <nav aria-label="Primary navigation">{groups.filter(group => !('admin' in group) || role === 'administrator').map(group => <div className="nav-group" key={group.label}><p>{group.label}</p>{group.items.map(([to, label, Icon]) => <NavLink key={to} to={to} end={to === '/'} onClick={() => setOpen(false)}><Icon /><span>{label}</span></NavLink>)}</div>)}</nav>
      <div className="sidebar-footer"><div className="monitoring-state"><span className="live-dot" /><span><strong>Collector online</strong><small>Monitoring enabled</small></span></div><div className="build-meta"><span>v{build?.version ?? 'dev'}</span>{build?.commit !== 'unknown' && build?.commit && <code>{build.commit.slice(0, 8)}</code>}</div></div>
    </aside>
    <main className="app-main">
      <header className="topbar"><button className="icon-button menu-button" aria-label="Open navigation" onClick={() => setOpen(true)}><Menu /></button><div className="context-controls"><label className="context-selector"><span className={`server-dot ${monitoring.selectedServer?.status?.toLowerCase() ?? 'unknown'}`} /><span className="context-copy"><small>Server</small><strong>{monitoring.selectedServer?.name ?? (monitoring.serversLoading ? 'Loading…' : 'No server')}</strong></span><select aria-label="Global server" value={monitoring.selectedServerId} disabled={!monitoring.servers.length} onChange={event => monitoring.setSelectedServerId(event.target.value)}>{monitoring.servers.map(server => <option value={server.id} key={server.id}>{server.name} · {server.status}</option>)}</select><ChevronsUpDown /></label><label className="context-selector database-context"><Database /><span className="context-copy"><small>Database</small><strong>{monitoring.selectedDatabase || 'All databases'}</strong></span><select aria-label="Global database" value={monitoring.selectedDatabase} disabled={!monitoring.databases.length} onChange={event => monitoring.setSelectedDatabase(event.target.value)}><option value="">All databases</option>{monitoring.databases.map(database => <option key={database.Name}>{database.Name}</option>)}</select><ChevronsUpDown /></label></div>
        <div className="topbar-actions"><label className="range-select"><span className="sr-only">Time range</span><select aria-label="Time range" value={monitoring.timeRange} onChange={event => monitoring.setTimeRange(event.target.value as TimeRange)}>{ranges.map(([value, label]) => <option value={value} key={value}>{label}</option>)}</select></label><button className="icon-button" aria-label={`Switch to ${dark ? 'light' : 'dark'} theme`} onClick={() => setDark(value => !value)}>{dark ? <Sun /> : <Moon />}</button><span className="account-label">{username}<small>{role}</small></span><button className="icon-button" aria-label="Sign out" onClick={() => void logout()}><LogOut /></button></div></header>
      <div className="page"><Outlet /></div>
    </main>
  </div>
}
