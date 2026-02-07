package engine

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/rmax-ai/ratelord/pkg/store"
)

// ClusterNode represents a known node in the cluster
type ClusterNode struct {
	NodeID   string                 `json:"node_id"`
	LastSeen time.Time              `json:"last_seen"`
	Status   string                 `json:"status"` // "active" | "offline"
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ClusterTopology maintains the state of the cluster based on grant heartbeats
type ClusterTopology struct {
	mu          sync.RWMutex
	nodes       map[string]*ClusterNode
	lastEventID string
}

// NewClusterTopology creates a new empty topology
func NewClusterTopology() *ClusterTopology {
	return &ClusterTopology{
		nodes: make(map[string]*ClusterNode),
	}
}

// Apply updates the topology with a single event
func (c *ClusterTopology) Apply(event store.Event) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.lastEventID = string(event.EventID)

	if event.EventType != store.EventTypeGrantIssued {
		return nil
	}

	var payload struct {
		FollowerID string                 `json:"follower_id"`
		PoolID     string                 `json:"pool_id"`
		Amount     int                    `json:"amount"`
		Metadata   map[string]interface{} `json:"metadata,omitempty"`
	}

	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload for event %s: %w", event.EventID, err)
	}

	nodeID := payload.FollowerID
	if nodeID == "" {
		return nil
	}

	node, exists := c.nodes[nodeID]
	if !exists {
		node = &ClusterNode{
			NodeID: nodeID,
		}
		c.nodes[nodeID] = node
	}
	node.LastSeen = event.TsEvent
	node.Status = "active" // Assumed active if we see an event
	if payload.Metadata != nil {
		node.Metadata = payload.Metadata
	}

	return nil
}

// Replay rebuilds the topology from a slice of events
func (c *ClusterTopology) Replay(events []*store.Event) error {
	for _, event := range events {
		if event == nil {
			continue
		}
		if err := c.Apply(*event); err != nil {
			return err
		}
	}
	return nil
}

// GetNodes returns all known nodes, updating status based on TTL
func (c *ClusterTopology) GetNodes(ttl time.Duration) []ClusterNode {
	c.mu.RLock()
	defer c.mu.RUnlock()

	list := make([]ClusterNode, 0, len(c.nodes))
	now := time.Now()

	for _, node := range c.nodes {
		// Calculate status on read
		status := "active"
		if now.Sub(node.LastSeen) > ttl {
			status = "offline"
		}

		list = append(list, ClusterNode{
			NodeID:   node.NodeID,
			LastSeen: node.LastSeen,
			Status:   status,
			Metadata: node.Metadata,
		})
	}
	return list
}
