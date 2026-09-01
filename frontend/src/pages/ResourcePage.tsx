import { useMemo, useState, type ReactNode } from "react";
import { ArrowDown, ArrowUp, Info, Search } from "lucide-react";
import { useParams } from "react-router-dom";
import { api, APIError } from "../api/client";
import { Empty, ErrorState, Loading } from "../components/Status";
import { KPI, KPIGrid, PageHeader } from "../components/UI";
import { useApi } from "../hooks/useApi";
import { useMonitoring } from "../context/MonitoringContext";
import type {
  IndexStat,
  LockInfo,
  CollectionResourceStatus,
  QueryStat,
  TableStat,
  WaitEventSample,
} from "../types";
import type { ReplicationStats, WALStats } from "../types/replication";

const titles: Record<string, [string, string, string]> = {
  queries: [
    "Query Performance",
    "Queries",
    "Total impact, latency, disk reads, temporary I/O, and WAL activity.",
  ],
  tables: [
    "Tables",
    "Tables",
    "Storage footprint, access patterns, and tuple health across monitored tables.",
  ],
  indexes: [
    "Index analysis",
    "Indexes",
    "Usage evidence and cautious identification of potentially unused indexes.",
  ],
  vacuum: [
    "Vacuum",
    "Vacuum",
    "Dead tuples and progress toward estimated autovacuum thresholds.",
  ],
  locks: [
    "Locks",
    "Locks",
    "Current blocked sessions, their blockers, and blocking duration.",
  ],
  "wait-events": [
    "Wait Events",
    "Current waits",
    "Current PostgreSQL sessions waiting on locks, I/O, internal synchronization, clients, and other backend resources.",
  ],
  replication: [
    "Replication intelligence",
    "Replication",
    "Role, timeline, LSN byte gaps, receiver state, and slot retention evidence.",
  ],
  wal: [
    "WAL & Archive",
    "WAL & archive",
    "WAL generation, checkpoint or restartpoint behavior, and archive outcomes.",
  ],
};
export function ResourcePage() {
  const { resource = "queries" } = useParams();
  const monitoring = useMonitoring();
  const selected = monitoring.selectedServerId;
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
        ? api
            .get<CollectionResourceStatus[]>(`/servers/${selected}/freshness`)
            .catch((error: unknown) => {
              if (error instanceof APIError && error.status === 404) return [];
              throw error;
            })
        : Promise.resolve([]),
    [selected, resource],
  );
  if (monitoring.serversLoading || result.loading || freshness.loading)
    return <Loading />;
  if (monitoring.serversError)
    return (
      <ErrorState
        error={monitoring.serversError}
        retry={monitoring.reloadServers}
      />
    );
  if (result.error || freshness.error)
    return (
      <ErrorState
        error={result.error ?? freshness.error!}
        retry={result.reload}
      />
    );
  const title = titles[resource] ?? [
    "Monitoring data",
    "Evidence",
    "Collected PostgreSQL evidence.",
  ];
  const currentServer = monitoring.selectedServer;
  const quality = Array.isArray(freshness.data)
    ? freshness.data.find((item) => item.resource === resource)
    : undefined;
  return (
    <>
      <PageHeader
        title={title[0]}
        description={title[2]}
        meta={
          resource === "queries"
            ? "Latest collected query snapshot"
            : resource === "wait-events"
              ? "Current evidence · Latest snapshot"
              : undefined
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
        <>
          <ResourceSummary
            resource={resource}
            data={result.data ?? []}
            database={monitoring.selectedDatabase}
            quality={quality}
          />
          <ResourceTable
            resource={resource}
            data={result.data ?? []}
            database={monitoring.selectedDatabase}
          />
        </>
      )}
    </>
  );
}
function ResourceSummary({
  resource,
  data,
  database,
  quality,
}: {
  resource: string;
  data: unknown;
  database: string;
  quality?: CollectionResourceStatus;
}) {
  if (!Array.isArray(data)) return null;
  const rows = data.filter(
    (row) => !database || (row as { Database?: string }).Database === database,
  );
  if (database && data.length > 0 && rows.length === 0) return null;
  if (resource === "indexes") {
    const indexes = rows as IndexStat[];
    const candidates = indexes.filter(
      (index) => index.Scans === 0 && !index.Primary && !index.Unique,
    ).length;
    return (
      <KPIGrid>
        <KPI label="Observed indexes" value={indexes.length} />
        <KPI
          label="Total index size"
          value={bytes(
            indexes.reduce((sum, index) => sum + index.SizeBytes, 0),
          )}
        />
        <KPI
          label="Potentially unused"
          value={candidates}
          tone={candidates ? "warning" : "success"}
        />
      </KPIGrid>
    );
  }
  if (resource === "vacuum") {
    const tables = rows as TableStat[];
    const approaching = tables.filter(
      (table) => table.VacuumProgress >= 80,
    ).length;
    return (
      <KPIGrid>
        <KPI label="Tables monitored" value={tables.length} />
        <KPI
          label="Total dead tuples"
          value={fmt(tables.reduce((sum, table) => sum + table.DeadTuples, 0))}
        />
        <KPI
          label="Approaching threshold"
          value={approaching}
          tone={approaching ? "warning" : "success"}
        />
      </KPIGrid>
    );
  }
  if (resource === "locks") {
    const locks = rows as LockInfo[];
    const hasFreshEvidence = quality?.state === "fresh";
    if (!locks.length && !hasFreshEvidence)
      return (
        <div className="status-panel warning">
          <strong>No current lock evidence available</strong>
          <span>
            A fresh lock snapshot is required before blocking can be ruled out.
          </span>
        </div>
      );
    return (
      <div className={`status-panel ${locks.length ? "danger" : "success"}`}>
        <strong>
          {locks.length
            ? `${locks.length} blocking session${locks.length === 1 ? "" : "s"} detected`
            : "No blocking sessions detected"}
        </strong>
        <span>
          {locks.length
            ? "Inspect the blocking chain and duration below."
            : "The latest lock snapshot contains no blocked sessions."}
        </span>
      </div>
    );
  }
  if (resource === "wait-events") {
    const waits = rows as WaitEventSample[];
    const hasFreshEvidence = quality?.state === "fresh";
    if (!waits.length && !hasFreshEvidence)
      return (
        <div className="status-panel warning">
          <strong>Current wait state unavailable</strong>
          <span>
            A fresh wait-event snapshot is required before current waits can be
            ruled out.
          </span>
        </div>
      );
    if (!waits.length)
      return (
        <div className="status-panel success">
          <strong>
            No sessions are currently reporting a PostgreSQL wait event.
          </strong>
          <span>The latest wait-event snapshot is fresh.</span>
        </div>
      );
    const databases = new Set(
      waits.map((wait) => wait.Database).filter(Boolean),
    );
    const classes = waitClassCounts(waits);
    const top = classes[0];
    return (
      <>
        <KPIGrid>
          <KPI label="Waiting sessions" value={waits.length} tone="warning" />
          <KPI label="Affected databases" value={databases.size} />
          <KPI label="Top wait class" value={top?.name ?? "—"} />
          <KPI
            label="Longest query age"
            value={duration(
              Math.floor(
                Math.max(...waits.map((wait) => wait.QueryAgeSeconds || 0)),
              ),
            )}
          />
        </KPIGrid>
        <section
          className="wait-distribution"
          aria-label="Wait class distribution"
        >
          <div className="section-heading">
            <div>
              <strong>Wait class distribution</strong>
              <span>Exact share of sessions in this latest snapshot.</span>
            </div>
          </div>
          {classes.map((item) => (
            <div className="wait-distribution-row" key={item.name}>
              <strong>{item.name || "Unknown"}</strong>
              <span className="wait-bar">
                <i style={{ width: `${item.share}%` }} />
              </span>
              <span>
                {item.count} · {item.share.toFixed(1)}%
              </span>
            </div>
          ))}
        </section>
      </>
    );
  }
  return null;
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
  database,
}: {
  resource: string;
  data: unknown;
  database: string;
}) {
  if (resource === "replication")
    return <ReplicationView value={data as ReplicationStats} />;
  if (resource === "wal") return <WALView value={data as WALStats} />;
  const allRows = data as unknown[];
  const rows = allRows.filter(
    (row) => !database || (row as { Database?: string }).Database === database,
  );
  if (database && allRows.length > 0 && rows.length === 0)
    return (
      <Empty
        title="No data for the selected database"
        detail={`The latest ${resourceLabel(resource)} snapshot contains data for this server, but none for ${database}.`}
      />
    );
  if (resource === "locks" && rows.length === 0) return null;
  if (resource === "wait-events") {
    if (rows.length === 0) return null;
    return <WaitEventsTable rows={rows as WaitEventSample[]} />;
  }
  if (rows.length === 0)
    return (
      <Empty
        title="No data collected yet"
        detail="Evidence will appear after a successful monitoring cycle."
      />
    );
  if (resource === "queries") return <QueryTable rows={rows as QueryStat[]} />;
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
        resource === "vacuum" ? (
          <span className="progress-cell">
            <i>
              <b
                style={{
                  width: `${Math.min(100, Math.max(0, item.VacuumProgress))}%`,
                }}
              />
            </i>
            <span>{Math.round(item.VacuumProgress)}%</span>
          </span>
        ) : item.LastAutovacuum ? (
          new Date(item.LastAutovacuum).toLocaleString()
        ) : (
          "Never"
        ),
      ])}
    />
  );
}
function resourceLabel(resource: string) {
  return (
    (
      {
        queries: "query",
        tables: "table",
        indexes: "index",
        vacuum: "vacuum",
        locks: "lock",
        "wait-events": "wait-event",
      } as Record<string, string>
    )[resource] ?? resource
  );
}

