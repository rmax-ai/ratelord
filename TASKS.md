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
- [ ] **M2.2: Event Writer (Append-Only)**
    - Implement `AppendEvent` function.
    - Ensure atomic writes.
    - *Acceptance*: `D-04` (Event Immutability).
- [ ] **M2.3: Event Reader & Replay**
    - Implement `ReadEvents` iterator (from offset).
    - Implement basic replay loop to restore in-memory state on startup.
    - *Acceptance*: `D-02` (Crash Recovery), `D-05` (State Derivation).

## Epic 3: API Layer & Agent Contract
Focus: The HTTP/Socket interface for agents to negotiate intents.
- [ ] **M3.1: HTTP Server Shell**
    - Bind listener to `127.0.0.1:8090`.
    - Setup router and middleware (logging, panic recovery).
- [ ] **M3.2: Intent Endpoint (Stub)**
    - Implement `POST /v1/intent` handler.
    - Validate JSON schema.
    - Return mock "Approved" decision to verify connectivity.
    - *Acceptance*: `A-01` (Approve Intent), `A-02` (Latency).
- [ ] **M3.3: Health & Diagnostics**
    - Implement `GET /v1/health`.
    - Implement `GET /v1/events` (basic list).

## Epic 4: Identity & CLI Basics
Focus: Allow registration of the first identity to prove the write-path works.
- [ ] **M4.1: Identity Registration Command**
    - Implement CLI `ratelord identity add`.
    - Emit `identity_registered` event to storage.
    - *Acceptance*: `D-06` (Identity Registration).
- [ ] **M4.2: Basic State Projection**
    - Implement in-memory `IdentityProjection` built during replay.
    - Serve `GET /v1/identities` to list registered actors.
    - *Acceptance*: `T-03` (Identity List).
