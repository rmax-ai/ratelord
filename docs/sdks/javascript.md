# JavaScript / TypeScript SDK

The official JavaScript/TypeScript SDK for `ratelord`. This SDK is designed for Node.js environments and strictly enforces the "Ask-Wait-Act" contract.

## Installation

```bash
npm install @ratelord/client
```

## Usage

```typescript
import { RatelordClient, RatelordIntent } from '@ratelord/client';

const client = new RatelordClient({
  endpoint: "http://localhost:8090",
  maxRetries: 3
});

async function run() {
  const intent: RatelordIntent = {
    agentId: "crawler-01",
    identityId: "pat:bot-user",
    workloadId: "repo_scan",
    scopeId: "repo:owner/project",
    urgency: "normal" // "high" | "normal" | "background"
  };

  try {
    // ask() handles network errors and waits automatically if the daemon requires throttling
    const decision = await client.ask(intent);

    if (decision.allowed) {
      console.log(`Proceeding (Intent ID: ${decision.intentId})`);
      // Perform your rate-limited action here...
    } else {
      console.log(`Denied: ${decision.reason}`);
      // Do not proceed
    }
  } catch (err) {
    console.error("Negotiation failed:", err);
  }
}

run();
```

## API Reference

### `RatelordClient(options)`

*   `options.endpoint`: Base URL of the Ratelord daemon (default: `http://localhost:8090`).
*   `options.maxRetries`: Number of times to retry failed connection attempts (default: 3).
*   `options.baseDelay`: Initial delay for backoff (ms).

### `client.ask(intent)`

Negotiates an intent with the daemon.

*   **Returns**: `Promise<RatelordDecision>`
*   **Behavior**:
    *   If the daemon returns `SHAPE` (throttle), this method **blocks** (awaits) for the specified duration before returning `allowed: true`.
    *   If the daemon returns `DENY`, it returns `allowed: false` immediately.
    *   If the network is unreachable after retries, it throws an error (fail-closed).
