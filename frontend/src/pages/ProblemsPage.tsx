import { useEffect, useState, type FormEvent } from "react";
import {
  Check,
  ChevronDown,
  Clock3,
  Filter,
  RotateCcw,
  Search,
  ShieldOff,
} from "lucide-react";
import { useSearchParams } from "react-router-dom";
import { api, APIError } from "../api/client";
import {
  Empty,
  ErrorState,
  Loading,
  SeverityBadge,
} from "../components/Status";
import { Notice, PageHeader } from "../components/UI";
import { useApi } from "../hooks/useApi";
import type { Finding, Severity } from "../types";

export function ProblemsPage() {
  const [urlParams] = useSearchParams();
  const [severity, setSeverity] = useState("");
  const [status, setStatus] = useState("active");
  const [category, setCategory] = useState("");
  const [search, setSearch] = useState("");
  const [message, setMessage] = useState("");
  const selectedId = urlParams.get("id");
  const query = new URLSearchParams({ status });
  if (severity) query.set("severity", severity);
  if (category.trim()) query.set("category", category.trim());
  if (search.trim()) query.set("search", search.trim());
  const { data, error, loading, reload } = useApi(
    () => api.get<Finding[]>(`/problems?${query}`),
    [status, severity, category, search],
  );
  useEffect(() => {
    if (selectedId && data)
      document
        .getElementById(`finding-${selectedId}`)
        ?.scrollIntoView({ block: "center" });
  }, [data, selectedId]);
  async function setFindingStatus(
    id: string,
    nextStatus: "active" | "acknowledged",
  ) {
    setMessage("");
    try {
      await api.put(`/problems/${id}/status`, { status: nextStatus });
      void reload();
    } catch (reason) {
      setMessage(
        reason instanceof APIError
          ? `${reason.message}: ${reason.detail}`
          : "Unable to update problem",
      );
    }
  }
  if (loading) return <Loading />;
  if (error) return <ErrorState error={error} retry={reload} />;
  const filtersActive = Boolean(
    severity || category || search || status !== "active",
  );
  return (
    <>
      <PageHeader
        title="Problems"
        description="Triage findings, inspect evidence, and track investigation state."
        actions={
          <span className="result-count">
            {data?.length ?? 0} {data?.length === 1 ? "finding" : "findings"}
          </span>
        }
      />
      {message && <Notice tone="danger">{message}</Notice>}
      <div className="problem-toolbar">
        <label className="search-field">
          <Search />
          <span className="sr-only">Search problems</span>
          <input
            aria-label="Search problems"
            placeholder="Search findings and evidence"
            value={search}
            maxLength={200}
            onChange={(event) => setSearch(event.target.value)}
          />
        </label>
        <div className="filter-controls">
          <span className="filter-label">
            <Filter /> Filters
          </span>
          <select
            aria-label="Severity"
            value={severity}
            onChange={(event) => setSeverity(event.target.value)}
          >
            <option value="">All severities</option>
            {(["CRITICAL", "HIGH", "MEDIUM", "LOW", "INFO"] as Severity[]).map(
              (item) => (
                <option key={item}>{item}</option>
              ),
            )}
          </select>
          <select
            aria-label="Status"
            value={status}
            onChange={(event) => setStatus(event.target.value)}
          >
            <option value="active">Active</option>
            <option value="acknowledged">Acknowledged</option>
            <option value="resolved">Resolved</option>
            <option value="all">All statuses</option>
          </select>
          <input
            className="category-filter"
            aria-label="Category"
            placeholder="Category"
            value={category}
            onChange={(event) => setCategory(event.target.value)}
          />
          {filtersActive && (
            <button
              className="button ghost compact"
              onClick={() => {
                setSeverity("");
                setStatus("active");
                setCategory("");
                setSearch("");
              }}
            >
              Clear
            </button>
          )}
        </div>
      </div>
      {data?.length === 0 ? (
        <Empty
          title="No matching findings"
          detail="The operations inbox has no findings for the selected filters."
        />
      ) : (
        <div className="inbox" aria-label="Problem findings">
          {data?.map((finding) => (
            <FindingRow
              key={finding.id}
              finding={finding}
              defaultOpen={finding.id === selectedId}
              onStatus={setFindingStatus}
              reload={reload}
            />
          ))}
        </div>
      )}
    </>
  );
}

