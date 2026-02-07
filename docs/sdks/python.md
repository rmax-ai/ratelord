# Python SDK

The official Python SDK for `ratelord` is currently under active development. This SDK will provide idiomatic Python bindings for the `ratelord` daemon, strictly enforcing the "Ask-Wait-Act" contract.

## Design Goals

- **Fail-Closed**: Ensures safety by defaulting to a "Denied" state if the daemon is unreachable.
- **Blocking**: Automatically handles wait times mandated by the daemon.
- **Async Support**: Native `asyncio` support for high-performance applications.
- **Type Hinting**: Full type coverage for `Intent`, `Decision`, and `Identity` objects.

## Usage Preview (Conceptual)

```python
import asyncio
from ratelord import RatelordClient, Intent

async def main():
    client = RatelordClient(endpoint="http://localhost:8081")
    
    intent = Intent(
        agent_id="crawler-01",
        identity_id="pat:user",
        workload_id="repo_scan",
        scope_id="repo:owner/project",
        urgency="normal" # "high" | "normal" | "background"
    )

    # Ask() handles network errors and waits automatically
    decision = await client.ask(intent)

    if decision.allowed:
        print(f"Proceeding (Token: {decision.token})")
        # Perform action...
    else:
        print(f"Denied: {decision.reason}")

if __name__ == "__main__":
    asyncio.run(main())
```

## Implementation Status

Please refer to the [Client SDK Specification](../../CLIENT_SDK_SPEC.md) for the authoritative implementation guide.
