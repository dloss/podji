# Phase 2: Tab Ownership + Drop Column Cycling

Make `app.go` own the Tab key so it can be used as a panel-focus switcher in Phase 3. Remove column cycling from `listview.go`, which is the only current Tab handler in child views.

## Why child views currently swallow Tab

Bubbletea routes messages to the root model's `Update` first (`app.go`). `app.go` handles its global keys in a `switch`, then passes the message to the current top-of-stack view. If `app.go` has no `case "tab":` branch, Tab falls through to `listview.Update()`, where the column cycling code consumes it. The bubbles list component itself also intercepts Tab as a focus-next action. Either way, `app.go` never processes it.

Fix: add `case "tab":` to `app.go`'s global key switch *before* the message is routed to child views. Child views never see Tab after this.

## 1. Intercept Tab in `app.go`

Add to the global key switch in `Update()` (after the `"?"` case, before `default`):

```go
case "tab":
    // Phase 3 will toggle side panel focus here.
    // For now, Tab is a no-op at the app level.
    return m, nil
```

This single line makes all column cycling code in `listview.go` unreachable dead code.

## 2. Remove column cycling from `listview.go`

**Remove from `View` struct (line 106):**
```go
colOffset int
```

**Remove methods:**
- `visibleNonFirstCount()` (lines 121–129)
- `visibleColumns()` (lines 134–145)
- `visibleRow()` (lines 148–159)

**Remove key handlers (lines 354–370):**
```go
// Remove entirely:
if key.Type == bubbletea.KeyShiftTab || key.String() == "shift+tab" || key.String() == "backtab" {
    ...
}

// Remove from the switch:
case "tab":
    ...
```

**Replace all call sites:**
- `v.visibleColumns()` → `v.columns` (wherever used in rendering and width calculation)
- `v.visibleRow(row)` → `row` (wherever rows are built for display)

**Update `refreshItems()`:** Remove any logic that recalculates layout based on `colOffset`. Width is now always calculated for the full column set.

**Update `Footer()`:** Remove any `tab columns` or `tab view` hints. (CONCEPT.md listed `Tab view` in the workloads footer — this is now obsolete.)

## 3. Remove column cycling tests

In `listview_test.go`, remove the test cases that exercise `colOffset` / Tab cycling:

- Tab cycles colOffset by visibleNonFirstCount (~lines 355–370)
- Shift+Tab reverses colOffset (~lines 372–382)
- Tab wraps at boundary (~lines 384–396)
- Tab advances by visible count (~lines 398–414)
- First column stays pinned after cycling (~lines 416–425)

Check for any other tests that reference `colOffset` directly and remove them.

## 4. Verify no regressions

After removal, the full column set is always shown. On narrow terminals this may cause columns to be truncated or squeezed. This is acceptable — the column cycling feature existed to work around narrow terminals, but it was undiscoverable and non-standard. If column truncation becomes a real problem, a future phase can address it with a different mechanism (e.g., a toggle to hide less-important columns, invoked by a dedicated key like `c`).

Run `go test ./...` and `go build ./cmd/podji`. Launch with `dev/ui.sh start` and verify:
- Tab does nothing visible (correct — side panel not yet wired)
- Shift+Tab does nothing (correct)
- All columns are shown in the workloads list
- Arrow-key navigation and drill-down still work
