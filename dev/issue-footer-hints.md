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

## `o logs` hint absent from workloads footer

`o` opens logs directly from a workload (skipping the pods list). The footer shows no hint for this.

### Fix

Add `o logs` to the workloads footer actions:

```go
if isWorkloads {
    actions = append(actions, style.B("o", "logs"))
}
```
