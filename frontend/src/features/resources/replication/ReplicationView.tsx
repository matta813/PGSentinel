import { Empty } from "../../../components/Status";
import type { ReplicationStats } from "../../../types/replication";
import { bytes, bytesRate, fmt } from "../shared/formatters";
import { Table } from "../shared/Table";

export function ReplicationView({ value }: { value: ReplicationStats }) {
  const standbys = value.Standbys ?? [];
  const slots = value.Slots ?? [];
  return (
    <div className="form-stack">
      <div className="evidence-grid">
        <article>
          <span>Server role</span>
          <strong>{value.InRecovery ? "Standby" : "Primary"}</strong>
        </article>
        <article>
          <span>Timeline</span>
          <strong>{value.TimelineID || "—"}</strong>
        </article>
        <article>
          <span>Recovery replay</span>
          <strong>{value.RecoveryPaused ? "Paused" : "Running"}</strong>
        </article>
        <article>
          <span>Configured replay delay</span>
          <strong>{fmt(value.ReplayDelaySeconds || 0)} sec</strong>
        </article>
      </div>
      {standbys.length > 0 && (
        <Table
          headers={[
            "Standby",
            "State",
            "Sync",
            "Replay lag",
            "Pending replay",
            "Gap growth",
            "Reply age",
          ]}
          numeric={[3, 4, 5, 6]}
          rows={standbys.map((item) => [
            item.Application || item.ClientAddress,
            item.State,
            item.SyncState,
            `${fmt(item.ReplayLagSeconds)} sec`,
            bytes(item.PendingReplayBytes),
            `${bytesRate(item.PendingReplayGrowthBytesPerSecond)}`,
            `${fmt(item.ReplyAgeSeconds)} sec`,
          ])}
        />
      )}{" "}
      {slots.length > 0 && (
        <Table
          headers={[
            "Slot",
            "Type",
            "State",
            "WAL status",
            "Retained",
            "Growth",
            "Inactive",
          ]}
          numeric={[4, 5, 6]}
          rows={slots.map((item) => [
            item.Name,
            item.Type,
            item.Active ? "Active" : "Inactive",
            item.WALStatus || "reserved",
            bytes(item.RetainedBytes),
            bytesRate(item.RetainedGrowthBytesPerSecond),
            `${fmt(item.InactiveSeconds)} sec`,
          ])}
        />
      )}{" "}
      {standbys.length === 0 && slots.length === 0 && !value.InRecovery && (
        <Empty
          title="No replication consumers observed"
          detail="A primary without replicas or slots is not considered unhealthy."
        />
      )}
    </div>
  );
}
