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
