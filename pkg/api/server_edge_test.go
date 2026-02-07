package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/rmax-ai/ratelord/pkg/engine"
	"github.com/rmax-ai/ratelord/pkg/protocol"
	"github.com/rmax-ai/ratelord/pkg/provider"
	"github.com/rmax-ai/ratelord/pkg/store"
)

// MockStoreError simulates errors
type MockStoreError struct {
	MockStore
}

func (m *MockStoreError) AppendEvent(ctx context.Context, event *store.Event) error {
	return errors.New("store error")
}

func (m *MockStoreError) GetUsageStats(ctx context.Context, filter store.UsageFilter) ([]store.UsageStat, error) {
	return nil, errors.New("usage stats error")
}

func (m *MockStoreError) PruneEvents(ctx context.Context, retention time.Duration, includeType string, excludeTypes []string) (int64, error) {
	return 0, errors.New("prune error")
}

func (m *MockStoreError) RegisterWebhook(ctx context.Context, cfg *store.WebhookConfig) error {
	return errors.New("webhook error")
}

func (m *MockStoreError) DeleteIdentityData(ctx context.Context, id string) error {
	return errors.New("delete error")
}

func TestHandleIntent_StoreError(t *testing.T) {
	mockStore := &MockStoreError{}
	mockPolicy := &MockPolicyEngine{}
	mockUsage := &MockUsageProjection{}
	mockIdentities := &MockIdentityProjection{}
	mockGraph := &MockGraph{}

	server := createServerWithMocks(mockStore, mockIdentities, mockUsage, mockPolicy, mockGraph, nil)

	reqBody := protocol.IntentRequest{
		AgentID:    "agent1",
		IdentityID: "identity1",
		ScopeID:    "scope1",
		WorkloadID: "workload1",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/v1/intent", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Should proceed despite store error (logs only)
	server.handleIntent(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 even with store error (best effort logging), got %d", w.Code)
	}
}

func TestHandleTrends_Validation(t *testing.T) {
	server := &Server{}

	// Invalid time format
	req := httptest.NewRequest("GET", "/v1/trends?from=invalid", nil)
	w := httptest.NewRecorder()
	server.handleTrends(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for invalid from, got %d", w.Code)
	}

	// Invalid bucket
	req = httptest.NewRequest("GET", "/v1/trends?bucket=year", nil)
	w = httptest.NewRecorder()
	server.handleTrends(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for invalid bucket, got %d", w.Code)
	}

	// To before From
	from := time.Now().Add(1 * time.Hour).Format(time.RFC3339)
	to := time.Now().Format(time.RFC3339)
	req = httptest.NewRequest("GET", "/v1/trends?from="+from+"&to="+to, nil)
	w = httptest.NewRecorder()
	server.handleTrends(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for to before from, got %d", w.Code)
	}
}

func TestHandleTrends_StoreError(t *testing.T) {
	mockStore := &MockStoreError{}
	server := &Server{store: mockStore}

	from := time.Now().Add(-1 * time.Hour).Format(time.RFC3339)
	to := time.Now().Format(time.RFC3339)

	q := url.Values{}
	q.Set("from", from)
	q.Set("to", to)

	req := httptest.NewRequest("GET", "/v1/trends?"+q.Encode(), nil)
	w := httptest.NewRecorder()
	server.handleTrends(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected 500 for store error, got %d", w.Code)
	}
}

func TestHandleReports_Error(t *testing.T) {
	server := &Server{}

	// Missing type
	req := httptest.NewRequest("GET", "/v1/reports", nil)
	w := httptest.NewRecorder()
	server.handleReports(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for missing type, got %d", w.Code)
	}

	// Invalid time
	req = httptest.NewRequest("GET", "/v1/reports?type=usage&from=invalid", nil)
	w = httptest.NewRecorder()
	server.handleReports(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for invalid from, got %d", w.Code)
	}

	// Invalid Type
	req = httptest.NewRequest("GET", "/v1/reports?type=invalid_type", nil)
	w = httptest.NewRecorder()
	// Need store to avoid panic if NewReportGenerator checks it (it assigns it)
	// But NewReportGenerator checks type first.
	// Actually NewReportGenerator(type, store) usually validates type.
	// Let's assume it doesn't panic on nil store if type is invalid.
	server.store = &MockStoreError{}
	server.handleReports(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for invalid type, got %d", w.Code)
	}
}

func TestHandleReports_Filters(t *testing.T) {
	mockStore := &MockStore{}
	server := &Server{store: mockStore}

	// Pass all filters
	q := url.Values{}
	q.Set("type", "usage")
	q.Set("identity_id", "i1")
	q.Set("scope_id", "s1")
	q.Set("provider_id", "p1")
	q.Set("pool_id", "pool1")
	q.Set("bucket", "day")
	q.Set("from", time.Now().Add(-24*time.Hour).Format(time.RFC3339))

	req := httptest.NewRequest("GET", "/v1/reports?"+q.Encode(), nil)
	w := httptest.NewRecorder()
	server.handleReports(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}
}

// MockProvider for Injection
type MockProvider struct {
	id       provider.ProviderID
	injected bool
}

func (m *MockProvider) ID() provider.ProviderID { return m.id }
func (m *MockProvider) Poll(ctx context.Context) (provider.PollResult, error) {
	return provider.PollResult{}, nil
}
func (m *MockProvider) Restore(state []byte) error { return nil }
func (m *MockProvider) InjectUsage(poolID string, amount int64) error {
	m.injected = true
	return nil
}

func TestHandleDebugInject_WithPoller(t *testing.T) {
	// engine.NewPoller needs a *store.Store, but we can pass nil if it's not used by GetProvider/Register
	poller := engine.NewPoller(nil, time.Minute, nil, nil)
	mockProv := &MockProvider{id: "p1"}
	poller.Register(mockProv)

	server := &Server{poller: poller}

	// Success case
	body, _ := json.Marshal(map[string]interface{}{
		"provider_id": "p1",
		"pool_id":     "pool1",
		"amount":      100,
	})
	req := httptest.NewRequest("POST", "/debug/provider/inject", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleDebugInject(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}
	if !mockProv.injected {
		t.Error("Expected InjectUsage to be called")
	}

	// Provider not found
	body2, _ := json.Marshal(map[string]interface{}{
		"provider_id": "p2", // Not registered
		"pool_id":     "pool1",
		"amount":      100,
	})
	req2 := httptest.NewRequest("POST", "/debug/provider/inject", bytes.NewReader(body2))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	server.handleDebugInject(w2, req2)

	if w2.Code != http.StatusNotFound {
		t.Errorf("Expected 404 for unknown provider, got %d", w2.Code)
	}
}

// MockFS
type MockFS struct {
	files map[string]string
}

func (m *MockFS) Open(name string) (fs.File, error) {
	// Remove leading slash if present (standard fs behavior usually expects no leading slash or handles it)
	name = strings.TrimPrefix(name, "/")
	if content, ok := m.files[name]; ok {
		return &MockFile{name: name, content: content}, nil
	}
	return nil, fs.ErrNotExist
}

type MockFile struct {
	name    string
	content string
	offset  int64
}

func (m *MockFile) Stat() (fs.FileInfo, error) {
	return &MockFileInfo{name: m.name, size: int64(len(m.content))}, nil
}
func (m *MockFile) Read(b []byte) (int, error) {
	if m.offset >= int64(len(m.content)) {
		return 0, io.EOF
	}
	n := copy(b, m.content[m.offset:])
	m.offset += int64(n)
	return n, nil
}
func (m *MockFile) Close() error { return nil }

type MockFileInfo struct {
	name string
	size int64
}

func (m *MockFileInfo) Name() string       { return m.name }
func (m *MockFileInfo) Size() int64        { return m.size }
func (m *MockFileInfo) Mode() fs.FileMode  { return 0444 }
func (m *MockFileInfo) ModTime() time.Time { return time.Now() }
func (m *MockFileInfo) IsDir() bool        { return false }
func (m *MockFileInfo) Sys() interface{}   { return nil }

func TestHandleStatic_Files(t *testing.T) {
	mockFS := &MockFS{
		files: map[string]string{
			"index.html": "<html></html>",
			"style.css":  "body {}",
		},
	}
	server := &Server{staticFS: mockFS}
	handler := server.handleStatic()

	// Serve file
	req := httptest.NewRequest("GET", "/style.css", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}
	if w.Header().Get("Content-Type") != "text/css" {
		t.Errorf("Expected text/css, got %s", w.Header().Get("Content-Type"))
	}

	// Serve fallback
	req = httptest.NewRequest("GET", "/unknown", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Expected 200 (fallback), got %d", w.Code)
	}
	if w.Header().Get("Content-Type") != "text/html" {
		t.Errorf("Expected text/html, got %s", w.Header().Get("Content-Type"))
	}

	// API skip
	req = httptest.NewRequest("GET", "/v1/api", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected 404 for API path, got %d", w.Code)
	}
}

func TestHandlePrune_StoreError(t *testing.T) {
	mockStore := &MockStoreError{}
	server := &Server{store: mockStore}

	body, _ := json.Marshal(map[string]string{"retention": "720h"})
	req := httptest.NewRequest("POST", "/v1/admin/prune", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handlePrune(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected 500 for prune error, got %d", w.Code)
	}
}

func TestHandleClusterNodes_Error(t *testing.T) {
	server := &Server{} // cluster is nil
	req := httptest.NewRequest("GET", "/v1/cluster/nodes", nil)
	w := httptest.NewRecorder()
	server.handleClusterNodes(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected 503 when cluster not initialized, got %d", w.Code)
	}
}

func TestHandleGraph_Error(t *testing.T) {
	server := &Server{} // graph is nil
	req := httptest.NewRequest("GET", "/v1/graph", nil)
	w := httptest.NewRecorder()
	server.handleGraph(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected 503 when graph not initialized, got %d", w.Code)
	}
}

func TestNewServer_Defaults(t *testing.T) {
	s := NewServer(nil, nil, nil, nil, nil, nil, "")
	if s.server.Addr != ":8090" {
		t.Errorf("Expected default addr :8090, got %s", s.server.Addr)
	}
}

func TestHandleWebhooks_StoreError(t *testing.T) {
	mockStore := &MockStoreError{}
	server := &Server{store: mockStore}

	body, _ := json.Marshal(map[string]interface{}{"url": "http://x.com", "events": []string{"all"}})
	req := httptest.NewRequest("POST", "/v1/webhooks", bytes.NewReader(body))
	w := httptest.NewRecorder()

	server.handleWebhooks(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected 500 for webhook store error, got %d", w.Code)
	}
}

func TestServer_StartError(t *testing.T) {
	// Port -1 is invalid
	s := NewServer(nil, nil, nil, nil, nil, nil, ":-1")
	err := s.Start()
	if err == nil {
		t.Error("Expected error starting on invalid port")
	}
}

func TestServer_StartTLS_Error(t *testing.T) {
	s := NewServer(nil, nil, nil, nil, nil, nil, ":0") // Random port
	s.SetTLS("invalid.crt", "invalid.key")
	err := s.Start()
	if err == nil {
		t.Error("Expected error starting TLS with invalid certs")
	}
}

func TestLeaderCheckMiddleware_Errors(t *testing.T) {
	mockElection := &MockElectionManager{
		IsLeaderFunc: func() bool { return false },
		GetLeaderFunc: func(ctx context.Context) (string, bool, error) {
			return "", false, errors.New("election error")
		},
	}
	server := &Server{election: mockElection}

	handler := server.withLeaderCheck(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("POST", "/v1/intent", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected 500 for election error, got %d", w.Code)
	}

	// No leader elected
	mockElection.GetLeaderFunc = func(ctx context.Context) (string, bool, error) {
		return "", false, nil
	}
	req = httptest.NewRequest("POST", "/v1/intent", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected 503 for no leader, got %d", w.Code)
	}
}

func TestHandleIdentities_Errors(t *testing.T) {
	mockStore := &MockStoreError{}
	server := &Server{store: mockStore, identities: &MockIdentityProjection{}}

	// Delete Error
	req := httptest.NewRequest("DELETE", "/v1/identities/id1", nil)
	w := httptest.NewRecorder()
	server.handleIdentities(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected 500 for delete error, got %d", w.Code)
	}

	// Register missing fields
	body, _ := json.Marshal(map[string]string{})
	req = httptest.NewRequest("POST", "/v1/identities", bytes.NewReader(body))
	w = httptest.NewRecorder()
	server.handleIdentities(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for missing fields, got %d", w.Code)
	}

	// Register store error
	body, _ = json.Marshal(map[string]string{"identity_id": "id1", "kind": "agent"})
	req = httptest.NewRequest("POST", "/v1/identities", bytes.NewReader(body))
	w = httptest.NewRecorder()
	server.handleIdentities(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected 500 for register store error, got %d", w.Code)
	}
}

type MockTracker struct {
	called bool
}

func (m *MockTracker) TrackUsage(p, pool string, amt int64) { m.called = true }

func TestHandleIntent_UsageTracking(t *testing.T) {
	mockStore := &MockStore{}
	tracker := &MockTracker{}
	server := createServerWithMocks(mockStore, &MockIdentityProjection{}, &MockUsageProjection{}, &MockPolicyEngine{}, &MockGraph{}, nil)
	server.SetUsageTracker(tracker)

	// Request that approves
	reqBody := protocol.IntentRequest{AgentID: "a", IdentityID: "i", ScopeID: "s", WorkloadID: "w"}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/v1/intent", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleIntent(w, req)

	if !tracker.called {
		t.Error("Expected usage tracker to be called")
	}
}

func TestNewServerWithPoller_RegistersDebug(t *testing.T) {
	poller := engine.NewPoller(nil, time.Minute, nil, nil)
	s := NewServerWithPoller(nil, nil, nil, nil, nil, nil, poller, "")

	// Server Handler is wrapped with middleware.
	req := httptest.NewRequest("POST", "/debug/provider/inject", nil)
	w := httptest.NewRecorder()
	s.server.Handler.ServeHTTP(w, req)

	// If route exists, it should not be 404.
	// We expect 400 because body is nil (invalid json)
	if w.Code == http.StatusNotFound {
		t.Error("Expected debug route to be registered")
	}
}

func TestServer_Stop(t *testing.T) {
	s := NewServer(nil, nil, nil, nil, nil, nil, ":0")
	// Stop without start should be fine
	err := s.Stop(context.Background())
	if err != nil {
		t.Errorf("Stop failed: %v", err)
	}
}

func TestHandleIntent_WithEpoch(t *testing.T) {
	mockStore := &MockStore{}
	mockElection := &MockElectionManager{
		GetEpochFunc: func() int64 { return 123 },
	}
	server := createServerWithMocks(mockStore, &MockIdentityProjection{}, &MockUsageProjection{}, &MockPolicyEngine{}, &MockGraph{}, mockElection)

	reqBody := protocol.IntentRequest{AgentID: "a", IdentityID: "i", ScopeID: "s", WorkloadID: "w"}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/v1/intent", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleIntent(w, req)

	// Check if event has Epoch set
	if len(mockStore.events) > 0 {
		if mockStore.events[0].Epoch != 123 {
			t.Errorf("Expected epoch 123, got %d", mockStore.events[0].Epoch)
		}
	}
}

func TestHandleIntent_UsageEventError(t *testing.T) {
	mockStore := &MockStoreError{}
	// Setup usage so exists=true
	poolStates := map[string]map[string]engine.PoolState{
		"mock-provider-1": {"default": engine.PoolState{}},
	}
	mockUsage := &MockUsageProjection{poolStates: poolStates}

	server := createServerWithMocks(mockStore, &MockIdentityProjection{}, mockUsage, &MockPolicyEngine{}, &MockGraph{}, nil)

	reqBody := protocol.IntentRequest{AgentID: "a", IdentityID: "i", ScopeID: "s", WorkloadID: "w"}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/v1/intent", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleIntent(w, req)

	// It logs error but returns 200. Coverage should be hit.
	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}
}
