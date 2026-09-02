import { Pencil, Trash2, Wifi } from "lucide-react";
import { StatusIndicator } from "../../components/Status";
import type { Server } from "../../types";

export function ServerTable({ servers, edit, test, remove }: { servers: Server[]; edit: (server: Server) => void; test: (id: string) => Promise<void>; remove: (server: Server) => Promise<void> }) {
  return <div className="data-table"><table>
    <thead><tr><th>Name</th><th>Host</th><th className="numeric">Port</th><th>Status</th><th>Tags</th><th>SSL mode</th><th>Last connection</th><th><span className="sr-only">Actions</span></th></tr></thead>
    <tbody>{servers.map((server) => <tr key={server.id}>
      <td><strong>{server.name}</strong><small>{server.version ?? "Version not reported"}</small></td>
      <td><code>{server.host}</code></td><td className="numeric">{server.port}</td><td><StatusIndicator status={server.status} /></td>
      <td><div className="tag-list">{server.tags.length ? server.tags.map((tag) => <span key={tag}>{tag}</span>) : <span>—</span>}</div></td>
      <td><code>{server.sslMode}</code></td>
      <td>{formatDate(server.lastConnectedAt)}{server.lastError && <small className="danger-text" title={server.lastError}>Connection error</small>}</td>
      <td><div className="row-actions">
        <button className="icon-button" aria-label={`Edit ${server.name}`} title="Edit server" onClick={() => edit(server)}><Pencil /></button>
        <button className="icon-button" aria-label={`Test ${server.name} connection`} title="Test connection" onClick={() => void test(server.id)}><Wifi /></button>
        <button className="icon-button danger" aria-label={`Delete ${server.name}`} title="Delete server" onClick={() => void remove(server)}><Trash2 /></button>
      </div></td>
    </tr>)}</tbody>
  </table></div>;
}

function formatDate(value?: string) {
  if (!value) return "Never";
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? "Unknown" : date.toLocaleString();
}
