# NEXT_STEPS: Post-Release v1.0.0

## Current Context
- **v1.0.0 Released**: The system has been tagged and released.
- **Phase 9 Initiated**: Ecosystem expansion (Node.js SDK) and hardening.
- **Scope Expanded**: Phase 9 tasks have been detailed to include testing and scaffolding.

## Immediate Actions

 1. **Epic 19: Node.js / TypeScript SDK**:
     - [x] **M19.1: SDK Specification**: Define TypeScript interfaces in `CLIENT_SDK_SPEC.md` or `sdk/js/README.md`.
     - [x] **M19.2: Project Scaffold**: Initialize `sdk/js` directory and tools.
     - [x] **M19.3: Core Implementation**: Implement `RatelordClient` and `ask` method.
     - [x] **M19.4: Integration Verification**: Create a sample script and verify against daemon.
     - [ ] **M19.5: Release Prep**: Configure `package.json` exports/files and document publish process.

 2. **Epic 20: Operational Visibility**:
     - [ ] **M20.1: Prometheus Exporter**: Expose `/metrics`.

## Reference

- **Spec**: `CLIENT_SDK_SPEC.md`
- **Plan**: `TASKS.md`
- **Progress**: `PROGRESS.md`
