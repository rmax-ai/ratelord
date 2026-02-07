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
- [x] **M1.4: Configuration**
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
- [x] **M10.4: Final Acceptance Run**
    - Execute full suite of Acceptance Tests.
    - *Result*: Partial Pass (See `ACCEPTANCE_REPORT.md`).

# Phase 5: Remediation & 1.0 Release

## Epic 11: Policy Engine Fixes
Focus: Ensure policies are loaded, hot-reloaded, and correctly evaluated to enable denial/throttling.
- [x] **M11.1: Debug Policy Loading**
    - Investigate why `policy.yaml` rules are not applying.
    - Fix `LoadPolicyConfig` and `Evaluate`.
- [x] **M11.2: Implement Wait/Modify Actions**
    - [x] Ensure `approve_with_modifications` works.
    - [x] Implement shape/defer logic.
- [x] **M11.3: Verify Hot Reload**
    - Ensure SIGHUP updates rules without restart.

## Epic 12: Persistence & Robustness
Focus: Ensure drift detection and provider state survive restarts.
- [x] **M12.1: Persist Provider State**
    - Ensure provider offsets/drift are saved to SQLite.
- [x] **M12.2: TUI Verification**
    - Manually verify TUI dashboard connects and displays data.

# Phase 6: Release Prep

## Epic 13: 1.0.0 Release
- [x] **M13.1: Release Tagging & Notes**
    - Tag v1.0.0.
    - Write Release Notes.
    - [x] Tag v1.0.0.
    - [x] Write Release Notes.
- [x] **M13.2: Deployment Guide**
    - Write DEPLOYMENT.md (Systemd, Docker, K8s).

# Phase 7: Day 2 Operations - Real Providers

## Epic 14: GitHub Provider
Focus: Implement the first real provider to track GitHub API rate limits (Core, Search, GraphQL, Integration Manifests).
- [x] **M14.1: GitHub Configuration**
    - Define config structure (PAT, Enterprise URL).
    - Update `pkg/engine/config.go`.
- [x] **M14.2: GitHub Poller**
    - Implement `pkg/provider/github/github.go`.
    - Fetch limits via `GET /rate_limit`.
    - Map `core`, `search`, `graphql` to pools.
- [x] **M14.3: GitHub Integration Test**
    - Verify against public GitHub API (using a safe/dummy token or recorded mock).

## Epic 15: OpenAI Provider
Focus: Track OpenAI usage limits (RPM, TPM) via header inspection or Tier API.
- [x] **M15.1: OpenAI Configuration**
    - Define config structure (API Key, Org ID).
- [x] **M15.2: OpenAI Poller**
    - Implement `pkg/provider/openai/openai.go`.
    - Note: OpenAI limits are often response-header based, necessitating a "Probe" or "Proxy" approach, or just polling the `dashboard/billing` hidden APIs if available (unlikely stable).
    - *Decision*: Start with a "Probe" mode or just manual quota setting + local counting if API is unavailable.
    - *Refinement*: OpenAI's headers `x-ratelimit-limit-requests` etc. are returned on requests. We might need a "Passive" provider that ingests data from a sidecar/proxy, or we proactively poll a lightweight endpoint to check headers.

## Epic 16: Dogfooding & Tuning
Focus: Internal usage to validate stability using real GitHub tokens.
- [x] **M16.1: Dogfood Environment Setup**
    - Create `deploy/dogfood` directory.
    - Create `deploy/dogfood/policy.json` (or `policy.yaml`) monitoring GitHub rate limits for the current user/token.
    - Create `deploy/dogfood/run.sh` to boot the daemon with this local configuration.
- [x] **M16.2: Operational Run**
    - Execute `run.sh` locally.
    - Generate usage (via `gh` CLI or `ratelord` identity) to populate the event log.
    - *Verify*: Ensure `provider_poll_observed` and `usage_observed` events are recorded.
