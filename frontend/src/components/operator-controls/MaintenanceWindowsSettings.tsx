import { CalendarClock } from "lucide-react";
import { useState, type FormEvent } from "react";
import { api } from "../../api/client";
import type { Server } from "../../types";
import type { MaintenanceWindow } from "../../types/operator-controls";
import type { OperatorControlAction } from "../OperatorControlsSettings";
import { SectionHeader } from "../UI";
import { ControlList, DeleteControl } from "./ControlList";
import { localDateTime, operatorControlsNow } from "./dateTime";

export function MaintenanceWindowsSettings({ items, servers, reload, act }: { items: MaintenanceWindow[]; servers: Server[]; reload: () => Promise<void>; act: OperatorControlAction }) {
  const [value, setValue] = useState({ description: "", serverId: "", serverTag: "", category: "", ruleId: "", startsAt: localDateTime(operatorControlsNow), endsAt: localDateTime(new Date(operatorControlsNow.getTime() + 3600000)) });
  function save(event: FormEvent) {
    event.preventDefault();
    act(() => api.post("/maintenance-windows", { ...value, startsAt: new Date(value.startsAt).toISOString(), endsAt: new Date(value.endsAt).toISOString() }), "Maintenance window saved.", reload);
  }
  return <section id="maintenance">
    <SectionHeader title="Maintenance windows" description="Temporarily silence matching notifications without deleting findings or evidence." />
    <form className="settings-form" onSubmit={save}>
      <div className="form-grid two">
        <label>Reason<input required maxLength={500} value={value.description} onChange={(event) => setValue({ ...value, description: event.target.value })} /></label>
        <label>Server <small>Optional</small><select value={value.serverId} onChange={(event) => setValue({ ...value, serverId: event.target.value })}><option value="">Any server</option>{servers.map((server) => <option key={server.id} value={server.id}>{server.name}</option>)}</select></label>
      </div>
      <div className="form-grid two">
        <label>Starts<input type="datetime-local" required value={value.startsAt} onChange={(event) => setValue({ ...value, startsAt: event.target.value })} /></label>
        <label>Ends<input type="datetime-local" required value={value.endsAt} onChange={(event) => setValue({ ...value, endsAt: event.target.value })} /></label>
      </div>
      <div className="form-grid three">
        <label>Server tag <small>Optional</small><input value={value.serverTag} onChange={(event) => setValue({ ...value, serverTag: event.target.value })} /></label>
        <label>Category <small>Optional</small><input value={value.category} onChange={(event) => setValue({ ...value, category: event.target.value })} /></label>
        <label>Rule ID <small>Optional</small><input className="mono" value={value.ruleId} onChange={(event) => setValue({ ...value, ruleId: event.target.value })} /></label>
      </div>
      <div className="form-actions"><button className="button primary"><CalendarClock /> Save window</button></div>
    </form>
    <ControlList empty="No maintenance windows" items={items} render={(item) => <><div><strong>{item.description}</strong><span>{item.state} · {scope(item)} · until {new Date(item.endsAt).toLocaleString()}</span></div><DeleteControl action={() => act(() => api.delete(`/maintenance-windows/${item.id}`), "Maintenance window removed.", reload)} /></>} />
  </section>;
}

function scope(item: MaintenanceWindow) {
  return [item.serverId && "server", item.serverTag && `tag ${item.serverTag}`, item.category && item.category, item.ruleId && item.ruleId].filter(Boolean).join(" · ");
}
