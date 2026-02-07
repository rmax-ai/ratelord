# Core Data Model

Ratelord uses a graph-based model to represent who is doing what, where they are doing it, and what limits apply. This ensures that complex relationships (like shared organizational budgets vs. personal API keys) are modeled explicitly.

## The Constraint Graph

Imagine a network where "Identities" (users) act within "Scopes" (projects), drawing from "Pools" (budgets) that are governed by "Constraints" (limits). This is the **Constraint Graph**.

### 1. Identity ("Who")
An **Identity** represents an actor that consumes resources.
- **Examples**: `user:alice`, `service:ci-bot`, `token:gh-pat-123`.
- **Function**: Usage is attributed to an identity. This allows you to track "who spent the budget."

### 2. Scope ("Where")
A **Scope** represents the boundary or context of the action.
- **Examples**: `repo:owner/name`, `project:backend-api`, `global`.
- **Function**: Allows granular policy. You might allow aggressive usage in `scope:dev` but restrict it in `scope:prod`.

### 3. Workload ("What")
A **Workload** describes the logical task being performed.
- **Examples**: `scan-dependencies`, `generate-docs`, `triage-issues`.
- **Function**: Helps categorize usage patterns. A "deep scan" workload might have a different burn rate profile than a "quick check."

### 4. Pool ("The Budget")
A **Pool** is a bucket of capacity. This is a critical concept for modeling **Shared vs. Isolated** resources.
- **Shared Pool**: Multiple identities draw from the same pool. (e.g., An Organization API limit shared by all developers).
- **Isolated Pool**: An identity has its own dedicated pool. (e.g., A personal Per-User rate limit).
- **Rule**: Every constraint must resolve to exactly one Pool.

### 5. Constraint ("The Rule")
A **Constraint** defines the limit and the reset window.
- **Properties**:
  - `limit`: The max value (e.g., 5000 requests).
  - `window`: The time period (e.g., 1 hour).
  - `reset`: When the window restarts (Fixed time or Rolling).

## Attribution Invariants

Ratelord requires **strict attribution**. Every event in the system must answer four questions:

1. **Agent ID**: Which software agent initiated this?
2. **Identity ID**: Which credentials/user is responsible?
3. **Workload ID**: What task is being run?
4. **Scope ID**: Where is this happening?

If any of these are unknown, specific **Sentinels** (like `sentinel:unknown` or `sentinel:global`) are used. Null values are not permitted. This ensures that the Event Log is always queryable and complete.
