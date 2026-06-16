package services

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

type ClusterService struct {
	mu                    sync.RWMutex
	store                 KVStore
	nodes                 []Node
	activityLogs          []ActivityLog
	replicationLogs       []ReplicationLog
	heartbeatTimeout      time.Duration
	replicatedCommands    int
	failedReplications    int
	replicationAttempts   int
	heartbeatChecks       int
	heartbeatSamples      int64
	heartbeatLatencySum   int64
	latencyTrend          []MetricPoint
	replicationDelayTrend []MetricPoint
}

func NewClusterService(store KVStore, heartbeatTimeout time.Duration) *ClusterService {
	now := time.Now()
	s := &ClusterService{
		store:            store,
		heartbeatTimeout: heartbeatTimeout,
		nodes: []Node{
			{Name: "Master", Role: "master", Port: 7379, Status: "Online", MemoryUsage: "32 MB", UptimeStartedAt: now, LastHeartbeat: now, LastSeen: "just now", HealthScore: 100, HealthStatus: "Healthy"},
			{Name: "Replica1", Role: "replica", Port: 7380, Status: "Online", MemoryUsage: "28 MB", UptimeStartedAt: now, LastHeartbeat: now, LastSeen: "just now", HealthScore: 100, HealthStatus: "Healthy"},
			{Name: "Replica2", Role: "replica", Port: 7381, Status: "Online", MemoryUsage: "29 MB", UptimeStartedAt: now, LastHeartbeat: now, LastSeen: "just now", HealthScore: 100, HealthStatus: "Healthy"},
		},
	}
	s.refreshKeyCountLocked()
	s.appendActivityLocked("CLUSTER", "INFO", "Dashboard monitoring service initialized")
	s.recordHistoryLocked(0, 0)
	return s
}

func (s *ClusterService) StartHeartbeatSimulation() {
	ticker := time.NewTicker(3 * time.Second)
	go func() {
		defer ticker.Stop()
		for range ticker.C {
			s.tickHeartbeat()
		}
		}()
}

func (s *ClusterService) tickHeartbeat() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	var avgLatency int64
	var maxDelay int64

	for i := range s.nodes {
		node := &s.nodes[i]
		s.heartbeatChecks++

		if node.SimulatedFailure {
			node.Status = "Offline"
			node.ConsecutiveFailures++
			node.UptimeSeconds = 0
			node.LastSeen = fmt.Sprintf("%ds ago", int(time.Since(node.LastHeartbeat).Seconds()))
			node.HealthScore, node.HealthStatus = calculateHealth(node.Status, node.LatencyMs, node.ConsecutiveFailures, node.ReplicationDelayMs)
			s.appendActivityLocked("HEARTBEAT", "WARNING", fmt.Sprintf("%s heartbeat timeout detected", node.Name))
			continue
		}

		latency := int64(2 + i*3)
		if node.InjectedLatencyMs > 0 {
			latency += node.InjectedLatencyMs
		}
		node.Status = "Online"
		node.LastHeartbeat = now
		node.LastSeen = "just now"
		node.LatencyMs = latency
		node.UptimeSeconds = int64(now.Sub(node.UptimeStartedAt).Seconds())
		node.ConsecutiveFailures = 0
		node.HealthScore, node.HealthStatus = calculateHealth(node.Status, node.LatencyMs, node.ConsecutiveFailures, node.ReplicationDelayMs)
		s.heartbeatSamples++
		s.heartbeatLatencySum += latency
		avgLatency += latency
		if node.ReplicationDelayMs > maxDelay {
			maxDelay = node.ReplicationDelayMs
		}
	}

	if len(s.nodes) > 0 {
		avgLatency = avgLatency / int64(len(s.nodes))
	}
	s.recordHistoryLocked(avgLatency, maxDelay)
}

