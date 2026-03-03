# Podji Kubernetes Integration Plan Status

This file now reflects completion status after the pre-wiring architecture pass.

## Completed

1. Data freshness + cache readiness UX
- explicit store states (`loading`, `partial`, `forbidden`, `unreachable`, `degraded`, `ready`)
- cache-warming vs cache-ready status messaging (`warming cache for <resource>`, `cache ready for <resource>`)
- render-time store-status sync so background transitions are visible without extra key input
- scope-switch loading -> ready freshness transition covered with app rendering tests

2. Streaming lifecycle + cancellation hardening
- option-aware logs/events contracts (`tail`, `follow`, `limit`) wired through read-backed resources
- bounded client-go log buffering
- async cancellable log/event loads with request correlation
- disposable view lifecycle hook and app-level disposal on pop/replace/scope/resource stack resets
- test coverage for in-flight cancellation and disposal-triggered cleanup

3. Integration contract coverage for mode/scope/query flows
- app startup mode coverage via injectable store factory path (warning/no-warning + scope seeding)
- scope selection status sync coverage
- query status sync coverage (`unhealthy`, `restarts`)
- query navigation consistency across mock and kube-like store adapters
- stack lifecycle/disposal behavior covered across navigation/scope transitions

4. Initial typed Kubernetes object wiring
- `KubeReadModel` now prefers typed kube API object reads for YAML/describe when available
- `clientGoAPI` now serves typed object YAML/describe for live resources (`pods`, `services`, `deployments`, workload kinds, `ingresses`, `configmaps`, `secrets`, `persistentvolumeclaims`, `nodes`, `namespaces`, `events`)
- `KubeReadModel` now also prefers typed kube API object reads for detail when available (with safe fallback)
- typed detail mapping now includes richer workload and ingress summaries beyond deployment-only coverage

5. Related-resource index hardening
- relation lookups now use a scope-scoped snapshot cache over read-model list data, reducing repeated list fanout during related-panel navigation
- owner resolution now prefers controller UID when available (stable identity), with kind-aware fallback matching
- explicit tests now cover per-scope caching and TTL-based snapshot invalidation

6. Native Kubernetes API path cleanup
- removed legacy shell-based `kubectl` implementation from the data-layer API contract file
- kube mode data access path is now fully `client-go` based for store/read operations

## Constraints (Still Active)

Mock mode remains a first-class parallel path for development, demos, and deterministic testing.

## Remaining Before Full Kubernetes Wiring

No architecture-prep blockers remain. Next work can focus on direct feature wiring against real cluster data paths.
