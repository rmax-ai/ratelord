# DATA MODEL: ratelord

This document defines the conceptual data model for `ratelord` with a strict event-sourcing contract. It is docs-first and intentionally avoids implementation details (no SQL schema).

## Invariants

- `ratelord-d` is the only write authority. Clients are read-only.
- The append-only event log is the source of truth; all projections/read-models are derived and rebuildable.
- Every event is scoped: `agent_id`, `identity_id`, `workload_id`, `scope_id` are mandatory dimensions; null is never allowed for these.
- Time-domain reasoning is primary: reset windows, burn rates, and time-to-exhaustion drive decisions.
- Secrets never appear in event payloads (no raw tokens, no credential material).

---

## Event-Sourcing Contract

### What counts as “truth”

- The event log is the authoritative record of:
  - provider observations and errors
  - normalized constraint/usage/reset observations
  - computed forecasts
  - submitted intents and daemon decisions
  - policy triggers and advisories
- Projections (snapshots, “current posture”, provider status, forecasts tables) are optimizations only.

### Replay requirement

Given the same ordered event stream (and the same model/policy versions recorded in events), `ratelord-d` must be able to rebuild:

- constraint posture read models
- provider status read models
- intent history read models
- forecast read models (subject to explicitly versioned model semantics)

---

## Canonical Event Envelope (Conceptual)

Every event is an immutable record with a canonical envelope plus a type-specific payload.

### Required fields (envelope)

- `event_id`: unique identifier for this event (stable; immutable)
- `event_type`: one of the core event types (initial set below)
- `schema_version`: integer, scoped to `event_type` (payload schema version)
- `ts_event`: event time (logical time of the thing being recorded)
- `ts_ingest`: ingest time (when `ratelord-d` accepted/appended the event)
- `source` (origin metadata; writer is always `ratelord-d`):
  - `origin_kind`: `daemon` | `provider` | `client` | `operator` (who/what originated the observation/request)
  - `origin_id`: identifier for the origin (e.g., provider integration name, client tool id, operator id)
  - `writer_id`: always `ratelord-d` for persisted events
- `dimensions` (mandatory, non-null):
  - `agent_id`
  - `identity_id`
  - `workload_id`
  - `scope_id`
- `correlation`:
  - `correlation_id`: groups events belonging to the same logical flow (poll cycle, intent evaluation, incident)
  - `causation_id`: the immediate causal parent `event_id` (or a sentinel when none)
- `payload`: a structured object whose meaning is defined by `(event_type, schema_version)`

### Mandatory dimensions

These four dimensions exist on every event, even if “not applicable”:

- `agent_id`: who is responsible for the activity (or system sentinel)
- `identity_id`: what identity is charged/associated (or sentinel)
- `workload_id`: what class of work is being performed (or sentinel)
- `scope_id`: what boundary the event applies to (repo/org/account/global/unknown sentinel)

If the daemon cannot determine a specific value, it must choose an explicit sentinel rather than omitting or nulling.

### Sentinel IDs (non-exhaustive)

Sentinels are ordinary IDs with reserved meaning; they make “unknown / not applicable” explicit and queryable.

- `sentinel:system`: daemon-internal activity (housekeeping, polling, projection rebuild)
- `sentinel:global`: global/root scope or global identity context when applicable
- `sentinel:unknown`: value not known at ingest time (allowed, but should be minimized)

Rule: prefer a specific sentinel (`sentinel:system`, `sentinel:global`) when appropriate; otherwise use `sentinel:unknown`.

### Event vs ingest time

- `ts_event` answers: “When did the observed thing happen (or when does it semantically apply)?”
- `ts_ingest` answers: “When did `ratelord-d` learn about it and commit it to the log?”

Rationale:

- providers can report stale data, delayed resets, or out-of-order observations
- local clocks can drift; ingest time is still needed for freshness/staleness reasoning

### Optional (strongly recommended) envelope fields

These are not mandatory for the conceptual model, but expected to exist in practice:

- `provider_id`: which provider namespace applies (e.g., `github`)
- `pool_id`: which constraint pool is impacted (REST/GraphQL/Search/etc.)
- `constraint_id`: which constraint/window definition applies
- `dedupe_key`: idempotency key for ingestion (e.g., provider cursor + observation hash)
- `severity`: for error/advisory style events (`info`/`warn`/`error`)
- `redaction`: indicates payload scrubbing decisions (`scrubbed_fields`, `scrubbed_reason`)

---

## Core Event Types (Initial Set)

Event types are minimal, provider-agnostic, and intentionally describe “what happened” rather than “how it is stored”.

### `provider_poll_observed`

A provider poll occurred and produced an observation.

Payload (typical):

- `provider_id`
- `poll_id`: unique identifier for this poll cycle
- `state`: opaque provider-specific state blob (cursors, watermarks) for next poll
- `status`: success/partial
- `observation_summary`: high-level counts/markers (not raw secrets)
- `raw_ref`: optional reference to locally stored raw data if explicitly enabled and scrubbed

### `provider_error`

A provider interaction failed or degraded.

