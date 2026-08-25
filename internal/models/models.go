package models

import "time"

type User struct {
	ID                 string
	Username           string
	PasswordHash       []byte
	PasswordSalt       []byte
	MustChangePassword bool
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type Severity string

const (
	SeverityInfo     Severity = "INFO"
	SeverityLow      Severity = "LOW"
	SeverityMedium   Severity = "MEDIUM"
	SeverityHigh     Severity = "HIGH"
	SeverityCritical Severity = "CRITICAL"
)

type Confidence string

const (
	ConfidenceLow    Confidence = "LOW"
	ConfidenceMedium Confidence = "MEDIUM"
	ConfidenceHigh   Confidence = "HIGH"
)

type Server struct {
	ID              string     `json:"id"`
	Name            string     `json:"name"`
	Host            string     `json:"host"`
	Port            int        `json:"port"`
	User            string     `json:"user"`
	Password        string     `json:"password,omitempty"`
	SSLMode         string     `json:"sslMode"`
	Version         string     `json:"version,omitempty"`
	Status          string     `json:"status"`
	LastConnectedAt *time.Time `json:"lastConnectedAt,omitempty"`
	LastError       string     `json:"lastError,omitempty"`
	Tags            []string   `json:"tags"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
}
type Evidence struct {
	Label string `json:"label"`
	Value string `json:"value"`
	Unit  string `json:"unit,omitempty"`
}
type Suggestion struct {
	Title  string `json:"title"`
	Detail string `json:"detail,omitempty"`
}
type Finding struct {
	ID                string       `json:"id"`
	RuleID            string       `json:"ruleId"`
	Fingerprint       string       `json:"fingerprint"`
	ServerID          string       `json:"serverId"`
	Database          string       `json:"database,omitempty"`
	Resource          string       `json:"resource,omitempty"`
	Severity          Severity     `json:"severity"`
	Category          string       `json:"category"`
	Title             string       `json:"title"`
	Summary           string       `json:"summary"`
	Cause             string       `json:"cause"`
	Impact            string       `json:"impact"`
	Evidence          []Evidence   `json:"evidence"`
	Suggestions       []Suggestion `json:"suggestions"`
	Confidence        Confidence   `json:"confidence"`
	Status            string       `json:"status"`
	StartedAt         time.Time    `json:"startedAt"`
	UpdatedAt         time.Time    `json:"updatedAt"`
	ResolvedAt        *time.Time   `json:"resolvedAt,omitempty"`
	Suppressed        bool         `json:"suppressed"`
	SuppressionReason string       `json:"suppressionReason,omitempty"`
	Maintenance       bool         `json:"maintenance"`
}
type MaintenanceWindow struct {
	ID          string    `json:"id"`
	Description string    `json:"description"`
	ServerID    string    `json:"serverId,omitempty"`
	ServerTag   string    `json:"serverTag,omitempty"`
	Category    string    `json:"category,omitempty"`
	RuleID      string    `json:"ruleId,omitempty"`
	StartsAt    time.Time `json:"startsAt"`
	EndsAt      time.Time `json:"endsAt"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
	State       string    `json:"state"`
}
type FindingSuppression struct {
	ID        string    `json:"id"`
	FindingID string    `json:"findingId,omitempty"`
	RuleID    string    `json:"ruleId,omitempty"`
	ServerID  string    `json:"serverId,omitempty"`
	ServerTag string    `json:"serverTag,omitempty"`
	Reason    string    `json:"reason"`
	ExpiresAt time.Time `json:"expiresAt"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	State     string    `json:"state"`
}
type ThresholdOverride struct {
	ID         string    `json:"id"`
	RuleID     string    `json:"ruleId"`
	ScopeType  string    `json:"scopeType"`
	ScopeValue string    `json:"scopeValue"`
	Reason     string    `json:"reason"`
	Value      float64   `json:"value"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}
type Metric struct {
	ServerID    string            `json:"serverId"`
	Database    string            `json:"database,omitempty"`
	Name        string            `json:"name"`
	Value       float64           `json:"value"`
	Labels      map[string]string `json:"labels,omitempty"`
	CollectedAt time.Time         `json:"collectedAt"`
}
type NotificationDestination struct {
	ID        string            `json:"id"`
	Provider  string            `json:"provider"`
	Name      string            `json:"name"`
	Config    map[string]string `json:"-"`
	Enabled   bool              `json:"enabled"`
	CreatedAt time.Time         `json:"createdAt"`
	UpdatedAt time.Time         `json:"updatedAt"`
}
type NotificationRoute struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Enabled         bool      `json:"enabled"`
	Priority        int       `json:"priority"`
	Severities      []string  `json:"severities"`
	Categories      []string  `json:"categories"`
	ServerIDs       []string  `json:"serverIds"`
	ServerTags      []string  `json:"serverTags"`
	Transitions     []string  `json:"transitions"`
	DestinationIDs  []string  `json:"destinationIds"`
	CooldownSeconds int       `json:"cooldownSeconds"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}
type NotificationDeliveryHistory struct {
	EventID         string     `json:"eventId"`
	DestinationID   string     `json:"destinationId"`
	DestinationName string     `json:"destinationName"`
	EventType       string     `json:"eventType"`
	FindingID       string     `json:"findingId"`
	FindingTitle    string     `json:"findingTitle"`
	ServerID        string     `json:"serverId"`
	ServerName      string     `json:"serverName"`
	Severity        string     `json:"severity"`
	Category        string     `json:"category"`
	Status          string     `json:"status"`
	LastError       string     `json:"lastError,omitempty"`
	Attempts        int        `json:"attempts"`
	CreatedAt       time.Time  `json:"createdAt"`
	LastAttemptAt   *time.Time `json:"lastAttemptAt,omitempty"`
	DeliveredAt     *time.Time `json:"deliveredAt,omitempty"`
	NextAttemptAt   *time.Time `json:"nextAttemptAt,omitempty"`
}
type FindingNotificationDelivery struct {
	EventID, DestinationID, EventType string
	Finding                           Finding
	Attempts                          int
}
type Snapshot struct {
	ServerID      string            `json:"serverId"`
	CollectedAt   time.Time         `json:"collectedAt"`
	Version       string            `json:"version"`
	UptimeSeconds float64           `json:"uptimeSeconds"`
	Connections   ConnectionStats   `json:"connections"`
	Databases     []DatabaseStat    `json:"databases"`
	Locks         []LockInfo        `json:"locks"`
	Queries       []QueryStat       `json:"queries"`
	Tables        []TableStat       `json:"tables"`
	Indexes       []IndexStat       `json:"indexes"`
	Settings      map[string]string `json:"settings"`
	Capabilities  map[string]bool   `json:"capabilities"`
	Replication   ReplicationStats  `json:"replication"`
	WAL           WALStats          `json:"wal"`
	ServerTags    []string          `json:"serverTags,omitempty"`
}

type ReplicationStats struct {
	InRecovery         bool                 `json:"inRecovery"`
	TimelineID         int                  `json:"timelineId"`
	RecoveryPaused     bool                 `json:"recoveryPaused"`
	ReplayDelaySeconds float64              `json:"replayDelaySeconds"`
	ReplayLSN          string               `json:"replayLsn,omitempty"`
	ReceiveLSN         string               `json:"receiveLsn,omitempty"`
	CollectedAt        time.Time            `json:"collectedAt"`
	Standbys           []ReplicationStandby `json:"standbys"`
	Receiver           *WALReceiver         `json:"receiver,omitempty"`
	Slots              []ReplicationSlot    `json:"slots"`
}
type ReplicationStandby struct {
	Application, ClientAddress, State, SyncState                               string
	WriteLagSeconds, FlushLagSeconds, ReplayLagSeconds                         float64
	SentLSN, WriteLSN, FlushLSN, ReplayLSN                                     string
	PendingSendBytes, PendingWriteBytes, PendingFlushBytes, PendingReplayBytes float64
	ReplyAgeSeconds                                                            float64
	PendingReplayGrowthBytesPerSecond                                          float64
}
type WALReceiver struct {
	Status, SenderHost, LatestEndLSN string
	LastMessageSeconds               float64
}
type ReplicationSlot struct {
	Name, Type, Database, WALStatus string
	Active                          bool
	RetainedBytes                   float64
	InactiveSeconds                 float64
	RetainedGrowthBytesPerSecond    float64
}
type WALStats struct {
	TimedCheckpoints, RequestedCheckpoints, WriteTimeMS, SyncTimeMS, BuffersWritten float64
	StatsReset                                                                      *time.Time
	WALRecords, WALFullPageImages, WALBytes, WALBuffersFull                         float64
	WALStatsReset                                                                   *time.Time
	ArchiveMode, CurrentLSN, LastArchivedWAL, LastFailedWAL                         string
	ArchiveConfigured                                                               bool
	ArchivedCount, FailedArchiveCount                                               float64
	LastArchivedAt, LastFailedAt                                                    *time.Time
	RestartpointsTimed, RestartpointsRequested, RestartpointsDone                   float64
	CollectedAt                                                                     time.Time
	GenerationBytesPerSecond, BufferFullEventsPerSecond                             float64
}
type ConnectionStats struct {
	Active, Idle, IdleInTransaction, Waiting, Total, Max                  int
	Utilization, LongestTransactionSeconds, LongestIdleTransactionSeconds float64
}
type DatabaseStat struct {
	Name                                                                                  string  `json:"name"`
	SizeBytes, Commits, Rollbacks, Deadlocks, TempFiles, TempBytes, BlocksRead, BlocksHit float64 `json:",omitempty"`
}
type LockInfo struct {
	BlockedPID, BlockingPID                           int
	DurationSeconds                                   float64
	Database, User, Application, Query, BlockingQuery string
}
type QueryStat struct {
	QueryID, Database, Query                                                                                                        string
	Calls, TotalExecMS, MeanExecMS, MinExecMS, MaxExecMS, Rows, SharedHit, SharedRead, TempRead, TempWritten, WALBytes, ImpactScore float64
}
type TableStat struct {
	Database, Schema, Table                                                                                                             string
	EstimatedRows, TotalSize, TableSize, IndexSize, LiveTuples, DeadTuples, SeqScans, IndexScans, Inserts, Updates, Deletes, HotUpdates float64
	LastVacuum, LastAutovacuum, LastAnalyze, LastAutoanalyze                                                                            *time.Time
	VacuumCount, AutovacuumCount                                                                                                        float64
	VacuumThreshold, VacuumProgress                                                                                                     float64
}
type IndexStat struct {
	Database, Schema, Table, Index, Definition string
	SizeBytes, Scans                           float64
	Unique, Primary                            bool
}
