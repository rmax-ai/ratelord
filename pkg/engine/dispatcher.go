package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/rmax-ai/ratelord/pkg/store"
)

const (
	// CursorKey is the key used in system_state to store the last processed event timestamp.
	CursorKey = "webhook_dispatcher_cursor"
	// BatchSize is the number of events to fetch per poll.
	BatchSize = 50
	// PollInterval is how often to check for new events.
	PollInterval = 1 * time.Second
	// DefaultTimeout is the HTTP client timeout for webhook requests.
	DefaultTimeout = 5 * time.Second
	// MaxRetries is the number of delivery attempts.
	MaxRetries = 3
)

// Dispatcher handles the delivery of events to registered webhooks.
type Dispatcher struct {
	store      *store.Store
	client     *http.Client
	pollTicker *time.Ticker
}

// NewDispatcher creates a new webhook dispatcher.
func NewDispatcher(s *store.Store) *Dispatcher {
	return &Dispatcher{
		store: s,
		client: &http.Client{
			Timeout: DefaultTimeout,
		},
		pollTicker: time.NewTicker(PollInterval),
	}
}

// Start begins the event polling and dispatch loop.
// It blocks until the context is cancelled.
func (d *Dispatcher) Start(ctx context.Context) {
	log.Println("Starting Webhook Dispatcher...")

	// Load initial cursor
	cursor, err := d.loadCursor(ctx)
	if err != nil {
		log.Printf("Failed to load dispatcher cursor: %v. Defaulting to now.", err)
		cursor = time.Now().UTC()
	}

	for {
		select {
		case <-ctx.Done():
			log.Println("Stopping Webhook Dispatcher...")
			d.pollTicker.Stop()
			return
		case <-d.pollTicker.C:
			// Poll for events
			newCursor, count, err := d.processBatch(ctx, cursor)
			if err != nil {
				log.Printf("Error processing webhook batch: %v", err)
				continue
			}
			// If we processed events, update cursor
			if count > 0 {
				cursor = newCursor
				if err := d.saveCursor(ctx, cursor); err != nil {
					log.Printf("Failed to save dispatcher cursor: %v", err)
				}
			}
		}
	}
}

// processBatch fetches and processes a batch of events.
// Returns the timestamp of the last processed event, the count of events, and any error.
func (d *Dispatcher) processBatch(ctx context.Context, since time.Time) (time.Time, int, error) {
	events, err := d.store.ReadEvents(ctx, since, BatchSize)
	if err != nil {
		return since, 0, err
	}

	if len(events) == 0 {
		return since, 0, nil
	}

	// Fetch active webhooks
	// Optimization: This could be cached and refreshed periodically.
	webhooks, err := d.store.ListWebhooks(ctx)
	if err != nil {
		return since, 0, fmt.Errorf("failed to list webhooks: %w", err)
	}

	// Filter for active webhooks only
	var activeWebhooks []*store.WebhookConfig
	for _, w := range webhooks {
		if w.Active {
			activeWebhooks = append(activeWebhooks, w)
		}
	}

	if len(activeWebhooks) == 0 {
		// No listeners, just advance cursor
		lastEvent := events[len(events)-1]
		return lastEvent.TsIngest, len(events), nil
	}

	lastTs := since
	for _, evt := range events {
		d.dispatchToWebhooks(ctx, evt, activeWebhooks)
		lastTs = evt.TsIngest
	}

	return lastTs, len(events), nil
}

// dispatchToWebhooks sends the event to all interested webhooks.
func (d *Dispatcher) dispatchToWebhooks(ctx context.Context, evt *store.Event, webhooks []*store.WebhookConfig) {
	for _, wh := range webhooks {
		if d.shouldDispatch(wh, evt) {
			// Dispatch async or sync? Sync for now to ensure reliability before advancing cursor.
			// Parallelizing per webhook is a good optimization for later.
			if err := d.send(ctx, wh, evt); err != nil {
				log.Printf("Failed to dispatch event %s to webhook %s: %v", evt.EventID, wh.WebhookID, err)
			}
		}
	}
}

// shouldDispatch checks if the webhook is interested in the event.
func (d *Dispatcher) shouldDispatch(wh *store.WebhookConfig, evt *store.Event) bool {
	for _, interestedType := range wh.Events {
		if interestedType == "*" || interestedType == string(evt.EventType) {
			return true
		}
	}
	return false
}

// send performs the HTTP POST with retries.
func (d *Dispatcher) send(ctx context.Context, wh *store.WebhookConfig, evt *store.Event) error {
	payload, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	var lastErr error
	for i := 0; i < MaxRetries; i++ {
		if i > 0 {
			// Linear backoff: 1s, 2s, 3s
			time.Sleep(time.Duration(i) * time.Second)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, wh.URL, bytes.NewBuffer(payload))
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "ratelord-dispatcher/1.0")
		req.Header.Set("X-Ratelord-Event-ID", string(evt.EventID))
		req.Header.Set("X-Ratelord-Event-Type", string(evt.EventType))

		// TODO: Add HMAC signature (M26.3)
		// req.Header.Set("X-Ratelord-Signature", signature)

		resp, err := d.client.Do(req)
		if err != nil {
			lastErr = err
			continue // Retry on network error
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return nil // Success
		}

		lastErr = fmt.Errorf("webhook responded with status: %d", resp.StatusCode)
		// Retry on 5xx, give up on 4xx?
		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			return lastErr // Don't retry client errors
		}
	}

	return fmt.Errorf("max retries reached: %w", lastErr)
}

// loadCursor retrieves the last processed timestamp from system_state.
func (d *Dispatcher) loadCursor(ctx context.Context) (time.Time, error) {
	val, err := d.store.GetSystemState(ctx, CursorKey)
	if err != nil {
		return time.Time{}, err
	}
	return time.Parse(time.RFC3339Nano, val)
}

// saveCursor persists the last processed timestamp.
func (d *Dispatcher) saveCursor(ctx context.Context, t time.Time) error {
	return d.store.SetSystemState(ctx, CursorKey, t.Format(time.RFC3339Nano))
}
