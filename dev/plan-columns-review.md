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

**Resolved**: Assemble `[]string` from map at refresh time (keep `item.row []string`):
```go
row := make([]string, len(v.columns))
for i, col := range v.columns {
    row[i] = tableRowMap[col.ID]
}
```
`item.row` stays positionally bound to `v.columns` at the moment of refresh. `refreshItems()` must always be called after `v.columns` changes — this must be explicit in the plan.

### 4. Column picker overlay: rendering architecture not resolved

The existing `overlaypicker` is managed by `app.go`, which holds `m.overlay` and composites it in `app.View()`. The plan puts the column picker inside `listview.View` (`v.colPicker`), but `listview.View()` currently has no overlay compositing mechanism — it just returns a plain string.

**Resolved**: Mirror the namespace/context overlay pattern. `listview` emits `OpenColumnPickerMsg`; `app.go` catches it and creates a picker overlay stored in `m.overlay`. `ColumnPickedMsg` flows back to the active view via normal message routing. All overlay management stays in one place, matching the existing pattern exactly.

---

## Design gaps — resolved

### 5. NAME column must be non-hideable

`item.FilterValue()` returns `item.data.Name` (hardcoded). Drill-down uses `selected.data.Name`. If NAME is hidden, the table has no first column and the child hint in the header would display over nothing.

**Resolved**: NAME (column ID `"name"`) is always visible and non-removable. In the picker, render it with a lock indicator (e.g., `• NAME`) and have space/enter skip over it. `ColumnPickedMsg` always prepends `"name"` to `Visible` regardless of picker state.

### 6. `colStore` injection strategy

`app.go` creates views with `listview.New(res, m.registry)` in at least 6 places. Adding a third parameter touches all call sites.

**Resolved**: Package-level singleton — `columnconfig.Default()` returns a global store. No signature changes during development. Refactor to explicit injection later if testability becomes a concern.

### 7. Wide mode state lost on view recreation

Namespace switch and resource hotkey navigation both call `listview.New()`, creating a fresh view. `wideMode = false` on fresh views.

**Resolved**: Wide mode is per-view (resets on navigation). Document this explicitly in the help text and footer. If persistence becomes wanted, add `WideMode bool` to `ColumnConfig` in Phase 2.

### 8. Wide mode + column picker interaction

**Resolved**: Pressing `p` while wide mode is active exits wide mode first, then opens the normal column picker. Show a brief footer hint: `w to re-enable wide`.

### 9. Label column header display and width

Label key `app.kubernetes.io/version` is 26 chars. The header `label:app.kubernetes.io/version` (31 chars) would be mostly truncated at any reasonable column width.

**Resolved**: Display name is the full label key without the `label:` prefix — e.g., `APP.KUBERNETES.IO/VERSION`. ID stays `label:app.kubernetes.io/version` for storage. Cap label column width at 20 in the pool definition; the existing shrink algorithm handles overflow.

### 10. Column ordering after picker

When a user enables a previously-hidden column, where does it appear?

**Resolved**: Canonical pool order. The picker shows columns in pool order; the assembled visible list respects the same order. No reordering in Phase 2. `ColumnPickedMsg.Visible` is always in pool order.

### 11. `[custom]` indicator placement

The plan says append to the header row, but that line is already complex and width-constrained.

**Resolved**: Place in footer line 1 (the status/indicator line) alongside filter and sort indicators. Add a `style.B("columns", "custom")` binding to that slice when config differs from default. Zero layout complexity.

---

## Implementation notes — handle during coding

### 12. `namespacedRow` and map rows

After Phase 2 migration, each resource's `TableRow()` returns `map[string]string`. Every resource always includes `"namespace": item.Namespace` in its map. `namespacedColumns()` continues to conditionally prepend the NAMESPACE column to the pool. `listview` looks up `row["namespace"]` for that column. Resources need no conditional logic — just always include it.

### 13. Picker cursor must skip section headers

Section header rows (`── standard ──`, `── labels ──`) are non-selectable. The picker's `j`/`k` movement must skip them. Track which indices are headers (e.g., a `[]bool` parallel slice `isHeader`). When moving cursor, skip entries where `isHeader[newPos]` is true.

### 14. `TableRowWide()` as positional `[]string` vs map

Phase 1 uses `[]string` for `TableRowWide()`. After Phase 2's migration, the codebase will have two row formats in flight. When Phase 2 is done, `TableRowWide()` should also be migrated to `map[string]string`. Phase 1 PR should include a `TODO: migrate TableRowWide to map in Phase 2`.

### 15. Minimum column count

If every optional column is hidden and only NAME remains, `columnWidthsForRows` with one column expands NAME to fill the terminal. Technically correct. Verify visually.

### 16. Key for column picker: `p`

All resource navigation keys are uppercase. Use `p` (lowercase) for the column picker — free across all resource keys and all existing listview bindings. Footer: `p columns  s sort  c copy  w wide`.

### 17. Status color cell detection with hidden columns

`item.Title()` colors a cell if `i.row[idx] == i.status`. If STATUS is hidden, no coloring — acceptable. If STATUS moves position (due to column config), the index still matches because `item.row` is assembled in `v.columns` order at refresh time. No action needed, just verify.

### 18. Wide mode `w` key and the logs view

`w` is wrap toggle in `logview`. No conflict since keys go to the top of the stack. Worth noting in help text that `w` means different things in list vs log views.

---

## Summary table

| # | Issue | Phase | Status |
|---|-------|-------|--------|
| 1 | `refreshItems()` uses wrong row function in wide mode | 1 | Blocker |
| 2 | Wide mode + namespace column interaction unspecified | 1 | Blocker |
| 3 | `item.row []string` vs map row assembly | 2 | Resolved |
| 4 | Column picker overlay architecture | 2 | Resolved |
| 5 | NAME column must be non-hideable | 2 | Resolved |
| 6 | `colStore` injection strategy | 2 | Resolved |
| 7 | Wide mode resets on view recreation | 1/2 | Resolved |
| 8 | Wide mode + `p` interaction: exit wide first | 1+2 | Resolved |
| 9 | Label column header display and width | 3 | Resolved |
| 10 | Column ordering when adding new columns | 2 | Resolved |
| 11 | `[custom]` indicator placement | 2 | Resolved |
| 12 | `namespacedRow` replacement strategy | 2 | Impl note |
| 13 | Picker cursor skipping section headers | 3 | Impl note |
| 14 | `TableRowWide()` format inconsistency (tech debt) | 1→2 | Impl note |
| 15 | Single-column view looks odd | 2 | Impl note |
| 16 | Use `p` for column picker | 2 | Resolved |
| 17 | Status color cell detection with hidden columns | 2 | Impl note |
| 18 | `w` key muscle memory between views | 1 | Impl note |
