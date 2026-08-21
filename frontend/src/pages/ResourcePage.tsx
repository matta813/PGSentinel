import { useState, type ReactNode } from 'react'
import { Database, Info } from 'lucide-react'
import { useParams } from 'react-router-dom'
import { api } from '../api/client'
import { Empty, ErrorState, Loading } from '../components/Status'
import { PageHeader } from '../components/UI'
import { useApi } from '../hooks/useApi'
import type { IndexStat, LockInfo, QueryStat, Server, TableStat } from '../types'

const titles: Record<string, [string, string, string]> = { queries: ['Query performance', 'Queries', 'Total impact, latency, disk reads, temporary I/O, and WAL activity.'], tables: ['Table health', 'Tables', 'Storage footprint, access patterns, and tuple health across monitored tables.'], indexes: ['Index analysis', 'Indexes', 'Usage evidence and cautious identification of potentially unused indexes.'], vacuum: ['Vacuum health', 'Vacuum', 'Dead tuples and progress toward estimated autovacuum thresholds.'], locks: ['Live blocking', 'Locks', 'Current blocked sessions, their blockers, and blocking duration.'] }
export function ResourcePage() {
  const { resource = 'queries' } = useParams()
  const [server, setServer] = useState('')
  const servers = useApi(() => api.get<Server[]>('/servers'), [])
  const selected = server || servers.data?.[0]?.id || ''
  const result = useApi(() => selected ? api.get<unknown[]>(`/servers/${selected}/${resource}`) : Promise.resolve([]), [selected, resource])
  if (servers.loading || result.loading) return <Loading />
  if (servers.error || result.error) return <ErrorState error={servers.error ?? result.error!} retry={result.reload} />
  const title = titles[resource] ?? ['Monitoring data', 'Evidence', 'Collected PostgreSQL evidence.']
  const currentServer = servers.data?.find(item => item.id === selected)
  return <><PageHeader title={title[0]} description={title[2]} actions={<label className="server-select"><Database /><span className="sr-only">Server</span><select aria-label="Server" value={selected} onChange={event => setServer(event.target.value)}>{servers.data?.map(item => <option value={item.id} key={item.id}>{item.name}</option>)}</select></label>} />
    {selected && <div className="table-context"><div><strong>{title[1]}</strong><span>{result.data?.length ?? 0} rows · {currentServer?.name}</span></div><span><Info /> Latest collected snapshot</span></div>}
    {!selected ? <Empty title="No server configured" detail="Add a PostgreSQL server before viewing collected evidence." /> : <ResourceTable resource={resource} rows={result.data ?? []} />}</>
}
function ResourceTable({ resource, rows }: { resource: string; rows: unknown[] }) {
  if (rows.length === 0) return <Empty title="No data collected yet" detail="Evidence will appear after a successful monitoring cycle." />
  if (resource === 'queries') return <Table headers={['Query', 'Database', 'Calls', 'Avg latency', 'Total runtime', 'Impact']} numeric={[2, 3, 4, 5]} rows={(rows as QueryStat[]).map(query => [<code className="query-text" title={query.Query}>{query.Query}</code>, query.Database, fmt(query.Calls), `${fmt(query.MeanExecMS)} ms`, `${fmt(query.TotalExecMS)} ms`, query.ImpactScore.toFixed(1)])} />
  if (resource === 'indexes') return <Table headers={['Database', 'Index', 'Table', 'Size', 'Scans', 'Assessment']} numeric={[3, 4]} rows={(rows as IndexStat[]).map(index => [index.Database, <code>{index.Index}</code>, <code>{index.Schema}.{index.Table}</code>, bytes(index.SizeBytes), fmt(index.Scans), <span className={index.Scans === 0 && !index.Primary && !index.Unique ? 'assessment warning' : 'assessment'}>{index.Scans === 0 && !index.Primary && !index.Unique ? 'Potentially unused' : 'Observed usage'}</span>])} />
  if (resource === 'locks') return <Table headers={['Blocked PID', 'Blocking PID', 'Duration', 'Database', 'Application', 'Query']} numeric={[0, 1, 2]} rows={(rows as LockInfo[]).map(lock => [<code>{lock.BlockedPID}</code>, <code>{lock.BlockingPID}</code>, `${fmt(lock.DurationSeconds)} sec`, lock.Database, lock.Application, <code className="query-text" title={lock.Query}>{lock.Query}</code>])} />
  const table = rows as TableStat[]
  return <Table headers={['Database', 'Table', 'Rows', 'Total size', 'Dead tuples', resource === 'vacuum' ? 'Autovacuum trigger' : 'Index scans', resource === 'vacuum' ? 'Progress' : 'Last autovacuum']} numeric={[2, 3, 4, 5, 6]} rows={table.map(item => [item.Database, <code>{item.Schema}.{item.Table}</code>, fmt(item.EstimatedRows), bytes(item.TotalSize), `${fmt(item.DeadTuples)} (${deadRatio(item)}%)`, resource === 'vacuum' ? fmt(item.VacuumThreshold) : fmt(item.IndexScans), resource === 'vacuum' ? `${Math.round(item.VacuumProgress)}%` : item.LastAutovacuum ? new Date(item.LastAutovacuum).toLocaleString() : 'Never'])} />
}
function Table({ headers, rows, numeric = [] }: { headers: string[]; rows: ReactNode[][]; numeric?: number[] }) { return <div className="data-table"><table><thead><tr>{headers.map((header, index) => <th className={numeric.includes(index) ? 'numeric' : ''} key={header}>{header}</th>)}</tr></thead><tbody>{rows.map((row, rowIndex) => <tr key={rowIndex}>{row.map((value, columnIndex) => <td className={numeric.includes(columnIndex) ? 'numeric' : ''} key={columnIndex}>{value}</td>)}</tr>)}</tbody></table></div> }
function fmt(value: number) { return Intl.NumberFormat('en', { notation: value > 9999 ? 'compact' : 'standard', maximumFractionDigits: 1 }).format(value || 0) }
function bytes(value: number) { if (value >= 1024 ** 3) return `${fmt(value / 1024 ** 3)} GB`; if (value >= 1024 ** 2) return `${fmt(value / 1024 ** 2)} MB`; return `${fmt(value / 1024)} KB` }
function deadRatio(table: TableStat) { const count = table.LiveTuples + table.DeadTuples; return count ? ((table.DeadTuples / count) * 100).toFixed(1) : '0' }
