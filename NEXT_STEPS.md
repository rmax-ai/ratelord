# NEXT STEPS: Phase 4 - Implementation & Verification

The forecasting engine is now integrated (Epic 7). The final epic before functional completion is the TUI dashboard to visualize the system state.

## Current Objective: Epic 8 - TUI & Visualization

We need to implement a terminal user interface to visualize the system state (Identities, Pools, Forecasts).

### Tasks for Next Session:
1. **Initialize TUI Foundation**:
   - Create `cmd/ratelord-tui/main.go` (or similar).
   - Initialize Bubbletea model.
   - Connect to `GET /v1/events` and `GET /v1/identities`.
   - M8.1: TUI Foundation.

2. **Implement Dashboard View**:
   - Render Usage Bars per pool (Capacity vs Used).
   - Render Time-to-Exhaustion (Forecast) if available.
   - Render recent Event Log scrolling view.
   - M8.2: Dashboard View.

3. **Verify End-to-End**:
   - Run daemon.
   - Register identity.
   - Ingest mock usage.
   - Verify TUI shows updates.

## Reference
- **Plan**: `TASKS.md` (Epic 8)
- **Design**: `TUI_SPEC.md`
