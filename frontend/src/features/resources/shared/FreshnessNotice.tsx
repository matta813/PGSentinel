import type { CollectionResourceStatus } from "../../../types";

export function FreshnessNotice({ quality }: { quality?: CollectionResourceStatus }) {
  if (quality?.state === "fresh") return null;
  const heading =
    quality?.state === "stale"
      ? "This evidence is stale"
      : quality?.state === "partial"
        ? "This evidence is incomplete"
        : "Current evidence is unavailable";
  return (
    <div
      className={`freshness-notice ${quality?.state ?? "unavailable"}`}
      role="status"
    >
      <strong>{heading}</strong>
      <span>
        {quality?.errorSummary || "No successful collection has completed yet."}{" "}
        {quality?.lastSuccessfulCollection &&
          `Last success ${new Date(quality.lastSuccessfulCollection).toLocaleString()}.`}
      </span>
    </div>
  );
}
