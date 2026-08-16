import { FormEvent, useState } from 'react';
import { Save, Send, Trash2 } from 'lucide-react';
import { api, APIError } from '../api/client';
import { useApi } from '../hooks/useApi';
import type { NotificationDestination } from '../types/notifications';

export function SettingsPage() {
  const destinations = useApi(() => api.get<NotificationDestination[]>('/notifications'), []);
  const [provider, setProvider] = useState('ntfy');
  const [name, setName] = useState('');
  const [form, setForm] = useState({ serverUrl: 'https://ntfy.sh', topic: '', token: '', username: '', password: '', webhookUrl: '' });
  const [status, setStatus] = useState('');
  const payload = { name, provider, enabled: true, ...form };

  async function test(e: FormEvent) {
    e.preventDefault();
    setStatus('Sending…');
    try {
      await api.post('/notifications/test', payload);
      setStatus('Test notification delivered successfully.');
    } catch (e) {
      setStatus(e instanceof APIError ? `${e.message}: ${e.detail}` : 'Notification failed');
    }
  }
  async function save() {
    setStatus('Saving…');
    try {
      await api.post('/notifications', payload);
      setName('');
      setStatus('Notification destination saved securely.');
      void destinations.reload();
    } catch (e) {
      setStatus(e instanceof APIError ? `${e.message}: ${e.detail}` : 'Unable to save destination');
    }
  }
  return <>
    <div className="title-row"><div><p className="eyebrow">Settings</p><h1>Notifications</h1><p>Save encrypted destinations and verify delivery before depending on alerts.</p></div></div>
    <form className="settings-card" onSubmit={test}>
      <label>Name<input required value={name} onChange={e => setName(e.target.value)} placeholder="Operations" /></label>
      <label>Provider<select value={provider} onChange={e => setProvider(e.target.value)}><option value="ntfy">ntfy</option><option value="webhook">Generic webhook</option></select></label>
      {provider === 'ntfy' ? <><label>Server URL<input type="url" required value={form.serverUrl} onChange={e => setForm({ ...form, serverUrl: e.target.value })} /></label><label>Topic<input required value={form.topic} onChange={e => setForm({ ...form, topic: e.target.value })} /></label><label>Token (optional)<input type="password" value={form.token} onChange={e => setForm({ ...form, token: e.target.value })} /></label><label>Username (optional)<input value={form.username} onChange={e => setForm({ ...form, username: e.target.value })} /></label><label>Password (optional)<input type="password" value={form.password} onChange={e => setForm({ ...form, password: e.target.value })} /></label></> : <label>Webhook URL<input type="url" required value={form.webhookUrl} onChange={e => setForm({ ...form, webhookUrl: e.target.value })} /></label>}
      <div className="form-actions"><button type="button" className="secondary" onClick={() => void save()}><Save /> Save destination</button><button type="submit"><Send /> Send test</button></div>
      {status && <div className="notice">{status}</div>}
    </form>
    <div className="section-title"><h2>Saved destinations</h2></div>
    <div className="server-table">{destinations.data?.map(destination => <article key={destination.id}><div><strong>{destination.name}</strong><span>{destination.provider} · {destination.enabled ? 'enabled' : 'disabled'}</span></div><button className="danger icon" title="Delete destination" onClick={async () => { if (confirm(`Delete ${destination.name}?`)) { await api.delete(`/notifications/${destination.id}`); void destinations.reload(); } }}><Trash2 /></button></article>)}</div>
  </>;
}
