package engine

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rmax-ai/ratelord/pkg/store"
)

func TestDispatcher_DispatchEvent(t *testing.T) {
	// Setup temporary SQLite DB
	tmpDir, err := os.MkdirTemp("", "ratelord-dispatcher-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "ratelord.db")
	s, err := store.NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer s.Close()

	// Setup test server to capture webhook requests
	receivedPayload := make(chan []byte, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST request, got %s", r.Method)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("failed to read request body: %v", err)
			return
		}

		// Verify signature
		signature := r.Header.Get("X-Ratelord-Signature")
		if signature == "" {
			t.Errorf("missing X-Ratelord-Signature header")
		}

		// Reconstruct signature
		mac := hmac.New(sha256.New, []byte("test_secret"))
		mac.Write(body)
		expectedSignature := "sha256=" + hex.EncodeToString(mac.Sum(nil))

		if signature != expectedSignature {
			t.Errorf("expected signature %s, got %s", expectedSignature, signature)
		}

		receivedPayload <- body
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Register webhook
	webhook := &store.WebhookConfig{
		WebhookID: "test_webhook",
		URL:       server.URL,
		Secret:    "test_secret",
		Events:    []string{"test_event"},
		CreatedAt: time.Now().UTC(),
		Active:    true,
	}
	if err := s.RegisterWebhook(context.Background(), webhook); err != nil {
		t.Fatalf("failed to register webhook: %v", err)
	}

	// Create and append test event
	testEvent := &store.Event{
		EventID:       "test_event_123",
		EventType:     "test_event",
		SchemaVersion: 1,
		TsEvent:       time.Now().UTC(),
		TsIngest:      time.Now().UTC(),
		Source: store.EventSource{
			OriginKind: "test",
			OriginID:   "test_origin",
			WriterID:   "ratelord-d",
		},
		Dimensions: store.EventDimensions{
			AgentID:    "test_agent",
			IdentityID: "test_identity",
			WorkloadID: "test_workload",
			ScopeID:    "test_scope",
		},
		Correlation: store.EventCorrelation{
			CorrelationID: "test_corr",
			CausationID:   "test_cause",
		},
		Payload: json.RawMessage(`{"test":"data"}`),
	}
	if err := s.AppendEvent(context.Background(), testEvent); err != nil {
		t.Fatalf("failed to append event: %v", err)
	}

	// Set the dispatcher cursor to before the event so it gets picked up
	cursorTime := testEvent.TsIngest.Add(-1 * time.Millisecond)
	if err := s.SetSystemState(context.Background(), "webhook_dispatcher_cursor", cursorTime.Format(time.RFC3339Nano)); err != nil {
		t.Fatalf("failed to set cursor: %v", err)
	}

	// Create dispatcher
	dispatcher := NewDispatcher(s)

	// Start dispatcher in goroutine
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go dispatcher.Start(ctx)

	// Wait for the webhook to be called
	select {
	case payload := <-receivedPayload:
		// Verify the received payload matches the event
		var receivedEvent store.Event
		if err := json.Unmarshal(payload, &receivedEvent); err != nil {
			t.Fatalf("failed to unmarshal received payload: %v", err)
		}
		if receivedEvent.EventID != testEvent.EventID {
			t.Errorf("expected event ID %s, got %s", testEvent.EventID, receivedEvent.EventID)
		}
		if receivedEvent.EventType != testEvent.EventType {
			t.Errorf("expected event type %s, got %s", testEvent.EventType, receivedEvent.EventType)
		}
		if string(receivedEvent.Payload) != string(testEvent.Payload) {
			t.Errorf("expected payload %s, got %s", testEvent.Payload, receivedEvent.Payload)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for webhook request")
	}

	// Cancel context to stop dispatcher
	cancel()

	// Wait a bit for dispatcher to stop
	time.Sleep(100 * time.Millisecond)
}
