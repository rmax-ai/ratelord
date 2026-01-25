# API_SPEC: ratelord

This document defines the technical interface between the `ratelord` daemon (`ratelord-d`) and its clients (agents, TUI, CLI). It specifies the transport mechanism, request/response schemas, and error handling protocols.

## 1. Transport Layer

The daemon exposes a **JSON-over-HTTP** interface on `localhost`.

### 1.1 Rationale
*   **Ubiquity**: HTTP is natively supported by every language (Python, Node, Go, Rust, Curl).
*   **Simplicity**: JSON payloads are human-readable and easy to debug.
*   **Zero-Ops**: No complex binary protocols or custom drivers required for basic integration.
*   **Local-First**: Bound strictly to `127.0.0.1` (or a Unix Domain Socket in future strict modes).

### 1.2 Configuration
*   **Default Port**: `8090` (configurable via env `RATELORD_PORT`)
*   **Bind Address**: `127.0.0.1`
*   **Content-Type**: `application/json` required for all POST bodies.

---

## 2. API Reference

### 2.1 Intent Negotiation (RPC)

The core "Ask-Wait-Act" primitive. This endpoint blocks until a decision is reached or the request times out.

**`POST /v1/intent`**

#### Request (`Intent`)
```json
{
  "agent_id": "string",       // Who is asking? (e.g., "crawler-01")
  "identity_id": "string",    // Credentials (e.g., "pat:rmax")
  "workload_id": "string",    // Abstract task (e.g., "repo_scan")
  "scope_id": "string",       // Target boundary (e.g., "repo:owner/name")
  "urgency": "string",        // "high" | "normal" | "background"
  "expected_cost": number,    // Optional: Estimated consumption units
  "duration_hint": number,    // Optional: Expected runtime in seconds
  "client_context": object    // Optional: arbitrary metadata for logs
}
```

#### Response (`Decision`)
```json
{
  "intent_id": "string",      // Unique ID assigned by daemon
  "decision": "string",       // "approve" | "approve_with_modifications" | "deny_with_reason"
  "modifications": {
    "wait_seconds": number,   // If > 0, client MUST sleep before acting
    "identity_switch": "string" // Optional: Alternate identity to use
  },
  "reason": "string",         // Present if denied (human-readable)
  "evaluation": {
    "as_of_ts": "string",     // ISO8601 timestamp of decision
    "risk_summary": "string"  // Why this decision was made
  }
}
```

#### Status Codes
*   `200 OK`: Decision reached (Approve OR Deny). Note: A Deny is a successful HTTP 200 response with `decision: "deny_with_reason"`.
*   `400 Bad Request`: Invalid schema (missing mandatory fields).
*   `503 Service Unavailable`: Daemon is initializing or overloaded.

---

### 2.2 System Observability

**`GET /v1/health`**
Returns the operational status of the daemon.

#### Response
```json
{
  "status": "ok",           // "ok" | "degraded" | "initializing"
  "version": "string",      // Daemon version
  "uptime_seconds": number,
  "db_checkpoint_lag": number // Optional: Health metric
}
```

**`GET /v1/forecast`**
Returns current predictions for a specific scope/pool. Useful for debugging or TUI "what-if" analysis.

#### Query Params
*   `scope_id`: (Required) e.g., `repo:owner/name`
*   `identity_id`: (Optional) Filter by identity
*   `pool_id`: (Optional) Filter by resource pool (e.g., `core`, `search`)

#### Response (`Forecast`)
```json
{
  "scope_id": "string",
  "pool_id": "string",
  "as_of_ts": "string",
  "burn_rate": {
    "1m": number,
    "1h": number
  },
  "time_to_exhaustion": {
    "p50_seconds": number,
    "p99_seconds": number
  },
  "risk_score": number,      // 0.0 - 1.0
  "is_gated": boolean        // If true, new intents are likely to be denied
}
```

---

### 2.3 Event Streaming (TUI)

**`GET /v1/events`**
Stream recent events for real-time visualization (TUI live tail).

#### Query Params
*   `limit`: Number of past events to return (default 50).
*   `stream`: `true` to enable Server-Sent Events (SSE).

#### Response (Standard JSON)
Array of `Event` objects (see DATA_MODEL.md).

#### Response (SSE Mode)
Line-delimited JSON events.
```text
data: {"event_id": "...", "type": "intent_submitted", ...}
data: {"event_id": "...", "type": "usage_observed", ...}
```

---

## 3. Schemas & Validation

All endpoints enforce strict JSON Schema validation.

### 3.1 Common Types

*   **Timestamps**: ISO 8601 strings (`2024-01-01T12:00:00Z`).
*   **IDs**: String, case-sensitive. Recommended format: `type:value` (e.g., `scope:repo:rmax/ratelord`).
*   **Enums**:
    *   `urgency`: `["high", "normal", "background"]`
    *   `decision`: `["approve", "approve_with_modifications", "deny_with_reason"]`

### 3.2 Error Payload
Standardized error format for 4xx/5xx responses.

```json
{
  "error": {
    "code": "string",       // Machine-readable (e.g., "invalid_scope")
    "message": "string",    // Human-readable detail
    "request_id": "string"  // For correlation
  }
}
```

---

## 4. Authentication & Security

Since `ratelord` is a local-first daemon (Phase 1), security relies on **process isolation** and **localhost binding**.

*   **Binding**: The daemon MUST listen only on `127.0.0.1` (IPv4) or `::1` (IPv6). It MUST NOT bind to `0.0.0.0` by default.
*   **Agent Identity**: The `agent_id` in the `Intent` payload is trusted. (In Phase 2/3, we may add mTLS or process-owner verification if needed).
*   **Secrets**: The API MUST NOT accept or return raw credentials (PATs, keys) in payloads. Identity is referenced by ID (`identity_id`), not by value.

---

## 5. Client Behavior

### 5.1 Timeouts
Clients MUST set a read timeout on the `/v1/intent` call.
*   **Recommended**: 5000ms.
*   **Fallback**: If timeout occurs, assume **DENY** (Fail Safe).

### 5.2 Retry Logic
*   **503 Unavailable**: Retry with exponential backoff.
*   **429 Too Many Requests** (from Daemon): Retry after `Retry-After` header.
*   **400 Bad Request**: Do not retry; fix the payload.

### 5.3 Keep-Alive
Clients engaging in long-running streaming usage should periodically submit new intents (e.g., every 60s) to re-confirm budget availability.

---

## 6. Future Extensions (Phase 2+)

*   **Unix Domain Sockets**: For lower latency and file-permission based security on Linux/macOS.
*   **Batch Intents**: `POST /v1/intents/batch` for high-throughput agents requesting multiple slots.
*   **Webhooks**: Daemon pushing alerts to a registered URL (e.g., for desktop notifications).
