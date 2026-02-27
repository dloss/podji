# Issue: Footer Hint Inconsistencies

The footer is the primary discoverability surface. Several hints are either missing, wrong, or shown in contexts where the action does nothing.

## `s sort` shown on every list, but only works on workloads

**File:** `internal/ui/listview/listview.go:642`

```go
actions = append(actions, style.B("s", "sort"))
```

This is unconditional. Pressing `s` on services, configmaps, namespaces etc. is silently ignored — the handler checks `if sortable, ok := v.resource.(resources.ToggleSortable); ok` and does nothing if the interface isn't implemented.

### Fix

Only add the sort hint when the resource supports it:

```go
if _, ok := v.resource.(resources.ToggleSortable); ok {
    actions = append(actions, style.B("s", "sort"))
}
```

## `/` filter hint is absent

**Spec says:** `/ filter` should appear in the workloads footer.

**Current footer (workloads):** `s sort   v state   tab cols   r related   c copy   x execute`

The filter is one of the most-used actions and is completely invisible. The list model supports filtering via the built-in bubbles filter — it just isn't surfaced.

### Fix

Add `/ filter` to the actions slice. For filtered state, the status line (line 1) already shows the active filter value, so the action hint should always be present:

```go
actions = append(actions, style.B("/", "filter"))
```

Place it after the drill-down/log hints and before the secondary actions.

## Drill-down target hint is absent

**Spec says:** `-> pods` should appear in the workloads footer to show what Enter does.

**Current footer:** no drill-down indicator at all.

### Fix

For resources that have a known forward target, prepend a hint:

```go
// rough sketch — adapt to how forwardView determines the next resource
if resourceName == "workloads" {
    actions = append([]style.Binding{style.B("→", "pods")}, actions...)
}
```

The hint text should match the target resource name (pods, logs, containers, backends, etc.) and can be computed from the same logic used in `forwardView`.

## `l logs` hint absent from workloads footer

**Spec says:** `l logs` should appear in the workloads footer (and pods footer where applicable).

Currently `l` and `enter` are treated identically in workloads — both navigate to the pods list. The `o` key is what actually opens logs directly. The footer shows neither hint.

This is entangled with the `l` vs `o` key confusion (see `issue-navigation-keys.md`). Once the key mapping is resolved, add the appropriate hint.

## Related view footer shows `tab lens` but Tab does nothing there

**File:** `internal/ui/relatedview/relatedview.go:180`

```go
line2 := style.ActionFooter([]style.Binding{style.B("tab", "lens")}, v.list.Width())
```

Tab is not handled in `relatedview.Update()`. It falls through to the list model, which cycles columns (if any) — not lenses. The hint is wrong and misleading.

### Fix

Either:
1. Handle `tab` in relatedview by returning a `viewstate.Pop` so the app-level handler can switch lens, or
2. Remove the hint from the related view footer entirely, since lens-switching is a global concern that doesn't need to be advertised from every sub-view.

Option 2 is simpler and less surprising. The help screen (`?`) is the right place for global keys.

## Related list footer (drill-down from related) has the same problem

**File:** `internal/ui/relatedview/relatedview.go` (relation list view footer, ~line 492)

Same `tab lens` hint, same issue. Apply the same fix.
