# PREDICTION: ratelord

This document defines how `ratelord` forecasts the future state of constraint pools.

The system’s core value proposition is **predictive governance**: instead of blocking actions only when a limit is hit (reactive), `ratelord` blocks or shapes actions when the *risk of future exhaustion* becomes unacceptable.

---

## 1. Core Philosophy

### 1.1 Prediction > Observation
Raw counters ("4500/5000 remaining") are insufficient for decision-making. A counter without a rate is meaningless. `ratelord` treats observations as inputs to a probabilistic model that outputs **Time-to-Exhaustion (TTE)** and **Risk**.

### 1.2 Time-Domain Reasoning
All governance decisions are made in the time domain.
- **Wrong**: "Deny because remaining < 500."
- **Right**: "Deny because P90 TTE is 5 minutes, but reset is in 60 minutes."

### 1.3 Probabilistic Nature
The future is uncertain. Agents burst, providers suffer latency, and clocks drift. Therefore, `ratelord` never predicts a single number. It predicts a distribution:
- **P50 (Expected)**: The median outcome. "We probably have 30 minutes left."
- **P90 (Conservative)**: The safe bet. "We are 90% sure we have at least 15 minutes left."
- **P99 (Worst Case)**: The defensive bound. "In the worst 1% of scenarios, we survive 5 minutes."

Governance policies typically gate on **P90** or **P99** to ensure reliability.

---

## 2. Burn Rate Modeling

Burn rate ($B$) is the velocity of consumption (units per second). It is the fundamental derivative used to calculate TTE.

### 2.1 Exponential Moving Average (EMA)
To balance responsiveness with stability, `ratelord` tracks burn rate using Exponential Moving Averages over multiple windows:
- **Instantaneous ($B_{inst}$)**: Very short window (e.g., last 1 minute or last $N$ events). Detects spikes immediately.
- **Trend ($B_{trend}$)**: Medium/Long window (e.g., 15 minutes, 1 hour). Captures baseline load.

### 2.2 Handling Variance
We do not just track the mean burn rate $\mu$; we track the variance $\sigma^2$.
High variance implies a volatile workload, which widens the confidence intervals for TTE.

- **Stable Agent**: Low variance $\rightarrow$ P50 and P99 are close.
- **Bursty Agent**: High variance $\rightarrow$ P99 is much lower (shorter TTE) than P50.

### 2.3 Decay and Silence
If no usage events occur, the burn rate must decay toward zero (or a known floor).
- **Decay Function**: `ratelord` applies a decay factor based on time-since-last-event.
- **Silence != Safety**: A silent provider might mean a network partition, not zero usage. See "Uncertainty" below.

---

## 3. Time-to-Exhaustion (TTE)

TTE is the estimated duration until a pool's capacity reaches zero, assuming no reset occurs.

### 3.1 The Formula
Fundamentally:
$$ TTE = \frac{\text{Remaining Capacity}}{\text{Burn Rate}} $$

### 3.2 Probabilistic TTE
Because Burn Rate is a distribution $N(\mu, \sigma)$, TTE is also a distribution.
Instead of complex closed-form integration, `ratelord` approximates quantiles:

- **P50 TTE**: Derived from Mean Burn Rate.
  $$ TTE_{P50} \approx \frac{C_{rem}}{B_{mean}} $$

- **P99 TTE (Worst Case)**: Derived from a high-percentile Burn Rate (e.g., Mean + 2$\sigma$ or Peak observed).
  $$ TTE_{P99} \approx \frac{C_{rem}}{B_{P99}} $$

*Note: As $C_{rem}$ approaches zero, TTE approaches zero regardless of burn rate.*

---

## 4. Reset-Aware Risk

The absolute TTE is less important than the relationship between **TTE** and **Time-to-Reset (TTR)**.

### 4.1 The Critical Question
"Will we exhaust *before* the reset?"

We are safe if:
$$ TTE_{min} > TTR_{max} $$

### 4.2 Risk Probability
Risk is the probability that exhaustion occurs before reset:
$$ P(\text{Exhaustion}) = P(TTE < TTR) $$

- If $P(\text{Exhaustion}) > \text{Threshold}$ (e.g., 10%), the system enters a **Yellow/Warning** state.
- If $P(\text{Exhaustion})$ is near 100%, the system enters a **Red/Critical** state (throttling active).

### 4.3 Safety Margin
The **Safety Margin** is the buffer between the worst-case survival time and the reset time.
$$ \text{Margin} = TTE_{P99} - TTR $$

