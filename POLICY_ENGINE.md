# POLICY_ENGINE: ratelord

This document defines the **Decision Layer** of `ratelord`.

While the **Prediction Engine** observes the world and forecasts risk, the **Policy Engine** decides what to do about it. It acts as the "judge," taking raw forecasts and agent intents as input, and issuing binding verdicts (`intent_decided`) as output.

---

## 1. Core Philosophy

### 1.1 Governance by Risk, Not Arithmetic
Traditional rate limiters are arithmetic: `if count > limit then block`.
`ratelord` is probabilistic: `if P(exhaustion) > risk_tolerance then shape`.

Policies govern **future risk**, not past consumption. A 90% utilized pool might be perfectly safe if the reset is in 1 second. A 10% utilized pool might be critical if the burn rate is massive and reset is in 1 hour.

### 1.2 Hierarchy of Authority
Policies exist in a strict hierarchy. A lower-level policy cannot permit what a higher-level policy forbids.

1.  **Global Safety** (System-wide Hard Rules)
2.  **Organization / Scope** (Business Rules)
3.  **Pool / Resource** (Provider constraints)
4.  **Identity / Agent** (Local optimization)

### 1.3 Shaping over Blocking
The goal of `ratelord` is to maximize successful throughput, not to block traffic.
Policies prefer **Shaping** (delay, defer, degrade) over **Denial** whenever possible. A "Soft Limit" is a signal to slow down, not a wall.

---

## 2. Policy Structure

A **Policy** is a named collection of **Rules** bound to a specific **Target** (Scope, Pool, or Identity).

### 2.1 The Rule Object
A Rule consists of:
*   **Condition**: A logic predicate based on Forecasts, Metadata, or Time.
*   **Action**: The decision to apply if the condition matches.
*   **Priority**: Precedence order (critical vs. optimization).

### 2.2 Conditions
Conditions evaluate the `forecast_computed` and `intent_submitted` events.

*   **Risk Metrics**: `risk.p_exhaustion`, `tte.p99`, `margin.seconds`.
*   **Identity Metadata**: `agent.role` (prod/ci/dev), `agent.priority`.
*   **Pool State**: `pool.remaining_percent`, `pool.is_resetting`.
*   **Time**: `time.is_business_hours`, `time.seconds_to_reset`.

### 2.3 Actions
The engine outputs one of the following verdicts:

| Action | Description | Behavior |
| :--- | :--- | :--- |
| **APPROVE** | Risk is acceptable. | Agent proceeds immediately. |
| **SHAPE (Throttle)** | Risk is elevated. | Agent must wait `wait_seconds` before proceeding. |
| **DEFER** | Risk is high, but transient. | Agent must wait until **after** the next reset. |
| **DENY** | Risk is critical or rule violated. | Agent must abort the action completely. |
| **SWITCH** | Pool is exhausted/risky. | Agent must retry using a different `identity_id` (fallback). |

---

## 3. The Arbitration Process

When an agent submits an `intent_submitted` event, the Daemon triggers the Arbitration Cycle.

### 3.1 Input
*   **Intent**: Who, What, Where (Scope), How much (Cost).
*   **Forecast**: The latest `forecast_computed` for the relevant pool(s).
*   **Policies**: All active policies matching the Intent's scope hierarchy.

### 3.2 Evaluation Logic (The "Stack")
The engine evaluates rules from **Top to Bottom** (Global -> Local).

1.  **Check Hard Limits (Global/Pool)**:
    *   *Condition*: `risk.p_exhaustion > 0.99` OR `pool.remaining == 0`.
    *   *Action*: `DENY` or `DEFER`.
    *   *Result*: If matched, stop and return.

2.  **Check Business Rules (Org/Scope)**:
    *   *Condition*: `agent.role == 'dev' AND risk.p_exhaustion > 0.5`.
    *   *Action*: `SHAPE` (add delay) or `DENY`.
    *   *Result*: If matched, apply modifier.

