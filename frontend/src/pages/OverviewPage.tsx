import { ArrowRight, Server as ServerIcon } from 'lucide-react'
import { Link } from 'react-router-dom'
import { api } from '../api/client'
import { HealthScore } from '../components/HealthScore'
import { Empty, ErrorState, Loading, SeverityBadge, StatusIndicator } from '../components/Status'
import { PageHeader, SectionHeader } from '../components/UI'
import { useApi } from '../hooks/useApi'
import type { Overview } from '../types'

export function OverviewPage() {
  const { data, error, loading, reload } = useApi(() => api.get<Overview>('/overview'), [])
  if (loading) return <Loading />
  if (error || !data) return <ErrorState error={error ?? new Error('No data')} retry={reload} />
  const urgent = (data.counts.CRITICAL ?? 0) + (data.counts.HIGH ?? 0)
  const connected = data.servers.filter(server => server.status.toLowerCase() === 'healthy').length
  const lastCollection = data.servers.map(server => server.lastConnectedAt).filter(Boolean).sort().at(-1)
  return <>
    <PageHeader title="Overview" description="Infrastructure health across your PostgreSQL estate." meta={lastCollection ? `Last collection ${relativeTime(lastCollection)}` : 'Awaiting first collection'} />
    <div className="overview-top"><HealthScore value={data.score.overall} /><section className="estate-summary"><div className="estate-heading"><span>Estate</span><strong>{data.servers.length} {data.servers.length === 1 ? 'server' : 'servers'}</strong><small>{connected} online</small></div><div className="estate-metrics"><Metric label="Active findings" value={data.problems.length} detail={data.problems.length ? 'Require triage' : 'Inbox clear'} /><Metric label="Critical / high" value={urgent} detail={urgent ? 'Immediate attention' : 'No urgent findings'} tone={urgent ? 'danger' : 'success'} /></div></section></div>
    <section className="category-health"><div className="category-heading"><h2>Health categories</h2><p>Score by analysis area</p></div><div className="category-list">{Object.entries(data.score.categories).length ? Object.entries(data.score.categories).map(([name, score]) => <div className="category-row" key={name}><span>{humanize(name)}</span><div className="category-track"><i style={{ width: `${Math.max(0, Math.min(100, score))}%` }} /></div><strong>{score}</strong></div>) : <span className="muted">Category scores appear after the first analysis.</span>}</div></section>
    <SectionHeader title="Needs attention" description="Highest-priority active findings across the monitored estate." action={<Link className="text-link" to="/problems">View operations inbox <ArrowRight /></Link>} />
    {data.problems.length === 0 ? <Empty positive title="No active findings" detail="Your PostgreSQL estate currently has no actionable findings." action={<Link className="text-link" to="/problems">View inbox <ArrowRight /></Link>} /> : <div className="attention-list">{data.problems.slice(0, 6).map(finding => <Link className={`attention-row severity-${finding.severity.toLowerCase()}`} to={`/problems?id=${finding.id}`} key={finding.id}><SeverityBadge severity={finding.severity} /><div className="attention-copy"><strong>{finding.title}</strong><span>{serverName(data, finding.serverId)}{finding.database && ` / ${finding.database}`} · {finding.category}</span></div><p>{finding.summary}</p><span className="row-arrow"><ArrowRight /></span></Link>)}</div>}
    <SectionHeader title="Monitored servers" description="Connection state and latest PostgreSQL identity." action={<Link className="text-link" to="/servers">Manage servers <ArrowRight /></Link>} />
    {data.servers.length === 0 ? <Empty title="No servers yet" detail="Add your first PostgreSQL server to start monitoring." action={<Link className="button primary" to="/servers">Add server</Link>} /> : <div className="overview-servers">{data.servers.map(server => <article className="overview-server" key={server.id}><span className="server-icon"><ServerIcon /></span><div><strong>{server.name}</strong><code>{server.host}:{server.port}</code></div><div className="server-version"><span>PostgreSQL</span><strong>{server.version ?? 'Not reported'}</strong></div><StatusIndicator status={server.status} /></article>)}</div>}
  </>
}

function Metric({ label, value, detail, tone = '' }: { label: string; value: number; detail: string; tone?: string }) { return <div className={`summary-metric ${tone}`}><span>{label}</span><strong>{value}</strong><small>{detail}</small></div> }
function serverName(overview: Overview, id: string) { return overview.servers.find(server => server.id === id)?.name ?? 'Unknown server' }
function humanize(value: string) { return value.replace(/([a-z])([A-Z])/g, '$1 $2').replace(/[_-]/g, ' ').replace(/^./, char => char.toUpperCase()) }
function relativeTime(value: string) { const minutes = Math.max(0, Math.floor((Date.now() - new Date(value).getTime()) / 60000)); if (minutes < 1) return 'just now'; if (minutes < 60) return `${minutes}m ago`; const hours = Math.floor(minutes / 60); return hours < 24 ? `${hours}h ago` : `${Math.floor(hours / 24)}d ago` }
