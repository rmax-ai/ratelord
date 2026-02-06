package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rmax-ai/ratelord/pkg/api"
)

func main() {
	var (
		scenarioFile string
		apiURL       string
	)

	flag.StringVar(&scenarioFile, "scenario", "", "Path to scenario JSON file")
	flag.StringVar(&apiURL, "api", "http://127.0.0.1:8090", "Base URL of ratelord-d API")
	flag.Parse()

	var scenario Scenario

	if scenarioFile != "" {
		data, err := os.ReadFile(scenarioFile)
		if err != nil {
			log.Fatalf("Failed to read scenario file: %v", err)
		}
		if err := json.Unmarshal(data, &scenario); err != nil {
			log.Fatalf("Failed to parse scenario file: %v", err)
		}
	} else {
		// Default Scenario (Legacy flags support can be mapped here if needed, but keeping it simple)
		fmt.Println("No scenario file provided, running default demo scenario...")
		scenario = Scenario{
			Name:        "Default Demo",
			Duration:    30 * time.Second,
			Description: "Simple periodic load",
			Agents: []AgentConfig{
				{
					Name:       "agent-default",
					Count:      5,
					IdentityID: "service-default",
					Behavior:   BehaviorPeriodic,
					Rate:       2,
				},
			},
		}
	}

	runScenario(scenario, apiURL)
}

func runScenario(s Scenario, apiURL string) {
	fmt.Printf("Running Scenario: %s\n", s.Name)
	fmt.Printf("Description: %s\n", s.Description)
	fmt.Printf("Duration: %s\n", s.Duration)

	if s.Seed == 0 {
		s.Seed = time.Now().UnixNano()
		fmt.Printf("Random Seed: %d\n", s.Seed)
	} else {
		fmt.Printf("Fixed Seed: %d\n", s.Seed)
	}

	// Global seed for fallback (though we try to use local RNGs)
	rand.Seed(s.Seed)

	ctx, cancel := context.WithTimeout(context.Background(), s.Duration)
	defer cancel()

	var (
		totalRequests uint64
		totalApproved uint64
		totalDenied   uint64
		totalModified uint64
		totalErrors   uint64
		totalInjected uint64
	)

	var wg sync.WaitGroup

	// Start Saboteur
	if s.Sabotage != nil && s.Sabotage.Enabled {
		wg.Add(1)
		go func() {
			defer wg.Done()
			log.Println("😈 Saboteur started")
			ticker := time.NewTicker(s.Sabotage.Interval)
			defer ticker.Stop()

			// Saboteur needs its own RNG
			sabotageRng := rand.New(rand.NewSource(s.Seed + 9999))

			parts := strings.Split(s.Sabotage.Target, "/")
			providerID := "mock-provider-1"
			poolID := "default"
			if len(parts) == 2 {
				providerID = parts[0]
				poolID = parts[1]
			}

			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					// Add some randomness to amount if needed, currently fixed
					if err := injectUsage(apiURL, providerID, poolID, s.Sabotage.Amount); err != nil {
						log.Printf("😈 Saboteur failed: %v", err)
					} else {
						atomic.AddUint64(&totalInjected, uint64(s.Sabotage.Amount))
						log.Printf("😈 Saboteur injected %d units", s.Sabotage.Amount)
					}
					// Consume RNG to advance state if we add randomness later
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
			// Deterministic seed per agent: Base + AgentGroupIndex*1000 + InstanceIndex
			agentSeed := s.Seed + int64(agentIdx*1000) + int64(i)

			go func(cfg AgentConfig, aID string, seed int64) {
				defer wg.Done()
				runAgent(ctx, apiURL, aID, cfg, seed, &totalRequests, &totalApproved, &totalDenied, &totalModified, &totalErrors)
			}(agentCfg, agentID, agentSeed)
		}
	}

	wg.Wait()

	fmt.Println("\n--- Simulation Complete ---")
	fmt.Printf("Total Requests: %d\n", totalRequests)
	fmt.Printf("Approved:       %d\n", totalApproved)
	fmt.Printf("Modified:       %d\n", totalModified)
	fmt.Printf("Denied:         %d\n", totalDenied)
	fmt.Printf("Errors:         %d\n", totalErrors)
	if s.Sabotage != nil && s.Sabotage.Enabled {
		fmt.Printf("Injected Usage: %d\n", totalInjected)
	}
}

