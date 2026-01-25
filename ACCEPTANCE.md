# ACCEPTANCE: ratelord Success Criteria

This document defines the formal "Definition of Done" for `ratelord`. It translates the project's high-level goals into verifiable, observable test cases. These criteria must be met before the system is considered production-ready.

**Status**: DRAFT

---

## 1. Daemon (ratelord-d)

The daemon is the central authority and event-sourced brain of the system.

### 1.1 Lifecycle & Persistence
| ID | Scenario | Procedure | Acceptance Criteria |
|----|----------|-----------|---------------------|
| D-01 | **Clean Start** | Start `ratelord-d` with no existing DB. | Process starts; `ratelord.db` and `ratelord.db-wal` are created; `system_started` event is logged. |
| D-02 | **Crash Recovery** | Kill `ratelord-d` (SIGKILL); Restart it. | Daemon restarts; Replays events from WAL; Internal state (counters, forecasts) matches state before kill. |
| D-03 | **Graceful Shutdown** | Send `SIGINT` or `SIGTERM` to `ratelord-d`. | Daemon stops accepting new intents; Flushes pending WAL writes; Exits with code 0. |

### 1.2 Event Sourcing
| ID | Scenario | Procedure | Acceptance Criteria |
|----|----------|-----------|---------------------|
| D-04 | **Event Immutability** | Inspect SQLite DB after a series of operations. | `events` table contains sequential entries; No rows are ever updated or deleted (append-only). |
| D-05 | **State Derivation** | Wipe `snapshots` table; Trigger a replay/rebuild command. | `snapshots` are regenerated and match the values derived from the `events` stream. |

### 1.3 Identity Management
| ID | Scenario | Procedure | Acceptance Criteria |
|----|----------|-----------|---------------------|
| D-06 | **Identity Registration** | Register a new Identity (e.g., `ratelord identity add`). | `identity_registered` event is emitted; Secrets are *not* logged in plain text; Identity appears in TUI/CLI list. |
| D-07 | **Initial Polling** | Register a valid Provider Identity. | Daemon automatically polls the Provider within 5s; `limits_polled` event is logged with correct values. |

---

## 2. API & Agent Contract

Agents interact with the daemon via the Unix Socket API. This contract is strict.

### 2.1 Intent Negotiation (The Happy Path)
| ID | Scenario | Procedure | Acceptance Criteria |
|----|----------|-----------|---------------------|
| A-01 | **Approve Intent** | Submit `POST /intent` with valid AgentID, Identity, Scope, and low usage. | Response is `200 OK`; Body contains `"decision": "approve"`; `intent_decision` event is logged. |
| A-02 | **Latency** | Measure RTT of `POST /intent` under light load. | 99th percentile latency < 10ms (daemon overhead must be negligible). |

### 2.2 Intent Negotiation (The Throttled Path)
| ID | Scenario | Procedure | Acceptance Criteria |
|----|----------|-----------|---------------------|
| A-03 | **Wait Instruction** | Configure policy to rate-limit; Submit `POST /intent`. | Response is `200 OK`; Body contains `"decision": "approve_with_modifications"`, `"wait_seconds": N` (where N > 0). |
| A-04 | **Denial** | Submit `POST /intent` that violates a "Hard Rule" (e.g., cost limit). | Response is `200 OK` (not 4xx); Body contains `"decision": "deny"`, `"reason": "..."`. |

---

## 3. Prediction & Drift

The system must predict exhaustion and correct itself when reality diverges.

### 3.1 Forecasting
| ID | Scenario | Procedure | Acceptance Criteria |
|----|----------|-----------|---------------------|
| P-01 | **Forecast Emission** | Emit a sequence of `intent_decision` events. | Daemon emits `forecast_updated` events; `tte_p50` (Time To Exhaustion) decreases as usage increases. |
| P-02 | **Reset Awareness** | Advance time across a Provider's reset window. | `forecast_updated` shows TTE jumping back to maximum/infinity immediately after reset time. |

### 3.2 Drift Correction
| ID | Scenario | Procedure | Acceptance Criteria |
|----|----------|-----------|---------------------|
| P-03 | **Detect External Usage** | Manually consume quota (e.g., `curl` GitHub API) bypassing `ratelord`. | Next Daemon Poll detects mismatch; `drift_detected` event is logged; Local state updates to match Provider. |
| P-04 | **Variance Adjustment** | Trigger P-03 multiple times. | Prediction uncertainty (variance) increases; P99 TTE becomes more conservative (shorter) than P50. |

---

## 4. Policy Engine

The "Constitution" of the system.

### 4.1 Rule Evaluation
| ID | Scenario | Procedure | Acceptance Criteria |
|----|----------|-----------|---------------------|
| Pol-01 | **Hard Limit** | Set Policy: "Deny if Remaining < 100". Simulate state where Remaining = 90. Submit Intent. | Intent is **Denied**. Reason cites the specific policy rule. |
| Pol-02 | **Load Shedding** | Set Policy: "If System Risk = High, Deny Urgency = Low". Trigger High Risk. Submit Low Urgency Intent. | Intent is **Denied**. Submit High Urgency Intent -> **Approved**. |

### 4.2 Hot Reloading
| ID | Scenario | Procedure | Acceptance Criteria |
|----|----------|-----------|---------------------|
| Pol-03 | **Policy Hot Reload** | Modify `policy.yaml` while daemon runs; Send `SIGHUP`. | Daemon logs `policy_updated`; Next intent is evaluated against the *new* rules; No restart occurred. |

---

## 5. TUI (User Interface)

The primary operator interface.

### 5.1 Visualization
| ID | Scenario | Procedure | Acceptance Criteria |
|----|----------|-----------|---------------------|
| T-01 | **Real-time Stream** | Generate traffic. Open TUI. | Activity Stream updates in near real-time (< 1s delay); Shows Intent decisions. |
| T-02 | **Status Indicators** | Trigger a "Warning" state (low quota). | TUI header changes color (e.g., Green -> Yellow/Red); "TTE" display highlights urgency. |
| T-03 | **Identity List** | Register multiple identities. | TUI lists all identities, their current scope, remaining quota, and reset time. |

---

## 6. Constraints & Pools

Verifying the graph model.

### 6.1 Shared vs. Isolated
| ID | Scenario | Procedure | Acceptance Criteria |
|----|----------|-----------|---------------------|
| C-01 | **Shared Pool** | Register 2 Agents using the *same* Identity/Scope. | Agent A's consumption reduces the available budget for Agent B visible in the next poll/state update. |
| C-02 | **Isolated Pool** | Register 2 Agents using *different* Identities (different PATs). | Agent A's consumption has *zero* effect on Agent B's budget. |
