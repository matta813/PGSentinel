export interface AuditEvent { id: string; occurredAt: string; actor: string; action: string; resourceType: string; resourceId?: string; summary: string }
