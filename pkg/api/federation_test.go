package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHandleGrant(t *testing.T) {
	// Setup minimal server
	s := &Server{}

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

	if grantResp.RemainingGlobal != 10000 {
		t.Errorf("Expected remaining 10000, got %d", grantResp.RemainingGlobal)
	}

	if grantResp.ValidUntil.Before(time.Now()) {
		t.Error("Expected ValidUntil to be in the future")
	}
}

func TestHandleGrant_InvalidJSON(t *testing.T) {
	s := &Server{}
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
