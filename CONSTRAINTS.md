# CONSTRAINTS: ratelord

This document defines how `ratelord` represents, observes, predicts, and governs constrained resources. It is provider-agnostic in structure, with GitHub as the first concrete provider.

`ratelord` is not a dashboard for “remaining quota.” It is a control plane that reasons in time: burn rate, reset windows, and probabilistic time-to-exhaustion (P50/P90/P99), then uses those forecasts to approve/modify/deny intents.

---

## Invariants (From VISION + CONSTITUTION)

- Constraints are first-class primitives, not incidental counters.
- `ratelord-d` is the sole authority for constraint state and intent arbitration; clients are read-only.
- All observations, decisions, and derived forecasts are captured as immutable events; snapshots are derived.
- Every datum is scoped (Agent, Identity, Workload, Scope); “global” is simply the root scope. Sentinel IDs are used for unknown/not-applicable dimensions.
- Defensive governance is triggered by forecasts (risk) rather than threshold breaches (reactive).
- Shared vs isolated semantics must be explicit; never assume “one agent = one limit.”
- The system prefers `approve_with_modifications` over `deny_with_reason` when safety can be preserved.
- Local-first, zero-ops, privacy-preserving by default.

---

## Definitions

### Constraint

A rule that bounds consumption of a finite resource over time. A constraint is evaluated against:

- a **pool** (what is being consumed),
- a **scope** (who/where it applies),
- a **capacity** (how much can be consumed),
- a **reset window** (when capacity replenishes),
- and a **consumption unit** (how consumption is measured).

Constraints may be **provider-enforced** (hard external reality) or **local policy caps** (self-imposed governance).

### Pool

A named consumption bucket with its own capacity and reset behavior (e.g., “GitHub REST core,” “GitHub Search,” “GitHub GraphQL”). A single action may consume from multiple pools.

### Capacity

The total available consumption units for a pool over its reset window. Capacity is time-bound; it is not meaningful without a reset window.

### Reset window

The time interval over which capacity replenishes (e.g., per hour, rolling minute, fixed epoch, or provider-specific schedule). Reset windows can have uncertainty (clock skew, provider jitter, delayed propagation), and `ratelord` must model that uncertainty.

### Consumption units

The unit by which a pool is depleted. Examples:

- “requests” (REST core, Search)
- “points” or “cost units” (GraphQL)
- “tokens” (LLM providers, future)
- “dollars” (spend budgets, future)

Units are pool-specific and not interchangeable.

### Observation vs prediction

- **Observation**: a measured fact at a time (e.g., remaining units, reset timestamp, observed cost of a request). Observations are recorded as events.
- **Prediction**: a derived estimate about the future (e.g., burn rate, P90 time-to-exhaustion, probability of exhaustion before reset). Predictions are also recorded as events, but are always derivable from prior observations + model assumptions.

---

## Constraint Graph Model

`ratelord` represents constraints as a directed graph to encode sharing, scoping, and multi-pool consumption.

### Core entities

- **Actor (Agent)**: an autonomous agent or human-driven process that submits intents.
- **Identity**: credentials used to perform actions (PAT, OAuth token, GitHub App installation token, etc.). Identities determine which pools/scopes are impacted and how sharing works.
- **Workload / Action**: a type of operation that consumes from one or more pools (e.g., “repo scan,” “issue triage,” “search + hydrate,” “graphql query”).
- **Scope**: where the constraint applies. Scopes are explicit and hierarchical:
  - repo
  - org
  - account (user or app owner)
  - global (provider root)
- **Pool**: the consumption bucket (REST/GraphQL/Search/etc.).

### Graph shape (conceptual)

- **Workloads** connect to **pools** they consume.
- **Agents** connect to **identities** they are allowed to use.
- **Identities** connect into one or more **pool-at-scope** nodes representing where that identity draws capacity from.
- **Scopes** form a hierarchy (repo → org → account → global). A pool node may exist at multiple scope levels depending on the provider’s semantics.

A useful mental model is that a single intent evaluation traverses:

