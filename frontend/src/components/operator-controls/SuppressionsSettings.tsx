import { ShieldOff } from "lucide-react";
import { useState, type FormEvent } from "react";
import { api } from "../../api/client";
import type { Server } from "../../types";
import type { FindingSuppression } from "../../types/operator-controls";
import type { OperatorControlAction } from "../OperatorControlsSettings";
import { SectionHeader } from "../UI";
import { ControlList, DeleteControl } from "./ControlList";
import { localDateTime, operatorControlsNow } from "./dateTime";

export function SuppressionsSettings({ items, servers, reload, act }: { items: FindingSuppression[]; servers: Server[]; reload: () => Promise<void>; act: OperatorControlAction }) {
  const [value, setValue] = useState({ ruleId: "", serverId: "", serverTag: "", reason: "", expiresAt: localDateTime(new Date(operatorControlsNow.getTime() + 3600000)) });
  function save(event: FormEvent) {
    event.preventDefault();
    act(() => api.post("/suppressions", { ...value, expiresAt: new Date(value.expiresAt).toISOString() }), "Rule suppression saved.", reload);
  }
  return <section id="suppressions">
    <SectionHeader title="Suppressions" description="Temporarily suppress a rule for a server or tag. Individual findings can be suppressed from the Problems page." />
    <form className="settings-form" onSubmit={save}>
      <div className="form-grid two">
        <label>Rule ID<input className="mono" required value={value.ruleId} onChange={(event) => setValue({ ...value, ruleId: event.target.value })} /></label>
        <label>Expires<input type="datetime-local" required value={value.expiresAt} onChange={(event) => setValue({ ...value, expiresAt: event.target.value })} /></label>
      </div>
      <div className="form-grid two">
        <label>Server<select value={value.serverId} onChange={(event) => setValue({ ...value, serverId: event.target.value })}><option value="">Use tag scope</option>{servers.map((server) => <option key={server.id} value={server.id}>{server.name}</option>)}</select></label>
        <label>Server tag<input value={value.serverTag} onChange={(event) => setValue({ ...value, serverTag: event.target.value })} /></label>
      </div>
      <label>Reason<input required maxLength={500} value={value.reason} onChange={(event) => setValue({ ...value, reason: event.target.value })} /></label>
      <div className="form-actions"><button className="button primary"><ShieldOff /> Save suppression</button></div>
    </form>
    <ControlList empty="No suppressions" items={items} render={(item) => <><div><strong>{item.reason}</strong><span>{item.state} · {item.findingId ? `finding ${item.findingId}` : `rule ${item.ruleId}`} · until {new Date(item.expiresAt).toLocaleString()}</span></div><DeleteControl action={() => act(() => api.delete(`/suppressions/${item.id}`), "Suppression removed.", reload)} /></>} />
  </section>;
}
