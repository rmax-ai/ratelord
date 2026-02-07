package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/rmax-ai/ratelord/pkg/engine"
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

	// M38.1: Use Policy Engine to validate grant
	// We treat the grant request as an Intent
	intent := engine.Intent{
		IntentID:     "grant_" + fmt.Sprintf("%d", time.Now().UnixNano()),
		IdentityID:   req.FollowerID,
		WorkloadID:   "federation_sync",
		ScopeID:      "global", // Default scope for federation grants
		ProviderID:   req.ProviderID,
		PoolID:       req.PoolID,
		ExpectedCost: req.Amount,
	}

	result := s.policy.Evaluate(intent)

	var granted int64
	var validUntil time.Time

	if result.Decision == engine.DecisionApprove || result.Decision == engine.DecisionApproveWithModifications {
		granted = req.Amount
		validUntil = time.Now().Add(1 * time.Minute) // Default TTL

		// Apply modifications if any
		if result.Decision == engine.DecisionApproveWithModifications {
			// For grants, we might support "shaping" by reducing the amount
			// But the current Policy Engine returns "wait_seconds".
			// If wait is requested, we effectively deny this grant (return 0).
			if _, ok := result.Modifications["wait_seconds"]; ok {
				granted = 0
			}
		}
	} else {
		// Denied
		granted = 0
		validUntil = time.Now() // Immediate expiry
	}

	remaining := int64(0)
	if poolState, ok := s.usage.GetPoolState(req.ProviderID, req.PoolID); ok {
		remaining = poolState.Remaining
	}

	resp := GrantResponse{
		Granted:         granted,
		ValidUntil:      validUntil,
		RemainingGlobal: remaining,
	}

	// Emit GrantIssued Event only if granted > 0
	if granted > 0 {
		now := time.Now()
		payload, _ := json.Marshal(map[string]interface{}{
			"follower_id": req.FollowerID,
			"provider_id": req.ProviderID, // Added M38.1
			"pool_id":     req.PoolID,
			"amount":      granted,
			"metadata":    req.Metadata,
		})

		evt := store.Event{
			EventID:       store.EventID(fmt.Sprintf("grant_%d", now.UnixNano())),
			EventType:     store.EventTypeGrantIssued,
			SchemaVersion: 1,
			TsEvent:       now,
			TsIngest:      now,
			Epoch:         s.getEpoch(),
			Source: store.EventSource{
				OriginKind: "daemon",
				OriginID:   "api",
				WriterID:   "ratelord-d",
			},
			Dimensions: store.EventDimensions{
				AgentID:    store.SentinelSystem,
				IdentityID: req.FollowerID,
				WorkloadID: "federation",
				ScopeID:    "global",
			},
			Correlation: store.EventCorrelation{
				CorrelationID: fmt.Sprintf("grant_req_%s", req.FollowerID),
				CausationID:   store.SentinelUnknown,
			},
			Payload: payload,
		}

		if err := s.store.AppendEvent(r.Context(), &evt); err != nil {
			fmt.Printf(`{"level":"error","msg":"failed_to_append_grant_event","error":"%v"}`+"\n", err)
			// Proceed but logging error
		} else {
			// Update topology
			if s.cluster != nil {
				s.cluster.Apply(evt)
			}
			// Update local usage immediately so subsequent grants see the usage
			s.usage.Apply(evt)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		fmt.Printf(`{"level":"error","msg":"failed_to_encode_grant_response","trace_id":"%s","error":"%v"}`+"\n", getTraceID(r.Context()), err)
	}

	// Log grant
	fmt.Printf(`{"level":"info","msg":"grant_issued","trace_id":"%s","follower_id":"%s","provider_id":"%s","pool_id":"%s","granted":%d}`+"\n",
		getTraceID(r.Context()), req.FollowerID, req.ProviderID, req.PoolID, resp.Granted)
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