func runAgent(ctx context.Context, apiURL, agentID string, cfg AgentConfig, seed int64, reqs, app, den, mod, errs *uint64) {
	rng := rand.New(rand.NewSource(seed))

	// 1. Register
	if err := registerIdentity(apiURL, cfg.IdentityID); err != nil {
		log.Printf("[%s] Failed to register: %v", agentID, err)
		return
	}

	// 2. Behavior Loop
	switch cfg.Behavior {
	case BehaviorGreedy:
		for {
			select {
			case <-ctx.Done():
				return
			default:
				doRequest(apiURL, agentID, cfg.IdentityID, rng, reqs, app, den, mod, errs)
			}
		}
	case BehaviorPoisson:
		lambda := float64(cfg.Rate)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				// Inter-arrival time for Poisson process is exponential
				// interval = -ln(U) / lambda
				interval := -math.Log(rng.Float64()) / lambda
				time.Sleep(time.Duration(interval * float64(time.Second)))
				doRequest(apiURL, agentID, cfg.IdentityID, rng, reqs, app, den, mod, errs)
			}
		}
	case BehaviorBursty:
		// Example: sleep 90% of time, burst in 10%
		// Or: wait N seconds, then fire M requests
		ticker := time.NewTicker(time.Second) // 1 burst per second
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				for k := 0; k < cfg.Burst; k++ {
					doRequest(apiURL, agentID, cfg.IdentityID, rng, reqs, app, den, mod, errs)
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
				doRequest(apiURL, agentID, cfg.IdentityID, rng, reqs, app, den, mod, errs)
			}
		}
	}
}

func doRequest(apiURL, agentID, identityID string, rng *rand.Rand, reqs, app, den, mod, errs *uint64) {
	req := api.IntentRequest{
		AgentID:    agentID,
		IdentityID: identityID,
		ScopeID:    "default",
		WorkloadID: fmt.Sprintf("job-%d", rng.Intn(1000)),
		Priority:   "normal",
	}

	decision, err := sendIntent(apiURL, req)
	atomic.AddUint64(reqs, 1)

	if err != nil {
		log.Printf("[%s] Error: %v", agentID, err)
		atomic.AddUint64(errs, 1)
		return
	}

	if decision.Decision == "approve" {
		atomic.AddUint64(app, 1)
	} else if decision.Decision == "approve_with_modifications" {
		atomic.AddUint64(mod, 1)
		handleModifications(agentID, decision.Modifications)
	} else {
		atomic.AddUint64(den, 1)
	}
}

func handleModifications(agentID string, mods map[string]interface{}) {
	if val, ok := mods["wait_seconds"]; ok {
		var seconds float64
		switch v := val.(type) {
		case float64:
			seconds = v
		case string:
			fmt.Sscanf(v, "%fs", &seconds)
		}

		if seconds > 0 {
			// log.Printf("[%s] Throttled for %.2fs", agentID, seconds)
			time.Sleep(time.Duration(seconds * float64(time.Second)))
		}
	}
}

// Helpers (Same as before but cleaned up)

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

func registerIdentity(baseURL, identityID string) error {
	reg := api.IdentityRegistration{
		IdentityID: identityID,
		Kind:       "simulated-service",
		Metadata:   map[string]interface{}{"env": "simulation"},
	}

	body, _ := json.Marshal(reg)
	resp, err := http.Post(baseURL+"/v1/identities", "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("bad status: %d", resp.StatusCode)
	}
	return nil
}

func sendIntent(baseURL string, req api.IntentRequest) (*api.DecisionResponse, error) {
	body, _ := json.Marshal(req)
	resp, err := http.Post(baseURL+"/v1/intent", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status: %d", resp.StatusCode)
	}

	var decision api.DecisionResponse
	if err := json.NewDecoder(resp.Body).Decode(&decision); err != nil {
		return nil, err
	}
	return &decision, nil
}
