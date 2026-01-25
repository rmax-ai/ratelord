# CONSTITUTION: ratelord

This document defines the immutable laws that govern the development and runtime behavior of `ratelord`. All sub-agents and implementations must adhere to these articles.

## Article I: The Authority of the Daemon
1. The `ratelord-d` daemon is the sole source of truth for constraint state and intent arbitration.
2. Clients (TUI, Web) are strictly read-only and must not bypass the daemon to update state.

## Article II: The Eventual Source of Truth
1. All state changes, polls, and decisions MUST be recorded as immutable events.
2. Snapshot tables are merely optimizations; the system must be capable of rebuilding its entire state from the event log.

## Article III: The Negotiation Mandate
1. No agent shall perform a constrained action without first submitting an `Intent`.
2. The system shall prefer `approve_with_modifications` (throttling/deferral) over `deny_with_reason`, provided safety is maintained.
3. Every `deny_with_reason` response MUST include a clear, actionable rationale based on forecasts or hard limits.

## Article IV: Scoping Rigor
1. No data point exists in a vacuum. Every event must be scoped to an Agent, an Identity, a Workload, and a Scope (e.g., repo/org).
2. Use sentinel identifiers (e.g., `sentinel:global`) when a specific scope dimension is unknown or not applicable; nulls are not permitted for core dimensions.
3. Global limits are simply the root node of the constraint graph.

## Article V: Predictive Priority
1. Defensive logic must be triggered by *forecasts* (e.g., P90 time-to-exhaustion) rather than raw threshold breaches.
2. The system must account for variance and uncertainty in provider reset windows.

## Article VI: Local-First Privacy
1. All telemetry, credentials, and logs remain local to the user's machine by default.
2. Zero-config SQLite (WAL mode) is the mandatory storage engine.

## Article VII: Graceful Degradation
1. When constraints tighten, the system must signal agents to degrade functionality (e.g., lower frequency, smaller payloads) rather than halting entirely, unless a Hard Rule is violated.
