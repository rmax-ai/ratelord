# AGENTS: ratelord Working Rules

This repo is in a docs-first bootstrapping phase. Agents must optimize for conceptual consistency and auditability.

## Non-negotiables (from PROJECT_SEED.md)

- Orchestrator coordinates work; docs first; no implementation yet.
- Local-first, zero-ops: design for a single-machine daemon + local state.
- Daemon authority: all write authority lives in `ratelord-d`; clients are read-only.
- Event-sourced + replayable: the event log is the source of truth; snapshots are derived.
- Predictive, not reactive: forecast time-to-exhaustion (P50/P90/P99) and risk.
- Intent negotiation: agents submit intents before acting; daemon approves/denies/modifies.
- Shared vs isolated must be explicit: never assume isolation; model identity/scope/pools.
- Time-domain reasoning: reason in time-to-exhaustion / burn rate / reset windows, not raw counts.

## Repo operational rules

- Ignore (never read/modify): `bin/`, `node_modules/`, `package.json`, `bun.lock`.
- Small iterations: produce the smallest coherent delta; update tracking docs each iteration.
- Maintain tracking docs:
  - `TASKS.md`: hierarchical task list (what remains)
  - `PROGRESS.md`: current status table (what is in-flight)
  - `PHASE_LEDGER.md`: immutable-ish history (what completed/decided)
  - `NEXT_STEPS.md`: exact restart point for next session; must be read at session start

## Document-first workflow

- Start by reading: `PROJECT_SEED.md`, then `NEXT_STEPS.md`, then the document(s) you will touch.
- Work off a manifest: keep a clear list of required docs and current status in `PROGRESS.md`.
- Each doc change must preserve the system invariants (see Non-negotiables) and avoid scope drift.
- No code changes unless explicitly requested by the user or the phase has been advanced in writing.

## Document manifest (authoritative set)

Required order (see `PROJECT_SEED.md`):

1. `VISION.md`
2. `CONSTITUTION.md`
3. `ARCHITECTURE.md`
4. `CONSTRAINTS.md`
5. `IDENTITIES.md`
6. `DATA_MODEL.md`
7. `PREDICTION.md`
8. `POLICY_ENGINE.md`
9. `AGENT_CONTRACT.md`
10. `API_SPEC.md`
11. `TUI_SPEC.md`
12. `WEB_UI_SPEC.md`
13. `WORKFLOWS.md`
14. `ACCEPTANCE.md`

Optional:

- `PHASE_LEDGER.md`, `POSTMORTEM_TEMPLATE.md`, `EXTENSIONS.md`

## Review checklist (before marking “done”)

- Does this change reinforce: local-first, daemon authority, event sourcing, prediction, intent negotiation?
- Are identities, scopes, pools, and shared/isolated semantics explicit (no implicit assumptions)?
- Is time-domain reasoning present (reset windows, burn rate, uncertainty) rather than counters-only?
- Are definitions consistent across docs (same nouns, same boundaries, same component roles)?
- Are tradeoffs/unknowns recorded as decisions or open questions (and tracked in `TASKS.md`)?

## Sub-agent usage

- Orchestrator delegates drafting/refinement per document; sub-agents work strictly from `PROJECT_SEED.md` + current manifest.
- Sub-agents should propose: outline, key definitions, invariants, and open questions; avoid speculative implementation.
- Orchestrator performs final integration pass for consistency and updates tracking docs.

## Commit conventions (when commits are requested)

- Prefer small, focused commits; one intent per commit.
- Message style: `type(scope): intent` (imperative, why-focused). Examples:
  - `docs(seed): add agent working rules`
  - `docs(architecture): clarify daemon authority and event sourcing`
- Never commit secrets, tokens, or private logs; scrub samples.

## Security & privacy

- Treat identity material (PATs, OAuth tokens, GitHub App keys) as secrets; never write them to repo.
- Avoid copying third-party data into docs unless necessary; anonymize agent IDs and scopes in examples.
- Prefer local references over external links when capturing invariants/decisions.

## Tooling rules (local CLI)

- Search: use ripgrep.
  - `rg "Constraint graph" -n`
  - `rg "intent_approved|deny_with_reason" -n **/*.md`
- File discovery: prefer globbing patterns over ad-hoc scanning.
- Edits: make minimal, reviewable patches; preserve existing style and headings.
