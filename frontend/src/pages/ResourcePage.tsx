import { useState, type ReactNode } from "react";
import { Database, Info } from "lucide-react";
import { useParams } from "react-router-dom";
import { api, APIError } from "../api/client";
import { Empty, ErrorState, Loading } from "../components/Status";
import { PageHeader } from "../components/UI";
import { useApi } from "../hooks/useApi";
import type {
  IndexStat,
  LockInfo,
  CollectionResourceStatus,
  QueryStat,
  Server,
  TableStat,
} from "../types";
import type { ReplicationStats, WALStats } from "../types/replication";

const titles: Record<string, [string, string, string]> = {
  queries: [
    "Query performance",
    "Queries",
    "Total impact, latency, disk reads, temporary I/O, and WAL activity.",
  ],
  tables: [
    "Table health",
    "Tables",
    "Storage footprint, access patterns, and tuple health across monitored tables.",
  ],
  indexes: [
    "Index analysis",
    "Indexes",
    "Usage evidence and cautious identification of potentially unused indexes.",
  ],
  vacuum: [
    "Vacuum health",
    "Vacuum",
    "Dead tuples and progress toward estimated autovacuum thresholds.",
  ],
  locks: [
    "Live blocking",
    "Locks",
    "Current blocked sessions, their blockers, and blocking duration.",
  ],
  replication: [
    "Replication intelligence",
    "Replication",
    "Role, timeline, LSN byte gaps, receiver state, and slot retention evidence.",
  ],
  wal: [
    "WAL and archive intelligence",
    "WAL & archive",
    "WAL generation, checkpoint or restartpoint behavior, and archive outcomes.",
  ],
};
export function ResourcePage() {
  const { resource = "queries" } = useParams();
  const [server, setServer] = useState("");
  const servers = useApi(() => api.get<Server[]>("/servers"), []);
  const selected = server || servers.data?.[0]?.id || "";
  const result = useApi(
    () =>
      selected
        ? api.get<unknown>(`/servers/${selected}/${resource}`)
        : Promise.resolve([]),
    [selected, resource],
  );
  const freshness = useApi(
    () =>
      selected
        ? api.get<CollectionResourceStatus[]>(`/servers/${selected}/freshness`).catch((error: unknown) => {
            if (error instanceof APIError && error.status === 404) return [];
            throw error;
          })
        : Promise.resolve([]),
    [selected, resource],
  );
  if (servers.loading || result.loading || freshness.loading)
    return <Loading />;
  if (servers.error || result.error || freshness.error)
    return (
      <ErrorState
        error={servers.error ?? result.error ?? freshness.error!}
        retry={result.reload}
      />
    );
  const title = titles[resource] ?? [
    "Monitoring data",
    "Evidence",
    "Collected PostgreSQL evidence.",
  ];
  const currentServer = servers.data?.find((item) => item.id === selected);
  const quality = Array.isArray(freshness.data)
    ? freshness.data.find((item) => item.resource === resource)
    : undefined;
  return (
    <>
      <PageHeader
        title={title[0]}
        description={title[2]}
        actions={
          <label className="server-select">
            <Database />
            <span className="sr-only">Server</span>
            <select
              aria-label="Server"
              value={selected}
              onChange={(event) => setServer(event.target.value)}
            >
              {servers.data?.map((item) => (
                <option value={item.id} key={item.id}>
                  {item.name}
                </option>
              ))}
            </select>
          </label>
        }
      />
      {selected && (
        <>
          <FreshnessNotice quality={quality} />
          <div className="table-context">
            <div>
              <strong>{title[1]}</strong>
              <span>{currentServer?.name}</span>
            </div>
            <span
              className={`freshness-state ${quality?.state ?? "unavailable"}`}
            >
              <Info /> {freshnessLabel(quality)}
            </span>
          </div>
        </>
      )}
      {!selected ? (
        <Empty
          title="No server configured"
          detail="Add a PostgreSQL server before viewing collected evidence."
        />
      ) : (
        <ResourceTable resource={resource} data={result.data ?? []} />
      )}
    </>
  );
}
function FreshnessNotice({ quality }: { quality?: CollectionResourceStatus }) {
  if (quality?.state === "fresh") return null;
  const heading =
    quality?.state === "stale"
      ? "This evidence is stale"
      : quality?.state === "partial"
        ? "This evidence is incomplete"
        : "Current evidence is unavailable";
  return (
    <div
      className={`freshness-notice ${quality?.state ?? "unavailable"}`}
      role="status"
    >
      <strong>{heading}</strong>
      <span>
        {quality?.errorSummary || "No successful collection has completed yet."}{" "}
        {quality?.lastSuccessfulCollection &&
          `Last success ${new Date(quality.lastSuccessfulCollection).toLocaleString()}.`}
      </span>
    </div>
  );
}
function freshnessLabel(quality?: CollectionResourceStatus) {
  if (!quality) return "Not collected";
  if (quality.state === "fresh" && quality.ageSeconds !== undefined)
    return `Fresh · ${duration(quality.ageSeconds)} old`;
  return `${quality.state} · ${quality.consecutiveFailures} consecutive failure${quality.consecutiveFailures === 1 ? "" : "s"}`;
}
function duration(seconds: number) {
  if (seconds < 60) return `${seconds}s`;
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m`;
  return `${Math.floor(seconds / 3600)}h`;
}
function ResourceTable({
  resource,
  data,
}: {
  resource: string;
  data: unknown;
}) {
  if (resource === "replication")
    return <ReplicationView value={data as ReplicationStats} />;
  if (resource === "wal") return <WALView value={data as WALStats} />;
  const rows = data as unknown[];
  if (rows.length === 0)
    return (
      <Empty
        title="No data collected yet"
        detail="Evidence will appear after a successful monitoring cycle."
      />
    );
  if (resource === "queries")
    return (
      <Table
        headers={[
          "Query",
          "Database",
          "Calls",
          "Avg latency",
          "Total runtime",
          "Impact",
        ]}
        numeric={[2, 3, 4, 5]}
        rows={(rows as QueryStat[]).map((query) => [
          <code className="query-text" title={query.Query}>
            {query.Query}
          </code>,
          query.Database,
          fmt(query.Calls),
          `${fmt(query.MeanExecMS)} ms`,
          `${fmt(query.TotalExecMS)} ms`,
          query.ImpactScore.toFixed(1),
        ])}
      />
    );
  if (resource === "indexes")
    return (
      <Table
        headers={["Database", "Index", "Table", "Size", "Scans", "Assessment"]}
        numeric={[3, 4]}
        rows={(rows as IndexStat[]).map((index) => [
          index.Database,
          <code>{index.Index}</code>,
          <code>
            {index.Schema}.{index.Table}
          </code>,
          bytes(index.SizeBytes),
          fmt(index.Scans),
          <span
            className={
              index.Scans === 0 && !index.Primary && !index.Unique
                ? "assessment warning"
                : "assessment"
            }
          >
            {index.Scans === 0 && !index.Primary && !index.Unique
              ? "Potentially unused"
              : "Observed usage"}
          </span>,
        ])}
      />
    );
  if (resource === "locks")
    return (
      <Table
        headers={[
          "Blocked PID",
          "Blocking PID",
          "Duration",
          "Database",
          "Application",
          "Query",
        ]}
        numeric={[0, 1, 2]}
        rows={(rows as LockInfo[]).map((lock) => [
          <code>{lock.BlockedPID}</code>,
          <code>{lock.BlockingPID}</code>,
          `${fmt(lock.DurationSeconds)} sec`,
          lock.Database,
          lock.Application,
          <code className="query-text" title={lock.Query}>
            {lock.Query}
          </code>,
        ])}
      />
    );
  const table = rows as TableStat[];
  return (
    <Table
      headers={[
        "Database",
        "Table",
        "Rows",
        "Total size",
        "Dead tuples",
        resource === "vacuum" ? "Autovacuum trigger" : "Index scans",
        resource === "vacuum" ? "Progress" : "Last autovacuum",
      ]}
      numeric={[2, 3, 4, 5, 6]}
      rows={table.map((item) => [
        item.Database,
        <code>
          {item.Schema}.{item.Table}
        </code>,
        fmt(item.EstimatedRows),
        bytes(item.TotalSize),
        `${fmt(item.DeadTuples)} (${deadRatio(item)}%)`,
        resource === "vacuum"
          ? fmt(item.VacuumThreshold)
          : fmt(item.IndexScans),
        resource === "vacuum"
          ? `${Math.round(item.VacuumProgress)}%`
          : item.LastAutovacuum
            ? new Date(item.LastAutovacuum).toLocaleString()
            : "Never",
      ])}
    />
  );
}
function ReplicationView({ value }: { value: ReplicationStats }) {
  const standbys = value.Standbys ?? [];
  const slots = value.Slots ?? [];
  return (
    <div className="form-stack">
      <div className="evidence-grid">
        <article>
          <span>Server role</span>
          <strong>{value.InRecovery ? "Standby" : "Primary"}</strong>
        </article>
        <article>
          <span>Timeline</span>
          <strong>{value.TimelineID || "—"}</strong>
        </article>
        <article>
          <span>Recovery replay</span>
          <strong>{value.RecoveryPaused ? "Paused" : "Running"}</strong>
        </article>
        <article>
          <span>Configured replay delay</span>
          <strong>{fmt(value.ReplayDelaySeconds || 0)} sec</strong>
        </article>
      </div>
      {standbys.length > 0 && (
        <Table
          headers={[
            "Standby",
            "State",
            "Sync",
            "Replay lag",
            "Pending replay",
            "Gap growth",
            "Reply age",
          ]}
          numeric={[3, 4, 5, 6]}
          rows={standbys.map((item) => [
            item.Application || item.ClientAddress,
            item.State,
            item.SyncState,
            `${fmt(item.ReplayLagSeconds)} sec`,
            bytes(item.PendingReplayBytes),
            `${bytesRate(item.PendingReplayGrowthBytesPerSecond)}`,
            `${fmt(item.ReplyAgeSeconds)} sec`,
          ])}
        />
      )}{" "}
      {slots.length > 0 && (
        <Table
          headers={[
            "Slot",
            "Type",
            "State",
            "WAL status",
            "Retained",
            "Growth",
            "Inactive",
          ]}
          numeric={[4, 5, 6]}
          rows={slots.map((item) => [
            item.Name,
            item.Type,
            item.Active ? "Active" : "Inactive",
            item.WALStatus || "reserved",
            bytes(item.RetainedBytes),
            bytesRate(item.RetainedGrowthBytesPerSecond),
            `${fmt(item.InactiveSeconds)} sec`,
          ])}
        />
      )}{" "}
      {standbys.length === 0 && slots.length === 0 && !value.InRecovery && (
        <Empty
          title="No replication consumers observed"
          detail="A primary without replicas or slots is not considered unhealthy."
        />
      )}
    </div>
  );
}
function WALView({ value }: { value: WALStats }) {
  return (
    <div className="form-stack">
      <div className="evidence-grid">
        <article>
          <span>WAL generation</span>
          <strong>{bytesRate(value.GenerationBytesPerSecond)}</strong>
        </article>
        <article>
          <span>WAL generated</span>
          <strong>{bytes(value.WALBytes)}</strong>
        </article>
        <article>
          <span>Archive mode</span>
          <strong>{value.ArchiveMode || "unknown"}</strong>
        </article>
        <article>
          <span>Archive destination</span>
          <strong>
            {value.ArchiveConfigured ? "Configured" : "Not configured"}
          </strong>
        </article>
      </div>
      <Table
        headers={[
          "Archived",
          "Failed",
          "Last archived WAL",
          "Last failed WAL",
          "Timed checkpoints",
          "Requested checkpoints",
          "Restartpoints done",
        ]}
        numeric={[0, 1, 4, 5, 6]}
        rows={[
          [
            fmt(value.ArchivedCount),
            fmt(value.FailedArchiveCount),
            value.LastArchivedWAL || "—",
            value.LastFailedWAL || "—",
            fmt(value.TimedCheckpoints),
            fmt(value.RequestedCheckpoints),
            fmt(value.RestartpointsDone),
          ],
        ]}
      />
    </div>
  );
}
function Table({
  headers,
  rows,
  numeric = [],
}: {
  headers: string[];
  rows: ReactNode[][];
  numeric?: number[];
}) {
  return (
    <div className="data-table">
      <table>
        <thead>
          <tr>
            {headers.map((header, index) => (
              <th
                className={numeric.includes(index) ? "numeric" : ""}
                key={header}
              >
                {header}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.map((row, rowIndex) => (
            <tr key={rowIndex}>
              {row.map((value, columnIndex) => (
                <td
                  className={numeric.includes(columnIndex) ? "numeric" : ""}
                  key={columnIndex}
                >
                  {value}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
function fmt(value: number) {
  return Intl.NumberFormat("en", {
    notation: value > 9999 ? "compact" : "standard",
    maximumFractionDigits: 1,
  }).format(value || 0);
}
function bytes(value: number) {
  if (value >= 1024 ** 3) return `${fmt(value / 1024 ** 3)} GB`;
  if (value >= 1024 ** 2) return `${fmt(value / 1024 ** 2)} MB`;
  return `${fmt(value / 1024)} KB`;
}
function bytesRate(value: number) {
  const prefix = value < 0 ? "-" : "";
  return `${prefix}${bytes(Math.abs(value || 0))}/s`;
}
function deadRatio(table: TableStat) {
  const count = table.LiveTuples + table.DeadTuples;
  return count ? ((table.DeadTuples / count) * 100).toFixed(1) : "0";
}
