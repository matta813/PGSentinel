import { Info } from "lucide-react";
import { useParams } from "react-router-dom";
import { api, APIError } from "../../api/client";
import { Empty, ErrorState, Loading } from "../../components/Status";
import { PageHeader } from "../../components/UI";
import { useMonitoring } from "../../context/MonitoringContext";
import { useApi } from "../../hooks/useApi";
import type { CollectionResourceStatus } from "../../types";
import { FreshnessNotice } from "./shared/FreshnessNotice";
import { freshnessLabel } from "./shared/freshness";
import { ResourceSummary } from "./shared/ResourceSummary";
import { ResourceTable } from "./shared/ResourceTable";

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
