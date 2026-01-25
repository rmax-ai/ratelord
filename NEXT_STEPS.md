# NEXT STEPS: Phase 4 - Implementation & Verification

The API server shell (M3.1) is now implemented and wired into the daemon. It starts in the background and shuts down gracefully.

## Current Objective: Epic 3 - API Layer (M3.2 & M3.3)

We need to make the API useful by implementing the Intent negotiation endpoint and basic diagnostics.

### Tasks for Next Session:
1.  **Implement Intent Endpoint (Stub)**: Update `pkg/api/server.go`.
    -   Define `IntentRequest` and `IntentResponse` structs (matching `API_SPEC.md`).
    -   Implement `handleIntent` to accept POST requests.
    -   Validate input (agent_id, scope, etc.).
    -   Return a hardcoded `approved` decision for now (just to prove plumbing).
    -   M3.2: Intent Endpoint.
2.  **Add Diagnostics**:
    -   Implement `GET /v1/events` to list recent events from the store.
    -   This helps verification without needing a CLI tool yet.
    -   M3.3: Health & Diagnostics.

## Reference
- **Plan**: `TASKS.md` (Epic 3)
- **Architecture**: `ARCHITECTURE.md` (API Layer)
