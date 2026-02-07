package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/rmax-ai/ratelord/pkg/engine"
	"github.com/rmax-ai/ratelord/pkg/graph"
	"github.com/rmax-ai/ratelord/pkg/store"
)

// Mock implementations for testing

type MockStore struct {
	*store.Store
	events     []store.Event
	usageStats []store.UsageStat
}

func (m *MockStore) AppendEvent(ctx context.Context, event *store.Event) error {
	m.events = append(m.events, *event)
	return nil
}

func (m *MockStore) ReadRecentEvents(ctx context.Context, limit int) ([]*store.Event, error) {
	if limit > len(m.events) {
		limit = len(m.events)
	}
	// Return pointers
	res := make([]*store.Event, limit)
	start := len(m.events) - limit
	for i := 0; i < limit; i++ {
		res[i] = &m.events[start+i]
	}
	return res, nil
}

func (m *MockStore) QueryEvents(ctx context.Context, filter store.EventFilter) ([]*store.Event, error) {
	// Return some mock events
	return []*store.Event{
		{
			EventID:   "event1",
			EventType: store.EventTypeIntentDecided,
			TsEvent:   time.Now(),
		},
		{
			EventID:   "event2",
			EventType: store.EventTypeIdentityRegistered,
			TsEvent:   time.Now(),
		},
	}, nil
}

func (m *MockStore) RegisterWebhook(ctx context.Context, cfg *store.WebhookConfig) error {
	return nil
}

func (m *MockStore) ListWebhooks(ctx context.Context) ([]*store.WebhookConfig, error) {
	return nil, nil
}

func (m *MockStore) DeleteWebhook(ctx context.Context, webhookID string) error {
	return nil
}

func (m *MockStore) DeleteIdentityData(ctx context.Context, id string) error {
	return nil
}

func (m *MockStore) PruneEvents(ctx context.Context, retention time.Duration, includeType string, excludeTypes []string) (int64, error) {
	return 5, nil
}

func (m *MockStore) GetUsageStats(ctx context.Context, filter store.UsageFilter) ([]store.UsageStat, error) {
	return m.usageStats, nil
}

type MockPolicyEngine struct {
	EvaluateFunc func(intent engine.Intent) engine.PolicyEvaluationResult
}

func (m *MockPolicyEngine) Evaluate(intent engine.Intent) engine.PolicyEvaluationResult {
	if m.EvaluateFunc != nil {
		return m.EvaluateFunc(intent)
	}
	return engine.PolicyEvaluationResult{
		Decision:      "approve",
		Reason:        "test",
		Modifications: map[string]interface{}{},
		Warnings:      []string{},
		Trace:         []engine.RuleTrace{},
	}
}

type MockIdentityProjection struct {
	*engine.IdentityProjection
	identities []engine.Identity
	tokenMap   map[string]string
}

func (m *MockIdentityProjection) GetAll() []engine.Identity {
	return m.identities
}

func (m *MockIdentityProjection) GetByTokenHash(hash string) (engine.Identity, bool) {
	if id, ok := m.tokenMap[hash]; ok {
		for _, identity := range m.identities {
			if identity.ID == id {
				return identity, true
			}
		}
	}
	return engine.Identity{}, false
}

func (m *MockIdentityProjection) Apply(event store.Event) error {
	return nil
}

type MockUsageProjection struct {
	*engine.UsageProjection
	poolStates map[string]map[string]engine.PoolState
}

func (m *MockUsageProjection) GetPoolState(providerID, poolID string) (engine.PoolState, bool) {
	if pools, ok := m.poolStates[providerID]; ok {
		if state, ok := pools[poolID]; ok {
			return state, true
		}
	}
	return engine.PoolState{}, false
}

func (m *MockUsageProjection) Apply(event store.Event) error {
	return nil
}

type MockGraph struct {
	*graph.Projection
}

func (m *MockGraph) GetGraph() *graph.Graph {
	return &graph.Graph{}
}

type MockElectionManager struct {
	IsLeaderFunc  func() bool
	GetLeaderFunc func(ctx context.Context) (string, bool, error)
	GetEpochFunc  func() int64
}

func (m *MockElectionManager) IsLeader() bool {
	if m.IsLeaderFunc != nil {
		return m.IsLeaderFunc()
	}
	return true // Default leader
}

func (m *MockElectionManager) GetLeader(ctx context.Context) (string, bool, error) {
	if m.GetLeaderFunc != nil {
		return m.GetLeaderFunc(ctx)
	}
	return "http://localhost:8090", true, nil
}

