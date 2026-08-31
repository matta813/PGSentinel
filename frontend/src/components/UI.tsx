import type { ReactNode } from 'react'
import { CheckCircle2, Inbox, LoaderCircle, TriangleAlert } from 'lucide-react'

export function PageHeader({ eyebrow, title, description, actions, meta }: { eyebrow?: string; title: string; description?: string; actions?: ReactNode; meta?: ReactNode }) {
  return <div className="page-header"><div>{eyebrow && <p className="eyebrow">{eyebrow}</p>}<h1>{title}</h1>{description && <p className="page-description">{description}</p>}</div>{(actions || meta) && <div className="page-actions">{meta && <span className="page-meta">{meta}</span>}{actions}</div>}</div>
}

export function SectionHeader({ title, description, action }: { title: string; description?: string; action?: ReactNode }) {
  return <div className="section-header"><div><h2>{title}</h2>{description && <p>{description}</p>}</div>{action}</div>
}

export function KPIGrid({ children }: { children: ReactNode }) { return <div className="kpi-grid">{children}</div> }
export function KPI({ label, value, detail, tone = 'neutral' }: { label: string; value: ReactNode; detail?: string; tone?: 'neutral' | 'success' | 'warning' | 'danger' }) { return <div className={`kpi ${tone}`}><span>{label}</span><strong>{value}</strong>{detail && <small>{detail}</small>}</div> }
export function DataPanel({ title, meta, children }: { title?: string; meta?: ReactNode; children: ReactNode }) { return <section className="data-panel">{(title || meta) && <header><h2>{title}</h2>{meta}</header>}{children}</section> }
export function PageTabs({ children, label = 'Page sections' }: { children: ReactNode; label?: string }) { return <nav className="page-tabs" aria-label={label}>{children}</nav> }

export function Notice({ children, tone = 'neutral' }: { children: ReactNode; tone?: 'neutral' | 'danger' | 'success' | 'warning' }) {
  return <div className={`notice notice-${tone}`} role={tone === 'danger' ? 'alert' : 'status'}>{(tone === 'danger' || tone === 'warning') && <TriangleAlert />}{children}</div>
}

export function EmptyState({ title, detail, action, positive = false }: { title: string; detail: string; action?: ReactNode; positive?: boolean }) {
  const Icon = positive ? CheckCircle2 : Inbox
  return <div className={`empty-state ${positive ? 'positive' : ''}`}><span className="empty-state-icon"><Icon /></span><div><h3>{title}</h3><p>{detail}</p></div>{action && <div className="empty-action">{action}</div>}</div>
}

export function LoadingState({ label = 'Loading monitoring data' }: { label?: string }) {
  return <div className="loading-state" role="status"><LoaderCircle className="spin" /><span>{label}</span><div className="loading-lines"><i /><i /><i /></div></div>
}
