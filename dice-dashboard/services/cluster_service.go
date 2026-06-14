package services

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type ClusterService struct {
	mu               sync.RWMutex
	store            KVStore
	nodes            []Node
	activityLogs     []ActivityLog
	replicationLogs  []ReplicationLog
	heartbeatTimeout time.Duration
}

func NewClusterService(store KVStore, timeout time.Duration) *ClusterService {
	now := time.Now()
	s := &ClusterService{
		store:            store,
		heartbeatTimeout: timeout,
		nodes: []Node{
			{Name: "Master Node", Role: "master", Port: 7379, Status: "Online", KeyCount: 0, MemoryUsage: "24 MB", LastHeartbeat: now},
			{Name: "Replica Node 1", Role: "replica", Port: 7380, Status: "Online", KeyCount: 0, MemoryUsage: "18 MB", LastHeartbeat: now},
			{Name: "Replica Node 2", Role: "replica", Port: 7381, Status: "Online", KeyCount: 0, MemoryUsage: "19 MB", LastHeartbeat: now},
		},
		activityLogs:    make([]ActivityLog, 0, 64),
		replicationLogs: make([]ReplicationLog, 0, 64),
	}

	s.seedData()
	return s
}

func (s *ClusterService) seedData() {
	_ = s.Set("project:title", "Dice Distributed KV Store Dashboard")
	_ = s.Set("project:course", "He Phan Tan")
	_ = s.Set("cluster:mode", "Master-Replica Simulation")
}

func (s *ClusterService) StartHeartbeatSimulation() {
	go func() {
		for {
			time.Sleep(3 * time.Second)
			s.tickHeartbeat()
		}
	}()
}

func (s *ClusterService) tickHeartbeat() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for i := range s.nodes {
		shouldBeat := true
		if i > 0 {
			shouldBeat = rand.Intn(100) > 18
		}

		if shouldBeat {
			s.nodes[i].LastHeartbeat = now
			s.nodes[i].Status = "Online"
			s.appendActivityLocked("HEARTBEAT", fmt.Sprintf("%s gửi heartbeat lúc %s", s.nodes[i].Name, now.Format("15:04:05")))
		}

		if now.Sub(s.nodes[i].LastHeartbeat) > s.heartbeatTimeout {
			s.nodes[i].Status = "Offline"
		}
	}
}

func (s *ClusterService) Set(key, value string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := s.store.Set(ctx, key, value); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.refreshKeyCountLocked()
	s.appendActivityLocked("SET", fmt.Sprintf("SET key '%s' = '%s'", key, value))
	s.appendReplicationLocked("Replica Node 1", "SET", fmt.Sprintf("Replicated to Replica1: key '%s'", key))
	s.appendReplicationLocked("Replica Node 2", "SET", fmt.Sprintf("Replicated to Replica2: key '%s'", key))
	return nil
}

func (s *ClusterService) Get(key string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	value, err := s.store.Get(ctx, key)
	if err != nil {
		if err == redis.Nil {
			s.mu.Lock()
			s.appendActivityLocked("GET", fmt.Sprintf("GET key '%s' không tồn tại", key))
			s.mu.Unlock()
		}
		return "", err
	}

	s.mu.Lock()
	s.appendActivityLocked("GET", fmt.Sprintf("GET key '%s' -> '%s'", key, value))
	s.mu.Unlock()
	return value, nil
}

func (s *ClusterService) Delete(key string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := s.store.Delete(ctx, key); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.refreshKeyCountLocked()
	s.appendActivityLocked("DELETE", fmt.Sprintf("DELETE key '%s'", key))
	s.appendReplicationLocked("Replica Node 1", "DELETE", fmt.Sprintf("Replicated to Replica1: deleted key '%s'", key))
	s.appendReplicationLocked("Replica Node 2", "DELETE", fmt.Sprintf("Replicated to Replica2: deleted key '%s'", key))
	return nil
}

func (s *ClusterService) GetOverview() DashboardOverview {
	s.mu.RLock()
	defer s.mu.RUnlock()

	online := 0
	for _, node := range s.nodes {
		if node.Status == "Online" {
			online++
		}
	}

	count := 0
	if len(s.nodes) > 0 {
		count = s.nodes[0].KeyCount
	}

	return DashboardOverview{
		TotalNodes:  len(s.nodes),
		NodeOnline:  online,
		NodeOffline: len(s.nodes) - online,
		TotalKeys:   count,
		LastUpdated: time.Now().Format("02/01/2006 15:04:05"),
		StorageMode: s.store.Mode(),
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
			Node:          node.Name,
			LastHeartbeat: node.LastHeartbeat.Format("15:04:05 - 02/01/2006"),
			Status:        node.Status,
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
	return reverseLimitReplication(s.replicationLogs, limit)
}

func (s *ClusterService) refreshKeyCountLocked() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	count, err := s.store.Count(ctx)
	if err != nil {
		return
	}

	for i := range s.nodes {
		s.nodes[i].KeyCount = count
		base := 18 + count/2 + i
		s.nodes[i].MemoryUsage = fmt.Sprintf("%d MB", base)
	}
}

func (s *ClusterService) appendActivityLocked(logType, message string) {
	s.activityLogs = append(s.activityLogs, ActivityLog{
		Timestamp: time.Now().Format("15:04:05"),
		Type:      logType,
		Message:   message,
	})
	if len(s.activityLogs) > 100 {
		s.activityLogs = s.activityLogs[len(s.activityLogs)-100:]
	}
}

func (s *ClusterService) appendReplicationLocked(target, action, message string) {
	entry := ReplicationLog{
		Timestamp: time.Now().Format("15:04:05"),
		Target:    target,
		Action:    action,
		Message:   message,
	}
	s.replicationLogs = append(s.replicationLogs, entry)
	if len(s.replicationLogs) > 100 {
		s.replicationLogs = s.replicationLogs[len(s.replicationLogs)-100:]
	}
	s.appendActivityLocked("REPLICATION", message)
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

func reverseLimitReplication(items []ReplicationLog, limit int) []ReplicationLog {
	if limit <= 0 || limit > len(items) {
		limit = len(items)
	}
	result := make([]ReplicationLog, 0, limit)
	for i := len(items) - 1; i >= 0 && len(result) < limit; i-- {
		result = append(result, items[i])
	}
	return result
}
