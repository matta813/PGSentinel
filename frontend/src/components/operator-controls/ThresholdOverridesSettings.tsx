import { SlidersHorizontal } from "lucide-react";
import { useState, type FormEvent } from "react";
import { api } from "../../api/client";
import type { Server } from "../../types";
import type { ThresholdResponse } from "../../types/operator-controls";
import type { OperatorControlAction } from "../OperatorControlsSettings";
import { SectionHeader } from "../UI";
import { ControlList, DeleteControl } from "./ControlList";

export function ThresholdOverridesSettings({ response, servers, reload, act }: { response?: ThresholdResponse; servers: Server[]; reload: () => Promise<void>; act: OperatorControlAction }) {
  const firstRule = Object.keys(response?.specs ?? {})[0] ?? "standby-replay-lag";
  const [value, setValue] = useState({ ruleId: "", scopeType: "global", scopeValue: "", value: "", reason: "" });
  const selectedRule = value.ruleId || firstRule;
  const spec = response?.specs[selectedRule];
  function save(event: FormEvent) {
    event.preventDefault();
    act(() => api.post("/threshold-overrides", { ...value, ruleId: selectedRule, scopeValue: value.scopeType === "global" ? "" : value.scopeValue, value: Number(value.value) }), "Threshold override saved.", reload);
  }
  return <section id="thresholds">
    <SectionHeader title="Scoped thresholds" description="Override selected analyzer thresholds within enforced safe ranges. Server overrides take precedence over tags, then global defaults." />
    <form className="settings-form" onSubmit={save}>
      <div className="form-grid two">
        <label>Rule<select value={selectedRule} onChange={(event) => setValue({ ...value, ruleId: event.target.value, value: "" })}>{Object.entries(response?.specs ?? {}).map(([id, item]) => <option key={id} value={id}>{item.label}</option>)}</select></label>
        <label>Value <small>{spec && `${spec.min}–${spec.max} ${spec.unit}; default ${spec.default}`}</small><input type="number" step="any" required min={spec?.min} max={spec?.max} value={value.value} onChange={(event) => setValue({ ...value, value: event.target.value })} /></label>
      </div>
      <div className="form-grid two">
        <label>Scope<select value={value.scopeType} onChange={(event) => setValue({ ...value, scopeType: event.target.value, scopeValue: "" })}><option value="global">Global default</option><option value="server">Server</option><option value="tag">Server tag</option></select></label>
        <label>Scope value{value.scopeType === "server" ? <select required value={value.scopeValue} onChange={(event) => setValue({ ...value, scopeValue: event.target.value })}><option value="">Select server</option>{servers.map((server) => <option key={server.id} value={server.id}>{server.name}</option>)}</select> : <input disabled={value.scopeType === "global"} required={value.scopeType === "tag"} value={value.scopeValue} onChange={(event) => setValue({ ...value, scopeValue: event.target.value })} />}</label>
      </div>
      <label>Reason<input required maxLength={500} value={value.reason} onChange={(event) => setValue({ ...value, reason: event.target.value })} /></label>
      <div className="form-actions"><button className="button primary"><SlidersHorizontal /> Save override</button></div>
    </form>
    <ControlList empty="No threshold overrides" items={response?.items ?? []} render={(item) => <><div><strong>{response?.specs[item.ruleId]?.label ?? item.ruleId}: {item.value}</strong><span>{item.scopeType}{item.scopeValue && ` · ${item.scopeValue}`} · {item.reason}</span></div><DeleteControl action={() => act(() => api.delete(`/threshold-overrides/${item.id}`), "Threshold override removed.", reload)} /></>} />
  </section>;
}
