# Phase 1: Scope System → Overlay Pickers

Remove the three-scope state machine from `app.go` and replace `N`/`X` with lightweight floating overlay pickers. The scope corruption bug class (issue-scope-corruption.md) is eliminated as a consequence.

## 1. Remove scope system from `app.go`

**Fields to remove from `Model` (lines 33–34):**
```go
scope   int       // remove
history []snapshot // remove
```

**Types to remove:**
- `const scopeContext = 0`, `scopeNamespace = 1`, `scopeResources = 2` (lines 16–20)
- `type snapshot struct { stack, crumbs, scope }` (lines 22–26)
- `type selectedBreadcrumbProvider interface` (lines 45–47)

**Methods to remove:**
- `saveHistory()` (~lines 253–259)
- `restoreHistory()` (~lines 261–272)
- `switchToScope()` (~lines 274–292)
- `isScopeSelectionMsg()` (~lines 215–226)

**Key handlers to remove or simplify:**

`N` handler (lines 113–118) — replace entirely (see §3 below).

`X` handler (lines 119–124) — replace entirely (see §3 below).

Left/Backspace handlers (lines 100–112, 88–98) — remove the `else if m.scope == scopeResources` and `else if m.scope == scopeNamespace` branches. Both handlers reduce to:
```go
if len(m.stack) > 1 {
    m.stack = m.stack[:len(m.stack)-1]
    m.crumbs = m.crumbs[:len(m.crumbs)-1]
    m.crumbs[len(m.crumbs)-1] = normalizeBreadcrumbPart(m.top().Breadcrumb())
}
```

**Push handler cleanup (lines 158–185):** Remove the entire `if (m.scope == scopeNamespace || m.scope == scopeContext) && isScopeSelectionMsg(...)` block. The Push case simplifies to just the breadcrumb update + stack append.

**`New()` cleanup (line 60):** Remove `scope: scopeResources` from the Model literal.

**Imports:** Remove `"strings"` if it's only used by `isScopeSelectionMsg` (check other usages first).

## 2. New `overlaypicker` component

Create `internal/ui/overlaypicker/overlaypicker.go`.

```go
package overlaypicker

import bubbletea "github.com/charmbracelet/bubbletea"
import "github.com/dloss/podji/internal/ui/viewstate"

// SelectedMsg is emitted as a Cmd when the user confirms a selection.
type SelectedMsg struct {
    Kind  string // "namespace" or "context"
    Value string
}

type Picker struct {
    kind    string   // "namespace" or "context" — used in SelectedMsg
    items   []string // full unfiltered list
    filter  string
    cursor  int
    width   int
    height  int
}

func New(kind string, items []string) *Picker
```

**Behavior:**

| Key | Action |
|---|---|
| Any printable char | Append to filter, reset cursor to 0 |
| `Backspace` | Delete last filter char |
| `Up` / `k` | Move cursor up in filtered list |
| `Down` / `j` | Move cursor down in filtered list |
| `Enter` | Emit `SelectedMsg{Kind, Value}` via Cmd, return `Pop` |
| `Esc` | Return `Pop` (cancel, no selection) |

`Update()` returns `viewstate.Update{Action: viewstate.Pop, Cmd: cmd}` on Enter (cmd emits SelectedMsg) and `viewstate.Update{Action: viewstate.Pop}` on Esc.

**Rendering:** Render as a floating box centered in the allocated width/height using `lipgloss.Place`. The box contains:
1. Title line: `"  namespace  "` or `"  context  "` (bold)
2. Filter prompt: `> <filter>`
3. Filtered item list, cursor item highlighted

Box sizing: `min(42, width-4)` wide, `min(len(filtered)+4, height-4)` tall.

`SetSize(w, h int)` stores dimensions for use in View().

`Breadcrumb()` and `Footer()` return `""` — the overlay does not participate in breadcrumbs or footers.

## 3. Overlay integration in `app.go`

**Add to `Model`:**
```go
overlay *overlaypicker.Picker
```

**Add to imports:** `"github.com/dloss/podji/internal/ui/overlaypicker"`

**New `N` handler:**
```go
case "N":
    items := resources.NamespaceNames()
    m.overlay = overlaypicker.New("namespace", items)
    m.overlay.SetSize(m.width, m.height)
    return m, nil
```

**New `X` handler:**
```go
case "X":
    items := resources.ContextNames()
    m.overlay = overlaypicker.New("context", items)
    m.overlay.SetSize(m.width, m.height)
    return m, nil
```

**Route all input to overlay when active.** At the top of `Update()`, before the `bubbletea.KeyMsg` switch, add:
```go
if m.overlay != nil {
    update := m.overlay.Update(msg)
    if update.Action == viewstate.Pop {
        m.overlay = nil
    }
    return m, update.Cmd
}
```

**Handle `SelectedMsg`.** In the `msg` type switch (before the `bubbletea.KeyMsg` case):
```go
case overlaypicker.SelectedMsg:
    m.overlay = nil
    if msg.Kind == "namespace" {
        m.namespace = msg.Value
        resources.ActiveNamespace = msg.Value
    } else {
        m.context = msg.Value
    }
    // Reload workloads so the new namespace/context takes effect.
    if res := m.registry.ResourceByKey('W'); res != nil {
        view := listview.New(res, m.registry)
        view.SetSize(m.width, m.availableHeight())
        m.stack = []viewstate.View{view}
        m.crumbs = []string{normalizeBreadcrumbPart(view.Breadcrumb())}
    }
    return m, nil
```

**Render overlay over main content.** In `View()`:
```go
func (m Model) View() string {
    main := m.renderMain() // extract existing View() body into renderMain()
    if m.overlay != nil {
        return overlayOnTop(main, m.overlay.View(), m.width, m.height)
    }
    return main
}
```

`overlayOnTop` centers the overlay string over the main content string using `lipgloss.Place`. The main content is rendered at full size first; the overlay box is then placed on top at center/center position.

## 4. Stub data additions

Add to `internal/resources/` (whichever file owns stub data):

```go
func NamespaceNames() []string  // returns stub namespace names
func ContextNames() []string    // returns stub context names
```

These are used only by the overlay pickers. The existing namespace/context ResourceTypes remain for the `A` (all resources) browser.

## 5. Tests

**Remove** any test exercising: `saveHistory`, `restoreHistory`, `switchToScope`, scope constants, `isScopeSelectionMsg`, `selectedBreadcrumbProvider`.

**Add `overlaypicker_test.go`:**
- Filter narrows the item list
- Backspace removes last filter character
- Enter emits `SelectedMsg` with correct Kind and Value
- Esc returns `{Action: Pop}` with nil Cmd
- Cursor clamps at list boundaries

**Add to `app_test.go`** (or create):
- `N` key sets `m.overlay` non-nil with kind "namespace"
- `X` key sets `m.overlay` non-nil with kind "context"
- `SelectedMsg{Kind: "namespace", Value: "staging"}` updates `m.namespace` and reloads workloads
- Input is routed to overlay when `m.overlay != nil` and not to the main stack
