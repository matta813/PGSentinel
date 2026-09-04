import { KPI, KPIGrid } from "../../../components/UI";
import type { CollectionResourceStatus, IndexStat, LockInfo, TableStat, WaitEventSample } from "../../../types";
import { waitClassCounts } from "../wait-events/waitEventCounts";
import { bytes, duration, fmt } from "./formatters";

export function ResourceSummary({
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
    if (!indexes.length && quality?.state !== "fresh")
      return (
        <div className="status-panel warning">
          <strong>Index evidence unavailable</strong>
          <span>
            A successful index snapshot is required before index counts and
            usage can be assessed.
          </span>
        </div>
      );
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
    if (!tables.length && quality?.state !== "fresh")
      return (
        <div className="status-panel warning">
          <strong>Vacuum evidence unavailable</strong>
          <span>
            A successful table-statistics snapshot is required before vacuum
            health can be assessed.
          </span>
        </div>
      );
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
