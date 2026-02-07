package simulation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rmax-ai/ratelord/pkg/protocol"
)

func RunScenario(s Scenario, apiURL string) SimulationResult {
	if s.Seed == 0 {
		s.Seed = time.Now().UnixNano()
	}
	rand.Seed(s.Seed)

	log.Printf("Running Scenario: %s (Seed: %d)", s.Name, s.Seed)

	ctx, cancel := context.WithTimeout(context.Background(), s.Duration)
	defer cancel()

	res := SimulationResult{
		ScenarioName: s.Name,
		Duration:     s.Duration,
		AgentStats:   make(map[string]*AgentStats),
	}

	// Initialize Stats Map
	var statsMutex sync.Mutex
	getAgentStats := func(name string) *AgentStats {
		statsMutex.Lock()
		defer statsMutex.Unlock()
		if _, ok := res.AgentStats[name]; !ok {
			res.AgentStats[name] = &AgentStats{}
		}
		return res.AgentStats[name]
	}

	var wg sync.WaitGroup

	// Start Saboteur
	if s.Sabotage != nil && s.Sabotage.Enabled {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ticker := time.NewTicker(s.Sabotage.Interval)
			defer ticker.Stop()
			sabotageRng := rand.New(rand.NewSource(s.Seed + 9999))
			parts := strings.Split(s.Sabotage.Target, "/")
			providerID, poolID := "mock-provider-1", "default"
			if len(parts) == 2 {
				providerID, poolID = parts[0], parts[1]
			}

			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					if err := injectUsage(apiURL, providerID, poolID, s.Sabotage.Amount); err == nil {
						atomic.AddUint64(&res.TotalInjected, uint64(s.Sabotage.Amount))
					}
					sabotageRng.Int63()
				}
			}
		}()
	}

	// Start Agents
	for agentIdx, agentCfg := range s.Agents {
		for i := 0; i < agentCfg.Count; i++ {
			wg.Add(1)
			agentID := fmt.Sprintf("%s-%d", agentCfg.Name, i)
			agentSeed := s.Seed + int64(agentIdx*1000) + int64(i)
			stats := getAgentStats(agentCfg.Name) // Group stats by Agent Config Name

			go func(cfg AgentConfig, aID string, seed int64, st *AgentStats) {
				defer wg.Done()
				runAgent(ctx, apiURL, aID, cfg, seed, &res, st)
			}(agentCfg, agentID, agentSeed, stats)
		}
	}

	wg.Wait()

	// Evaluate Invariants
	evaluateInvariants(&res, s.Invariants)

	// Determine overall success
	res.Success = true
	for _, inv := range res.Invariants {
		if !inv.Passed {
			res.Success = false
			break
		}
	}

	return res
}

func runAgent(ctx context.Context, apiURL, agentID string, cfg AgentConfig, seed int64, global *SimulationResult, stats *AgentStats) {
	rng := rand.New(rand.NewSource(seed))

	// Register
	token, err := registerIdentity(apiURL, cfg.IdentityID)
	if err != nil {
		log.Printf("[%s] Failed to register: %v", agentID, err)
		return
	}

	// Helper to track request
	track := func(decision string, err error) {
		atomic.AddUint64(&global.TotalRequests, 1)
		atomic.AddUint64(&stats.Requests, 1)
		if err != nil {
			atomic.AddUint64(&global.TotalErrors, 1)
			atomic.AddUint64(&stats.Errors, 1)
			return
		}
		switch decision {
		case "approve":
			atomic.AddUint64(&global.TotalApproved, 1)
			atomic.AddUint64(&stats.Approved, 1)
		case "approve_with_modifications":
			atomic.AddUint64(&global.TotalModified, 1)
			atomic.AddUint64(&stats.Modified, 1)
		default:
			atomic.AddUint64(&global.TotalDenied, 1)
			atomic.AddUint64(&stats.Denied, 1)
		}
	}

	// Behavior Loop
	action := func() {
		scope := cfg.ScopeID
		if scope == "" {
			scope = "default"
		}
		priority := cfg.Priority
		if priority == "" {
			priority = "normal"
		}
		req := protocol.IntentRequest{
			AgentID:    agentID,
			IdentityID: cfg.IdentityID,
			ScopeID:    scope,
			WorkloadID: fmt.Sprintf("job-%d", rng.Intn(10000)),
			Priority:   priority,
			ClientContext: map[string]interface{}{
				"provider_id": "mock-provider-1",
				"pool_id":     "default",
			},
		}

		resp, err := sendIntent(apiURL, req, token)
		if err != nil {
			track("", err)
			return
		}

		decision := resp.Decision
		track(decision, nil)

		if decision == "approve_with_modifications" {
			handleModifications(resp.Modifications)
		}
	}

	switch cfg.Behavior {
	case BehaviorGreedy:
		for {
			select {
			case <-ctx.Done():
				return
			default:
				action()
			}
		}
	case BehaviorPoisson:
		lambda := float64(cfg.Rate)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				interval := -math.Log(rng.Float64()) / lambda
				time.Sleep(time.Duration(interval * float64(time.Second)))
				action()
			}
		}
	case BehaviorBursty:
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				for k := 0; k < cfg.Burst; k++ {
					action()
				}
			}
		}
	case BehaviorPeriodic:
		fallthrough
	default:
		interval := time.Second / time.Duration(cfg.Rate)
		if interval == 0 {
			interval = time.Millisecond * 10
		}
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if cfg.Jitter > 0 {
					time.Sleep(time.Duration(rng.Int63n(int64(cfg.Jitter))))
				}
				action()
			}
		}
	}
}

