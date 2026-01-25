# NEXT STEPS: Phase 4 - Implementation & Verification

The storage layer foundation (SQLite connection, schema, types) is in place (M2.1).

## Current Objective: Epic 2 - Storage Layer (Event Sourcing)

We need to implement the write path for the event log.

### Tasks for Next Session:
1.  **Implement Event Writer**: Add `AppendEvent` method to `Store` in `pkg/store/sqlite.go` (M2.2).
    - Must serialize the payload and other fields to JSON.
    - Must be atomic.
2.  **Verify Writer**: Add a test case to `pkg/store/store_test.go` to write an event and verify no errors occur.
    - (Note: We can't read it back yet until M2.3, but we can check the DB row count or lack of error).

## Reference
- **Plan**: `TASKS.md` (Epic 2)
- **Validation**: `TEST_STRATEGY.md` (Integration Testing)
- **Architecture**: `ARCHITECTURE.md` (Data Model)