Payload (typical):

- `provider_id`
- `error_kind`: timeout/auth/5xx/429/parse/other
- `retry_after`: optional duration (if known)
- `message`: scrubbed, non-sensitive description
- `context`: poll/endpoint class identifiers (scrubbed)

### `constraint_observed`

A constraint definition or limit signal was observed/updated (capacity, window shape, known limits).

Payload (typical):

- `provider_id`
- `pool_id`
- `constraint_id`
- `limit`: capacity/ceiling (units and value)
- `window`: window type and reset semantics (fixed/rolling/unknown) and any provider-reported reset anchor
- `as_reported_by_provider`: timestamps/fields needed to interpret the observation (no secrets)

### `reset_observed`

A reset boundary was observed (provider-reported reset time, actual reset detection, or inferred reset).

Payload (typical):

- `provider_id`
- `pool_id`
- `constraint_id`
- `reset_at`: provider-reported or inferred reset timestamp
- `reset_kind`: provider_reported | inferred | detected
- `uncertainty`: optional bounds/notes (jitter, skew)

### `usage_observed`

Consumption/usage signal was observed (remaining units, used units, request cost, etc.).

Payload (typical):

- `provider_id`
- `pool_id`
- `units`: unit name (requests/points/tokens/etc.)
- `remaining`: optional (if provider supplies)
- `used`: optional (if provider supplies)
- `delta`: optional consumption delta attributable to a known action
- `attribution`: optional enriched details (endpoint class, request kind) without raw request bodies

### `forecast_computed`

A forecast was computed from observations (time-to-exhaustion quantiles, risk).

Payload (typical):

- `provider_id`
- `pool_id`
- `constraint_id`
- `as_of_ts`: the observation time the forecast is anchored to
- `model`:
  - `model_id`: forecast model identifier
  - `model_version`: version of the model logic
  - `inputs_summary`: pointers/high-level stats (burn rate window, sample count, staleness)
- `tte`: P50/P90/P99 time-to-exhaustion (durations)
- `risk`: probability of exhaustion before next reset (and the reset horizon used)
- `confidence`: optional qualitative/quantitative indicator

### `intent_submitted`

An actor requested permission to perform a constrained action.

Payload (typical):

- `intent_id`: stable identifier for this intent
- `requested`:
  - `identity_id` (required concrete identity_id; daemon may still modify identity selection via `approve_with_modifications`)
  - `workload_id`
  - `scope_targets`: one or more scope references (still requires a single `scope_id` dimension; targets may be additional)
  - `expected_consumption`: optional estimate per pool (may be unknown)
  - `urgency`: interactive/batch/urgent (conceptual)
  - `duration_hint`: optional
- `client_context`: optional UI/tool metadata (non-sensitive)

### `intent_decided`

The daemon issued an authoritative decision for an intent.

Payload (typical):

- `intent_id`
- `decision`: `approve` | `approve_with_modifications` | `deny_with_reason`
- `modifications`: present only for `approve_with_modifications` (throttle, defer, narrow scope, switch identity, etc.)
- `reason`: present only for `deny_with_reason` (actionable, forecast/policy grounded)
- `evaluation`:
  - `as_of_ts`
  - `policy_version`
  - `forecast_refs`: which forecasts were considered (by ids or by `(provider_id,pool_id,constraint_id,as_of_ts)`)
  - `risk_summary`: concise explanation of the “tightest” constraint and why it gates

### `policy_triggered`

A policy condition activated (for auditing “why governance fired”).

Payload (typical):

- `policy_id`
- `policy_version`
- `trigger_kind`: hard_rule | soft_rule | reserve | degraded_mode | other
- `triggered_by`: references to relevant events (often a forecast or provider_error)
- `effect`: what it constrained (e.g., tightened gating threshold, reserved budget)

### `throttle_advised`

A non-binding advisory emitted to shape behavior (even absent a specific intent decision).

Payload (typical):

- `advice_id`
- `advice_kind`: reduce_rate | backoff | defer | switch_identity | reduce_scope | other
- `target`: which dimensions/pools/scopes it applies to
- `recommended_until`: optional timestamp or condition (e.g., “until reset”, “until P90 TTE > 30m”)
- `rationale`: forecast/policy grounded explanation

---

## Correlation and Causation Semantics

### `correlation_id` (grouping)

Use `correlation_id` to group events that belong to the same logical activity:

- a provider poll cycle (one poll yields many observations)
- one intent evaluation (submission, forecasts considered, decision, policy triggers)
- a degraded-mode incident window (errors, tightened policy, advisories)

Guideline: correlation should be stable across the whole flow, even if multiple event types are emitted.

### `causation_id` (lineage)

Use `causation_id` to identify the immediate parent event that caused this event to be emitted:

- `provider_poll_observed` causes `usage_observed`, `constraint_observed`, `reset_observed`
- `forecast_computed` is caused by the latest relevant observation(s) (often the poll)
- `intent_decided` is caused by `intent_submitted` (and may reference forecasts in payload)
- `policy_triggered` is caused by the forecast/error/decision that activated it

