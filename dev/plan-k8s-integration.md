# Podji Kubernetes Integration Plan

This plan defines how to move from a polished stub-driven UX to a real Kubernetes-connected TUI without losing interaction quality.

## Goal

Connect Podji to real clusters via `client-go` while preserving:

- fast navigation
- deterministic key behavior
- reversible stack-based flows
- readable, low-noise visuals

## Recommended Approach

Use a staged migration with a strict UI/data boundary first, then replace data sources incrementally behind that boundary.

Do **not** wire `client-go` directly into current resource mock structs.

## Step Count

Recommended: **8 steps**.

This keeps risk low, allows frequent testable checkpoints, and avoids a large rewrite.

## Step 1: Freeze UX Contract and Remove Demo Noise

Scope:

- preserve current keybindings and navigation semantics as a compatibility contract
- remove synthetic `zz-*` expansion from primary list and scope picker paths
- keep synthetic expansion only in explicit stress-test/dev mode

Why first:

- prevents fake data patterns from shaping production architecture
- gives a trustworthy baseline before backend changes

Exit criteria:

- normal mode has no `zz-*` rows in workloads/pods/namespaces/contexts pickers
- `dev/ui.sh start` capture remains stable and readable

## Step 2: Introduce an App-Level Scope Model (No Globals)

Scope:

- replace `resources.ActiveNamespace` global reads with explicit scope state
- define a scope object (context + namespace + all-namespaces flag)
- thread scope from app model into list/resource queries

Why:

- real multi-context/multi-namespace behavior cannot rely on package globals

Exit criteria:

- no resource behavior depends on mutable global namespace state
- copy actions and commands resolve namespace from selected item/scope, not globals

## Step 3: Create Data Provider Interfaces

Scope:

- add interfaces for list/detail/events/logs relations (read-only v1)
- keep UI views consuming stable domain DTOs (not kube API structs)
- support cancellation/timeouts in API shapes where needed (especially logs)

Suggested interface layers:

- `Store` / query layer for cached list/detail data
- `LogSource` for stream/snapshot log retrieval
- optional `ActionService` placeholder for future mutating actions

Exit criteria:

- `internal/ui/*` no longer depends on concrete mock resource constructors for data retrieval
- command queries (`unhealthy`, `restarts`, label selectors) run through shared provider interfaces

## Step 4: Build a Mock-Backed Store Adapter

Scope:

- adapt existing stub data into the new interfaces
- keep all current tests passing against this adapter
- maintain deterministic behavior for UI tests

Why:

- enables architecture refactor without requiring live cluster plumbing yet

Exit criteria:

- app runs unchanged visually against mock adapter
- integration tests can swap providers without UI changes

## Step 5: Add Kubernetes Store (Informer-Backed)

Scope:

- implement a real store using `client-go` shared informers and indexed caches
- include minimum required indexes:
  - owner -> children
  - service selector -> endpoints/pods
- expose readiness/sync state so UI can render loading/degraded states

Constraints:

- no direct API calls from view rendering path
- reads served from cache; background watches refresh cache

Exit criteria:

- list and detail screens work from informer cache with acceptable latency
- startup handles sync delay explicitly (loading banner/state)

## Step 6: Implement Context/Namespace Discovery and Switching

Scope:

- replace stub `ContextNames`/`NamespaceNames` with real kubeconfig + API discovery
- make `N`/`X` picker selections rebind data provider/store scope
- preserve current “stay on same resource type when switching scope” UX

Exit criteria:

- context picker reflects real kubeconfig contexts
- namespace picker reflects selected context permissions and availability
- switching context/namespace does not corrupt navigation stack

## Step 7: Logs and Events Realization

Scope:

- replace mock logs/events with real API-backed retrieval
- support tail/follow behavior with bounded buffers
- degrade gracefully for unavailable logs/events (RBAC, crashloop edge cases)

Exit criteria:

- logs view handles large streams without UI stalls
- event view handles empty/forbidden states clearly

## Step 8: Hardening, Testing, and Rollout Flags

Scope:

- expand tests:
  - contract tests for provider interfaces
  - app-level tests for scope switching and command-bar queries
  - smoke tests against a disposable cluster (kind)
- add feature flag or mode switch:
  - `--mode=mock|kube` (or env equivalent)
- add observability hooks for cache sync and API errors

Exit criteria:

- `go test ./...` passes
- mock mode remains stable for offline UI iteration
- kube mode is reliable enough for daily read-only debugging workflows

## Risks and Mitigations

1. Risk: scope bugs from mixed global and explicit state.
Mitigation: complete Step 2 before any informer work.

2. Risk: UI freezes from synchronous API calls.
Mitigation: cache-backed store and async updates only.

3. Risk: relation panel regressions due to mock fallbacks.
Mitigation: relation queries must use indexed cache and explicit “no relation found” states.

4. Risk: logs memory/CPU pressure.
Mitigation: bounded ring buffers, truncation policy, and incremental rendering.

## Suggested Delivery Slices

- Slice A (Steps 1-3): architecture prep, no live cluster dependency
- Slice B (Steps 4-5): mock adapter + informer-backed store
- Slice C (Steps 6-7): scope switching + logs/events
- Slice D (Step 8): hardening and release gating

## Definition of Done (v1 Kubernetes Connection)

- real context + namespace discovery
- read-only lists/details/relations/events/logs from cluster cache/APIs
- no global namespace mutable state
- command bar queries operate on shared store data
- mock mode still available for fast UX iteration
