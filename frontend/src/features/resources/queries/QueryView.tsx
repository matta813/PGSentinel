import { useMemo, useState } from "react";
import { ArrowDown, ArrowUp, Search } from "lucide-react";
import { Empty } from "../../../components/Status";
import type { QueryStat } from "../../../types";
import { fmt } from "../shared/formatters";
import { Table } from "../shared/Table";

type QuerySort = "Calls" | "MeanExecMS" | "TotalExecMS" | "ImpactScore";

export function QueryTable({ rows }: { rows: QueryStat[] }) {
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
