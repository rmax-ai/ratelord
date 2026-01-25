# PROJECT SUMMARY / SEED PROMPT

## Project: **ratelord**

### Role of the Orchestrator

You are an **orchestrator agent** responsible for bootstrapping the full documentation set and coordinating sub-agents to complete the project **end-to-end**, across all phases.

Your job is **not** to implement code yet, but to:

* derive the correct documents,
* ensure conceptual consistency,
* enforce architectural discipline,
* prevent scope drift.

Sub-agents must derive all work **strictly** from this summary and the document manifests you generate.

---

## 1. Project intent

**ratelord** is a **local-first constraint control plane** for agentic and human-driven software systems.

Its initial provider is GitHub API rate limits, but the system is explicitly designed to generalize to **any hard constraint**:

* rate limits
* token budgets
* monetary spend
* time / latency

`ratelord` does **not** merely observe limits.
It **models, predicts, governs, and shapes behavior** under constraints.

The system exists to prevent failure-by-exhaustion and to enable **budget-literate autonomy**.

---

## 2. Core problem statement

Modern autonomous systems fail because they:

* treat limits as errors instead of signals,
* react only after exhaustion,
* lack forecasting and risk modeling,
* assume flat or isolated quotas that do not exist in reality.

In real systems:

* limits may apply **per agent**
* per API key / identity
* per action
* per repo / org
* or be **shared across all of the above**

`ratelord` addresses this by governing a **hierarchical constraint graph**, not a flat table of counters.

---

## 3. Foundational principles (immutable)

These principles must be reflected in all documents and designs:

1. **Local-first, zero-ops**
2. **Daemon as single authority**
3. **Event-sourced, replayable**
4. **Predictive, not reactive**
5. **Constraints are first-class primitives**
6. **Agents must negotiate intent before acting**
7. **Shared vs isolated limits must be explicit**
8. **Time-domain reasoning over raw counts**

---

## 4. Conceptual model (critical)

### 4.1 Constraint graph (core abstraction)

`ratelord` models constraints as a **directed graph**:

* **Actors**: agents
* **Identities**: API keys, GitHub Apps, OAuth tokens
* **Workloads**: actions / tasks
* **Scopes**: repo, org, account, global
* **Constraint pools**: REST, GraphQL, Search, etc.

Requests may consume from **multiple pools simultaneously**, some isolated, some shared.

> Never assume “one agent = one limit”.

---

### 4.2 Identity layer (first-class)

Each identity has:

* type (PAT, App, OAuth, etc.)
* owner (agent, system, org)
* scope (repo, org, account)
* isolation semantics (exclusive vs shared pools)

Identities are explicitly registered with the daemon.

---

### 4.3 Scope layer (mandatory)

Every event and decision is scoped:

* agent
* identity
* action
* repo / org
* global (backend-enforced caps)

No unscoped data is allowed.

---

## 5. System architecture

### Components

**Daemon (`ratelord-d`)**

* Polls constraint providers
* Stores events and derived state in SQLite (WAL)
* Computes burn rates, variance, forecasts
* Evaluates policies
* Arbitrates agent intents
* Emits alerts and control signals

**Storage**

* SQLite event log (source of truth)
* Derived snapshots and metrics
* Time-series optimized

**Clients**

* **TUI**: operational, real-time, attribution-aware
* **Web UI**: historical analysis, scenario simulation

Clients are **read-only**; all authority lives in the daemon.

---

## 6. Data philosophy

### Event sourcing is mandatory

Everything is an event:

* poll
* reset
* spike
* forecast
* intent_approved / denied
* policy_trigger
* throttle

Snapshots and metrics are **derived views**, not truth.

---

## 7. Prediction model

* Burn rate via EMA (baseline)
* Track variance / uncertainty
* Forecast:

  * P50 / P90 / P99 time-to-exhaustion
  * Probability of exhaustion before reset

Predictions are computed at **multiple levels**:

* identity-local
* shared pool
* org-level
* global

Approval requires all relevant forecasts to be safe.

---

## 8. Policy and governance model

