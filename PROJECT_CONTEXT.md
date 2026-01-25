# PROJECT CONTEXT: ratelord

## Project: **ratelord**

### Role of the Orchestrator (Implementation Phase)

You are an **orchestrator agent** responsible for the active **implementation, maintenance, and verification** of the `ratelord` system.

Your job is to:

*   **Implement and Maintain Code**: Write high-quality, testable Go code that adheres to the specs.
*   **Enforce Conceptual Consistency**: Ensure all code changes align with `AGENTS.md` and `PROJECT_CONTEXT.md`.
*   **Verify System Behavior**: Use tests, simulations (`ratelord-sim`), and the TUI to validate correctness.
*   **Prevent Drift**: Ensure documentation (`specs/`) and code (`pkg/`) remain in sync.

---

## 1. Project intent

**ratelord** is a **local-first constraint control plane** for agentic and human-driven software systems.

Its initial provider is GitHub API rate limits, but the system is explicitly designed to generalize to **any hard constraint**:

*   rate limits
*   token budgets
*   monetary spend
*   time / latency

`ratelord` does **not** merely observe limits. It **models, predicts, governs, and shapes behavior** under constraints.

---

## 2. Core problem statement

Modern autonomous systems fail because they:

*   treat limits as errors instead of signals,
*   react only after exhaustion,
*   lack forecasting and risk modeling,
*   assume flat or isolated quotas that do not exist in reality.

In real systems:

*   limits may apply **per agent**
*   per API key / identity
*   per action
*   per repo / org
*   or be **shared across all of the above**

`ratelord` addresses this by governing a **hierarchical constraint graph**, not a flat table of counters.

---

## 3. Foundational principles (immutable)

These principles must be reflected in all documents and designs:

1.  **Local-first, zero-ops**
2.  **Daemon as single authority**
3.  **Event-sourced, replayable**
4.  **Predictive, not reactive**
5.  **Constraints are first-class primitives**
6.  **Agents must negotiate intent before acting**
7.  **Shared vs isolated limits must be explicit**
8.  **Time-domain reasoning over raw counts**

---

## 4. Conceptual model (critical)

### 4.1 Constraint graph (core abstraction)

`ratelord` models constraints as a **directed graph**:

*   **Actors**: agents
*   **Identities**: API keys, GitHub Apps, OAuth tokens
*   **Workloads**: actions / tasks
*   **Scopes**: repo, org, account, global
*   **Constraint pools**: REST, GraphQL, Search, etc.

Requests may consume from **multiple pools simultaneously**, some isolated, some shared.

> Never assume “one agent = one limit”.

---

### 4.2 Identity layer (first-class)

Each identity has:

*   type (PAT, App, OAuth, etc.)
*   owner (agent, system, org)
*   scope (repo, org, account)
*   isolation semantics (exclusive vs shared pools)

Identities are explicitly registered with the daemon.

---

### 4.3 Scope layer (mandatory)

Every event and decision is scoped:

*   agent
*   identity
*   action
*   repo / org
*   global (backend-enforced caps)

No unscoped data is allowed.

---

## 5. System architecture

### Components

**Daemon (`ratelord-d`)**

*   Polls constraint providers
*   Stores events and derived state in SQLite (WAL)
*   Computes burn rates, variance, forecasts
*   Evaluates policies
*   Arbitrates agent intents
*   Emits alerts and control signals

**Storage**

*   SQLite event log (source of truth)
*   Derived snapshots and metrics
*   Time-series optimized

**Clients**

*   **TUI**: operational, real-time, attribution-aware
*   **Web UI**: historical analysis, scenario simulation

Clients are **read-only**; all authority lives in the daemon.

---

## 6. Data philosophy

### Event sourcing is mandatory

Everything is an event:

*   poll
*   reset
*   spike
*   forecast
*   intent_approved / denied
*   policy_trigger
*   throttle

Snapshots and metrics are **derived views**, not truth.

---

## 7. Prediction model

*   Burn rate via EMA (baseline)
*   Track variance / uncertainty
*   Forecast:
    *   P50 / P90 / P99 time-to-exhaustion
    *   Probability of exhaustion before reset

Predictions are computed at **multiple levels**:

*   identity-local
*   shared pool
*   org-level
*   global

Approval requires all relevant forecasts to be safe.

---

## 8. Policy and governance model

Policies are declarative and hierarchical:

*   **Hard rules** (never violate)
*   **Soft rules** (optimization goals)
*   **Local rules** (agent / identity)
*   **Global rules** (system safety)

Policies may:

*   notify
*   throttle
*   deny intents
*   force adaptation

This forms a **constitutional layer for autonomy**.

---

## 9. Agent interaction contract (non-optional)

Agents must submit **intents before acting**.

Each intent declares:

*   agent ID
*   identity to be used
*   action type
*   scope(s)
*   expected consumption
*   duration / urgency

Daemon responses:

*   approve
*   approve_with_modifications
*   deny_with_reason

Agents must adapt behavior accordingly.

---

## 10. Adaptive behavior

The system actively reshapes execution:

*   route load across identities
*   shift REST ↔ GraphQL
*   reduce polling frequency
*   defer non-urgent work
*   degrade gracefully

Constraints are **feedback signals**, not blockers.

---

## 11. Attribution and accountability

Every event includes:

*   agent_id
*   identity_id
*   action_id
*   scope
*   constraint pool

This enables:

*   root cause analysis
*   conflict detection
*   automatic postmortems
*   learning optimal strategies
