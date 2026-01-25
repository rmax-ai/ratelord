package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/rmax/ratelord/pkg/store"
)

// Server encapsulates the HTTP API server
type Server struct {
	store  *store.Store
	server *http.Server
}

// NewServer creates a new API server instance
func NewServer(st *store.Store) *Server {
	mux := http.NewServeMux()

	// Register routes
	mux.HandleFunc("/v1/health", handleHealth)

	// Middleware: Logging & Panic Recovery
	handler := withLogging(withRecovery(mux))

	return &Server{
		store: st,
		server: &http.Server{
			Addr:         "127.0.0.1:8090",
			Handler:      handler,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  15 * time.Second,
		},
	}
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