function waitClassCounts(rows: WaitEventSample[]) {
  const counts = new Map<string, number>();
  rows.forEach((row) =>
    counts.set(
      row.WaitEventType || "Unknown",
      (counts.get(row.WaitEventType || "Unknown") ?? 0) + 1,
    ),
  );
  return [...counts.entries()]
    .map(([name, count]) => ({
      name,
      count,
      share: rows.length ? (count / rows.length) * 100 : 0,
    }))
    .sort((a, b) => b.count - a.count || a.name.localeCompare(b.name));
}

function WaitEventsTable({ rows }: { rows: WaitEventSample[] }) {
  const [search, setSearch] = useState("");
  const [waitClass, setWaitClass] = useState("");
  const [application, setApplication] = useState("");
  const [state, setState] = useState("");
  const values = (field: "WaitEventType" | "Application" | "State") =>
    [...new Set(rows.map((row) => row[field]).filter(Boolean))].sort();
  const visible = rows.filter(
    (row) =>
      (!waitClass || row.WaitEventType === waitClass) &&
      (!application || row.Application === application) &&
      (!state || row.State === state) &&
      (!search ||
        `${row.Query} ${row.WaitEvent} ${row.User} ${row.PID}`
          .toLowerCase()
          .includes(search.toLowerCase())),
  );
  return (
    <>
      <div className="table-toolbar wait-toolbar">
        <label className="search-field">
          <Search />
          <span className="sr-only">Search wait evidence</span>
          <input
            aria-label="Search wait evidence"
            placeholder="Search query, event, user, or PID"
            value={search}
            onChange={(event) => setSearch(event.target.value)}
          />
        </label>
        <select
          aria-label="Filter wait class"
          value={waitClass}
          onChange={(event) => setWaitClass(event.target.value)}
        >
          <option value="">All wait classes</option>
          {values("WaitEventType").map((value) => (
            <option key={value}>{value}</option>
          ))}
        </select>
        <select
          aria-label="Filter application"
          value={application}
          onChange={(event) => setApplication(event.target.value)}
        >
          <option value="">All applications</option>
          {values("Application").map((value) => (
            <option key={value}>{value}</option>
          ))}
        </select>
        <select
          aria-label="Filter state"
          value={state}
          onChange={(event) => setState(event.target.value)}
        >
          <option value="">All states</option>
          {values("State").map((value) => (
            <option key={value}>{value}</option>
          ))}
        </select>
        <span>
          {visible.length} of {rows.length} sessions
        </span>
      </div>
      {visible.length ? (
        <Table
          headers={[
            "Database",
            "PID",
            "User",
            "Application",
            "State",
            "Wait class",
            "Wait event",
            "Query age",
            "Transaction age",
            "Query",
          ]}
          numeric={[1, 7, 8]}
          rows={visible.map((wait) => [
            wait.Database || "—",
            <code>{wait.PID}</code>,
            wait.User || "—",
            wait.Application || "—",
            wait.State || "—",
            <span className="assessment warning">
              {wait.WaitEventType || "Unknown"}
            </span>,
            wait.WaitEvent || "Unknown",
            duration(Math.floor(wait.QueryAgeSeconds || 0)),
            wait.TransactionStartedAt
              ? duration(Math.floor(wait.TransactionAgeSeconds || 0))
              : "—",
            <code className="query-text" title={wait.Query}>
              {wait.Query || "Query text unavailable"}
            </code>,
          ])}
        />
      ) : (
        <Empty
          title="No matching waits"
          detail="No current wait evidence matches these filters."
        />
      )}
    </>
  );
}

