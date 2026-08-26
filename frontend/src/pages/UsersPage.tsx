import { FormEvent, useState } from 'react'
import { UserPlus } from 'lucide-react'
import { api, APIError } from '../api/client'
import { ErrorState, Loading } from '../components/Status'
import { Notice, PageHeader, SectionHeader } from '../components/UI'
import { useApi } from '../hooks/useApi'
import type { AuthSession, UserAccount } from '../types/auth'

const roles: AuthSession['role'][] = ['administrator', 'operator', 'viewer']

export function UsersPage({ currentUser }: { currentUser: string }) {
  const users = useApi(() => api.get<UserAccount[]>('/users'), [])
  const [form, setForm] = useState({ username: '', password: '', role: 'viewer' as AuthSession['role'] })
  const [status, setStatus] = useState('')
  async function create(event: FormEvent) {
    event.preventDefault(); setStatus('Creating user…')
    try { await api.post('/users', form); setForm({ username: '', password: '', role: 'viewer' }); setStatus('User created. They must change the initial password at first login.'); void users.reload() }
    catch (reason) { setStatus(reason instanceof APIError ? reason.message : 'Unable to create user') }
  }
  if (users.loading) return <Loading />
  if (users.error) return <ErrorState error={users.error} retry={users.reload} />
  return <><PageHeader title="Users" description="Manage the small, explicit access model for this PGSentinel installation." />
    <div className="settings-content single-column"><section><SectionHeader title="Add user" description="Initial passwords are hashed with Argon2id and must be changed at first login." />
      <form className="settings-form" onSubmit={create}><div className="form-grid three"><label>Username<input required maxLength={100} autoComplete="off" value={form.username} onChange={event => setForm({ ...form, username: event.target.value })} /></label><label>Initial password<input required minLength={12} type="password" autoComplete="new-password" value={form.password} onChange={event => setForm({ ...form, password: event.target.value })} /></label><label>Role<select value={form.role} onChange={event => setForm({ ...form, role: event.target.value as AuthSession['role'] })}>{roles.map(role => <option key={role}>{role}</option>)}</select></label></div><div className="form-actions"><button className="button primary"><UserPlus />Add user</button></div></form>
    </section><section><SectionHeader title="Accounts" description="Role changes immediately invalidate that user's active sessions." /><div className="table-scroll"><table className="data-table"><thead><tr><th>User</th><th>Role</th><th>Password</th><th>Created</th></tr></thead><tbody>{users.data?.map(user => <tr key={user.id}><td><strong>{user.username}</strong>{user.username === currentUser && <small>Current account</small>}</td><td><select aria-label={`Role for ${user.username}`} disabled={user.username === currentUser} value={user.role} onChange={async event => { await api.put(`/users/${user.id}/role`, { role: event.target.value }); setStatus(`Role updated for ${user.username}.`); void users.reload() }}>{roles.map(role => <option key={role}>{role}</option>)}</select></td><td>{user.mustChangePassword ? 'Change required' : 'Set'}</td><td>{new Date(user.createdAt).toLocaleDateString()}</td></tr>)}</tbody></table></div></section>{status && <Notice>{status}</Notice>}</div>
  </>
}
