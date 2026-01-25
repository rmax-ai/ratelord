# ARCHITECTURE: ratelord

This document describes the ratelord system architecture at a conceptual level. It is docs-first: it defines components, responsibilities, dataflows, and invariants without prescribing implementation.

## Architecture Invariants (Non-Negotiable)

- Local-first, zero-ops: runs on a single machine; local state is authoritative; no required external services beyond the provider APIs being monitored.
- Single write authority: all write decisions and state transitions happen in `ratelord-d` (the daemon). All clients are read-only views.
- Event-sourced + replayable: the append-only event log is the source of truth; all derived state is a projection that can be rebuilt.
- Predictive, not reactive: the primary outputs are forecasts (P50/P90/P99 time-to-exhaustion and risk) in time-domain terms.
- Intent negotiation: actions are expressed as intents; the daemon arbitrates intents (approve/deny/modify) under policy.
- Explicit shared vs isolated: no implicit isolation; all constraints explicitly declare sharing semantics via identities, scopes, and pools.
- Time-domain reasoning: budgets/limits are interpreted as rates over windows; risk is expressed as time-to-exhaustion under uncertainty.

## System Overview

At a high level, `ratelord-d` continuously:

1. Polls and ingests provider signals (usage, limits, reset windows, errors).
2. Normalizes observations into domain events and appends them to the event log.
3. Updates projections/snapshots for fast reads.
4. Produces forecasts (time-to-exhaustion distributions + confidence/risk).
5. Evaluates policy and arbitrates intents.
6. Emits decisions and advisory signals for clients to display.

Clients (TUI/Web) do not mutate state. They subscribe to read models and render:

- Current constraint posture (remaining budget, resets, pools, scopes).
- Forecasts and risk.
- Intent decisions and rationale.

## Components and Responsibilities

### `ratelord-d` (Daemon, Single Authority)

Responsibilities:

- Provider integration orchestration (poll scheduling, cursors, backoff).
- Event ingestion and normalization (validate, de-dup, attribute identity/scope/pool).
- Append-only event log writes (backed by SQLite in WAL mode).
- Projection maintenance (materialized read models).
- Forecasting (burn rates, reset windows, uncertainty modeling).
- Policy evaluation + intent arbitration (approve/approve_with_modifications/deny_with_reason + rationale).
- Signal emission (advisories, alerts, intent decisions, status).

Non-responsibilities:

- Direct user interaction beyond exposing read APIs/streams for clients.
- Any “side-effect” actions outside the daemon’s authority (clients must not perform them).

### Storage (Local State)

Responsibilities:

- Durable event log (source of truth).
- Projection tables (derived read models).
- Periodic snapshots/checkpoints for faster rebuild.
- Integrity metadata (schema versioning, checksums where appropriate).

Constraints:

- Local-only; no required cloud database.
- Must support deterministic replay to reproduce projections and forecasts given the same event stream (subject to time/clock modeling rules defined elsewhere).

### Clients: TUI (Read-Only) and Web UI (Read-Only)

Responsibilities:

- Render projections and forecasts.
- Display intent status (pending/approved/denied/modified) and rationale.
- Provide a safe interface for proposing intents (submission is a request; daemon decides).
- Visualize shared vs isolated semantics (pools, scopes, identities).

Non-responsibilities:

- Writing to storage.
- Calling providers to “fix” state.
- Making policy decisions.

### Provider Integrations (Conceptual)

A provider integration is a logical module owned by `ratelord-d` that:

- Knows how to poll the provider and interpret responses.
- Produces provider observation events (and errors) in a normalized form.
- Declares which constraints and windows the provider exposes (rate limits, quotas, concurrency caps, etc.).

## Core Abstractions

### Constraint Graph (Nodes + Edges)

Ratelord models constraint posture as a typed constraint graph. The graph is a conceptual backbone used for:

- Explaining “why” forecasts and policy decisions exist.
- Making shared vs isolated semantics explicit.
- Enabling consistent projections across providers.

Nodes (examples; exact taxonomy may evolve):

- Provider: a source of observations and a namespace for constraints.
- Identity: an entity that “owns” or is charged for usage (human, service account, org, token, machine).
- Scope: a boundary/context within which usage applies (repo, environment, project, workspace, endpoint class).
- Pool: a shared bucket of constraints/budgets (explicitly models shared vs isolated).
- Constraint: a limit with a window (e.g., per-minute rate limit, daily quota, concurrent sessions cap).
- Window: reset cadence and timing semantics (fixed reset, rolling window, provider-defined).
- Workload/Flow (optional concept): a categorized consumer (agent, job class) used for attribution and policy.

