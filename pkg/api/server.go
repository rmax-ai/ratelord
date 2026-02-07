package api

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
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
	"github.com/rmax-ai/ratelord/pkg/graph"
	"github.com/rmax-ai/ratelord/pkg/provider"
	"github.com/rmax-ai/ratelord/pkg/reports"
	"github.com/rmax-ai/ratelord/pkg/store"
)

// Context keys
type contextKey string

const traceIDKey contextKey = "trace_id"

// Interfaces for dependencies to enable mocking

type StoreInterface interface {
	AppendEvent(ctx context.Context, event *store.Event) error
	ReadRecentEvents(ctx context.Context, limit int) ([]*store.Event, error)
	DeleteIdentityData(ctx context.Context, id string) error
	PruneEvents(ctx context.Context, retention time.Duration, includeType string, excludeTypes []string) (int64, error)
	GetUsageStats(ctx context.Context, filter store.UsageFilter) ([]store.UsageStat, error)
	QueryEvents(ctx context.Context, filter store.EventFilter) ([]*store.Event, error)

	// Webhooks
	RegisterWebhook(ctx context.Context, cfg *store.WebhookConfig) error
	ListWebhooks(ctx context.Context) ([]*store.WebhookConfig, error)
	DeleteWebhook(ctx context.Context, webhookID string) error
}

type IdentityProjectionInterface interface {
	GetByTokenHash(hash string) (engine.Identity, bool)
	Apply(event store.Event) error
	GetAll() []engine.Identity
}

type UsageProjectionInterface interface {
	GetPoolState(providerID, poolID string) (engine.PoolState, bool)
	Apply(event store.Event) error
}

type GraphProjectionInterface interface {
	GetGraph() *graph.Graph
}

type ElectionManagerInterface interface {
	IsLeader() bool
	GetLeader(ctx context.Context) (string, bool, error)
	GetEpoch() int64
}

// PolicyEngineInterface defines the interface for policy engine
type PolicyEngineInterface interface {
	Evaluate(intent engine.Intent) engine.PolicyEvaluationResult
}

// API Request/Response Structs

// Server encapsulates the HTTP API server
type Server struct {
	store      StoreInterface
	server     *http.Server
	identities IdentityProjectionInterface
	usage      UsageProjectionInterface
	policy     PolicyEngineInterface
	poller     *engine.Poller
	cluster    *engine.ClusterTopology
	graph      GraphProjectionInterface
	staticFS   fs.FS

	// TLS Config
	tlsCertFile string
	tlsKeyFile  string

	// Federated Usage Tracker
	tracker UsageTracker

	// High Availability
	election ElectionManagerInterface
}

// UsageTracker defines an interface for tracking local usage
type UsageTracker interface {
	TrackUsage(providerID, poolID string, amount int64)
}

// NewServer creates a new API server instance
func NewServer(st *store.Store, identities *engine.IdentityProjection, usage *engine.UsageProjection, policy PolicyEngineInterface, cluster *engine.ClusterTopology, graphProj *graph.Projection, addr string) *Server {
	return NewServerWithPoller(st, identities, usage, policy, cluster, graphProj, nil, addr)
}

