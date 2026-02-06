# NEXT_STEPS: Post-Release v1.0.0

## Current Context
- **v1.0.0 Released**: The system has been tagged and released.
- **Phase 11 In-Flight**: Advanced Capabilities (Simulation, Finance, Federation).
- **Completed**: Phase 10 (Epics 24-27) + Epic 28 + Epic 29 + Epic 30 + Epic 31 + Epic 32.
- **Phase 12**: Release Engineering (Completed).
- **Future**: Phase 13 (Scale) & Phase 14 (Architecture Convergence).

## Immediate Actions

 1. **Epic 35: Canonical Constraint Graph** (`TASKS.md`):
       - [x] **M35.1: Graph Schema Definition**: Define Nodes and Edges in `pkg/graph`.
       - [x] **M35.2: Graph Projection**: Implement in-memory graph builder from events (and Policies).
       - [x] **M35.3: Policy Matcher**: Use graph for policy lookups.
           - [x] M35.3.1: Traversal Logic (FindConstraintsForScope).
           - [x] M35.3.2: Engine Integration.
       - [ ] **M35.4: Visualization**: API and UI.

 2. **Epic 38: Architecture Convergence** (`TASKS.md`):
       - [ ] **M38.1: Constraint Graph Integration**: Refactor Policy Engine to use Graph.
           - [x] Refactor Policy Engine (Done in M35.3).
           - [ ] Ensure Federation respects Graph.
       - [ ] **M38.2: Unified Store Audit**: Verify Redis/SQLite parity.

 3. **Epic 34: Federation UI** (`TASKS.md`):
       - [ ] **M34.2: Node Diagnostics**: Visualize Replication Lag & Election Status.

## Phase History
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
