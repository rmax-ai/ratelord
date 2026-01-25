package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/rmax-ai/ratelord/pkg/engine"
	"github.com/rmax-ai/ratelord/pkg/provider"
	"github.com/rmax-ai/ratelord/pkg/store"
)

// API Request/Response Structs

// Server encapsulates the HTTP API server
type Server struct {
	store      *store.Store
	server     *http.Server
	identities *engine.IdentityProjection
	usage      *engine.UsageProjection
	policy     *engine.PolicyEngine
	poller     *engine.Poller
}

// NewServer creates a new API server instance
func NewServer(st *store.Store, identities *engine.IdentityProjection, usage *engine.UsageProjection, policy *engine.PolicyEngine) *Server {
	return NewServerWithPoller(st, identities, usage, policy, nil)
}

// NewServerWithPoller creates a new API server instance with poller access (for debug endpoints)
func NewServerWithPoller(st *store.Store, identities *engine.IdentityProjection, usage *engine.UsageProjection, policy *engine.PolicyEngine, poller *engine.Poller) *Server {
	mux := http.NewServeMux()

	// Register routes
	mux.HandleFunc("/v1/health", handleHealth)

	s := &Server{
		store:      st,
		identities: identities,
		usage:      usage,
		policy:     policy,
		poller:     poller,
	}

	mux.HandleFunc("/v1/intent", s.handleIntent)
	mux.HandleFunc("/v1/identities", s.handleIdentities)
	mux.HandleFunc("/v1/events", s.handleEvents)

	// Debug endpoints
	if poller != nil {
		mux.HandleFunc("/debug/provider/inject", s.handleDebugInject)
	}

	// Middleware: Logging & Panic Recovery
	handler := withLogging(withRecovery(mux))

	s.server = &http.Server{
		Addr:         ":8090",
		Handler:      handler,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}

	return s
}

// Start runs the HTTP server (blocking)
func (s *Server) Start() error {
	fmt.Printf(`{"level":"info","msg":"server_starting","addr":"%s"}`+"\n", s.server.Addr)
	if err := s.server.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}
	return nil
}

// Stop gracefully shuts down the server
func (s *Server) Stop(ctx context.Context) error {
	fmt.Println(`{"level":"info","msg":"server_stopping"}`)
	return s.server.Shutdown(ctx)
}

// handleIntent processes intent negotiation requests.
// STUB IMPLEMENTATION: Always returns "approve".
func (s *Server) handleIntent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method_not_allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var req IntentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid_json_body"}`, http.StatusBadRequest)
		return
	}

	// Basic validation of mandatory fields
	if req.AgentID == "" || req.ScopeID == "" || req.WorkloadID == "" {
		http.Error(w, `{"error":"missing_required_fields"}`, http.StatusBadRequest)
		return
	}

	// M5.2: Use Policy Engine
	// Construct Intent object
	// Note: In a real system we would infer ProviderID/PoolID from context or graph.
	// For now, we assume if they are passed in client_context or similar we use them,
	// or we default to empty.
	intent := engine.Intent{
		IntentID:   "intent_" + fmt.Sprintf("%d", time.Now().UnixNano()),
		IdentityID: req.IdentityID,
		WorkloadID: req.WorkloadID,
		ScopeID:    req.ScopeID,
		// ProviderID and PoolID would be resolved here.
		// For bootstrapping, map scope "default" to mock provider
		ProviderID:   "mock-provider-1",
		PoolID:       "default",
		ExpectedCost: 1, // Default cost
	}

	// Evaluate
	result := s.policy.Evaluate(intent)

	// Persist the decision
	// TODO: Write intent_submitted and intent_decided events to store.
	// For M5.2, we just return the result.

	resp := DecisionResponse{
		IntentID:   intent.IntentID,
		Decision:   string(result.Decision),
		Reason:     result.Reason,
		ValidUntil: time.Now().Add(5 * time.Minute).Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		fmt.Printf(`{"level":"error","msg":"failed_to_encode_response","error":"%v"}`+"\n", err)
	}

	// Update usage on approval
	if result.Decision == "approve" {
		poolState, exists := s.usage.GetPoolState("mock-provider-1", "default")
		if exists {
			newUsed := poolState.Used + 1
			newRemaining := poolState.Remaining - 1
			now := time.Now()
			payload, _ := json.Marshal(map[string]interface{}{
				"provider_id": "mock-provider-1",
				"pool_id":     "default",
				"used":        newUsed,
				"remaining":   newRemaining,
			})
			evt := store.Event{
				EventID:       store.EventID(fmt.Sprintf("usage_intent_%d", now.UnixNano())),
				EventType:     store.EventTypeUsageObserved,
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
					CorrelationID: fmt.Sprintf("intent_%s", intent.IntentID),
					CausationID:   store.SentinelUnknown,
				},
				Payload: payload,
			}
			if err := s.store.AppendEvent(r.Context(), &evt); err != nil {
				fmt.Printf(`{"level":"error","msg":"failed_to_append_usage_event","error":"%v"}`+"\n", err)
			} else {
				s.usage.Apply(evt)
			}
		}
	}
}

