# TASKS: ratelord Documentation & Bootstrapping

- [x] Phase 1: Foundational Documents
    - [x] 1. VISION.md (Orchestrator Initial Draft)
    - [x] 2. CONSTITUTION.md (Orchestrator Initial Draft)
    - [x] 3. ARCHITECTURE.md
    - [x] 4. CONSTRAINTS.md
    - [x] 5. IDENTITIES.md
    - [x] 6. DATA_MODEL.md
- [x] Phase 2: Behavioral & Engine Specs
    - [x] 7. PREDICTION.md
    - [x] 8. POLICY_ENGINE.md
    - [x] 9. AGENT_CONTRACT.md
    - [x] 10. API_SPEC.md
- [x] Phase 3: Interface & Workflow Specs
    - [x] 11. TUI_SPEC.md
    - [x] 12. WEB_UI_SPEC.md
    - [x] 13. WORKFLOWS.md
    - [x] 14. ACCEPTANCE.md
- [x] Phase 4: Project Management
    - [x] Initialize PROGRESS.md
    - [x] Initialize PHASE_LEDGER.md
    - [x] Harden orchestration loop (loop.sh & LOOP_PROMPT.md)
    - [x] Define TEST_STRATEGY.md

# Phase 4: Implementation - Core Infrastructure

## Epic 1: Foundation & Daemon Lifecycle
Focus: Getting the process to run, manage its lifecycle, and handle signals correctly.
- [x] **M1.1: Project Skeleton**
    - Create directory structure (`cmd/ratelord-d`, `pkg/engine`, `pkg/store`, `pkg/api`).
    - *Dependency*: None
- [x] **M1.2: Daemon Entrypoint & Signal Handling**
    - Implement main process loop.
    - Handle `SIGINT`/`SIGTERM` for graceful shutdown.
    - *Acceptance*: `D-03` (Graceful Shutdown).
- [x] **M1.3: Logging & Observability**
    - Setup structured logging (stdout/stderr).
    - Emit `system_started` log on boot.
- [ ] **M1.4: Configuration**
    - Implement configuration loader (env vars, defaults).
    - *Note*: Split from M1.1 to ensure atomic commits.

## Epic 2: Storage Layer (Event Sourcing)
Focus: The immutable SQLite ledger that serves as the source of truth.
- [x] **M2.1: SQLite Initialization**
    - Implement DB connection with WAL mode enabled.
    - Create `events` table schema (id, type, payload, dimensions, ts).
    - *Acceptance*: `D-01` (Clean Start).
- [x] **M2.2: Event Writer (Append-Only)**
    - Implement `AppendEvent` function.
    - Ensure atomic writes.
    - *Acceptance*: `D-04` (Event Immutability).
- [x] **M2.3: Event Reader & Replay**
    - [x] Implement `ReadEvents` iterator (from offset).
    - [x] Implement basic replay loop to restore in-memory state on startup.
    - *Acceptance*: `D-02` (Crash Recovery), `D-05` (State Derivation).

## Epic 3: API Layer & Agent Contract
Focus: The HTTP/Socket interface for agents to negotiate intents.
- [x] **M3.1: HTTP Server Shell**
    - Bind listener to `127.0.0.1:8090`.
    - Setup router and middleware (logging, panic recovery).
- [x] **M3.2: Intent Endpoint (Stub)**
    - Implement `POST /v1/intent` handler.
    - Validate JSON schema.
    - Return mock "Approved" decision to verify connectivity.
    - *Acceptance*: `A-01` (Approve Intent), `A-02` (Latency).
- [x] **M3.3: Health & Diagnostics**
    - Implement `GET /v1/health`.
    - Implement `GET /v1/events` (basic list).

## Epic 4: Identity & CLI Basics
Focus: Allow registration of the first identity to prove the write-path works.
- [x] **M4.1: Identity Registration Command**
    - Implement CLI `ratelord identity add`.
    - Emit `identity_registered` event to storage.
    - *Acceptance*: `D-06` (Identity Registration).
- [x] **M4.2: Basic State Projection**
    - Implement in-memory `IdentityProjection` built during replay.
    - Serve `GET /v1/identities` to list registered actors.
    - *Acceptance*: `T-03` (Identity List).

