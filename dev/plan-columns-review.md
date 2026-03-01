# Column Plan — Stress Test & Edge Case Analysis

Issues found by reading the actual code against the plan. Grouped by severity.

---

## Blockers — must resolve before implementing

### 1. Wide mode: `refreshItems()` uses the wrong row function

`refreshItems()` always calls `tableRow(v.resource, res)` which calls `resource.TableRow()` — the normal row. But when `wideMode` is true, `v.columns` is the wide column set. The row slice from the normal `TableRow()` has no NODE, IP, QOS values, so all the wide-only columns render empty. The data and column definitions are mismatched on every refresh (resize, sort, etc.).

**Fix**: `refreshItems()` needs a branch:

```go
var rowFn func(ResourceItem) []string
if v.wideMode {
    if wide, ok := v.resource.(resources.WideResource); ok {
        rowFn = wide.TableRowWide
    }
}
if rowFn == nil {
    rowFn = func(item resources.ResourceItem) []string {
        return tableRow(v.resource, item)
    }
}
rows = append(rows, rowFn(res))
```

Or store `v.rowFn func(ResourceItem) []string` on the view and swap it when toggling wide mode. The plan's proposed `refreshColumns()` method only swaps `v.columns` but never addresses the row function — the plan is incomplete here.

### 2. Wide mode + namespace columns are unspecified

`namespacedColumns()` is called inside each resource's `TableColumns()`. Wide resources will need to do the same inside `TableColumnsWide()` — otherwise switching to all-namespaces mode and enabling wide mode will show rows with a leading NAMESPACE value but no NAMESPACE column header (or vice versa).

However, there's a subtler issue: wide rows need to also prepend the namespace string in all-namespaces mode. `TableRowWide()` returning `[]string` needs to call `namespacedRow()` internally, just like `TableRow()` does. Every `TableRowWide()` implementation must remember to do this. It's easy to forget and hard to test without explicitly testing the all-namespaces + wide combination.

**Preferred fix**: Move namespace injection out of individual resources and into `listview.refreshItems()`. Check `resources.ActiveNamespace == AllNamespaces` there, prepend the namespace column to `v.columns` if needed, and prepend the namespace value to each assembled row. Resources then never call `namespacedColumns()` or `namespacedRow()` — those become internal to listview. This is a bigger refactor but eliminates the class of "forgot to handle namespace in WideRow" bugs.

### 3. `refreshItems()` and map rows: the `item.row []string` type

After the Phase 2 map migration, `resource.TableRow()` returns `map[string]string`. But `refreshItems()` populates `item.row []string`, and `item.Title()` renders from `item.row`. The plan migrates the resource interface but doesn't specify how `item` changes.

Two options:

**a. Assemble `[]string` from map at refresh time** (keep `item.row []string`):
```go
row := make([]string, len(v.columns))
for i, col := range v.columns {
    row[i] = tableRowMap[col.ID]
}
```
`item.row` stays positionally bound to `v.columns` at the moment of refresh. This is correct, but means items become stale if `v.columns` changes after they were built — which is exactly what happens when the user applies a new column config. `refreshItems()` must be called after any column change, which it is. So this is fine, but it must be explicit in the plan.

**b. Store `map[string]string` on item, assemble in `Title()`**:
Requires changing `item.Title()` to take `v.columns` (or store a reference). Makes items less self-contained. Not recommended.

**Decision needed**: the plan should specify option (a) and make explicit that `refreshItems()` must always be called after `v.columns` changes.

### 4. Column picker overlay: rendering architecture not resolved

The existing `overlaypicker` is managed by `app.go`, which holds `m.overlay` and composites it in `app.View()`. The plan puts the column picker inside `listview.View` (`v.colPicker`), but `listview.View()` currently has no overlay compositing mechanism — it just returns a plain string.

There are three approaches, each with real trade-offs:

**a. Mirror the namespace/context overlay pattern** — `listview` emits a new message type (e.g., `OpenColumnPickerMsg`), `app.go` catches it, creates a picker overlay and stores it in `m.overlay`. `ColumnPickedMsg` is caught by `app.go` and forwarded to the listview. Cleanest architecturally, matches existing patterns, but requires `app.go` to know about column picking.

**b. `listview.View()` does its own compositing** — duplicate or extract `compositeOverlay` into a shared package, call it inside `listview.View()` on the rendered list content. More self-contained but duplicates app.go's overlay logic.

**c. Render inline, not as a floating overlay** — like the sort picker, which shows in the footer line. The column picker could render as a full-height panel on the right side of the list, not floating. Less elegant visually but architecturally trivial.

