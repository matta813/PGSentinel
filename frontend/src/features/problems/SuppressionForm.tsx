import type { Dispatch, FormEvent, SetStateAction } from "react";
import { api } from "../../api/client";
import type { Finding } from "../../types";

export function SuppressionForm({ finding, reason, setReason, hours, setHours, reload, onComplete }: {
  finding: Finding; reason: string; setReason: Dispatch<SetStateAction<string>>;
  hours: number; setHours: Dispatch<SetStateAction<number>>;
  reload: () => Promise<void>; onComplete: () => void;
}) {
  async function suppress(event: FormEvent) {
    event.preventDefault();
    await api.post("/suppressions", { findingId: finding.id, reason, expiresAt: new Date(Date.now() + hours * 3600000).toISOString() });
    onComplete();
    await reload();
  }
  return <form className="inline-control-form" onSubmit={(event) => void suppress(event)}>
    <label>Reason<input required maxLength={500} value={reason} onChange={(event) => setReason(event.target.value)} /></label>
    <label>Duration<select value={hours} onChange={(event) => setHours(Number(event.target.value))}><option value={1}>1 hour</option><option value={4}>4 hours</option><option value={24}>24 hours</option><option value={168}>7 days</option></select></label>
    <button className="button primary">Suppress temporarily</button>
  </form>;
}
