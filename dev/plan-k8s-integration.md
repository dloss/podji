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

## Constraint (Still Active)

Mock mode remains a first-class parallel path for development, demos, and deterministic testing.

## Optional Follow-Ups (Not blockers for Kubernetes wiring)

- add explicit telemetry hooks for cache source decisions (informer vs direct API) if needed for debugging
- add stress/integration tests that simulate repeated rapid scope switching while log/event loads are active
- refine live describe/yaml formatting from metadata snapshots to richer typed object output as needed