**Recommendation**: Option (a). Define a `OpenColumnPickerMsg` that `app.go` handles, same as it handles `OpenRelated`. `app.go` creates and owns the `columnpicker.Picker`. `ColumnPickedMsg` flows back to the active view via the normal message routing. This keeps all overlay management in one place.

---

## Design gaps — decisions needed before implementation

### 5. NAME column must be non-hideable

`item.FilterValue()` returns `item.data.Name` (hardcoded in `listview.go:96`). Drill-down uses `selected.data.Name`. Status coloring compares `i.row[idx]` against `i.status` to find which cell to color — if the status column is hidden, no coloring happens (acceptable), but if NAME is hidden, the table has no first column and the child hint in the header would display over nothing.

**Resolution**: NAME (column ID `"name"`) must always be visible and non-removable. In the picker, render it with a lock indicator (e.g., `• NAME`) and skip over it with space/enter. The `ColumnPickedMsg` always prepends `"name"` to `Visible` regardless of what the picker shows.

### 6. `colStore` singleton vs constructor injection

`app.go` creates views with `listview.New(res, m.registry)` in at least 6 places (namespace change, context change, resource hotkey, bookmark jump, and similar in relatedview). Adding a third parameter would touch all call sites.

The plan says "pass through app.go" but doesn't commit to how. Three realistic options:

**a. Add `colStore` to `listview.New()` signature**: All call sites updated. Clean, explicit. ~6 call sites.

**b. Package-level singleton in `columnconfig`**: `columnconfig.Default()` returns a global store. No signature changes. Less testable but very practical.

**c. Functional options on `View`**: `listview.New(res, registry, listview.WithColumnStore(store))`. Clean, backwards-compatible. Slightly verbose.

**Recommendation**: Option (b) for now (no signature churn during development), refactor to (a) or (c) when the store is battle-tested.

### 7. Wide mode state lost on view recreation

Namespace switch and resource hotkey navigation both call `listview.New()`, creating a fresh view. `wideMode = false` on fresh views. This is consistent but may surprise users who switched namespace and expected wide mode to persist.

**Decision needed**: is wide mode per-view (reset on navigation) or per-resource (persisted in `colStore`)?

Recommendation: per-view (reset) for Phase 1. If it becomes annoying, integrate wide mode as a flag in `ColumnConfig` in Phase 2. Document the per-view behavior explicitly.

### 8. Wide mode + column picker interaction

