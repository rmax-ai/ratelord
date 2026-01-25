# LOOP PROMPT: Ratelord Orchestrator

Maintain the **implementation and maintenance** process for **ratelord** following the rules in `AGENTS.md`.

## Execution Protocol
- **Single Task Focus**: Each iteration (one execution) must pick the **highest leverage task** from `NEXT_STEPS.md` or `TASKS.md` and complete it. **Do not multitask.**
- **Frequent Progress Tracking**:
    - Update `PROGRESS.md`, `TASKS.md`, `PHASE_LEDGER.md`, and `NEXT_STEPS.md` immediately as sub-steps are completed.
    - **Commit often**: Use the terminal to commit small, coherent changes to track progress.
- **Mandatory Final Action**: At the end of EVERY iteration, before signaling completion, you MUST use the terminal agent to commit all tracked and untracked changes (excluding ignored files) with a descriptive commit message following the conventions in `AGENTS.md`. Maintain `.gitignore` to ensure unwanted files (logs, binaries, local state) are never committed.
- **Resilience & Resumability**:
    - Assume work may be halted at any point.
    - Every iteration should start by reading `NEXT_STEPS.md` to pick up exactly where the last one left off.

## Context & Constraints
- **Role**: Orchestrator (Coordinate sub-agents, ensure consistency, enforce architecture).
- **Core Principles**: Local-first, Daemon Authority, Event-sourced, Predictive, Intent Negotiation (See `PROJECT_CONTEXT.md`).
- **Rules**:
    - Always read `NEXT_STEPS.md` first.
    - Document decisions before implementation.
    - Ensure code changes are verified by tests (`pkg/`) or acceptance tools (`ratelord-sim`).

## Signaling Completion
- If you have completed the current high-leverage task and there are **pending tasks** remaining, output `<promise>NEXT_TASK</promise>`.
- If **all tasks** in the current scope/phase are done and no more work remains, output `<promise>DONE</promise>`.

## Sub-Agent Usage
- Delegate drafting, implementation, or verification tasks to sub-agents.
- Ensure sub-agents are briefed with `PROJECT_CONTEXT.md` and the specific relevant specs.
- Perform a consistency check on all sub-agent output before finalizing.