function FindingRow({
  finding,
  defaultOpen,
  onStatus,
  reload,
}: {
  finding: Finding;
  defaultOpen: boolean;
  onStatus: (id: string, status: "active" | "acknowledged") => Promise<void>;
  reload: () => Promise<void>;
}) {
  const [suppressing, setSuppressing] = useState(false);
  const [reason, setReason] = useState("");
  const [hours, setHours] = useState(1);
  async function suppress(event: FormEvent) {
    event.preventDefault();
    await api.post("/suppressions", {
      findingId: finding.id,
      reason,
      expiresAt: new Date(Date.now() + hours * 3600000).toISOString(),
    });
    setSuppressing(false);
    await reload();
  }
  return (
    <details
      id={`finding-${finding.id}`}
      open={defaultOpen || undefined}
      className={`finding severity-${finding.severity.toLowerCase()}`}
    >
      <summary>
        <SeverityBadge severity={finding.severity} />
        <div className="finding-primary">
          <strong>{finding.title}</strong>
          <span>
            {finding.database || "Server scope"}
            {finding.resource && ` · ${finding.resource}`}
          </span>
        </div>
        <span className="finding-category">{finding.category}</span>
        <span
          className={`finding-status ${finding.suppressed ? "suppressed" : finding.status}`}
        >
          {finding.suppressed ? "suppressed" : finding.status}
        </span>
        <time dateTime={finding.updatedAt}>
          <Clock3 />
          {relativeTime(finding.updatedAt)}
        </time>
        <ChevronDown className="disclosure" />
      </summary>
      <div className="finding-detail">
        {finding.suppressed && (
          <Notice tone="warning">
            <strong>
              {finding.maintenance
                ? "Maintenance window active."
                : "Finding suppressed."}
            </strong>{" "}
            {finding.suppressionReason} Evidence and lifecycle state are
            preserved.
          </Notice>
        )}
        <section>
          <h3>What is wrong</h3>
          <p>{finding.summary}</p>
          {finding.cause && <p>{finding.cause}</p>}
        </section>
        <section>
          <h3>Why it matters</h3>
          <p>{finding.impact}</p>
        </section>
        <section className="finding-evidence">
          <h3>Evidence</h3>
          {finding.evidenceQuality &&
            finding.evidenceQuality.state !== "fresh" && (
              <div
                className={`evidence-quality ${finding.evidenceQuality.state}`}
              >
                <strong>{finding.evidenceQuality.state} evidence</strong>
                <span>
                  {finding.evidenceQuality.errorSummary} Existing findings
                  remain open until a complete collection can verify recovery.
                </span>
              </div>
            )}
          <div className="evidence-grid">
            {finding.evidence?.map((item) => (
              <div key={item.label}>
                <span>{item.label}</span>
                <strong>
                  {item.value}
                  {item.unit && <small> {item.unit}</small>}
                </strong>
              </div>
            ))}
          </div>
        </section>
        <section>
          <h3>What to investigate</h3>
          <ol>
            {finding.suggestions?.map((suggestion, index) => (
              <li key={index}>
                <strong>{suggestion.title}</strong>
                {suggestion.detail && <p>{suggestion.detail}</p>}
              </li>
            ))}
          </ol>
        </section>
        {suppressing && (
          <form
            className="inline-control-form"
            onSubmit={(event) => void suppress(event)}
          >
            <label>
              Reason
              <input
                required
                maxLength={500}
                value={reason}
                onChange={(event) => setReason(event.target.value)}
              />
            </label>
            <label>
              Duration
              <select
                value={hours}
                onChange={(event) => setHours(Number(event.target.value))}
              >
                <option value={1}>1 hour</option>
                <option value={4}>4 hours</option>
                <option value={24}>24 hours</option>
                <option value={168}>7 days</option>
              </select>
            </label>
            <button className="button primary">Suppress temporarily</button>
          </form>
        )}
        <footer>
          <span>
            Confidence <strong>{finding.confidence}</strong>
          </span>
          {!finding.suppressed && (
            <button
              className="button secondary"
              onClick={() => setSuppressing((value) => !value)}
            >
              <ShieldOff /> Suppress
            </button>
          )}
          {finding.status === "active" && (
            <button
              className="button secondary"
              onClick={() => void onStatus(finding.id, "acknowledged")}
            >
              <Check /> Acknowledge
            </button>
          )}
          {finding.status === "acknowledged" && (
            <button
              className="button secondary"
              onClick={() => void onStatus(finding.id, "active")}
            >
              <RotateCcw /> Reopen
            </button>
          )}
        </footer>
      </div>
    </details>
  );
}

function relativeTime(value: string) {
  const timestamp = new Date(value).getTime();
  if (!Number.isFinite(timestamp)) return "Unknown";
  const minutes = Math.max(0, Math.floor((Date.now() - timestamp) / 60000));
  if (minutes < 1) return "Just now";
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  return `${Math.floor(hours / 24)}d ago`;
}