1) the agent/identity relationship,
2) the workload’s pool consumption edges,
3) the scope hierarchy for each impacted pool,

and gates approval on the “tightest” forecast among all relevant pool/scope nodes.

### Shared vs isolated semantics (must be explicit)

`ratelord` never assumes isolation; it encodes it.

- **Shared pool semantics**: multiple agents and/or identities drain the same underlying pool node.
  - Example: two agents using the same PAT share the same REST core pool at the account scope.
- **Isolated pool semantics**: consumption is isolated into separate pool nodes.
  - Example: a dedicated identity reserved for one agent’s workload, enforced by local policy (even if the provider would allow sharing).

Both are represented by whether identities point to the same pool node (shared) or distinct pool nodes (isolated). Isolation can be “real” (provider-enforced) or “virtual” (local cap partitioning). Virtual isolation is typically a policy overlay or reservation over a shared provider pool rather than a distinct physical pool node.

---

## Time-Domain Reasoning (Not Counters)

Raw “remaining” is a snapshot; governance requires time dynamics.

### Burn rate

Burn rate is consumption per unit time for a pool-at-scope node (e.g., units/sec), derived from observations. It is modeled with uncertainty (variance), not as a single number.

### Time-to-exhaustion (TTE)

TTE is a random variable: how long until the pool reaches exhaustion given current remaining units and uncertain burn rate. `ratelord` computes quantiles:

- **P50 TTE**: median expected time to exhaustion
- **P90 TTE**: conservative time to exhaustion
- **P99 TTE**: worst-case planning bound

### Reset-aware risk

Because capacity replenishes on reset, the key question is not “Will we hit zero?” but:

- “What is the probability we exhaust before reset?”
- “How much margin do we retain at P90/P99?”
- “Does this intent meaningfully increase the tail risk of exhaustion?”

All governance logic is expressed relative to the reset window and the time remaining until reset.

---

## Governance Logic (Forecast-Gated Intent Arbitration)

When an agent submits an intent, `ratelord-d` evaluates it against all relevant constraints implied by the constraint graph.

### Inputs to evaluation (conceptual)

- Intent declaration: agent, identity, workload, scope targets, urgency/duration, expected consumption (units per pool) if known.
- Observations: recent remaining units, reset timestamps, observed costs.
- Model outputs: burn rate distribution per pool-at-scope node, TTE quantiles, probability of exhaustion before reset.

### Risk metrics used for gating

For each impacted pool-at-scope node:

- **P50/P90/P99 TTE**
- **Probability(exhaustion before reset)**

These are computed under “status quo” (baseline) and “with intent” (incremental load) to quantify marginal risk.

### Decision outcomes (per Constitution)

- **approve**: forecasts remain safe across all relevant pools/scopes.
- **approve_with_modifications**: forecasts are unsafe as requested, but can be made safe by shaping behavior, e.g.:
  - throttle / rate-shape (reduce burn rate)
  - defer start time (wait for reset)
  - switch identity (route to a less-loaded shared pool)
  - switch protocol (REST ↔ GraphQL) to change pool mix
  - batch or reduce scope (smaller query set)
- **deny_with_reason**: a hard constraint would be violated or risk remains unacceptable even with modifications.

### Hard vs soft constraints

- **Hard constraints**: must never be violated.
  - Provider-enforced caps (true exhaustion leads to errors/blocks).
  - Safety reserves that the operator marks as inviolable (local “red line”).
- **Soft constraints**: optimization preferences that can bend if necessary.
  - Prefer keeping 20% reserve, but allow dipping lower for urgent workloads.
  - Prefer spreading load across identities to reduce attribution conflicts.

### Provider-enforced vs local policy caps

- **Provider-enforced**: the external reality `ratelord` must respect (e.g., GitHub rate limits).
- **Local policy caps**: stricter limits imposed for governance (e.g., “cap this agent to 30% of shared pool,” “never use search pool below P90 10 minutes-to-reset margin”).

