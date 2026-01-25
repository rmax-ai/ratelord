# NEXT STEPS: Phase 4 - Implementation & Verification

The system now has a provider interface and a mock implementation. The next critical step is to actually run the provider and ingest its data.

## Current Objective: Epic 6 - Provider Integration (M6.3)

We need to implement the polling loop that drives the provider and writes observations to the store.

### Tasks for Next Session:
1.  **Implement Polling Orchestrator**:
    -   Create `pkg/engine/poller.go`.
    -   Implement `Poller` struct that manages a set of Providers.
    -   Run a loop (ticker) that calls `Poll()` on each provider.
    -   Convert `PollResult` into `store.Event` (provider_poll_observed, usage_observed, etc.).
    -   Append events to the store.
    -   M6.3: Continuous Polling.

2.  **Wire up Main**:
    -   Update `cmd/ratelord-d/main.go` to initialize the `Poller`, register the `MockProvider`, and start the polling loop.

## Reference
- **Plan**: `TASKS.md` (Epic 6)
- **Architecture**: `ARCHITECTURE.md` (Provider Integration)
