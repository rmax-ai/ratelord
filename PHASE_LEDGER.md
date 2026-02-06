# PHASE LEDGER: ratelord

## 2024-05-22
- **Action**: Initialized Project Seed and Execution Protocol.
- **Action**: Drafted VISION.md and CONSTITUTION.md.
- **Action**: Created TASKS.md, PROGRESS.md, and PHASE_LEDGER.md for tracking.

## 2026-01-25
- **Action**: Drafted ARCHITECTURE.md and CONSTRAINTS.md (sub-agent drafts, orchestrator integration).
- **Action**: Updated TASKS.md and PROGRESS.md to reflect current draft status.
- **Action**: Conceptualized ARCHITECTURE.md storage (removed physical schema), aligned intent terminology (`approve`, `approve_with_modifications`, `deny_with_reason`), and hardened scoping rigor (mandatory dimensions + sentinel IDs).
- **Action**: Updated CONSTITUTION.md and CONSTRAINTS.md to match new terminology and scoping rules.
- **Action**: Drafted IDENTITIES.md and DATA_MODEL.md (sub-agent drafts, orchestrator integration).
- **Action**: Drafted PREDICTION.md, POLICY_ENGINE.md, AGENT_CONTRACT.md, and API_SPEC.md.
- **Action**: Drafted TUI_SPEC.md, WEB_UI_SPEC.md, WORKFLOWS.md, and ACCEPTANCE.md.
- **Action**: Completed Phase 3 (Interface & Workflow Specs) and finalized the Required Document Set.
- **Action**: Hardened orchestration tooling; upgraded `loop.sh` with bash strict mode, ANSI colors, pre-flight checks, and timing; rewritten `LOOP_PROMPT.md` to enforce `AGENTS.md` rules and prescriptive tracking.
- **Action**: Optimized loop for single-task leverage, frequent commits, and resettable progress tracking; added `NEXT_TASK` signal handling.
- **Action**: Defined Phase 4 Implementation Plan (Epics, Milestones, and Test Strategy) in `TASKS.md` and `TEST_STRATEGY.md`.
- [x] **M2.1: SQLite Initialization** (Epic 2) - Implemented `pkg/store/types.go` (Event structs) and `pkg/store/sqlite.go` (connection, WAL, schema migration). Verified with `pkg/store/store_test.go`.

## 2026-01-26
- **Action**: Improved `loop.sh` orchestration script with better argument handling (positional parameters for goal) and added post-run statistics (iteration count, total time, average/min/max duration).
- **Action**: Fixed typoes and cleaned up usage messaging in `loop.sh`.
- **Action**: Committed M2.1 code changes to git repository.
- [x] **M2.2 & M2.3: Event Reader & Replay** (Epic 2) - Implemented `ReadEvents` in `pkg/store/sqlite.go` and verified with comprehensive tests in `pkg/store/store_test.go`.
- [x] **M4.1 & M4.2: Identity & CLI** (Epic 4) - Implemented `ratelord identity add`, `POST /v1/identities` (event write), and `GET /v1/identities` (projection read). Verified end-to-end registration and state restoration.
- [x] **Epic 7: Forecasting** - Implemented forecast model interface, linear burn model, and forecast loop integration. Added `pkg/engine/forecast/` package with types, linear model, projection for history, and service for event emission. Wired into poller and main daemon.
- [x] **M8.1 & M8.2: TUI Foundation & Dashboard** (Epic 8) - Initialized `ratelord-tui` with Bubbletea, implemented polling loop, connected to daemon, and built dashboard view with identity list and live event stream.
- [x] **M10.1: End-to-End Simulation Script** (Epic 10) - Implemented `ratelord-sim` tool to generate realistic traffic patterns and verify system stability.
- [x] **M10.4: Final Acceptance Run** (Epic 10) - Executed full acceptance suite. Identified critical issues in Policy Engine and Drift Persistence (see `ACCEPTANCE_REPORT.md`). Defined Phase 5 Remediation plan.
- **Action**: Verified TUI backend: built `ratelord-d` and `ratelord-tui`, started daemon, confirmed health check and event streaming, killed cleanly.
- **Action**: Updated NEXT_STEPS.md with manual TUI verification instructions.
- **Action**: Marked M12.2 as ready for manual verification in PROGRESS.md and TASKS.md.
- **Action**: Defined Phase 7 (Real Providers) in `TASKS.md`, including Epic 14 (GitHub) and Epic 15 (OpenAI). Updated `NEXT_STEPS.md` to prioritize GitHub integration.
- **Action**: Refined Epic 16 (Dogfooding & Tuning) into concrete steps (M16.1, M16.2, M16.3).
- [x] **M16.1 & M16.2: Dogfood Setup & Run** (Epic 16) - Created `deploy/dogfood` environment with real policy and run scripts. Executed operational run and verified event logging (3 events, 2 polls) with `verify_events.go`.
- [x] **M16.3: Analysis & Tuning** (Epic 16) - Analyzed forecast accuracy with `analyze_forecast.go`. The linear burn model correctly predicted exhaustion times based on synthetic bursty traffic, though with expected variance due to the randomness of the simulation.
- [x] **M13.2: Deployment Guide** (Epic 13) - Drafted `DEPLOYMENT.md` covering Systemd, Docker, and Kubernetes Sidecar patterns.
- [x] **M17.2: Go SDK** (Epic 17) - Implemented `pkg/client` with types, client, and tests. Created example usage in `examples/go/basic/`.
- [x] **M17.3: Python SDK** (Epic 17) - Implemented `sdk/python` package with client, tests, and README. Verified PyPI structure and fail-closed behavior.
- **Action**: Implemented Web UI Dashboard View: Created API client (`web/src/lib/api.ts`), AppShell layout (`web/src/layouts/AppShell.tsx`), Dashboard page (`web/src/pages/Dashboard.tsx`), and updated App.tsx for routing. Built successfully with TypeScript and Vite.
- [x] **M18.4: Build Integration** (Epic 18) - Integrated Web UI into daemon using `//go:embed`. Updated Makefile to build web assets and embed them into `cmd/ratelord-d`. Added `--web-dir` flag to serve dev builds or embedded assets by default.
- [x] **Epic 18: Web UI Implementation** - Status: Completed. Implemented Dashboard, History, and Identity Explorer. React + Vite + Tailwind stack embedded in Go binary.
- [x] **M21.1: Robust Config Loader** (Epic 21) - Implemented `Config` struct in `ratelord-d` with priority: Flags > Env Vars > Defaults. Supported `db`, `policy`, `port`, and `web-dir`. Updated `NewServer` to accept custom address. Verified build.

