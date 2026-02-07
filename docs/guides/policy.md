# Policy Guide

The Policy Engine acts as the "judge" of `ratelord`. While the Prediction Engine forecasts risk, the Policy Engine decides what to do about it.

Policies allow you to govern **future risk** rather than just past consumption. This guide explains how to define, structure, and manage your rate limiting policies.

## 1. Concepts

### Hierarchy of Authority

Policies exist in a strict hierarchy. A lower-level policy cannot permit what a higher-level policy forbids. The engine evaluates rules from top to bottom:

1.  **Global Safety** (System-wide Hard Rules): Prevent catastrophic exhaustion.
2.  **Organization / Scope** (Business Rules): Enforce contracts or environmental limits.
3.  **Pool / Resource** (Provider constraints): manage specific API quotas.
4.  **Identity / Agent** (Local optimization): Ensure fairness among agents.

### Actions vs. Decisions

A **Rule Action** describes what to do when a specific rule matches (e.g., "shape traffic"). The **Policy Engine Decision** is the final outcome returned to the agent after evaluating all rules.

| Rule Action | Description | Behavior |
| :--- | :--- | :--- |
| **approve** | Risk is acceptable. | Agent proceeds immediately. |
| **shape** | Risk is elevated. | Agent must wait `wait_seconds` (throttle) before proceeding. Result: `approve_with_modifications`. |
| **defer** | Risk is high, but transient. | Agent must wait until **after** the next reset. Result: `approve_with_modifications`. |
| **deny** | Risk is critical or rule violated. | Agent must abort the action completely. Result: `deny_with_reason`. |
| **switch** | Pool is exhausted/risky. | Agent must retry using a different `identity_id` (fallback). |

Note: Code uses lowercase strings for actions (e.g., `"shape"`, `"deny"`).

## 2. Policy Structure

Policies are defined in a declarative YAML file (e.g., `policy.yaml`). A policy consists of:

*   **Target**: The scope, pool, or identity the policy applies to.
*   **Rules**: A list of logic predicates and actions.

### The Rule Object

Each rule contains:

*   **Condition**: A logic expression based on Forecasts, Metadata, or Time.
*   **Action**: What to do if the condition matches.
*   **TimeWindow**: (Optional) specific days/hours when this rule applies.
*   **Priority**: Precedence order (higher numbers evaluated first).

## 3. Writing Policy Rules

Conditions can evaluate various metrics:

*   **Risk**: `risk.p_exhaustion` (probability of hitting 0), `margin.seconds`.
*   **Pool**: `pool.remaining_percent`, `pool.utilization`.
*   **Identity**: `agent.role` (e.g., 'prod', 'ci'), `identity.burn_rate_share`.
*   **Time**: `time.is_business_hours`.

### Examples

#### Hard Safety Net (Global)
Prevent the system from hitting a hard limit.

```yaml
- name: "prevent-exhaustion"
  condition: "risk.p_exhaustion > 0.99"
  action: "deny"
```

#### Temporal Rules
Apply limits only during business hours.

```yaml
- name: "business-hours-only"
  action: "deny"
  time_window:
    start_time: "09:00"
    end_time: "17:00"
    days: ["Mon", "Tue", "Wed", "Thu", "Fri"]
    timezone: "UTC"
```

#### Soft Throttling (Business Logic)
Slow down development environments when the pool is half empty.

```yaml
- name: "slow-down-devs"
  condition: "agent.role == 'dev' AND pool.utilization > 0.50"
  action: "shape"
  params:
    factor: 2.0  # Linear backoff multiplier
```

#### Fairness (Identity)
Prevent one agent from hogging the pool.

```yaml
- name: "fair-share"
  condition: "identity.burn_rate_share > 0.50"
  action: "shape"
```

## 4. Complete Policy Example

Here is a comprehensive `policy.yaml` example demonstrating different levels of control.

```yaml
policies:
  # 1. Global Safety Net (Highest Priority)
  - id: "global-safety"
    scope: "global"
    type: "hard"
    rules:
      - name: "emergency-stop"
        # If we have less than 5 seconds of margin, stop everything.
        condition: "margin.seconds < 5"
        action: "deny"
        priority: 100

      - name: "defer-highly-risky"
        # If exhaustion is certain (>99%), wait for reset.
        condition: "risk.p_exhaustion > 0.99"
        action: "defer"
        priority: 90

  # 2. Development Environment Constraints
  - id: "dev-environment"
    scope: "env:dev"
    type: "soft"
    rules:
      - name: "throttle-dev-usage"
        # If dev uses > 50% of pool, slow them down.
        condition: "pool.utilization > 0.50"
        action: "shape"
        params:
          algorithm: "linear"
          factor: 2.0
        priority: 50

  # 3. Production Optimizations
  - id: "prod-environment"
    scope: "env:prod"
    type: "soft"
    rules:
      - name: "protect-prod-traffic"
        # Allow prod to burn until very high risk.
        condition: "risk.p_exhaustion < 0.90"
        action: "approve"
        priority: 60
```

## 5. Shared vs. Isolated Pools

One of the most powerful features of Ratelord is the ability to model both shared resources and isolated quotas using **Identities** and **Scopes**.

### How it works

*   **Shared Pool**: If multiple Agents use the **same** `identity_id` (e.g., they all sign requests with the same `gh-pat-team` credential), they will automatically share the rate limit pool associated with that credential. If Agent A burns 90% of the limit, Agent B only has 10% left.
*   **Isolated Pool**: If each Agent uses a **unique** `identity_id` (e.g., `user:alice` vs `user:bob`), they each get their own tracking context. However, you can still enforce a shared "global" limit on top of them using a higher-level Policy Scope.

### Example: Shared Organization Limit

This policy tracks a shared resource (an Organization-wide API limit) regardless of which specific agent is making the call.

```yaml
- id: "shared-org-limit"
  scope: "org:my-company"
  type: "hard"
  limit: 10000
  rules:
    - name: "enforce-shared-limit"
      # This condition applies to the aggregate usage of ALL agents
      # operating within the 'org:my-company' scope.
      condition: "pool.remaining < 100"
      action: "deny"
      params:
        reason: "Organization-wide limit exhausted"
```
