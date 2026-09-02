import type { WALStats } from "../../../types/replication";
import { bytes, bytesRate, fmt } from "../shared/formatters";
import { Table } from "../shared/Table";

export function WALView({ value }: { value: WALStats }) {
  return (
    <div className="form-stack">
      <div className="evidence-grid">
        <article>
          <span>WAL generation</span>
          <strong>{bytesRate(value.GenerationBytesPerSecond)}</strong>
        </article>
        <article>
          <span>WAL generated</span>
          <strong>{bytes(value.WALBytes)}</strong>
        </article>
        <article>
          <span>Archive mode</span>
          <strong>{value.ArchiveMode || "unknown"}</strong>
        </article>
        <article>
          <span>Archive destination</span>
          <strong>
            {value.ArchiveConfigured ? "Configured" : "Not configured"}
          </strong>
        </article>
      </div>
      <Table
        headers={[
          "Archived",
          "Failed",
          "Last archived WAL",
          "Last failed WAL",
          "Timed checkpoints",
          "Requested checkpoints",
          "Restartpoints done",
        ]}
        numeric={[0, 1, 4, 5, 6]}
        rows={[
          [
            fmt(value.ArchivedCount),
            fmt(value.FailedArchiveCount),
            value.LastArchivedWAL || "—",
            value.LastFailedWAL || "—",
            fmt(value.TimedCheckpoints),
            fmt(value.RequestedCheckpoints),
            fmt(value.RestartpointsDone),
          ],
        ]}
      />
    </div>
  );
}
