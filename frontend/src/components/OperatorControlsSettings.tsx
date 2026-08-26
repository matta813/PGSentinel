import { FormEvent, useState, type ReactNode } from "react";
import {
  CalendarClock,
  ShieldOff,
  SlidersHorizontal,
  Trash2,
} from "lucide-react";
import { api, APIError } from "../api/client";
import { useApi } from "../hooks/useApi";
import type { Server } from "../types";
import type {
  FindingSuppression,
  MaintenanceWindow,
  ThresholdResponse,
} from "../types/operator-controls";
import { Empty, ErrorState, Loading } from "./Status";
import { Notice, SectionHeader } from "./UI";

const local = (date: Date) =>
  new Date(date.getTime() - date.getTimezoneOffset() * 60000)
    .toISOString()
    .slice(0, 16);
const now = new Date();
export function OperatorControlsSettings() {
  const windows = useApi(
    () => api.get<MaintenanceWindow[]>("/maintenance-windows"),
    [],
  );
  const suppressions = useApi(
    () => api.get<FindingSuppression[]>("/suppressions"),
    [],
  );
  const thresholds = useApi(
    () => api.get<ThresholdResponse>("/threshold-overrides"),
    [],
  );
  const servers = useApi(() => api.get<Server[]>("/servers"), []);
  const [message, setMessage] = useState("");
  const [maintenance, setMaintenance] = useState({
    description: "",
    serverId: "",
    serverTag: "",
    category: "",
    ruleId: "",
    startsAt: local(now),
    endsAt: local(new Date(now.getTime() + 60 * 60 * 1000)),
  });
  const [suppression, setSuppression] = useState({
    ruleId: "",
    serverId: "",
    serverTag: "",
    reason: "",
    expiresAt: local(new Date(now.getTime() + 60 * 60 * 1000)),
  });
  const firstRule =
    Object.keys(thresholds.data?.specs ?? {})[0] ?? "standby-replay-lag";
  const [threshold, setThreshold] = useState({
    ruleId: "",
    scopeType: "global",
    scopeValue: "",
    value: "",
    reason: "",
  });
  const failed =
    windows.error || suppressions.error || thresholds.error || servers.error;
  if (
    windows.loading ||
    suppressions.loading ||
    thresholds.loading ||
    servers.loading
  )
    return <Loading />;
  if (failed)
    return (
      <ErrorState
        error={failed}
        retry={() => {
          void windows.reload();
          void suppressions.reload();
          void thresholds.reload();
          void servers.reload();
        }}
      />
    );
  const selectedRule = threshold.ruleId || firstRule;
  const spec = thresholds.data?.specs[selectedRule];
  async function act(
    task: () => Promise<unknown>,
    success: string,
    reload: () => unknown,
  ) {
    setMessage("Saving operator control…");
    try {
      await task();
      setMessage(success);
      reload();
    } catch (reason) {
      setMessage(
        reason instanceof APIError
          ? `${reason.message}: ${reason.detail}`
          : "Unable to save operator control",
      );
    }
  }
  function saveMaintenance(event: FormEvent) {
    event.preventDefault();
    void act(
      () =>
        api.post("/maintenance-windows", {
          ...maintenance,
          startsAt: new Date(maintenance.startsAt).toISOString(),
          endsAt: new Date(maintenance.endsAt).toISOString(),
        }),
      "Maintenance window saved.",
      windows.reload,
    );
  }
  function saveSuppression(event: FormEvent) {
    event.preventDefault();
    void act(
      () =>
        api.post("/suppressions", {
          ...suppression,
          expiresAt: new Date(suppression.expiresAt).toISOString(),
        }),
      "Rule suppression saved.",
      suppressions.reload,
    );
  }
  function saveThreshold(event: FormEvent) {
    event.preventDefault();
    void act(
      () =>
        api.post("/threshold-overrides", {
          ...threshold,
          ruleId: selectedRule,
          scopeValue:
            threshold.scopeType === "global" ? "" : threshold.scopeValue,
          value: Number(threshold.value),
        }),
      "Threshold override saved.",
      thresholds.reload,
    );
  }
  return (
    <div className="operator-controls">
      <section id="maintenance">
        <SectionHeader
          title="Maintenance windows"
          description="Temporarily silence matching notifications without deleting findings or evidence."
        />
        <form className="settings-form" onSubmit={saveMaintenance}>
          <div className="form-grid two">
            <label>
              Reason
              <input
                required
                maxLength={500}
                value={maintenance.description}
                onChange={(event) =>
                  setMaintenance({
                    ...maintenance,
                    description: event.target.value,
                  })
                }
              />
            </label>
            <label>
              Server <small>Optional</small>
              <select
                value={maintenance.serverId}
                onChange={(event) =>
                  setMaintenance({
                    ...maintenance,
                    serverId: event.target.value,
                  })
                }
              >
                <option value="">Any server</option>
                {servers.data?.map((server) => (
                  <option key={server.id} value={server.id}>
                    {server.name}
                  </option>
                ))}
              </select>
            </label>
          </div>
          <div className="form-grid two">
            <label>
              Starts
              <input
                type="datetime-local"
                required
                value={maintenance.startsAt}
                onChange={(event) =>
                  setMaintenance({
                    ...maintenance,
                    startsAt: event.target.value,
                  })
                }
              />
            </label>
            <label>
              Ends
              <input
                type="datetime-local"
                required
                value={maintenance.endsAt}
                onChange={(event) =>
                  setMaintenance({ ...maintenance, endsAt: event.target.value })
                }
              />
            </label>
          </div>
          <div className="form-grid three">
            <label>
              Server tag <small>Optional</small>
              <input
                value={maintenance.serverTag}
                onChange={(event) =>
                  setMaintenance({
                    ...maintenance,
                    serverTag: event.target.value,
                  })
                }
              />
            </label>
            <label>
              Category <small>Optional</small>
              <input
                value={maintenance.category}
                onChange={(event) =>
                  setMaintenance({
                    ...maintenance,
                    category: event.target.value,
                  })
                }
              />
            </label>
            <label>
              Rule ID <small>Optional</small>
              <input
                className="mono"
                value={maintenance.ruleId}
                onChange={(event) =>
                  setMaintenance({ ...maintenance, ruleId: event.target.value })
                }
              />
            </label>
          </div>
          <div className="form-actions">
            <button className="button primary">
              <CalendarClock /> Save window
            </button>
          </div>
        </form>
        <ControlList
          empty="No maintenance windows"
          items={windows.data ?? []}
          render={(item) => (
            <>
              <div>
                <strong>{item.description}</strong>
                <span>
                  {item.state} · {scope(item)} · until{" "}
                  {new Date(item.endsAt).toLocaleString()}
                </span>
              </div>
              <Delete
                action={() =>
                  void act(
                    () => api.delete(`/maintenance-windows/${item.id}`),
                    "Maintenance window removed.",
                    windows.reload,
                  )
                }
              />
            </>
          )}
        />
      </section>
      <section id="suppressions">
        <SectionHeader
          title="Suppressions"
          description="Temporarily suppress a rule for a server or tag. Individual findings can be suppressed from the Problems page."
        />
        <form className="settings-form" onSubmit={saveSuppression}>
          <div className="form-grid two">
            <label>
              Rule ID
              <input
                className="mono"
                required
                value={suppression.ruleId}
                onChange={(event) =>
                  setSuppression({ ...suppression, ruleId: event.target.value })
                }
              />
            </label>
            <label>
              Expires
              <input
                type="datetime-local"
                required
                value={suppression.expiresAt}
                onChange={(event) =>
                  setSuppression({
                    ...suppression,
                    expiresAt: event.target.value,
                  })
                }
              />
            </label>
          </div>
          <div className="form-grid two">
            <label>
              Server
              <select
                value={suppression.serverId}
                onChange={(event) =>
                  setSuppression({
                    ...suppression,
                    serverId: event.target.value,
                  })
                }
              >
                <option value="">Use tag scope</option>
                {servers.data?.map((server) => (
                  <option key={server.id} value={server.id}>
                    {server.name}
                  </option>
                ))}
              </select>
            </label>
            <label>
              Server tag
              <input
                value={suppression.serverTag}
                onChange={(event) =>
                  setSuppression({
                    ...suppression,
                    serverTag: event.target.value,
                  })
                }
              />
            </label>
          </div>
          <label>
            Reason
            <input
              required
              maxLength={500}
              value={suppression.reason}
              onChange={(event) =>
                setSuppression({ ...suppression, reason: event.target.value })
              }
            />
          </label>
          <div className="form-actions">
            <button className="button primary">
              <ShieldOff /> Save suppression
            </button>
          </div>
        </form>
        <ControlList
          empty="No suppressions"
          items={suppressions.data ?? []}
          render={(item) => (
            <>
              <div>
                <strong>{item.reason}</strong>
                <span>
                  {item.state} ·{" "}
                  {item.findingId
                    ? `finding ${item.findingId}`
                    : `rule ${item.ruleId}`}{" "}
                  · until {new Date(item.expiresAt).toLocaleString()}
                </span>
              </div>
              <Delete
                action={() =>
                  void act(
                    () => api.delete(`/suppressions/${item.id}`),
                    "Suppression removed.",
                    suppressions.reload,
                  )
                }
              />
            </>
          )}
        />
      </section>
      <section id="thresholds">
        <SectionHeader
          title="Scoped thresholds"
          description="Override selected analyzer thresholds within enforced safe ranges. Server overrides take precedence over tags, then global defaults."
        />
        <form className="settings-form" onSubmit={saveThreshold}>
          <div className="form-grid two">
            <label>
              Rule
              <select
                value={selectedRule}
                onChange={(event) =>
                  setThreshold({
                    ...threshold,
                    ruleId: event.target.value,
                    value: "",
                  })
                }
              >
                {Object.entries(thresholds.data?.specs ?? {}).map(
                  ([id, item]) => (
                    <option key={id} value={id}>
                      {item.label}
                    </option>
                  ),
                )}
              </select>
            </label>
            <label>
              Value{" "}
              <small>
                {spec &&
                  `${spec.min}–${spec.max} ${spec.unit}; default ${spec.default}`}
              </small>
              <input
                type="number"
                step="any"
                required
                min={spec?.min}
                max={spec?.max}
                value={threshold.value}
                onChange={(event) =>
                  setThreshold({ ...threshold, value: event.target.value })
                }
              />
            </label>
          </div>
          <div className="form-grid two">
            <label>
              Scope
              <select
                value={threshold.scopeType}
                onChange={(event) =>
                  setThreshold({
                    ...threshold,
                    scopeType: event.target.value,
                    scopeValue: "",
                  })
                }
              >
                <option value="global">Global default</option>
                <option value="server">Server</option>
                <option value="tag">Server tag</option>
              </select>
            </label>
            <label>
              Scope value
              {threshold.scopeType === "server" ? (
                <select
                  required
                  value={threshold.scopeValue}
                  onChange={(event) =>
                    setThreshold({
                      ...threshold,
                      scopeValue: event.target.value,
                    })
                  }
                >
                  <option value="">Select server</option>
                  {servers.data?.map((server) => (
                    <option key={server.id} value={server.id}>
                      {server.name}
                    </option>
                  ))}
                </select>
              ) : (
                <input
                  disabled={threshold.scopeType === "global"}
                  required={threshold.scopeType === "tag"}
                  value={threshold.scopeValue}
                  onChange={(event) =>
                    setThreshold({
                      ...threshold,
                      scopeValue: event.target.value,
                    })
                  }
                />
              )}
            </label>
          </div>
          <label>
            Reason
            <input
              required
              maxLength={500}
              value={threshold.reason}
              onChange={(event) =>
                setThreshold({ ...threshold, reason: event.target.value })
              }
            />
          </label>
          <div className="form-actions">
            <button className="button primary">
              <SlidersHorizontal /> Save override
            </button>
          </div>
        </form>
        <ControlList
          empty="No threshold overrides"
          items={thresholds.data?.items ?? []}
          render={(item) => (
            <>
              <div>
                <strong>
                  {thresholds.data?.specs[item.ruleId]?.label ?? item.ruleId}:{" "}
                  {item.value}
                </strong>
                <span>
                  {item.scopeType}
                  {item.scopeValue && ` · ${item.scopeValue}`} · {item.reason}
                </span>
              </div>
              <Delete
                action={() =>
                  void act(
                    () => api.delete(`/threshold-overrides/${item.id}`),
                    "Threshold override removed.",
                    thresholds.reload,
                  )
                }
              />
            </>
          )}
        />
      </section>
      {message && <Notice>{message}</Notice>}
    </div>
  );
}
function ControlList<T>({
  items,
  render,
  empty,
}: {
  items: T[];
  render: (item: T) => ReactNode;
  empty: string;
}) {
  if (!items.length)
    return (
      <Empty
        title={empty}
        detail="Operator controls appear here after they are created."
      />
    );
  return (
    <div className="control-list">
      {items.map((item, index) => (
        <article key={index}>{render(item)}</article>
      ))}
    </div>
  );
}
function Delete({ action }: { action: () => void }) {
  return (
    <button
      className="icon-button danger"
      aria-label="Delete operator control"
      onClick={action}
    >
      <Trash2 />
    </button>
  );
}
function scope(item: MaintenanceWindow) {
  return [
    item.serverId && "server",
    item.serverTag && `tag ${item.serverTag}`,
    item.category && item.category,
    item.ruleId && item.ruleId,
  ]
    .filter(Boolean)
    .join(" · ");
}
