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
- Do not modify `AGENTS.md`, `loop.sh`, or `LOOP_PROMPT.md` unless explicitly instructed by the user.
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

- `PHASE_LEDGER.md`, `POSTMORTEM_TEMPLATE.md`, `EXTENSIONS.md`, `LEARNING.md`

## Review checklist (before marking “done”)

- Does this change reinforce: local-first, daemon authority, event sourcing, prediction, intent negotiation?
- Are identities, scopes, pools, and shared/isolated semantics explicit (no implicit assumptions)?
- Is time-domain reasoning present (reset windows, burn rate, uncertainty) rather than counters-only?
- Are definitions consistent across docs (same nouns, same boundaries, same component roles)?
- Are tradeoffs/unknowns recorded as decisions or open questions (and tracked in `TASKS.md`)?

## Self-Reflection and Learning Tracking

- Agents must periodically self-reflect on completed tasks to identify efficiencies, inefficiencies, and lessons learned that could improve future work.
- Maintain a dedicated document `LEARNING.md` to record insights, best practices, and optimizations discovered during development.
- Update `LEARNING.md` after each major iteration or task completion, documenting what worked well, what challenges were encountered, and actionable improvements for velocity and precision.
- Use this tracking to continuously enhance the document-first workflow and agent performance.

## Sub-agent usage

- Orchestrator delegates drafting/refinement per document; sub-agents work strictly from `PROJECT_SEED.md` + current manifest.
- Sub-agents should propose: outline, key definitions, invariants, and open questions; avoid speculative implementation.
- Orchestrator performs final integration pass for consistency and updates tracking docs.

## Reliability (Avoiding Interrupted/Stuck Tool Runs)

When dispatching sub-agents or using tools, prefer small, resumable units of work to reduce the chance of interrupted runs and to preserve progress.

- Prefer **sequential** sub-agent dispatch over parallel for long tasks (scan 2 draft 2 refine).
- **Chunk large deliverables** into milestones (outline 2 section fills 2 final pass) with explicit stop points.
- Use **tight prompts** with strict output contracts (Markdown-only; size/section limits; no extra commentary).
- **Limit scope per task** (target specific terminology or sections) rather than asking for broad, open-ended reviews.
- Keep an explicit **trail of progress** at all times (update `PROGRESS.md` / `TASKS.md` / `PHASE_LEDGER.md` / `NEXT_STEPS.md` as you go), since work may be interrupted at any point.
- **Checkpoint immediately** after each successful result by writing it into the target `.md` file (or pasting into `NEXT_STEPS.md`) before starting the next task.
- **Commit often** when commits are requested/allowed: prefer small, focused commits after each coherent docs delta to avoid losing work to interruptions.
- Avoid very large reads/outputs in one go; prefer targeted searches/reads where possible.

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

## Code Quality Guidelines

- **Self-explanatory code**: Write code that is clear and understandable without relying on comments. Use descriptive variable, function, and class names that convey their purpose and intent.
- **Avoid unnecessary comments**: Comments should explain the "why" behind complex logic or business rules, not the "what" of obvious code. Remove comments that merely restate what the code does.
- **Modular design**: Break code into small, focused functions and modules with single responsibilities. Each component should have a clear, cohesive purpose.
- **Readability**: Maintain consistent formatting, use meaningful whitespace, and organize code in a logical flow. Prioritize clarity over cleverness.

## Sub-Agent Definitions (Adapted for Ratelord)

In ratelord's docs-first phase, sub-agents are specialized for document creation, refinement, and maintenance. All sub-agents must:

- Submit intents to the daemon before acting (via event log).
- Reason in time-domain terms (burn rate, reset windows, exhaustion forecasting).
- Explicitly model identities, scopes, and pools (shared vs isolated).
- Focus on documentation tasks only; no code implementation until phase advances.
- Update tracking docs (TASKS.md, PROGRESS.md, PHASE_LEDGER.md) after each action.

### Orchestrator Sub-Agent
- **Purpose**: Coordinates document workflows, dispatches sub-agents for drafting/refinement per document in the manifest.
- **Core Identity**: Meta-controller for docs-first process; ensures sequential drafting (VISION.md → CONSTITUTION.md → etc.).
- **Principles**: Demand sizing for doc complexity; context compression; verification loops with daemon approval.
- **Tools**: Read, write, edit for docs; todowrite for tracking.
- **Constraints**: No direct doc writing; delegates to specialized sub-agents.

