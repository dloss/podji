Here’s a **single, cohesive vNext spec** with the Workloads-screen enhancements folded in (Unified list, deterministic status, CronJob drill-down, empty/error states, sorting, and related-panel tweak).

---

## Podji

A fast, opinionated Kubernetes navigation TUI

---

## Vision

Podji is a terminal UI that helps engineers move through Kubernetes the way they debug it.

It prioritizes:

* fast navigation
* predictable keybindings
* short paths to logs and relationships

Podji intentionally avoids modeling the full Kubernetes ontology. Instead, it provides a practical navigation model optimized for real debugging workflows.

---

## Non-Goals

Podji does not aim to:

* expose every Kubernetes resource as first-class UI
* replace kubectl for advanced operations
* provide a complete graph explorer
* model Kubernetes perfectly

Podji is a read-focused navigation and debugging tool.

---

## Core UX Principles

1. One obvious forward path per object.
2. Graph relationships exist, but are optional and consistent.
3. Most debugging ends in logs → make logs extremely fast.
4. Few concepts, repeated everywhere.
5. Arrow keys define structure and feel natural.

---

## Navigation Overview

### Top-Level Views (Lenses)

Podji provides three task-oriented lenses.

Press **Tab** to switch views.

| View           | Purpose                  |
| -------------- | ------------------------ |
| Apps           | Debug applications       |
| Network        | Debug traffic & services |
| Infrastructure | Debug nodes & scheduling |

Namespace context persists across view switches.

### Default Landing Pages

| View           | Landing Page |
| -------------- | ------------ |
| Apps           | Workloads    |
| Network        | Services     |
| Infrastructure | Nodes        |

Namespaces remain visible and easily changeable but are not the default landing page.

---

## Universal Navigation Keys

These work everywhere.

| Key           | Action                         |
| ------------- | ------------------------------ |
| → Right Arrow | Drill down                     |
| ← Left Arrow  | Go up (back)                   |
| Tab           | Switch view                    |
| r             | Related panel                  |
| /             | Filter list                    |
| l             | Open logs from current context |

Right/Left arrows define structural navigation.

---

## Logs Shortcut (Global Fast Path)

**l** opens the most relevant logs from the current context.

| Context             | Behavior                           |
| ------------------- | ---------------------------------- |
| Container           | Open logs                          |
| Pod (1 container)   | Open logs immediately              |
| Pod (>1 containers) | Container picker → logs            |
| Workload            | Pod picker (default newest) → logs |
| Other objects       | No action                          |

This shortcut never replaces drill-down navigation.

---

## Primary Drill-Down Model

Every object has one default next step.

| Object              | Drill-Down Target |
| ------------------- | ----------------- |
| Namespace           | Workloads         |
| Workload            | Pods              |
| Pod (1 container)   | Logs              |
| Pod (>1 containers) | Containers        |
| Container           | Logs              |
| Node                | Pods on node      |
| Service             | Backends          |
| Backends            | Pod               |
| Ingress             | Services          |
| ConfigMap / Secret  | Consumers         |
| PVC                 | Mounted-by        |

Drill-down must always be predictable and reversible.

---

## Apps Lens

### Workloads (Landing Page)

Workloads is the default Apps landing page and must be fast to scan and deterministic to navigate.

#### Workloads Included (v1)

* Deployment
* StatefulSet
* DaemonSet
* Job
* CronJob

ReplicaSets are not first-class in main navigation.

#### Workloads List Layout (Unified)

One table containing all workload kinds.

Columns (left → right):

1. **NAME**
2. **KIND** badge: `DEP`, `STS`, `DS`, `JOB`, `CJ`
3. **READY**

   * DEP/STS: `ready/desired`
   * DS: `ready/desiredScheduled`
   * JOB: `succeeded/completions` (default completions=1)
   * CJ: `Last: <age>` or `Last: —` (replaces READY)
4. **STATUS** (fixed vocabulary)
5. **RESTARTS** (sum across owned pods; default total; “last N minutes” optional later)
6. **AGE**

Optional later: **PODS** column (count), if it doesn’t add clutter.

#### Default Sorting

Default sort is problem-first to support debugging:

1. Failed
2. Degraded
3. Progressing
4. Healthy
5. Suspended

Secondary sort: Name.

Optional toggle:

* `s` toggles sort mode: `problem` ↔ `name`

#### Workload Status Model

Status must be:

* cheap to compute
* deterministic
* consistent across kinds
* useful for debugging

Status vocabulary:

* **Healthy**
* **Degraded**
* **Progressing**
* **Failed**
* **Suspended** (CronJob only)

Pod sampling rule (for hard-failure signals):

* Podji may inspect up to **N newest owned pods** (suggest N=20)
* hard-failure reasons include:

  * `CrashLoopBackOff`
  * `ImagePullBackOff`
  * `ErrImagePull`
  * `CreateContainerConfigError`

Status rules:

**Deployment**

* Failed: condition indicates `ProgressDeadlineExceeded`
* Progressing: `observedGeneration < generation` OR `updatedReplicas < desired`
* Degraded: `available < desired` AND not Progressing, OR hard-failure signals in sampled pods
* Healthy: `available == desired` and no hard-failure signals

**StatefulSet**

* Progressing: `updatedReplicas < desired` (or update in flight)
* Degraded: `ready < desired` AND not Progressing, OR hard-failure signals
* Healthy: `ready == desired`

