# NEXT_STEPS: Post-Release v1.0.0

## Current Context
- **v1.0.0 Released**: The system has been tagged and released.
- **Core Features**: Daemon authority, Event sourcing, Policy Engine, Forecasting, TUI, and initial Mock Provider are live.

## Immediate Actions

1. **Phase 7: Real Providers (Continued)**:
   - **M16: Dogfooding & Tuning**:
     - Setup `deploy/dogfood` with real policy/scripts (M16.1).
     - Execute operational run (M16.2).
     - Analyze forecast accuracy on real data (M16.3).
   - Deploy `ratelord-d` internally to monitor CI/CD tokens.
   - Tune the linear burn model based on real bursty data.

## Reference
- **Release Notes**: `RELEASE_NOTES.md`
- **Report**: `ACCEPTANCE_REPORT.md`
- **Plan**: `TASKS.md`
