# TEST STRATEGY: ratelord

This document defines how we verify the `ratelord` system at different layers of abstraction, ensuring adherence to `ACCEPTANCE.md`.

## 1. Unit Testing (Logic Layer)
*   **Scope**: Pure functions and isolated logic (Policy Engine, Event Serializers, Projection Reducers).
*   **Approach**:
    *   Test `PolicyEvaluator`: Given a set of Inputs + Rules, verify Decision (Approve/Deny).
    *   Test `ForecastModel`: Given a time-series of usage, verify P99 calculation.
    *   Test `EventNormalization`: Given a raw provider payload, verify canonical Event struct.
*   **Goal**: High coverage of business logic without IO.

## 2. Integration Testing (Persistence Layer)
*   **Scope**: `ratelord-d` interactions with SQLite.
*   **Approach**:
    *   **Store Test**:
        1. Initialize fresh DB.
        2. Append 100 events.
        3. Close DB.
        4. Re-open and Read events.
        5. Assert Read == Appended.
    *   **Concurrency Test**: Multiple writers (if supported) or rapid serial writes to verify WAL integrity.
*   **Goal**: Prove `D-01` (Clean Start), `D-02` (Crash Recovery), and `D-04` (Immutability).

## 3. End-to-End (E2E) Testing (Daemon Layer)
*   **Scope**: The running `ratelord-d` process and its API.
*   **Tooling**: `curl`, `jq`, or a custom test harness script.
*   **Procedure**:
    1. **Setup**: Start `ratelord-d` (clean env).
    2. **Register**: Call `POST /identities` (or CLI) -> Verify 200 OK.
    3. **Traffic**: Loop `POST /intent` 100 times.
    4. **Verify**: Call `GET /events` -> Assert count >= 101.
    5. **Recovery**: `kill -9` daemon -> Restart -> Call `GET /health`.
*   **Goal**: Prove `A-01`, `A-02`, `D-03`, `T-01`.

## 4. Acceptance Matrix

| Epic | Test Level | Acceptance Criteria |
| :--- | :--- | :--- |
| **Storage** | Integration | `D-01`, `D-02`, `D-04`, `D-05` |
| **API** | E2E | `A-01`, `A-02` |
| **Identity** | E2E | `D-06`, `T-03` |
| **Lifecycle** | E2E | `D-03` |