3.  **Check Fairness/Optimization (Local/Identity)**:
    *   *Condition*: `identity.burn_rate > target_share`.
    *   *Action*: `SHAPE` (smooth out the spikes).

### 3.3 Output
The cycle emits an `intent_decided` event:
```json
{
  "event_type": "intent_decided",
  "intent_id": "uuid",
  "decision": "approve_with_modifications",
  "modifications": {
    "throttle_wait_seconds": 2.5,
    "cost_adjustment": 0
  },
  "reason": "policy:soft_limit_buffer",
  "risk_score": 0.45
}
```

---

## 4. Standard Rule Types

### 4.1 Hard Rules (The "Red Line")
Safety constraints that prevent catastrophic exhaustion.
*   **Trigger**: `margin.seconds < 0` (We will die before reset).
*   **Action**: `DENY` (if urgent) or `DEFER` (if waitable).
*   **Target**: All agents, regardless of priority.

### 4.2 Soft Rules (The "Yellow Zone")
Optimization constraints to preserve buffer.
*   **Trigger**: `risk.p_exhaustion > 0.2` (20% chance of failure).
*   **Action**: `SHAPE` (Linear backoff).
*   **Goal**: Slow down consumption just enough to land the plane safely at the reset time.

### 4.3 Fairness Rules (The "Bad Neighbor")
Prevents one agent from monopolizing a shared pool.
*   **Trigger**: `identity.burn_rate_share > 0.5` (Using >50% of pool capacity).
*   **Action**: `SHAPE` (Throttle specific identity).
*   **Note**: Applied *per identity*, preserving the pool for others.

### 4.4 Priority Rules (The "VIP Lane")
Differentiates between critical and non-critical workloads.
*   **Rule A**: `if role == 'ci' AND risk.level == 'high' THEN DENY`.
*   **Rule B**: `if role == 'prod' AND risk.level == 'high' THEN APPROVE`.
*   **Result**: Production traffic eats into the safety margin; CI traffic stops to save it.

---

## 5. Degraded Mode Policies

The Policy Engine must be robust against system failures (e.g., disconnected daemon, stale forecasts).

### 5.1 Stale Forecasts
If `time.now - forecast.as_of > stale_threshold`:
*   **Policy**: "Uncertainty Principle"
*   **Action**: Widen safety margins. `risk.p_exhaustion` is treated as 100% for non-critical traffic.
*   **Result**: Fail-safe defaults (throttle/deny) rather than fail-open (allow and crash).

### 5.2 Missing Provider Data
If the provider is down or returning errors:
*   **Policy**: "Emergency Stop"
*   **Action**: `DENY` all non-essential intents. `SHAPE` essential intents with massive backoff.

---

## 6. Configuration (Declarative)

Policies are defined in declarative files (e.g., `policies.yaml`).

```yaml
policies:
  - id: "global-safety-net"
    scope: "global"
    type: "hard"
    rules:
      - name: "prevent-exhaustion"
        condition: "risk.p99_exhaustion_before_reset == true"
        action: "defer"
        priority: 100

  - id: "dev-throttling"
    scope: "env:dev"
    type: "soft"
    rules:
      - name: "slow-down-devs"
        condition: "pool.utilization > 0.50"
        action: "shape"
        params:
          algorithm: "linear"
          factor: 2.0
        priority: 50
```

---

## 7. Open Questions

1.  **Feedback Loops**: If policies throttle agents, the *observed* burn rate drops. The Predictor might see this drop and signal "Risk is low", causing the Policy to relax, causing a spike.
    *   *Mitigation*: The Predictor needs to know *why* burn dropped (Natural vs. Artificial).
2.  **Switching Logic**: When a `SWITCH` action is issued (use backup key), how does the agent discover the backup `identity_id`? Does the daemon provide it, or is it client-side config?
3.  **Pre-emption**: Can a high-priority intent *revoke* a previously approved (but long-running) lower-priority intent? (Likely out of scope for Phase 1).
