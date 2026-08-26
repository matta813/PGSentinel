import type { Finding, Severity } from './index'

export interface IncidentEvent { at: string; type: 'finding_started' | 'finding_resolved'; findingId: string; title: string; detail: string; severity: Severity }
export interface Incident { id: string; serverId: string; title: string; summary: string; rationale: string[]; severity: Severity; status: 'active' | 'resolved'; startedAt: string; updatedAt: string; resolvedAt?: string; findings?: Finding[]; timeline?: IncidentEvent[] }