- [x] **M16.3: Analysis & Tuning**
    - Analyze the resulting event log to compare `forecast_computed` vs actual usage trends.
    - Determine if the Linear Burn Model needs tuning for bursty traffic.

# Phase 8: Expansion

## Epic 17: Client SDKs
Focus: Language-specific bindings for the Agent Contract (Intent Negotiation).
- [x] **M17.1: SDK Specification**
    - Draft `CLIENT_SDK_SPEC.md`.
    - Define interfaces for `Ask`, `Propose`, `Feedback`.
- [x] **M17.2: Go SDK**
    - Implement `pkg/client`.
    - Provide `NewClient(httpEndpoint)`.
    - Implement `Ask(ctx, intent)`.
- [x] **M17.3: Python SDK**
    - Implement `ratelord` package (PyPI structure).
    - Implement `Client` class with `ask()` method.
    - Ensure fail-closed behavior and auto-wait.
    - *Dependency*: M17.1.

## Epic 18: Web UI Implementation
Focus: Modern, graphical interface for observing system state.
- [x] **M18.1: Spec Refinement**
    - Refine `WEB_UI_SPEC.md` with implementation details.
- [x] **M18.2: Project Scaffold**
    - Initialize `web/` with React + Vite + Tailwind.
    - Setup proxy to daemon API.
- [x] **M18.3: Dashboard Implementation**
    - Implement `AppShell` and `Dashboard` view.
    - Connect to `GET /v1/events` and `GET /v1/identities`.
- [x] **M18.4: Build Integration**
    - Create `Makefile` rules for web build.
    - Use `//go:embed` to serve UI from `ratelord-d`.
    - *Acceptance*: `ratelord-d --web` serves the UI.
- [x] **M18.5: History View**
    - Implement `/history` route with `TimeRangePicker`, `EventTimeline`, and `EventList`.
    - Support server-side filtering via URL params.
- [x] **M18.6: Identity Explorer**
    - Implement `/identities` route.
    - Visualize hierarchy of Agents, Scopes, and Pools.

# Phase 9: Ecosystem & Hardening

## Epic 19: Node.js / TypeScript SDK
Focus: Enable the largest ecosystem of agents (JS/TS) to use Ratelord.
- [x] **M19.1: SDK Specification**
    - Define TypeScript interfaces for Intent, Decision, and Client options.
    - Create `sdk/js/SPEC.md` or update `CLIENT_SDK_SPEC.md`.
- [x] **M19.2: Project Scaffold**
    - Initialize `sdk/js` with `package.json`, `tsconfig.json`.
    - Setup Jest/Vitest for testing.
- [x] **M19.3: Core Implementation**
    - Implement `RatelordClient` class.
    - Implement `ask(intent)` with retries and fail-closed logic.
    - *Acceptance*: Unit tests pass.
- [x] **M19.4: Integration Verification**
    - Create a sample script `sdk/js/examples/basic.ts`.
    - Verify against running `ratelord-d`.
- [x] **M19.5: Release Prep**
    - Configure `package.json` exports/files.
    - Create `sdk/js/npmignore`.
    - Document publish process.

## Epic 20: Operational Visibility
Focus: Export internal metrics to standard observability tools.
- [x] **M20.1: Prometheus Exporter**
    - Expose `/metrics` endpoint.
    - Export `ratelord_usage`, `ratelord_limit`, `ratelord_forecast_seconds` gauges.
    - Export `ratelord_intent_total` counters.
- [x] **M20.2: Logging Correlation**
    - [x] Ensure `trace_id` / `intent_id` is threaded through all logs for a request.
- [x] **M20.3: Grafana Dashboard**
    - [x] Create `deploy/grafana/dashboard.json`.
    - [x] Visualize `ratelord_usage` and `ratelord_limit` per pool.

