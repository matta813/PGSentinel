import { Filter, Search } from "lucide-react";
import type { Dispatch, SetStateAction } from "react";
import type { Severity } from "../../types";

type Setter = Dispatch<SetStateAction<string>>;

export function ProblemFilters({ severity, setSeverity, status, setStatus, category, setCategory, search, setSearch, filtersActive, clear }: {
  severity: string; setSeverity: Setter; status: string; setStatus: Setter;
  category: string; setCategory: Setter; search: string; setSearch: Setter;
  filtersActive: boolean; clear: () => void;
}) {
  return <div className="problem-toolbar">
    <label className="search-field">
      <Search />
      <span className="sr-only">Search problems</span>
      <input aria-label="Search problems" placeholder="Search findings and evidence" value={search} maxLength={200} onChange={(event) => setSearch(event.target.value)} />
    </label>
    <div className="filter-controls">
      <span className="filter-label"><Filter /> Filters</span>
      <select aria-label="Severity" value={severity} onChange={(event) => setSeverity(event.target.value)}>
        <option value="">All severities</option>
        {(["CRITICAL", "HIGH", "MEDIUM", "LOW", "INFO"] as Severity[]).map((item) => <option key={item}>{item}</option>)}
      </select>
      <select aria-label="Status" value={status} onChange={(event) => setStatus(event.target.value)}>
        <option value="active">Active</option><option value="acknowledged">Acknowledged</option><option value="resolved">Resolved</option><option value="all">All statuses</option>
      </select>
      <input className="category-filter" aria-label="Category" placeholder="Category" value={category} onChange={(event) => setCategory(event.target.value)} />
      {filtersActive && <button className="button ghost compact" onClick={clear}>Clear</button>}
    </div>
  </div>;
}
