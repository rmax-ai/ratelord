# NEXT STEPS: Phase 4 - Implementation & Verification

The storage layer foundation (SQLite connection, schema, types) is in place (M2.1).

## Current Objective: Epic 2 - Storage Layer (Event Sourcing)

We have the write path (M2.2). Now we need to implement the read path for replay and hydration.

### Tasks for Next Session:
1.  **Implement Event Reader**: Add `ReadEvents` method to `Store` in `pkg/store/sqlite.go` (M2.3).
    - Should accept an offset (sequence/limit) and return a channel or slice.
    - Must deserialize the JSON payload correctly.
2.  **Verify Reader**: Update `pkg/store/store_test.go`.
    - Add a test that writes multiple events, then reads them back in order.
    - Verify data integrity (payloads match).

## Reference
- **Plan**: `TASKS.md` (Epic 2)
- **Validation**: `TEST_STRATEGY.md` (Integration Testing)
- **Architecture**: `ARCHITECTURE.md` (Data Model)
