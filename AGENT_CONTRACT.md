# AGENT_CONTRACT: ratelord

This document defines the **binding protocol** between agents (clients) and the `ratelord` daemon. It establishes the "Ask-Wait-Act" loop as a non-negotiable standard for all constrained operations.

---

## 1. The Core Mandate

### "Ask Before You Act"

The fundamental law of `ratelord` is the **Negotiation Principle**:

> **No agent shall perform a constrained action without first submitting an Intent and receiving an Approval.**

Agents do not own their consumption budget; the daemon does. Agents are petitioners. The daemon is the governor.

### The Interaction Loop

1.  **Formulate Intent**: The agent identifies what it wants to do (`identity`, `workload`, `scope`).
2.  **Submit Intent**: The agent sends an `intent_submitted` event to the daemon.
3.  **Wait for Decision**: The agent **pauses execution** until a decision is returned.
4.  **Respect Decision**:
    *   **Approve**: Execute the action immediately.
    *   **ApproveWithModifications**: Apply required changes (e.g., sleep for `wait_seconds`) then execute.
    *   **Deny**: **Do not** execute. Handle the rejection (retry later, fail task, or degrade).
5.  **Emit Result** (Optional but recommended): Report the actual cost/outcome via `usage_observed`.

---

## 2. The Intent Protocol

### 2.1 Intent Submission

An intent is a structured request for permission. It must contain sufficient context for the Policy Engine to make a safe decision.

**Required Fields:**

*   **`agent_id`**: Who is asking? (e.g., `crawler-01`, `ci-runner`).
*   **`identity_id`**: Which credentials will be used? (e.g., `pat:rmax`, `app:ratelord-bot`).
*   **`workload_id`**: What abstract task is this? (e.g., `repo_scan`, `issue_comment`).
*   **`scope_id`**: Where is this happening? (e.g., `repo:owner/name`, `org:my-org`).
*   **`urgency`**: How critical is this? (`high` | `normal` | `background`).

**Optional Fields:**

*   **`expected_cost`**: Estimate of consumption (e.g., "100 points"). If unknown, daemon uses historical averages.
*   **`duration_hint`**: Expected runtime (for long-running jobs).

### 2.2 Decision Handling

The daemon responds with one of three verdict types. Agents **MUST** handle all three.

#### A. `Approve`
*   **Meaning**: The action is safe.
*   **Obligation**: Proceed immediately.

#### B. `ApproveWithModifications` (The "Yellow Light")
*   **Meaning**: The action is safe **only if** modified.
*   **Modifications**:
    *   `wait_seconds`: Agent **MUST** sleep for this duration before executing.
    *   `identity_switch`: (Advanced) Agent should use a different credential if capable.
*   **Obligation**: Apply the modification (sleep), then proceed.

#### C. `Deny`
*   **Meaning**: The action is unsafe or forbidden.
*   **Reason**: Provided in the response (e.g., `risk_too_high`, `policy_violation`, `hard_limit_reached`).
*   **Obligation**: **Abort** the specific provider call.
    *   *Retry*: Agents may retry after a backoff (unless reason is permanent).
    *   *Fail*: Agents may fail the task if it is non-retryable.
    *   *Degrade*: Agents may skip the optional step (e.g., skip fetching comments).

---

## 3. Behavioral Obligations

### 3.1 Attribution & Honesty
Agents must truthfully self-identify.
*   **Do not spoof**: Never use `urgency: high` for background tasks to bypass checks.
*   **Do not mask**: Never use `sentinel:unknown` for Identity/Scope if the real values are known.
*   **Consequence**: The Policy Engine may penalize dishonest agents (future feature: "sin bin").

### 3.2 Adaptability
Agents are expected to be **elastic**.
*   **Handle Delays**: A 5-second throttle should not crash the agent.
*   **Handle Rejection**: A denial is a valid state, not an exception.
*   **Degrade Gracefully**: If the "Full Scan" workload is denied, fall back to "Metadata Only" if possible.

### 3.3 Budget Awareness (Pre-emption)
Smart agents should check the **System Health** (via SDK) before forming complex plans.
*   If `system.status == degraded`, the agent should voluntarily pause low-priority work, reducing noise in the intent log.

---

## 4. Client Library (SDK) Responsibilities

The `ratelord` SDK (e.g., `ratelord-client`) wraps the raw event protocol to simplify compliance.

### 4.1 "Ask-Wait-Act" Wrapper
The SDK should provide a blocking primitive:
```python
# Conceptual Python SDK
with ratelord.guard(
    identity="pat:gh-123",
    scope="repo:foo/bar",
    workload="issue_triage",
    urgency="normal"
) as decision:
    if decision.accepted:
        # SDK automatically handled any 'wait_seconds' before entering this block
        api.call(...)
    else:
        # Decision was Denied
        logger.warn(f"Skipped due to: {decision.reason}")
```

### 4.2 Local Fallbacks
If the daemon is unreachable:
*   **Fail Safe (Default)**: Assume `Deny` to prevent unmonitored exhaustion.
*   **Fail Open (Configurable)**: Allow critical actions with a warning log (risk accepted by operator).

### 4.3 Timeout Handling
If the daemon does not reply within `timeout_ms`:
*   The SDK treats this as a `ProviderError` (daemon unavailable).
*   Default behavior: **Fail Safe** (Deny).

---

## 5. Specific Scenarios

### 5.1 Batch Operations
**Guidance**: Negotiate **per item** or **per small chunk**.
*   *Bad*: Ask for 1000 items at once. (Hard to estimate, risk of partial failure).
*   *Good*: Ask for chunk of 10.
*   *Why*: Allows the daemon to interleave high-priority requests from other agents in between chunks.

### 5.2 Long-Running / Streaming
**Guidance**: **Periodic Re-negotiation**.
*   For a stream consuming points over time, the agent must submit a "Keep-Alive" intent every N seconds or M units.
*   If a re-negotiation is Denied, the agent must terminate the stream.

### 5.3 Multi-Pool Actions
Some actions hit multiple limits (e.g., "Search API" hits both `search` pool and `core` pool).
*   **Submission**: The intent specifies the primary workload.
*   **Daemon Logic**: The daemon maps the workload to *all* impacted pools and checks *all* relevant forecasts.
*   **Result**: Approval is granted only if *all* pools are safe.

---

## 6. Error Handling & Edge Cases

### 6.1 Daemon Unreachable
*   **Behavior**: Agent/SDK must log the connection error.
*   **Fallback**:
    *   `Strict Mode` (Prod default): Block execution.
    *   `Loose Mode` (Dev default): Log warning, proceed blindly (blind spot in data).

### 6.2 Decision Timeout
*   The agent cannot wait forever.
*   After `intent_timeout` (default 5s), the agent must abandon the intent.
*   The agent records a `client_error` event (local log).

### 6.3 "Soft Denials" (Deferral)
*   A `Deny` with reason `defer_until_reset` suggests the action *will* be allowed later.
*   **Agent Behavior**: Sleep until the suggested time (if provided in metadata) or requeue the job.

---

## 7. Summary of Invariants

| Invariant | Description |
| :--- | :--- |
| **No Ghost Traffic** | Every API call must have a corresponding Intent. |
| **Blocking Wait** | Agents must actually wait if told to wait. |
| **Scoped Identity** | Agents must not hide their Identity or Scope. |
| **Fail Safe** | When in doubt (no daemon), do nothing. |

---
