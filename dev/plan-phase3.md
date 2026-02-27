# Phase 3: Related Panel as Persistent Side Panel

Move the related panel from a full-screen stack view to a persistent side panel. Tab switches focus between main and side panels. Enter in the side panel pushes to the main navigation stack. Empty drill-down auto-opens the side panel.

## 1. New state in `app.go`

Add to `Model`:
```go
side       viewstate.View
sideActive bool
```

Add to imports: `"github.com/dloss/podji/internal/ui/relatedview"`

## 2. Width helpers

```go
func (m *Model) mainWidth() int {
    if m.side != nil {
        return (m.width * 60) / 100
    }
    return m.width
}

func (m *Model) sideWidth() int {
    return m.width - m.mainWidth()
}
```

Call `m.top().SetSize(m.mainWidth(), m.availableHeight())` and `m.side.SetSize(m.sideWidth(), m.availableHeight())` whenever the side panel is opened or closed, and in `WindowSizeMsg` handling.

## 3. `r` key handler

Replace the existing `r` handler (which pushes related view onto the main stack) with:

```go
case "r":
    if m.side != nil {
        // Close side panel, restore full width to main.
        m.side = nil
        m.sideActive = false
        m.top().SetSize(m.width, m.availableHeight())
    } else {
        // Open side panel.
        side := relatedview.NewForSelection(m.top())
        side.SetSize(m.sideWidth(), m.availableHeight())
        m.side = side
        m.top().SetSize(m.mainWidth(), m.availableHeight())
    }
    return m, nil
```

## 4. Tab handler (completing Phase 2 stub)

```go
case "tab":
    if m.side != nil {
        m.sideActive = !m.sideActive
    }
    return m, nil
```

## 5. Key routing for side panel

Insert before the existing child-view routing in `Update()`:

```go
if m.sideActive && m.side != nil {
    update := m.side.Update(msg)
    switch update.Action {
    case viewstate.Push:
        // Navigation from side panel goes to the main stack, not the side.
        if len(m.crumbs) > 0 {
            if sel, ok := m.side.(viewstate.SelectionProvider); ok {
                if val := normalizeBreadcrumbPart(sel.SelectedBreadcrumb()); val != "" {
                    m.crumbs[len(m.crumbs)-1] = val
                }
            }
        }
        update.Next.SetSize(m.mainWidth(), m.availableHeight())
        m.stack = append(m.stack, update.Next)
        m.crumbs = append(m.crumbs, normalizeBreadcrumbPart(update.Next.Breadcrumb()))
        m.sideActive = false // focus follows to main
    case viewstate.Pop:
        // Side panel closed itself (Esc).
        m.side = nil
        m.sideActive = false
        m.top().SetSize(m.width, m.availableHeight())
    case viewstate.Replace:
        update.Next.SetSize(m.sideWidth(), m.availableHeight())
        m.side = update.Next
    case viewstate.OpenRelated:
        // Side panel requested a related panel for a sub-item — ignore for now.
    }
    return m, update.Cmd
}
```

## 6. `View()` rendering

Extract the current `View()` body into `renderMain()`. Then:

```go
func (m Model) View() string {
    main := m.renderMain()
    if m.overlay != nil {
        return overlayOnTop(main, m.overlay.View(), m.width, m.height)
    }
    if m.side != nil {
        side := m.side.View()
        return lipgloss.JoinHorizontal(lipgloss.Top, main, side)
    }
    return main
}
```

The side panel renders its own header inline (the related view already has a title line). It does not get a breadcrumb bar from `app.go`.

## 7. Focus indicator

When `sideActive` is false and a side panel is open, the side panel should appear visually subordinate. When `sideActive` is true, the main panel should recede.

Add `SetFocused(focused bool)` to `relatedview.View` and `relatedview.relationList`:
- Unfocused: render the title/header in dim style (reuse `style.Dim` if it exists, or apply `lipgloss.NewStyle().Faint(true)`)
- Focused: render normally

