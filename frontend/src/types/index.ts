export type Severity = "INFO" | "LOW" | "MEDIUM" | "HIGH" | "CRITICAL";
export interface Server {
  id: string;
  name: string;
  host: string;
  port: number;
  user: string;
  sslMode: string;
  version?: string;
  status: string;
  lastConnectedAt?: string;
  lastError?: string;
  tags: string[];
}
export interface Evidence {
  label: string;
  value: string;
  unit?: string;
}
export interface CollectionResourceStatus {
  serverId: string;
  resource: string;
  state: "fresh" | "stale" | "partial" | "unavailable";
  collectedAt?: string;
  lastAttemptAt?: string;
  lastSuccessfulCollection?: string;
  ageSeconds?: number;
  expectedIntervalSeconds: number;
  consecutiveFailures: number;
  errorSummary?: string;
}
export interface Finding {
  id: string;
  serverId: string;
  database?: string;
  resource?: string;
  severity: Severity;
  category: string;
  title: string;
  summary: string;
  cause: string;
  impact: string;
  evidence: Evidence[];
  suggestions: { title: string; detail?: string }[];
  confidence: string;
  status: string;
  startedAt: string;
  updatedAt: string;
  suppressed: boolean;
  suppressionReason?: string;
  maintenance: boolean;
  evidenceQuality?: CollectionResourceStatus;
}
export interface Score {
  overall: number;
  categories: Record<string, number>;
}
export interface Overview {
  servers: Server[];
  problems: Finding[];
  counts: Partial<Record<Severity, number>>;
  score: Score;
  freshness: Record<string, CollectionResourceStatus[]>;
}
export interface ConnectionStats {
  Active: number;
  Idle: number;
  IdleInTransaction: number;
  Waiting: number;
  Total: number;
  Max: number;
  Utilization: number;
}
export interface Snapshot {
  serverId: string;
  collectedAt: string;
  version: string;
  uptimeSeconds: number;
  connections: ConnectionStats;
  databases: DatabaseStat[];
  capabilities: Record<string, boolean>;
}
export interface DatabaseStat {
  Name: string;
  SizeBytes: number;
  Commits: number;
  Rollbacks: number;
  Deadlocks: number;
  TempFiles: number;
  TempBytes: number;
  BlocksRead: number;
  BlocksHit: number;
}
export interface QueryStat {
  QueryID: string;
  Database: string;
  Query: string;
  Calls: number;
  TotalExecMS: number;
  MeanExecMS: number;
  MinExecMS: number;
  MaxExecMS: number;
  Rows: number;
  SharedHit: number;
  SharedRead: number;
  TempRead: number;
  TempWritten: number;
  WALBytes: number;
  ImpactScore: number;
}
export interface TableStat {
  Database: string;
  Schema: string;
  Table: string;
  EstimatedRows: number;
  TotalSize: number;
  IndexSize: number;
  LiveTuples: number;
  DeadTuples: number;
  SeqScans: number;
  IndexScans: number;
  LastAutovacuum?: string;
  VacuumThreshold: number;
  VacuumProgress: number;
}
export interface IndexStat {
  Database: string;
  Schema: string;
  Table: string;
  Index: string;
  Definition: string;
  SizeBytes: number;
  Scans: number;
  Unique: boolean;
  Primary: boolean;
}
export interface LockInfo {
  BlockedPID: number;
  BlockingPID: number;
  DurationSeconds: number;
  Database: string;
  User: string;
  Application: string;
  Query: string;
  BlockingQuery: string;
}
export interface WaitEventSample {
  PID: number;
  Database: string;
  User: string;
  Application: string;
  ClientAddress: string;
  BackendType: string;
  State: string;
  WaitEventType: string;
  WaitEvent: string;
  Query: string;
  QueryStartedAt?: string;
  TransactionStartedAt?: string;
  StateChangedAt?: string;
  QueryAgeSeconds: number;
  TransactionAgeSeconds: number;
  StateAgeSeconds: number;
}
