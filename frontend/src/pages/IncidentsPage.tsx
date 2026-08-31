import { ArrowRight, Clock3, Link2, ListTree } from "lucide-react";
import { useMemo, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { api } from "../api/client";
import {
  Empty,
  ErrorState,
  Loading,
  SeverityBadge,
} from "../components/Status";
import { Notice, PageHeader } from "../components/UI";
import { useApi } from "../hooks/useApi";
import type { Server } from "../types";
import type { Incident } from "../types/incidents";
import { useMonitoring } from "../context/MonitoringContext";

export function IncidentsPage() {
  const [params, setParams] = useSearchParams();
  const [status, setStatus] = useState(
    params.get("status") === "all" ? "all" : "active",
  );
  const selected = params.get("id");
  const monitoring = useMonitoring();
  const incidentQuery = new URLSearchParams({ status, limit: "50" });
  if (monitoring.selectedServerId)
    incidentQuery.set("serverId", monitoring.selectedServerId);
  const {
    data: incidents,
    error,
    loading,
    reload,
  } = useApi(
    () => api.get<Incident[]>(`/incidents?${incidentQuery}`),
    [status, monitoring.selectedServerId],
  );
  const { data: servers } = useApi(() => api.get<Server[]>("/servers"), []);
  const {
    data: detail,
    error: detailError,
    loading: detailLoading,
  } = useApi(
    () =>
      selected
        ? api.get<Incident>(`/incidents/${selected}`)
        : Promise.resolve(undefined),
    [selected],
  );
  const names = useMemo(
    () => new Map(servers?.map((server) => [server.id, server.name]) ?? []),
    [servers],
  );
  if (loading) return <Loading />;
  if (error) return <ErrorState error={error} retry={reload} />;
  return (
    <>
      <PageHeader
        title="Incidents"
        description="Inspect cautiously grouped PostgreSQL evidence that overlapped in time."
        actions={
          <label className="compact-select">
            Status
            <select
              aria-label="Incident status"
              value={status}
              onChange={(event) => {
                const next = event.target.value;
                setStatus(next);
                const values: Record<string, string> = { status: next };
                if (selected) values.id = selected;
                if (monitoring.selectedServerId)
                  values.serverId = monitoring.selectedServerId;
                setParams(values);
              }}
            >
              <option value="active">Active</option>
              <option value="all">All</option>
            </select>
          </label>
        }
      />
      <Notice>
        <strong>Correlation is not causation.</strong> PGSentinel groups only
        explicit PostgreSQL relationships that began within the same 15-minute
        period.
      </Notice>
      {incidents?.length === 0 ? (
        <Empty
          title="No correlated incidents"
          detail="No findings currently meet the conservative time and relationship criteria. Individual evidence remains available in Problems."
          action={
            <Link className="text-link" to="/problems">
              Open problems <ArrowRight />
            </Link>
          }
        />
      ) : (
        <div className={`incident-layout ${selected ? "has-selection" : ""}`}>
          <div className="incident-list" aria-label="Correlated incidents">
            {incidents?.map((incident) => (
              <button
                key={incident.id}
                className={`incident-row ${selected === incident.id ? "selected" : ""}`}
                onClick={() =>
                  setParams({
                    status,
                    id: incident.id,
                    ...(monitoring.selectedServerId
                      ? { serverId: monitoring.selectedServerId }
                      : {}),
                  })
                }
              >
                <SeverityBadge severity={incident.severity} />
                <span>
                  <strong>{incident.title}</strong>
                  <small>
                    {names.get(incident.serverId) ?? "PostgreSQL server"} ·{" "}
                    {formatDate(incident.startedAt)}
                  </small>
                </span>
                <span className={`finding-status ${incident.status}`}>
                  {incident.status}
                </span>
                <ArrowRight />
              </button>
            ))}
          </div>
          {selected && (
            <aside className="incident-detail" aria-label="Incident detail">
              {detailLoading ? (
                <Loading />
              ) : detailError ? (
                <ErrorState error={detailError} />
              ) : (
                detail && (
                  <IncidentDetail
                    incident={detail}
                    serverName={names.get(detail.serverId)}
                  />
                )
              )}
            </aside>
          )}
        </div>
      )}
    </>
  );
}

function IncidentDetail({
  incident,
  serverName,
}: {
  incident: Incident;
  serverName?: string;
}) {
  return (
    <>
      <header>
        <div>
          <span className="eyebrow">
            {serverName ?? "PostgreSQL server"} · {incident.status}
          </span>
          <h2>{incident.title}</h2>
        </div>
        <SeverityBadge severity={incident.severity} />
      </header>
      <p>{incident.summary}</p>
      <section>
        <h3>
          <Link2 /> Why these were grouped
        </h3>
        <ul>
          {incident.rationale.map((reason) => (
            <li key={reason}>{reason}</li>
          ))}
        </ul>
      </section>
      <section>
        <h3>
          <ListTree /> Overlapping evidence
        </h3>
        <div className="incident-findings">
          {incident.findings?.map((finding) => (
            <Link to={`/problems?id=${finding.id}`} key={finding.id}>
              <SeverityBadge severity={finding.severity} />
              <span>
                <strong>{finding.title}</strong>
                <small>
                  {finding.category}
                  {finding.database && ` · ${finding.database}`}
                </small>
              </span>
              <ArrowRight />
            </Link>
          ))}
        </div>
      </section>
      <section>
        <h3>
          <Clock3 /> Timeline
        </h3>
        <ol className="incident-timeline">
          {incident.timeline?.map((event, index) => (
            <li key={`${event.findingId}-${event.type}-${index}`}>
              <time dateTime={event.at}>{formatDate(event.at)}</time>
              <i className={event.type} />
              <div>
                <strong>
                  {event.type === "finding_started"
                    ? "Finding observed"
                    : "Finding resolved"}
                </strong>
                <span>{event.title}</span>
                <small>{event.detail}</small>
              </div>
            </li>
          ))}
        </ol>
      </section>
    </>
  );
}

function formatDate(value: string) {
  const date = new Date(value);
  return Number.isFinite(date.getTime())
    ? date.toLocaleString([], { dateStyle: "medium", timeStyle: "short" })
    : "Unknown time";
}
