# CLUSTER_FEDERATION: Distributed Rate Governance

**Status**: IMPLEMENTING
**Owner**: Orchestrator
**Related**: `ARCHITECTURE.md`

## 1. Overview

Currently, `ratelord` follows a "Same Machine Daemon" model. To support Kubernetes clusters or multi-machine agent fleets, we introduce **Cluster Federation**. This allows multiple daemons to share a global quota pool while maintaining local latency guarantees for most decisions.

## 2. Goals

1.  **Global Limits**: Enforce "10,000 RPM across 50 nodes" correctly.
2.  **Local Autonomy**: Don't require a network roundtrip for every `Intent` check (batched synchronization).
3.  **Partition Tolerance**: Degrade gracefully if the "Leader" is unreachable (fail-open or fail-safe modes).

## 3. Architecture Proposal: Leader-Follower

### 3.1 Topology
- **Leader**: One `ratelord-d` instance (or a Redis/Etcd backing store) holds the "Authoritative Token Bucket".
- **Follower**: Sidecar daemons run next to agents. They request "Grants" (blocks of quota) from the Leader.

### 3.2 The Grant Protocol
(See `API_SPEC.md` section 2.5 for the formal `POST /v1/federation/grant` specification)

Instead of checking every request:
1.  **Follower** boots -> Requests "Grant: 1000 tokens" from Leader.
2.  **Leader** deducts 1000 from Global Bucket -> Returns OK.
3.  **Follower** serves local agents from this cache.
4.  **Follower** runs low -> Requests refill asynchronously.

### 3.3 Complexity Risks
- **Over-allocation**: If 50 nodes ask for 1000 tokens, we might exhaust the global bucket instantly (Thundering Herd).
- **Drift**: If a Follower dies, its granted tokens are "lost" until TTL expires.

## 4. Storage Backend
- **Phase 1**: Leader talks to SQL/Redis.
- **Phase 2**: CRDT-based P2P sync (Advanced).
