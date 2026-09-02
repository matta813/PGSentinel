import type { WaitEventSample } from "../../../types";

export function waitClassCounts(rows: WaitEventSample[]) {
  const counts = new Map<string, number>();
  rows.forEach((row) =>
    counts.set(
      row.WaitEventType || "Unknown",
      (counts.get(row.WaitEventType || "Unknown") ?? 0) + 1,
    ),
  );
  return [...counts.entries()]
    .map(([name, count]) => ({
      name,
      count,
      share: rows.length ? (count / rows.length) * 100 : 0,
    }))
    .sort((a, b) => b.count - a.count || a.name.localeCompare(b.name));
}
