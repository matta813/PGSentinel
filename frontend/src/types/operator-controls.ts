export interface MaintenanceWindow {
  id: string;
  description: string;
  serverId?: string;
  serverTag?: string;
  category?: string;
  ruleId?: string;
  startsAt: string;
  endsAt: string;
  state: "active" | "upcoming" | "expired";
}
export interface FindingSuppression {
  id: string;
  findingId?: string;
  ruleId?: string;
  serverId?: string;
  serverTag?: string;
  reason: string;
  expiresAt: string;
  state: "active" | "expired";
}
export interface ThresholdOverride {
  id: string;
  ruleId: string;
  scopeType: "global" | "server" | "tag";
  scopeValue: string;
  value: number;
  reason: string;
}
export interface ThresholdSpec {
  label: string;
  min: number;
  max: number;
  default: number;
  unit: string;
}
export interface ThresholdResponse {
  items: ThresholdOverride[];
  specs: Record<string, ThresholdSpec>;
}
