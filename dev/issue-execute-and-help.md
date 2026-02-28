# Issue: Resource Browser Footer Missing Hints

**File:** `internal/ui/resourcebrowser/resourcebrowser.go`

The resource browser already supports filtering (`model.SetFilteringEnabled(true)`) and `f` find-by-first-char mode (lines 127–137, `computeFindTargets`), but neither is surfaced in the footer or help overlay.

### Fix

Add both hints to the footer:

```go
actions := []style.Binding{
    style.B("/", "filter"),
    style.B("f", "find"),
}
line2 := style.ActionFooter(actions, v.list.Width())
```

The help overlay currently lists nothing useful for the browser section. Update to:

```
RESOURCE BROWSER (A)
  / (slash)            Filter by kind or group
  f <char>             Jump to first resource by char
  enter / right        Open resource list
```