Rule: every event has a causation pointer; use a sentinel causation id when there is no meaningful parent (e.g., daemon startup).

### Multiple causes

If an event conceptually depends on multiple prior events (common for forecasts and policy), use:

- `causation_id` for the “primary” trigger
- additional references inside payload (e.g., `triggered_by`, `forecast_refs`, `input_event_refs`)

---

## Projections / Read Models (Derived Views)

Projections are materialized views built from the event log. They are rebuildable, can be dropped, and must track freshness relative to the log.

### Core projections (conceptual)

- Current posture view:
  - latest known remaining/used/reset per `(provider_id, pool_id, constraint_id, scope_id, identity_id)` as applicable
  - staleness indicators (time since last observation; last `event_id` applied)
- Forecast view:
  - latest `forecast_computed` per relevant pool/scope node
  - includes `as_of_ts`, model version, and confidence/staleness
- Intent history view:
  - all `intent_submitted` and `intent_decided` by `intent_id`
  - supports audit: “who asked, what was decided, why, under what forecasts/policies”
- Provider status view:
  - last success/error per provider integration, backoff state, degraded mode markers (derived from error events)
- Constraint graph mapping views (derived):
  - identity/scope/workload registries and relationships (as expressed through events in future docs), with sentinel-aware resolution

### Projection invariants

- Projections never “correct” history; they only reflect the result of replaying events.
- Every projection should be able to explain its provenance:
  - which `event_id` high-water mark it has processed
  - which model/policy versions it assumes for derived values
- Projections are allowed to change when rebuilt under new projection logic, but the underlying event history does not.

---

## Versioning and Migrations

### Event schema versioning

- Each event has `schema_version` scoped to its `event_type`.
- Evolution rules:
  - additive fields are preferred (backward compatible)
  - breaking changes require a `schema_version` bump and an explicit reader/upcaster strategy
- Events are immutable: migrations do not rewrite historical rows; they update readers and projections.

### Projection versioning and rebuild

- Projections have their own `projection_version` (conceptual) independent of event schema.
- Any projection schema change is handled by:
  1) dropping/recreating the projection
  2) replaying the event log (optionally starting from a checkpoint/snapshot)
- Forecast/policy determinism under replay depends on explicit versioning:
  - forecast events record `model_id` and `model_version`
  - policy-trigger/decision events record `policy_version`
  - any “now” semantics must be represented via timestamps in the events, not implicit wall-clock time

---

## Retention and Compaction (Local-First, Auditable)

The event log is append-only, but local disk is finite. Compaction must preserve auditability.

### Retention principles

- Never discard the ability to answer:
  - “What did we observe?”
  - “What did we decide?”
  - “Why did we decide it (which forecasts/policies)?”
- Decision lineage (`intent_*`, `policy_triggered`) is the highest-value audit trail and should have the longest retention.
- High-volume raw observation events may be eligible for compaction if replaced by faithful rollups.

### Compaction strategy (conceptual)

- Prefer checkpoint/snapshot acceleration over deletion:
  - periodic projection checkpoints to speed rebuilds
  - keep events; use checkpoints to avoid full replay cost
- If pruning is necessary, prune only with explicit, auditable substitution:
  - replace dense low-level observations with rollup summaries (time-bucketed usage/reset summaries)
  - retain enough granularity to reproduce posture and explain forecast/decision inputs at the time
- Compaction must be policy-driven and transparent:
  - compaction configuration should be visible in the system and treated as governance, not a hidden maintenance task
  - compaction should never remove security-relevant evidence (auth failures, persistent 429s, repeated denials)

Suggested retention tiers (non-normative):

- Keep indefinitely (or longest available): `intent_submitted`, `intent_decided`, `policy_triggered`, `provider_error` (at least summaries)
- Keep long horizon: `forecast_computed` (enables postmortems on model behavior)
- Keep medium horizon or roll up: `usage_observed`, `reset_observed`, `constraint_observed`, `provider_poll_observed`

---

## Open Questions / TODO

- Define canonical ID formats and lifecycles for: `agent_id`, `identity_id`, `workload_id`, `scope_id`, `pool_id`, `constraint_id` (and where these are registered/emitted as events).
- Decide the exact sentinel set (and whether a distinct “not applicable” sentinel is needed beyond `sentinel:unknown`).
- Specify `event_id` generation requirements (monotonic ordering vs uniqueness only) and ordering guarantees under concurrency.
- Formalize deduplication/idempotency keys for provider observations (how to avoid double-counting usage signals).
- Define the minimal payload fields required per event type (what is mandatory vs optional) without leaking provider secrets.
- Clarify multi-scope intents: how to represent multiple target scopes while preserving the single mandatory `scope_id` dimension semantics.

Terminology note:

- `workload_id` here corresponds to the seed’s `action_id` (same concept: the classified unit of work being attributed and governed).
- Establish a strict redaction policy: which provider response fields are allowed in events (and how to safely store optional raw references).
- Determine compaction/rollup invariants: what must remain to preserve “explainability” of forecasts and intent decisions across long histories.