- Positive Margin: "Even at worst-case burn, we survive past the reset."
- Negative Margin: "At worst-case burn, we die before reset."

### 4.4 Reset Uncertainty
$TTR$ is also a distribution, not a scalar.
- **Fixed Windows**: TTR is precise (modulo clock skew).
- **Rolling Windows**: TTR is complex (capacity trickles back in).
- **Unknown/Jitter**: TTR has uncertainty bounds.

`ratelord` conservatively assumes the *latest* probable reset time ($TTR_{max}$) when calculating risk.

---

## 5. Hierarchy of Prediction

Predictions are not flat; they exist at every node in the constraint graph where a limit or burn rate can be observed.

### 5.1 Pool-Level Prediction (Physical Reality)
*Scope: The Provider's limit (e.g., GitHub REST Core for an Org).*
- The most authoritative prediction.
- Aggregates usage from *all* identities and scopes sharing this pool.
- If this TTE is critical, *everyone* sharing the pool gets throttled.

### 5.2 Identity-Level Prediction (Attribution)
*Scope: Specific Token / Identity.*
- Tracks the burn rate attributable to a single identity.
- Useful for detecting "noisy neighbors" or rogue agents.
- Used to target `approve_with_modifications` (throttle specific offender) rather than global panic.

### 5.3 Scope-Level Prediction (Logical)
*Scope: Repo, Project, or Environment.*
- "How fast is the `frontend-repo` CI pipeline burning budget?"
- Used for high-level resource planning and "virtual" limits.

---

## 6. Uncertainty & Degraded States

Forecasts must be honest about what they don't know.

### 6.1 Staleness
As data ages (time since last `provider_poll_observed`), uncertainty grows.
- **Mechanism**: Inflate the variance $\sigma$ of the burn rate model as a function of staleness.
- **Result**: P99 TTE drops rapidly (becomes more conservative) as data gets old. "If we haven't seen data in 5 minutes, assume the worst."

### 6.2 Unknown Reset Times
If a provider does not report a reset time (or we haven't seen one yet):
- Assume $TTR = \infty$ (worst case) or a provider-specific heuristic default.
- Risk becomes effectively 100% if any burn exists, forcing conservative throttling until a reset is observed.

### 6.3 Missing Data (Degraded Provider)
If `provider_error` events occur:
- Freeze $C_{rem}$ at last known value (or decay it pessimistically).
- Widen confidence intervals.
- Governance policies typically switch to "Safe Mode" (deny non-critical intents).

---

## 7. Inputs & Outputs

### 7.1 Inputs (from Event Log)
- `usage_observed`: Used to update $C_{rem}$ and calculate instantaneous burn $B_{inst}$.
- `reset_observed`: Used to anchor the $TTR$ calculation.
- `constraint_observed`: Updates the capacity ceiling $C_{total}$.
- `provider_poll_observed`: Timestamp provides the "freshness" anchor.

### 7.2 Outputs (Events)
The Forecaster emits `forecast_computed` events. These are the **only** signals Policy uses to make decisions.

**Event Payload (`forecast_computed`)**:
```json
{
  "provider_id": "github",
  "pool_id": "rest_core",
  "scope_id": "org:acme",
  "as_of_ts": 1700000000,
  "tte": {
    "p50_seconds": 3600,
    "p90_seconds": 1800,
    "p99_seconds": 300
  },
  "risk": {
    "probability_exhaustion_before_reset": 0.05,
    "safety_margin_seconds": 240,
    "ttr_seconds": 60
  },
  "burn_rate": {
    "mean": 1.5,
    "variance": 0.2,
    "unit": "req/sec"
  }
}
```

---

## 8. Open Questions

1.  **Cold Start**: How do we predict burn rate for a brand new identity with zero history? (Likely need a "pessimistic default" or "learning phase" state).
2.  **Rolling Windows**: Mathematically modeling TTE for rolling windows (where capacity returns continuously) is complex. Do we simulate it, or approximate with "worst case fixed window"?
3.  **Seasonality**: Should the burn rate model eventually account for time-of-day (e.g., "CI bursts at 9am")? (Out of scope for Phase 1, but architectural hooks should exist).
4.  **Feedback Loops**: If `ratelord` throttles an agent, observed burn rate drops. This increases TTE, which might relax throttling, causing burn to spike again. We need to distinguish "natural burn" from "throttled burn".