func (m *MockElectionManager) GetEpoch() int64 {
	if m.GetEpochFunc != nil {
		return m.GetEpochFunc()
	}
	return 1
}

// Helper to create server with mocks
func createServerWithMocks(store StoreInterface, identities IdentityProjectionInterface, usage UsageProjectionInterface, policy PolicyEngineInterface, graph GraphProjectionInterface, election ElectionManagerInterface) *Server {
	server := &Server{}
	server.store = store
	server.identities = identities
	server.usage = usage
	server.policy = policy
	server.graph = graph
	server.election = election
	return server
}

func TestSecureHeaders(t *testing.T) {
	// Create a handler that just returns 200 OK
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Wrap it with our middleware
	secureHandler := withSecureHeaders(handler)

	// Create a request
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	// Serve
	secureHandler.ServeHTTP(w, req)

	// Check headers
	expectedHeaders := map[string]string{
		"Content-Security-Policy":   "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data:;",
		"Strict-Transport-Security": "max-age=63072000; includeSubDomains",
		"X-Content-Type-Options":    "nosniff",
		"X-Frame-Options":           "DENY",
		"Referrer-Policy":           "no-referrer",
		"X-XSS-Protection":          "1; mode=block",
	}

	for key, expected := range expectedHeaders {
		got := w.Header().Get(key)
		if got != expected {
			t.Errorf("Header %s: expected %q, got %q", key, expected, got)
		}
	}
}

func TestHandleHealth(t *testing.T) {
	req := httptest.NewRequest("GET", "/v1/health", nil)
	w := httptest.NewRecorder()

	handleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp["status"] != "ok" {
		t.Errorf("Expected status 'ok', got %q", resp["status"])
	}
}

func TestHandleIntent(t *testing.T) {
	mockStore := &MockStore{}
	mockPolicy := &MockPolicyEngine{
		EvaluateFunc: func(intent engine.Intent) engine.PolicyEvaluationResult {
			return engine.PolicyEvaluationResult{
				Decision:      "approve",
				Reason:        "test",
				Modifications: map[string]interface{}{},
				Warnings:      []string{},
				Trace:         []engine.RuleTrace{},
			}
		},
	}
	mockUsage := &MockUsageProjection{
		poolStates: map[string]map[string]engine.PoolState{
			"mock-provider-1": {
				"default": {Used: 0, Remaining: 100},
			},
		},
	}
	mockIdentities := &MockIdentityProjection{}
	mockGraph := &MockGraph{}

	server := createServerWithMocks(mockStore, mockIdentities, mockUsage, mockPolicy, mockGraph, nil)

	reqBody := IntentRequest{
		AgentID:    "agent1",
		IdentityID: "identity1",
		ScopeID:    "scope1",
		WorkloadID: "workload1",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/v1/intent", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleIntent(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp DecisionResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Decision != "approve" {
		t.Errorf("Expected decision 'approve', got %q", resp.Decision)
	}
}

func TestHandleIdentity_Register(t *testing.T) {
	mockStore := &MockStore{}
	mockIdentities := &MockIdentityProjection{}
	mockUsage := &MockUsageProjection{}
	mockPolicy := &MockPolicyEngine{}
	mockGraph := &MockGraph{}

	server := createServerWithMocks(mockStore, mockIdentities, mockUsage, mockPolicy, mockGraph, nil)

	reqBody := IdentityRegistration{
		IdentityID: "test-id",
		Kind:       "agent",
		Metadata:   map[string]interface{}{"key": "value"},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/v1/identities", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleIdentities(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp IdentityResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.IdentityID != "test-id" {
		t.Errorf("Expected identity_id 'test-id', got %q", resp.IdentityID)
	}

	if resp.Status != "registered" {
		t.Errorf("Expected status 'registered', got %q", resp.Status)
	}

	// Check that event was appended
	if len(mockStore.events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(mockStore.events))
	}
}

func TestHandleIdentity_Delete(t *testing.T) {
	mockStore := &MockStore{}
	mockIdentities := &MockIdentityProjection{}
	mockUsage := &MockUsageProjection{}
	mockPolicy := &MockPolicyEngine{}
	mockGraph := &MockGraph{}

	server := createServerWithMocks(mockStore, mockIdentities, mockUsage, mockPolicy, mockGraph, nil)

	req := httptest.NewRequest("DELETE", "/v1/identities/test-id", nil)
	w := httptest.NewRecorder()

	server.handleIdentities(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d", w.Code)
	}

	// Check event appended
	if len(mockStore.events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(mockStore.events))
	}
}

