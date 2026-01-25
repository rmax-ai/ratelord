package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/rmax/ratelord/pkg/engine"
	"github.com/rmax/ratelord/pkg/store"
)

// API Request/Response Structs

// Server encapsulates the HTTP API server
type Server struct {
	store      *store.Store
	server     *http.Server
	identities *engine.IdentityProjection
}

// NewServer creates a new API server instance
func NewServer(st *store.Store, identities *engine.IdentityProjection) *Server {
	mux := http.NewServeMux()

	// Register routes
	mux.HandleFunc("/v1/health", handleHealth)

	// We need a closure here because NewServer is creating the mux before the Server struct exists entirely.
	// However, we are inside NewServer, so we don't have 's' yet.
	// But we will create 's' at the end.
	// A cleaner way is to define handlers as methods on Server, but we need the instance to register them.
	// Since standard http.ServeMux doesn't support registering methods on a nil instance easily without wrappers.
	// Let's defer registration or use a wrapper struct that we can close over.

	// Refactoring NewServer to instantiate Server first, then register routes.
	s := &Server{
		store:      st,
		identities: identities,
	}

	mux.HandleFunc("/v1/intent", s.handleIntent)
	mux.HandleFunc("/v1/identities", s.handleIdentities)
	mux.HandleFunc("/v1/events", s.handleEvents)

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

	// STUB LOGIC: Hardcoded approval
	// In the future, this will consult the Policy Engine.
	resp := DecisionResponse{
		IntentID:   "intent_stub_" + fmt.Sprintf("%d", time.Now().UnixNano()),
		Decision:   "approve",
		Reason:     "policy_engine_stub_auto_approval",
		ValidUntil: time.Now().Add(5 * time.Minute).Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		fmt.Printf(`{"level":"error","msg":"failed_to_encode_response","error":"%v"}`+"\n", err)
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
