# NEXT STEPS: Phase 4 - Implementation & Verification

The system now has a basic identity registration loop (CLI -> API -> Store -> Projection).

## Current Objective: Epic 5 - Usage Tracking & Policy Engine (M5.1 & M5.2)

We need to implement the core logic for tracking usage against limits and making policy decisions.

### Tasks for Next Session:
1.  **Implement Usage Tracking**:
    -   Create `pkg/engine/usage.go`.
    -   Implement `UsageProjection` to track usage by identity/scope/window.
    -   Hook it into the `Replay` loop.
    -   M5.1: Usage Tracking.

2.  **Implement Policy Engine**:
    -   Create `pkg/engine/policy.go`.
    -   Implement `Evaluate(intent)` which checks usage against limits.
    -   Update `POST /v1/intent` to use the real policy engine instead of the stub.
    -   M5.2: Policy Enforcement.

## Reference
- **Plan**: `TASKS.md` (Epic 5 - To Be Added)
- **Architecture**: `ARCHITECTURE.md` (Policy Engine)
