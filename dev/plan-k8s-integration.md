# Podji Kubernetes Integration Remaining Plan

This plan contains only the remaining architectural work before full Kubernetes wiring.

## Non-Negotiable Constraint

Keep **mock mode as a first-class parallel path** even after real-cluster support lands.

Reason:

- local/offline development and demos need deterministic, rich datasets
- large realistic clusters are not always available for day-to-day iteration
- UI regression testing is faster and more stable in mock mode

## Remaining Work

## 1. Split Read Models from Stub Resource Implementations (In Progress)

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

- app command/query paths can resolve data through read-model abstractions
- mock adapter and kube adapter satisfy the same interface contract

## 2. Replace Kubectl Shelling with Client-Go Store

Scope:

- move kube discovery/query paths from `kubectl` command execution to `client-go`
- introduce shared informer caches and typed getters
- ensure view rendering reads from cache, not direct blocking API calls

Exit criteria:

- no runtime dependency on external `kubectl` binary for core data flows
- cache sync/readiness surfaced as explicit app state

## 3. Add Explicit Loading/Error/Permission States in App Flow

Scope:

- represent and render:
  - loading
  - forbidden
  - unreachable
  - partial data
- ensure scope switches (`N`/`X`) and command queries handle these states predictably

Exit criteria:

- no silent fallback behavior on cluster errors
- user always gets clear state feedback

## 4. Normalize Object Identity for Navigation and Relations

Scope:

- ensure domain items carry stable identity fields (`kind`, `apiVersion`, `namespace`, `name`, optional `uid`)
- use normalized identity keys for related lookups and stack restoration

Exit criteria:

- relation lookups are deterministic and not dependent on fragile display strings

## 5. Tighten Logs/Events Contracts for Streaming and Cancellation

Scope:

- add context-aware fetch APIs and bounded buffers
- support follow/tail semantics through interfaces (not view-local hacks)
- define clear fallback behavior for empty/forbidden/unavailable sources

Exit criteria:

- logs/events remain responsive under high volume
- cancellation and scope changes do not leak background work

## 6. Integration Test Matrix for Mode and Scope

Scope:

- add tests for:
  - `mock` vs `kube` mode startup
  - invalid mode fallback
  - context/namespace switching across modes
  - command query consistency (`unhealthy`, `restarts`) across adapters

Exit criteria:

- mode switching and scope behavior are contract-tested end-to-end

## 7. Keep Mock Dataset as a Productized Dev Tool

Scope:

- preserve deterministic mock scenarios for demos, UX work, and CI
- allow running mock and kube modes in parallel without code divergence
- document scenario toggles and intended usage

Exit criteria:

- mock mode remains feature-complete for navigation and debugging workflows
- kube mode can evolve without breaking mock reliability
