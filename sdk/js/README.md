# Ratelord Client SDK

A Node.js/TypeScript SDK for interacting with the Ratelord daemon, providing client-side access to event sourcing, intent negotiation, and predictive resource management features.

## Usage

```typescript
import { RatelordClient } from '@ratelord/client';

const client = new RatelordClient({ endpoint: 'http://localhost:8090' });

async function main() {
  const decision = await client.ask({
    identity: 'my-service',
    pool: 'github-api',
    size: 1
  });

  if (decision.allowed) {
    // Proceed
  } else {
    // Wait or retry
  }
}
```

## Publishing

1. Update version in `package.json`.
2. Run `npm run build`.
3. Run `npm publish --access public`.