## Epic 21: Configuration & CLI Polish
Focus: Production-grade configuration management.
- [x] **M21.1: Robust Config Loader**
    - Support `RATELORD_DB_PATH`, `RATELORD_POLICY_PATH`, `RATELORD_PORT`.
    - Support CLI flags to override env vars.
    - Resolve M1.4 debt.

## Epic 22: Advanced Policy Engine
Focus: More expressive governance rules.
- [x] **M22.1: Soft Limits & Shaping**
    - [x] **M22.1.1: Policy Action Types**: Add `warn` and `delay` to Policy Action definition.
    - [x] **M22.1.2: Evaluator Updates**: Update `Evaluate` to handle soft limits (return `Approve` with warning, or `ApproveWithModifications` with wait).
    - [x] **M22.1.3: API Response Update**: Ensure `v1/intent` response captures warnings and wait instructions.
     - [x] **M22.2: Temporal Rules**
         - [x] **M22.2.1: TimeWindow Matcher**: Add `time_window` (start_time, end_time, days_of_week) to Policy Rule.
         - [x] **M22.2.2: Evaluator Time Check**: Implement time checking in `Evaluate`.

## Epic 23: Security Hardening
Focus: Secure the daemon for production usage beyond localhost.
- [x] **M23.1: TLS Termination**
    - Support `RATELORD_TLS_CERT` and `RATELORD_TLS_KEY` env vars.
    - Serve HTTPS if configured.
- [x] **M23.2: API Authentication**
    - [x] **M23.2.1: Auth Token Management**: Extend `identity add` to generate/accept an API token (store hashed).
    - [x] **M23.2.2: Auth Middleware**: Validate `Authorization: Bearer <token>` against registered identities.
- [x] **M23.3: Secure Headers**
    - Add HSTS, CSP, and other security headers to HTTP responses.

# Phase 10: Advanced Intelligence & Integration

## Epic 24: Adaptive Throttling
Focus: Move beyond static limits to dynamic flow control.
- [x] **M24.1: Dynamic Delay Controller**
    - Implement a PID or AIMD controller to calculate wait times.
    - Inputs: Current burn rate, remaining budget, time to reset.
    - Outputs: Suggested wait time (duration).
- [x] **M24.2: Feedback Loop Integration**
    - Feed "actual consumption" back into the delay calculator.
    - Adjust aggression based on "drift" (forecast vs actual).
- [x] **M24.3: Configuration & Tuning**
    - Allow configuration of controller parameters (Kp, Ki, Kd) via policy.

## Epic 25: Long-term Trends & Aggregation
Focus: Efficient querying for historical data.
- [x] **M25.1: Aggregation Schema**
    - [x] Update `pkg/store/sqlite.go` `migrate()` to include `usage_hourly`, `usage_daily`, and `system_state`.
    - [x] Add `GetSystemState(key)` and `SetSystemState(key, val)` methods to `Store`.
    - [x] Add `UpsertUsageStats(batch)` method to `Store`.
- [x] **M25.2: Rollup Worker Core**
    - [x] Create `pkg/engine/rollup.go`.
    - [x] Implement `RollupWorker` struct with `Run(ctx)` loop.
    - [x] Implement aggregation logic (bucketing and delta calculation).
    - [x] Integrate worker into `cmd/ratelord-d/main.go` startup.
- [x] **M25.3: Trend API**
    - [x] Add `GetTrends` method to `Store` (query with filters).
    - [x] Implement `GET /v1/trends` handler in `pkg/api`.
    - [x] Add query param parsing and validation.
	- [x] **M25.4: Integration Test**
	    - [x] Generate synthetic usage events.
	    - [x] Force a rollup cycle.
	    - [x] Verify `GET /v1/trends` returns expected aggregates.

## Epic 26: Webhooks & Notifications
Focus: Push alerts to external systems.
- [x] **M26.1: Webhook Registry**
    - [x] Create `webhook_configs` table.
    - [x] Implement `POST /v1/webhooks` to register URLs.
