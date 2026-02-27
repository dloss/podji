# Issue: Related View Footer and Navigation Problems

> **Note:** Phase 3 (`dev/plan-phase3.md`) moves the related view from a full-screen stack view to a persistent side panel. The footer hints and Tab behaviour described here are updated in that plan.

## Related footer focus hint should be `Tab main`

Both the category view (`relatedview.View`) and the drill-down resource list (`relatedview.relationList`) show an outdated footer hint:

```go
// relatedview.go:180
line2 := style.ActionFooter([]style.Binding{style.B("tab", "main")}, v.list.Width())

// relatedview.go:492
actions := []style.Binding{style.B("tab", "main")}
```

Tab should indicate panel focus, not a nonexistent view mode.

After Phase 2, `app.go` intercepts Tab before any child view sees it — the related view never receives it regardless. After Phase 3, the related view runs as a persistent side panel, and Tab means "return focus to the main panel". The hint should say `Tab main`.

### Fix (applicable after Phase 3)

Use `Tab main` in both footer methods:

```go
style.B("tab", "main")
```

The Phase 3 plan specifies the full side-panel footer as: `→ open   Tab main   Esc close`.

## Category view footer missing Enter hint

The related category list (`relatedview.View`) footer shows only `Tab main`. There is no hint that `enter` / `→` opens the selected category.

```go
func (v *View) Footer() string {
    // ...
    line2 := style.ActionFooter([]style.Binding{style.B("tab", "main")}, v.list.Width())
    return line1 + "\n" + line2
}
```

### Fix

Add an Enter hint:

```go
actions := []style.Binding{
    style.B("→", "open"),
    style.B("/", "filter"),
    style.B("f", "find"),
}
line2 := style.ActionFooter(actions, v.list.Width())
```

The `f` find-by-key mode (already implemented in `relatedview.Update`) is also undiscoverable — it has no footer hint either.

## Relation list footer already guards `s sort` correctly

`relationList.Footer()` does:

```go
if _, ok := v.resource.(resources.ToggleSortable); ok {
    actions = append(actions, style.B("s", "sort"))
}
```

This is the correct pattern. The top-level list view (`listview.go:642`) should follow the same guard — it currently doesn't (see `issue-footer-hints.md`).

## `f` find mode is undocumented

The related category view supports a `f` key that highlights first-letter jump targets. It is not mentioned in the footer or in the help screen. Once the footer is updated, add:

```go
style.B("f", "find")
```

to the actions slice.
