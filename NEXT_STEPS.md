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
     - [x] **M19.5: Release Prep**: Configure `package.json` exports/files and document publish process.

 2. **Epic 20: Operational Visibility**:
     - [x] **M20.1: Prometheus Exporter**: Expose `/metrics`.
     - [x] **M20.2: Logging Correlation**: Ensure `trace_id` / `intent_id` is threaded through logs.
     - [ ] **M20.3: Grafana Dashboard**: Create visualization.

 3. **Epic 22: Advanced Policy Engine**:
     - [x] **M22.1: Soft Limits & Shaping**:
         - [x] **M22.1.1: Policy Action Types**: Add `warn` and `delay` actions.
         - [x] **M22.1.2: Evaluator Updates**: Update `Evaluate` logic.
         - [x] **M22.1.3: API Response**: Ensure warnings/delays are propagated.
     - [ ] **M22.2: Temporal Rules**: Implement time-based matching.

 4. **Epic 23: Security Hardening**:
     - [ ] **M23.1: TLS Termination**: Support HTTPS.
     - [ ] **M23.2: API Authentication**: Implement Bearer auth (Token Mgmt + Middleware).
     - [ ] **M23.3: Secure Headers**: Add security headers.

## Reference

- **Spec**: `CLIENT_SDK_SPEC.md`
- **Plan**: `TASKS.md`
- **Progress**: `PROGRESS.md`