func (s *ClusterService) GetOverview() DashboardOverview {
	s.mu.RLock()
	defer s.mu.RUnlock()
	online, offline := s.onlineOfflineLocked()
	return DashboardOverview{
		TotalNodes:         len(s.nodes),
		NodeOnline:         online,
		NodeOffline:        offline,
		TotalKeys:          s.store.Count(),
		ReplicatedCommands: s.replicatedCommands,
		AverageLatencyMs:   s.averageHeartbeatLatencyLocked(),
		LastUpdated:        time.Now().Format("15:04:05"),
		StorageMode:        "In-memory demo mode",
	}
}

func (s *ClusterService) GetClusterStats() ClusterStats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	online, offline := s.onlineOfflineLocked()
	maxDelay := int64(0)
	lastReplication := "No replication yet"
	for _, node := range s.nodes {
		if node.ReplicationDelayMs > maxDelay {
			maxDelay = node.ReplicationDelayMs
		}
		if !node.LastReplicationAt.IsZero() && (lastReplication == "No replication yet" || node.LastReplicationAt.After(parseTimeFallback(lastReplication))) {
			lastReplication = node.LastReplicationAt.Format("15:04:05 02/01/2006")
		}
	}
	return ClusterStats{
		TotalNodes:             len(s.nodes),
		OnlineNodes:            online,
		OfflineNodes:           offline,
		TotalKeys:              s.store.Count(),
		ReplicatedCommands:     s.replicatedCommands,
		FailedReplications:     s.failedReplications,
		ReplicationAttempts:    s.replicationAttempts,
		ReplicationSuccessRate: s.replicationSuccessRateLocked(),
		TotalHeartbeatChecks:   s.heartbeatChecks,
		AverageLatencyMs:       s.averageHeartbeatLatencyLocked(),
		MaxReplicationDelay:    maxDelay,
		LastReplicationAt:      lastReplication,
		StorageMode:            "In-memory demo mode",
	}
}

func (s *ClusterService) GetMetricsHistory() MetricsHistory {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return MetricsHistory{
		LatencyTrend:          append([]MetricPoint(nil), s.latencyTrend...),
		ReplicationDelayTrend: append([]MetricPoint(nil), s.replicationDelayTrend...),
	}
}

func (s *ClusterService) GetNodes() []Node {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]Node, len(s.nodes))
	copy(result, s.nodes)
	return result
}

func (s *ClusterService) GetHeartbeatRows() []HeartbeatRow {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rows := make([]HeartbeatRow, 0, len(s.nodes))
	for _, node := range s.nodes {
		rows = append(rows, HeartbeatRow{
			Node:                node.Name,
			LastHeartbeat:       node.LastHeartbeat.Format("15:04:05 02/01"),
			LastSeen:            node.LastSeen,
			Status:              node.Status,
			LatencyMs:           node.LatencyMs,
			Uptime:              formatDuration(node.UptimeSeconds),
			ConsecutiveFailures: node.ConsecutiveFailures,
			HealthScore:         node.HealthScore,
			HealthStatus:        node.HealthStatus,
		})
	}
	return rows
}

func (s *ClusterService) GetActivityLogs(limit int) []ActivityLog {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return reverseLimit(s.activityLogs, limit)
}

func (s *ClusterService) GetReplicationLogs(limit int) []ReplicationLog {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return reverseLimit(s.replicationLogs, limit)
}

func (s *ClusterService) ClearLogs() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.activityLogs = nil
	s.replicationLogs = nil
	s.appendActivityLocked("SYSTEM", "INFO", "Logs cleared from dashboard")
}

func (s *ClusterService) Set(key, value string) ([]ReplicationResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.store.Set(key, value); err != nil {
		return nil, err
	}
	s.refreshKeyCountLocked()
	s.appendActivityLocked("SET", "SUCCESS", fmt.Sprintf("SET %s=%s", key, value))
	results := s.replicateCommandLocked("SET", fmt.Sprintf("%s=%s", key, value))
	return results, nil
}

func (s *ClusterService) Get(key string) (string, error) {
	return s.store.Get(key)
}

func (s *ClusterService) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.store.Delete(key); err != nil {
		return err
	}
	s.refreshKeyCountLocked()
	s.appendActivityLocked("DELETE", "SUCCESS", fmt.Sprintf("DELETE %s", key))
	s.replicateCommandLocked("DELETE", key)
	return nil
}

