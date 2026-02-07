package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/rmax-ai/ratelord/pkg/graph"
	"net/http"
	"time"
)

// Client is the ratelord SDK client.
type Client struct {
	endpoint string
	http     *http.Client
	backoff  BackoffStrategy
	retries  int
}

// NewClient creates a new ratelord client.
// endpoint defaults to "http://127.0.0.1:8090" if empty.
func NewClient(endpoint string) *Client {
	if endpoint == "" {
		endpoint = "http://127.0.0.1:8090"
	}
	return &Client{
		endpoint: endpoint,
		http: &http.Client{
			Timeout: 10 * time.Second,
		},
		backoff: DefaultBackoff(),
		retries: 3,
	}
}

// SetBackoff configures the retry strategy.
func (c *Client) SetBackoff(strategy BackoffStrategy, maxRetries int) {
	c.backoff = strategy
	c.retries = maxRetries
}

// Ask sends an intent to the daemon and returns a decision.
// It implements the "Ask-Wait-Act" pattern by automatically sleeping if required.
// It performs retries on network errors or 5xx responses using the configured backoff strategy.
// It is fail-closed: final failure returns a Denied decision.
func (c *Client) Ask(ctx context.Context, intent Intent) (Decision, error) {
	// 1. Validate mandatory fields
	if intent.AgentID == "" || intent.IdentityID == "" || intent.WorkloadID == "" || intent.ScopeID == "" {
		return Decision{}, fmt.Errorf("invalid intent: missing required fields")
	}

	// 2. Serialize Intent (once)
	body, err := json.Marshal(intent)
	if err != nil {
		return Decision{}, fmt.Errorf("failed to marshal intent: %w", err)
	}

	var lastErr error
	var lastStatus int

	// Retry Loop
	for attempt := 0; attempt <= c.retries; attempt++ {
		if attempt > 0 {
			// Wait before retry
			waitDuration := c.backoff.Next(attempt - 1)
			select {
			case <-time.After(waitDuration):
			case <-ctx.Done():
				return failClosed("context_canceled_during_retry"), ctx.Err()
			}
		}

		// 3. Create Request (must be fresh for each attempt if Body is read? No, bytes.NewReader is seekable but http.NewRequest might need care.
		// Actually, bytes.NewReader is efficient. We can recreate the request.)
		req, err := http.NewRequestWithContext(ctx, "POST", c.endpoint+"/v1/intent", bytes.NewReader(body))
		if err != nil {
			return failClosed("request_creation_failed"), nil
		}
		req.Header.Set("Content-Type", "application/json")

		// 4. Send Request
		resp, err := c.http.Do(req)
		if err != nil {
			lastErr = err
			continue // Network error -> Retry
		}

		// 5. Handle HTTP Status Codes
		lastStatus = resp.StatusCode
		if resp.StatusCode >= 500 {
			resp.Body.Close()
			continue // Server error -> Retry
		}

		// Non-retryable status codes
		defer resp.Body.Close()

		if resp.StatusCode == 400 {
			return Decision{}, fmt.Errorf("invalid_intent: bad request from daemon")
		}
		if resp.StatusCode != 200 {
			return failClosed(fmt.Sprintf("unexpected_status_%d", resp.StatusCode)), nil
		}

		// 6. Parse Response (Success)
		var decision Decision
		if err := json.NewDecoder(resp.Body).Decode(&decision); err != nil {
			// If JSON is malformed, maybe we shouldn't retry? Or maybe we should?
			// Let's assume malformed 200 OK is fatal or weird enough not to retry blindly.
			return failClosed("response_parsing_failed"), nil
		}

		// 7. Auto-Wait (on the successful decision)
		if decision.Modifications.WaitSeconds > 0 {
			select {
			case <-time.After(time.Duration(decision.Modifications.WaitSeconds * float64(time.Second))):
				// Wait completed
			case <-ctx.Done():
				return failClosed("context_canceled_during_wait"), ctx.Err()
			}
		}

		return decision, nil
	}

	// If we exhausted retries
	reason := "upstream_unavailable"
	if lastErr != nil {
		reason = "network_error" // simplified
	} else if lastStatus >= 500 {
		reason = "upstream_error"
	}

	// We could log lastErr here if we had a logger
	return failClosed(reason), nil
}

// Ping checks the health of the daemon.
func (c *Client) Ping(ctx context.Context) (Status, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.endpoint+"/health", nil)
	if err != nil {
		return Status{}, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return Status{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return Status{}, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var status Status
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return Status{}, err
	}

	return status, nil
}

// GetEvents fetches recent events from the daemon.
func (c *Client) GetEvents(ctx context.Context, limit int) ([]Event, error) {
	if limit <= 0 {
		limit = 50
	}
	url := fmt.Sprintf("%s/v1/events?limit=%d", c.endpoint, limit)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var events []Event
	if err := json.NewDecoder(resp.Body).Decode(&events); err != nil {
		return nil, err
	}

	return events, nil
}

// GetTrends fetches usage stats based on filters.
func (c *Client) GetTrends(ctx context.Context, opts TrendsOptions) ([]UsageStat, error) {
	url := fmt.Sprintf("%s/v1/trends?bucket=%s", c.endpoint, opts.Bucket)
	if !opts.From.IsZero() {
		url += fmt.Sprintf("&from=%s", opts.From.Format(time.RFC3339))
	}
	if !opts.To.IsZero() {
		url += fmt.Sprintf("&to=%s", opts.To.Format(time.RFC3339))
	}
	if opts.ProviderID != "" {
		url += fmt.Sprintf("&provider_id=%s", opts.ProviderID)
	}
	if opts.PoolID != "" {
		url += fmt.Sprintf("&pool_id=%s", opts.PoolID)
	}
	if opts.IdentityID != "" {
		url += fmt.Sprintf("&identity_id=%s", opts.IdentityID)
	}
	if opts.ScopeID != "" {
		url += fmt.Sprintf("&scope_id=%s", opts.ScopeID)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var stats []UsageStat
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return nil, err
	}

	return stats, nil
}

// GetGraph fetches the current constraint graph.
func (c *Client) GetGraph(ctx context.Context) (*graph.Graph, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.endpoint+"/v1/graph", nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var g graph.Graph
	if err := json.NewDecoder(resp.Body).Decode(&g); err != nil {
		return nil, err
	}

	return &g, nil
}

// failClosed returns a denied decision with a specific reason.
func failClosed(reason string) Decision {
	return Decision{
		Allowed: false,
		Status:  "deny_with_reason",
		Reason:  reason,
	}
}
