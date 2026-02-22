# Podji

A fast, opinionated Kubernetes navigation TUI focused on debugging workflows.

Same scope as k9s for day-to-day exploration, but with instant navigation, calmer visuals, and detail/log flows optimized for finding problems quickly.

## Vision

Podji helps engineers move through Kubernetes the way they debug it:

- fast navigation
- predictable keybindings
- short paths to logs and relationships

It intentionally avoids modeling the full Kubernetes ontology and instead optimizes for practical troubleshooting.

## Non-Goals

Podji does not aim to:

- expose every Kubernetes resource as first-class UI
- replace `kubectl` for advanced operations
- provide a complete graph explorer
- model Kubernetes perfectly

Podji is a read-focused navigation and debugging tool.

## Core UX Principles

1. One obvious forward path per object.
2. Graph relationships exist, but are optional and consistent.
3. Most debugging ends in logs, so logs must be extremely fast.
4. Few concepts, repeated everywhere.
5. Arrow keys define structure and feel natural.
6. Color indicates status only (no decorative color).
7. Whitespace over borders and dim non-essential noise.

## Top-Level Navigation

Podji provides three task-oriented lenses. Press `Tab` to cycle.

- Apps: debug applications
- Network: debug traffic and services
- Infrastructure: debug nodes and scheduling

Namespace context persists across lens switches.

## Global Navigation Keys

- Right / Enter / `l`: drill down
- Left / Backspace / `h`: back
- `Tab`: switch lens
- `r`: related panel
- `/`: filter current list
- `l`: open logs from current context when meaningful

## Primary Drill-Down Model

Every object has one default next step.

- Namespace -> Workloads
- Workload -> Pods
- Pod (1 container) -> Logs
- Pod (>1 containers) -> Containers
- Container -> Logs
- Node -> Pods on node
- Service -> Backends
- Backends -> Pod
- Ingress -> Services
- ConfigMap / Secret -> Consumers
- PVC -> Mounted-by

Drill-down is predictable and reversible.

## Apps Lens

### Workloads (Landing Page)

Workloads is the default Apps landing page and must be fast to scan and deterministic to navigate.

#### Included in v1

- Deployment
- StatefulSet
- DaemonSet
- Job
- CronJob

ReplicaSets are not first-class in main navigation.

#### Unified List Layout

Single table for all workload kinds.

Columns:

1. NAME
2. KIND badge: DEP, STS, DS, JOB, CJ
3. READY
4. STATUS
5. RESTARTS
6. AGE

READY semantics:

- DEP/STS: ready/desired
- DS: ready/desiredScheduled
- JOB: succeeded/completions
- CJ: `Last: <age>` or `Last: -`

#### Default Sorting

Problem-first default ordering:

1. Failed
2. Degraded
3. Progressing
4. Healthy
5. Suspended

Secondary sort: name.

Optional toggle: `s` toggles `problem` <-> `name`.

### Workload Status Model

Status should be cheap to compute, deterministic, consistent across kinds, and debugging-friendly.

Vocabulary:

- Healthy
- Degraded
- Progressing
- Failed
- Suspended (CronJob only)

Pod sampling rule for hard failures:

- Inspect up to N newest owned pods (suggest N=20)
- Hard-failure reasons:
  - `CrashLoopBackOff`
  - `ImagePullBackOff`
  - `ErrImagePull`
  - `CreateContainerConfigError`

Rules:

Deployment

- Failed: condition indicates `ProgressDeadlineExceeded`
- Progressing: `observedGeneration < generation` OR `updatedReplicas < desired`
- Degraded: `available < desired` and not Progressing, OR hard-failure signals
- Healthy: `available == desired` and no hard-failure signals

StatefulSet

- Progressing: `updatedReplicas < desired` (or update in flight)
- Degraded: `ready < desired` and not Progressing, OR hard-failure signals
- Healthy: `ready == desired`

DaemonSet

- Progressing: `updatedNumberScheduled < desiredNumberScheduled`
- Degraded: `numberReady < desiredNumberScheduled` and not Progressing, OR hard-failure signals
- Healthy: `numberReady == desiredNumberScheduled`

Job

- Failed: Job condition Failed OR `failed > backoffLimit`
- Progressing: `active > 0` and not Failed
- Healthy: `succeeded >= completions`
- Degraded: anything not matching above

CronJob

- Suspended: `.spec.suspend == true`
- No Jobs yet: Healthy (`Last: -`)
- Newest Job Active: Progressing
- Newest Job Failed: Degraded
- Newest Job Succeeded: Healthy

### CronJob Drill-Down

To preserve a single obvious forward path:

