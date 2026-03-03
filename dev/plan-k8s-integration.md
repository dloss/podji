# Podji Kubernetes Integration Plan

## Active Constraints

- Keep `mock` mode first-class for deterministic tests, demos, and offline development.
- Keep `kube` mode and `mock` mode behavior aligned through shared store/read-model contracts.

## Remaining Work

1. Feature-level live wiring
- Replace remaining mock-first data behavior in views with cluster-backed read model paths where not already wired.
- Keep graceful fallback to mock data only for explicit unsupported/error cases.

2. Live-update UX polish
- Ensure visible list/detail views refresh predictably from informer-backed cache changes without requiring extra key input.
- Validate status messaging for cache warming/ready transitions in real cluster scenarios.

3. End-to-end Kubernetes verification
- Add repeatable smoke flow using `dev/kube/` fixtures/scripts and assert core drilldowns (list -> detail -> logs/events/yaml/describe).
- Add at least one CI-friendly integration gate (optional/skippable locally).

4. Execution-path realism
- Decide and implement real-vs-simulated behavior for execute actions (`delete`, `restart`, `scale`, `port-forward`, shell exec).
- Keep explicit safety/confirmation and clear failure reporting.
