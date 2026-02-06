# ADVANCED_SIMULATION: Scenarios & Framework Specs

**Status**: DRAFT
**Owner**: Orchestrator
**Related**: `TEST_STRATEGY.md`, `ACCEPTANCE.md`

## 1. Overview

To validate `ratelord` as a resilient constraint control plane, simple load testing is insufficient. We must model complex, adversarial, and high-entropy scenarios that reflect real-world distributed systems.

This document defines the **Advanced Simulation Framework**, extending `ratelord-sim` to support deterministic, multi-agent scenarios with configurable topologies, policies, and failure modes.

## 2. Goals

1.  **Behavioral Validation**: Prove that `ratelord` correctly shapes traffic (e.g., smoothing bursts, enforcing priorities) under stress.
2.  **Risk Modeling**: Verify that "time-to-exhaustion" predictions hold true when usage patterns change abruptly.
3.  **Resilience**: Ensure the daemon recovers gracefully from state drift (e.g., missed webhooks, partition) and protects the provider.

## 3. Simulation Framework Requirements

The `ratelord-sim` tool must start supporting a **Scenario Manifest** (JSON/YAML) rather than just CLI flags.

### 3.1 Scenario Manifest Schema
A scenario defines:
- **Topology**: Hierarchy of identities, groups, and limits.
- **Agents**: Distinct actors with specific behaviors (greedy, periodic, bursty).
- **Phases**: Time-bounded stages (e.g., "Warmup", "Attack", "Cooldown").
- **Invariants**: Success criteria (e.g., "Provider never returns 429").

### 3.2 Determinism & Seeds
All scenarios must be replayable. Random number generators for jitter, request timing, and "sabotage" must accept a fixed seed.

## 4. Advance Scenarios

### S-01: The Thundering Herd
*   **Description**: 100+ agents wake up simultaneously (e.g., cron job alignment).
*   **Topology**: Flat, single shared pool.
*   **Behavior**: All agents request `POST /intent` within a 100ms window.
*   **Expectation**:
    *   Daemon queues/throttles requests (no 500s).
    *   Approvals are distributed over time (smoothing).
    *   Provider rate limit is **not** exceeded.

### S-02: Drift & Correction
*   **Description**: The daemon's local state diverges from the provider (simulated by "sabotage" or external usage).
*   **Topology**: Single identity.
*   **Behavior**:
    *   Agents consume quota normally.
    *   "Saboteur" makes hidden requests directly to Provider (bypassing daemon).
    *   Provider usage increases faster than Daemon tracks.
*   **Expectation**:
    *   Daemon detects drift on next poll/webhook.
    *   Forecast adapts (Time-to-Exhaustion drops).
    *   Daemon tightens policy (rejects/throttles) to prevent actual exhaustion.

### S-03: Priority Inversion Defense
*   **Description**: High-volume low-priority traffic threatens to block critical operations.
*   **Topology**:
    *   `Group A` (Low Priority, High Volume).
    *   `Group B` (High Priority, Low Volume).
*   **Behavior**: `Group A` floods the system. `Group B` attempts sporadic requests.
*   **Expectation**:
    *   `Group A` is throttled aggressively.
    *   `Group B` requests are approved immediately (reserved capacity or priority queue).

### S-04: The "Noisy Neighbor" (Shared vs. Isolated)
*   **Description**: One misbehaving agent in a shared pool affects others? Or is isolated?
*   **Topology**:
    *   `Agent Bad`: Greedy, infinite loop.
    *   `Agent Good`: Periodic, highly critical.
    *   **Variant A**: Both in Shared Pool.
    *   **Variant B**: Separate Isolated Pools.
*   **Expectation**:
    *   **Variant A**: `Agent Good` may see increased wait times, but fairness logic should prevent starvation.
    *   **Variant B**: `Agent Good` is completely unaffected by `Agent Bad`'s exhaustion.

### S-05: Cascading Failure Recovery
*   **Description**: Provider goes down (503s) or effectively blocks all requests (429s).
*   **Behavior**: Provider returns error for all Polls/Intents.
*   **Expectation**:
    *   Daemon enters "Circuit Breaker" mode.
    *   Auto-denies local intents without hitting provider (fail fast).
    *   Exponential backoff on polling.
    *   Self-heals when Provider recovers.

## 5. Implementation Roadmap

### Phase 1: Engine Upgrade
- [ ] Refactor `ratelord-sim` to load scenarios from file.
- [ ] Implement `AgentBehavior` interface (Greedy, Poisson, Periodic).
- [ ] Add structured result reporting (JSON output).

### Phase 2: Scenario Definition
- [ ] Define `S-01` and `S-02` in JSON/YAML.
- [ ] Implement support for modifying topology (creating identities/limits) as part of setup.

### Phase 3: CI Integration
- [ ] Run selected scenarios as part of the build pipeline.
- [ ] Assert pass/fail based on `Invariants`.

