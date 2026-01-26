# NEXT_STEPS: Phase 5 - Remediation

## Current Context
- **Policy Engine**: Full loop is working. Loading (M11.1), enforcement (M5.2), shaping/deferring (M11.2), and hot reloading (M11.3) are verified.
- **Provider State**: Currently, the poller replays events but might not be fully persisting granular provider state (cursors/offsets) explicitly in a way that survives restarts without full replay. Actually, the event sourcing model *is* the persistence. `ratelord-d` startup replays events to rebuild state. The question for M12.1 is: does `Poller` properly initialize its *internal cursor* from those replayed events?
- **TUI**: Needs manual verification (M12.2).

## Immediate Actions

1. **Persist Provider State (M12.1)**:
   - Audit `pkg/engine/poller.go` and `pkg/provider/mock.go`.
   - Verify: When `ratelord-d` restarts and replays events, does the `Poller` (or `Provider`) know where it left off? Or does it start from scratch/zero?
   - If it starts from scratch, we might get duplicate events or miss data.
   - Task: Ensure `Poller` or `Provider` exposes a way to "restore" state from the replayed event stream (e.g., last seen cursor/timestamp).

2. **TUI Verification (M12.2)**:
    - Backend verified: daemon builds, starts, serves health/events, and polls providers.
    - Run the daemon: `go run cmd/ratelord-d/main.go` (or `./ratelord-d` if built).
    - In another terminal, run the TUI: `go run cmd/ratelord-tui/main.go` (or `./ratelord-tui`).
    - Verify the TUI displays usage bars, event log, and updates in real-time.
    - Manual step: Confirm visuals and interactivity work as expected.

## Reference
- **Report**: `ACCEPTANCE_REPORT.md`
- **Plan**: `TASKS.md`
