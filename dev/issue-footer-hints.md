# Issue: Footer Hint Gaps

## Drill-down target hint is absent

**Spec says:** `→ pods` should appear in the workloads footer to show what Enter does.

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