type QuerySort = "Calls" | "MeanExecMS" | "TotalExecMS" | "ImpactScore";
function QueryTable({ rows }: { rows: QueryStat[] }) {
  const [search, setSearch] = useState("");
  const [sort, setSort] = useState<QuerySort>("ImpactScore");
  const [ascending, setAscending] = useState(false);
  const visible = useMemo(
    () =>
      rows
        .filter((row) => row.Query.toLowerCase().includes(search.toLowerCase()))
        .sort((a, b) => (a[sort] - b[sort]) * (ascending ? 1 : -1)),
    [rows, search, sort, ascending],
  );
  function heading(label: string, key?: QuerySort) {
    return key ? (
      <button
        className="sort-button"
        onClick={() => {
          if (sort === key) setAscending((value) => !value);
          else {
            setSort(key);
            setAscending(false);
          }
        }}
        aria-label={`Sort by ${label}`}
      >
        {label}
        {sort === key && (ascending ? <ArrowUp /> : <ArrowDown />)}
      </button>
    ) : (
      label
    );
  }
  return (
    <>
      <div className="table-toolbar">
        <label className="search-field">
          <Search />
          <span className="sr-only">Search query text</span>
          <input
            aria-label="Search query text"
            placeholder="Search SQL text"
            value={search}
            onChange={(event) => setSearch(event.target.value)}
          />
        </label>
        <span>
          {visible.length} of {rows.length} queries
        </span>
      </div>
      {visible.length ? (
        <Table
          headers={[
            heading("Query"),
            heading("Database"),
            heading("Calls", "Calls"),
            heading("Avg time", "MeanExecMS"),
            heading("Total runtime", "TotalExecMS"),
            heading("Impact", "ImpactScore"),
          ]}
          numeric={[2, 3, 4, 5]}
          rows={visible.map((query) => [
            <code className="query-text" title={query.Query}>
              {query.Query}
            </code>,
            <code>{query.Database}</code>,
            fmt(query.Calls),
            `${fmt(query.MeanExecMS)} ms`,
            `${fmt(query.TotalExecMS)} ms`,
            <span className="impact-cell">
              <i
                style={{
                  width: `${Math.min(100, Math.max(2, query.ImpactScore))}%`,
                }}
              />
              {query.ImpactScore.toFixed(1)}
            </span>,
          ])}
        />
      ) : (
        <Empty
          title="No matching queries"
          detail="No collected query text matches this search."
        />
      )}
    </>
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
  headers: ReactNode[];
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
                key={index}
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
