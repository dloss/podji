# Customizable Table Columns — Implementation Plan

Three independently shippable phases:

- **Phase 1 — Wide mode** (`w` toggle): each resource defines an optional wide column set
- **Phase 2 — Column visibility picker** (`C` overlay): user hides/shows columns from a per-resource pool
- **Phase 3 — Label columns**: dynamic columns sourced from item labels, selectable in the Phase 2 picker

Phase 1 ships without touching the data model. Phases 2 and 3 require a one-time data model migration (positional rows → map rows) that is mechanical but touches every resource file.

---

## Architecture: the positional row problem

The core complication is that `TableRow(item)` currently returns `[]string` matched positionally to `TableColumns()`. Hiding column index 2 breaks the alignment — the row slice at index 2 no longer maps to the visible column at index 2.

**Phase 1 sidesteps this** by keeping wide columns as a separate method on a separate interface; nothing changes in the row assembly pipeline.

**Phases 2 and 3 require** migrating to a map-keyed row, so that `listview` can select any subset of columns safely.

---

## Data model changes (prerequisite for Phases 2+)

### Add a stable `ID` to `TableColumn`

```go
// internal/resources/types.go
type TableColumn struct {
    ID      string // stable key, e.g. "status", "restarts", "label:app"
    Name    string // display header
    Width   int    // default width hint
    Default bool   // included in default view; false = opt-in only
}
```

### Change `TableRow` to return a map

```go
type TableResource interface {
    TableColumns() []TableColumn
    TableRow(item ResourceItem) map[string]string // keyed by column ID
}
```

`listview` builds the visible ordered column list from the pool (filtered by user config or defaults), then assembles each display row as `row[col.ID]`.

**Migration** is mechanical: every resource changes its `TableRow` from returning `[]string` to returning `map[string]string`. Same data, explicit keys. `namespacedColumns` and `namespacedRow` become `namespacedColumns` (unchanged) and the namespace value is inserted via the `"namespace"` ID key.

---

## Phase 1 — Wide mode

**Key**: `w` in list views (no conflict; logs view uses `w` for wrap, a different view stack level).

### New interface

```go
// internal/resources/types.go
type WideResource interface {
    TableColumnsWide() []TableColumn
    TableRowWide(item ResourceItem) []string // still positional — independent pipeline
}
```

Wide mode uses the existing positional row pipeline. No data model migration needed.

### `listview.View` changes

Add:
```go
wideMode bool
```

On `w` keypress, if `resource` implements `WideResource`, toggle `wideMode` and call `refreshColumns()`.

`refreshColumns()` (new private method):
```go
func (v *View) refreshColumns() {
    if v.wideMode {
        if wide, ok := v.resource.(resources.WideResource); ok {
            v.columns = wide.TableColumnsWide()
        }
    } else {
        v.columns = tableColumns(v.resource)
    }
    v.refreshItems() // already recomputes widths
}
```

Footer hint: append `w wide` to the action bar when `WideResource` is implemented. When wide mode is active, render as `w [wide]` (dimmed label, bright key).

### Wide column sets per resource

| Resource    | Normal columns                        | Added in wide                  |
|-------------|---------------------------------------|-------------------------------|
| Pods        | NAME STATUS READY RESTARTS AGE        | NODE IP QOS                   |
| Deployments | NAME READY STATUS AGE                 | SELECTOR STRATEGY             |
| Services    | NAME TYPE CLUSTER-IP ENDPOINTS AGE    | EXTERNAL-IP PORT(S)           |
| Nodes       | NAME STATUS ROLES VERSION AGE         | OS ARCH KERNEL-VERSION        |
| Ingresses   | NAME CLASS HOSTS AGE                  | ADDRESS TLS RULES             |

`ResourceItem` currently lacks NODE, IP, etc. Add an `Extra map[string]string` bag rather than growing the struct with fields only used by wide mode:

```go
type ResourceItem struct {
    Name      string
    Namespace string
    Kind      string
    Status    string
    Ready     string
    Restarts  string
    Age       string
    Labels    map[string]string
    Selector  map[string]string
    Extra     map[string]string // wide-mode and future fields: "node", "ip", "qos", etc.
}
```

### Files touched — Phase 1

- `internal/resources/types.go` — `WideResource` interface, `Extra` field on `ResourceItem`
- `internal/resources/pods.go`, `deployments.go`, `services.go`, `nodes.go`, `ingresses.go` — `TableColumnsWide()` + `TableRowWide()`, populate `Extra` in stub data
- `internal/ui/listview/listview.go` — `wideMode` field, `w` handler, `refreshColumns()`, footer hint

### Testing — Phase 1

- Unit: wide column set is a strict superset of the normal set (every normal column ID appears in wide)
- Integration:
  ```
  dev/ui.sh key w   # header gains NODE, IP columns
  dev/ui.sh key w   # header returns to normal
  ```

---

## Phase 2 — Column visibility picker

**Key**: `C` (uppercase; consistent with `N`/`X` for settings-like overlays; `c` is already copy mode).

**Prerequisite**: data model migration above must be complete.

### `ColumnConfig` and `Store`

