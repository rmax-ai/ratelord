# NEXT STEPS: Phase 4 - Implementation & Verification

The planning and specification phase is complete. The project is now in active implementation mode.

## Current Objective: Epic 1 - Foundation & Daemon Lifecycle

We are starting with the absolute basics: getting the Go module initialized, the directory structure created, and a process that can start and stop cleanly.

### Tasks for Next Session:
1.  **Initialize Go Module**: Run `go mod init` (or similar) to set up the project.
2.  **Create Directory Skeleton**: Create `cmd/ratelord-d`, `pkg/engine`, `pkg/store`, `pkg/api` as defined in M1.1.
3.  **Implement Entrypoint**: Write `cmd/ratelord-d/main.go` to print a startup message and handle `SIGINT`/`SIGTERM` (M1.2).
4.  **Verify**: Build and run the daemon; confirm it logs startup and exits cleanly on Ctrl+C.

## Reference
- **Plan**: `TASKS.md` (Epic 1)
- **Validation**: `TEST_STRATEGY.md` (Lifecycle E2E)
- **Architecture**: `ARCHITECTURE.md`
