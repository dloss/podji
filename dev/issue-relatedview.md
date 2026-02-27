# Issue: Related View Footer and Navigation Problems

## `tab lens` shown but Tab is not handled

Both the category view (`relatedview.View`) and the drill-down resource list (`relatedview.relationList`) show a `tab lens` footer hint:

```go
// relatedview.go:180
line2 := style.ActionFooter([]style.Binding{style.B("tab", "lens")}, v.list.Width())

// relatedview.go:492
actions := []style.Binding{style.B("tab", "lens")}
```

Neither `Update()` function handles the `tab` key. It falls through to the bubbles list model which may do nothing or cycle its own internal state — it does not switch the app lens.

### Fix option A — remove the hint

Lens switching is a global concern; it works by the app-level handler receiving Tab before child views do. If the child views don't intercept Tab, the bubbles model swallows it before the app sees it, so the hint is both wrong and actively prevents the global binding from working.

Remove `tab lens` from both footers. If a global key guide is needed, the help screen (`?`) is the right place.

### Fix option B — forward Tab upward

Return a `viewstate.Pop` on Tab so the app regains focus and processes its own Tab handler:

```go
case "tab":
    return viewstate.Update{Action: viewstate.Pop}
```

This would pop back to the list, switch lens, and lose the related context — probably not the right UX. Option A is simpler.

## Category view footer missing Enter hint

The related category list (`relatedview.View`) footer shows only `tab lens`. There is no hint that `enter` / `→` opens the selected category.

```go
func (v *View) Footer() string {
    // ...
    line2 := style.ActionFooter([]style.Binding{style.B("tab", "lens")}, v.list.Width())
    return line1 + "\n" + line2
}
```

### Fix

Add an Enter hint (after removing the broken `tab lens`):

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