- [x] **M26.2: Dispatcher**
    - [x] Async worker to send HTTP POST payloads to registered webhooks.
    - [x] Handle retries and backoff.
- [x] **M26.3: Security (HMAC)**
    - [x] Sign webhook payloads with a shared secret.
    - [x] Include `X-Ratelord-Signature` header.

# Phase 11: Advanced Capabilities

## Epic 27: State Snapshots & Compaction
Focus: Improve startup time and manage disk usage.
- [x] **M27.1: Snapshot Schema**
    - Create `snapshots` table (snapshot_id, timestamp, payload blob).
- [x] **M27.2: Snapshot Worker**
    - Implement a worker that periodically serializes the `Projection` state (Usage, Limits, etc.) to a snapshot.
	- [x] **M27.3: Startup Optimization**
	    - Update `Loader` to load the latest snapshot first.
	    - Replay events only *after* the snapshot timestamp.
	    - *Acceptance*: Startup time is O(1) + O(recent_events) instead of O(all_events).
	- [x] **M27.4: Event Pruning**
	    - Implement a command or worker to delete events older than retention policy (if they are snapshotted).


## Epic 28: Advanced Simulation Framework
Focus: Validate complex scenarios and stress test the system (as per `ADVANCED_SIMULATION.md`).
- [x] **M28.1: Simulation Engine Upgrade**
    - [x] Refactor `ratelord-sim` to support configurable scenarios (JSON/YAML).
    - [x] Implement deterministic seeding for RNG.
    - [x] Implement `AgentBehavior` interface (Greedy, Poisson, Periodic).
- [x] **M28.2: Scenario Definitions**
    - [x] **S-01**: Implement "Thundering Herd" scenario config.
    - [x] **S-02**: Implement "Drift & Correction" scenario (requires Drift Saboteur).
    - [x] **S-03**: Implement "Priority Inversion Defense" scenario.
    - [x] **S-04**: Implement "Noisy Neighbor" (Shared vs Isolated) scenario.
    - [x] **S-05**: Implement "Cascading Failure Recovery" scenario.
- [x] **M28.3: Reporting & Verification**
    - Output structured JSON results (latency histograms, approval rates).
    - Add assertions to verify scenario success criteria.

## Epic 29: Financial Governance
Focus: Elevating "Cost" to a first-class constraint alongside Rate Limits (`FINANCIAL_GOVERNANCE.md`).
- [x] **M29.1: Currency Types & Usage Extension**
    - [x] Create `pkg/engine/currency` package.
    - [x] Add `MicroUSD` type (int64).
    - [x] Update `Usage` struct to include `Cost MicroUSD`.
- [x] **M29.2: Pricing Registry**
    - [x] Update `pkg/engine/config.go` to include `Pricing` map.
    - [x] Implement lookup logic: `GetCost(provider, model, units)`.
    - [x] Update `UsageProjection` to calculate cost on ingestion.
- [x] **M29.3: Cost Policy**
    - [x] Add `budget_cap` rule type to `pkg/policy`.
    - [x] Implement `Evaluate` logic for cost-based rejections.
    - [x] Add `cost_efficiency` rule type for provider selection suggestions.
- [x] **M29.4: Forecast Cost**
    - [x] Update `ForecastModel` to predict `Cost` exhaustion.
    - [x] Emit `forecast_cost_computed` events.

## Epic 30: Cluster Federation
Focus: Expanding from single-node daemon to distributed fleet governance (`CLUSTER_FEDERATION.md`).
- [x] **M30.1: Grant Protocol Definition**
    - [x] Define protocol in `API_SPEC.md` and `CLUSTER_FEDERATION.md`.
    - [x] Define `GrantRequest` and `GrantResponse` structs (Implementation).
    - [x] Implement `POST /v1/federation/grant` endpoint on Leader.