func (s *ClusterService) SimulateNodeFailure(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	node := s.findNodeLocked(name)
	if node == nil {
		return fmt.Errorf("node %s not found", name)
	}
	node.SimulatedFailure = true
	node.Status = "Offline"
	node.ConsecutiveFailures++
	node.UptimeSeconds = 0
	node.HealthScore, node.HealthStatus = calculateHealth(node.Status, node.LatencyMs, node.ConsecutiveFailures, node.ReplicationDelayMs)
	s.appendActivityLocked("FAULT", "ERROR", fmt.Sprintf("Simulated failure injected on %s", node.Name))
	return nil
}

func (s *ClusterService) RecoverNode(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	node := s.findNodeLocked(name)
	if node == nil {
		return fmt.Errorf("node %s not found", name)
	}
	node.SimulatedFailure = false
	node.Status = "Online"
	node.UptimeStartedAt = time.Now()
	node.UptimeSeconds = 0
	node.ConsecutiveFailures = 0
	node.LastHeartbeat = time.Now()
	node.LastSeen = "just now"
	node.HealthScore, node.HealthStatus = calculateHealth(node.Status, node.LatencyMs, node.ConsecutiveFailures, node.ReplicationDelayMs)
	s.appendActivityLocked("RECOVERY", "SUCCESS", fmt.Sprintf("Node %s recovered", node.Name))
	return nil
}

func (s *ClusterService) InjectLatency(name string, milliseconds int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	node := s.findNodeLocked(name)
	if node == nil {
		return fmt.Errorf("node %s not found", name)
	}
	node.InjectedLatencyMs = milliseconds
	node.LatencyMs = milliseconds + 5
	node.HealthScore, node.HealthStatus = calculateHealth(node.Status, node.LatencyMs, node.ConsecutiveFailures, node.ReplicationDelayMs)
	s.appendActivityLocked("LATENCY", "WARNING", fmt.Sprintf("Injected %dms latency on %s", milliseconds, node.Name))
	return nil
}

func (s *ClusterService) ClearLatency(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	node := s.findNodeLocked(name)
	if node == nil {
		return fmt.Errorf("node %s not found", name)
	}
	node.InjectedLatencyMs = 0
	node.LatencyMs = 0
	node.HealthScore, node.HealthStatus = calculateHealth(node.Status, node.LatencyMs, node.ConsecutiveFailures, node.ReplicationDelayMs)
	s.appendActivityLocked("LATENCY", "INFO", fmt.Sprintf("Cleared injected latency on %s", node.Name))
	return nil
}

func (s *ClusterService) replicateCommandLocked(action, payload string) []ReplicationResult {
	results := make([]ReplicationResult, 0, 2)
	var maxDelay int64
	for i := range s.nodes {
		node := &s.nodes[i]
		if node.Role != "replica" {
			continue
		}
		s.replicationAttempts++
		delay := int64(4 + i*2)
		if node.InjectedLatencyMs > 0 {
			delay += node.InjectedLatencyMs
		}
		if node.SimulatedFailure {
			s.failedReplications++
			node.ReplicationDelayMs = 0
			node.HealthScore, node.HealthStatus = calculateHealth("Offline", node.LatencyMs, node.ConsecutiveFailures+1, node.ReplicationDelayMs)
			message := fmt.Sprintf("%s failed on %s", action, node.Name)
			results = append(results, ReplicationResult{Target: node.Name, Status: "failed", DelayMs: 0, Message: message})
			s.replicationLogs = append(s.replicationLogs, ReplicationLog{Timestamp: time.Now().Format("15:04:05"), Target: node.Name, Action: action, Message: message, DelayMs: 0, Success: false})
			s.appendActivityLocked("REPLICATION", "ERROR", message)
			continue
		}
		node.ReplicatedCommands++
		node.ReplicationDelayMs = delay
		node.LastReplicationAt = time.Now()
		node.HealthScore, node.HealthStatus = calculateHealth(node.Status, node.LatencyMs, node.ConsecutiveFailures, node.ReplicationDelayMs)
		s.replicatedCommands++
		if delay > maxDelay {
			maxDelay = delay
		}
		message := fmt.Sprintf("%s replicated to %s (%s)", action, node.Name, payload)
		results = append(results, ReplicationResult{Target: node.Name, Status: "success", DelayMs: delay, Message: message})
		s.replicationLogs = append(s.replicationLogs, ReplicationLog{Timestamp: time.Now().Format("15:04:05"), Target: node.Name, Action: action, Message: message, DelayMs: delay, Success: true})
		s.appendActivityLocked("REPLICATION", "SUCCESS", message)
	}
	s.recordHistoryLocked(s.averageHeartbeatLatencyLocked(), maxDelay)
	return results
}

