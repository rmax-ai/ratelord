# API Reference

The Ratelord daemon (`ratelord-d`) exposes a JSON-over-HTTP API. This is the primary interface for Agents, CLIs, and UIs.

**Base URL**: `http://127.0.0.1:8090` (default)

## Core Concepts

- **Local-Only**: The API binds to `localhost`. It is designed for single-machine coordination.
- **Fail-Safe**: Clients should handle timeouts (default 5000ms) by assuming a "Deny" or "Safe Mode" state.

---

## Endpoints

### Intent Negotiation

#### `POST /v1/intent`
The main entry point for agents. Submits an intent to perform an action and waits for a decision.

**Request:**
```json
{
  "agent_id": "crawler-01",
  "identity_id": "pat:user",
  "workload_id": "repo_scan",
  "scope_id": "repo:owner/project",
  "urgency": "normal"
}
```

**Response:**
```json
{
  "decision": "approve", // or "deny_with_reason", "approve_with_modifications"
  "intent_id": "evt_12345...",
  "evaluation": {
    "risk_summary": "Low risk (P99 TTE > 1h)"
  }
}
```

### System Status

#### `GET /v1/health`
Returns the operational status of the daemon.

**Response:**
```json
{
  "status": "ok", // or "degraded", "initializing"
  "version": "0.1.0",
  "uptime_seconds": 3600
}
```

#### `GET /v1/graph`
Returns the current constraint graph as a JSON object, describing nodes (pools, scopes) and edges (relationships).

**Response:**
```json
{
  "nodes": [...],
  "edges": [...]
}
```

### Analytics & Reporting

#### `GET /v1/trends`
Returns aggregated usage statistics over a specified time window.

**Parameters:**
- `from`: Start timestamp (ISO 8601).
- `to`: End timestamp (ISO 8601).
- `bucket`: Aggregation interval (e.g., "1h", "15m").
- `provider_id`: Filter by provider (optional).

#### `GET /v1/reports`
Generates and downloads CSV reports for audit or analysis.

**Parameters:**
- `type`: Report type (`usage`, `access_log`, or `events`).
- `from`: Start timestamp.
- `to`: End timestamp.

### Federation & Clustering

#### `GET /v1/cluster/nodes`
Returns the topology of the cluster, listing all known leaders and followers.

#### `POST /v1/federation/grant`
Used during leader-follower negotiation to issue a grant for resource usage.

**Request:**
```json
{
  "node_id": "follower-01",
  "resource": "github-api",
  "amount": 1000
}
```

### Integrations

#### `POST /v1/webhooks`
Registers a webhook URL to receive real-time event notifications.

**Request:**
```json
{
  "url": "https://hooks.slack.com/...",
  "events": ["intent.denied", "system.alert"]
}
```

### Administration

#### `POST /v1/admin/prune`
Manually triggers the pruning of old events from the ledger to free up storage.

**Request:**
```json
{
  "before": "2023-01-01T00:00:00Z"
}
```

### Event Streaming

#### `GET /v1/events`
Returns a stream of recent events. Useful for TUIs and dashboards.

**Parameters:**
- `limit`: Number of past events to fetch (default 50).
- `stream`: Set to `true` for Server-Sent Events (SSE).

### Identity Management

#### `POST /v1/identities`
Registers a new identity in the system.

**Request:**
```json
{
  "identity_id": "pat:github:rmax",
  "kind": "user",
  "metadata": { "name": "RMax" }
}
```

#### `DELETE /v1/identities/{id}`
Permanently removes an identity and scrubs its history from the Event Log (GDPR compliance).

---

## Error Handling

Standard HTTP status codes are used:

- **200 OK**: Request processed (includes `deny_with_reason` decisions).
- **400 Bad Request**: Invalid payload or missing fields.
- **503 Service Unavailable**: Daemon is initializing or overloaded.

**Note**: A "Deny" decision is a successful HTTP 200 response, not an error. It represents a valid policy enforcement.

### Observability

#### `GET /metrics`
Returns Prometheus-formatted metrics for system usage, including:
- `ratelord_usage`: Current usage per pool.
- `ratelord_limit`: Current limit per pool.
- `ratelord_intent_total`: Total number of processed intents.
- `ratelord_forecast_seconds`: Predicted time to exhaustion.

### Debugging

#### `POST /debug/provider/inject`
**Internal use only.** Inject fake usage or limit data into a provider poller for testing.

**Request:**
```json
{
  "provider_id": "github-mock",
  "pool_id": "core",
  "usage": 4500,
  "limit": 5000
}
```