- CronJob -> Pods of newest owned Job
- Newest Job by `.status.startTime`, fallback `creationTimestamp`
- If no Jobs exist: show empty pods list and hint to use Related (`r`) for Jobs/Events

Pods header should include context, for example: `CronJob pods (newest job: <job-name>)`.

### Workloads Footer Hints

Example:

`-> pods   l logs   r related   / filter   Tab view   s sort`

Show `s sort` only if sorting is implemented.

## Related Panel (`r`)

Related exposes curated graph relationships without becoming a full graph browser.

Order:

1. Owner (if present)
2. Events (promoted near top for workloads)
3. Relationship page (Backends / Consumers / Mounted-by)
4. Pods
5. Network (Services / Ingress)
6. Config (ConfigMaps / Secrets)
7. Storage (PVC / PV)

Rules:

- show counts when possible
- selecting an item navigates to that list
- drill-down remains consistent from there

## Relationship Pages

### Backends (Service)

Source of truth: EndpointSlices.

Shows:

- observed endpoints (ready / not ready)
- matched pods count (if selector exists)

If mismatch detected, show clear hint (pods not ready or port mismatch).

### Consumers (ConfigMap / Secret)

Reverse lookup references from Pods and Workloads:

- `env` / `envFrom`
- volumes / projected volumes

### Mounted-by (PVC)

Shows:

- pods mounting the PVC
- bound PV (if present)

## Relationship Resolution Rules

Practical over theoretical.

Ownership via ownerReferences:

- Deployment -> ReplicaSet -> Pod
- StatefulSet -> Pod
- DaemonSet -> Pod
- Job -> Pod
- CronJob -> Job -> Pod

Network:

- Service -> EndpointSlices -> Pod (`targetRef`)
- If Service has selector, also compute matched pod count
- Ingress -> referenced Services

Config references:

- `envFrom`
- `env.valueFrom`
- `volumes`
- projected volumes

Storage:

- PVC -> PV binding
- PVC -> Pods via volume mounts

Node scheduling:

- Node -> Pods via `spec.nodeName`

## Detail Views

Drill into a resource to see structured sections, not raw YAML.
Raw YAML remains available as an escape hatch via `y`.

Pod detail sections:

- Status line: phase, node, IP, QoS class
- Containers table: name, image (short), state, restarts, last restart reason
- Conditions: show only non-True conditions
- Recent events: last 10, reverse-chronological, warnings highlighted
- Labels/annotations: collapsed by default

## Logs View

Accessible via drill-down or `l`. Must be good enough to replace `kubectl logs`/`stern` for common debugging.

Defaults:

- follow on
- `--since=5m`
- line wrap on

Key actions:

- `f`: pause/resume (follow toggle)
- `space` / `pgdn`: page down
- `pgup`: page up
- `/`: search, with `n` / `N` navigation
- `w`: wrap toggle
- `t`: current/previous mode toggle
- `c`: container picker
- `[` / `]`: cycle `--since` window (1m, 5m, 15m, 1h, all)
- `Esc`: back

Goal: crashloop logs reachable in <= 4 steps.

## Namespace Model

- Default: single namespace
- Fast namespace switching
- Optional future mode: all namespaces

## Empty and Error States

### Workloads Page

- No workloads: show namespace-specific guidance
- Filter no matches: show filter-specific guidance (`Esc` to clear)
- RBAC forbidden: show denied message and required high-level permissions
- Partial RBAC: keep page usable, mark as partial, list hidden kinds
- Cluster unreachable/stale store: show stale mode and last successful sync time (if available)

### Workload -> Pods Page

- No pods found: show workload-specific hint to use `r` for Events/Config/Network

## Architecture (High Level)

Podji uses a thin in-memory cluster projection:

- client-go shared informers
- thin view models
- minimal eager indexes: owner -> children, service -> EndpointSlices

Other relationships are computed lazily. UI reads from Store only.

## v1 Scope

In:

- core resources (pods, deployments, services, configmaps, secrets, nodes, namespaces, events)
- full navigation model
- detail views
- log viewer
- context and namespace switching

Out:

- mutating actions (delete, cordon, restart)
- metrics/resource usage
- CRDs
- multi-cluster
- config persistence
- plugin system

## Success Criteria

Users can:

- reach crash logs in <= 4 steps
- diagnose "Service has no backends" in one view + one drill-down
- find ConfigMap consumers quickly
- navigate with arrows + `Tab` + `r` + `l`

## Future Extensions

- all-namespaces mode
- graph overlay
- NetworkPolicy lens
- RBAC lens
- bookmarks/pins
