package monitoring

import (
	"sort"
	"sync"
	"time"
)

type NodeStatus struct {
	ID                  string    `json:"id"`
	Status              string    `json:"status"`
	LastSeen            time.Time `json:"last_seen"`
	CommandsSynced      int       `json:"commands_synced"`
	LatencyMs           int64     `json:"latency_ms"`
	UptimeSeconds       int64     `json:"uptime_seconds"`
	ConsecutiveFailures int       `json:"consecutive_failures"`
	ReplicationDelayMs  int64     `json:"replication_delay_ms"`
	HealthScore         int       `json:"health_score"`
	HealthStatus        string    `json:"health_status"`
	StartedAt           time.Time `json:"started_at"`
	Online              bool      `json:"online"`
}

type HeartbeatMonitor struct {
	mu       sync.RWMutex
	statuses map[string]NodeStatus
}

func NewHeartbeatMonitor() *HeartbeatMonitor {
	return &HeartbeatMonitor{statuses: make(map[string]NodeStatus)}
}

func (m *HeartbeatMonitor) UpdateHeartbeat(id string, commandsSynced int, latencyMs int64, replicationDelayMs int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	status := m.statuses[id]
	if status.StartedAt.IsZero() || !status.Online {
		status.StartedAt = now
	}
	status.ID = id
	status.Status = "Online"
	status.LastSeen = now
	status.CommandsSynced = commandsSynced
	status.LatencyMs = latencyMs
	status.ReplicationDelayMs = replicationDelayMs
	status.ConsecutiveFailures = 0
	status.Online = true
	status.UptimeSeconds = int64(now.Sub(status.StartedAt).Seconds())
	status.HealthScore, status.HealthStatus = calculateHealth(status.Online, status.LatencyMs, status.ConsecutiveFailures, status.ReplicationDelayMs)
	m.statuses[id] = status
}

func (m *HeartbeatMonitor) MarkFailure(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	status := m.statuses[id]
	status.ID = id
	if status.StartedAt.IsZero() {
		status.StartedAt = now
	}
	status.Status = "Offline"
	status.LastSeen = now
	status.Online = false
	status.ConsecutiveFailures++
	status.UptimeSeconds = 0
	status.HealthScore, status.HealthStatus = calculateHealth(false, status.LatencyMs, status.ConsecutiveFailures, status.ReplicationDelayMs)
	m.statuses[id] = status
}

func (m *HeartbeatMonitor) CheckNodeHealth(timeout time.Duration) []NodeStatus {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	items := make([]NodeStatus, 0, len(m.statuses))
	for id, status := range m.statuses {
		if now.Sub(status.LastSeen) > timeout {
			status.Status = "Offline"
			status.Online = false
			status.ConsecutiveFailures++
			status.UptimeSeconds = 0
			status.HealthScore, status.HealthStatus = calculateHealth(false, status.LatencyMs, status.ConsecutiveFailures, status.ReplicationDelayMs)
			m.statuses[id] = status
		}
		items = append(items, status)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items
}

func (m *HeartbeatMonitor) Snapshot() []NodeStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()
	items := make([]NodeStatus, 0, len(m.statuses))
	for _, status := range m.statuses {
		items = append(items, status)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items
}

func calculateHealth(online bool, latencyMs int64, failures int, replicationDelayMs int64) (int, string) {
	score := 100
	if !online {
		score -= 55
	}
	score -= int(latencyMs / 5)
	score -= failures * 8
	score -= int(replicationDelayMs / 10)
	if score < 0 {
		score = 0
	}
	switch {
	case score >= 80:
		return score, "Healthy"
	case score >= 55:
		return score, "Warning"
	default:
		return score, "Critical"
	}
}
