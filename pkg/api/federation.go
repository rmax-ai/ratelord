package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// handleGrant processes federation grant requests.
// STUB IMPLEMENTATION: Mock response.
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

	// Mock Logic
	resp := GrantResponse{
		Granted:         req.Amount,
		ValidUntil:      time.Now().Add(1 * time.Minute),
		RemainingGlobal: 10000,
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
