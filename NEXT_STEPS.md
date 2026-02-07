# NEXT_STEPS: User Documentation & Quality Assurance

## Current Context
- **Documentation**: Drafted core user guides (`concepts/architecture.md`, `concepts/core-model.md`, `reference/api.md`, `guides/mcp.md`).
- **Phase 16 (QA)**: Test coverage goals met for API/Store/Engine.
- **Phase 14 & 15**: Complete.

## Immediate Actions
- [x] **Review Documentation**: Verify the newly created user docs in `docs/` align with the implementation and are clear.
- [x] **M42.4 (Audit)**: Updated `docs/index.md` to link all guides (`web-ui.md`, `cli.md`) and verified `docs/installation.md` and `docs/configuration.md`.
- [x] **M42.5 (Improvement)**: Enhanced `docs/installation.md`, switched to YAML in `docs/configuration.md` & `docs/guides/deployment.md`, and implemented YAML support in `pkg/engine`.
- [x] **M42.3 (Refined)**: Added Policy Guide and moved/refined Deployment Guide.
- [x] **SDK Docs**: Refreshed JS, Python, and Go SDK documentation to remove draft warnings and match implementation.
- [x] **M42.5+**: Refined Policy Guide (Shared vs Isolated) and Configuration Guide (Defaults).
- [x] **MCP Guide**: Created `docs/guides/mcp.md` and linked from index.
- [x] **Docs Polish**: Added Financial Governance to Policy Guide and Cluster Federation to Deployment Guide.
- [x] **Project Assessment**: Verified test coverage and identified missing features. See `ASSESSMENT.md`.
- [x] **M43.1 (Reports)**: Implement real CSV generation in `pkg/reports/csv.go` and add tests.
- [x] **M43.2 (Graph)**: Implement `PolicyUpdated` handling in `pkg/graph/projection.go` and add adjacency index.
- [ ] **M43.3 (Hardening)**: Fix hardcoded `resetAt`, fix API pool ID, and add tests for `pkg/mcp` and `pkg/blob`.
- [ ] **M43.4 (Cleanup)**: Address TODOs in Federation, Poller, and Provider packages (from Assessment).
- [ ] **Phase 16 Continues**: Final pre-release validation and debt paydown.

## Phase History
- [x] **M42.1, M42.2, M42.3**: User Documentation (Concepts, API Ref, Guides).
- [x] **M41.3: Engine Package Coverage** (Achieved 80.9%).
- [x] **M41.2: Store Package Coverage** (Achieved 80.8%).
- [x] **M41.1: API Package Coverage** (Achieved 80.1%).
- [x] **M31.3**: Documentation & Changelog (Release Automation).
- [x] **M37.3**: Policy Debugging (Trace Mode & UI).
- [x] **M36.3**: Compliance & Deletion (GDPR support).
- [x] **M36.2**: Cold Storage Offload (ArchiveWorker + LocalBlobStore).
- [x] **Epic 40**: Client Resilience (M40.1, M40.2, M40.3).
- [x] **Epic 33**: High Availability (M33.1, M33.4).
- [x] **Epic 35**: Canonical Constraint Graph.
- [x] **Epic 39**: MCP Integration.
- [x] **M36.1**: Retention Policy Engine.

## Reference
- **Spec**: `CLIENT_SDK_SPEC.md`
- **Plan**: `TASKS.md`
- **Progress**: `PROGRESS.md`
