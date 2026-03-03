# Podji Kubernetes Integration Remaining Plan

This plan now contains only unresolved architecture work before full Kubernetes wiring.

## Non-Negotiable Constraint

Keep mock mode as a first-class parallel path even after real-cluster support lands.

Reason:

- local/offline development and demos need deterministic, rich datasets
- realistic multi-workload clusters are not always available for day-to-day iteration
- UI regression testing is faster and more stable in mock mode

## Remaining Work

## 1. Explicit Data Freshness and Cache Readiness UX

Current state:

- store status already supports `loading`, `partial`, `forbidden`, `unreachable`, `degraded`, `ready`
- informer-backed list reads have direct-list fallback
- kube list reads now signal cache-backed vs direct-list paths; direct-list paths surface `loading` with a cache-warming message
- status transition from cache-warming (`loading`) to `ready` on cache-backed list reads is explicitly covered in tests
- app-level tests cover clearing temporary store loading banners once status returns to `ready`

Scope:

- surface cache/readiness metadata as explicit user-visible state when useful
- distinguish "live but warming cache" from hard errors
- ensure scope/context changes always converge from loading to a final visible state

Exit criteria:

- no ambiguous silent transitions during cache warm-up or scope changes
- users can tell whether they are seeing warm-cache or direct-list backed data

## 2. Streaming Lifecycle and Cancellation Hardening

Current state:

- read-model supports context-aware logs/events
- option-aware log/event reads are wired (tail/follow/limit)
- bounded buffering is in place for client-go log streaming
- follow-mode toggles in log view now refetch through option-aware readers with bounded context timeouts
- log view reloads run through cancellable commands, and app stack removal now calls view `Dispose()` hooks for cleanup
- cancellation and disposal behavior are covered by logview/app tests (in-flight reload cancellation + pop disposal)
- event view now uses cancellable async load in `Init`, and app push/replace flows execute pushed-view `Init` commands

Scope:

- introduce cancellable background fetch lifecycle for long-running log/event refresh flows
- guarantee cleanup on view pop and scope/context switches
- define follow-mode behavior against kube APIs without leaking goroutines or requests

Exit criteria:

- no leaked background work when navigation/scope changes interrupt active log/event flows
- follow-mode behavior is consistent and bounded for both mock and kube modes

## 3. Integration Contract Coverage for End-to-End Mode Flow

Current state:

- mode startup/fallback tests exist
- env-mode and scope/query contract tests exist in data layer
- app-level scope selection now has explicit loading-status synchronization coverage
- app command queries (`unhealthy`, `restarts`) now have status synchronization coverage

Scope:

- add end-to-end app-level tests that exercise mode + scope + status transitions together
- verify command-bar/query behavior remains consistent across mock and kube adapters

Exit criteria:

- adapter and app-flow contracts are test-covered from startup through scope switching and query paths
