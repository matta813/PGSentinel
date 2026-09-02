import { useState } from "react";
import { Search } from "lucide-react";
import { Empty } from "../../../components/Status";
import type { WaitEventSample } from "../../../types";
import { duration } from "../shared/formatters";
import { Table } from "../shared/Table";

export function WaitEventsTable({ rows }: { rows: WaitEventSample[] }) {
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
