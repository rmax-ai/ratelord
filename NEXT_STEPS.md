# NEXT STEPS: Phase 5 - Remediation

The Policy Engine loading and evaluation logic (M11.1) and the client-side modification handling (M11.2) have been implemented and verified with basic manual tests. The simulator now correctly sleeps if a `wait_seconds` modification is received.

## Current Objective: Finalize Robustness

### Tasks for Next Session:

1.  **M11.3: Verify Hot Reload**
    - Create a test case that modifies `policy.json` and sends `SIGHUP`.
    - Verify that new rules (e.g., stricter limits) take effect immediately without restart.
2.  **M12.1: Persist Provider State**
    - Ensure that when the daemon restarts, it remembers the last known external usage to prevent drift resets.
    - Check `engine/poller.go` replay logic.
3.  **M12.2: TUI Verification**
    - Manually verify TUI dashboard connects and displays data.

## Reference
- **Report**: `ACCEPTANCE_REPORT.md`
- **Plan**: `TASKS.md`
