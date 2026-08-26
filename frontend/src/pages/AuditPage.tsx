import { ChevronLeft, ChevronRight, Filter, Search, ShieldCheck } from 'lucide-react'
import { useState } from 'react'
import { Empty, ErrorState, Loading } from '../components/Status'
import { PageHeader } from '../components/UI'
import { api } from '../api/client'
import { useApi } from '../hooks/useApi'
import type { AuditEvent } from '../types/audit'

const pageSize = 50
export function AuditPage() {
  const [search, setSearch] = useState('')
  const [action, setAction] = useState('')
  const [resourceType, setResourceType] = useState('')
  const [offset, setOffset] = useState(0)
  const query = new URLSearchParams({ limit: String(pageSize), offset: String(offset) })
  if (search.trim()) query.set('search', search.trim())
  if (action) query.set('action', action)
  if (resourceType) query.set('resourceType', resourceType)
  const { data, error, loading, reload } = useApi(() => api.get<AuditEvent[]>(`/audit-events?${query}`), [search, action, resourceType, offset])
  if (loading) return <Loading />
  if (error) return <ErrorState error={error} retry={reload} />
  return <>
    <PageHeader title="Audit log" description="Review security-sensitive and configuration-changing operator actions." actions={<span className="retention-note"><ShieldCheck /> 365-day bounded retention</span>} />
    <div className="audit-toolbar">
      <label className="search-field"><Search /><span className="sr-only">Search audit events</span><input aria-label="Search audit events" maxLength={200} placeholder="Search actor, action, resource, or summary" value={search} onChange={event => { setSearch(event.target.value); setOffset(0) }} /></label>
      <div className="filter-controls"><span className="filter-label"><Filter /> Filters</span><select aria-label="Audit action" value={action} onChange={event => { setAction(event.target.value); setOffset(0) }}><option value="">All actions</option><option value="auth.login.failed">Failed logins</option><option value="auth.login.succeeded">Successful logins</option><option value="auth.password.changed">Password changes</option><option value="server.created">Targets added</option><option value="server.updated">Targets edited</option><option value="server.credentials_rotated">Credential rotations</option><option value="notification_destination.updated">Destination changes</option><option value="notification_route.updated">Routing changes</option><option value="maintenance_window.created">Maintenance windows</option><option value="suppression.created">Suppressions</option><option value="threshold_override.created">Threshold changes</option><option value="finding.acknowledged">Finding acknowledgements</option></select><select aria-label="Audit resource" value={resourceType} onChange={event => { setResourceType(event.target.value); setOffset(0) }}><option value="">All resources</option><option value="user">Users</option><option value="server">PostgreSQL targets</option><option value="notification_destination">Destinations</option><option value="notification_route">Routing rules</option><option value="maintenance_window">Maintenance windows</option><option value="suppression">Suppressions</option><option value="threshold_override">Thresholds</option><option value="finding">Findings</option></select></div>
    </div>
    {data?.length === 0 ? <Empty title="No audit events" detail="No security or configuration changes match the selected filters." /> : <div className="audit-table"><table><thead><tr><th>Time</th><th>Actor</th><th>Action</th><th>Resource</th><th>Safe summary</th></tr></thead><tbody>{data?.map(event => <tr key={event.id}><td><time dateTime={event.occurredAt}>{formatDate(event.occurredAt)}</time></td><td><strong>{event.actor}</strong></td><td><code>{event.action}</code></td><td><span>{humanize(event.resourceType)}</span>{event.resourceId && <small title={event.resourceId}>{shortID(event.resourceId)}</small>}</td><td>{event.summary}</td></tr>)}</tbody></table></div>}
    {(data?.length ?? 0) > 0 && <footer className="audit-pagination"><span>Showing {offset + 1}–{offset + (data?.length ?? 0)}</span><div><button className="button secondary compact" disabled={offset === 0} onClick={() => setOffset(Math.max(0, offset - pageSize))}><ChevronLeft /> Previous</button><button className="button secondary compact" disabled={(data?.length ?? 0) < pageSize} onClick={() => setOffset(offset + pageSize)}>Next <ChevronRight /></button></div></footer>}
  </>
}

function formatDate(value: string) { const date = new Date(value); return Number.isFinite(date.getTime()) ? date.toLocaleString([], { dateStyle: 'medium', timeStyle: 'medium' }) : 'Unknown time' }
function humanize(value: string) { return value.replaceAll('_', ' ') }
function shortID(value: string) { return value.length > 16 ? `${value.slice(0, 8)}…${value.slice(-4)}` : value }
