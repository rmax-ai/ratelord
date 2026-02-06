package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rmax-ai/ratelord/pkg/engine"
	"github.com/rmax-ai/ratelord/pkg/store"
)

func setupTestServer(t *testing.T) (*Server, func()) {
	// Temp Dir
	tmpDir, err := os.MkdirTemp("", "ratelord-api-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// Store
	dbPath := filepath.Join(tmpDir, "test.db")
	st, err := store.NewStore(dbPath)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to create store: %v", err)
	}

	// Cluster
	cluster := engine.NewClusterTopology()

	s := &Server{
		store:   st,
		cluster: cluster,
	}

	cleanup := func() {
		st.Close()
		os.RemoveAll(tmpDir)
	}

	return s, cleanup
}

func TestHandleGrant(t *testing.T) {
	s, cleanup := setupTestServer(t)
	defer cleanup()

	// Setup router
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/federation/grant", s.handleGrant)

	// Create test server
	ts := httptest.NewServer(mux)
	defer ts.Close()

	// Create Request
	reqBody := GrantRequest{
		FollowerID: "follower-123",
		PoolID:     "pool-abc",
		Amount:     500,
	}
	bodyBytes, _ := json.Marshal(reqBody)

	// Send Request
	resp, err := http.Post(ts.URL+"/v1/federation/grant", "application/json", bytes.NewBuffer(bodyBytes))
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Assert Status
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Assert Response
	var grantResp GrantResponse
	if err := json.NewDecoder(resp.Body).Decode(&grantResp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if grantResp.Granted != 500 {
		t.Errorf("Expected granted 500, got %d", grantResp.Granted)
	}

	// Check Side Effects: Cluster Topology
	// Allow async update to propagate if needed (it's synchronous in code but good practice)
	nodes := s.cluster.GetNodes(1 * time.Minute)
	if len(nodes) != 1 {
		t.Fatalf("Expected 1 node in topology, got %d", len(nodes))
	}
	if nodes[0].NodeID != "follower-123" {
		t.Errorf("Expected node follower-123, got %s", nodes[0].NodeID)
	}
	if nodes[0].Status != "active" {
		t.Errorf("Expected node status active, got %s", nodes[0].Status)
	}
}

func TestHandleGrant_InvalidJSON(t *testing.T) {
	s := &Server{} // Doesn't need store for this test as it fails before
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/federation/grant", s.handleGrant)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/v1/federation/grant", "application/json", bytes.NewBuffer([]byte("invalid json")))
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid JSON, got %d", resp.StatusCode)
	}
}

func TestHandleClusterNodes(t *testing.T) {
	s, cleanup := setupTestServer(t)
	defer cleanup()

	// Pre-populate topology via Grant
	reqBody := GrantRequest{
		FollowerID: "node-1",
		PoolID:     "pool-1",
		Amount:     100,
	}
	// Call handleGrant directly or via mock request to populate store/topology
	reqBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/v1/federation/grant", bytes.NewBuffer(reqBytes))
	w := httptest.NewRecorder()
	s.handleGrant(w, req)

	// Now test handleClusterNodes
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/cluster/nodes", s.handleClusterNodes)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/v1/cluster/nodes")
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var clusterResp struct {
		LeaderID string `json:"leader_id"`
		Nodes    []struct {
			NodeID string `json:"node_id"`
			Status string `json:"status"`
		} `json:"nodes"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&clusterResp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if clusterResp.LeaderID != "self" {
		t.Errorf("Expected leader self, got %s", clusterResp.LeaderID)
	}

	if len(clusterResp.Nodes) != 1 {
		t.Fatalf("Expected 1 node, got %d", len(clusterResp.Nodes))
	}
	if clusterResp.Nodes[0].NodeID != "node-1" {
		t.Errorf("Expected node-1, got %s", clusterResp.Nodes[0].NodeID)
	}
}
