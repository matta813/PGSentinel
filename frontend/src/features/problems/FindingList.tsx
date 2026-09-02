import type { Finding } from "../../types";
import { FindingRow } from "./FindingRow";

export function FindingList({ findings, selectedId, onStatus, reload }: {
  findings: Finding[]; selectedId: string | null;
  onStatus: (id: string, status: "active" | "acknowledged") => Promise<void>;
  reload: () => Promise<void>;
}) {
  return <div className="inbox" aria-label="Problem findings">
    {findings.map((finding) => <FindingRow key={finding.id} finding={finding} defaultOpen={finding.id === selectedId} onStatus={onStatus} reload={reload} />)}
  </div>;
}
