package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rmax-ai/ratelord/pkg/provider"
)

type GitHubProvider struct {
	id            provider.ProviderID
	token         string
	enterpriseURL string
	client        *http.Client
}

func NewGitHubProvider(id provider.ProviderID, token string, enterpriseURL string) *GitHubProvider {
	return &GitHubProvider{
		id:            id,
		token:         token,
		enterpriseURL: enterpriseURL,
		client:        &http.Client{Timeout: 10 * time.Second},
	}
}

func (g *GitHubProvider) ID() provider.ProviderID {
	return g.id
}

func (g *GitHubProvider) Poll(ctx context.Context) (provider.PollResult, error) {
	baseURL := "https://api.github.com"
	if g.enterpriseURL != "" {
		baseURL = g.enterpriseURL
	}
	url := baseURL + "/rate_limit"

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return provider.PollResult{}, err
	}
	if g.token != "" {
		req.Header.Set("Authorization", "token "+g.token)
	}

	resp, err := g.client.Do(req)
	if err != nil {
		return provider.PollResult{ProviderID: g.id, Status: "error", Error: err, Timestamp: time.Now()}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return provider.PollResult{ProviderID: g.id, Status: "error", Error: fmt.Errorf("HTTP %d", resp.StatusCode), Timestamp: time.Now()}, nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return provider.PollResult{ProviderID: g.id, Status: "error", Error: err, Timestamp: time.Now()}, nil
	}

	var rateLimitResp struct {
		Resources map[string]struct {
			Limit     int   `json:"limit"`
			Remaining int   `json:"remaining"`
			Reset     int64 `json:"reset"`
		} `json:"resources"`
	}

	err = json.Unmarshal(body, &rateLimitResp)
	if err != nil {
		return provider.PollResult{ProviderID: g.id, Status: "error", Error: err, Timestamp: time.Now()}, nil
	}

	var usages []provider.UsageObservation
	resourceMap := map[string]string{
		"core":                 "github:core",
		"search":               "github:search",
		"graphql":              "github:graphql",
		"integration_manifest": "github:integration_manifest",
	}

	for res, data := range rateLimitResp.Resources {
		if poolID, ok := resourceMap[res]; ok {
			usages = append(usages, provider.UsageObservation{
				PoolID:    poolID,
				Used:      int64(data.Limit - data.Remaining),
				Remaining: int64(data.Remaining),
				Limit:     int64(data.Limit),
				ResetAt:   time.Unix(data.Reset, 0),
			})
		}
	}

	return provider.PollResult{
		ProviderID: g.id,
		Status:     "success",
		Timestamp:  time.Now(),
		Usage:      usages,
		State:      nil, // stateless
	}, nil
}

func (g *GitHubProvider) Restore(state []byte) error {
	// No-op, stateless
	return nil
}
