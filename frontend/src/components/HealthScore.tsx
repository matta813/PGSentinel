export function HealthScore({ value }: { value: number }) {
  const label = value >= 90 ? 'Healthy' : value >= 70 ? 'Needs attention' : value >= 40 ? 'Degraded' : 'Critical'
  const tone = value >= 90 ? 'healthy' : value >= 70 ? 'warning' : 'critical'
  const clamped = Math.max(0, Math.min(100, value))
  return <article className={`health-score ${tone}`} aria-label={`Health score ${value} of 100, ${label}`}>
    <div className="health-score-heading"><span>Overall health</span></div>
    <div className="health-score-value"><strong>{value}</strong><span className="health-label"><i />{label}</span></div>
    <div className="health-track" aria-hidden="true"><i style={{ width: `${clamped}%` }} /></div>
    <p>Based on active findings and collection availability across the monitored estate.</p>
  </article>
}
