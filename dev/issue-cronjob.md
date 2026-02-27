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

## Empty state for CronJob with no jobs should trigger auto-open

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

The CronJob-specific case (no jobs ever run) is not distinguished from the generic empty state.

> **Updated by redesign:** Phase 3 (`dev/plan-phase3.md`) makes empty drill-down auto-open the related side panel via a new `viewstate.OpenRelated` action from `forwardView()`. When the related panel opens automatically, a message is still useful for context. The fix below provides both.

### Fix

In `forwardView()` (`listview.go`), when the result set is empty, return `viewstate.OpenRelated` (Phase 3 adds this). In `EmptyMessage`, distinguish the CronJob-with-no-jobs case for when the user sees the empty view momentarily before the panel opens, or when filtering is active:

```go
func (w *WorkloadPods) EmptyMessage(filtered bool, filter string) string {
    if filtered {
        return "No pods match `" + filter + "`."
    }
    if w.workload.Kind == "CJ" && w.NewestJobName() == "—" {
        return "No jobs have run for CronJob `" + w.workload.Name + "` yet."
    }
    return "No pods found for workload `" + w.workload.Name + "`."
}
```

The "press r" hint is dropped — Phase 3 opens the panel automatically, making the hint redundant.
