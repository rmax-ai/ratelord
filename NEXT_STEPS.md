# NEXT_STEPS: Post-Release v1.0.0

## Current Context
- **v1.0.0 Released**: The system has been tagged and released.
- **Core Features**: Daemon authority, Event sourcing, Policy Engine, Forecasting, TUI, and initial Mock Provider are live.
- **Phase 7 Complete**: Real providers (GitHub, OpenAI) implemented. Dogfooding environment validated forecast accuracy.
- **Documentation**: Deployment guide (`DEPLOYMENT.md`) is now available. `CLIENT_SDK_SPEC.md` has been drafted.

## Immediate Actions

 1. **Phase 9: Ecosystem & Hardening**:
     - **Epic 21: Configuration & CLI Polish**:
       - [ ] **M21.1: Robust Config Loader**: Implement Env Var and Flag support for DB path, Policy path, and Port. (Resolves M1.4 debt).
     - **Epic 19: Node.js SDK**:
       - [ ] **M19.1: SDK Specification**: Draft TypeScript interfaces.

 ## Reference

- **Spec**: `CLIENT_SDK_SPEC.md`
- **Release Notes**: `RELEASE_NOTES.md`
- **Deployment**: `DEPLOYMENT.md`
- **Report**: `ACCEPTANCE_REPORT.md`
- **Plan**: `TASKS.md`
