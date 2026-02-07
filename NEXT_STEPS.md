# NEXT_STEPS: Post-Release v1.0.0

## Current Context
- **v1.0.0 Released**: The system has been tagged and released.
- **Phase 14 In-Flight**: Architecture Convergence & Advanced Logic.
- **Completed**: Phase 10-11, Epic 35, 36.1.
- **Phase 12**: Release Engineering (Completed).
- **Future**: Phase 13 (Scale - Leader Election), Phase 15 (Ecosystem).

## Immediate Actions

 1. **Epic 37: Explainability & Audit** (`TASKS.md`):
       - [x] **M37.1: Decision Explainability**:
           - [x] **M37.1.1: Trace Structs**: Define `RuleTrace` in `pkg/engine`.
           - [x] **M37.1.2: Evaluator Trace**: Update `Evaluate` to capture trace.
           - [x] **M37.1.3: API Exposure**: Return trace in `POST /v1/intent` response.
       - [ ] **M37.2: Compliance Reports**.

 2. **Epic 38: Architecture Convergence** (`TASKS.md`):
       - [ ] **M38.2: Unified Store Audit**: Verify Redis/SQLite parity.

 3. **Epic 33: High Availability** (`TASKS.md`):
       - [ ] **M33.1: Leader Election**.

## Phase History
- [x] **Epic 35**: Canonical Constraint Graph.
- [x] **Epic 34**: Federation UI (M34.2 Done).
- [x] **Epic 39**: MCP Integration.

    - [x] M35.1: Graph Schema.
    - [x] M35.2: Graph Projection.
    - [x] M35.3: Policy Matcher.
    - [x] M35.4: Visualization (API & UI).

- [x] **M34.1**: Cluster View UI & API.
- [x] **Phase 10**: Epics 24-27 Complete.

- [x] **Epic 24**: Adaptive Throttling.
- [x] **Epic 25**: Trends.
- [x] **Epic 26**: Webhooks.
- [x] **Epic 27**: Snapshots.
- [x] **Epic 28**: Advanced Simulation.
- [x] **Epic 29**: Financial Governance.
- [x] **Epic 30**: Cluster Federation.
    - [x] M30.1: Protocol.
    - [x] M30.2: Follower Mode.
    - [x] M30.3: Leader Store (Refactored to M32.1).
- [x] **Epic 31**: Release Automation.
- [x] **Epic 32**: External State Stores.

## Reference
- **Spec**: `CLIENT_SDK_SPEC.md`
- **Plan**: `TASKS.md`
- **Progress**: `PROGRESS.md`