// NewServerWithPoller creates a new API server instance with poller access (for debug endpoints)
func NewServerWithPoller(st *store.Store, identities *engine.IdentityProjection, usage *engine.UsageProjection, policy PolicyEngineInterface, cluster *engine.ClusterTopology, graphProj *graph.Projection, poller *engine.Poller, addr string) *Server {
	mux := http.NewServeMux()

	// Register routes
	mux.HandleFunc("/v1/health", handleHealth)
	mux.Handle("/metrics", promhttp.Handler())

	s := &Server{
		store:      st,
		identities: identities,
		usage:      usage,
		policy:     policy,
		cluster:    cluster,
		graph:      graphProj,
		poller:     poller,
	}

	mux.HandleFunc("/v1/intent", s.withLeaderCheck(s.withAuth(s.handleIntent)))
	mux.HandleFunc("/v1/identities", s.withLeaderCheck(s.handleIdentities)) // handleIdentities checks method inside
	mux.HandleFunc("/v1/events", s.handleEvents)
	mux.HandleFunc("/v1/trends", s.handleTrends)
	mux.HandleFunc("/v1/reports", s.handleReports)
	mux.HandleFunc("/v1/graph", s.handleGraph)
	mux.HandleFunc("/v1/webhooks", s.withLeaderCheck(s.withAuth(s.handleWebhooks)))
	mux.HandleFunc("/v1/federation/grant", s.withLeaderCheck(s.handleGrant))
	mux.HandleFunc("/v1/cluster/nodes", s.handleClusterNodes)
	mux.HandleFunc("/v1/admin/prune", s.withLeaderCheck(s.withAuth(s.handlePrune)))

	// Debug endpoints
	if poller != nil {
		mux.HandleFunc("/debug/provider/inject", s.withLeaderCheck(s.handleDebugInject))
	}

	// Static file handler (catch-all for SPA)
	if s.staticFS != nil {
		mux.Handle("/", s.handleStatic())
	}

	// Middleware: Logging, Panic Recovery, Security Headers
	handler := withLogging(withRecovery(withSecureHeaders(mux)))

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

// SetTLS configures the server to use TLS
func (s *Server) SetTLS(certFile, keyFile string) {
	s.tlsCertFile = certFile
	s.tlsKeyFile = keyFile
}

// SetUsageTracker sets the usage tracker for federation
func (s *Server) SetUsageTracker(t UsageTracker) {
	s.tracker = t
}

// SetElectionManager sets the election manager for HA routing
func (s *Server) SetElectionManager(em ElectionManagerInterface) {
	s.election = em
}

// getEpoch returns the current leadership epoch.
func (s *Server) getEpoch() int64 {
	if s.election != nil {
		return s.election.GetEpoch()
	}
	return 0
}

// Start runs the HTTP server (blocking)
func (s *Server) Start() error {
	if s.tlsCertFile != "" && s.tlsKeyFile != "" {
		fmt.Printf(`{"level":"info","msg":"server_starting_tls","addr":"%s"}`+"\n", s.server.Addr)
		if err := s.server.ListenAndServeTLS(s.tlsCertFile, s.tlsKeyFile); err != http.ErrServerClosed {
			return err
		}
	} else {
		fmt.Printf(`{"level":"info","msg":"server_starting","addr":"%s"}`+"\n", s.server.Addr)
		if err := s.server.ListenAndServe(); err != http.ErrServerClosed {
			return err
		}
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
		Debug:        req.Debug,
	}

	// Evaluate
	result := s.policy.Evaluate(intent)

	// Update metrics
	engine.RatelordIntentTotal.WithLabelValues(intent.IdentityID, string(result.Decision)).Inc()

	// Persist the decision
	// Create payload
	decPayload, _ := json.Marshal(map[string]interface{}{
		"decision":      result.Decision,
		"reason":        result.Reason,
		"modifications": result.Modifications,
		"warnings":      result.Warnings,
		"trace":         result.Trace,
	})

	now := time.Now()
	decEvent := store.Event{
		EventID:       store.EventID(fmt.Sprintf("dec_%s", intent.IntentID)),
		EventType:     store.EventTypeIntentDecided,
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
			AgentID:    intent.IdentityID, // Using IdentityID as AgentID proxy for now if AgentID not explicit
			IdentityID: intent.IdentityID,
			WorkloadID: intent.WorkloadID,
			ScopeID:    intent.ScopeID,
		},
		Correlation: store.EventCorrelation{
			CorrelationID: fmt.Sprintf("intent_%s", intent.IntentID),
			CausationID:   store.SentinelUnknown,
		},
		Payload: decPayload,
	}

	if err := s.store.AppendEvent(r.Context(), &decEvent); err != nil {
		fmt.Printf(`{"level":"error","msg":"failed_to_append_decision_event","trace_id":"%s","error":"%v"}`+"\n", getTraceID(r.Context()), err)
	}

	// Convert trace to []interface{} for JSON serialization
	var trace []interface{}
	for _, t := range result.Trace {
		trace = append(trace, t)
	}

	resp := DecisionResponse{
		IntentID:      intent.IntentID,
		Decision:      string(result.Decision),
		Reason:        result.Reason,
		ValidUntil:    time.Now().Add(5 * time.Minute).Format(time.RFC3339),
		Modifications: result.Modifications,
		Warnings:      result.Warnings,
		Trace:         trace,
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
	if result.Decision == "approve" || result.Decision == "approve_with_modifications" {
		// Federation Hook
		if s.tracker != nil {
			// TODO: Use correct pool ID from policy evaluation or intent
			s.tracker.TrackUsage(intent.ProviderID, intent.PoolID, intent.ExpectedCost)
		}

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
				Epoch:         s.getEpoch(),
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
	if r.Method == http.MethodDelete {
		// Extract ID from path
		path := strings.TrimPrefix(r.URL.Path, "/v1/identities")
		if path == "" || path == "/" {
			http.Error(w, `{"error":"missing_identity_id"}`, http.StatusBadRequest)
			return
		}
		id := strings.TrimPrefix(path, "/")
		if id == "" {
			http.Error(w, `{"error":"invalid_identity_id"}`, http.StatusBadRequest)
			return
		}

		// Emit event
		now := time.Now()
		payload, _ := json.Marshal(map[string]interface{}{
			"reason": "api_request",
		})
		evt := store.Event{
			EventID:       store.EventID(fmt.Sprintf("evt_del_%d", now.UnixNano())),
			EventType:     store.EventTypeIdentityDeleted,
			SchemaVersion: 1,
			TsEvent:       now,
			TsIngest:      now,
			Epoch:         s.getEpoch(),
			Source: store.EventSource{
				OriginKind: "client",
				OriginID:   "api",
				WriterID:   "ratelord-d",
			},
			Dimensions: store.EventDimensions{
				AgentID:    store.SentinelSystem,
				IdentityID: id,
				WorkloadID: store.SentinelSystem,
				ScopeID:    store.SentinelGlobal,
			},
			Correlation: store.EventCorrelation{
				CorrelationID: fmt.Sprintf("del_%s", id),
				CausationID:   store.SentinelUnknown,
			},
			Payload: payload,
		}
		if err := s.store.AppendEvent(r.Context(), &evt); err != nil {
			fmt.Printf(`{"level":"error","msg":"failed_to_append_delete_event","trace_id":"%s","error":"%v"}`+"\n", getTraceID(r.Context()), err)
			http.Error(w, `{"error":"internal_server_error"}`, http.StatusInternalServerError)
			return
		}

		// Delete data
		if err := s.store.DeleteIdentityData(r.Context(), id); err != nil {
			fmt.Printf(`{"level":"error","msg":"failed_to_delete_identity_data","trace_id":"%s","error":"%v"}`+"\n", getTraceID(r.Context()), err)
			http.Error(w, `{"error":"internal_server_error"}`, http.StatusInternalServerError)
			return
		}

		// Apply to projection
		if err := s.identities.Apply(evt); err != nil {
			fmt.Printf(`{"level":"error","msg":"failed_to_update_projection","trace_id":"%s","error":"%v"}`+"\n", getTraceID(r.Context()), err)
			// We don't fail the request, but we log the inconsistency
		}

		w.WriteHeader(http.StatusNoContent)
		return
	}

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

	// Handle Token Logic
	var token, tokenHash string
	if req.Token != "" {
		// User provided a token, hash it
		tokenHash = hashToken(req.Token)
	} else {
		// Generate a new token
		token = generateToken()
		tokenHash = hashToken(token)
	}

	payload, _ := json.Marshal(map[string]interface{}{
		"kind":       req.Kind,
		"metadata":   req.Metadata,
		"token_hash": tokenHash,
	})

	evt := store.Event{
		EventID:       store.EventID(fmt.Sprintf("evt_id_%d", now.UnixNano())),
		EventType:     store.EventTypeIdentityRegistered,
		SchemaVersion: 1,
		TsEvent:       now,
		TsIngest:      now,
		Epoch:         s.getEpoch(),
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
	resp := IdentityResponse{
		IdentityID: req.IdentityID,
		Status:     "registered",
		EventID:    string(evt.EventID),
		Token:      token, // Will be empty if user provided it
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		fmt.Printf(`{"level":"error","msg":"failed_to_encode_response","trace_id":"%s","error":"%v"}`+"\n", getTraceID(r.Context()), err)
	}
}

// handlePrune allows admin to delete old events.
func (s *Server) handlePrune(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method_not_allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Retention string `json:"retention"` // e.g., "720h"
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid_json_body"}`, http.StatusBadRequest)
		return
	}

	retention, err := time.ParseDuration(req.Retention)
	if err != nil {
		http.Error(w, `{"error":"invalid_retention_format","example":"720h"}`, http.StatusBadRequest)
		return
	}

	count, err := s.store.PruneEvents(r.Context(), retention, "", nil)
	if err != nil {
		fmt.Printf(`{"level":"error","msg":"failed_to_prune_events","trace_id":"%s","error":"%v"}`+"\n", getTraceID(r.Context()), err)
		http.Error(w, fmt.Sprintf(`{"error":"prune_failed","details":"%v"}`, err), http.StatusInternalServerError)
		return
	}

	resp := map[string]interface{}{
		"status":         "success",
		"pruned_count":   count,
		"retention_used": retention.String(),
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

// handleTrends returns aggregated usage statistics.
func (s *Server) handleTrends(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method_not_allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")
	bucket := r.URL.Query().Get("bucket")
	if bucket == "" {
		bucket = "hour" // default
	}
	if bucket != "hour" && bucket != "day" {
		http.Error(w, `{"error":"invalid_bucket","valid":["hour","day"]}`, http.StatusBadRequest)
		return
	}

	from, err := time.Parse(time.RFC3339, fromStr)
	if err != nil {
		http.Error(w, `{"error":"invalid_from","format":"RFC3339"}`, http.StatusBadRequest)
		return
	}

	to, err := time.Parse(time.RFC3339, toStr)
	if err != nil {
		http.Error(w, `{"error":"invalid_to","format":"RFC3339"}`, http.StatusBadRequest)
		return
	}

	if to.Before(from) {
		http.Error(w, `{"error":"to_before_from"}`, http.StatusBadRequest)
		return
	}

	filter := store.UsageFilter{
		From:       from,
		To:         to,
		Bucket:     bucket,
		ProviderID: r.URL.Query().Get("provider_id"),
		PoolID:     r.URL.Query().Get("pool_id"),
		IdentityID: r.URL.Query().Get("identity_id"),
		ScopeID:    r.URL.Query().Get("scope_id"),
	}

	// Query store
	stats, err := s.store.GetUsageStats(r.Context(), filter)
	if err != nil {
		fmt.Printf(`{"level":"error","msg":"failed_to_get_usage_stats","trace_id":"%s","error":"%v"}`+"\n", getTraceID(r.Context()), err)
		http.Error(w, `{"error":"internal_server_error"}`, http.StatusInternalServerError)
		return
	}

	// Return JSON
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(stats); err != nil {
		fmt.Printf(`{"level":"error","msg":"failed_to_encode_trends","trace_id":"%s","error":"%v"}`+"\n", getTraceID(r.Context()), err)
	}
}

// handleGraph returns the current constraint graph.
func (s *Server) handleGraph(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method_not_allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	if s.graph == nil {
		http.Error(w, `{"error":"graph_not_available"}`, http.StatusServiceUnavailable)
		return
	}

	g := s.graph.GetGraph()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(g); err != nil {
		fmt.Printf(`{"level":"error","msg":"failed_to_encode_graph","trace_id":"%s","error":"%v"}`+"\n", getTraceID(r.Context()), err)
	}
}

// handleReports generates and streams reports.
func (s *Server) handleReports(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method_not_allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	// Parse parameters
	q := r.URL.Query()
	reportType := reports.ReportType(q.Get("type"))
	if reportType == "" {
		http.Error(w, `{"error":"missing_type"}`, http.StatusBadRequest)
		return
	}

	fromStr := q.Get("from")
	toStr := q.Get("to")

	// Default time range: last 24h if not specified
	to := time.Now()
	if toStr != "" {
		var err error
		to, err = time.Parse(time.RFC3339, toStr)
		if err != nil {
			http.Error(w, `{"error":"invalid_to","format":"RFC3339"}`, http.StatusBadRequest)
			return
		}
	}

	from := to.Add(-24 * time.Hour)
	if fromStr != "" {
		var err error
		from, err = time.Parse(time.RFC3339, fromStr)
		if err != nil {
			http.Error(w, `{"error":"invalid_from","format":"RFC3339"}`, http.StatusBadRequest)
			return
		}
	}

	// Build params
	params := reports.ReportParams{
		Start:   from,
		End:     to,
		Filters: make(map[string]interface{}),
	}

	// Pass through filters
	if id := q.Get("identity_id"); id != "" {
		params.Filters["identity_id"] = id
	}
	if sc := q.Get("scope_id"); sc != "" {
		params.Filters["scope_id"] = sc
	}
	if prov := q.Get("provider_id"); prov != "" {
		params.Filters["provider_id"] = prov
	}
	if pool := q.Get("pool_id"); pool != "" {
		params.Filters["pool_id"] = pool
	}
	if bucket := q.Get("bucket"); bucket != "" {
		params.Filters["bucket"] = bucket
	}

	// Create generator
	gen, err := reports.NewReportGenerator(reportType, s.store)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"invalid_report_type","details":"%v"}`, err), http.StatusBadRequest)
		return
	}

	// Generate
	reader, err := gen.Generate(r.Context(), params)
	if err != nil {
		fmt.Printf(`{"level":"error","msg":"failed_to_generate_report","trace_id":"%s","error":"%v"}`+"\n", getTraceID(r.Context()), err)
		http.Error(w, `{"error":"report_generation_failed"}`, http.StatusInternalServerError)
		return
	}

	// Set headers
	w.Header().Set("Content-Type", "text/csv")
	filename := fmt.Sprintf("report_%s_%d.csv", reportType, time.Now().Unix())
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))

	// Stream response
	if _, err := io.Copy(w, reader); err != nil {
		fmt.Printf(`{"level":"error","msg":"failed_to_stream_report","trace_id":"%s","error":"%v"}`+"\n", getTraceID(r.Context()), err)
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
		if s.staticFS == nil {
			http.NotFound(w, r)
			return
		}

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

// Middleware: Auth
func (s *Server) withAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, `{"error":"unauthorized","reason":"missing_token"}`, http.StatusUnauthorized)
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, `{"error":"unauthorized","reason":"invalid_token_format"}`, http.StatusUnauthorized)
			return
		}

		token := parts[1]
		hash := hashToken(token)

		_, ok := s.identities.GetByTokenHash(hash)
		if !ok {
			http.Error(w, `{"error":"unauthorized","reason":"invalid_token"}`, http.StatusUnauthorized)
			return
		}

		next(w, r)
	}
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

func generateToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano()) // Fallback
	}
	return hex.EncodeToString(b)
}

func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

// Middleware: Secure Headers
func withSecureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data:;")
		w.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		next.ServeHTTP(w, r)
	})
}

// Middleware: Leader Check (Redirects writes to leader)
func (s *Server) withLeaderCheck(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Skip check if no election manager configured (standalone mode)
		if s.election == nil {
			next(w, r)
			return
		}

		// Only check for write methods (POST, PUT, PATCH, DELETE)
		if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch || r.Method == http.MethodDelete {
			if s.election.IsLeader() {
				next(w, r)
				return
			}

			// Not leader, find who is
			leaderAddr, ok, err := s.election.GetLeader(r.Context())
			if err != nil {
				// Don't expose internal error details, but log them
				fmt.Printf(`{"level":"error","msg":"failed_to_check_leader","trace_id":"%s","error":"%v"}`+"\n", getTraceID(r.Context()), err)
				http.Error(w, `{"error":"internal_server_error"}`, http.StatusInternalServerError)
				return
			}
			if !ok {
				http.Error(w, `{"error":"service_unavailable","reason":"no_leader_elected"}`, http.StatusServiceUnavailable)
				return
			}

			// Redirect
			// If leaderAddr is "http://host:port", we just append request path
			// Ensure leaderAddr doesn't have trailing slash
			leaderAddr = strings.TrimRight(leaderAddr, "/")
			targetURL := fmt.Sprintf("%s%s", leaderAddr, r.URL.Path)
			if r.URL.RawQuery != "" {
				targetURL += "?" + r.URL.RawQuery
			}

			http.Redirect(w, r, targetURL, http.StatusTemporaryRedirect) // 307
			return
		}

		// Read methods allowed on followers
		next(w, r)
	}
}