**DaemonSet**

* Progressing: `updatedNumberScheduled < desiredNumberScheduled`
* Degraded: `numberReady < desiredNumberScheduled` AND not Progressing, OR hard-failure signals
* Healthy: `numberReady == desiredNumberScheduled`

**Job**

* Failed: Job condition Failed OR `failed > backoffLimit`
* Progressing: `active > 0` and not Failed
* Healthy: `succeeded >= completions`
* Degraded: not Healthy/Progressing/Failed (e.g. stuck Pending)

**CronJob**

* Suspended: `.spec.suspend == true`
* If no Jobs exist yet: Healthy (shows `Last: —`)
* Newest Job Active: Progressing
* Newest Job Failed: Degraded
* Newest Job Succeeded: Healthy

#### CronJob Drill-Down (Deterministic)

To preserve “one obvious forward path”:

* **CronJob → Pods of newest owned Job**

  * newest determined by Job `.status.startTime` if present, else creationTimestamp
* If CronJob has no Jobs:

  * show empty pods list + hint to use Related (`r`) to see Jobs/Events

Pods page header should indicate context:

* `CronJob pods (newest job: <job-name>)`

#### Minimal Footer Hints (Workloads)

Example footer:

* `→ pods   l logs   r related   / filter   Tab view   s sort`

(Only show `s sort` if implemented.)

---

## Related Panel (r)

The Related panel exposes Kubernetes as a graph without making the UI a graph browser.

Entries are curated and ordered:

1. Owner (if present)
2. Relationship page (Backends / Consumers / Mounted-by)
3. Pods
4. Network (Services / Ingress)
5. Config (ConfigMaps / Secrets)
6. Storage (PVC / PV)
7. Events

Workload context tweak (debug-first):

* For workloads, **Events should be promoted near the top** (immediately after Owner where present), since events are a common next step after logs/pods.

Rules:

* show counts when possible
* selecting an item navigates to that list
* drill-down continues normally from there

---

## Pages

### List Pages

* One per resource kind
* Filterable via `/`
* Namespace context visible

### Detail Page (Optional)

Shows summary, status, and key metadata.

---

## Relationship Pages

### Backends (Service)

Source of truth: EndpointSlices.

Shows:

* observed endpoints (ready / not ready)
* matched pods count (if selector exists)

If mismatch detected:

* “Pods not Ready or port mismatch”

Selecting a backend drills to Pod.

### Consumers (ConfigMap / Secret)

Reverse lookup of references from:

* Pods
* Workloads

Reference types:

* env / envFrom
* volumes / projected

### Mounted-by (PVC)

Shows:

* pods mounting the PVC
* bound PV (if present)

---

## Relationship Resolution Rules

Practical over theoretical.

### Ownership

Use ownerReferences:

* Deployment → ReplicaSet → Pod
* StatefulSet → Pod
* DaemonSet → Pod
* Job → Pod
* CronJob → Job → Pod

ReplicaSets are not first-class in main navigation.

### Network

Service → EndpointSlices → Pod (targetRef)

If Service has selector:

* also compute matched pod count

Ingress → referenced Services.

### Config

Referenced via:

* envFrom
* env.valueFrom
* volumes
* projected volumes

### Storage

PVC → PV binding
PVC → Pods via volume mounts

### Node Scheduling

Node → Pods via spec.nodeName.

---

## Logs View

Accessible via drill-down or **l**.

Must provide:

* toggle: current / previous
* follow mode

Goal: crashloop logs reachable in ≤4 steps.

---

## Namespace Model

Default mode: single namespace.

User can switch namespaces quickly.

“All namespaces” mode may be added later.

---

## Empty and Error States

Podji should render informative states instead of failing.

### Workloads Page

* **No workloads**

  * “No workloads found in namespace `<ns>`.”
  * Hint: switch namespace or clear filter
* **Filter yields no results**

  * “No workloads match `<filter>`.”
  * Hint: Esc to clear
* **RBAC forbidden**

  * “Access denied: cannot list workloads in namespace `<ns>`.”
  * Provide a high-level note about required get/list permissions on workload kinds
* **Partial RBAC**

  * Page still works with available kinds
  * Header indicates “partial”
  * Message lists hidden kinds (e.g. Jobs/CronJobs)
* **Cluster unreachable / stale store**

  * “Cluster unreachable”
  * Show last successful sync time if available
  * UI continues in read-only “stale” mode until reconnection

### Workload → Pods Page

* **No pods found**

  * “No pods found for workload `<name>`.”
  * Hint: `r` to view Events / Config / Network relationships

---

## Architecture (High Level)

Podji maintains a thin in-memory projection of the cluster.

Uses:

* client-go shared informers
* thin view models (not full objects)
* minimal eager indexes:

  * owner → children
  * service → endpointSlices

Other relationships computed lazily.

UI reads only from the Store.

---

## Success Criteria

Podji succeeds if users can:

* reach crash logs in ≤4 steps
* diagnose “Service has no backends” in one view + one drill-down
* find ConfigMap consumers quickly
* navigate using only arrows + Tab + r + l

---

## Future Extensions (Optional)

* all-namespaces mode
* graph overlay
* NetworkPolicy lens
* RBAC lens
* bookmarks / pins

---


