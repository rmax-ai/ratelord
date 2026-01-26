package openai

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/rmax-ai/ratelord/pkg/provider"
)

type OpenAIProvider struct {
	id      provider.ProviderID
	token   string
	orgID   string
	baseURL string
	client  *http.Client
}

func NewOpenAIProvider(id provider.ProviderID, token string, orgID string, baseURL string) *OpenAIProvider {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	return &OpenAIProvider{
		id:      id,
		token:   token,
		orgID:   orgID,
		baseURL: baseURL,
		client:  &http.Client{Timeout: 10 * time.Second},
	}
}

func (o *OpenAIProvider) ID() provider.ProviderID {
	return o.id
}

// Poll performs a lightweight request (List Models) to capture rate limit headers.
// Note: This consumes a small amount of quota/requests itself.
func (o *OpenAIProvider) Poll(ctx context.Context) (provider.PollResult, error) {
	// "List Models" is generally cheap/cached and good for checking connectivity + headers.
	url := fmt.Sprintf("%s/models", strings.TrimRight(o.baseURL, "/"))

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return provider.PollResult{}, err
	}
	if o.token != "" {
		req.Header.Set("Authorization", "Bearer "+o.token)
	}
	if o.orgID != "" {
		req.Header.Set("OpenAI-Organization", o.orgID)
	}

	resp, err := o.client.Do(req)
	if err != nil {
		return provider.PollResult{ProviderID: o.id, Status: "error", Error: err, Timestamp: time.Now()}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Even if error, headers might be present, but usually we want a clean success for a poll.
		// For 429 specifically, we definitely want the headers, but let's stick to standard flow for now.
		// If 429, the headers are definitely there.
		if resp.StatusCode != http.StatusTooManyRequests {
			return provider.PollResult{ProviderID: o.id, Status: "error", Error: fmt.Errorf("HTTP %d", resp.StatusCode), Timestamp: time.Now()}, nil
		}
	}

	var usages []provider.UsageObservation

	// OpenAI headers usually come in pairs: requests and tokens.
	// We map them to pool IDs like "openai:requests" and "openai:tokens".
	// Sometimes they differ by model (e.g. gpt-4 vs gpt-3.5), but the global/org headers are often generic.
	// Recent OpenAI API updates might return headers specific to the requested resource model.
	// Since we are calling /models, these limits might be "global" or specific to management API.
	// However, usually the main inference limits are what we care about.
	// If /models returns general limits, we use them. If not, this "Probe" strategy might need
	// to hit a cheap chat completion with max_tokens=1.
	//
	// Let's assume for this v1 implementation that standard headers are returned.
	// x-ratelimit-limit-requests: 5000
	// x-ratelimit-remaining-requests: 4999
	// x-ratelimit-reset-requests: 100ms
	//
	// x-ratelimit-limit-tokens: 160000
	// x-ratelimit-remaining-tokens: 159000
	// x-ratelimit-reset-tokens: 2s

	extract := func(metric string, poolSuffix string) {
		limitStr := resp.Header.Get(fmt.Sprintf("x-ratelimit-limit-%s", metric))
		remStr := resp.Header.Get(fmt.Sprintf("x-ratelimit-remaining-%s", metric))
		resetStr := resp.Header.Get(fmt.Sprintf("x-ratelimit-reset-%s", metric))

		if limitStr != "" && remStr != "" {
			limit, _ := strconv.ParseInt(limitStr, 10, 64)
			rem, _ := strconv.ParseInt(remStr, 10, 64)

			// OpenAI Reset string is often like "100ms" or "2s" or "6m0s"
			// Go's time.ParseDuration handles this well.
			resetDur, err := time.ParseDuration(resetStr)
			resetAt := time.Now()
			if err == nil {
				resetAt = resetAt.Add(resetDur)
			} else {
				// Fallback: sometimes it might be seconds-integer? Unlikely for OpenAI modern API.
				// If parse fails, use Now (conservative).
				resetAt = time.Now()
			}

			usages = append(usages, provider.UsageObservation{
				PoolID:    fmt.Sprintf("openai:%s", poolSuffix),
				Used:      limit - rem,
				Remaining: rem,
				Limit:     limit,
				ResetAt:   resetAt,
			})
		}
	}

	extract("requests", "requests")
	extract("tokens", "tokens")

	return provider.PollResult{
		ProviderID: o.id,
		Status:     "success",
		Timestamp:  time.Now(),
		Usage:      usages,
		State:      nil, // stateless
	}, nil
}

func (o *OpenAIProvider) Restore(state []byte) error {
	// No-op
	return nil
}
