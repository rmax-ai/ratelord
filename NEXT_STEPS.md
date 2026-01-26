# NEXT_STEPS: Post-Release v1.0.0

## Current Context
- **v1.0.0 Released**: The system has been tagged and released.
- **Core Features**: Daemon authority, Event sourcing, Policy Engine, Forecasting, TUI, and initial Mock Provider are live.
- **Phase 7 Complete**: Real providers (GitHub, OpenAI) implemented. Dogfooding environment validated forecast accuracy.
- **Documentation**: Deployment guide (`DEPLOYMENT.md`) is now available.

## Immediate Actions

1. **Phase 8: Operations & Expansion**:
   - **M17.1: Production Deployment Guide**:
     - [x] Draft `DEPLOYMENT.md` for running Ratelord in production (systemd, docker, k8s sidecar).
     - [x] Document configuration best practices (env vars, secrets).
   - **M17.2: Client SDKs**:
     - [ ] Outline requirements for Go/TS/Python SDKs in `CLIENT_SDK_SPEC.md`.
     - [ ] Implement basic Go SDK `pkg/client`.

## Reference
- **Release Notes**: `RELEASE_NOTES.md`
- **Deployment**: `DEPLOYMENT.md`
- **Report**: `ACCEPTANCE_REPORT.md`
- **Plan**: `TASKS.md`
