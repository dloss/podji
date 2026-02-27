# Issue: CronJob UX Gaps

## Breadcrumb doesn't show the newest job name

**Spec says** (`dev/CONCEPT.md`):
> "Pods header should include context, for example: `CronJob pods (newest job: <job-name>)`."

**Implementation (`internal/resources/relations.go:192-193`):**

```go
func (w *WorkloadPods) Name() string {
    return "pods (" + w.workload.Name + ")"
}
```

This produces `pods (nightly-backup)` regardless of kind. For CronJobs the spec wants the nearest job surfaced so the user knows which job's pods they're looking at.

The `NewestJobName()` method already exists (`relations.go:327`) but is never called from `Name()`.

### Fix

Specialise `Name()` for CronJobs:

```go
func (w *WorkloadPods) Name() string {
    if w.workload.Kind == "CJ" {
        job := w.NewestJobName()
        return "pods (CronJob: " + w.workload.Name + ", newest job: " + job + ")"
    }
    return "pods (" + w.workload.Name + ")"
}
```

This propagates into the breadcrumb automatically since the app renders `resource.Name()` there.

## Empty state for CronJob with no jobs doesn't hint at `r`

**Spec says** (`dev/CONCEPT.md`):
> "If no Jobs exist: show empty pods list and hint to use Related (`r`) for Jobs/Events"

**Implementation (`relations.go:319-325`):**

```go
func (w *WorkloadPods) EmptyMessage(filtered bool, filter string) string {
    if filtered {
        return "No pods match `" + filter + "`."
    }
    return "No pods found for workload `" + w.workload.Name + "`."
}
```

The CronJob-specific case (no jobs ever run) is not distinguished from the generic empty state. A user landing here has no idea where to go next.

### Fix

Detect the CronJob-with-no-jobs case and add a hint:

```go
func (w *WorkloadPods) EmptyMessage(filtered bool, filter string) string {
    if filtered {
        return "No pods match `" + filter + "`."
    }
    if w.workload.Kind == "CJ" && w.NewestJobName() == "—" {
        return "No jobs have run for CronJob `" + w.workload.Name + "` yet.\n" +
            "Press r to view Related (Jobs, Events, Config)."
    }
    return "No pods found for workload `" + w.workload.Name + "`.\n" +
        "Press r to view Related (Events, Config, Network)."
}
```

The second paragraph ("Press r…") is also useful for non-CronJob workloads that have no pods, so it's worth adding generally.