## 2026-02-06
- [x] **M22.2: Temporal Rules** (Epic 22) - Implemented `TimeWindow` schema in `PolicyConfig`, `Matches` logic in `pkg/engine/policy_time.go`, and integrated it into the Policy Evaluator. Verified with unit tests covering timezone handling, day matching, and cross-midnight ranges.
- **Action**: Expanded Phase 10 Epics (Adaptive Throttling, Trends, Webhooks) in `TASKS.md` with detailed implementation milestones.
- [x] **M23.3: Secure Headers** (Epic 23) - Implemented `withSecureHeaders` middleware in `pkg/api/server.go` adding HSTS, CSP, XFO, etc. Verified with new unit test `pkg/api/server_test.go`.
- [x] **M25.1: Aggregation Schema** (Epic 25) - Added `UsageStat` struct to `pkg/store/types.go`. Updated `migrate()` in `pkg/store/sqlite.go` to create `system_state`, `usage_hourly`, and `usage_daily` tables. Implemented `GetSystemState`, `SetSystemState`, and `UpsertUsageStats` methods with transaction-based batch upsert and table selection logic based on bucket_ts.
- [x] **M25.2: Rollup Worker Core** (Epic 25) - Created `pkg/engine/rollup.go` with `RollupWorker` struct and `Run(ctx)` method. Implemented aggregation logic to read `usage_observed` events, group by bucket hour, calculate min/max/event_count/total_usage (sum deltas), and upsert to `usage_hourly`. Integrated into `cmd/ratelord-d/main.go` to start in background goroutine.
- [x] **M25.3: Trend API** (Epic 25) - Implemented `GetUsageStats` in Store with filtering support. Added `GET /v1/trends` endpoint to API server to serve aggregated usage data. Verified with integration tests.
- [x] **M25.4: Integration Test** (Epic 25) - Created `tests/integration/trends_test.go` to verify end-to-end functionality: from event ingestion -> rollup worker aggregation -> API serving correct statistics.
- [x] **M26.2: Dispatcher** (Epic 26) - Created `pkg/engine/dispatcher.go` with `Dispatcher` struct. Implemented async polling loop, event matching, HTTP dispatch with retries, and cursor management in `system_state`. Integrated into `cmd/ratelord-d/main.go` and verified with unit test `pkg/engine/dispatcher_test.go`.
- [x] **M27.2: Snapshot Worker** (Epic 27) - Implemented `SnapshotWorker` in `pkg/engine/snapshot.go`. Updates `IdentityProjection` and `UsageProjection` to track `lastEventID` and `lastIngestTime`. Snapshot worker periodically serializes both projections to `snapshots` table using the timestamp of the older state to ensure consistency.
- **Action**: Expanded scope of Epic 28 (Advanced Simulation) in `TASKS.md` to include detailed scenario definitions (S-01 to S-05) from `ADVANCED_SIMULATION.md`.
