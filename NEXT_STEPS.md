# NEXT STEPS: Phase 4 - Implementation & Verification

The API server now implements the Intent Stub (M3.2) and Health/Diagnostics endpoints (M3.3). We can negotiate mock intents and view recent events.

## Current Objective: Epic 4 - Identity & CLI Basics (M4.1 & M4.2)

We need to enable the registration of identities (M4.1) to prove the write-path works end-to-end, and project that state (M4.2) to make it queryable.

### Tasks for Next Session:
1.  **Implement CLI Identity Command**:
    -   Create `cmd/ratelord` (CLI entrypoint).
    -   Implement `identity add <name>` subcommand.
    -   The CLI should make an HTTP POST to the daemon (or write to DB if we choose direct access for now, but API is preferred for "daemon authority").
    -   *Correction*: `API_SPEC.md` doesn't have an identity registration endpoint yet. We might need to implement `POST /v1/identities` or have the CLI speak directly to the store for bootstrapping (or define the endpoint).
    -   *Decision*: Let's stick to Daemon Authority. Add `POST /v1/identities` to `API_SPEC.md` and implement it in `server.go`, then have CLI call it.
    -   M4.1: Identity Registration.

2.  **Implement Identity Projection**:
    -   Create `pkg/engine/projection.go`.
    -   Implement a basic in-memory map of identities.
    -   Hook it into the `Replay` loop in `main.go`.
    -   M4.2: State Projection.

## Reference
- **Plan**: `TASKS.md` (Epic 4)
- **Architecture**: `ARCHITECTURE.md` (Identity)
