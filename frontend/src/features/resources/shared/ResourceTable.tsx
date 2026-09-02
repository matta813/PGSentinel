import { Empty } from "../../../components/Status";
import type { IndexStat, LockInfo, QueryStat, TableStat, WaitEventSample } from "../../../types";
import type { ReplicationStats, WALStats } from "../../../types/replication";
import { QueryTable } from "../queries/QueryView";
import { ReplicationView } from "../replication/ReplicationView";
import { WaitEventsTable } from "../wait-events/WaitEventsView";
import { WALView } from "../wal/WALView";
import { bytes, deadRatio, fmt } from "./formatters";
import { Table } from "./Table";

export function ResourceTable({
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
