package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"sync/atomic"
	"time"

	"github.com/rmax-ai/ratelord/pkg/simulation"
)

func main() {
	var (
		scenarioFile string
		apiURL       string
		jsonOutput   bool
		outputFile   string
	)

	flag.StringVar(&scenarioFile, "scenario", "", "Path to scenario JSON file")
	flag.StringVar(&apiURL, "api", "http://127.0.0.1:8090", "Base URL of ratelord-d API")
	flag.BoolVar(&jsonOutput, "json", false, "Output results as JSON")
	flag.StringVar(&outputFile, "out", "", "Write output to file instead of stdout")
	flag.Parse()

	var scenario simulation.Scenario

	if scenarioFile != "" {
		data, err := os.ReadFile(scenarioFile)
		if err != nil {
			log.Fatalf("Failed to read scenario file: %v", err)
		}
		if err := json.Unmarshal(data, &scenario); err != nil {
			log.Fatalf("Failed to parse scenario file: %v", err)
		}
	} else {
		// Default Scenario
		fmt.Fprintln(os.Stderr, "No scenario file provided, running default demo scenario...")
		scenario = simulation.Scenario{
			Name:        "Default Demo",
			Duration:    10 * time.Second,
			Description: "Simple periodic load",
			Agents: []simulation.AgentConfig{
				{
					Name:       "agent-default",
					Count:      5,
					IdentityID: "service-default",
					Behavior:   simulation.BehaviorPeriodic,
					Rate:       2,
				},
			},
		}
	}

	result := simulation.RunScenario(scenario, apiURL)

	writeReport(result, jsonOutput, outputFile)

	if !result.Success {
		os.Exit(1)
	}
}

func writeReport(res simulation.SimulationResult, jsonFmt bool, filePath string) {
	var output []byte
	var err error

	if jsonFmt {
		output, err = json.MarshalIndent(res, "", "  ")
	} else {
		var buf bytes.Buffer
		buf.WriteString(fmt.Sprintf("\n--- Simulation Report: %s ---\n", res.ScenarioName))
		buf.WriteString(fmt.Sprintf("Duration: %s\n", res.Duration))
		buf.WriteString(fmt.Sprintf("Requests: %d | Approved: %d | Denied: %d | Errors: %d\n",
			atomic.LoadUint64(&res.TotalRequests),
			atomic.LoadUint64(&res.TotalApproved),
			atomic.LoadUint64(&res.TotalDenied),
			atomic.LoadUint64(&res.TotalErrors)))

		if len(res.Invariants) > 0 {
			buf.WriteString("\nInvariants:\n")
			for _, inv := range res.Invariants {
				status := "FAIL"
				if inv.Passed {
					status = "PASS"
				}
				buf.WriteString(fmt.Sprintf("[%s] %s (%s): Expected %s, Got %s\n", status, inv.Metric, inv.Scope, inv.Expected, inv.Actual))
			}
		}
		output = buf.Bytes()
	}

	if err != nil {
		log.Fatalf("Failed to marshal report: %v", err)
	}

	if filePath != "" {
		if err := os.WriteFile(filePath, output, 0644); err != nil {
			log.Fatalf("Failed to write report to %s: %v", filePath, err)
		}
		fmt.Printf("Report written to %s\n", filePath)
	} else {
		fmt.Println(string(output))
	}
}