- [x] **M30.2: Follower Mode**
    - [x] Add `--mode=follower` flag to `ratelord-d`.
    - [x] Implement `RemoteProvider` that requests grants from Leader instead of direct token bucket.
    - [x] Implement local cache for granted tokens.
- [x] **M30.3: Leader State Store**
    - [x] Abstract `TokenBucket` storage (See M32.1).
    - [x] Implement Leader Election (See M33.1).

# Phase 12: Release Engineering

## Epic 31: Automated Release Pipeline
Focus: Zero-touch versioning and artifact publication (`RELEASING.md`).
- [x] **M31.1: CI Workflows**
    - [x] Create `.github/workflows/test.yaml` (Go test, lint).
    - [x] Create `.github/workflows/release.yaml` (Trigger on tag).
- [x] **M31.2: Release Script / Goreleaser**
    - [x] Configure `.goreleaser.yaml`.
    - [x] Ensure cross-compilation (Darwin/Linux, AMD64/ARM64).
    - [x] Configure Docker build and push.
- [x] **M31.3: Documentation & Changelog**
    - [x] Configure changelog generation from Conventional Commits.
    - [x] Auto-update `RELEASE_NOTES.md` or GitHub Release body.

# Phase 13: Scale & Reliability

## Epic 32: External State Stores
Focus: Allow the Leader to persist state in shared storage (Redis/Etcd) for stateless deployments.
- [x] **M32.1: Usage Store Interface**
    - [x] Refactor `UsageProjection` to use `UsageStore` interface.
    - [x] Implement `MemoryUsageStore` (default).
- [x] **M32.2: Redis Implementation**
    - [x] Implement `RedisUsageStore` using `go-redis`.
    - [x] Add `RATELORD_REDIS_URL` config.
- [x] **M32.3: Atomic Operations**
    - [x] Refactor `PoolState` storage to Redis Hash (`HSET`) to support partial updates.
    - [x] Implement `IncrementUsed` with Lua scripts for atomicity.

## Epic 33: High Availability
Focus: Automatic Leader Election for failover.
- [x] **M33.1: Leader Election**
    - [x] Define `LeaseStore` interface.
    - [x] Define `Lease` struct (HolderID, Expiry).
    - [x] Implement `RedisLeaseStore`.
    - [x] Implement `SqliteLeaseStore` (as fallback).
    - [x] Implement `ElectionManager` with Acquire/Renew loop.
- [x] **M33.2: Standby Mode**
    - [x] Implement `ElectionManager` struct.
    - [x] Implement `StandbyLoop` (Polls lease, if free -> Acquire).
    - [x] Handle `OnPromote` (Load state, start Policy Engine).
    - [x] Handle `OnDemote` (Stop Policy Engine, flush state).
- [x] **M33.3: Client Routing**
    - [x] Implement `HTTP Middleware` to check Leader status.
    - [x] Proxy requests from Followers to Leader (or return 307 Redirect).
- [x] **M33.4: Split-Brain Protection**
    - [x] Add `Epoch` to `Lease` and `ElectionManager`.
    - [x] Include `Epoch` in `Event` metadata.
    - [x] Validate `Epoch` on critical state transitions.


## Epic 34: Federation UI
Focus: Visualize the entire cluster.
- [x] **M34.1: Cluster View**
    - [x] **M34.1.1: API**: Implement `GET /v1/cluster/nodes` and `ClusterTopology` projection.
    - [x] **M34.1.2: UI**: Add "Cluster" tab (Node Table) in Web UI.
- [x] **M34.2: Node Diagnostics**
    - [x] Visualize Replication Lag & Election Status (Implemented via Metadata & UI Update).

# Phase 14: Architecture Convergence

