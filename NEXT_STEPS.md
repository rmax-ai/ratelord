# NEXT STEPS: Phase 4 - Implementation & Verification

The storage layer (M2.1, M2.2, M2.3) is complete. We have an append-only event log and a reader for replay.

## Current Objective: Epic 3 - API Layer (M3.1)

Now we need to expose this functionality via an API.

### Tasks for Next Session:
1.  **Implement HTTP Server Shell**: Create `pkg/api/server.go`.
    -   Define a `Server` struct.
    -   Bind to `127.0.0.1:8090`.
    -   Setup a basic router (using `net/http` or `chi` if we decide to add deps, but `net/http` preferred for now).
    -   M3.1: Server Shell & Middleware.
2.  **Wire up Main**: Update `cmd/ratelord-d/main.go` to start the server.

## Reference
- **Plan**: `TASKS.md` (Epic 3)
- **Architecture**: `ARCHITECTURE.md` (API Layer)
