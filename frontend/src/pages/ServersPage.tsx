import { FormEvent, useState } from "react";
import { Pencil, Plus, Trash2, Wifi, X } from "lucide-react";
import { api, APIError } from "../api/client";
import {
  Empty,
  ErrorState,
  Loading,
  StatusIndicator,
} from "../components/Status";
import { Notice, PageHeader } from "../components/UI";
import { useApi } from "../hooks/useApi";
import type { Server } from "../types";

const initial = {
  name: "",
  host: "",
  port: 5432,
  user: "pgsentinel",
  password: "",
  sslMode: "prefer",
  tags: [] as string[],
};
export function ServersPage() {
  const { data, error, loading, reload } = useApi(
    () => api.get<Server[]>("/servers"),
    [],
  );
  const [form, setForm] = useState(initial);
  const [editingID, setEditingID] = useState("");
  const [open, setOpen] = useState(false);
  const [message, setMessage] = useState("");
  function closeForm() {
    setForm(initial);
    setEditingID("");
    setOpen(false);
  }
  function edit(server: Server) {
    setForm({
      name: server.name,
      host: server.host,
      port: server.port,
      user: server.user,
      password: "",
      sslMode: server.sslMode,
      tags: server.tags,
    });
    setEditingID(server.id);
    setOpen(true);
  }
  async function submit(event: FormEvent) {
    event.preventDefault();
    setMessage("");
    try {
      if (editingID) await api.put(`/servers/${editingID}`, form);
      else await api.post("/servers", form);
      closeForm();
      void reload();
    } catch (reason) {
      setMessage(
        reason instanceof APIError
          ? `${reason.message}. ${reason.detail}`
          : "Unable to save server",
      );
    }
  }
  async function test(id: string) {
    setMessage("Testing connection…");
    try {
      const value = await api.post<{ version: string }>(`/servers/${id}/test`);
      setMessage(`Connected to PostgreSQL ${value.version}`);
      void reload();
    } catch (reason) {
      setMessage(
        reason instanceof APIError
          ? `${reason.message}. ${reason.detail}`
          : "Connection failed",
      );
    }
  }
  if (loading) return <Loading />;
  if (error) return <ErrorState error={error} retry={reload} />;
  return (
    <>
      <PageHeader
        title="Servers"
        description="PostgreSQL instances monitored by this PGSentinel deployment."
        actions={
          <button
            className="button primary"
            onClick={() => {
              closeForm();
              setOpen(true);
            }}
          >
            <Plus /> Add server
          </button>
        }
      />
      {message && <Notice>{message}</Notice>}
      {open && (
        <form className="server-form" onSubmit={submit}>
          <div className="form-heading">
            <div>
              <h2>{editingID ? "Edit server" : "Add PostgreSQL server"}</h2>
              <p>
                Connection credentials are encrypted at rest and never returned
                to the browser.
              </p>
            </div>
            <button
              type="button"
              className="icon-button"
              aria-label="Close server form"
              onClick={closeForm}
            >
              <X />
            </button>
          </div>
          <fieldset>
            <legend>Connection</legend>
            <div className="form-grid">
              <label>
                Name
                <input
                  required
                  value={form.name}
                  onChange={(event) =>
                    setForm({ ...form, name: event.target.value })
                  }
                  placeholder="Primary database"
                />
              </label>
              <label>
                Host
                <input
                  className="mono"
                  required
                  value={form.host}
                  onChange={(event) =>
                    setForm({ ...form, host: event.target.value })
                  }
                  placeholder="db.internal"
                />
              </label>
              <label>
                Port
                <input
                  className="mono"
                  required
                  type="number"
                  min="1"
                  max="65535"
                  value={form.port}
                  onChange={(event) =>
                    setForm({ ...form, port: Number(event.target.value) })
                  }
                />
              </label>
              <label>
                User
                <input
                  className="mono"
                  required
                  value={form.user}
                  onChange={(event) =>
                    setForm({ ...form, user: event.target.value })
                  }
                />
              </label>
              <label>
                Password{" "}
                <small>
                  {editingID ? "Leave empty to keep current" : "Required"}
                </small>
                <input
                  required={!editingID}
                  type="password"
                  autoComplete="new-password"
                  value={form.password}
                  onChange={(event) =>
                    setForm({ ...form, password: event.target.value })
                  }
                />
              </label>
              <label>
                SSL mode
                <select
                  value={form.sslMode}
                  onChange={(event) =>
                    setForm({ ...form, sslMode: event.target.value })
                  }
                >
                  {[
                    "prefer",
                    "require",
                    "verify-ca",
                    "verify-full",
                    "disable",
                  ].map((value) => (
                    <option key={value}>{value}</option>
                  ))}
                </select>
              </label>
            </div>
          </fieldset>
          <fieldset>
            <legend>Organization</legend>
            <label>
              Tags <small>Comma separated</small>
              <input
                value={form.tags.join(", ")}
                onChange={(event) =>
                  setForm({
                    ...form,
                    tags: event.target.value
                      .split(",")
                      .map((tag) => tag.trim())
                      .filter(Boolean),
                  })
                }
                placeholder="production, eu-west"
              />
            </label>
          </fieldset>
          <div className="form-actions">
            <button
              type="button"
              className="button secondary"
              onClick={closeForm}
            >
              Cancel
            </button>
            <button type="submit" className="button primary">
              {editingID ? "Update server" : "Save server"}
            </button>
          </div>
        </form>
      )}
      {data?.length === 0 ? (
        <Empty
          title="No servers yet"
          detail="Add your first PostgreSQL server to begin collection and health analysis."
          action={
            <button className="button primary" onClick={() => setOpen(true)}>
              <Plus /> Add server
            </button>
          }
        />
      ) : (
        <div className="data-table">
          <table>
            <thead>
              <tr>
                <th>Name</th>
                <th>Host</th>
                <th className="numeric">Port</th>
                <th>Status</th>
                <th>Tags</th>
                <th>SSL mode</th>
                <th>Last connection</th>
                <th>
                  <span className="sr-only">Actions</span>
                </th>
              </tr>
            </thead>
            <tbody>
              {data?.map((server) => (
                <tr key={server.id}>
                  <td>
                    <strong>{server.name}</strong>
                    <small>{server.version ?? "Version not reported"}</small>
                  </td>
                  <td>
                    <code>{server.host}</code>
                  </td>
                  <td className="numeric">{server.port}</td>
                  <td>
                    <StatusIndicator status={server.status} />
                  </td>
                  <td>
                    <div className="tag-list">
                      {server.tags.length ? (
                        server.tags.map((tag) => <span key={tag}>{tag}</span>)
                      ) : (
                        <span>—</span>
                      )}
                    </div>
                  </td>
                  <td>
                    <code>{server.sslMode}</code>
                  </td>
                  <td>
                    {formatDate(server.lastConnectedAt)}
                    {server.lastError && (
                      <small className="danger-text" title={server.lastError}>
                        Connection error
                      </small>
                    )}
                  </td>
                  <td>
                    <div className="row-actions">
                      <button
                        className="icon-button"
                        aria-label={`Edit ${server.name}`}
                        title="Edit server"
                        onClick={() => edit(server)}
                      >
                        <Pencil />
                      </button>
                      <button
                        className="icon-button"
                        aria-label={`Test ${server.name} connection`}
                        title="Test connection"
                        onClick={() => void test(server.id)}
                      >
                        <Wifi />
                      </button>
                      <button
                        className="icon-button danger"
                        aria-label={`Delete ${server.name}`}
                        title="Delete server"
                        onClick={async () => {
                          if (confirm(`Delete ${server.name}?`)) {
                            await api.delete(`/servers/${server.id}`);
                            void reload();
                          }
                        }}
                      >
                        <Trash2 />
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </>
  );
}
function formatDate(value?: string) {
  if (!value) return "Never";
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? "Unknown" : date.toLocaleString();
}
