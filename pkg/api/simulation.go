package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/rmax-ai/ratelord/pkg/simulation"
)

// handleSimulation executes a simulation scenario.
func (s *Server) handleSimulation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method_not_allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var scenario simulation.Scenario
	if err := json.NewDecoder(r.Body).Decode(&scenario); err != nil {
		http.Error(w, `{"error":"invalid_json_body"}`, http.StatusBadRequest)
		return
	}

	// Validate minimal requirements
	if scenario.Duration == 0 {
		http.Error(w, `{"error":"missing_duration"}`, http.StatusBadRequest)
		return
	}

	// Determine local API URL
	// s.server.Addr is typically ":8090" or "0.0.0.0:8090"
	port := "8090" // default
	if s.server != nil && s.server.Addr != "" {
		parts := strings.Split(s.server.Addr, ":")
		if len(parts) > 1 {
			port = parts[len(parts)-1]
		}
	}

	// We use 127.0.0.1 to ensure agents hit the local instance
	apiURL := fmt.Sprintf("http://127.0.0.1:%s", port)

	// Log start
	fmt.Printf(`{"level":"info","msg":"starting_simulation","scenario":"%s","duration":"%s"}`+"\n",
		scenario.Name, scenario.Duration)

	// Run Simulation (Synchronous for now, as per requirements)
	result := simulation.RunScenario(scenario, apiURL)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(result); err != nil {
		fmt.Printf(`{"level":"error","msg":"failed_to_encode_simulation_result","error":"%v"}`+"\n", err)
	}
}
