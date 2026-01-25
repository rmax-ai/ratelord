# NEXT STEPS: Phase 5 - Remediation

The Final Acceptance Run (M10.4) revealed critical gaps in the Policy Engine and Persistence layer. While the core daemon works, it fails to enforce rules or persist provider drift. The next session must focus on debugging and fixing these specific issues before the system can be considered "production ready".

## Current Objective: Fix Policy Engine & Persistence

### Tasks for Next Session:

1.  **M11.1: Debug Policy Loading**
    - Investigate `pkg/engine/loader.go` and `pkg/engine/policy.go`.
    - Ensure rules from `policy.yaml` are actually loaded into memory and matched against intents.
2.  **M11.2: Implement Wait/Modify Actions**
    - The `Evaluate` function needs to return correct `Wait` or `Modify` instructions, not just Approve/Deny.
3.  **M12.1: Persist Provider State**
    - Ensure that when the daemon restarts, it remembers the last known external usage to prevent drift resets.

## Reference
- **Report**: `ACCEPTANCE_REPORT.md`
- **Plan**: `TASKS.md`