func TestHandleTrends(t *testing.T) {
	mockStore := &MockStore{
		usageStats: []store.UsageStat{
			{
				BucketTs:   time.Now(),
				ProviderID: "prov1",
				PoolID:     "pool1",
				IdentityID: "id1",
				ScopeID:    "scope1",
				TotalUsage: 10,
				MinUsage:   1,
				MaxUsage:   5,
				EventCount: 2,
			},
		},
	}
	mockIdentities := &MockIdentityProjection{}
	mockUsage := &MockUsageProjection{}
	mockPolicy := &MockPolicyEngine{}
	mockGraph := &MockGraph{}

	server := createServerWithMocks(mockStore, mockIdentities, mockUsage, mockPolicy, mockGraph, nil)

	fromStr := "2023-12-07T10:00:00Z"
	toStr := "2023-12-07T11:00:00Z"
	req := httptest.NewRequest("GET", "/v1/trends?from="+fromStr+"&to="+toStr+"&bucket=hour", nil)
	w := httptest.NewRecorder()

	server.handleTrends(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var stats []store.UsageStat
	if err := json.NewDecoder(w.Body).Decode(&stats); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(stats) != 1 {
		t.Errorf("Expected 1 stat, got %d", len(stats))
	}
}

func TestHandleWebhooks_Create(t *testing.T) {
	mockStore := &MockStore{}
	mockIdentities := &MockIdentityProjection{}
	mockUsage := &MockUsageProjection{}
	mockPolicy := &MockPolicyEngine{}
	mockGraph := &MockGraph{}

	server := createServerWithMocks(mockStore, mockIdentities, mockUsage, mockPolicy, mockGraph, nil)

	reqBody := map[string]interface{}{
		"url":    "http://example.com",
		"events": []string{"intent_decided"},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/v1/webhooks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleWebhooks(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", w.Code)
	}

	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if _, ok := resp["webhook_id"]; !ok {
		t.Errorf("Expected webhook_id in response")
	}

	if _, ok := resp["secret"]; !ok {
		t.Errorf("Expected secret in response")
	}
}

func TestHandleReports(t *testing.T) {
	mockStore := &MockStore{}
	mockIdentities := &MockIdentityProjection{}
	mockUsage := &MockUsageProjection{}
	mockPolicy := &MockPolicyEngine{}
	mockGraph := &MockGraph{}

	server := createServerWithMocks(mockStore, mockIdentities, mockUsage, mockPolicy, mockGraph, nil)

	req := httptest.NewRequest("GET", "/v1/reports?type=usage&from=2023-01-01T00:00:00Z&to=2023-01-02T00:00:00Z", nil)
	w := httptest.NewRecorder()

	server.handleReports(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Since it's streaming CSV, check content type
	if w.Header().Get("Content-Type") != "text/csv" {
		t.Errorf("Expected Content-Type text/csv, got %s", w.Header().Get("Content-Type"))
	}
}

func TestHandlePrune(t *testing.T) {
	mockStore := &MockStore{}
	mockIdentities := &MockIdentityProjection{}
	mockUsage := &MockUsageProjection{}
	mockPolicy := &MockPolicyEngine{}
	mockGraph := &MockGraph{}

	server := createServerWithMocks(mockStore, mockIdentities, mockUsage, mockPolicy, mockGraph, nil)

	reqBody := map[string]string{
		"retention": "720h",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/v1/admin/prune", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handlePrune(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp["status"] != "success" {
		t.Errorf("Expected status 'success', got %v", resp["status"])
	}

	if resp["pruned_count"] != float64(5) {
		t.Errorf("Expected pruned_count 5, got %v", resp["pruned_count"])
	}
}

func TestHandleGraph(t *testing.T) {
	mockStore := &MockStore{}
	mockIdentities := &MockIdentityProjection{}
	mockUsage := &MockUsageProjection{}
	mockPolicy := &MockPolicyEngine{}
	mockGraph := &MockGraph{}

	server := createServerWithMocks(mockStore, mockIdentities, mockUsage, mockPolicy, mockGraph, nil)

	req := httptest.NewRequest("GET", "/v1/graph", nil)
	w := httptest.NewRecorder()

	server.handleGraph(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var graph graph.Graph
	if err := json.NewDecoder(w.Body).Decode(&graph); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
}

func TestHandleIntent_Validation(t *testing.T) {
	mockStore := &MockStore{}
	mockPolicy := &MockPolicyEngine{}
	mockUsage := &MockUsageProjection{}
	mockIdentities := &MockIdentityProjection{}
	mockGraph := &MockGraph{}

	server := createServerWithMocks(mockStore, mockIdentities, mockUsage, mockPolicy, mockGraph, nil)

	tests := []struct {
		name           string
		body           string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "empty body",
			body:           "",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid_json_body",
		},
		{
			name:           "missing agent_id",
			body:           `{"identity_id":"id1","scope_id":"scope1","workload_id":"workload1"}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "missing_required_fields",
		},
		{
			name:           "missing scope_id",
			body:           `{"agent_id":"agent1","identity_id":"id1","workload_id":"workload1"}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "missing_required_fields",
		},
		{
			name:           "missing workload_id",
			body:           `{"agent_id":"agent1","identity_id":"id1","scope_id":"scope1"}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "missing_required_fields",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/v1/intent", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			server.handleIntent(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedError != "" {
				var resp map[string]string
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}
				if resp["error"] != tt.expectedError {
					t.Errorf("Expected error '%s', got '%s'", tt.expectedError, resp["error"])
				}
			}
		})
	}
}

func TestHandleIntent_Policy(t *testing.T) {
	mockStore := &MockStore{}
	mockUsage := &MockUsageProjection{}
	mockIdentities := &MockIdentityProjection{}
	mockGraph := &MockGraph{}

	reqBody := IntentRequest{
		AgentID:    "agent1",
		IdentityID: "identity1",
		ScopeID:    "scope1",
		WorkloadID: "workload1",
	}
	body, _ := json.Marshal(reqBody)

	tests := []struct {
		name             string
		policyResult     engine.PolicyEvaluationResult
		expectedDecision string
		checkReason      bool
		checkMods        bool
	}{
		{
			name: "deny",
			policyResult: engine.PolicyEvaluationResult{
				Decision:      "deny",
				Reason:        "insufficient_quota",
				Modifications: map[string]interface{}{},
				Warnings:      []string{},
				Trace:         []engine.RuleTrace{},
			},
			expectedDecision: "deny",
			checkReason:      true,
		},
		{
			name: "approve_with_modifications",
			policyResult: engine.PolicyEvaluationResult{
				Decision:      "approve_with_modifications",
				Reason:        "adjusted_cost",
				Modifications: map[string]interface{}{"cost": 2},
				Warnings:      []string{"high_usage"},
				Trace:         []engine.RuleTrace{},
			},
			expectedDecision: "approve_with_modifications",
			checkReason:      true,
			checkMods:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPolicy := &MockPolicyEngine{
				EvaluateFunc: func(intent engine.Intent) engine.PolicyEvaluationResult {
					return tt.policyResult
				},
			}

			server := createServerWithMocks(mockStore, mockIdentities, mockUsage, mockPolicy, mockGraph, nil)

			req := httptest.NewRequest("POST", "/v1/intent", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			server.handleIntent(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}

			var resp DecisionResponse
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if resp.Decision != tt.expectedDecision {
				t.Errorf("Expected decision '%s', got '%s'", tt.expectedDecision, resp.Decision)
			}

			if tt.checkReason && resp.Reason != tt.policyResult.Reason {
				t.Errorf("Expected reason '%s', got '%s'", tt.policyResult.Reason, resp.Reason)
			}

			if tt.checkMods {
				if len(resp.Modifications) != len(tt.policyResult.Modifications) {
					t.Errorf("Expected modifications length %d, got %d", len(tt.policyResult.Modifications), len(resp.Modifications))
				}
			}
		})
	}
}

func TestHandleDebugInject(t *testing.T) {
	mockStore := &MockStore{}
	mockPolicy := &MockPolicyEngine{}
	mockUsage := &MockUsageProjection{}
	mockIdentities := &MockIdentityProjection{}
	mockGraph := &MockGraph{}

	reqBody := map[string]interface{}{
		"provider_id": "test-provider",
		"pool_id":     "test-pool",
		"amount":      10,
	}
	body, _ := json.Marshal(reqBody)

	t.Run("no poller", func(t *testing.T) {
		server := createServerWithMocks(mockStore, mockIdentities, mockUsage, mockPolicy, mockGraph, nil)
		// poller is nil by default

		req := httptest.NewRequest("POST", "/debug/provider/inject", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		server.handleDebugInject(w, req)

		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("Expected status 503, got %d", w.Code)
		}

		var resp map[string]string
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if resp["error"] != "poller_not_configured" {
			t.Errorf("Expected error 'poller_not_configured', got '%s'", resp["error"])
		}
	})

	// For provider not found, we'd need to mock the poller, but since it's concrete, skip for now as per instructions
}

func TestHandleStatic(t *testing.T) {
	// Import testing/fstest
	// We need to add the import
	// For now, mock fs.FS manually

	mockStore := &MockStore{}
	mockPolicy := &MockPolicyEngine{}
	mockUsage := &MockUsageProjection{}
	mockIdentities := &MockIdentityProjection{}
	mockGraph := &MockGraph{}

	server := &Server{
		store:      mockStore,
		identities: mockIdentities,
		usage:      mockUsage,
		policy:     mockPolicy,
		graph:      mockGraph,
		staticFS:   nil, // Will test with nil first, but need to mock
	}

	// Since fs.FS is an interface, we can mock it
	// But for simplicity, test the case where staticFS is nil or file not found

	req := httptest.NewRequest("GET", "/index.html", nil)
	w := httptest.NewRecorder()

	handler := server.handleStatic()
	handler.ServeHTTP(w, req)

	// Should return 404 since staticFS is nil
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestAuthMiddleware(t *testing.T) {
	mockIdentities := &MockIdentityProjection{
		tokenMap: map[string]string{
			"hashed_token": "user1",
		},
		identities: []engine.Identity{
			{ID: "user1"},
		},
	}
	server := &Server{identities: mockIdentities}

	handler := server.withAuth(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Case 1: No Header
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 for missing header, got %d", w.Code)
	}

	// Case 2: Invalid Format
	req = httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "InvalidFormat")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 for invalid format, got %d", w.Code)
	}

	// Case 3: Invalid Token
	req = httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer invalid_token")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 for invalid token, got %d", w.Code)
	}

	// Case 4: Valid Token
	validToken := "valid_secret_token"
	hashed := hashToken(validToken)
	mockIdentities.tokenMap[hashed] = "user1"

	req = httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+validToken)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Expected 200 for valid token, got %d", w.Code)
	}
}

func TestLeaderCheckMiddleware(t *testing.T) {
	mockElection := &MockElectionManager{
		IsLeaderFunc: func() bool { return false },
		GetLeaderFunc: func(ctx context.Context) (string, bool, error) {
			return "http://leader-host:8090", true, nil
		},
	}
	server := &Server{election: mockElection}

	handler := server.withLeaderCheck(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// POST should redirect
	req := httptest.NewRequest("POST", "/v1/intent", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusTemporaryRedirect {
		t.Errorf("Expected 307 Redirect, got %d", w.Code)
	}
	loc := w.Header().Get("Location")
	if loc != "http://leader-host:8090/v1/intent" {
		t.Errorf("Expected location http://leader-host:8090/v1/intent, got %s", loc)
	}

	// GET should pass through
	req = httptest.NewRequest("GET", "/v1/trends", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Expected 200 OK for GET, got %d", w.Code)
	}
}

func TestLoggingMiddleware(t *testing.T) {
	handler := withLogging(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Header().Get("X-Trace-ID") == "" {
		t.Error("Expected X-Trace-ID header")
	}
}

func TestRecoveryMiddleware(t *testing.T) {
	handler := withRecovery(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("oops")
	}))

	req := httptest.NewRequest("GET", "/panic", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected 500 on panic, got %d", w.Code)
	}
}

func TestHandleEvents(t *testing.T) {
	mockStore := &MockStore{
		events: []store.Event{
			{EventID: "e1", EventType: "test"},
		},
	}
	server := createServerWithMocks(mockStore, nil, nil, nil, nil, nil)

	req := httptest.NewRequest("GET", "/v1/events?limit=1", nil)
	w := httptest.NewRecorder()

	server.handleEvents(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	var events []store.Event
	json.NewDecoder(w.Body).Decode(&events)
	if len(events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(events))
	}
}

func TestHandleIdentities_List(t *testing.T) {
	mockIdentities := &MockIdentityProjection{
		identities: []engine.Identity{
			{ID: "id1", Kind: "agent"},
		},
	}
	server := createServerWithMocks(nil, mockIdentities, nil, nil, nil, nil)

	req := httptest.NewRequest("GET", "/v1/identities", nil)
	w := httptest.NewRecorder()

	server.handleIdentities(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	var list []engine.Identity
	json.NewDecoder(w.Body).Decode(&list)
	if len(list) != 1 || list[0].ID != "id1" {
		t.Errorf("Unexpected list: %v", list)
	}
}
