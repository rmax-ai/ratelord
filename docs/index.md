# Ratelord

**The missing control plane for constrained execution.**

> "Constraints are not blockers; they are the contours of the solution space."

Ratelord is a local-first, zero-ops daemon designed to govern, model, and predict the consumption of hard constraints—API rate limits, token budgets, financial spend, and throughput capacity—for autonomous agents and human-driven systems.

---

## The Core Problem

Modern autonomous systems are **resource-blind**.

They treat API limits and budgets as external "weather"—unpredictable errors that happen to them—rather than internal constraints they must navigate. This leads to:

*   **Brittleness**: Systems crash or stall when quotas are exhausted.
*   **Inefficiency**: Agents under-utilize available resources out of fear of hitting limits.
*   **Lack of Coordination**: Multiple agents compete for the same shared limits without awareness of each other, leading to "tragedy of the commons" scenarios.

**Ratelord flips this model.** Instead of treating limits as errors, it treats them as **signals**. It provides agents with a "sensory organ" for resource availability and a "prefrontal cortex" for budget planning.

---

## Key Features

### 1. Daemon Authority
Ratelord runs as a **single-authority daemon** (`ratelord-d`). It is the source of truth for all constraint states. Agents cannot "check" limits themselves; they must **negotiate intent** with the daemon before taking action.

*   **Intent Negotiation**: Agents submit an intent (e.g., "I need to make 50 search requests"). The daemon approves, denies, or modifies this intent based on current system health.
*   **Centralized Governance**: All decisions flow through one brain, preventing race conditions and ensuring global safety.

### 2. Event Sourcing & Replayability
The system is built on an **immutable event log**. Every usage, refill, policy change, and intent decision is recorded as an event.

*   **Auditability**: Every millisecond of latency or unit of quota is traceable back to a specific agent, identity, and intent.
*   **Resilience**: The state is derived from the log. If the daemon restarts, it replays the log to restore the exact state of the world.

### 3. Predictive Forecasting
Ratelord is **predictive, not reactive**. It doesn't just wait for a limit to hit 0.

*   **Time-to-Exhaustion**: It calculates burn rates and forecasts when resources will run out (P50, P90, P99).
*   **Risk Modeling**: If a proposed action carries a high risk of exhausting a shared pool before the next reset window, Ratelord will deny it to preserve system stability.

### 4. Hierarchical Constraint Graph
Real-world limits are not flat counters. Ratelord models them as a **directed graph**:
*   **Identities**: API Keys, OAuth Tokens, User Accounts.
*   **Scopes**: Repositories, Organizations, Projects.
*   **Pools**: REST API, GraphQL, Search, Embedding Tokens.

A single action might consume from a local repo limit, an org-wide budget, and a global cost cap simultaneously. Ratelord manages this complexity automatically.

---

## Why Ratelord?

### For Developers
*   **Drop-in Protection**: Run `ratelord-d` locally to immediately protect your dev environment from hitting API caps during testing.
*   **Universal Interface**: Use a standard API to negotiate resources, whether it's GitHub API limits, OpenAI tokens, or AWS spend.

### For Agent Systems
*   **Adaptive Behavior**: Agents can shift strategies (e.g., switch from REST to GraphQL, or defer non-urgent work) based on Ratelord's feedback.
*   **Safety**: Prevent runaway loops from draining your wallet or getting your API keys banned.

### For Operations
*   **Attribution**: Know exactly *who* consumed *what*, *when*, and *why*.
*   **Policy Enforcement**: Define rules like "No non-critical background jobs if budget is < 10%" and enforce them system-wide.

---

## Getting Started

Check out the [Installation Guide](installation.md) or explore the [Architecture](concepts/architecture.md) to understand how Ratelord works under the hood.

### Guides

*   [Policy Guide](guides/policy.md): Learn how to write effective rate limit policies.
*   [Deployment Guide](guides/deployment.md): Best practices for deploying `ratelord-d`.
