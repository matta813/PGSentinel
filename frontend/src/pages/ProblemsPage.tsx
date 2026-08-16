import { useState } from 'react';
import { Check, RotateCcw } from 'lucide-react';
import { api, APIError } from '../api/client';
import { Empty, ErrorState, Loading, SeverityBadge } from '../components/Status';
import { useApi } from '../hooks/useApi';
import type { Finding, Severity } from '../types';

export function ProblemsPage() {
  const [severity, setSeverity] = useState('');
  const [status, setStatus] = useState('active');
  const [category, setCategory] = useState('');
  const [search, setSearch] = useState('');
  const [message, setMessage] = useState('');
  const query = new URLSearchParams({ status });
  if (severity) query.set('severity', severity);
  if (category.trim()) query.set('category', category.trim());
  if (search.trim()) query.set('search', search.trim());
  const { data, error, loading, reload } = useApi(() => api.get<Finding[]>(`/problems?${query}`), [status, severity, category, search]);
  async function setFindingStatus(id: string, nextStatus: 'active' | 'acknowledged') {
    setMessage('');
    try { await api.put(`/problems/${id}/status`, { status: nextStatus }); void reload(); }
    catch (e) { setMessage(e instanceof APIError ? `${e.message}: ${e.detail}` : 'Unable to update problem'); }
  }
  if (loading) return <Loading />;
  if (error) return <ErrorState error={error} retry={reload} />;
  return <>
    <div className="title-row"><div><p className="eyebrow">Problems inbox</p><h1>What needs attention</h1><p>Acknowledge investigated findings without hiding their health impact.</p></div></div>
    {message && <div className="notice">{message}</div>}
    <div className="filters">
      <select aria-label="Severity" value={severity} onChange={e => setSeverity(e.target.value)}><option value="">All severities</option>{(['CRITICAL', 'HIGH', 'MEDIUM', 'LOW', 'INFO'] as Severity[]).map(s => <option key={s}>{s}</option>)}</select>
      <select aria-label="Status" value={status} onChange={e => setStatus(e.target.value)}><option value="active">Active</option><option value="acknowledged">Acknowledged</option><option value="resolved">Resolved</option><option value="all">All</option></select>
      <input aria-label="Category" placeholder="Category" value={category} onChange={e => setCategory(e.target.value)} />
      <input aria-label="Search problems" placeholder="Search title, evidence…" value={search} maxLength={200} onChange={e => setSearch(e.target.value)} />
    </div>
    {data?.length === 0 ? <Empty title="Inbox is clear" detail="No problems match the selected filters." /> : <div className="inbox">{data?.map(f => <details key={f.id} className="finding"><summary><SeverityBadge severity={f.severity} /><div><strong>{f.title}</strong><span>{f.database || 'Server'}{f.resource && ` · ${f.resource}`} · {f.category} · {f.status}</span><p>{f.summary}</p></div></summary><div className="finding-detail"><h3>Why it matters</h3><p>{f.impact}</p><h3>Observed evidence</h3><div className="evidence">{f.evidence?.map(e => <div key={e.label}><span>{e.label}</span><strong>{e.value}</strong></div>)}</div><h3>Suggested investigation</h3><ol>{f.suggestions?.map((s, i) => <li key={i}>{s.title}{s.detail && <small>{s.detail}</small>}</li>)}</ol><p className="confidence">Confidence: <strong>{f.confidence}</strong></p>{f.status === 'active' && <button onClick={() => void setFindingStatus(f.id, 'acknowledged')}><Check /> Acknowledge</button>}{f.status === 'acknowledged' && <button className="secondary" onClick={() => void setFindingStatus(f.id, 'active')}><RotateCcw /> Reopen</button>}</div></details>)}</div>}
  </>;
}
