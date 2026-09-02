import { X } from "lucide-react";
import type { Dispatch, FormEvent, SetStateAction } from "react";
import type { ServerFormValue } from "./serverForm";

export function ServerEditor({ value, setValue, editingID, close, submit }: {
  value: ServerFormValue; setValue: Dispatch<SetStateAction<ServerFormValue>>;
  editingID: string; close: () => void; submit: (event: FormEvent) => Promise<void>;
}) {
  return <form className="server-form" onSubmit={(event) => void submit(event)}>
    <div className="form-heading"><div><h2>{editingID ? "Edit server" : "Add PostgreSQL server"}</h2><p>Connection credentials are encrypted at rest and never returned to the browser.</p></div><button type="button" className="icon-button" aria-label="Close server form" onClick={close}><X /></button></div>
    <fieldset><legend>Connection</legend><div className="form-grid">
      <label>Name<input required value={value.name} onChange={(event) => setValue({ ...value, name: event.target.value })} placeholder="Primary database" /></label>
      <label>Host<input className="mono" required value={value.host} onChange={(event) => setValue({ ...value, host: event.target.value })} placeholder="db.internal" /></label>
      <label>Port<input className="mono" required type="number" min="1" max="65535" value={value.port} onChange={(event) => setValue({ ...value, port: Number(event.target.value) })} /></label>
      <label>User<input className="mono" required value={value.user} onChange={(event) => setValue({ ...value, user: event.target.value })} /></label>
      <label>Password <small>{editingID ? "Leave empty to keep current" : "Required"}</small><input required={!editingID} type="password" autoComplete="new-password" value={value.password} onChange={(event) => setValue({ ...value, password: event.target.value })} /></label>
      <label>SSL mode<select value={value.sslMode} onChange={(event) => setValue({ ...value, sslMode: event.target.value })}>{["prefer", "require", "verify-ca", "verify-full", "disable"].map((mode) => <option key={mode}>{mode}</option>)}</select></label>
    </div></fieldset>
    <fieldset><legend>Organization</legend><label>Tags <small>Comma separated</small><input value={value.tags.join(", ")} onChange={(event) => setValue({ ...value, tags: event.target.value.split(",").map((tag) => tag.trim()).filter(Boolean) })} placeholder="production, eu-west" /></label></fieldset>
    <div className="form-actions"><button type="button" className="button secondary" onClick={close}>Cancel</button><button type="submit" className="button primary">{editingID ? "Update server" : "Save server"}</button></div>
  </form>;
}
