### Workloads screen example (ASCII mock)

Below is a concrete, scan-friendly mock that reflects:

* **Unified Workloads list**
* **problem-first sort**
* **CronJob uses `Last:`**
* **status vocabulary**
* **footer key hints**

```
Workloads                                  ns: payments     (23)     filter: —
────────────────────────────────────────────────────────────────────────────────
NAME                         KIND  READY      STATUS        RESTARTS   AGE
api                           DEP   2/3        Degraded      14         3d
worker                        DEP   1/1        Healthy       0          12d
db                            STS   2/3        Progressing   0          6h
node-exporter                 DS    5/6        Degraded      0          30d
seed-users                    JOB   0/1        Failed        3          18m
nightly-backup                CJ    Last: 6h   Healthy       —          90d
sync-reports                  CJ    Last: —    Healthy       —          2d
cleanup-tmp                   CJ    Last: 22m  Degraded      —          15d
────────────────────────────────────────────────────────────────────────────────
→ pods   l logs   r related   / filter   s sort   ← back
```

#### Notes embodied by the mock

* **Problem-first sort** puts `Failed/Degraded/Progressing` above `Healthy/Suspended`.
* **CronJobs** show `Last: <age>` (or `Last: —` if never run). `RESTARTS` is `—` for CJ rows since restarts belong to pods/jobs.
* **READY** varies by kind but stays predictable:

  * DEP/STS show `ready/desired`
  * DS shows `ready/desiredScheduled`
  * JOB shows `succeeded/completions`
  * CJ uses `Last:` instead

---

### Drill-down example: CronJob `→` Pods (newest Job)

When selecting `nightly-backup (CJ)` and pressing `→`:

```
Pods (CronJob: nightly-backup)            newest job: nightly-backup-289173
ns: payments     (1)     filter: —
────────────────────────────────────────────────────────────────────────────────
NAME                                    READY   STATUS      RESTARTS   AGE
nightly-backup-289173-7m2kq             1/1     Running     0          2m
────────────────────────────────────────────────────────────────────────────────
→ logs   l logs   r related   / filter   ← back
```

If the CronJob has **no Jobs yet** — Phase 3 auto-opens the related panel:

```
Pods (CronJob: sync-reports)             newest job: —
ns: payments     (0)     filter: —        ┌─ Related ──────────────────────────┐
────────────────────────────────────────  │  Jobs                           0  │
No jobs have run for CronJob              │> Events                         2  │
"sync-reports" yet.                       │  Config                         1  │
                                          │  Network                        1  │
                                          │                                    │
                                          │  Tab main   Esc close              │
──────────────────────────────────────    └────────────────────────────────────┘
/ filter   ← back
```

---

### Drill-down example: Workload `l` logs (fast path)

From `api (DEP)`, press `l`:

* Pod picker opens (default newest highlighted)
* Enter selects → logs view

```
Logs › api-7d9c7c9d4f-qwz8p / app         ns: payments   mode: current   follow: on
────────────────────────────────────────────────────────────────────────────────
...log stream...
────────────────────────────────────────────────────────────────────────────────
t toggle current/previous   f follow   ← back
```

(Exact log controls can vary; the key is the *path length and determinism*.)

---

If you’d like, next we can add **one more mock** for the Workloads page with an active filter (`/ api`) and a **partial RBAC** banner so the error/empty states feel equally concrete.
