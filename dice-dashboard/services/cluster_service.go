package services

import (
	"context"
	"errors"
	"fmt"
	"net" // Đã thêm thư viện net để kết nối TCP thật
	"sync"
	"time"
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
			{Name: "Master", Role: "master", Port: 7379, Status: "Online", KeyCount: 0, MemoryUsage: "24 MB", LastHeartbeat: now},
			{Name: "Replica1", Role: "replica", Port: 7380, Status: "Online", KeyCount: 0, MemoryUsage: "18 MB", LastHeartbeat: now},
			{Name: "Replica2", Role: "replica", Port: 7381, Status: "Online", KeyCount: 0, MemoryUsage: "19 MB", LastHeartbeat: now},
		},
		activityLogs:    make([]ActivityLog, 0, 128),
		replicationLogs: make([]ReplicationLog, 0, 128),
	}

	s.seedData()
	return s
}

func (s *ClusterService) seedData() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = s.store.Set(ctx, "project:title", "Dice Distributed KV Store Dashboard")
	_ = s.store.Set(ctx, "project:course", "He Phan Tan")
	_ = s.store.Set(ctx, "cluster:mode", "Master-Replica Simulation")
	s.mu.Lock()
	s.refreshKeyCountLocked()
	s.mu.Unlock()
}

func (s *ClusterService) StartHeartbeatSimulation() {
	go func() {
		for {
			time.Sleep(3 * time.Second)
			s.tickHeartbeat()
		}
	}()
}

// BƯỚC 1: ĐÃ SỬA THÀNH PING TCP THẬT (CHUẨN RESP)
func (s *ClusterService) tickHeartbeat() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for i := range s.nodes {
		addr := fmt.Sprintf("127.0.0.1:%d", s.nodes[i].Port)
		
		// Thử kết nối TCP tới Node trong 1 giây
		conn, err := net.DialTimeout("tcp", addr, 1*time.Second)
		if err == nil {
			// Nếu kết nối thành công, gửi PING chuẩn RESP để không làm sập server
			_, _ = conn.Write([]byte("*1\r\n$4\r\nPING\r\n"))
			conn.Close()

			s.nodes[i].LastHeartbeat = now
			s.appendActivityLocked("HEARTBEAT", fmt.Sprintf("%s heartbeat received", s.nodes[i].Name))
		}
	}

	// Hàm này sẽ tự động chuyển node thành Offline nếu quá hạn timeout
	s.refreshNodeStatusesLocked()
}

// BƯỚC 2: ĐÃ SỬA ĐỂ TỰ ĐỘNG REPLICATE SANG REPLICA QUA TCP
func (s *ClusterService) Set(key, value string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// 1. Lưu vào CSDL nội bộ trước
	if err := s.store.Set(ctx, key, value); err != nil {
		return err
	}

	s.mu.Lock()
	s.refreshKeyCountLocked()
	s.appendActivityLocked("SET", fmt.Sprintf("SET %s=%s", key, value))
	
	// 2. Định dạng lệnh SET sang chuẩn RESP để đồng bộ
	respCmd := fmt.Sprintf("*3\r\n$3\r\nSET\r\n$%d\r\n%s\r\n$%d\r\n%s\r\n", len(key), key, len(value), value)

	// 3. Đẩy sang các máy Replica qua mạng
	for i := range s.nodes {
		if s.nodes[i].Role == "replica" {
			go func(node Node, cmd string) {
				addr := fmt.Sprintf("127.0.0.1:%d", node.Port)
				conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
				if err == nil {
					defer conn.Close()
					_, _ = conn.Write([]byte(cmd))

					s.mu.Lock()
					s.appendReplicationLocked(node.Name, "SET", fmt.Sprintf("Replicated to %s", node.Name))
					s.mu.Unlock()
				}
			}(s.nodes[i], respCmd)
		}
	}
	s.mu.Unlock()

	return nil
}

func (s *ClusterService) Get(key string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	value, err := s.store.Get(ctx, key)
	if err != nil {
		if errors.Is(err, ErrKeyNotFound) {
			s.mu.Lock()
			s.appendActivityLocked("GET", fmt.Sprintf("GET %s not found", key))
			s.mu.Unlock()
		}
		return "", err
	}

	s.mu.Lock()
	s.appendActivityLocked("GET", fmt.Sprintf("GET %s=%s", key, value))
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
	s.refreshKeyCountLocked()
	s.appendActivityLocked("DELETE", fmt.Sprintf("DELETE %s", key))

	// Bổ sung đồng bộ lệnh DEL sang các node Replica cho đồng bộ toàn diện
	respCmdDEL := fmt.Sprintf("*2\r\n$3\r\nDEL\r\n$%d\r\n%s\r\n", len(key), key)
	
	for i := range s.nodes {
		if s.nodes[i].Role == "replica" {
			go func(node Node, cmd string) {
				addr := fmt.Sprintf("127.0.0.1:%d", node.Port)
				conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
				if err == nil {
					defer conn.Close()
					_, _ = conn.Write([]byte(cmd))

					s.mu.Lock()
					s.appendReplicationLocked(node.Name, "DELETE", fmt.Sprintf("Delete replicated to %s", node.Name))
					s.mu.Unlock()
				}
			}(s.nodes[i], respCmdDEL)
		}
	}
	s.mu.Unlock()

	return nil
}

func (s *ClusterService) GetOverview() DashboardOverview {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.refreshNodeStatusesLocked()

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
	s.mu.Lock()
	defer s.mu.Unlock()
	s.refreshNodeStatusesLocked()
	result := make([]Node, len(s.nodes))
	copy(result, s.nodes)
	return result
}

func (s *ClusterService) GetHeartbeatRows() []HeartbeatRow {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.refreshNodeStatusesLocked()
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

func (s *ClusterService) ClearLogs() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.activityLogs = make([]ActivityLog, 0, 128)
	s.replicationLogs = make([]ReplicationLog, 0, 128)
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

func (s *ClusterService) refreshNodeStatusesLocked() {
	now := time.Now()
	for i := range s.nodes {
		nextStatus := "Online"
		if now.Sub(s.nodes[i].LastHeartbeat) > s.heartbeatTimeout {
			nextStatus = "Offline"
		}

		if s.nodes[i].Status != nextStatus && nextStatus == "Offline" {
			s.appendActivityLocked("TIMEOUT", fmt.Sprintf("%s timeout", s.nodes[i].Name))
		}

		s.nodes[i].Status = nextStatus
	}
}

func (s *ClusterService) appendActivityLocked(logType, message string) {
	s.activityLogs = append(s.activityLogs, ActivityLog{
		Timestamp: time.Now().Format("15:04:05"),
		Type:      logType,
		Message:   message,
	})
	if len(s.activityLogs) > 200 {
		s.activityLogs = s.activityLogs[len(s.activityLogs)-200:]
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
// trigger linter run