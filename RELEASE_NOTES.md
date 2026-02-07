## [Unreleased] - 2026-02-07

### Features
- implement trace mode persistence and UI visualization (391a64a)
- implement trace mode logging and api flag (54a9ea4)
- implement identity deletion and GDPR pruning (M36.3) (1a77402)
- implement cold storage offload worker and blob store (e446fdd)
- implement random exponential backoff (M40.3) (c84291f)
- implement leadership epochs for split-brain protection (9452634)
- implement backoff and jitter for resilience (M40.2) (8bb00c4)
- implement redis lease epochs and fix follower mode (ba596a6)
- implement backoff and retry logic for SDK resilience (23e1255)
- implement compliance reports (M37.2) (a403ead)
- implement rule traces for decision explainability (9a60e2c)
- implement advanced tools (config resource, check_usage tool) (5f6b27f)
- implement mcp server, resources, and tools (Epic 39) (5551fb1)
- implement node diagnostics (metadata in heartbeat) and expand epics scope (389af91)
- ensure grant requests respect graph policy and usage (2251b1a)
- implement M36.1 retention policy engine (4197317)
- implement graph visualization API and UI (84e9563)
- integrate constraint graph for policy lookup (4a7b28a)
- populate constraint nodes from policy config (ce68f56)
- implement canonical constraint graph (M35.1, M35.2) (50ad60c)
- implement cluster view UI (83e7117)
- implement cluster topology API and expand phase 14 epics (9eecd48)
- implement client routing redirect to leader (9605816)
- implement ElectionManager and integrate into daemon for M33.2 (626a81d)
- implement distributed lease stores (redis, sqlite) (8227c02)
- implement atomic Redis operations (M32.3) (fd98593)
- implement RedisUsageStore and expand roadmap epics (1088a99)
- implement RedisUsageStore and config (31d89b1)
- complete cluster federation with storage abstraction (1a927ec)
- implement follower mode and remote provider (M30.2) (55f19f6)
- implement federation grant endpoint (M30.1) (c9834f0)
- implement Epic 29 Financial Governance (a7bdd04)
- implement structured reporting and invariant verification (M28.3) (32de83a)
- implement scenarios S-01 to S-05 and update sim config (cdd4c63)
- upgrade simulation engine for scenario support (M28.1) (caecd99)
- implement event pruning (M27.4) and admin cli (6c6c866)
- implement startup optimization and full state recovery (0e459f2)
- add CLUSTER_FEDERATION and FINANCIAL_GOVERNANCE documentation (614caf7)
- update document manifest with new required documents for release and simulation (a326591)
- add user documentation section and update guidelines for user docs synchronization (87df645)
- add example verification guidelines to LOOP_PROMPT (0264995)
- implement snapshot worker and expand sim scope (e19fcdc)
- enhance test verification loop with failure handling and documentation synchronization (8cbc93c)
- enhance test verification loop with failure handling and coverage requirements (a505f22)
- enhance simulation engine with new scenarios and AgentBehavior interface (dabfe00)
- implement snapshot schema and migration (5df7af2)
- update advanced simulation framework documentation and add new epic for simulation tasks (5a108de)
- implement HMAC signing (M26.3) and Grafana dashboard (M20.3) (ab7983e)
- implement webhook dispatcher (M26.2) (b45072d)
- implement webhook registry (schema + api) (e6a57af)
- implement long-term aggregation and API (Epic 25) (bfe76e3)
- implement dynamic delay controller (Epic 24) (5d1875b)
- implement M23.3 secure headers and expand phase 10 epics (6c07fb5)
- implement API authentication and expand roadmap (52709a1)
- implement TLS termination (M23.1) (bb67326)
- implement temporal policy rules (M22.2) (019dd15)
- implement soft limits (warn/delay) and api response updates (8761cc8)
- verify integration and expand plan scope (cadf383)
- implement core client logic and expand tracking scope (8ca227d)
- initialize node.js/typescript sdk scaffold (6619e39)
- implement robust config loader with env vars and CLI flags (50837e0)
- implement identity explorer with client-side hierarchy (07f0a97)
- implement history view (M18.5) (2de696c)
- implement basic web ui and dashboard (bf597ad)
- implement python client (M17.3) (b7e067b)
- implement basic Go client SDK and examples (a3c1d34)
- implement initial Go client SDK (f077eea)
- add forecast analysis and enhance event verification (f11c380)
- implement openai rate limit provider (46b2334)
- register github providers from policy config (M14) (d01098d)
- implement github rate limit provider (Epic 14) (4b78b94)
- add stats and improve goal handling in loop script (cae69c8)
- add iteration performance statistics (6f34eca)

