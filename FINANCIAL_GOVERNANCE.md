# FINANCIAL_GOVERNANCE: Cost as a Constraint

**Status**: DRAFT
**Owner**: Orchestrator
**Related**: `DATA_MODEL.md`, `POLICY_ENGINE.md`

## 1. Overview

While `ratelord` currently models "requests" and "tokens", the ultimate constraint for many AI systems is **Budget ($)**. This initiative elevates financial cost to a first-class dimension for tracking, policy enforcement, and forecasting.

## 2. Goals

1.  **Cost Awareness**: Agents know the approximate dollar cost of their actions *before* execution.
2.  **Budget Caps**: Hard limits on spend ("Stop agent X if > $10/day").
3.  **Route Optimization**: Select cheaper providers (e.g., GPT-3.5 vs GPT-4) when budget is tight.

## 3. Data Model Extensions

### 3.1 Currency Dimension
We treat currency as just another dimension in the `Usage` struct, but with fixed precision.
- **Unit**: Micro-USD (integers, 1/1,000,000th of a dollar) to avoid float drift.
- **Example**: `1000` = $0.001.

### 3.2 Pricing Registry
A new configuration section mapping usage units to cost.
```json
"pricing": {
  "openai": {
    "gpt-4-input": 3000,  // $0.03 per 1k
    "gpt-4-output": 6000  // $0.06 per 1k
  }
}
```

## 4. Policy Changes

New Rule Types:
- **`budget_cap`**: "Reject if `sum(cost)` > $50 in window `24h`."
- **`cost_efficiency`**: "If `forecast_exhaustion < 1h`, reject high-cost providers."

## 5. Forecasting

- **Burn Rate**: Expressed in `$/hour`.
- **Runway**: "At current spend, budget is exhausted in 3 days."
