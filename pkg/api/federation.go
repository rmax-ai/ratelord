package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/rmax-ai/ratelord/pkg/store"
)

// handleGrant processes federation grant requests.
func (s *Server) handleGrant(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method_not_allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var req GrantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid_json_body"}`, http.StatusBadRequest)
		return
	}

	// Basic validation
	if req.FollowerID == "" || req.PoolID == "" || req.Amount <= 0 {
		http.Error(w, `{"error":"missing_or_invalid_fields"}`, http.StatusBadRequest)
		return
	}

	// Mock Logic for Grant Allocation (In a real system this would check usage store)
	// For now we just approve up to some limit or blindly.
	// TODO: Integrate with M32.1 usage store for atomic deduct.

	resp := GrantResponse{
		Granted:         req.Amount,
		ValidUntil:      time.Now().Add(1 * time.Minute),
		RemainingGlobal: 10000,
	}

	// Emit GrantIssued Event
	now := time.Now()
	payload, _ := json.Marshal(map[string]interface{}{
		"follower_id": req.FollowerID,
		"pool_id":     req.PoolID,
		"amount":      resp.Granted,
	})

	evt := store.Event{
		EventID:       store.EventID(fmt.Sprintf("grant_%d", now.UnixNano())),
		EventType:     store.EventTypeGrantIssued,
		SchemaVersion: 1,
		TsEvent:       now,
		TsIngest:      now,
		Source: store.EventSource{
			OriginKind: "daemon",
			OriginID:   "api",
			WriterID:   "ratelord-d",
		},
		Dimensions: store.EventDimensions{
			AgentID:    store.SentinelSystem,
			IdentityID: store.SentinelGlobal,
			WorkloadID: store.SentinelSystem,
			ScopeID:    store.SentinelGlobal,
		},
		Correlation: store.EventCorrelation{
			CorrelationID: fmt.Sprintf("grant_req_%s", req.FollowerID),
			CausationID:   store.SentinelUnknown,
		},
		Payload: payload,
	}

	if err := s.store.AppendEvent(r.Context(), &evt); err != nil {
		fmt.Printf(`{"level":"error","msg":"failed_to_append_grant_event","error":"%v"}`+"\n", err)
		// We proceed anyway as this is observability for now, but in strict mode we might fail
	} else {
		// Update topology immediately
		if s.cluster != nil {
			s.cluster.Apply(evt)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		fmt.Printf(`{"level":"error","msg":"failed_to_encode_grant_response","trace_id":"%s","error":"%v"}`+"\n", getTraceID(r.Context()), err)
	}

	// Log grant
	fmt.Printf(`{"level":"info","msg":"grant_issued","trace_id":"%s","follower_id":"%s","pool_id":"%s","granted":%d}`+"\n",
		getTraceID(r.Context()), req.FollowerID, req.PoolID, resp.Granted)
}

// handleClusterNodes returns the current topology.
func (s *Server) handleClusterNodes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method_not_allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	if s.cluster == nil {
		http.Error(w, `{"error":"cluster_topology_not_initialized"}`, http.StatusServiceUnavailable)
		return
	}

	// Get nodes seen in the last 5 minutes
	nodes := s.cluster.GetNodes(5 * time.Minute)

	// Determine leader
	leaderID := "unknown"
	if s.election != nil {
		id, ok, _ := s.election.GetLeader(r.Context())
		if ok {
			leaderID = id
		}
	} else {
		// Standalone mode
		leaderID = "self"
	}

	// Self ID? We don't have it easily available in Server struct unless we inject config.
	// For now "self" is fine or empty.

	resp := map[string]interface{}{
		"leader_id": leaderID,
		"nodes":     nodes,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		fmt.Printf(`{"level":"error","msg":"failed_to_encode_cluster_nodes","trace_id":"%s","error":"%v"}`+"\n", getTraceID(r.Context()), err)
	}
}
