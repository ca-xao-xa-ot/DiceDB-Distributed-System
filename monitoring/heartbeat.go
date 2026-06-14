package monitoring

import (
	"sync"
	"time"
)

type NodeStatus struct {
	ID             string    `json:"id"`
	Status         string    `json:"status"`
	LastSeen       time.Time `json:"last_seen"`
	CommandsSynced int       `json:"commands_synced"`
}

var (
	Nodes = make(map[string]*NodeStatus)
	Mu    sync.RWMutex
)

func UpdateHeartbeat(nodeID string) {
	Mu.Lock()
	defer Mu.Unlock()

	node, exists := Nodes[nodeID]

	if !exists {
		node = &NodeStatus{
			ID: nodeID,
		}
		Nodes[nodeID] = node
	}

	node.Status = "online"
	node.LastSeen = time.Now()
}

func CheckNodeHealth(timeout time.Duration) {
	Mu.Lock()
	defer Mu.Unlock()

	for _, node := range Nodes {
		if time.Since(node.LastSeen) > timeout {
			node.Status = "offline"
		}
	}
}