```go
// internal/columnconfig/columnconfig.go

type ColumnConfig struct {
    Visible []string // column IDs in display order
}

type Store struct {
    mu      sync.RWMutex
    configs map[string]ColumnConfig // keyed by resource.Name()
}

// Get returns the active column list for a resource.
// If no config is set, returns defaults from the pool.
func (s *Store) Get(resourceName string, pool []resources.TableColumn) []resources.TableColumn

// Set stores user-chosen visible column IDs for a resource.
func (s *Store) Set(resourceName string, visible []string)

// Reset removes user config, reverting to defaults.
func (s *Store) Reset(resourceName string)
```

The store is a singleton (or passed through `app.go`), shared across all list view instances so config persists when navigating back.

### Column picker overlay

New package: `internal/ui/columnpicker/`

Visual layout (rendered as a floating box, same technique as `overlaypicker`):

```
╭─ columns ──────────────────────╮
│  ✓ NAME                        │
│  ✓ STATUS                      │
│ ▶ ✓ READY                      │  ← cursor
│  ✓ RESTARTS                    │
│  ✓ AGE                         │
│  ──────────────────────────── │
│  ○ NODE              [wide]    │
│  ○ IP                [wide]    │
│  ──────────────────────────── │
│  ○ label: app                  │  ← Phase 3
│  ○ label: env                  │
│                                │
│  spc toggle  enter apply       │
│  r reset     esc cancel        │
╰────────────────────────────────╯
```

Keys:
- `j`/`k` or arrows: move cursor
- `space`: toggle check on cursor item
- `enter`: emit `ColumnPickedMsg` with current selection and close
- `r`: reset to defaults (does not close; shows checkboxes restored to default state)
- `esc`: close without applying

```go
type ColumnPickedMsg struct {
    ResourceName string
    Visible      []string // column IDs in chosen order
}
```

Picker constructor takes: `resourceName string`, `pool []resources.TableColumn`, `current []string` (currently active IDs). The pool is split into sections: defaults, wide-only (if `WideResource`), label columns (Phase 3).

### `listview.View` changes

```go
colPickMode bool
colStore    *columnconfig.Store
colPicker   *columnpicker.Picker // non-nil while picker is open
```

On `C` keypress:
```go
case "C":
    if _, ok := v.resource.(resources.TableResource); ok {
        pool := v.resource.(resources.TableResource).TableColumns()
        current := columnIDs(v.columns)
        v.colPicker = columnpicker.New(v.resource.Name(), pool, current)
        v.colPickMode = true
    }
```

On `ColumnPickedMsg`:
```go
v.colStore.Set(msg.ResourceName, msg.Visible)
v.columns = v.colStore.Get(v.resource.Name(), pool)
v.refreshItems()
```

Render: when `colPickMode`, overlay `v.colPicker.View()` in the bottom-right area (or centered, TBD by visual testing).

`SuppressGlobalKeys` returns `true` when `colPickMode` is active — prevents `N`, `X`, etc. firing while picker is open.

Footer indicator: when user has a non-default column config, append a dimmed `[custom]` label to the header row (right-aligned, similar to `[wide]`).

### Files added — Phase 2

- `internal/columnconfig/columnconfig.go`
- `internal/ui/columnpicker/columnpicker.go`
- `internal/ui/columnpicker/columnpicker_test.go`

### Files touched — Phase 2

- `internal/resources/types.go` — `ID`/`Default` fields on `TableColumn`, `TableRow` return type change
- All resource files — migrate `TableRow` to `map[string]string`, add `ID`/`Default` to each `TableColumn` literal
- `internal/resources/registry.go` — update `namespacedColumns`/`namespacedRow` helpers for map rows
- `internal/ui/listview/listview.go` — `colPickMode`, `colStore`, `colPicker`, `C` handler, overlay render, footer hint
- `internal/app/app.go` — construct `columnconfig.Store`, pass into each new list view

### Testing — Phase 2

- Unit: `columnconfig.Store` — get returns defaults when empty; set/reset round-trip; concurrent access safe
- Unit: `columnpicker` — space toggles; enter emits correct visible list; esc preserves prior state; r resets without closing
- Unit: row assembly from `map[string]string` — hidden column absent from display; unknown ID → empty string
- Integration:
  ```
  dev/ui.sh key C          # picker opens
  dev/ui.sh key Space      # toggle RESTARTS off
  dev/ui.sh key Enter      # confirm
  dev/ui.sh cap            # header must not contain RESTARTS
  dev/ui.sh key C          # reopen
  dev/ui.sh key r          # reset to defaults
  dev/ui.sh key Enter      # confirm
  dev/ui.sh cap            # RESTARTS visible again
  ```

---

## Phase 3 — Label columns

**Entry point**: label columns appear as a third section in the Phase 2 column picker. No new key needed.

**Prerequisite**: Phase 2 complete.

### Label discovery

After `resource.Items()` loads, `listview` scans all `item.Labels` across the full result set, collects unique keys, sorts alphabetically, and passes them to the column picker as additional `TableColumn` entries:

