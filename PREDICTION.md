# PREDICTION: ratelord

This document defines how `ratelord` transforms raw observations into actionable forecasts. It shifts governance from "reactive counting" to "predictive risk management."

## Core Principles

1.  **Forecasts are Probabilistic**: We never know the exact future burn rate or reset time. We model distributions (P50, P90, P99) to quantify risk.
2.  **Time-Domain Reasoning**: The primary unit of risk is *time*, not *count*. "100 requests remaining" is meaningless without a burn rate. "10 minutes to exhaustion" is actionable.
3.  **Scoped & Hierarchical**: Forecasts are computed for every node in the Constraint Graph (Identity, Pool, Scope). A single intent may check multiple forecasts (e.g., Local Identity forecast AND Shared Pool forecast).
4.  **Reset-Aware**: The finish line is the *reset window*. Survival means reaching the reset timestamp with > 0 capacity.

---

## Burn Rate Modeling

Burn rate ($B$) is the speed of consumption (units/second). It is not a constant; it is a stochastic process.

### Exponential Moving Average (EMA)
For each Pool-at-Scope, `ratelord-d` maintains an EMA of consumption to establish a baseline trend.

-   **Short-term EMA**: Reacts quickly to bursts (for tactical throttling).
-   **Long-term EMA**: Captures sustained load (for strategic admission).

### Variance Tracking
The system tracks the *variance* ($\sigma^2$) of the burn rate. High variance implies high uncertainty, widening the gap between P50 and P99 forecasts.

-   **Stable Agent**: Low variance $\rightarrow$ P99 $\approx$ P50.
-   **Bursty Agent**: High variance $\rightarrow$ P99 $\ll$ P50 (P99 TTE is much shorter).

*Note: In Shared Pools, variance includes the aggregate noise of all actors.*

---

## Reset Uncertainty

Provider reset timestamps are not absolute truth. They suffer from:
1.  **Clock Skew**: Difference between provider server time and local daemon time.
2.  **Jitter**: Reset logic on the provider side may delay by seconds.
3.  **Propagation Delay**: API headers may reflect a state from milliseconds ago.

`ratelord` models the **Effective Reset Time** ($T_{reset}$) as a distribution, not a scalar.

-   **Optimistic Reset**: Earliest possible reset (provider claim).
-   **P99 Reset**: The latest likely reset (provider claim + max observed jitter + skew buffer).

Risk calculations use the *P99 Reset* timestamp to be conservative.

---

## Forecasting Logic

### Time-to-Exhaustion (TTE)

Given:
-   Current Capacity ($C_{now}$)
-   Modeled Burn Rate distribution ($B \sim N(\mu, \sigma^2)$)

We compute the time until $C=0$.

$$TTE = \frac{C_{now}}{B}$$

Since $B$ is a distribution, TTE is also a distribution.

-   **P50 TTE** (Expected): How long we last under normal conditions.
-   **P90 TTE** (Conservative): How long we last if load increases moderately.
-   **P99 TTE** (Worst Case): The "guaranteed" survival time for high-reliability planning.

### Probability of Exhaustion Before Reset ($P_{exhaust}$)

This is the **primary risk metric** for governance. It answers: "What are the odds we run dry before the refill?"

It is the probability that $TTE < TimeToReset$.

$$P(TTE < (T_{reset} - T_{now}))$$

-   If $P_{exhaust} > 0.1\%$ (P99 breach): **High Risk**.
-   If $P_{exhaust} > 50\%$ (P50 breach): **Critical Failure Imminent**.

---

## Risk Metrics (Signals to Policy)

The Prediction Engine emits `forecast_computed` events containing these signals for the Policy Engine:

### 1. Reserve Margin
$$Margin = TTE_{P99} - (T_{reset} - T_{now})$$
-   Positive: We are safe; we expect to reach reset with capacity to spare.
-   Negative: We are in the "Danger Zone"; we rely on luck (variance) to survive.

### 2. Burst Headroom
How much *additional* immediate consumption ($\Delta C$) can be accepted without spiking $P_{exhaust}$ above a safety threshold (e.g., 1%)?
This allows "burst budgeting" for urgent intents.

### 3. Shared Pool Saturation
For shared pools, we metricize the "Background Noise" (consumption by others).
-   High noise reduces the confidence of local forecasts.
-   Policy may force **isolation** (switch Identity) if saturation crosses a threshold.

---

## Data Flow

### Inputs (Observations)
The model ingests a stream of events:
-   `usage_observed`: "Used 50 tokens", "Remaining: 4950".
-   `reset_observed`: "Reset at 12:00:00 UTC".
-   `intent_submitted` (Hypothetical): "I plan to use ~100 tokens".

### Model State (Per Node)
For every active Pool/Scope/Identity node in the Constraint Graph, the daemon maintains:
-   Current $C_{now}$
-   Current $T_{reset}$ (with uncertainty bounds)
-   Burn Rate state ($\mu, \sigma^2$)

### Outputs (`forecast_computed`)
On every significant state change (or periodic tick), the model emits:
-   **Target**: `provider_id`, `pool_id`, `scope_id`
-   **TTE Vectors**: `{ p50: "15m", p90: "12m", p99: "8m" }`
-   **Risk**: `{ prob_exhaustion: 0.05, safe_margin: "-2m" }`

These events trigger the **Policy Engine**, which updates the rules (e.g., "Stop approving non-urgent intents").

### "What-If" Analysis (Intent Arbitration)
When an `intent_submitted` arrives:
1.  Daemon creates a temporary "Hypothetical State" ($C_{hyp} = C_{now} - Intent_{cost}$).
2.  Re-runs risk model on $C_{hyp}$.
3.  If $P_{exhaust}$ jumps significantly (Marginal Risk), the intent is **Flagged**.

---

## Shared vs. Isolated Semantics

### Isolated Pools
-   **Input**: Only the specific Agent's past usage.
-   **Forecast**: High confidence, specific to that Agent's behavior.

### Shared Pools
-   **Input**: Aggregate usage of ALL Agents on that Identity/Scope.
-   **Forecast**:
    -   **Global Forecast**: "Will the pool survive total load?" (Used to protect the pool).
    -   **Attributed Contribution**: "How much of the variance is Agent X?" (Used to blame/throttle the noisy neighbor).

*Note: An Agent usually checks BOTH. It must pass the Global Shared Forecast (Pool health) AND its Local Allocation limits.*

---

## Open Questions

-   **Cold Starts**: How to forecast burn rate for a brand new Agent/Identity? (Default priors? Probationary period?).
-   **Seasonality**: Do we model daily/weekly cycles? (Likely Phase 2).
-   **Feedback Loops**: How does the model react when Policy *throttles* usage? (Burn rate drops $\rightarrow$ Forecast improves $\rightarrow$ Policy relaxes $\rightarrow$ Burn rate spikes). We need to model "Unconstrained Demand" vs "Observed Usage".