func (s *ClusterService) refreshKeyCountLocked() {
	count := s.store.Count()
	for i := range s.nodes {
		s.nodes[i].KeyCount = count
	}
}

func (s *ClusterService) onlineOfflineLocked() (int, int) {
	online, offline := 0, 0
	for _, node := range s.nodes {
		if strings.EqualFold(node.Status, "Online") {
			online++
		} else {
			offline++
		}
	}
	return online, offline
}

func (s *ClusterService) averageHeartbeatLatencyLocked() int64 {
	if s.heartbeatSamples == 0 {
		return 0
	}
	return s.heartbeatLatencySum / s.heartbeatSamples
}

func (s *ClusterService) replicationSuccessRateLocked() float64 {
	if s.replicationAttempts == 0 {
		return 100
	}
	successes := s.replicationAttempts - s.failedReplications
	return (float64(successes) / float64(s.replicationAttempts)) * 100
}

func (s *ClusterService) appendActivityLocked(kind, severity, message string) {
	s.activityLogs = append(s.activityLogs, ActivityLog{Timestamp: time.Now().Format("15:04:05"), Type: kind, Severity: severity, Message: message})
	if len(s.activityLogs) > 80 {
		s.activityLogs = s.activityLogs[len(s.activityLogs)-80:]
	}
}

func (s *ClusterService) recordHistoryLocked(avgLatency, maxDelay int64) {
	timestamp := time.Now().Format("15:04:05")
	s.latencyTrend = append(s.latencyTrend, MetricPoint{Timestamp: timestamp, Value: avgLatency})
	s.replicationDelayTrend = append(s.replicationDelayTrend, MetricPoint{Timestamp: timestamp, Value: maxDelay})
	if len(s.latencyTrend) > 20 {
		s.latencyTrend = s.latencyTrend[len(s.latencyTrend)-20:]
	}
	if len(s.replicationDelayTrend) > 20 {
		s.replicationDelayTrend = s.replicationDelayTrend[len(s.replicationDelayTrend)-20:]
	}
}

func (s *ClusterService) findNodeLocked(name string) *Node {
	for i := range s.nodes {
		if strings.EqualFold(s.nodes[i].Name, name) {
			return &s.nodes[i]
		}
	}
	return nil
}

func calculateHealth(status string, latencyMs int64, failures int, replicationDelayMs int64) (int, string) {
	score := 100
	if !strings.EqualFold(status, "Online") {
		score -= 55
	}
	score -= int(latencyMs / 5)
	score -= failures * 10
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

func formatDuration(seconds int64) string {
	h := seconds / 3600
	m := (seconds % 3600) / 60
	s := seconds % 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm %ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm %ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

func reverseLimit[T any](items []T, limit int) []T {
	if limit <= 0 || limit > len(items) {
		limit = len(items)
	}
	result := make([]T, 0, limit)
	for i := len(items) - 1; i >= 0 && len(result) < limit; i-- {
		result = append(result, items[i])
	}
	return result
}

func parseTimeFallback(value string) time.Time {
	t, err := time.Parse("15:04:05 02/01/2006", value)
	if err != nil {
		return time.Time{}
	}
	return t
}
