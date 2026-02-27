# Issue: Execute Mode Silent Dismiss and Help Gaps

## 1. Execute mode: any unrecognised key silently closes the menu

**File:** `internal/ui/listview/listview.go:270-295`

When execute mode is active (`execState == execMenu`), the key handler unconditionally clears the state before checking which key was pressed:

```go
if v.execState == execMenu {
    v.execState = execNone      // cleared regardless of key
    switch key.String() {
    case "d":
        v.execState = execConfirmDelete
    case "r":
        if v.supportsRestart() {
            v.execState = execConfirmRestart
        }
    case "s":
        if v.supportsScale() {
            v.execState = execInputScale
            v.execInput = v.currentReplicas()
        }
    ...
    }
    return viewstate.Update{Action: viewstate.None, Next: v}
}
```

**The problem:** if `s` is pressed but `supportsScale()` is false (e.g. on a Service or ConfigMap), the menu closes with no visual feedback. The user sees nothing happen, assumes the keypress was dropped, and presses `s` again — which now fires sort on the list.

The same applies to any key not listed in the switch: pressing `w` or `p` in exec mode silently dismisses the menu.

### Fix

Only dismiss the menu on `Esc` or on a recognised action key. Unrecognised keys should be ignored while the menu is open:

```go
if v.execState == execMenu {
    switch key.String() {
    case "esc":
        v.execState = execNone
    case "d":
        v.execState = execNone
        v.execState = execConfirmDelete
    case "r":
        if v.supportsRestart() {
            v.execState = execNone
            v.execState = execConfirmRestart
        }
    case "s":
        if v.supportsScale() {
            v.execState = execNone
            v.execState = execInputScale
            v.execInput = v.currentReplicas()
        }
    case "f":
        if v.supportsPortFwd() {
            v.execState = execNone
            v.execState = execInputPortFwd
            v.execInput = "8080:8080"
        }
    case "x":
        if v.supportsShellExec() {
            v.execState = execNone
            return viewstate.Update{Action: viewstate.None, Next: v, Cmd: v.shellExecCmd()}
        }
    // unrecognised key: stay in exec menu, do nothing
    }
    return viewstate.Update{Action: viewstate.None, Next: v}
}
```

A simpler alternative: keep the current dismissal behaviour but only dismiss on Esc — use a `default: // ignore` case and only exit on recognised keys or Esc.

---

## 2. `c copy` missing from help overlay

**File:** `internal/ui/helpview/helpview.go`

The TABLE section of the help text:

```
TABLE
  tab                  Focus related panel (when open)
  / (slash)            Filter
  esc                  Clear filter
  s                    Sort (name/problem)
  v                    State (workloads)
  f <char>             Jump to first item by char
  d                    Describe
  y                    YAML
  e                    Events for selected item
  r                    Toggle related panel
  o                    Logs (or next table)
  space / pgup / pgdn  Page up / down
  c                    Copy mode (n name, k kind/name, p -n ns name)
  x                    Execute mode (d delete, r restart, s scale, f port-fwd, x shell)
```

`c` (copy) is not listed. It should appear alongside `x`:

```
  c                    Copy mode (n name, k kind/name, p -n ns name)
  x                    Execute mode (d delete, r restart, s scale, f port-fwd, x shell)
```

---

## 3. Resource browser footer should show `/ filter`

**File:** `internal/ui/resourcebrowser/resourcebrowser.go:217`

```go
line2 := style.ActionFooter([]style.Binding{style.B("/", "filter")}, v.list.Width())
```

The resource browser already supports filtering (`model.SetFilteringEnabled(true)` at line 110), and `FilterValue()` is implemented on `browserItem` to search by kind and group. The footer should surface this directly:

```go
actions := []style.Binding{
    style.B("/", "filter"),
    style.B("f", "find"),
}
line2 := style.ActionFooter(actions, v.list.Width())
```

The `f` find-by-first-char mode is also implemented in the browser (lines 127-137, `computeFindTargets`) but undocumented in the footer and the help overlay. Add it to both.

The help overlay currently mentions only `Tab` for the browser section (which it doesn't have, and Tab no longer cycles columns). Update:

```
RESOURCE BROWSER (A)
  / (slash)            Filter by kind or group
  f <char>             Jump to first resource by char
  enter / right        Open resource list
```
