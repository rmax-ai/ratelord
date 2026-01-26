# NEXT_STEPS: Post-Release v1.0.0

## Current Context
- **v1.0.0 Released**: The system has been tagged and released.
- **Core Features**: Daemon authority, Event sourcing, Policy Engine, Forecasting, TUI, and initial Mock Provider are live.

## Immediate Actions

1. **Phase 7: Real Providers (Continued)**:
   - **M16: Dogfooding & Tuning**:
     - [x] Setup `deploy/dogfood` with real policy/scripts (M16.1).
     - [x] Execute operational run (M16.2).
     - [ ] **Analyze forecast accuracy on real data (M16.3)**:
       - Run `deploy/dogfood/run.sh` for a longer period (e.g. 1 hour) or simulate bursty usage.
       - Query `forecast_computed` events.
       - Compare predicted `exhaustion_time` vs actual usage slope.
   - Tune the linear burn model based on real bursty data.

## Reference
- **Release Notes**: `RELEASE_NOTES.md`
- **Report**: `ACCEPTANCE_REPORT.md`
- **Plan**: `TASKS.md`
