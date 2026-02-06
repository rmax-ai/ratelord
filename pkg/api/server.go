package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/rmax-ai/ratelord/pkg/engine"
	"github.com/rmax-ai/ratelord/pkg/provider"
	"github.com/rmax-ai/ratelord/pkg/store"
)

// Context keys
type contextKey string

const traceIDKey contextKey = "trace_id"

// API Request/Response Structs

// Server encapsulates the HTTP API server
type Server struct {
	store      *store.Store
	server     *http.Server
	identities *engine.IdentityProjection
	usage      *engine.UsageProjection
	policy     *engine.PolicyEngine
	poller     *engine.Poller
	staticFS   fs.FS
}

// NewServer creates a new API server instance
func NewServer(st *store.Store, identities *engine.IdentityProjection, usage *engine.UsageProjection, policy *engine.PolicyEngine, addr string) *Server {
	return NewServerWithPoller(st, identities, usage, policy, nil, addr)
}

// NewServerWithPoller creates a new API server instance with poller access (for debug endpoints)
func NewServerWithPoller(st *store.Store, identities *engine.IdentityProjection, usage *engine.UsageProjection, policy *engine.PolicyEngine, poller *engine.Poller, addr string) *Server {
	mux := http.NewServeMux()

	// Register routes
	mux.HandleFunc("/v1/health", handleHealth)
	mux.Handle("/metrics", promhttp.Handler())

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

	// Static file handler (catch-all for SPA)
	if s.staticFS != nil {
		mux.Handle("/", s.handleStatic())
	}

	// Middleware: Logging & Panic Recovery
	handler := withLogging(withRecovery(mux))

	// Use default port if addr is empty
	if addr == "" {
		addr = ":8090"
	}

	s.server = &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}

	return s
}

// SetStaticFS sets the filesystem for serving static web assets
func (s *Server) SetStaticFS(fs fs.FS) {
	s.staticFS = fs
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

	// Update metrics
	engine.RatelordIntentTotal.WithLabelValues(intent.IdentityID, string(result.Decision)).Inc()

	// Persist the decision
	// TODO: Write intent_submitted and intent_decided events to store.
	// For M5.2, we just return the result.

	resp := DecisionResponse{
		IntentID:      intent.IntentID,
		Decision:      string(result.Decision),
		Reason:        result.Reason,
		ValidUntil:    time.Now().Add(5 * time.Minute).Format(time.RFC3339),
		Modifications: result.Modifications,
		Warnings:      result.Warnings,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		fmt.Printf(`{"level":"error","msg":"failed_to_encode_response","trace_id":"%s","error":"%v"}`+"\n", getTraceID(r.Context()), err)
	}

	// Log decision
	fmt.Printf(`{"level":"info","msg":"intent_decided","trace_id":"%s","intent_id":"%s","decision":"%s","reason":"%s"}`+"\n",
		getTraceID(r.Context()), intent.IntentID, result.Decision, result.Reason)

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
			fmt.Printf(`{"level":"error","msg":"failed_to_encode_identities","trace_id":"%s","error":"%v"}`+"\n", getTraceID(r.Context()), err)
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
		fmt.Printf(`{"level":"error","msg":"failed_to_append_identity_event","trace_id":"%s","error":"%v"}`+"\n", getTraceID(r.Context()), err)
		http.Error(w, `{"error":"internal_server_error"}`, http.StatusInternalServerError)
		return
	}

	// Update projection
	if err := s.identities.Apply(evt); err != nil {
		fmt.Printf(`{"level":"error","msg":"failed_to_update_projection","trace_id":"%s","error":"%v"}`+"\n", getTraceID(r.Context()), err)
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
		fmt.Printf(`{"level":"error","msg":"failed_to_encode_response","trace_id":"%s","error":"%v"}`+"\n", getTraceID(r.Context()), err)
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
		fmt.Printf(`{"level":"error","msg":"failed_to_read_events","trace_id":"%s","error":"%v"}`+"\n", getTraceID(r.Context()), err)
		http.Error(w, `{"error":"internal_server_error"}`, http.StatusInternalServerError)
		return
	}

	// Return JSON
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(events); err != nil {
		fmt.Printf(`{"level":"error","msg":"failed_to_encode_events","trace_id":"%s","error":"%v"}`+"\n", getTraceID(r.Context()), err)
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

// handleStatic serves static web assets with SPA fallback
func (s *Server) handleStatic() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Skip API and debug routes
		if strings.HasPrefix(path, "/v1/") || strings.HasPrefix(path, "/debug/") {
			http.NotFound(w, r)
			return
		}

		// Try to serve the file directly
		if file, err := s.staticFS.Open(path); err == nil {
			defer file.Close()
			if stat, err := file.Stat(); err == nil && !stat.IsDir() {
				// Set content type based on extension
				if strings.HasSuffix(path, ".css") {
					w.Header().Set("Content-Type", "text/css")
				} else if strings.HasSuffix(path, ".js") {
					w.Header().Set("Content-Type", "application/javascript")
				} else if strings.HasSuffix(path, ".html") {
					w.Header().Set("Content-Type", "text/html")
				}
				io.Copy(w, file)
				return
			}
		}

		// Fallback to index.html for SPA routing
		if indexFile, err := s.staticFS.Open("index.html"); err == nil {
			defer indexFile.Close()
			w.Header().Set("Content-Type", "text/html")
			io.Copy(w, indexFile)
			return
		}

		// If index.html not found, 404
		http.NotFound(w, r)
	})
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

		// 1. Extract or Generate Trace ID
		traceID := r.Header.Get("X-Trace-ID")
		if traceID == "" {
			traceID = generateTraceID()
		}

		// 2. Inject into Context
		ctx := context.WithValue(r.Context(), traceIDKey, traceID)
		r = r.WithContext(ctx)

		// Wrap writer to capture status code
		ww := &statusWriter{ResponseWriter: w, status: http.StatusOK}

		// 3. Set response header
		w.Header().Set("X-Trace-ID", traceID)

		next.ServeHTTP(ww, r)

		duration := time.Since(start)
		fmt.Printf(`{"level":"info","msg":"http_request","trace_id":"%s","method":"%s","path":"%s","status":%d,"duration_ms":%d}`+"\n",
			traceID, r.Method, r.URL.Path, ww.status, duration.Milliseconds())
	})
}

func generateTraceID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Fallback if random fails (unlikely)
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

func getTraceID(ctx context.Context) string {
	if v, ok := ctx.Value(traceIDKey).(string); ok {
		return v
	}
	return ""
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
