package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rmax-ai/ratelord/pkg/api"
)

func main() {
	var (
		agentsCount int
		duration    time.Duration
		rate        int
		apiURL      string
		sabotage    bool
	)

	flag.IntVar(&agentsCount, "agents", 5, "Number of concurrent agents to simulate")
	flag.DurationVar(&duration, "duration", 30*time.Second, "Duration of the simulation")
	flag.IntVar(&rate, "rate", 2, "Approximate requests per second per agent")
	flag.StringVar(&apiURL, "api", "http://127.0.0.1:8090", "Base URL of ratelord-d API")
	flag.BoolVar(&sabotage, "sabotage", false, "Enable drift sabotage (inject unknown usage)")
	flag.Parse()

	fmt.Printf("Starting simulation with %d agents, duration %s, rate %d/s per agent\n", agentsCount, duration, rate)
	if sabotage {
		fmt.Println("⚠️  SABOTAGE MODE ENABLED: Will inject random unknown usage to simulate drift")
	}

	var (
		totalRequests uint64
		totalApproved uint64
		totalDenied   uint64
		totalModified uint64
		totalErrors   uint64
		totalInjected uint64
	)

	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(agentsCount)

	// Start Saboteur if enabled
	if sabotage {
		wg.Add(1)
		go func() {
			defer wg.Done()
			log.Println("😈 Saboteur started")
			ticker := time.NewTicker(2 * time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					// Inject a large burst
					amount := int64(rand.Intn(500) + 100)
					if err := injectUsage(apiURL, "mock-provider-1", "default", amount); err != nil {
						log.Printf("😈 Saboteur failed: %v", err)
					} else {
						atomic.AddUint64(&totalInjected, uint64(amount))
						log.Printf("😈 Saboteur injected %d units of usage", amount)
					}
				}
			}
		}()
	}

	for i := 0; i < agentsCount; i++ {
		agentID := fmt.Sprintf("sim-agent-%d", i)
		identityID := fmt.Sprintf("service-%d", i)

		go func(aID, iID string) {
			defer wg.Done()

			// 1. Register Identity
			if err := registerIdentity(apiURL, iID); err != nil {
				log.Printf("[%s] Failed to register identity: %v", aID, err)
				return
			}
			log.Printf("[%s] Identity registered: %s", aID, iID)

			// 2. Loop Intents
			ticker := time.NewTicker(time.Second / time.Duration(rate))
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					// Add jitter
					time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)

					req := api.IntentRequest{
						AgentID:    aID,
						IdentityID: iID,
						ScopeID:    "default", // Assuming default scope exists/is implied
						WorkloadID: fmt.Sprintf("job-%d", rand.Intn(1000)),
						Priority:   "normal",
					}

					decision, err := sendIntent(apiURL, req)
					atomic.AddUint64(&totalRequests, 1)

					if err != nil {
						log.Printf("[%s] Error sending intent: %v", aID, err)
						atomic.AddUint64(&totalErrors, 1)
						continue
					}

					if decision.Decision == "approve" {
						atomic.AddUint64(&totalApproved, 1)
					} else if decision.Decision == "approve_with_modifications" {
						atomic.AddUint64(&totalModified, 1)
						// If wait_seconds is in modification, sleep
						if mods := decision.Modifications; mods != nil {
							if val, ok := mods["wait_seconds"]; ok {
								var seconds float64
								fmt.Sscanf(val, "%fs", &seconds)
								// Actually wait to simulate compliance
								time.Sleep(time.Duration(seconds * float64(time.Second)))
							}
						}
					} else {
						atomic.AddUint64(&totalDenied, 1)
						// log.Printf("[%s] Intent denied: %s", aID, decision.Reason)
					}
				}
			}
		}(agentID, identityID)
	}

	wg.Wait()

	fmt.Println("\n--- Simulation Complete ---")
	fmt.Printf("Total Requests: %d\n", totalRequests)
	fmt.Printf("Approved:       %d\n", totalApproved)
	fmt.Printf("Modified:       %d\n", totalModified)
	fmt.Printf("Denied:         %d\n", totalDenied)
	fmt.Printf("Errors:         %d\n", totalErrors)
	if sabotage {
		fmt.Printf("Injected Usage: %d\n", totalInjected)
	}
}

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
