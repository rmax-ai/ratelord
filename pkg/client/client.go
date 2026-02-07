package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Client is the ratelord SDK client.
type Client struct {
	endpoint string
	http     *http.Client
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
	}
}

// Ask sends an intent to the daemon and returns a decision.
// It implements the "Ask-Wait-Act" pattern by automatically sleeping if required.
// It is fail-closed: network errors return a Denied decision.
func (c *Client) Ask(ctx context.Context, intent Intent) (Decision, error) {
	// 1. Validate mandatory fields
	if intent.AgentID == "" || intent.IdentityID == "" || intent.WorkloadID == "" || intent.ScopeID == "" {
		return Decision{}, fmt.Errorf("invalid intent: missing required fields")
	}

	// 2. Serialize Intent
	body, err := json.Marshal(intent)
	if err != nil {
		return Decision{}, fmt.Errorf("failed to marshal intent: %w", err)
	}

	// 3. Create Request
	req, err := http.NewRequestWithContext(ctx, "POST", c.endpoint+"/v1/intent", bytes.NewReader(body))
	if err != nil {
		return failClosed("request_creation_failed"), nil // Fail-closed, return denied decision
	}
	req.Header.Set("Content-Type", "application/json")

	// 4. Send Request (Handle Network Errors as Fail-Closed)
	resp, err := c.http.Do(req)
	if err != nil {
		return failClosed("daemon_unreachable"), nil
	}
	defer resp.Body.Close()

	// 5. Handle HTTP Status Codes
	if resp.StatusCode >= 500 {
		return failClosed("upstream_error"), nil
	}
	if resp.StatusCode == 400 {
		return Decision{}, fmt.Errorf("invalid_intent: bad request from daemon")
	}
	if resp.StatusCode != 200 {
		return failClosed(fmt.Sprintf("unexpected_status_%d", resp.StatusCode)), nil
	}

	// 6. Parse Response
	var decision Decision
	if err := json.NewDecoder(resp.Body).Decode(&decision); err != nil {
		return failClosed("response_parsing_failed"), nil
	}

	// 7. Auto-Wait
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

// failClosed returns a denied decision with a specific reason.
func failClosed(reason string) Decision {
	return Decision{
		Allowed: false,
		Status:  "deny_with_reason",
		Reason:  reason,
	}
}
