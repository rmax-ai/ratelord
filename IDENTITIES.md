# IDENTITIES: ratelord

This document defines how `ratelord` models **identities** as first-class entities in the constraint graph, with explicit sharing semantics, attribution, and privacy-preserving lifecycle management. It is docs-first: conceptual only; no implementation details.

---

## Invariants (From VISION + CONSTITUTION + ARCHITECTURE)

- `ratelord-d` is the sole authority for identity registration state and intent arbitration; clients are read-only.
- All identity lifecycle changes are recorded as immutable events; projections are derived and rebuildable.
- Every event is scoped to `agent_id`, `identity_id`, `workload_id`, `scope_id` (sentinels allowed; nulls forbidden).
- Shared vs isolated behavior is never inferred; it is made explicit via **constraint pools** and membership.
- Local-first privacy: secrets remain local; raw tokens are never written to the event log.

---

## Definitions: Identity vs Agent vs Workload vs Scope

### Identity

An **Identity** is “who the provider charges” when an action is performed. It is the bridge between:

- **credential material** (a secret used to authenticate), and
- a **provider principal** (user/app/installation/org/account context that receives rate-limit chargeback).

In `ratelord`, an identity is represented by a stable local `identity_id` plus provider-specific references (`provider_refs`). The identity record is non-secret; it must be safe to persist in events and projections.

### Agent

An **Agent** is an actor that submits intents (autonomous agent or human-driven process). Agents do not *own* limits directly; they consume constraints indirectly by selecting (or being assigned) an identity and executing a workload in a scope.

### Workload

A **Workload** is the logical action/flow being performed (e.g., “issue triage”, “repo inventory”, “search + hydrate”). Workloads determine which pools may be consumed and are used for attribution and policy shaping.

### Scope

A **Scope** is “where the action applies” (e.g., repo/org/account/global). Scopes are explicit and hierarchical, and are mandatory dimensions for events and decisions.

---

## Identity Types, Ownership, and Provider Principal Mapping

### Credential kinds (authentication material)

Credential kind describes how authentication is performed and how provider chargeback is determined:

- **PAT (Personal Access Token)**: authenticates as a provider user; charged to that user/account context.
- **OAuth token**: authenticates as a provider user via an OAuth app; charged to that user/account context (with app context for audit).
- **GitHub App**: authentication is as an app; actions are typically charged in the context of an **installation**.
- **Installation token**: short-lived credential material derived from a GitHub App installation; charged to the installation context.

Local operator/machine accountability is captured as metadata on identity registration events; it is not a provider-chargeback identity kind.

### Ownership (who controls/authorizes use)

Ownership captures who is accountable for lifecycle and policy constraints:

- **agent-owned**: registered for and primarily used by a specific agent (even if provider chargeback is shared).
- **system-owned**: managed by `ratelord-d` as an operator/system asset (e.g., “default org token”, “CI app”).
- **org-owned**: controlled by an organization/team (governed by organizational policy).

Ownership is about governance and allowed usage, not necessarily provider chargeback.

### Mapping to provider principals (who is actually charged)

`provider_refs` map a local `identity_id` to provider principals without storing raw secrets. Examples for GitHub include (opaque identifiers; names illustrative):

- `provider = github`
- `principal_kind = user | app | installation | org | unknown`
- `user_id`, `login` (optional, non-secret)
- `app_id`
- `installation_id`
- `org_id` / `org_login` (optional, non-secret)
- `token_kind = pat | oauth | installation_token`
- `token_fingerprint` (non-reversible, for rotation/revocation correlation)

Important: one local `identity_id` may span multiple credential rotations over time while retaining the same provider principal mapping (see lifecycle).

---

## Sharing Semantics: Pools, Shared vs Isolated, Virtual Isolation

### Constraint pools are the sharing primitive

A **pool** is the explicit consumption bucket (e.g., GitHub REST core / Search / GraphQL). If two identities drain the same real-world budget, they must be modeled as consuming from the same pool-at-scope node(s). If not, they must not.

### Shared vs isolated

- **Shared**: multiple identities (and therefore agents) are members of the same pool, so their workloads contend for the same remaining budget and reset window.
- **Isolated**: an identity consumes from a singleton pool (sharing = isolated) such that no other identity’s usage affects its posture.

### Virtual isolation via policy overlays

Providers often expose shared budgets that cannot be physically partitioned. `ratelord` supports “virtual isolation” as a governance layer:

- Provider reality: identities consume from shared pools.
- Local policy overlay: reserves/allocations/caps per agent/workload/identity create **virtual partitions** of shared capacity.
- The overlay affects intent arbitration (`approve` / `approve_with_modifications` / `deny_with_reason`), not provider behavior.

Virtual isolation must be explainable in time-domain terms (e.g., preserving P99 margin for a protected workload until reset).

---

## Registration Lifecycle (Daemon-Authority, Event-Sourced)