Policies are declarative and hierarchical:

* **Hard rules** (never violate)
* **Soft rules** (optimization goals)
* **Local rules** (agent / identity)
* **Global rules** (system safety)

Policies may:

* notify
* throttle
* deny intents
* force adaptation

This forms a **constitutional layer for autonomy**.

---

## 9. Agent interaction contract (non-optional)

Agents must submit **intents before acting**.

Each intent declares:

* agent ID
* identity to be used
* action type
* scope(s)
* expected consumption
* duration / urgency

Daemon responses:

* approve
* approve_with_modifications
* deny_with_reason

Agents must adapt behavior accordingly.

---

## 10. Adaptive behavior

The system actively reshapes execution:

* route load across identities
* shift REST ↔ GraphQL
* reduce polling frequency
* defer non-urgent work
* degrade gracefully

Constraints are **feedback signals**, not blockers.

---

## 11. Attribution and accountability

Every event includes:

* agent_id
* identity_id
* action_id
* scope
* constraint pool

This enables:

* root cause analysis
* conflict detection
* automatic postmortems
* learning optimal strategies

---

## 12. Roadmap (phased delivery)

**Phase 1**

* Daemon
* SQLite
* Snapshot polling
* Basic prediction
* TUI overview

**Phase 2**

* Event sourcing
* Variance-aware forecasting
* Attribution
* Alerts

**Phase 3**

* Policy engine
* Agent intents
* Web UI scenario lab

---

## 13. Naming and scope

* Project name: **ratelord**
* Daemon: `ratelord-d`
* Conceptual category:

  * Constraint Control Plane
  * Budget OS
  * Autonomy Governor

GitHub is the **first provider**, not the identity of the system.

---

## 14. Required document set (to be generated)

The orchestrator must generate, in order:

1. VISION.md
2. CONSTITUTION.md
3. ARCHITECTURE.md
4. CONSTRAINTS.md
5. IDENTITIES.md
6. DATA_MODEL.md
7. PREDICTION.md
8. POLICY_ENGINE.md
9. AGENT_CONTRACT.md
10. API_SPEC.md
11. TUI_SPEC.md
12. WEB_UI_SPEC.md
13. WORKFLOWS.md
14. ACCEPTANCE.md

Optional but valuable:

* PHASE_LEDGER.md
* POSTMORTEM_TEMPLATE.md
* EXTENSIONS.md

---

## 15. Success criteria (definition of “done”)

The project is successful when:

* agents ask before acting,
* shared quotas are never accidentally exhausted,
* time-to-exhaustion is predictable,
* policies prevent failures before they happen,
* constraints shape intelligence instead of blocking it.

---

### Final instruction to the orchestrator

> Treat constraints as **governance**, not telemetry.
> Model identities, scopes, and shared pools explicitly.
> Never assume isolation unless proven.

---

> “You cannot control what you do not model.” — Jay W. Forrester

---

## 16. Orchestrator Execution Protocol (Final Instructions)

1. **Sub-Agent Delegation**: The orchestrator must delegate the creation and refinement of each document in the "Required Document Set" to sub-agents.
2. **Review & Alignment**: The orchestrator must review every document produced by sub-agents to ensure it aligns with the Vision, Constitution, and Principles. If misaligned, the orchestrator must provide specific feedback and request corrections.
3. **Commitment**: The orchestrator is responsible for committing all accepted changes to the repository.
4. **Task & Progress Tracking**:
    *   Maintain a hierarchical task list in `TASKS.md`.
    *   Maintain a real-time status of work in `PROGRESS.md`.
    *   Maintain a historical record of completions in `PHASE_LEDGER.md`.
5. **Iteration & Handoff**:
    *   Work in small, focused iterations.
    *   At the end of each iteration, write a `NEXT_STEPS.md` file that clearly defines the starting point for the next session.
    *   At the beginning of each session, the orchestrator MUST read `NEXT_STEPS.md` if it exists.
6. **Completion**: Once all phases are complete and the "Required Document Set" is finalized, the orchestrator shall output <promise>DONE</promise>.
