# UI Review Notes

## Strengths

- Great information density in the workload table (kind, ready, status, restarts, age) without feeling cramped
- Color-coded statuses (green Healthy, red/magenta Failed/Degraded, yellow Progressing) enable fast visual scanning
- Main views grouped by operator concern (Apps/Network/Infrastructure) make resource navigation faster than a flat resource list
- Scope navigation (N namespace, X context) is quick with a clear breadcrumb trail at the top
- Filter (`/`) is responsive and narrows results instantly
- Sort by problem (`s` cycling to `sort:problem`) surfaces Failed/Degraded items to the top — great for incidents
- State simulation (`v` cycling normal/empty/forbidden/partial/offline) is clever for development and edge-case testing
- Drill-down flow (Workload → Pod → Container → Logs) feels natural

## Open Issues

### Related view feels underdeveloped
Pressing `r` from the pod list shows a "Related" screen with just "Events (3)" as a single selectable item. The events themselves only showed one line. Expected more depth.

### Lots of empty space
With only 9 workloads the bottom 60% of the screen is blank. Consider a detail/preview pane for the selected item, or collapsing the viewport.

## Resolved

- **Navigation back is confusing** — Escape now clears filter first, then navigates back on next press (same as Backspace).
- **`?` help key does nothing** — Help overlay with full keybinding reference, pushed as a view without polluting breadcrumbs.
- **Footer bar gets truncated** — Footer now truncates to terminal width with ellipsis; view-specific keys ordered first.
- **No visual selection highlight** — Selected row now has bold white text, dark background, and a heavier `▌` left border.
- **Namespace switch doesn't change data** — Stub data varies per namespace (production, staging, monitoring, kube-system, default).