Identity lifecycle is expressed as events appended by `ratelord-d`. These are conceptual event types; the exact schema is defined elsewhere.

### Register

Records that an identity exists and is eligible for use under policy.

- Includes `identity_id`, `identity_type`, `owner_kind`, initial `provider_refs`, and non-secret metadata/labels.
- May include a `secret_ref` (an opaque pointer to local secret material) but never the secret itself.

### Rotate

Records that the credential material changed while the identity remains the “same” for attribution.

- Updates `secret_ref` and `token_fingerprint` (if used).
- Must preserve stable `identity_id` so historical attribution remains coherent across rotations.

### Revoke

Records that an identity is no longer authorized for use.

- Can be operator-initiated, policy-initiated, or triggered by provider signals (e.g., persistent auth failures).
- Revocation is effective immediately for intent arbitration.

### Quarantine

A safety state for suspected compromise or anomalous behavior.

- Quarantine is stricter than revoke in intent logic: it may deny all intents or allow only minimal safe operations (policy-defined) while investigation occurs.
- Quarantine is auditable and reversible via subsequent events.

Lifecycle events must be attributable (who/what initiated) and must not leak secrets.

---

## Security and Privacy Rules (Local-First)

### Non-negotiables

- Secrets (PATs, OAuth tokens, app private keys, installation tokens) remain local to the machine by default.
- Raw secrets are never stored in the event log, projections, or client-visible payloads.
- Event payloads avoid storing full provider responses unless explicitly scrubbed and justified.

### Redaction rules (minimum)

When recording observations/decisions/errors, redact or omit:

- `Authorization` headers, bearer tokens, cookies, signatures.
- Query parameters or payload fields that contain tokens or one-time codes.
- App private keys, JWTs, installation tokens, refresh tokens.
- Any provider error blobs that echo credentials.

### Safe attribution fields

Allowed in events/projections:

- `identity_id` (stable local ID)
- `provider_refs` (opaque IDs; non-secret)
- `token_fingerprint` (non-reversible; used only for correlation)
- friendly labels that do not contain secrets (team, purpose, environment)

---

## Attribution Model: Stable IDs, Provider Refs, and Sentinels

### `identity_id` (stable local ID)

- Stable across time and credential rotations.
- Used as the primary join key for attribution (events, forecasts, intent decisions).
- Must be unique within the local daemon authority domain.

### `provider_refs` (principal mapping)

- Stores provider-specific identifiers needed to explain sharing and chargeback.
- Treated as opaque; no assumptions about format outside the provider integration.

### Sentinel identities

Sentinels are required when a real identity is unknown or not applicable (nulls forbidden). Examples:

- `sentinel:unknown` — identity cannot be determined from available information.
- `sentinel:system` — daemon-internal work without a provider-charged identity (paired with workload sentinel usage).
- `sentinel:global` — used only when the identity dimension is conceptually global/not-applicable for a provider signal.

Sentinels must be explicit and consistently used so replay and audits remain deterministic.

---

## Examples

### Example 1: Shared PAT across multiple agents (explicitly shared pool)

Setup:

- `identity_id = ident:github:pat:shared-dev`
- `identity_type = pat`
- `owner_kind = system` (operator-managed)
- `provider_refs` map to GitHub user principal (account-scoped chargeback)

Semantics:

- Agents A and B both submit intents using `ident:github:pat:shared-dev`.
- Both consume from the same GitHub REST core/Search/GraphQL pools at the relevant scope(s).
- `ratelord-d` arbitrates intents based on shared pool forecasts (P50/P90/P99 TTE), issuing `approve_with_modifications` to stagger/throttle when tail risk rises.

### Example 2: Dedicated GitHub App for CI (virtual isolation + clearer chargeback)

Setup:

- `identity_id = ident:github:app:ci`
- `identity_type = github_app` (with installation token material rotating underneath)
- `owner_kind = org`
- `provider_refs` include `app_id` and `installation_id` for the CI installation

Semantics:

- CI workload uses only `ident:github:app:ci`; interactive agents are denied from using it by policy.
- Provider chargeback is scoped to the installation principal; sharing is explicit (only CI is a member).
- If the provider budget is still shared internally, `ratelord` can apply a policy overlay reserving P99 margin for the CI workload until reset.

---

## Open Questions / TODO

- Define the canonical `provider_refs` set for GitHub (user/app/installation/org) and the minimum required for explainability.
- Decide when two identities should be considered the “same principal” for pooling (e.g., OAuth vs PAT for the same user).
- Formalize “virtual isolation” representation: overlay-only vs explicit sub-allocation nodes in the constraint graph.
- Specify quarantine policy defaults (deny-all vs allow-minimal) and what provider signals trigger quarantine automatically.
- Decide how much identity metadata is safe to persist (e.g., logins) vs requiring redacted/hashed forms.
- Define the deterministic rules for mapping provider observations to `identity_id` under partial/ambiguous data (including sentinel usage).