func evaluateInvariants(res *SimulationResult, invariants []Invariant) {
	for _, inv := range invariants {
		var actual float64
		var passed bool

		// Determine actual value based on scope
		var stats *AgentStats
		if inv.Scope == "global" || inv.Scope == "" {
			stats = &AgentStats{
				Requests: atomic.LoadUint64(&res.TotalRequests),
				Approved: atomic.LoadUint64(&res.TotalApproved),
				Denied:   atomic.LoadUint64(&res.TotalDenied),
				Errors:   atomic.LoadUint64(&res.TotalErrors),
			}
		} else {
			if s, ok := res.AgentStats[inv.Scope]; ok {
				// We need to read the atomic values from the pointers in the map
				stats = &AgentStats{
					Requests: atomic.LoadUint64(&s.Requests),
					Approved: atomic.LoadUint64(&s.Approved),
					Denied:   atomic.LoadUint64(&s.Denied),
					Errors:   atomic.LoadUint64(&s.Errors),
				}
			} else {
				// Agent not found
				res.Invariants = append(res.Invariants, InvariantResult{
					Metric: inv.Metric, Scope: inv.Scope, Expected: fmt.Sprintf("%s %.2f", inv.Condition, inv.Value), Actual: "N/A", Passed: false,
				})
				continue
			}
		}

		if stats.Requests == 0 {
			actual = 0
		} else {
			switch inv.Metric {
			case "approval_rate":
				actual = float64(stats.Approved) / float64(stats.Requests)
			case "denial_rate":
				actual = float64(stats.Denied) / float64(stats.Requests)
			case "error_rate":
				actual = float64(stats.Errors) / float64(stats.Requests)
			default:
				actual = 0
			}
		}

		switch inv.Condition {
		case ">":
			passed = actual > inv.Value
		case ">=":
			passed = actual >= inv.Value
		case "<":
			passed = actual < inv.Value
		case "<=":
			passed = actual <= inv.Value
		case "==":
			passed = math.Abs(actual-inv.Value) < 0.0001
		}

		res.Invariants = append(res.Invariants, InvariantResult{
			Metric:   inv.Metric,
			Scope:    inv.Scope,
			Expected: fmt.Sprintf("%s %.2f", inv.Condition, inv.Value),
			Actual:   fmt.Sprintf("%.4f", actual),
			Passed:   passed,
		})
	}
}

func handleModifications(mods map[string]interface{}) {
	if val, ok := mods["wait_seconds"]; ok {
		var seconds float64
		switch v := val.(type) {
		case float64:
			seconds = v
		case string:
			fmt.Sscanf(v, "%fs", &seconds)
		}
		if seconds > 0 {
			time.Sleep(time.Duration(seconds * float64(time.Second)))
		}
	}
}

// Reuse helper functions injectUsage, registerIdentity, sendIntent
func injectUsage(baseURL, providerID, poolID string, amount int64) error {
	payload := map[string]interface{}{
		"provider_id": providerID,
		"pool_id":     poolID,
		"amount":      amount,
	}
	body, _ := json.Marshal(payload)
	resp, err := http.Post(baseURL+"/debug/provider/inject", "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %d", resp.StatusCode)
	}
	return nil
}

func registerIdentity(baseURL, identityID string) (string, error) {
	reg := protocol.IdentityRegistration{
		IdentityID: identityID,
		Kind:       "simulated-service",
		Metadata:   map[string]interface{}{"env": "simulation"},
	}
	body, _ := json.Marshal(reg)
	resp, err := http.Post(baseURL+"/v1/identities", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("bad status: %d", resp.StatusCode)
	}

	var identityResp protocol.IdentityResponse
	if err := json.NewDecoder(resp.Body).Decode(&identityResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %v", err)
	}

	return identityResp.Token, nil
}

func sendIntent(baseURL string, req protocol.IntentRequest, token string) (*protocol.DecisionResponse, error) {
	body, _ := json.Marshal(req)
	r, err := http.NewRequest(http.MethodPost, baseURL+"/v1/intent", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	r.Header.Set("Content-Type", "application/json")
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status: %d", resp.StatusCode)
	}
	var decision protocol.DecisionResponse
	if err := json.NewDecoder(resp.Body).Decode(&decision); err != nil {
		return nil, err
	}
	return &decision, nil
}
