# Ratelord Python SDK

A Python SDK for interacting with the ratelord daemon, providing safe and blocking intent negotiation.

## Installation

```bash
pip install .
```

## Usage

```python
from ratelord import Client, Intent

client = Client("http://localhost:8090")

intent = Intent(
    agent_id="my-agent",
    identity_id="pat:my-token",
    workload_id="repo_scan",
    scope_id="repo:owner/repo"
)

decision = client.ask(intent)
if decision.allowed:
    # Proceed with action
    pass
else:
    print(f"Denied: {decision.reason}")
```

## Features

- Blocking `ask` method with automatic wait handling
- Fail-closed behavior on daemon errors
- Type-safe data models using dataclasses