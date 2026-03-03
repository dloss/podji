# ADR: Data Provision Architecture (Mock + Cluster)

## Status

Accepted (incremental rollout).

## Context

Podji must support:

- low-latency UI reads
- live updates while browsing
- limited cluster/API load
- deterministic mock mode for demos/testing

Related-resource lookups are high-risk for accidental N+1 cluster queries if handled naively.

## Decision

Adopt a layered, cache-first data architecture:

1. `DataSource` (cluster-facing)
- Watches/informers for cluster-backed mode (event-driven, not polling).
- Produces normalized objects into in-memory caches.

2. `Store` (app-facing)
- Owns scope (`context`, `namespace`), cache snapshot access, and query surfaces.
- UI reads from Store contracts only.
- Supports both `mock` and `kube` modes behind identical interfaces.

3. `ReadModel`
- Explicit read operations for list/detail/logs/events/describe/yaml.
- Mock and kube backends each provide adapters to this contract.

4. `RelationIndex`
- Dedicated related-resource lookup interface.
- Must be index/cache-based only; no network calls during lookup.
- Relation index is updated from source events (or deterministic mock data).

## Related-Resource Constraints

- Opening related panel must be local lookup only.
- Relation keys must use stable identity (eventually UID+GVK), not display labels.
- Expensive/deep relations should be asynchronous and explicitly marked as loading.
- Result fan-out must be capped/paginated to avoid UI and memory spikes.

## Why

- Latency: cache lookups are fast and predictable.
- Live updates: informer events can update index incrementally.
- Cluster safety: avoids repetitive list/get calls on each navigation event.
- Reliability: mock mode remains a first-class path for CI, demos, and offline work.

## Consequences

Positive:

- Clear separation between UI and cluster I/O.
- Easier performance tuning and error handling.
- Mock and kube modes stay feature-aligned.

Tradeoffs:

- More internal interfaces and adapters.
- Requires explicit index maintenance logic.
- Additional test surface (contract tests across adapters).

## Remaining Implications

- New features should be added via `Store` + `ReadModel` contracts first, then implemented for both `mock` and `kube` backends.
- Related-resource lookups must stay local/indexed; no network calls on picker open.
- Any new high-fanout query should define explicit cache/use limits and be covered by adapter contract tests.
