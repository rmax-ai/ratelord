# CLIENT_SDK_SPEC: ratelord

This document defines the specification for client libraries (SDKs) used by agents to interact with the `ratelord` daemon. These SDKs wrap the HTTP API defined in `API_SPEC.md` and strictly enforce the "Ask-Wait-Act" contract.

## 1. Introduction

The Client SDK is the bridge between an autonomous agent and the `ratelord` daemon. Its primary purpose is to make the **negotiation of intent** seamless, blocking, and safe.

The SDK abstracts away:
*   HTTP transport details.
*   JSON serialization/deserialization.
*   **Wait enforcement** (sleeping when the daemon requests it).
*   Error handling and fail-safe defaults.

## 2. Core Principles

1.  **Fail-Closed (Safety First)**: If the daemon is unreachable, times out, or returns a 500 error, the SDK MUST return a **Denied** decision. Agents should not proceed without explicit approval.
2.  **Blocking by Default**: The `Ask` method blocks the caller. If the daemon mandates a wait time (e.g., "sleep 2s then proceed"), the SDK handles this sleep internally before returning the result.
3.  **Lightweight**: The SDK should have minimal dependencies, suitable for inclusion in small scripts or large applications.
4.  **Context-Aware**: All operations must support cancellation and timeouts (e.g., via Go `context.Context`).

## 3. Architecture

The SDK is a thin wrapper around the HTTP API.

```
[ Agent / Script ]
       |
       v
[ ratelord SDK ]  <-- Enforces "Ask-Wait-Act"
       |
       v (HTTP / JSON)
       |
[ ratelord-d ]
```

### 3.1 Components
*   **Client**: The main entry point. Maintains configuration (endpoint URL, defaults).
*   **Types**: Strongly-typed definitions for `Intent`, `Decision`, `Identity`, etc.
*   **Transport**: HTTP client with configured timeouts and retry policies.

## 4. Interface Definition

The following interface description uses a generic, C-like pseudo-code. Specific language bindings (Go, Python, TS) should adapt this to their idioms.

### 4.1 Types

#### `Intent`
Represents the agent's request to perform an action.
```
struct Intent {
    agent_id: string        // Required. Identifier of the agent.
    identity_id: string     // Required. The identity/credential to use.
    workload_id: string     // Required. The type of task (e.g., "repo_scan").
    scope_id: string        // Required. Target (e.g., "repo:owner/name").
    urgency: string         // "high" | "normal" | "background" (default: "normal")
    expected_cost: number   // Optional. Default: 1.0.
    duration_hint: number   // Optional. Estimated runtime in seconds.
    client_context: map<string, any> // Optional. Metadata.
}
```

#### `Decision`
The result of the negotiation.
```
struct Decision {
    allowed: boolean        // Derived helper (true if approved/modified).
    intent_id: string       // UUID assigned by daemon.
    status: string          // "approve" | "approve_with_modifications" | "deny_with_reason"
    modifications: Mods     // Changes required by daemon.
    reason: string          // If denied.
}

struct Mods {
    wait_seconds: number    // Time the SDK slept (informational).
    identity_switch: string // If daemon forced an identity swap.
}
```

### 4.2 Client Interface

```
interface Client {
    // Constructor
    // endpoint default: "http://127.0.0.1:8090"
    NewClient(endpoint: string) -> Client

    // Core Negotiation Method (Blocking)
    // 1. Sends Intent to Daemon.
    // 2. If response includes `wait_seconds`, SLEEPS internally.
    // 3. Returns final Decision.
    // 4. On network error/timeout -> Returns Denied Decision (Fail-Closed).
    Ask(ctx: Context, intent: Intent) -> (Decision, Error)

    // Identity Registration
    RegisterIdentity(ctx: Context, reg: IdentityRegistration) -> (Identity, Error)

    // Health Check
    Ping(ctx: Context) -> (Status, Error)
}
```

## 5. Behavior & Logic

### 5.1 The `Ask` Workflow

1.  **Validate**: Check that mandatory fields (`agent_id`, `scope_id`, etc.) are present in `Intent`.
2.  **Request**: `POST /v1/intent` to the daemon.
3.  **Handle Error**:
    *   If Network Error / 5xx / 400: Log error, return `allowed: false`, `reason: "daemon_unreachable"`.
    *   **CRITICAL**: Do not throw an exception for reachability issues; return a safe Deny decision so the agent can handle it gracefully (e.g., by backing off).
4.  **Handle Success**:
    *   Parse JSON response.
    *   **Auto-Wait**: If `modifications.wait_seconds > 0`:
        *   Log "Rate limiting: waiting X seconds...".
        *   Sleep(X).
    *   Return `Decision`.

### 5.2 Usage Example (Go)

```go
import "github.com/rmax/ratelord/sdk/go/ratelord"

func main() {
    client := ratelord.NewClient("http://localhost:8090")

    intent := ratelord.Intent{
        AgentID:    "crawler-01",
        IdentityID: "pat:rmax",
        WorkloadID: "issue_fetch",
        ScopeID:    "repo:openai/gpt-3",
    }

    // Single line "Ask for Permission"
    // Handles waiting/sleeping automatically if needed.
    decision, err := client.Ask(context.Background(), intent)
    if err != nil {
        log.Fatalf("SDK error: %v", err)
    }

    if !decision.Allowed {
        log.Printf("Denied: %s", decision.Reason)
        return // Do not proceed
    }

    // ... Perform Action ...
}
```

## 6. Language Targets

### 6.1 Go (Primary)
*   **Package**: `pkg/client` or separate repo `ratelord-go`.
*   **Idioms**: Use `context.Context`, struct tags for JSON, functional options for constructor.
*   **Status**: Required for Phase 1.

### 6.2 Python (Secondary)
*   **Package**: `ratelord` (PyPI).
*   **Idioms**: Type hints (`TypedDict` or `Pydantic` models), context managers optional but nice.
*   **Status**: Planned for Phase 2 (Data Science / ML usage).

### 6.3 TypeScript / Node (Tertiary)
*   **Package**: `@ratelord/sdk` (npm).
*   **Idioms**: Promises/Async-Await, discriminated unions for Decision types.
*   **Status**: Planned for Phase 2 (Web Agents).

## 7. Error Handling Specification

| Scenario | SDK Behavior | Decision Returned |
| :--- | :--- | :--- |
| **Daemon OK, Approves** | Return Decision | `allowed: true` |
| **Daemon OK, Modifies (Wait)** | **Sleep**, then Return | `allowed: true` |
| **Daemon OK, Denies** | Return Decision | `allowed: false`, `reason: "..."` |
| **Daemon Unreachable** | Return Deny | `allowed: false`, `reason: "daemon_offline"` |
| **HTTP 500 / Timeout** | Return Deny | `allowed: false`, `reason: "upstream_error"` |
| **Invalid Intent (400)** | Return Error | `error: "invalid_intent"` |

## 8. Future Extensions

*   **`Feedback()` method**: To report actual consumption after the action is complete (closing the loop).
*   **Async/Non-blocking Ask**: For high-throughput clients that want to manage their own concurrency/waiting.