If the user has wide mode enabled and opens the column picker with `p`:
- The picker could show the **wide column pool** (what's currently visible)
- Or the **normal column pool** (editing the non-wide config)

These are fundamentally different UX models. Option 1 means wide mode and column config are merged into one. Option 2 means the picker edits the normal config but changes only take effect when wide mode is off — confusing.

**Resolved**: Pressing `p` while wide mode is active exits wide mode first, then opens the normal column picker. Show a brief footer hint: `w to re-enable wide`. Document this interaction in the plan.

### 9. Label column header truncation

Label key `app.kubernetes.io/version` is 26 chars. The header would display `label:app.kubernetes.io/version` (31 chars). With `padCell` truncating at the column width, this becomes `label:app.kuber…` at 20 chars — still not very readable.

**Two issues**:
1. **Display name**: Should the column display as `label:app.kubernetes.io/version` or as a shortened form? Suggestion: use the full label key as the column Name (not prefixed with `label:`) — e.g., `APP.KUBERNETES.IO/VERSION`. The `ID` stays `label:app.kubernetes.io/version`. This lets the header use the full standard column-name display without the redundant `label:` prefix.
2. **Width**: Cap label column widths at 20 in the pool definition. The existing shrink algorithm will handle overflow if terminal is narrow.

### 10. Column ordering after picker: new columns appear at the end

When a user checks a previously-unchecked column in the picker, where does it appear? The plan doesn't say. Options:
- **End of the visible list** (simplest)
- **At its canonical position in the pool** (feels natural)
- **After the last currently-visible column before it in the pool order**

Recommendation: canonical pool order. The picker shows all columns in pool order; the visible list respects the same order. No reordering in Phase 2. This simplifies `ColumnPickedMsg`: `Visible` is always in pool order, not arbitrary user order.

### 11. `[custom]` indicator placement

The plan says append to the header row. The header is already `RESOURCE → Pods  STATUS  READY  RESTARTS  AGE`. Adding `[custom]` right-aligned there requires knowing the remaining width and positioning it — non-trivial string manipulation on an already-complex line.

**Better placement**: footer line 1 (the status line), alongside filter/sort indicators. Already uses `style.StatusFooter(indicators, ...)` which takes a `[]style.Binding`. Add a `style.B("columns", "custom")` binding to that slice when non-default config is active. Zero layout complexity.

---

## Implementation notes — handle during coding

### 12. `namespacedRow` and map rows

After Phase 2 migration, each resource's `TableRow()` returns `map[string]string`. The namespace injection currently done by `namespacedRow()` (prepending a string to `[]string`) no longer applies.

Simplest approach: every resource always includes `"namespace": item.Namespace` in its map. `namespacedColumns()` continues to conditionally prepend the NAMESPACE column to the pool. `listview` looks up `row["namespace"]` for that column. Resources need no conditional logic in `TableRow()` — just always include it.

However, this changes every resource's `TableRow()` to always include namespace even when it's not displayed — minor overhead, acceptable.

### 13. Picker cursor must skip section headers

Section header rows (`── standard ──`, `── labels ──`) are non-selectable. The picker's `j`/`k` movement must skip them. Implement by tracking which indices in the displayed list are headers (e.g., a `[]bool` parallel slice `isHeader`). When moving cursor, skip entries where `isHeader[newPos]` is true.

### 14. `TableRowWide()` as positional `[]string` vs map

Phase 1 uses `[]string` for `TableRowWide()`. After Phase 2's migration, the codebase will have two row formats in flight. When Phase 2 is done, `TableRowWide()` should also be migrated to `map[string]string`. Track this as explicit tech debt: Phase 1 PR should include a `TODO: migrate to map in Phase 2`.

### 15. Minimum column count

If every optional column is hidden and only NAME remains, `columnWidthsForRows` with one column and lots of available width will expand NAME to fill the terminal. This looks odd but is technically correct. Not a blocker, just verify visually.

### 16. Key for column picker: use `p`, not `C`

`C` (uppercase) conflicts on two fronts: (1) all uppercase letters are reserved for navigation (resource hotkeys, scope pickers), and (2) `C` is already registered as the ConfigMaps resource key — pressing it would navigate away instead of opening a picker.

Use `p` (lowercase) for the column picker. It is free across all resource keys (all use uppercase) and all existing listview bindings. Reads naturally as "picker" or "preferences", sits alongside `s` (sort), `c` (copy), `w` (wide) as a view-level action.

Updated footer: `p columns  s sort  c copy  w wide` — all lowercase, no ambiguity with navigation.

### 17. Status color cell detection with hidden columns

`item.Title()` colors a cell if `i.row[idx] == i.status` (the status string value). If STATUS is hidden, the cell simply doesn't appear. This is correct — no coloring needed. But if the user moves STATUS to a different position by adding a column before it, the index check still works because `item.row` is assembled in `v.columns` order at refresh time. No problem here, just verify.

### 18. Wide mode `w` key and the logs view

`w` is wrap toggle in `logview`. Since `logview` is a different view type on the stack, there's no conflict — keys are dispatched to the top of the stack. But if a user is in logs (where `w` wraps) and hits `left`/`backspace` to go back to the pod list, they might instinctively press `w` again thinking it wraps. It will instead enable wide mode. Acceptable UX given they're in a different view, but worth a note in the help text.

---

## Summary table

| # | Issue | Phase | Severity |
|---|-------|-------|----------|
| 1 | `refreshItems()` uses wrong row function in wide mode | 1 | Blocker |
| 2 | Wide mode + namespace column interaction unspecified | 1 | Blocker |
| 3 | `item.row []string` vs map row assembly not specified | 2 | Blocker |
| 4 | Column picker overlay architecture not resolved | 2 | Blocker |
| 5 | NAME column must be non-hideable | 2 | Design gap |
| 6 | `colStore` injection strategy not decided | 2 | Design gap |
| 7 | Wide mode state lost on view recreation | 1/2 | Design gap |
| 8 | Wide mode + `p` interaction: exit wide first | 1+2 | Resolved |
| 9 | Label column header display and width | 3 | Design gap |
| 10 | Column ordering when adding new columns | 2 | Design gap |
| 11 | `[custom]` indicator placement | 2 | Design gap |
| 12 | `namespacedRow` replacement strategy | 2 | Impl note |
| 13 | Picker cursor skipping section headers | 3 | Impl note |
| 14 | `TableRowWide()` format inconsistency (tech debt) | 1→2 | Impl note |
| 15 | Single-column view looks odd | 2 | Impl note |
| 16 | Use `p` (not `C`) for column picker — `C` is ConfigMaps nav key | 2 | Resolved |
| 17 | Status color cell detection with hidden columns | 2 | Impl note |
| 18 | `w` key muscle memory between views | 1 | Impl note |
