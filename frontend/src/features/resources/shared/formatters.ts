import type { TableStat } from "../../../types";

export function duration(seconds: number) {
  if (seconds < 60) return `${seconds}s`;
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m`;
  return `${Math.floor(seconds / 3600)}h`;
}

export function fmt(value: number) {
  return Intl.NumberFormat("en", {
    notation: value > 9999 ? "compact" : "standard",
    maximumFractionDigits: 1,
  }).format(value || 0);
}
export function bytes(value: number) {
  if (value >= 1024 ** 3) return `${fmt(value / 1024 ** 3)} GB`;
  if (value >= 1024 ** 2) return `${fmt(value / 1024 ** 2)} MB`;
  return `${fmt(value / 1024)} KB`;
}
export function bytesRate(value: number) {
  const prefix = value < 0 ? "-" : "";
  return `${prefix}${bytes(Math.abs(value || 0))}/s`;
}
export function deadRatio(table: TableStat) {
  const count = table.LiveTuples + table.DeadTuples;
  return count ? ((table.DeadTuples / count) * 100).toFixed(1) : "0";
}