Edges (examples):

- `OBSERVES`: Provider -> Constraint (provider reports this constraint).
- `APPLIES_TO`: Constraint -> Scope (constraint applies within a scope).
- `CHARGES`: UsageEvent -> Identity (attribution).
- `CONSUMES_FROM`: Identity/Scope/Workload -> Pool (where budget is drawn).
- `BOUNDS`: Pool -> Constraint (pool is governed by constraint(s)).
- `SHARED_WITH`: Pool -> Identity (explicit sharing membership).
- `DERIVED_FROM`: Projection -> EventLog (provenance linkage; conceptual).

Graph requirements:

- Every constraint must resolve to exactly one pool (even if the pool is “isolated singleton”).
- Sharing is never implied: membership edges define sharing.
- Scopes are first-class: constraints and forecasts must reference scope where meaningful.

### Identity Layer

Purpose:

- Make attribution and chargeback explicit.
- Disambiguate “who is spending” from “what boundary is being constrained.”

Identity properties (conceptual):

- `identity_id`: stable local identifier.
- `kind`: human / service / org / token / machine / unknown / sentinel (none).
- `provider_refs`: provider-specific identifiers (opaque).
- `labels`: tags for grouping (team, owner, purpose).
- `trust_level` (optional): used by policy (e.g., stricter gating for untrusted identities).

Rules:

- Identity resolution must be deterministic for a given event stream.
- Unknown/ambiguous identities are allowed but must be explicit via sentinel IDs (no nullable/silent merging).
- Every event MUST be attributed to an identity (using `sentinel:global` or similar if no specific identity applies).

### Scope Layer

Purpose:

- Represent “where” usage applies and where policy should be enforced.

Scope properties (conceptual):

- `scope_id`: stable local identifier.
- `kind`: repo / project / env / endpoint-class / workspace / global / other.
- `parent_scope_id` (optional): hierarchical containment.
- `provider_refs`: provider-specific scope identifiers.
- `labels`: e.g., `prod`, `ci`, `sandbox`.

Rules:

- Scope must be explicit in events; if not known or applicable, events attach to a defined sentinel scope (e.g., `sentinel:unknown` or `sentinel:global`).
- Policy must be able to target scopes (allow/deny/shape intents).
- Every event MUST have a scope; null is not permitted for scope dimensions.

### Workload Layer

Purpose:

- Represent the logical action or flow being performed for attribution and policy.

Workload properties (conceptual):

- `workload_id`: stable local identifier.
- `name`: human-readable label.
- `category`: e.g., `agent-triage`, `sync-background`, `interactive-user`.

Rules:

- Every event MUST be attributed to a workload; use `sentinel:system` for daemon-internal work.
- Workload is a mandatory scope dimension (Agent + Identity + Workload + Scope).

### Constraint Pools

Purpose:

- Encode shared vs isolated budgets in a single explicit construct.

Pool properties (conceptual):

- `pool_id`: stable local identifier.
- `sharing`: `isolated` or `shared`.
- `members`: identities (and optionally scopes) that draw from the pool.
- `constraints`: one or more constraints governing the pool.
- `priority`/`tier` (optional): used for policy (e.g., reserve capacity for prod).

Rules:

- If two identities share the same real-world budget, they must be in the same pool.
- If an identity has an isolated budget, it must be a singleton pool (`sharing=isolated`).

## End-to-End Dataflow

### 1) Provider Polling (Ingress)

- `ratelord-d` schedules polls per provider with explicit cadence and backoff.
- Poll outputs are treated as observations, not truth: they may be stale, partial, or inconsistent.

Typical observation categories:

- Limit definitions (rate/quota, windows, reset times).
- Current usage counters or remaining budget.
- Error conditions (timeouts, auth failures, 429s, provider incidents).

### 2) Normalize to Events (Event-Sourcing Boundary)

- Observations are normalized into domain events and appended to the event log.
- Events must be immutable and self-describing (type + schema version + payload).
- De-duplication and idempotency occur at ingestion (e.g., provider cursor + hash of observation).

### 3) Projections / Snapshots (Derived Read Models)

- Projections are built from events and optimized for queries:
  - “Current posture” (latest known remaining/reset/burn).
  - “Recent history” (time series).
  - “Identity/scope/pool mapping” views.
