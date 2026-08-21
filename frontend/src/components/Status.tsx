import { AlertCircle } from 'lucide-react'
import type { Severity } from '../types'
import { EmptyState, LoadingState } from './UI'

export function SeverityBadge({ severity }: { severity: Severity }) {
  return <span className={`badge ${severity.toLowerCase()}`}><i aria-hidden="true" />{severity}</span>
}
export function StatusIndicator({ status }: { status: string }) {
  return <span className={`status-indicator ${status.toLowerCase()}`}><i aria-hidden="true" />{status}</span>
}
export const Empty = EmptyState
export const Loading = LoadingState
export function ErrorState({ error, retry }: { error: Error; retry?: () => void }) {
  return <div className="error-state" role="alert"><AlertCircle /><div><strong>Unable to load data</strong><p>{error.message}</p>{retry && <button className="button secondary" onClick={retry}>Try again</button>}</div></div>
}
