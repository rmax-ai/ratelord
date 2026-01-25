# NEXT STEPS: Phase 4 - Implementation & Verification

The TUI Dashboard (Epic 8) is now functional, providing visibility into the system state. The core implementation phase is largely complete. The next steps involve stabilizing the TUI and verifying the full system behavior.

## Current Objective: System Stabilization & Final Verification

We have a working daemon, policy engine, forecast engine, and TUI. Now we need to ensure they work cohesively.

### Tasks for Next Session:
1.  **M9.1: TUI Drill-Down Views**
    - Implement detail views for Events and Policy rules in the TUI.
    - Allow inspecting JSON payloads of events.
2.  **M9.2: Error Handling & Reconnection**
    - Ensure TUI recovers if the Daemon is restarted (critical for dev loop).
3.  **M9.3: Configurable Policy Loading**
    - Allow defining `policy.yaml` to control limits, rather than hardcoded defaults.

## Reference
- **Plan**: `TASKS.md`
- **Design**: `TUI_SPEC.md`