## Epic 35: Canonical Constraint Graph
Focus: Formalizing the constraint graph taxonomy as defined in ARCHITECTURE.md.
- [x] **M35.1: Graph Schema Definition**
    - [x] **M35.1.1: Node Types**: Define `Agent`, `Identity`, `Workload`, `Resource`, `Pool`, `Constraint` structs in `pkg/graph`.
    - [x] **M35.1.2: Edge Types**: Define `Owns`, `Triggers`, `Limits`, `Depletes`, `AppliesTo`, `Bounds` edge definitions.
    - [x] **M35.1.3: Graph Interface**: Define the `Graph` interface for adding nodes/edges and traversing.
- [x] **M35.2: In-Memory Graph Projection**
    - [x] **M35.2.1: Projection Struct**: Implement `GraphProjection` that holds the graph state.
    - [x] **M35.2.2: Event Handlers**: Implement handlers for `IdentityRegistered` (PolicyUpdated pending).
    - [x] **M35.2.3: Replay Integration**: Hook `GraphProjection` into the main `Loader` replay loop.
    - [x] **M35.2.4: Policy Graph Population**: Handle `PolicyUpdated` events (or load from config) to create `Constraint` and `Pool` nodes and `AppliesTo`/`Limits` edges.
- [x] **M35.3: Policy Matcher on Graph**
    - [x] **M35.3.1: Traversal Logic**: Implement `GetConstraintsForIdentity(id)` (Implemented `FindConstraintsForScope`).
    - [x] **M35.3.2: Engine Integration**: Wire `GraphProjection` into `PolicyEngine` to replace linear search with graph traversal.
- [x] **M35.4: Graph Visualization**
    - [x] **M35.4.1: API**: Add `GET /v1/graph` endpoint (JSON/Dot format).
    - [x] **M35.4.2: UI**: Visualize in Web UI (Force-directed graph).

## Epic 36: Advanced Retention & Compaction
Focus: Managing long-term storage and compliance.
- [x] **M36.1: Retention Policy Engine**
    - [x] Allow configuring TTL per Event Type.
    - [x] Implement `PruneWorker` (Refinement of M27.4).
- [x] **M36.2: Cold Storage Offload**
    - [x] Implement S3/GCS adapter for archiving old events/snapshots (Implemented `LocalBlobStore` as first adapter).
    - [x] Implement "Hydrate from Archive" for historical analysis (Partial - ArchiveWorker implemented).
- [x] **M36.3: Compliance & Deletion**
    - [x] Implement `DeleteIdentity` (GDPR "Right to be Forgotten").
    - [x] Prune all events associated with a specific Identity ID.

## Epic 37: Explainability & Audit
Focus: Answering "Why?" for every decision.
- [x] **M37.1: Decision Explainability**
    - [x] **M37.1.1: Trace Structs**: Define `RuleTrace` (RuleID, Input, Result) in `pkg/engine`.
    - [x] **M37.1.2: Evaluator Trace**: Update `Evaluate` to capture trace of all checked rules.
    - [x] **M37.1.3: Event Enrichment**: Add `Trace` to `Decision` event payload (Available in Result, Event pending if needed).
    - [x] **M37.1.4: API Exposure**: Return trace in `POST /v1/intent` response (debug mode).
- [x] **M37.2: Compliance Reports**
    - [x] **M37.2.1: Report Engine**: Create `pkg/reports` with interface `Generator`.
    - [x] **M37.2.2: CSV Generator**: Implement `CSVGenerator` for flat tabular data.
    - [x] **M37.2.3: Access Log Report**: Implement `AccessLogReport` (Date, Identity, Intent, Decision, RuleTrace).
    - [x] **M37.2.4: Usage Report**: Implement `UsageReport` (Date, Pool, Usage, Limit, Cost).
    - [x] **M37.2.5: API Endpoint**: Implement `GET /v1/reports` with `type` and `format` params.
- [x] **M37.3: Policy Debugging**
    - [x] Implement "Trace Mode" for Policy Engine (logs every rule result).
    - [x] Web UI: Visualize Policy Evaluation Tree.

