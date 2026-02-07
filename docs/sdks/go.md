# Go SDK

The Ratelord Go SDK provides a native client for interacting with the `ratelord` daemon. It abstracts the "Ask-Wait-Act" negotiation protocol, ensuring your agents comply with rate limits and governance policies automatically.

## Installation

```bash
go get github.com/rmax-ai/ratelord/pkg/client
```

## Core Concepts

The SDK is built around the **Ask-Wait-Act** pattern:

1.  **Ask**: You submit an `Intent` to the daemon.
2.  **Wait**: The SDK automatically sleeps if the daemon requests a wait (e.g., for throttling).
3.  **Act**: If approved, you proceed with your logic.

This "fail-closed" design ensures that if the daemon is unreachable or returns an error, your agent defaults to **not** acting, preventing unmanaged consumption.

## Usage

### 1. Initialize the Client

Create a new client instance pointing to your daemon's address (default: `http://localhost:8090`).

```go
import "github.com/rmax-ai/ratelord/pkg/client"

func main() {
    // Connect to local daemon
    c := client.NewClient("http://localhost:8090")
}
```

### 2. Define an Intent

An `Intent` describes *who* wants to do *what*, *where*, using *which* identity, and with what *urgency* (which maps to the `urgency` field in the JSON payload).

```go
intent := client.Intent{
    AgentID:    "crawler-01",        // Your agent's unique name
    IdentityID: "pat:rmax",          // The credential/identity key
    WorkloadID: "issue_fetch",       // Logical name of the task
    ScopeID:    "repo:openai/gpt-3", // Target resource scope
    Urgency:    "normal",            // "high" | "normal" | "background"
}
```

### 3. Ask for Permission

Call `Ask()` to negotiate. This method blocks if the daemon enforces a wait time.

```go
ctx := context.Background()
decision, err := c.Ask(ctx, intent)
if err != nil {
    // SDK handles connectivity errors by failing closed, but you can log them here.
    log.Fatalf("Negotiation failed: %v", err)
}

if !decision.Allowed {
    log.Printf("Action denied: %s", decision.Reason)
    return // Do not proceed
}

// ... Perform your API call or action here ...
log.Println("Action performed successfully.")
```

## Advanced Configuration

### Context and Timeouts
The `Ask` method accepts a standard `context.Context`. Use this to enforce timeouts on the negotiation itself (e.g., if you can't afford to wait too long).

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

decision, err := c.Ask(ctx, intent)
```

### Feedback (Coming Soon)
Future versions will support a `Feedback()` method to report actual consumption metrics back to the daemon, closing the loop for adaptive shaping.