- Snapshots are checkpoints to accelerate replay; they never replace the event log.

### 4) Prediction (Forecasting)

Forecasts are computed per (pool, constraint, scope as applicable) and expressed as:

- `time_to_exhaustion`: P50/P90/P99 estimates.
- `time_to_safe_zone` (optional): time until risk drops below a threshold (e.g., after reset).
- `risk`: probability of exhaustion before next reset window boundary.
- `assumptions`: burn-rate model, window type, data freshness, confidence.

Time-domain inputs:

- Reset windows (fixed time, rolling, unknown).
- Burn rate over time (recent, weighted, seasonal adjustments if later defined).
- Uncertainty due to partial observations and degraded mode.

### 5) Policy Evaluation

- Policy consumes forecasts + posture + identity/scope context.
- Policy produces constraints on behavior, not direct action:
  - gating thresholds (e.g., deny if P90 exhaustion < 30m),
  - shaping (e.g., reduce concurrency),
  - prioritization (e.g., reserve pool budget for prod).

### 6) Intent Negotiation and Arbitration

- Actors (users/tools/agents) submit intents, not commands.
- The daemon arbitrates:
  - `approve`: intent fits policy/risk envelope.
  - `approve_with_modifications`: intent is acceptable with modifications (e.g., lower rate, smaller scope, later start time deferral).
  - `deny_with_reason`: intent violates policy; include explicit reason.
- Decisions are emitted as events, enabling auditability and replay of “why we did/didn’t allow X.”

### 7) Signals and Client Updates

- Signals are read-model outputs intended for display and automation:
  - advisories (yellow/red posture),
  - incident-style messages (provider unreachable),
  - upcoming resets and recommended pacing,
  - intent decision stream with rationale.
- Clients subscribe and render. Any “action” remains outside clients; they can only propose new intents.

## Data Concepts (Conceptual, No Implementation Detail)

Storage is local (SQLite WAL), event-log-centric, and append-only. All derived state (projections) must be rebuildable from the log.

### Event Log (Source of Truth)

The event log is the immutable record of all system activity. Every entry represents a point-in-time fact or decision.

Key Invariants:
- **Append-only**: Events are never modified or deleted.
- **Versioning**: Every event carries a schema version to allow for evolution.
- **Provenance**: Events record their causation (what event triggered this) and correlation (what flow this belongs to).
- **Mandatory Scoping**: No event is unscoped. Every event MUST carry identifiers for Agent, Identity, Workload, and Scope. Sentinel IDs (e.g., `sentinel:global`, `sentinel:system`) are used where specific entities are not applicable or unknown. Nulls are not permitted for these core dimensions.

Illustrative fields (non-normative):
- `event_id`: unique, monotonic.
- `ts_event`: logical time.
- `ts_ingest`: daemon physical time.
- `type`: event category.
- `actor_id`: the entity that produced the event.
- `dimensions`: { agent_id, identity_id, workload_id, scope_id, pool_id, constraint_id }.
- `payload`: versioned structured data.

### Projections (Derived Read Models)

Projections are materialized views of the event log optimized for specific query patterns (e.g., current posture, forecasts, intent history).

Key Invariants:
- **Rebuildable**: Projections can be dropped and fully reconstructed from the event log.
- **Snapshots**: Periodic snapshots may be used as optional accelerators to speed up projection rebuilding, but they are never the source of truth.
- **Staleness**: Projections should track their own staleness/freshness relative to the event log high-water mark.

### Identity, Scope, and Pool Projections

These read models maintain the current state of the constraint graph nodes, including:
- **Identity status**: kinds, labels, and provider references.
- **Scope hierarchy**: repo/org/global containment.
- **Pool membership**: explicit mapping of identities/scopes to shared or isolated capacity buckets.
- **Constraint posture**: current remaining value, reset windows, and observed burn rates.
- **Forecasts**: predicted time-to-exhaustion (TTE) quantiles and risk metrics.
- **Intent status**: the current state of submitted intents and their arbitration results.

## Operational Modes

### Online (Steady State)

Characteristics:

- Provider polling succeeds at normal cadence.
- Projections update continuously.
- Forecasts have higher confidence; risk is based on recent burn and known windows.
- Intents are evaluated against current posture and forecasts.

Expected outputs:

- Up-to-date remaining budgets and reset windows.
- Stable time-to-exhaustion distributions.
- Clear policy gating.

### Degraded (Provider Unavailable or Partial)

