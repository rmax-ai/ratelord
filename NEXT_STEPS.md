# NEXT STEPS: Phase 4 - Implementation & Verification

The project foundation is set. We have a running daemon skeleton that handles signals and logs structured events.

## Current Objective: Epic 2 - Storage Layer (Event Sourcing)

We need to implement the persistent storage engine. This is the heart of the system.

### Tasks for Next Session:
1.  **Define Event Structs**: Create `pkg/store/types.go` to define the `Event` struct and basic interfaces.
2.  **Initialize SQLite**: Implement `pkg/store/sqlite.go` to open a DB connection and enable WAL mode (M2.1).
3.  **Schema Migration**: Write the SQL to create the `events` table on startup (M2.1).
4.  **Verify**: Write a small test or main-loop integration to open the DB and check if the file is created.

## Reference
- **Plan**: `TASKS.md` (Epic 2)
- **Validation**: `TEST_STRATEGY.md` (Integration Testing)
- **Architecture**: `ARCHITECTURE.md` (Data Model)
