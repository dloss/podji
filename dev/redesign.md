# Navigation Redesign

## Rationale

The current navigation model has accumulated two structural problems that compound each other.

**Two competing navigation models.** The app uses a stack navigator (Enter pushes, Back pops, breadcrumbs track depth) for resource drill-down, and a hotkey jumper (W, P, S, N, X…) for direct navigation. These have incompatible mental models: the stack assumes you retrace your steps, hotkeys assume teleportation. The app bridges them with `saveHistory()`/`restoreHistory()`, which creates invisible state the user cannot reason about. After pressing `W` from deep in a pod detail view, does Back take you to the pod list? No — `W` silently discarded the stack. After pressing `N` then `W`, is scope still `scopeNamespace`? Yes — and this is the root cause of bugs 1, 2, and 3 in `issue-scope-corruption.md`.

**Scope as a navigation destination.** `N` and `X` make namespace/context a *place you navigate to* — a third scope level in the hierarchy. This forces a save/restore state machine (the `snapshot` struct), and creates the question "which scope am I in?" that the user has no way to answer. It also breaks the arrow-key navigation model: Left from workloads is supposed to go "up", but what "up" means depends on invisible scope state.

**Tab wasted on column cycling.** After the lens concept was removed, Tab was reassigned to column cycling. This is non-standard, undiscoverable, and burns the most universal "move focus" key in any UI. It also requires the bubbles list component to consume Tab before app.go can intercept it, which makes Tab unavailable as a global key.

The redesign fixes all three by making the navigation model uniform:

- **Scope → overlay pickers.** `N`/`X` open a floating picker over the current view. Selecting a namespace closes the overlay and applies the filter; the stack is untouched. No scope state machine, no save/restore, no invisible scope level.
- **Related panel → persistent side panel.** `r` toggles a side panel that coexists with the main view rather than replacing it. Tab switches focus between them. Enter on a side-panel item pushes to the main stack, so drill-down is consistent regardless of which panel initiated it.
- **Tab → panel focus.** Tab is intercepted by app.go before child views see it. Column cycling is removed.

## New Navigation Model

| Key | Action |
|---|---|
| `W`, `P`, `S`, `D`, … | Jump to resource type (replaces stack root) |
| `Enter` / `Right` / `l` | Drill down (push to stack) |
| `Backspace` / `Left` / `Esc` | Go back (pop stack) |
| `N` | Open namespace picker overlay |
| `X` | Open context picker overlay |
| `r` | Toggle related side panel |
| `Tab` | Focus: main ↔ related panel |
| `/` | Filter current list |
| `l` | Logs (from pod/container context) |
| `?` | Help |
| `q` / `Ctrl+C` | Quit |

The stack and hotkey jumper coexist cleanly because namespace/context are no longer part of the navigation hierarchy.

## Phase Summary

| Phase | Change | Key benefit |
|---|---|---|
| 1 | Remove scope system, add overlay pickers | Scope bug class eliminated; `app.go` ~130 lines shorter |
| 2 | Tab owned by `app.go`, column cycling removed | Tab available as global key; listview simplified |
| 3 | Related panel as persistent side panel | Related resources first-class; Tab, Enter, and empty-drill-down all connected |

Each phase leaves the app in a working, testable state. Phase 1 must precede Phase 3 because it simplifies `app.go` before split-panel state is added. Phase 2 must precede Phase 3 because Tab cannot work as a panel-focus key until column cycling is removed.