// handleIdentities registers a new identity.
func (s *Server) handleIdentities(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		identities := s.identities.GetAll()
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(identities); err != nil {
			fmt.Printf(`{"level":"error","msg":"failed_to_encode_identities","error":"%v"}`+"\n", err)
		}
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method_not_allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var req IdentityRegistration
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid_json_body"}`, http.StatusBadRequest)
		return
	}

	if req.IdentityID == "" || req.Kind == "" {
		http.Error(w, `{"error":"missing_required_fields"}`, http.StatusBadRequest)
		return
	}

	// Construct the event
	now := time.Now()
	payload, _ := json.Marshal(map[string]interface{}{
		"kind":     req.Kind,
		"metadata": req.Metadata,
	})

	evt := store.Event{
		EventID:       store.EventID(fmt.Sprintf("evt_id_%d", now.UnixNano())),
		EventType:     store.EventTypeIdentityRegistered,
		SchemaVersion: 1,
		TsEvent:       now,
		TsIngest:      now,
		Source: store.EventSource{
			OriginKind: "client",
			OriginID:   "api", // In a real system, we might get this from context
			WriterID:   "ratelord-d",
		},
		Dimensions: store.EventDimensions{
			AgentID:    store.SentinelSystem,
			IdentityID: req.IdentityID,
			WorkloadID: store.SentinelSystem,
			ScopeID:    store.SentinelGlobal,
		},
		Correlation: store.EventCorrelation{
			CorrelationID: fmt.Sprintf("corr_%d", now.UnixNano()),
			CausationID:   store.SentinelUnknown,
		},
		Payload: payload,
	}

	if err := s.store.AppendEvent(r.Context(), &evt); err != nil {
		fmt.Printf(`{"level":"error","msg":"failed_to_append_identity_event","error":"%v"}`+"\n", err)
		http.Error(w, `{"error":"internal_server_error"}`, http.StatusInternalServerError)
		return
	}

	// Update projection
	if err := s.identities.Apply(evt); err != nil {
		fmt.Printf(`{"level":"error","msg":"failed_to_update_projection","error":"%v"}`+"\n", err)
		// We don't fail the request, but we log the inconsistency
	}

	// Response
	resp := map[string]string{
		"identity_id": req.IdentityID,
		"status":      "registered",
		"event_id":    string(evt.EventID),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		fmt.Printf(`{"level":"error","msg":"failed_to_encode_response","error":"%v"}`+"\n", err)
	}
}

// handleEvents returns recent events for diagnostics.
func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method_not_allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	// Parse limit query param
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val > 0 {
			limit = val
		}
	}

	// Query store
	events, err := s.store.ReadRecentEvents(r.Context(), limit)
	if err != nil {
		fmt.Printf(`{"level":"error","msg":"failed_to_read_events","error":"%v"}`+"\n", err)
		http.Error(w, `{"error":"internal_server_error"}`, http.StatusInternalServerError)
		return
	}

	// Return JSON
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(events); err != nil {
		fmt.Printf(`{"level":"error","msg":"failed_to_encode_events","error":"%v"}`+"\n", err)
	}
}

// handleDebugInject allows manual injection of usage into a provider (for drift testing)
func (s *Server) handleDebugInject(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method_not_allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ProviderID string `json:"provider_id"`
		PoolID     string `json:"pool_id"`
		Amount     int64  `json:"amount"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid_json_body"}`, http.StatusBadRequest)
		return
	}

	if s.poller == nil {
		http.Error(w, `{"error":"poller_not_configured"}`, http.StatusServiceUnavailable)
		return
	}

	prov := s.poller.GetProvider(provider.ProviderID(req.ProviderID))
	if prov == nil {
		http.Error(w, `{"error":"provider_not_found"}`, http.StatusNotFound)
		return
	}

	// Check if provider supports injection
	type Injector interface {
		InjectUsage(poolID string, amount int64) error
	}

	injector, ok := prov.(Injector)
	if !ok {
		http.Error(w, `{"error":"provider_does_not_support_injection"}`, http.StatusBadRequest)
		return
	}

	if err := injector.InjectUsage(req.PoolID, req.Amount); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"injection_failed","details":"%v"}`, err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"injected"}`))
}

// handleHealth returns simple status
func handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

// Middleware: Panic Recovery
func withRecovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				fmt.Printf(`{"level":"error","msg":"panic_recovered","error":"%v","path":"%s"}`+"\n", err, r.URL.Path)
				http.Error(w, `{"error":"internal_server_error"}`, http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// Middleware: Request Logging
func withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap writer to capture status code
		ww := &statusWriter{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(ww, r)

		duration := time.Since(start)
		fmt.Printf(`{"level":"info","msg":"http_request","method":"%s","path":"%s","status":%d,"duration_ms":%d}`+"\n",
			r.Method, r.URL.Path, ww.status, duration.Milliseconds())
	})
}

// statusWriter captures HTTP status code
type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}
