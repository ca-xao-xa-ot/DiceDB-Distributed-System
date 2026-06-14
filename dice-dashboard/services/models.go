package services

import "time"

type Node struct {
	Name          string    `json:"name"`
	Role          string    `json:"role"`
	Port          int       `json:"port"`
	Status        string    `json:"status"`
	KeyCount      int       `json:"key_count"`
	MemoryUsage   string    `json:"memory_usage"`
	LastHeartbeat time.Time `json:"last_heartbeat"`
}

type DashboardOverview struct {
	TotalNodes  int    `json:"total_nodes"`
	NodeOnline  int    `json:"node_online"`
	NodeOffline int    `json:"node_offline"`
	TotalKeys   int    `json:"total_keys"`
	LastUpdated string `json:"last_updated"`
	StorageMode string `json:"storage_mode"`
}

type HeartbeatRow struct {
	Node          string `json:"node"`
	LastHeartbeat string `json:"last_heartbeat"`
	Status        string `json:"status"`
}

type ActivityLog struct {
	Timestamp string `json:"timestamp"`
	Type      string `json:"type"`
	Message   string `json:"message"`
}

type ReplicationLog struct {
	Timestamp string `json:"timestamp"`
	Target    string `json:"target"`
	Action    string `json:"action"`
	Message   string `json:"message"`
}

type KeyValuePayload struct {
	Key   string `json:"key" binding:"required"`
	Value string `json:"value" binding:"required"`
}
