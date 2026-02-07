# Python SDK

The official Python SDK for `ratelord`. It provides idiomatic Python bindings for the `ratelord` daemon, strictly enforcing the "Ask-Wait-Act" contract.

## Installation

```bash
pip install ratelord
```
*(Note: Ensure you have built/installed the package locally or from your private registry, as it may not be on PyPI yet.)*

## Usage

The SDK currently provides a synchronous client that handles retries (via `tenacity`) and automatic waiting/throttling.

```python
from ratelord import Client, Intent

def main():
    # Initialize client (defaults to http://127.0.0.1:8090)
    client = Client(endpoint="http://localhost:8090")

    # Define the intent
    intent = Intent(
        agent_id="crawler-01",
        identity_id="pat:user",
        workload_id="repo_scan",
        scope_id="repo:owner/project",
        urgency="normal"  # "high" | "normal" | "background"
    )

    # Ask for permission
    # This call blocks if the daemon requires the agent to wait (throttle).
    decision = client.ask(intent)

    if decision.allowed:
        print(f"Proceeding (Intent ID: {decision.intent_id})")
        # Perform your action...
    else:
        print(f"Denied: {decision.reason}")

if __name__ == "__main__":
    main()
```

## API Reference

### `Client(endpoint)`

*   `endpoint`: Base URL of the Ratelord daemon.

### `client.ask(intent) -> Decision`

Negotiates an intent with the daemon.

*   **Behavior**:
    *   **Fail-Closed**: If the daemon is unreachable after retries, returns a `Decision` with `allowed=False` (reason: `daemon_unreachable`).
    *   **Blocking**: If the daemon returns a `wait_seconds` instruction (Shaping), this method **sleeps** automatically before returning `allowed=True`.
    *   **Retries**: Uses exponential backoff for transient 5xx errors or connection issues.

### `Intent` Data Class

*   `agent_id` (str): Unique identifier for the agent.
*   `identity_id` (str): Credential/User ID.
*   `workload_id` (str): Logical task name.
*   `scope_id` (str): Target resource scope.
*   `urgency` (str): "high", "normal", or "background".
*   `expected_cost` (float): Optional cost estimate.
