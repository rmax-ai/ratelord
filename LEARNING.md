# LEARNING: ratelord Insights & Reflections

## 2026-01-25: Documentation Phase Completion

### What Worked Well
1.  **Seed Prompt Constraints**: The `PROJECT_SEED.md` file acted as a highly effective "constitution". It prevented the common failure mode of sub-agents inventing new features (e.g., "let's add a Redis backend") by strictly enforcing the "local-first, zero-ops" constraint.
2.  **Explicit State Tracking**: The combination of `TASKS.md` (hierarchical), `PROGRESS.md` (status), and `NEXT_STEPS.md` (continuation context) meant that even with context window resets, the orchestrator never lost track of the sequence.
3.  **Sequential Drafting**: Creating the documents in the specific order defined in `PROJECT_SEED.md` allowed concepts to build naturally. `ARCHITECTURE.md` defined the components that `API_SPEC.md` later exposed, preventing circular dependencies in the specs.

### Challenges Encountered
1.  **Terminology Drift**: Initial drafts occasionally slipped into using "Quota" vs "Constraint" or "Limit". We had to enforce a strict vocabulary review to ensure "Constraint Graph" was the dominant mental model, not "Rate Limit Table".
2.  **Scope Creep in Policy**: There was a temptation to make the Policy Engine a full Turing-complete language (Lua/WASM). We restricted it to a YAML-based rule set to maintain the "Constitutional" non-negotiable nature and keep the daemon simple.

### Improvements for Implementation Phase
1.  **Code-Doc Lockstep**: When implementation begins, strict adherence to `API_SPEC.md` is required. If the code *needs* to diverge, the spec must be updated *first* (Spec-Driven Development).
2.  **Testing Strategy**: The `ACCEPTANCE.md` provides good end-to-end scenarios, but unit testing the "Prediction Engine" with deterministic time mocking will be critical and wasn't fully detailed in the docs.

### Metric of Success
The "Required Document Set" is complete with 0 implementation code written, preserving a pure design phase. This should reduce code churn significantly.
