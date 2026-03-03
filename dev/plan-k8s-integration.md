# Podji Kubernetes Integration Remaining Plan

This plan contains only the remaining architectural work before full Kubernetes wiring.

## Non-Negotiable Constraint

Keep **mock mode as a first-class parallel path** even after real-cluster support lands.

Reason:

- local/offline development and demos need deterministic, rich datasets
- large realistic clusters are not always available for day-to-day iteration
- UI regression testing is faster and more stable in mock mode

## Remaining Work

## 1. Complete Read-Model Split for Non-Stub Data Paths (Partially Done)

Done now:

- read-model contract exists for list/detail/logs/events/describe/yaml
- app list/query/navigation paths are routed through `Store.AdaptResource(...)`
- `KubeReadModel` now routes pod logs/events through `KubeAPI` instead of stub-only paths
- related picker consumes `RelationIndex` for category counts and indexed related list opening paths

Scope:

- introduce explicit read-model interfaces for:
  - list
  - detail
  - logs
  - events
  - describe
  - yaml
- route app-side query/navigation logic through these interfaces, not direct stub structs
- keep `resources/*` as mock dataset providers behind adapters, not as app-facing query APIs

Exit criteria:

- mock adapter and kube adapter satisfy the same interface contract end-to-end

## 2. Replace Kubectl Shelling with Client-Go Store (In Progress)

Done now:

- `NewKubeStore()` initializes a `client-go` based `KubeAPI` implementation
- contexts, namespaces, pod logs, and pod events are served via client-go calls
- namespace lookups have a short TTL cache to reduce repeated API calls during fast UI actions
- read-model list calls for `pods`, `services`, `deployments`, and `workloads` now use client-go data with short TTL caching

Scope:

- move kube discovery/query paths from `kubectl` command execution to `client-go`
- introduce shared informer caches and typed getters
- ensure view rendering reads from cache, not direct blocking API calls

Exit criteria:

- no runtime dependency on external `kubectl` binary for core data flows
- cache sync/readiness surfaced as explicit app state

## 3. Add Explicit Loading/Error/Permission States in App Flow (Partially Done)

Done now:

- `StoreStatus` expanded with `loading`, `partial`, `forbidden`, `unreachable`, `degraded`
- kube error classification maps discovery/log/event failures to explicit states
- app renders state-qualified store message (`store (<state>): ...`)
- kube read-model marks `partial` when list data falls back to mock due unsupported live list paths

Scope:

- represent and render:
  - loading
  - forbidden
  - unreachable
  - partial data
- ensure scope switches (`N`/`X`) and command queries handle these states predictably

Exit criteria:

- no silent fallback behavior on cluster errors in main data flows
- user always gets clear state feedback for startup/scope/discovery failures

## 4. Normalize Object Identity for Navigation and Relations (Done)

Scope:

- ensure domain items carry stable identity fields (`kind`, `apiVersion`, `namespace`, `name`, optional `uid`)
- use normalized identity keys for related lookups and stack restoration

Exit criteria:

- relation lookups are deterministic and not dependent on fragile display strings

## 5. Tighten Logs/Events Contracts for Streaming and Cancellation (Partially Done)

Done now:

- pod logs/events are centralized through `KubeReadModel` instead of view-local fetcher wiring
- kube read errors are surfaced via store status path
- read-model contracts now include context-aware logs/events hooks with backward-compatible fallback

Scope:

- add context-aware fetch APIs and bounded buffers
- support follow/tail semantics through interfaces (not view-local hacks)
- define clear fallback behavior for empty/forbidden/unavailable sources

Exit criteria:

- context-aware streaming APIs exist in read-model contracts
- cancellation and scope changes do not leak background work

## 6. Integration Test Matrix for Mode and Scope (Partially Done)

Done now:

- mode startup/fallback tests exist
- scope switch tests exist across mock/kube adapters
- contract tests validate `unhealthy` and `restarts` query consistency across adapters

Scope:

- add tests for:
  - `mock` vs `kube` mode startup
  - invalid mode fallback
  - context/namespace switching across modes
  - command query consistency (`unhealthy`, `restarts`) across adapters

Exit criteria:

- mode switching and scope behavior are contract-tested end-to-end

## 7. Keep Mock Dataset as a Productized Dev Tool (Done/Ongoing)

Scope:

- preserve deterministic mock scenarios for demos, UX work, and CI
- allow running mock and kube modes in parallel without code divergence
- document scenario toggles and intended usage

Exit criteria:

- mock mode remains feature-complete for navigation and debugging workflows
- kube mode can evolve without breaking mock reliability
