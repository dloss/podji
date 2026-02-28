# Issue: Navigation Key Inconsistencies

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
