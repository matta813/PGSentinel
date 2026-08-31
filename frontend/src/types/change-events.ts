export interface ChangeEvent {
  id: string
  serverId: string
  kind: 'deployment' | 'configuration'
  summary: string
  details: string[]
  occurredAt: string
  createdAt: string
}
