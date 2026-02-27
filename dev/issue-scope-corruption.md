# Issue: Scope State Corruption (Bugs 1, 2, 3)

These three reported bugs share a single root cause in `internal/app/app.go`.

## Root cause

Resource hotkey navigation (W, P, S, etc.) does not reset `m.scope`. If the user presses `N` first (which sets `m.scope = scopeNamespace`), then presses `W` to jump to workloads, `m.scope` stays `scopeNamespace` even though the workloads list is being shown.

**Relevant code (`app.go` default case in the key handler):**

```go
if res := m.registry.ResourceByKey(key); res != nil {
    m.saveHistory()
    view := listview.New(res, m.registry)
    view.SetSize(m.width, m.availableHeight())
    m.stack = []viewstate.View{view}
    m.crumbs = []string{normalizeBreadcrumbPart(view.Breadcrumb())}
    return m, nil   // m.scope is NOT reset
}
```

Once scope is corrupted to `scopeNamespace`, three different bugs follow.

---

## Bug 1: Context header corrupted by Enter on a workload row (Critical)

**Path:** `N` → select a namespace → `W` → Enter on `api`

After `N` → namespace selection, `restoreHistory()` is called which resets scope back to `scopeResources`. Then `W` is pressed: scope stays `scopeResources`. So far, so good.

But the `Push` handler in `app.go` (line 159) runs `isScopeSelectionMsg` for Enter/l/right/o and checks the current scope:

```go
if (m.scope == scopeNamespace || m.scope == scopeContext) && isScopeSelectionMsg(routedMsg) {
    if selected, ok := m.top().(selectedBreadcrumbProvider); ok {
        if value := normalizeBreadcrumbPart(selected.SelectedBreadcrumb()); value != "" {
            if idx := strings.Index(value, ": "); idx >= 0 {
                name := value[idx+2:]
                if m.scope == scopeNamespace {
                    m.namespace = name
                    resources.ActiveNamespace = name
                } else {
                    m.context = name
                }
            }
        }
    }
    ...
}
```

If scope has been left as `scopeNamespace` after a resource hotkey jump, then pressing Enter on `api` (the workload row) passes the `isScopeSelectionMsg` check, reads `api` as if it were a namespace name, and sets `m.namespace = "api"` (and `resources.ActiveNamespace = "api"`). The Context/Namespace header then shows corrupted values.

The corruption only triggers on Enter/l/right/o, which is why it's path-dependent.

---

## Bug 2: `N` does nothing from Workloads view (High)

**Direct consequence of Bug 1.**

After the path `N` → namespace → `W`, the `N` key handler checks:

```go
case "N":
    if m.scope != scopeNamespace {
        m.saveHistory()
        m.switchToScope(scopeNamespace)
    }
    return m, nil
```

If scope was left as `scopeNamespace` (from the corruption), the guard `m.scope != scopeNamespace` is false and the handler does nothing. The user presses `N` and nothing happens.

---

## Bug 3: Left from Workloads jumps directly to Contexts (High)

**Also a consequence of corrupted scope.**

The left-arrow handler:

```go
case "h", "left":
    if len(m.stack) > 1 {
        // pop within stack
    } else if m.scope == scopeResources {
        m.saveHistory()
        m.switchToScope(scopeNamespace)       // expected path
    } else if m.scope == scopeNamespace {
        m.saveHistory()
        m.switchToScope(scopeContext)          // jumps to context!
    }
```

If scope is `scopeNamespace` when the user is on the workloads list and presses left, the third branch fires and jumps all the way to Contexts, skipping Namespaces entirely.

---

## Fix

Reset scope to `scopeResources` whenever a resource hotkey navigates to a non-scope resource:

```go
if res := m.registry.ResourceByKey(key); res != nil {
    m.saveHistory()
    // 'N' and 'X' are scope resources; all others are content resources.
    if key != 'N' && key != 'X' {
        m.scope = scopeResources
    }
    view := listview.New(res, m.registry)
    view.SetSize(m.width, m.availableHeight())
    m.stack = []viewstate.View{view}
    m.crumbs = []string{normalizeBreadcrumbPart(view.Breadcrumb())}
    return m, nil
}
```

This is a one-line change that fixes all three bugs simultaneously.
