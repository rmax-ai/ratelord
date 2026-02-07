# JavaScript / TypeScript SDK

> **Status**: In Development
> **Specification**: [CLIENT_SDK_SPEC.md](../../CLIENT_SDK_SPEC.md)

The official JavaScript/TypeScript SDK for `ratelord` is currently under active development. This SDK is designed for Node.js environments and strictly enforces the "Ask-Wait-Act" contract.

## Design Goals

- **Fail-Closed**: Ensures safety by defaulting to a "Denied" state if the daemon is unreachable.
- **Promise-Based**: Modern `async/await` API.
- **TypeScript First**: Written in TypeScript with full type definitions included.
- **Zero-Dependency**: Minimal footprint for easy inclusion in any project.

## Usage Preview (Conceptual)

```typescript
import { RatelordClient, Intent } from '@ratelord/sdk';

const client = new RatelordClient({
  endpoint: "http://localhost:8081"
});

async function run() {
  const intent: Intent = {
    verb: "scrape",
    target: "example.com",
    identity: "scraper-bot-01"
  };

  // ask() handles network errors and waits automatically
  const decision = await client.ask(intent);

  if (decision.allowed) {
    console.log(`Proceeding (Token: ${decision.token})`);
    // Perform action...
  } else {
    console.log(`Denied: ${decision.reason}`);
  }
}

run().catch(console.error);
```

## Implementation Status

Please refer to the [Client SDK Specification](../../CLIENT_SDK_SPEC.md) for the authoritative implementation guide.