Triggers:

- Provider unreachable, auth failures, or responses missing critical fields.
- Local resource constraints (e.g., disk pressure) affecting ingestion/projection.

Behavior:

- Ingest error events and update provider status projection.
- Continue forecasting with increased uncertainty using last-known posture + conservative assumptions.
- Policy can tighten gating automatically (e.g., require larger safety margin, deny high-risk intents).
- Clients render “staleness” prominently and show rationale for conservative decisions.

Key principle:

- Degraded mode is explicit and auditable (events + projections), not an implicit silent failure.

### Replay / Rebuild (Event Log as Truth)

Use cases:

- Schema evolution of projections/forecasts.
- Recovery after corruption of derived tables.
- Auditing and debugging.

Behavior:

- Rebuild projections from `event_log` (optionally from a snapshot checkpoint).
- Deterministic replay for posture projections; forecasting determinism depends on explicitly versioned models and time semantics.
- Resulting projections must match the same event stream and same model versions.

## Failure Modes and Mitigations

### Provider Failures

- Mode: timeouts, 5xx, rate limiting, partial data, inconsistent reset times.
- Mitigations:
  - Backoff + jitter; cap poll frequency.
  - Record provider status and error events.
  - Prefer conservative forecasts under uncertainty.
  - De-dup observations; tolerate out-of-order updates via event time vs ingest time.

### Identity / Scope Misattribution

- Mode: events attributed to wrong identity/scope, or ambiguous mapping.
- Mitigations:
  - Make mapping rules explicit and versioned.
  - Allow “unknown identity/scope” placeholders; never auto-merge silently.
  - Emit mapping-change events; projections show effective mapping and provenance.

### Shared vs Isolated Misconfiguration

- Mode: pool membership wrong; shared budget treated as isolated (or vice versa).
- Mitigations:
  - Require explicit pool definitions and memberships.
  - Provide “explain” views in clients (why a constraint is shared).
  - Policy can detect suspicious patterns (e.g., two identities exhausting “independent” pools simultaneously).

### Event Log Integrity / Local Storage Issues

- Mode: disk full, file corruption, partial writes.
- Mitigations:
  - Atomic append discipline and integrity metadata.
  - Clear “read-only safe mode” if writes cannot be guaranteed.
  - Snapshots as acceleration only; never as the sole source of truth.
  - Operator-facing signals: disk pressure, write failures, rebuild required.

### Clock / Time Semantics Issues

- Mode: clock skew, DST changes, provider timestamps inconsistent with local time.
- Mitigations:
  - Preserve both provider-reported times and daemon ingest time.
  - Treat reset windows as provider-defined when available; otherwise mark unknown.
  - Forecasts include `as_of_ts` and confidence/staleness flags.

### Policy or Forecast Model Bugs

- Mode: overly permissive or overly restrictive decisions.
- Mitigations:
  - Version policy evaluation and forecast models; record model versions in events/projections.
  - Record intent decisions with rationale and inputs summary.
  - Support replay to reproduce decisions under the same versions.

### Client Version / Schema Drift

- Mode: clients misinterpret projections.
- Mitigations:
  - Versioned read APIs and projection schemas.
  - Clients treat unknown fields as opaque; show “incompatible” warnings rather than guessing.

## Security and Privacy Posture (Local-First)

- Secrets (provider tokens, keys) are local-only and never written into the event payload in raw form.
- Events should avoid storing full provider responses unless explicitly scrubbed and justified.
- Identity references in events should be stable local IDs plus opaque provider refs where needed; clients render friendly labels from projections.

## Open Questions / TODOs

- Define the canonical constraint graph taxonomy: exact node/edge types and minimum required edges for “explainability.”
- Formalize window semantics across providers (fixed reset vs rolling) and how to represent “unknown window” in forecasts.
- Specify the minimal event type set and required fields per event type (including schema versioning strategy).
- Decide how deterministic forecasting must be under replay (e.g., do we pin model randomness; how do we handle “now”).
- Define how intents are shaped (rate, concurrency, scope narrowing, start time deferral) in a provider-agnostic way.
- Decide on the boundary between policy and prediction (what belongs in forecast vs in policy thresholds).
- Clarify scope hierarchy rules (inheritance, aggregation) and how pools interact with scope containment.
- Define the “explain” contract: what provenance/rationale must be available to clients for posture, forecasts, and intent decisions.
- Establish retention/compaction guidance for event logs (local disk constraints) while preserving auditability.
