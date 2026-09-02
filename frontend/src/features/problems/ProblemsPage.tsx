import { useEffect, useState } from "react";
import { api, APIError } from "../../api/client";
import { Empty, ErrorState, Loading } from "../../components/Status";
import { Notice, PageHeader } from "../../components/UI";
import { useMonitoring } from "../../context/MonitoringContext";
import { useApi } from "../../hooks/useApi";
import type { Finding } from "../../types";
import { FindingList } from "./FindingList";
import { ProblemFilters } from "./ProblemFilters";
import { useProblemFilters } from "./useProblemFilters";

export function ProblemsPage() {
  const monitoring = useMonitoring();
  const filters = useProblemFilters(monitoring.selectedServerId, monitoring.selectedDatabase);
  const [message, setMessage] = useState("");
  const { data, error, loading, reload } = useApi(
    () => api.get<Finding[]>(`/problems?${filters.query}`),
    [filters.status, filters.severity, filters.category, filters.search, monitoring.selectedServerId, monitoring.selectedDatabase],
  );
  useEffect(() => {
    if (filters.selectedId && data)
      document.getElementById(`finding-${filters.selectedId}`)?.scrollIntoView({ block: "center" });
  }, [data, filters.selectedId]);
  async function setFindingStatus(id: string, nextStatus: "active" | "acknowledged") {
    setMessage("");
    try {
      await api.put(`/problems/${id}/status`, { status: nextStatus });
      void reload();
    } catch (reason) {
      setMessage(reason instanceof APIError ? `${reason.message}: ${reason.detail}` : "Unable to update problem");
    }
  }
  if (loading) return <Loading />;
  if (error) return <ErrorState error={error} retry={reload} />;
  return <>
    <PageHeader title="Problems" description="Triage findings, inspect evidence, and track investigation state." actions={<span className="result-count">{data?.length ?? 0} {data?.length === 1 ? "finding" : "findings"}</span>} />
    {message && <Notice tone="danger">{message}</Notice>}
    <ProblemFilters {...filters} />
    {data?.length === 0
      ? <Empty title="No matching findings" detail="The operations inbox has no findings for the selected filters." />
      : <FindingList findings={data ?? []} selectedId={filters.selectedId} onStatus={setFindingStatus} reload={reload} />}
  </>;
}
