package services

import "time"

type Node struct {
	Name                string    `json:"name"`
	Role                string    `json:"role"`
	Port                int       `json:"port"`
	Status              string    `json:"status"`
	KeyCount            int       `json:"key_count"`
	MemoryUsage         string    `json:"memory_usage"`
	LastHeartbeat       time.Time `json:"last_heartbeat"`
	LastSeen            string    `json:"last_seen"`
	LatencyMs           int64     `json:"latency_ms"`
	UptimeSeconds       int64     `json:"uptime_seconds"`
	UptimeStartedAt     time.Time `json:"-"`
	ConsecutiveFailures int       `json:"consecutive_failures"`
	ReplicationDelayMs  int64     `json:"replication_delay_ms"`
	LastReplicationAt   time.Time `json:"last_replication_at"`
	ReplicatedCommands  int       `json:"replicated_commands"`
	HealthScore         int       `json:"health_score"`
	HealthStatus        string    `json:"health_status"`
	SimulatedFailure    bool      `json:"simulated_failure"`
	InjectedLatencyMs   int64     `json:"injected_latency_ms"`
}

type DashboardOverview struct {
	TotalNodes         int    `json:"total_nodes"`
	NodeOnline         int    `json:"node_online"`
	NodeOffline        int    `json:"node_offline"`
	TotalKeys          int    `json:"total_keys"`
	ReplicatedCommands int    `json:"replicated_commands"`
	AverageLatencyMs   int64  `json:"average_latency_ms"`
	LastUpdated        string `json:"last_updated"`
	StorageMode        string `json:"storage_mode"`
}

type HeartbeatRow struct {
	Node                string `json:"node"`
	LastHeartbeat       string `json:"last_heartbeat"`
	LastSeen            string `json:"last_seen"`
	Status              string `json:"status"`
	LatencyMs           int64  `json:"latency_ms"`
	Uptime              string `json:"uptime"`
	ConsecutiveFailures int    `json:"consecutive_failures"`
	HealthScore         int    `json:"health_score"`
	HealthStatus        string `json:"health_status"`
}

type ClusterStats struct {
	TotalNodes             int     `json:"total_nodes"`
	OnlineNodes            int     `json:"online_nodes"`
	OfflineNodes           int     `json:"offline_nodes"`
	TotalKeys              int     `json:"total_keys"`
	ReplicatedCommands     int     `json:"replicated_commands"`
	FailedReplications     int     `json:"failed_replications"`
	ReplicationAttempts    int     `json:"replication_attempts"`
	ReplicationSuccessRate float64 `json:"replication_success_rate"`
	TotalHeartbeatChecks   int     `json:"total_heartbeat_checks"`
	AverageLatencyMs       int64   `json:"average_latency_ms"`
	MaxReplicationDelay    int64   `json:"max_replication_delay_ms"`
	LastReplicationAt      string  `json:"last_replication_at"`
	StorageMode            string  `json:"storage_mode"`
}

type ActivityLog struct {
	Timestamp string `json:"timestamp"`
	Type      string `json:"type"`
	Severity  string `json:"severity"`
	Message   string `json:"message"`
}

type ReplicationLog struct {
	Timestamp string `json:"timestamp"`
	Target    string `json:"target"`
	Action    string `json:"action"`
	Message   string `json:"message"`
	DelayMs   int64  `json:"delay_ms"`
	Success   bool   `json:"success"`
}

type ReplicationResult struct {
	Target  string `json:"target"`
	Status  string `json:"status"`
	DelayMs int64  `json:"delay_ms"`
	Message string `json:"message"`
}

type MetricPoint struct {
	Timestamp string `json:"timestamp"`
	Value     int64  `json:"value"`
}

type MetricsHistory struct {
	LatencyTrend          []MetricPoint `json:"latency_trend"`
	ReplicationDelayTrend []MetricPoint `json:"replication_delay_trend"`
}

type LatencyInjectionPayload struct {
	Milliseconds int64 `json:"milliseconds" binding:"required"`
}

type KeyValuePayload struct {
	Key   string `json:"key" binding:"required"`
	Value string `json:"value" binding:"required"`
}
