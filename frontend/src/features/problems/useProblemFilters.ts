import { useEffect, useState } from "react";
import { useSearchParams } from "react-router-dom";

export function useProblemFilters(serverId: string, database: string) {
  const [urlParams, setUrlParams] = useSearchParams();
  const [severity, setSeverity] = useState(urlParams.get("severity") ?? "");
  const [status, setStatus] = useState(urlParams.get("status") ?? "active");
  const [category, setCategory] = useState(urlParams.get("category") ?? "");
  const [search, setSearch] = useState(urlParams.get("search") ?? "");
  const selectedId = urlParams.get("id");
  const query = new URLSearchParams({ status });
  if (severity) query.set("severity", severity);
  if (category.trim()) query.set("category", category.trim());
  if (search.trim()) query.set("search", search.trim());
  if (serverId) query.set("serverId", serverId);
  if (database) query.set("database", database);

  useEffect(() => {
    const next = new URLSearchParams();
    if (selectedId) next.set("id", selectedId);
    if (serverId) next.set("serverId", serverId);
    if (database) next.set("database", database);
    for (const [key, value] of [
      ["status", status],
      ["severity", severity],
      ["category", category],
      ["search", search],
    ]) {
      if (value) next.set(key, value);
      else next.delete(key);
    }
    if (status === "active") next.delete("status");
    setUrlParams(next, { replace: true });
  }, [status, severity, category, search, selectedId, serverId, database, setUrlParams]);

  const clear = () => {
    setSeverity("");
    setStatus("active");
    setCategory("");
    setSearch("");
  };
  return {
    severity, setSeverity, status, setStatus, category, setCategory, search, setSearch,
    selectedId, query, clear,
    filtersActive: Boolean(severity || category || search || status !== "active"),
  };
}
