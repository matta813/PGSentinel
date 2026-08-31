import { useState, type FormEvent } from 'react'
import { Rocket, Trash2 } from 'lucide-react'
import { api, APIError } from '../api/client'
import { Empty, ErrorState, Loading } from './Status'
import { Notice, SectionHeader } from './UI'
import { useApi } from '../hooks/useApi'
import type { Server } from '../types'
import type { ChangeEvent } from '../types/change-events'

export function ChangeHistorySettings() {
  const servers = useApi(() => api.get<Server[]>('/servers'), [])
  const [serverId, setServerId] = useState('')
  const [summary, setSummary] = useState('')
  const [occurredAt, setOccurredAt] = useState(() => new Date().toISOString().slice(0, 16))
  const [message, setMessage] = useState('')
  const events = useApi(() => serverId ? api.get<ChangeEvent[]>(`/change-events?serverId=${encodeURIComponent(serverId)}`) : Promise.resolve([]), [serverId])
  async function submit(event: FormEvent) {
    event.preventDefault(); setMessage('Recording deployment…')
    try { await api.post('/deployments', { serverId, summary, occurredAt: new Date(occurredAt).toISOString() }); setSummary(''); setMessage('Deployment marker recorded.'); void events.reload() }
    catch (reason) { setMessage(reason instanceof APIError ? `${reason.message}: ${reason.detail}` : 'Unable to record deployment') }
  }
  if (servers.loading) return <Loading />
  if (servers.error) return <ErrorState error={servers.error} retry={servers.reload} />
  return <section id="change-history"><SectionHeader title="Change history" description="Record deployments and review detected PostgreSQL setting changes used to contextualize query regressions." />
    <form className="settings-form" onSubmit={submit}><div className="form-grid two"><label>Server<select required value={serverId} onChange={event => setServerId(event.target.value)}><option value="">Select server</option>{servers.data?.map(server => <option key={server.id} value={server.id}>{server.name}</option>)}</select></label><label>Occurred at<input required type="datetime-local" value={occurredAt} onChange={event => setOccurredAt(event.target.value)} /></label></div><label>Deployment summary<input required maxLength={300} value={summary} onChange={event => setSummary(event.target.value)} placeholder="Released checkout service 2.4.0" /></label><div className="form-actions"><button className="button primary" disabled={!serverId}><Rocket /> Record deployment</button></div></form>
    {events.loading ? <Loading /> : events.error ? <ErrorState error={events.error} retry={events.reload} /> : events.data?.length === 0 ? <Empty title="No recorded changes" detail={serverId ? 'Deployments and detected setting changes will appear here.' : 'Select a server to view its change history.'} /> : <div className="route-list">{events.data?.map(item => <article key={item.id}><div><strong>{item.summary}</strong><span>{item.kind} · {new Date(item.occurredAt).toLocaleString()}</span>{item.details.map(detail => <small key={detail}>{detail}</small>)}</div>{item.kind === 'deployment' && <button className="icon-button danger" aria-label={`Delete ${item.summary}`} onClick={async () => { if (confirm(`Delete deployment marker ${item.summary}?`)) { await api.delete(`/deployments/${item.id}`); void events.reload() } }}><Trash2 /></button>}</article>)}</div>}
    {message && <Notice>{message}</Notice>}
  </section>
}
