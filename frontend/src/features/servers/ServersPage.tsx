import { Plus } from "lucide-react";
import { useState, type FormEvent } from "react";
import { api, APIError } from "../../api/client";
import { Empty, ErrorState, Loading } from "../../components/Status";
import { Notice, PageHeader } from "../../components/UI";
import { useMonitoring } from "../../context/MonitoringContext";
import { useApi } from "../../hooks/useApi";
import type { Server } from "../../types";
import { ServerEditor } from "./ServerEditor";
import { ServerTable } from "./ServerTable";
import { initialServerForm, serverFormValue } from "./serverForm";

export function ServersPage() {
  const monitoring = useMonitoring();
  const { data, error, loading, reload } = useApi(() => api.get<Server[]>("/servers"), []);
  const [form, setForm] = useState(initialServerForm);
  const [editingID, setEditingID] = useState("");
  const [open, setOpen] = useState(false);
  const [message, setMessage] = useState("");
  async function reloadServerLists() { await Promise.all([reload(), monitoring.reloadServers()]); }
  function closeForm() { setForm(initialServerForm); setEditingID(""); setOpen(false); }
  function edit(server: Server) { setForm(serverFormValue(server)); setEditingID(server.id); setOpen(true); }
  async function submit(event: FormEvent) {
    event.preventDefault(); setMessage("");
    try {
      if (editingID) await api.put(`/servers/${editingID}`, form); else await api.post("/servers", form);
      closeForm(); await reloadServerLists();
    } catch (reason) { setMessage(reason instanceof APIError ? `${reason.message}. ${reason.detail}` : "Unable to save server"); }
  }
  async function test(id: string) {
    setMessage("Testing connection…");
    try { const value = await api.post<{ version: string }>(`/servers/${id}/test`); setMessage(`Connected to PostgreSQL ${value.version}`); await reloadServerLists(); }
    catch (reason) { setMessage(reason instanceof APIError ? `${reason.message}. ${reason.detail}` : "Connection failed"); }
  }
  if (loading) return <Loading />;
  if (error) return <ErrorState error={error} retry={reload} />;
  const add = () => { closeForm(); setOpen(true); };
  return <>
    <PageHeader title="Servers" description="PostgreSQL instances monitored by this PGSentinel deployment." actions={<button className="button primary" onClick={add}><Plus /> Add server</button>} />
    {message && <Notice>{message}</Notice>}
    {open && <ServerEditor value={form} setValue={setForm} editingID={editingID} close={closeForm} submit={submit} />}
    {data?.length === 0
      ? <Empty title="No servers yet" detail="Add your first PostgreSQL server to begin collection and health analysis." action={<button className="button primary" onClick={() => setOpen(true)}><Plus /> Add server</button>} />
      : <ServerTable servers={data ?? []} edit={edit} test={test} remove={async (server) => { if (confirm(`Delete ${server.name}?`)) { await api.delete(`/servers/${server.id}`); await reloadServerLists(); } }} />}
  </>;
}