## Epic 38: Architecture Convergence
Focus: Unifying subsystems and paying down technical debt.
- [x] **M38.1: Constraint Graph Integration**
    - [x] Refactor Policy Engine to use Graph (Done in M35.3).
    - [x] Ensure Federation Grant logic respects Graph constraints.
    - [x] Ensure Usage Projection handles Grant consumption.
- [x] **M38.2: Unified Store Audit**
    - [x] Verify Redis/SQLite parity for Usage Store.
    - [x] Consolidate "TokenBucket" vs "UsageStore" abstractions if any diverge.

# Phase 15: Ecosystem & Interoperability

## Epic 39: Model Context Protocol (MCP) Integration
Focus: Allow LLMs (Claude, Gemini, etc.) to natively discover and query Ratelord constraints.
- [x] **M39.1: MCP Server Core**
    - [x] **M39.1.1: Dependency**: Run `go get github.com/mark3labs/mcp-go`.
    - [x] **M39.1.2: Package Structure**: Create `pkg/mcp` and implementation stub.
    - [x] **M39.1.3: CLI Integration**: Update `cmd/ratelord/main.go` to add `mcp` subcommand (supports `--url` and `--token` flags).
    - [x] **M39.1.4: Client Wrapper**: Create a simple internal HTTP client helper in `pkg/mcp/client.go` to standardise API calls.
- [x] **M39.2: Resource Exporter**
    - [x] **M39.2.1: Events Resource**: Implement `ratelord://events` fetching from `GET /v1/events` (limit 50).
    - [x] **M39.2.2: Usage Resource**: Implement `ratelord://usage` fetching from `GET /v1/trends` (or `GET /v1/graph` for structure).
    - [x] **M39.2.3: Config Resource**: Implement `ratelord://config` to expose current policy rules (read-only).
- [x] **M39.3: Tool Exporter**
    - [x] **M39.3.1: Ask Intent Tool**: Implement `ask_intent` tool wrapping `POST /v1/intent`.
    - [x] **M39.3.2: Check Usage Tool**: Implement `check_usage` tool that allows querying specific pools/identities.
- [x] **M39.4: Prompts**
    - [x] **M39.4.1: System Prompt**: Implement `ratelord-aware` MCP prompt.

## Epic 40: Client Resilience Library
Focus: Standardize retry/backoff logic across SDKs to prevent thundering herds.
- [x] **M40.1: Go SDK Resilience**
    - [x] Add Backoff & Jitter.
- [x] **M40.2: JS SDK Resilience**
    - [x] Add `bottleneck` or custom backoff.
- [x] **M40.3: Python SDK Resilience**
    - [x] Add `tenacity` integration.

# Phase 16: Quality Assurance & Hardening

## Epic 41: Test Coverage Improvement
    - [x] **M41.1: API Package Coverage**
    - Target: `pkg/api` > 80% coverage.
    - Implement tests for handlers, middleware, and validation.
    - [x] **M41.2: Store Package Coverage**
    - [x] Target: `pkg/store` > 80% coverage.
    - [x] Implement tests for SQLite store, event reading/writing.
    - [x] **M41.3: Engine Package Coverage**
    - [x] Target: `pkg/engine` > 80% coverage.
    - [x] Improve tests for policy evaluation and state management.

## Epic 48: Deployment Verification
Focus: Ensure the system runs correctly in containerized environments.
- [x] **M48.1: Docker Composition**
    - Create `Dockerfile` for multi-stage build (Web + Go).
    - Create `docker-compose.yml` for local stack (Daemon + Redis).
- [x] **M48.2: End-to-End Testing**
    - Create `tests/e2e` suite.
    - Verify full flow: Identity -> Policy -> Intent -> Decision.
    - Verify Web UI availability.

## Epic 42: User Documentation
Focus: Create user-facing documentation to explain concepts and usage.
- [x] **M42.1: Concept Guides**
    - [x] Draft `docs/concepts/architecture.md` (Simplified "How it works").
    - [x] Draft `docs/concepts/core-model.md` (Identity, Scope, Pool, Constraint).