Call `m.side.(*relatedview.View).SetFocused(m.sideActive)` after toggling `m.sideActive`. Or, if relatedview exposes a `SetFocused` method on its `viewstate.View` interface indirectly, use a type assertion.

Alternative: define a `Focusable` interface in viewstate:
```go
type Focusable interface {
    SetFocused(bool)
}
```
App.go uses a type assertion: `if f, ok := m.side.(viewstate.Focusable); ok { f.SetFocused(m.sideActive) }`. This is cleaner and avoids importing relatedview from app.go just for the type assertion.

## 8. `relatedview.NewForSelection`

The related view needs to know which resource item is currently selected in the main panel. Define in `viewstate`:

```go
// SelectionProvider is implemented by views that have a selected item.
type SelectionProvider interface {
    SelectedItem() resources.ResourceItem
}
```

`listview.View` implements this by returning the currently highlighted item.

In `relatedview`:
```go
func NewForSelection(v viewstate.View) *View {
    if sel, ok := v.(viewstate.SelectionProvider); ok {
        item := sel.SelectedItem()
        return New(item) // existing constructor or refactored version
    }
    return NewEmpty() // no selection, show placeholder
}
```

If the current top-of-stack is not a list (e.g., a detail view), consider also implementing `SelectionProvider` on `detailview.View` to return the resource being viewed, so `r` still opens a useful related panel from detail views.

## 9. Auto-open related panel on empty drill-down

Add a new action constant to `viewstate/viewstate.go`:

```go
const (
    None    Action = iota
    Push
    Pop
    Replace
    OpenRelated // signal app.go to open the related side panel
)
```

In `listview.go`, `forwardView()`: before constructing and returning the forward view, check whether the result would be empty. If so, return `{Action: viewstate.OpenRelated}` instead of pushing an empty list.

The check is resource-specific but follows a pattern: query the data store for the expected items; if `len(items) == 0`, return `OpenRelated`.

In `app.go`, handle `OpenRelated` from the main stack:

```go
case viewstate.OpenRelated:
    if m.side == nil {
        side := relatedview.NewForSelection(m.top())
        side.SetSize(m.sideWidth(), m.availableHeight())
        m.side = side
        m.top().SetSize(m.mainWidth(), m.availableHeight())
    }
    m.sideActive = true // focus starts on side panel
```

This turns the empty-list dead end into the most useful view available, with focus already on the side panel so the user can navigate immediately.

## 10. Footer hints

**Main panel footer (listview):** When the side panel is open (passed via the view's context or a method), hints change:
- Side closed: `r related` (existing)
- Side open, main focused: `Tab related`
- Side open, side focused: (main footer dims or shows nothing)

The simplest approach: add `SetSideOpen(bool)` to listview.View or pass it through `SetSize` via a separate method. App.go calls this after opening/closing the side panel. Listview Footer() adjusts the hint accordingly.

**Side panel footer (relatedview):** When focused:
```
→ open   Tab main   Esc close
```
When unfocused: no footer (or just `Tab focus`).

Replace outdated relatedview focus hints with the above (`Tab main`).

## 11. Tests

- `r` with no side panel: opens side panel, calls `SetSize(sideWidth, h)` on side, calls `SetSize(mainWidth, h)` on main
- `r` with side panel open: closes it, calls `SetSize(fullWidth, h)` on main
- Tab with side open: toggles `sideActive`
- Tab with no side panel: no-op
- Push from side panel goes to `m.stack`, not side; `sideActive` becomes false
- Pop from side panel sets `m.side = nil`
- `OpenRelated` from child: opens side panel, sets `sideActive = true`
- `WindowSizeMsg`: resizes both panels correctly when side is open
- `relatedview.NewForSelection` with a `SelectionProvider` returns a non-nil view
- `relatedview.NewForSelection` with a non-provider returns a non-nil empty view
