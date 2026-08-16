import { FormEvent, useState } from 'react';
import { Pencil, Plus, Trash2, Wifi } from 'lucide-react';
import { api, APIError } from '../api/client';
import { ErrorState, Loading, Empty } from '../components/Status';
import { useApi } from '../hooks/useApi';
import type { Server } from '../types';

const initial = { name: '', host: '', port: 5432, user: 'pgsentinel', password: '', sslMode: 'prefer', tags: [] as string[] };

export function ServersPage() {
  const { data, error, loading, reload } = useApi(() => api.get<Server[]>('/servers'), []);
  const [form, setForm] = useState(initial);
  const [editingID, setEditingID] = useState('');
  const [open, setOpen] = useState(false);
  const [message, setMessage] = useState('');

  function closeForm() {
    setForm(initial);
    setEditingID('');
    setOpen(false);
  }
  function edit(server: Server) {
    setForm({ name: server.name, host: server.host, port: server.port, user: server.user, password: '', sslMode: server.sslMode, tags: server.tags });
    setEditingID(server.id);
    setOpen(true);
  }
  async function submit(e: FormEvent) {
    e.preventDefault();
    setMessage('');
    try {
      if (editingID) await api.put(`/servers/${editingID}`, form);
      else await api.post('/servers', form);
      closeForm();
      void reload();
    } catch (e) {
      setMessage(e instanceof APIError ? `${e.message}. ${e.detail}` : 'Unable to save server');
    }
  }
  async function test(id: string) {
    setMessage('Testing connection…');
    try {
      const v = await api.post<{ version: string }>(`/servers/${id}/test`);
      setMessage(`Connected to PostgreSQL ${v.version}`);
      void reload();
    } catch (e) {
      setMessage(e instanceof APIError ? `${e.message}. ${e.detail}` : 'Connection failed');
    }
  }
  if (loading) return <Loading />;
  if (error) return <ErrorState error={error} retry={reload} />;
  return <>
    <div className="title-row"><div><p className="eyebrow">Monitoring targets</p><h1>PostgreSQL servers</h1><p>Credentials are encrypted at rest and never returned to this browser.</p></div><button onClick={() => { closeForm(); setOpen(true); }}><Plus /> Add server</button></div>
    {message && <div className="notice">{message}</div>}
    {open && <form className="server-form" onSubmit={submit}>
      <label>Name<input required value={form.name} onChange={e => setForm({ ...form, name: e.target.value })} /></label>
      <label>Host<input required value={form.host} onChange={e => setForm({ ...form, host: e.target.value })} /></label>
      <label>Port<input required type="number" value={form.port} onChange={e => setForm({ ...form, port: Number(e.target.value) })} /></label>
      <label>User<input required value={form.user} onChange={e => setForm({ ...form, user: e.target.value })} /></label>
      <label>Password{editingID && ' (leave empty to keep current)'}<input required={!editingID} type="password" autoComplete="new-password" value={form.password} onChange={e => setForm({ ...form, password: e.target.value })} /></label>
      <label>SSL mode<select value={form.sslMode} onChange={e => setForm({ ...form, sslMode: e.target.value })}>{['prefer', 'require', 'verify-ca', 'verify-full', 'disable'].map(v => <option key={v}>{v}</option>)}</select></label>
      <div className="form-actions"><button type="button" className="secondary" onClick={closeForm}>Cancel</button><button type="submit">{editingID ? 'Update server' : 'Save server'}</button></div>
    </form>}
    {data?.length === 0 ? <Empty title="No servers configured" detail="Add a PostgreSQL server to begin health analysis." /> : <div className="server-table">{data?.map(s => <article key={s.id}><div><strong>{s.name}</strong><span>{s.host}:{s.port} · {s.user} · SSL {s.sslMode}</span><small>{s.version || 'Not connected yet'}</small></div><span className={`status ${s.status}`}>{s.status}</span><button className="secondary" onClick={() => edit(s)}><Pencil /> Edit</button><button className="secondary" onClick={() => void test(s.id)}><Wifi /> Test</button><button className="danger icon" title="Delete server" onClick={async () => { if (confirm(`Delete ${s.name}?`)) { await api.delete(`/servers/${s.id}`); void reload(); } }}><Trash2 /></button></article>)}</div>}
  </>;
}
