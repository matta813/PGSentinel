import { Notice } from "../../components/UI";
import type { Finding } from "../../types";

export function FindingDetails({ finding }: { finding: Finding }) {
  return <>
    {finding.suppressed && <Notice tone="warning"><strong>{finding.maintenance ? "Maintenance window active." : "Finding suppressed."}</strong>{" "}{finding.suppressionReason} Evidence and lifecycle state are preserved.</Notice>}
    <section><h3>Summary</h3><p>{finding.summary}</p>{finding.cause && <p>{finding.cause}</p>}</section>
    <section><h3>Operational impact</h3><p>{finding.impact}</p></section>
    <section className="finding-evidence">
      <h3>Evidence</h3>
      {finding.evidenceQuality && finding.evidenceQuality.state !== "fresh" && <div className={`evidence-quality ${finding.evidenceQuality.state}`}><strong>{finding.evidenceQuality.state} evidence</strong><span>{finding.evidenceQuality.errorSummary} Existing findings remain open until a complete collection can verify recovery.</span></div>}
      <div className="evidence-grid">{finding.evidence?.map((item) => <div key={item.label}><span>{item.label}</span><strong>{item.value}{item.unit && <small> {item.unit}</small>}</strong></div>)}</div>
    </section>
    <section><h3>Next investigation</h3><ol>{finding.suggestions?.map((suggestion, index) => <li key={index}><strong>{suggestion.title}</strong>{suggestion.detail && <p>{suggestion.detail}</p>}</li>)}</ol></section>
  </>;
}