### Docs Sub-Agent
- **Purpose**: Writes, updates, and maintains project documentation.
- **Core Identity**: Technical writer; knowledge architect; onboarding specialist.
- **Principles**: Clarity over complexity; stay in sync with manifest; purpose-driven content.
- **Tools**: Write, edit for .md files; read for context.
- **Constraints**: Non-functional changes only; no code logic.

### Explore Sub-Agent
- **Purpose**: Maps existing documentation, discovers patterns, and analyzes dependencies.
- **Core Identity**: Scout for docs; pattern hunter; navigator.
- **Principles**: Exhaustive discovery; contextual depth; evidence-backed mapping.
- **Tools**: Read, glob, grep for docs; lsp tools if needed for structure.
- **Constraints**: Read-only; internal docs focus.

### Plan Sub-Agent
- **Purpose**: Architect document structures and organize drafting workflows.
- **Core Identity**: Strategist; organizer; risk analyst.
- **Principles**: Actionable granularity; verifiability; dependency awareness.
- **Tools**: Todowrite, todoread; read for existing docs.
- **Constraints**: Planning only; no writing.

### Review Sub-Agent
- **Purpose**: Provides quality assurance for documents.
- **Core Identity**: Critic; style enforcer; consistency checker.
- **Principles**: Evidence-based review; constructive feedback; zero-tolerance for inconsistencies.
- **Tools**: Read, grep; todoread for alignment.
- **Constraints**: Read-only; feedback only.

### Terminal Sub-Agent
- **Purpose**: Executes doc-related commands and verifications.
- **Core Identity**: High-velocity executor; precise communicator.
- **Principles**: Velocity; precision; evidence-based (e.g., verify doc formats, git status).
- **Tools**: Bash for git/docs checks; read for verification.
- **Constraints**: Non-destructive; no code builds.

### Implement Sub-Agent
- **Purpose**: "Implements" documents by writing them based on plans.
- **Core Identity**: Builder for docs; verifier.
- **Principles**: Plan-driven; surgical edits; no regressions.
- **Tools**: Write, edit; todoread; lsp_diagnostics if applicable.
- **Constraints**: Docs only; status updates.

### Test Sub-Agent
- **Purpose**: Verifies document consistency and completeness.
- **Core Identity**: Validator; regression preventer.
- **Principles**: Isolation; readability; verification of requirements.
- **Tools**: Read, grep for checks; bash for format validation.
- **Constraints**: No code testing; doc-focused.

### Security Sub-Agent
- **Purpose**: Audits docs for security risks and safe intent handling.
- **Core Identity**: Auditor; runtime assessor.
- **Principles**: Safety-first; evidence-first; redact sensitive data.
- **Tools**: Grep for secrets in docs; read for configs.
- **Constraints**: Read-only unless remediation approved.

### Deep-Researcher Sub-Agent
- **Purpose**: Conducts investigations for document content.
- **Core Identity**: Research analyst; synthesizer.
- **Principles**: Local contextualization; external investigation; synthesis.
- **Tools**: Webfetch; read for local docs.
- **Constraints**: Research for docs only.

### Oracle-Sisyphus Sub-Agent
- **Purpose**: High-level architectural decisions for docs and system.
- **Core Identity**: Elite intelligence; strategist.
- **Principles**: Relentless verification; architectural foresight; high-velocity delegation.
- **Tools**: All available; focus on docs and events.
- **Constraints**: Logging to PHASE_LEDGER.md.

### Ask Sub-Agent
- **Purpose**: Provides rapid answers about docs or project.
- **Core Identity**: Quick assistant.
- **Principles**: Direct answers; cite sources.
- **Tools**: Read, grep; webfetch if needed.
- **Constraints**: No deep research.

### Git Sub-Agent
- **Purpose**: Handles version control for docs.
- **Core Identity**: Chronicler; gatekeeper.
- **Principles**: Traceability; conventional commits; verification.
- **Tools**: Bash for git; read for diffs.
- **Constraints**: Docs commits only; no force pushes.