## Epic 5: Usage Tracking & Policy Engine
Focus: Implement the core logic for tracking usage against limits and making policy decisions.
- [x] **M5.1: Usage Tracking**
    - Create `pkg/engine/usage.go`.
    - Implement `UsageProjection` to track usage by identity/scope/window.
    - Hook it into the `Replay` loop.
    - *Acceptance*: `D-07` (Usage Tracking).
- [x] **M5.2: Policy Enforcement**
    - Create `pkg/engine/policy.go`.
    - Implement `Evaluate(intent)` which checks usage against limits.
    - Update `POST /v1/intent` to use the real policy engine.
    - *Acceptance*: `A-03` (Policy Enforcement).

## Epic 6: Provider Integration (Ingestion)
Focus: Connect to external sources (or mocks) to ingest real usage/limit data.
- [x] **M6.1: Provider Interface & Registry**
    - Create `pkg/provider/types.go` (Provider interface).
    - Implement a `ProviderRegistry` in `pkg/engine`.
    - *Dependency*: None.
- [x] **M6.2: Mock Provider**
    - Create `pkg/provider/mock/mock.go`.
    - Implement a provider that emits synthetic usage/limit events.
    - *Acceptance*: `T-02` (Mock Data Ingestion).
- [x] **M6.3: Polling Orchestrator**
    - Create `pkg/engine/poller.go`.
    - Implement the loop that ticks and calls `Provider.Poll()`.
    - Ingest results into Event Log (`provider_poll_observed`).
    - *Acceptance*: `D-08` (Continuous Polling).

## Epic 7: Forecasting (Prediction Engine)
Focus: Translate raw usage history into time-to-exhaustion predictions.
- [x] **M7.1: Forecast Model Interface**
    - Create `pkg/engine/forecast/types.go`.
    - Define `Model` interface (Inputs -> Distribution).
- [x] **M7.2: Linear Burn Model**
    - Implement simple linear regression model.
    - Calculate P99 time-to-exhaustion based on recent history.
- [x] **M7.3: Forecast Loop**
    - Trigger forecasts after `usage_observed` events.
    - Emit `forecast_computed` events.
    - *Acceptance*: `D-09` (Forecast Emission).

## Epic 8: TUI & Visualization (Read-Only)
Focus: Visualize the state of the system for the operator.
- [x] **M8.1: TUI Foundation**
    - Initialize Bubbletea model.
    - Connect to `GET /v1/events` and `GET /v1/identities`.
- [x] **M8.2: Dashboard View**
    - Render Usage Bars per pool.
    - Render recent Event Log.
    - *Acceptance*: `T-04` (Dashboard).

## Epic 9: System Stabilization & TUI Refinement
Focus: Improving the robustness of the existing components and enhancing the TUI.
- [x] **M9.1: TUI Drill-Down Views**
    - View detailed Event payloads.
    - View active Policy rules and current Usage stats in detail.
    - *Acceptance*: `T-01` (Real-time Stream detailed view).
- [x] **M9.2: Error Handling & Reconnection**
    - Implement reconnection logic in TUI if Daemon restarts.
    - Handle missing configuration or DB errors gracefully.
    - *Acceptance*: Robustness during `D-02` (Crash Recovery).
- [x] **M9.3: Configurable Policy Loading**
    - Load `policy.yaml` from disk on startup.
    - Support `SIGHUP` to reload policy.
    - *Acceptance*: `Pol-03` (Policy Hot Reload).

## Epic 10: Full System Verification
Focus: Proving the system works as a cohesive whole using the strategies in `TEST_STRATEGY.md`.
- [x] **M10.1: End-to-End Simulation Script**
    - Create a script/tool to generate realistic mock workloads.
    - Simulate multiple agents with different consumption patterns.
- [x] **M10.2: Verification of Drift Detection**
    - Manually inject usage into Mock Provider.
    - Verify Daemon detects drift and adjusts.
    - *Acceptance*: `P-03` (Drift Detection).
- [x] **M10.3: Verification of Policy Enforcement**
    - Drive usage to limit.
    - Verify Intents are denied.
    - *Acceptance*: `Pol-01` (Hard Limit), `Pol-02` (Load Shedding).
- [ ] **M10.4: Final Acceptance Run**
    - Execute full suite of Acceptance Tests.
