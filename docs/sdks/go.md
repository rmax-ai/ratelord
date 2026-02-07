# Go SDK

The Ratelord Go SDK provides a native client for interacting with the `ratelord` daemon. It abstracts the "Ask-Wait-Act" negotiation protocol, ensuring your agents comply with rate limits and governance policies automatically.

## Installation

```bash
go get github.com/rmax-ai/ratelord/pkg/client
```

## Usage

### 1. Initialize the Client

```go
package main

import (
	"context"
	"log"

	"github.com/rmax-ai/ratelord/pkg/client"
)

func main() {
	// Connect to local daemon (defaults to http://127.0.0.1:8090)
	c := client.NewClient("http://localhost:8090")

	// Define the intent
	intent := client.Intent{
		AgentID:    "crawler-01",        // Your agent's unique name
		IdentityID: "pat:rmax",          // The credential/identity key
		WorkloadID: "issue_fetch",       // Logical name of the task
		ScopeID:    "repo:openai/gpt-3", // Target resource scope
		Urgency:    "normal",            // "high" | "normal" | "background"
	}

	// Ask for permission
	// This method blocks if the daemon enforces a wait time (throttling).
	decision, err := c.Ask(context.Background(), intent)
	if err != nil {
		// SDK handles connectivity errors by failing closed, but you can log them here.
		// "Fail-Closed" means err is non-nil if we couldn't confidently reach the daemon.
		log.Fatalf("Negotiation failed (Fail-Closed): %v", err)
	}

	if !decision.Allowed {
		log.Printf("Action denied: %s", decision.Reason)
		return // Do not proceed
	}

	// ... Perform your API call or action here ...
	log.Printf("Action allowed (Intent ID: %s)", decision.IntentID)
}
```

## API Reference

### `NewClient(endpoint string) *Client`

Creates a new client instance.

### `(c *Client) Ask(ctx context.Context, intent Intent) (*Decision, error)`

Negotiates an intent with the daemon.

*   **Behavior**:
    *   **Blocking**: If the daemon returns a `wait_seconds` instruction, this method **sleeps** automatically before returning.
    *   **Fail-Closed**: If the daemon is unreachable after retries (configured internally), it returns an error. The caller should treat this as a "Deny".
    *   **Context**: Respects `ctx` cancellation and timeouts.

### `Intent` Struct

*   `AgentID` (string): Unique identifier for the agent.
*   `IdentityID` (string): Credential/User ID.
*   `WorkloadID` (string): Logical task name.
*   `ScopeID` (string): Target resource scope.
*   `Urgency` (string): "high", "normal", or "background".
*   `ExpectedCost` (float64): Optional cost estimate.
