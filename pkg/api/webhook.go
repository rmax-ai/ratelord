package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/rmax-ai/ratelord/pkg/store"
)

// handleWebhooks manages webhook registration.
func (s *Server) handleWebhooks(w http.ResponseWriter, r *http.Request) {
	// Only POST is required for M26.1, but structure allows expansion
	if r.Method == http.MethodPost {
		s.createWebhook(w, r)
		return
	}
	http.Error(w, `{"error":"method_not_allowed"}`, http.StatusMethodNotAllowed)
}

func (s *Server) createWebhook(w http.ResponseWriter, r *http.Request) {
	var req struct {
		URL    string   `json:"url"`
		Events []string `json:"events"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid_json_body"}`, http.StatusBadRequest)
		return
	}

	if req.URL == "" {
		http.Error(w, `{"error":"missing_url"}`, http.StatusBadRequest)
		return
	}

	// Auto-generate ID and Secret
	webhookID := "wh_" + fmt.Sprintf("%d", time.Now().UnixNano())
	secret := generateToken() // Reuse helper from server.go

	cfg := &store.WebhookConfig{
		WebhookID: webhookID,
		URL:       req.URL,
		Secret:    secret,
		Events:    req.Events,
		CreatedAt: time.Now().UTC(),
		Active:    true,
	}

	if err := s.store.RegisterWebhook(r.Context(), cfg); err != nil {
		fmt.Printf(`{"level":"error","msg":"failed_to_register_webhook","error":"%v"}`+"\n", err)
		http.Error(w, `{"error":"internal_server_error"}`, http.StatusInternalServerError)
		return
	}

	resp := struct {
		WebhookID string `json:"webhook_id"`
		Secret    string `json:"secret"`
	}{
		WebhookID: webhookID,
		Secret:    secret,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		fmt.Printf(`{"level":"error","msg":"failed_to_encode_response","error":"%v"}`+"\n", err)
	}
}
