package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/rmax-ai/ratelord/pkg/client"
)

func main() {
	// 1. Initialize the client
	// Point to your running ratelord-d instance
	c := client.NewClient("http://127.0.0.1:8090")

	// 2. Check health
	ctx := context.Background()
	status, err := c.Ping(ctx)
	if err != nil {
		log.Fatalf("Failed to ping daemon: %v", err)
	}
	fmt.Printf("Daemon Status: %s (Version: %s)\n", status.Status, status.Version)

	// 3. Define an intent
	intent := client.Intent{
		AgentID:    "example-agent",
		IdentityID: "dev-user",
		WorkloadID: "api-scan",
		ScopeID:    "github:rmax-ai/ratelord",
		Urgency:    "normal",
	}

	// 4. Ask for permission
	fmt.Println("Asking for permission...")
	decision, err := c.Ask(ctx, intent)
	if err != nil {
		log.Fatalf("Error asking intent: %v", err)
	}

	// 5. Handle the decision
	if decision.Allowed {
		fmt.Printf("✅ Access Granted! (Intent ID: %s)\n", decision.IntentID)
		if decision.Modifications.WaitSeconds > 0 {
			fmt.Printf("   (Waited %.2fs before approval)\n", decision.Modifications.WaitSeconds)
		}
		// Perform the actual work here...
		doWork()
	} else {
		fmt.Printf("❌ Access Denied. Reason: %s\n", decision.Reason)
	}
}

func doWork() {
	fmt.Println("   Doing work...")
	time.Sleep(100 * time.Millisecond)
	fmt.Println("   Work complete.")
}