- [x] **M42.2: API Reference**
    - [x] Draft `docs/reference/api.md` (Endpoints and Client Behavior).
- [x] **M42.3: User Guides**
    - [x] Update `docs/guides/cli.md` (MCP, Identity, Mode).
    - [x] Update `docs/guides/web-ui.md` (Graph, Cluster, Reports).
    - [x] Create `docs/guides/mcp.md` (MCP Integration).

## Epic 43: Final Polish & Debt Paydown
Focus: Address technical debt, stubbed features, and missing tests identified during final assessment.
- [x] **M43.1: Complete Reporting Engine**
    - [x] Implement actual CSV generation logic in `pkg/reports/csv.go`.
    - [x] Add unit tests for `pkg/reports`.
- [x] **M43.2: Complete Graph Projection**
    - [x] Handle `PolicyUpdated` events in `pkg/graph/projection.go`.
    - [x] Optimize graph traversal (Index for O(1) lookup).
    - [x] **M43.3: Hardening & Configuration**
    - [x] Remove hardcoded `resetAt` in `pkg/engine/forecast/service.go`.
    - [x] Fix `pkg/api/server.go` correct pool ID usage.
    - [x] Add unit tests for `pkg/mcp` (Server and Handlers).
    - [x] Add unit tests for `pkg/blob` (Local Store).
    - [x] **M43.6: Inject Provider Version**
    - [x] Inject `ProviderVersion` at build time (ldflags) or derive from package.
    - [x] **M43.4: Codebase Cleanup (Assessment Findings)**
    - [x] `pkg/graph/projection.go`: Implement `ProviderObserved` handler.
    - [x] `pkg/provider/federated/provider.go`: Fix TODOs (error reporting, pre-seeding note).
    - [x] `pkg/engine/poller.go`: Verified configurable units implementation.
    - [x] `pkg/api/federation.go`: Verified `RemainingGlobal` lookup.

# Phase 17: Future & Roadmap (Post-v1.0)

## Epic 44: Advanced Adaptive Policy
Focus: Implement the sophisticated shaping behaviors from the original vision.
- [ ] **M44.1: Adaptive Actions**
    - Implement `route` action (load balancing across identities).
    - Implement `switch_protocol` action (REST <-> GraphQL).
- [ ] **M44.2: Client Negotiation**
    - Update `Intent` response to include detailed modification instructions beyond `wait_seconds`.
    - Update SDKs to handle complex negotiation.

## Epic 45: Enhanced Simulation & UI
Focus: Bring the simulation capabilities into the visual domain.
- [x] **M45.1: Web UI Simulation Lab**
    - Create "Simulation" tab in Web UI.
    - Implement frontend for configuring scenarios (Agents, Policies, Bursts).
    - Run `ratelord-sim` (wasm or server-side) and visualize results.
- [x] **M45.2: Simulation Integration**
    - Connect UI to `ratelord-sim` backend (or wasm).
    - Visualize real-time results.

## Epic 46: Distributed Consistency
    Focus: Hardening the distributed system guarantees.
    - [ ] **M46.1: Global State Aggregation**
        - Implement CRDT or gossip protocol for accurate global limit tracking in Federation.
        - Move beyond "Leader Local = Global" simplification.
    - [ ] **M46.2: Graph Concurrency**
        - Refactor `GraphProjection` for Copy-On-Write or safe concurrent access.

## Epic 47: Provider Intelligence
    Focus: Smarter provider integration.
    - [ ] **M47.1: OpenAI Smart Probing**
        - Implement "Probe" mode that hits a cheap endpoint (e.g. chat completion with max_tokens=1) if `/models` doesn't return relevant rate limit headers.
        - Handle model-specific rate limits (gpt-4 vs gpt-3.5).





