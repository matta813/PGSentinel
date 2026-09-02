import { Check, ChevronDown, Clock3, RotateCcw, ShieldOff } from "lucide-react";
import { useState } from "react";
import { SeverityBadge } from "../../components/Status";
import type { Finding } from "../../types";
import { FindingDetails } from "./FindingDetails";
import { relativeTime } from "./relativeTime";
import { SuppressionForm } from "./SuppressionForm";

export function FindingRow({ finding, defaultOpen, onStatus, reload }: {
  finding: Finding; defaultOpen: boolean;
  onStatus: (id: string, status: "active" | "acknowledged") => Promise<void>;
  reload: () => Promise<void>;
}) {
  const [suppressing, setSuppressing] = useState(false);
  const [reason, setReason] = useState("");
  const [hours, setHours] = useState(1);
  return <details id={`finding-${finding.id}`} open={defaultOpen || undefined} className={`finding severity-${finding.severity.toLowerCase()}`}>
    <summary>
      <SeverityBadge severity={finding.severity} />
      <div className="finding-primary"><strong>{finding.title}</strong><span>{finding.database || "Server scope"}{finding.resource && ` · ${finding.resource}`}</span></div>
      <span className="finding-category">{finding.category}</span>
      <span className={`finding-status ${finding.suppressed ? "suppressed" : finding.status}`}>{finding.suppressed ? "suppressed" : finding.status}</span>
      <time dateTime={finding.updatedAt}><Clock3 />{relativeTime(finding.updatedAt)}</time>
      <ChevronDown className="disclosure" />
    </summary>
    <div className="finding-detail">
      <FindingDetails finding={finding} />
      {suppressing && <SuppressionForm finding={finding} reason={reason} setReason={setReason} hours={hours} setHours={setHours} reload={reload} onComplete={() => setSuppressing(false)} />}
      <footer>
        <span>Confidence <strong>{finding.confidence}</strong></span>
        {!finding.suppressed && <button className="button secondary" onClick={() => setSuppressing((value) => !value)}><ShieldOff /> Suppress</button>}
        {finding.status === "active" && <button className="button secondary" onClick={() => void onStatus(finding.id, "acknowledged")}><Check /> Acknowledge</button>}
        {finding.status === "acknowledged" && <button className="button secondary" onClick={() => void onStatus(finding.id, "active")}><RotateCcw /> Reopen</button>}
      </footer>
    </div>
  </details>;
}
