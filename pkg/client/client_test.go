package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClient_Ask(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse Decision
		serverStatus   int
		intent         Intent
		wantAllowed    bool
		wantStatus     string
		wantErr        bool
		waitDuration   float64
	}{
		{
			name: "Approved",
			serverResponse: Decision{
				Status:   "approve",
				IntentID: "uuid-1",
				Allowed:  true,
			},
			serverStatus: http.StatusOK,
			intent: Intent{
				AgentID:    "agent-1",
				IdentityID: "user-1",
				WorkloadID: "work-1",
				ScopeID:    "scope-1",
			},
			wantAllowed: true,
			wantStatus:  "approve",
			wantErr:     false,
		},
		{
			name: "Denied",
			serverResponse: Decision{
				Status:  "deny_with_reason",
				Allowed: false,
				Reason:  "rate_limit_exceeded",
			},
			serverStatus: http.StatusOK,
			intent: Intent{
				AgentID:    "agent-1",
				IdentityID: "user-1",
				WorkloadID: "work-1",
				ScopeID:    "scope-1",
			},
			wantAllowed: false,
			wantStatus:  "deny_with_reason",
			wantErr:     false,
		},
		{
			name: "Wait",
			serverResponse: Decision{
				Status:   "approve_with_modifications",
				Allowed:  true,
				IntentID: "uuid-2",
				Modifications: Modifications{
					WaitSeconds: 0.1, // Small wait for test
				},
			},
			serverStatus: http.StatusOK,
			intent: Intent{
				AgentID:    "agent-1",
				IdentityID: "user-1",
				WorkloadID: "work-1",
				ScopeID:    "scope-1",
			},
			wantAllowed:  true,
			wantStatus:   "approve_with_modifications",
			wantErr:      false,
			waitDuration: 0.1,
		},
		{
			name:         "ServerError",
			serverStatus: http.StatusInternalServerError,
			intent: Intent{
				AgentID:    "agent-1",
				IdentityID: "user-1",
				WorkloadID: "work-1",
				ScopeID:    "scope-1",
			},
			wantAllowed: false,
			wantStatus:  "deny_with_reason",
			wantErr:     false, // Fail-closed means no error, just denied decision
		},
		{
			name:         "InvalidIntent",
			serverStatus: http.StatusBadRequest,
			intent: Intent{
				AgentID:    "agent-1",
				IdentityID: "user-1",
				WorkloadID: "work-1",
				ScopeID:    "scope-1",
			},
			wantAllowed: false,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/v1/intent" {
					t.Errorf("Expected path /v1/intent, got %s", r.URL.Path)
				}
				if r.Method != "POST" {
					t.Errorf("Expected method POST, got %s", r.Method)
				}

				w.WriteHeader(tt.serverStatus)
				if tt.serverStatus == http.StatusOK {
					json.NewEncoder(w).Encode(tt.serverResponse)
				}
			}))
			defer server.Close()

			c := NewClient(server.URL)
			start := time.Now()
			got, err := c.Ask(context.Background(), tt.intent)
			elapsed := time.Since(start)

			if (err != nil) != tt.wantErr {
				t.Errorf("Ask() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if got.Allowed != tt.wantAllowed {
					t.Errorf("Ask() allowed = %v, want %v", got.Allowed, tt.wantAllowed)
				}
				if got.Status != tt.wantStatus {
					t.Errorf("Ask() status = %v, want %v", got.Status, tt.wantStatus)
				}
			}

			if tt.waitDuration > 0 {
				if elapsed.Seconds() < tt.waitDuration {
					t.Errorf("Expected wait of at least %f seconds, got %f", tt.waitDuration, elapsed.Seconds())
				}
			}
		})
	}
}

func TestClient_Ping(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			t.Errorf("Expected path /health, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(Status{Status: "ok", Version: "v1.0.0"})
	}))
	defer server.Close()

	c := NewClient(server.URL)
	status, err := c.Ping(context.Background())
	if err != nil {
		t.Fatalf("Ping() error = %v", err)
	}

	if status.Status != "ok" {
		t.Errorf("Ping() status = %s, want ok", status.Status)
	}
}