Local caps can create virtual partitions of shared provider pools; they do not change provider behavior, but they change what `ratelord` will approve.

---

## GitHub as First Provider (Conceptual Pools + Mapping)

This section describes the initial provider’s known constraint pools at a conceptual level, without implementation details.

### Known GitHub pools (conceptual)

- **REST core**: request-based rate limit bucket for standard REST API endpoints.
- **Search**: request-based bucket for search endpoints; typically tighter and more sensitive.
- **GraphQL**: cost-based pool where queries consume “points/cost units” based on complexity.

GitHub also has additional specialized buckets in practice (and behaviors that differ by auth type), but `ratelord` starts with a minimal, explicit set and expands via observed evidence and documented provider semantics.

### Mapping into the constraint graph

- **Identity** selection determines which account/app context is charged.
- **Scope** is interpreted based on provider semantics:
  - Some limits are effectively **account-scoped** (shared across all repos accessible by that identity).
  - Some are effectively **global** within the provider’s model.
- **Workloads** declare which pools they may touch:
  - e.g., “search + hydrate” consumes from Search and REST core
  - e.g., “graphql repo inventory” consumes from GraphQL (and possibly REST core for follow-up calls)

`ratelord` treats “pool + scope + identity semantics” as the authoritative tuple for determining sharing.

---

## Scenarios

These scenarios demonstrate why explicit sharing and multi-pool modeling matter.

### Scenario 1: Two agents share one PAT (shared pool contention)

- Agent A (“triage”) and Agent B (“dependency-audit”) both use the same PAT.
- Both run against different repos, but the PAT charges the same **account-scoped REST core** pool.
- Observations show burn rate spikes when both run concurrently; P90 TTE collapses below time-to-reset.

Governance outcome:

- New intents from either agent are **approve_with_modifications**:
  - throttle one workload
  - stagger start times
  - or route one agent to a different identity (if available)
- `ratelord` explains the decision in shared-pool terms: different repos did not imply different capacity.

### Scenario 2: One intent consumes multiple pools (Search + REST hydration)

Workload: “Find all repos matching X and fetch details”

- Step 1: Search API queries (Search pool)
- Step 2: For each result, REST calls to fetch metadata (REST core pool)

Even if REST core has comfortable headroom, Search may be near exhaustion before reset. A counters-only view would approve; a graph + multi-pool view rejects or reshapes.

Governance outcome:

- If Search P90 risk is high: **approve_with_modifications**
  - reduce query frequency
  - narrow search scope
  - cache results / defer hydration
  - shift to GraphQL where appropriate (changing pool mix)
- Approval requires safety across *both* pools, not just the largest one.

### Scenario 3: Shared identity across “isolated” agents (virtual isolation via local caps)

- Three agents run on one machine; all can technically use a single GitHub App installation token.
- Operator wants “build agent” to be protected from “exploration agent” burstiness.

`ratelord` models:

- Provider reality: a shared pool-at-scope node (shared).
- Local policy: allocate virtual sub-budgets (soft or hard) per agent/workload.

Governance outcome:

- Exploration bursts are throttled earlier, preserving predicted P99 margin for the build workload, even though the provider would allow exploration to consume everything.

---

## Open Questions / TODOs

- Define the canonical vocabulary for GitHub “scope” (account vs org vs installation) and how `ratelord` records it without ambiguity.
- Specify the minimal set of pool nodes for GitHub v1 (REST core, Search, GraphQL) vs when to introduce additional pools (secondary limits, abuse detection, per-endpoint nuances).
- Decide how reset uncertainty is represented: fixed jitter bounds vs learned distributions from observations.
- Decide whether “local caps” are expressed primarily per-agent, per-workload, per-identity, or combinations; define precedence rules when they conflict.
- Define how intents must declare “expected consumption” when costs are unknown (exploration): default priors, conservative assumptions, or mandatory dry-run sampling.
- Clarify how to represent “virtual isolation” in the graph (policy overlays vs explicit sub-nodes) while keeping event sourcing and replay semantics clean.