```go
func labelColumnsFromItems(items []resources.ResourceItem) []resources.TableColumn {
    seen := map[string]bool{}
    var keys []string
    for _, item := range items {
        for k := range item.Labels {
            if !seen[k] {
                seen[k] = true
                keys = append(keys, k)
            }
        }
    }
    sort.Strings(keys)
    cols := make([]resources.TableColumn, 0, len(keys))
    for _, k := range keys {
        cols = append(cols, resources.TableColumn{
            ID:      "label:" + k,
            Name:    "label:" + k,
            Width:   max(len("label:"+k), 12),
            Default: false,
        })
    }
    return cols
}
```

Label columns use the ID prefix `label:` to avoid collisions with resource-defined column IDs and to make them identifiable for persistence.

### Row value resolution

In `listview`, when assembling a display row, label columns are resolved directly — no change needed in the resource's `TableRow`:

```go
for _, col := range v.columns {
    if strings.HasPrefix(col.ID, "label:") {
        labelKey := strings.TrimPrefix(col.ID, "label:")
        row = append(row, item.Labels[labelKey]) // empty string if absent
    } else {
        row = append(row, tableRowMap[col.ID])
    }
}
```

### Picker section layout

The picker receives three column slices and renders them in sections:
1. Resource-defined columns (`Default: true` shown first, then `Default: false` / wide)
2. Label columns (dynamically computed, always `Default: false`)

Section headers: `── standard ──` and `── labels ──` (dimmed, non-selectable).

### `TableColumn` — no additional field needed

`LabelKey` does not need to be a separate struct field; the `label:` prefix on `ID` is sufficient. The row assembly code checks the prefix.

### Persistence (optional, within Phase 3)

Config file at `~/.config/podji/columns.yaml`:

```yaml
pods:
  visible:
    - name
    - status
    - ready
    - age
    - "label:app"
services:
  visible:
    - name
    - type
    - cluster-ip
    - age
```

```go
// internal/columnconfig/columnconfig.go additions
func (s *Store) Load(path string) error
func (s *Store) Save(path string) error
```

Load on startup in `cmd/podji/main.go`; save after each `ColumnPickedMsg` is applied. If the file is missing, malformed, or references unknown column IDs, silently fall back to defaults (log a warning if a debug flag is set).

### Files added — Phase 3

- `internal/columnconfig/columnconfig.go` — `Load`/`Save` methods (additions)

### Files touched — Phase 3

- `internal/ui/listview/listview.go` — `labelColumnsFromItems()`, label column row resolution, pass label pool to picker
- `internal/ui/columnpicker/columnpicker.go` — section rendering (standard / labels)
- `cmd/podji/main.go` — load config on startup, pass config path to store

### Testing — Phase 3

- Unit: `labelColumnsFromItems` — deduplication, sorting, correct ID format, empty labels handled
- Unit: label value lookup — present key returns value; absent key returns empty string
- Unit: `Store.Load` / `Store.Save` — round-trip, unknown IDs ignored, missing file returns no error
- Integration:
  ```
  dev/ui.sh key C          # picker opens, labels section visible
  dev/ui.sh key j          # navigate to label:app
  dev/ui.sh key Space      # toggle on
  dev/ui.sh key Enter      # confirm
  dev/ui.sh cap            # header shows label:app column
  ```

---

## Dependency graph

```
Phase 1 (wide mode)
  └─ independent; ships without data model changes

Data model migration
  ├─ ID + Default fields on TableColumn
  ├─ TableRow: []string → map[string]string
  └─ prerequisite for Phases 2 and 3

Phase 2 (visibility picker)
  ├─ needs: data model migration
  ├─ adds: columnconfig.Store, columnpicker overlay
  └─ ships in-memory only (no persistence)

Phase 3 (label columns)
  ├─ needs: Phase 2 picker (for UI entry point)
  ├─ adds: label discovery, label row resolution, picker section
  └─ optional: YAML persistence

Phase 3 persistence
  └─ needs: Phase 3 data model
```

Phases can be reviewed and merged independently. The data model migration is a separate PR that unblocks Phase 2.

---

## Key design decisions

**Why map rows?** The alternative (keep `[]string`, filter by index) works for hiding but breaks with label columns (which have no resource-defined index) and with reordering. Maps are explicit and index-collision-free.

**Why `C` for the column picker?** `c` is already copy mode. Uppercase overlay-openers (`N`, `X`) set the precedent for settings-like pickers.

**Why `w` for wide mode?** `W` is taken by Workloads navigation. `w` is free in list views. Wide mode is a preview/toggle — a single keypress with immediate visual feedback — rather than a persistent settings change, which is why it lives on a lowercase letter rather than going through the `C` picker.

**Why not column reordering in Phase 2?** Reordering needs either drag-and-drop (impractical in TUI) or up/down movement within the picker. Up/down is already used for cursor movement. This can be added later with a modifier (e.g., `shift+j`/`shift+k` or a dedicated "move" mode within the picker) once visibility is working.

**Why `Extra map[string]string` on `ResourceItem`?** Adding NODE, IP, QOS as top-level struct fields grows the shared type with fields only relevant to wide mode on a few resources. `Extra` keeps the struct lean. Label columns already have `Labels map[string]string`; `Extra` covers the rest.