### Fixes
- add tenacity dependency (1d9de55)
- enhance cleanup and logging in orchestration loop (79f2498)
- filter out INFO logs from opencode output in background process (ee8b7c2)
- resolve build conflicts in dogfood scripts and fix flaky github provider test (0446711)

### Documentation
- update cluster/federation endpoints with metadata (a15d50b)
- update status for M33.1 completion (4174ba7)
- add test and release workflows (cee64bb)
- update next steps for Epic 30 (2a31233)
- define grant protocol in API spec (60105a4)
- complete phase 10, activate phase 11 (simulation) (81d2b49)
- implement trace_id logging (037f3b5)
- expand epics scope and define TS SDK spec (9f571f0)
- mark config tasks complete and update next steps (4ce7937)
- mark M18.6 and Epic 18 as complete (e56ba86)
- check off M17.3 (6b59db2)
- mark python sdk as complete (dd29fef)
- track Python SDK implementation tasks (9cba803)
- complete client SDK specification and update tracking docs (c091f60)
- update deployment status and add client SDKs phase (68659e2)
- add DEPLOYMENT.md and update tracking docs (a53aaa0)
- update progress and tasks for current phase (2f96eee)
- record M16.1 and M16.2 completion (4977951)
- update NEXT_STEPS with dogfood status (12971b6)
- refine M16 dogfooding plan (5388184)
- update progress and tasks after OpenAI provider completion (a36853a)
- update status after completing Epic 14 (959d3a9)
- define Phase 7 real providers and github epic (649d4ea)
- update progress, tasks, and next steps for v1.0.0 release (ed62aa6)

### Maintenance
- verify and fix usage store parity (M38.2) (4e218ba)
- verify operational run and event logging (e10a189)
- setup dogfood environment (14cbbde)
- add github integration test (M14) (bbdb934)

# Release Notes: ratelord v1.0.0

## Overview

ratelord v1.0.0 marks the initial stable release of the local-first, daemon-authoritative rate limit orchestrator. This release establishes the core event-sourcing architecture, policy engine, and predictive capabilities designed to manage API consumption quotas across distributed agents.

## Key Features

### Core Infrastructure
- **Daemon-Authority Model**: `ratelord-d` acts as the single source of truth for rate limit state.
- **Event Sourcing**: Immutable SQLite-based event log for auditability and replayable state.
- **Resumability**: Robust crash recovery and state reconstruction from the event stream.

### Usage & Policy Management
- **Hierarchical Tracking**: Tracks usage across identities, scopes, and shared/isolated pools.
- **Policy Engine**: Dynamic rule evaluation for Approving, Denying, or Shaping traffic.
- **Hot Reloading**: Support for `SIGHUP` to reload policy configurations without downtime.
- **Drift Detection**: Automatic detection and correction of usage drift against external providers.

### Prediction & Forecasting
- **Time-to-Exhaustion**: Forecasts P50/P90/P99 exhaustion times based on usage history.
- **Linear Burn Model**: Initial forecasting model for linear consumption patterns.

### Observability
- **TUI Dashboard**: Terminal User Interface for real-time monitoring of usage, events, and forecasts.
- **Structured Logging**: JSON-formatted logs for easy integration with observability tools.

## Known Limitations
- **TUI**: Currently read-only; administrative actions must be performed via CLI or config files.
- **Providers**: Only Mock Provider is fully implemented; external providers (GitHub, OpenAI) are planned for future releases.

## Getting Started
1. Build the daemon: `make build`
2. Run the daemon: `./bin/ratelord-d`
3. Register an identity: `./bin/ratelord identity add <name> <kind>`
