# Issue: Navigation Key Inconsistencies

## `l` vs `o` for logs

**Spec says:** `l` opens logs from the current context when meaningful.

**Implementation (`listview.go:376, 1097`):**

```go
case "enter", "l", "right", "o":
    // ...
    if next := v.forwardView(selected.data, key.String()); next != nil {
```

```go
// forwardView, line 1096-1104:
if resourceName == "workloads" {
    if key == "o" {
        // navigate directly to logs
        return logview.New(...)
    }
    return New(resources.NewWorkloadPods(selected), v.registry) // enter/l/right all go to pods
}
```

So on the workloads list:
- `enter` / `l` / `right` → pods list
- `o` → logs directly

On the pods list:
- `enter` / `l` / `right` → containers (if >1) or logs

The mnemonic `l` means "logs" in the spec but behaves like "enter" on workloads. Users who expect `l` to open logs from a workload will instead land on the pods list. `o` is undocumented and undiscoverable.

### Fix

Remap `l` to go directly to logs on the workloads list (matching `o`'s current behavior), and drop `o` or keep it as an alias:

```go
case "enter", "right":
    // standard drill-down (workloads → pods, pods → logs/containers, etc.)
case "l":
    // logs shortcut: skip pods list, go straight to logs
    if next := v.forwardView(selected.data, "l"); next != nil { ... }
```

Update `forwardView`:

```go
if resourceName == "workloads" {
    if key == "l" || key == "o" {
        // direct to logs
    }
    return New(resources.NewWorkloadPods(selected), v.registry)
}
if strings.HasPrefix(resourceName, "pods") {
    if key == "l" {
        return logview.New(selected, v.resource)  // skip container picker
    }
    // enter/right: container picker if >1 containers
}
```

This makes `l` consistently mean "logs" from any list context, while `enter`/`right` always follow the canonical drill-down path.

## Spec inconsistency: `l` listed twice

`dev/CONCEPT.md` lines 50-51 list `l` under two conflicting entries:

```
- Right / Enter / `l`: drill down
- `l`: open logs from current context when meaningful
```

These are contradictory. The second definition is the intended one for list views — the first should read `Right / Enter` only. Update the spec when the implementation is settled.

## `Esc` doesn't clear filter when filter bar is visible

When the filter bar is open and has text, pressing `Esc` clears the filter. But if the filter bar is empty (user opened it and typed nothing), `Esc` falls through without closing the bar. The user must press `Esc` again to navigate back, which feels like the key didn't register.

**File:** `internal/ui/listview/listview.go:371-375`

```go
case "esc":
    if v.list.SettingFilter() || v.list.IsFiltered() {
        v.list.ResetFilter()
        return viewstate.Update{Action: viewstate.None, Next: v}
    }
    // falls through to pop
```

The bubbles list `SettingFilter()` returns false when the filter input is open but empty (it depends on whether the filter value is non-empty). Verify this edge case and ensure `Esc` always closes an open (even empty) filter bar on the first press.

## `Tab` column cycling is being removed

> **Superseded by `dev/plan-phase2.md`.** Column cycling (`tab cols` / `shift+tab`) is being removed from `listview.go`. Tab will be intercepted by `app.go` and used as a panel-focus switcher. Remove all `tab cols` / `tab view` hints from list footers as part of that phase.
