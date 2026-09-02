import type { CollectionResourceStatus } from "../../../types";
import { duration } from "./formatters";

export function freshnessLabel(quality?: CollectionResourceStatus) {
  if (!quality) return "Not collected";
  if (quality.state === "fresh" && quality.ageSeconds !== undefined)
    return `Fresh · ${duration(quality.ageSeconds)} old`;
  return `${quality.state} · ${quality.consecutiveFailures} consecutive failure${quality.consecutiveFailures === 1 ? "" : "s"}`;
}
