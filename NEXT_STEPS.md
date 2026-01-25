# NEXT STEPS: Phase 4 - Implementation & Verification

The system now has identity registration, usage tracking (stubbed data), and a basic policy engine wired up.

## Current Objective: Epic 6 - Provider Integration & Real Data (M6.1 & M6.2)

We need to start ingesting real data to drive the usage projection and make the policy engine useful.

### Tasks for Next Session:
1.  **Define Provider Interface**:
    -   Create `pkg/provider/types.go`.
    -   Define the `Provider` interface (Poll, status).
    -   M6.1: Provider Abstraction.

2.  **Implement Mock Provider**:
    -   Create `pkg/provider/mock.go`.
    -   Implement a provider that generates synthetic usage data.
    -   Wire it into `ratelord-d` main loop to emit `provider_poll_observed` events.
    -   M6.2: Mock Data Flow.

## Reference
- **Plan**: `TASKS.md` (Epic 6 - To Be Added)
- **Architecture**: `ARCHITECTURE.md` (Provider Integration)
