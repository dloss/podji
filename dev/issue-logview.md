# Issue: Log View Missing Features

The log view (`internal/ui/logview/logview.go`) advertises more functionality in its footer than it actually implements, and is missing several keybindings defined in the spec.

## Misleading `/` search hint

**Footer shows:** `/ search`
**Handler exists:** No — there is no `case "/"` in `Update()`.

Remove `/` from the footer until search is implemented, or implement it.

### How to fix

Implement in-viewport search using the viewport's built-in support or a simple scan:

```go
// in View struct
searchQuery  string
searchActive bool
matchOffsets []int
matchIndex   int
```

```go
case "/":
    v.searchActive = true
    v.searchQuery = ""
    // enter search input mode
case "n":
    if len(v.matchOffsets) > 0 {
        v.matchIndex = (v.matchIndex + 1) % len(v.matchOffsets)
        v.viewport.SetYOffset(v.matchOffsets[v.matchIndex])
    }
case "N":
    if len(v.matchOffsets) > 0 {
        v.matchIndex = (v.matchIndex - 1 + len(v.matchOffsets)) % len(v.matchOffsets)
        v.viewport.SetYOffset(v.matchOffsets[v.matchIndex])
    }
```

## Missing `[` / `]` to cycle `--since` window

**Spec says:** `[ / ]`: cycle `--since` window (1m, 5m, 15m, 1h, all)
**Implemented:** No handler exists.

This is important for crash-loop debugging: the user wants to expand the time window when the default 5m doesn't show the crash.

### How to fix

Add a `sinceIndex` field and a fixed slice of window labels:

```go
var sinceWindows = []string{"1m", "5m", "15m", "1h", "all"}

// in View struct
sinceIndex int  // default 1 → "5m"
```

```go
case "]":
    v.sinceIndex = (v.sinceIndex + 1) % len(sinceWindows)
    v.lines = v.resource.LogsSince(v.item, sinceWindows[v.sinceIndex])
    v.refreshContent()
case "[":
    v.sinceIndex = (v.sinceIndex - 1 + len(sinceWindows)) % len(sinceWindows)
    v.lines = v.resource.LogsSince(v.item, sinceWindows[v.sinceIndex])
    v.refreshContent()
```

Show the active window in the status line (line 1 of the footer) when it's not the default.

## Missing `space` / `pgdn` / `pgup` paging

**Spec says:** `space` / `pgdn`: page down, `pgup`: page up
**Implemented:** Only `up`/`k` and `down`/`j` for line scrolling.

The viewport already supports `GotoTop`, `GotoBottom`, `HalfViewDown`, `ViewDown`, `ViewUp` etc. Wire them up:

```go
case "pgdown", " ":
    v.viewport.ViewDown()
case "pgup":
    v.viewport.ViewUp()
```

## Missing `c` container picker

**Spec says:** `c`: container picker (switch container without leaving logs)
**Implemented:** No handler exists. Multi-container pods require the user to go back to the container list.

### How to fix

When `v.container` is set (entered via `NewWithContainer`), `c` should return a `viewstate.Pop` and re-push the container list — or open an inline picker overlay. The simplest approach is to pop back and let navigation resume:

```go
case "c":
    // Pop back to container list — the stack already holds it.
    return viewstate.Update{Action: viewstate.Pop}
```

A richer version would be an inline picker modal, consistent with how the execute mode is handled in `listview.go`.

## Wrap ignores ANSI escape codes

`wrapLine` (`logview.go:136`) counts runes including ANSI escape sequences, so colorized log lines wrap at the wrong column.

### How to fix

Use `ansi.PrintableRuneWidth` (already in the module via `charmbracelet/x/ansi`) to measure visible width, and strip/re-apply color spans around the wrap points. Alternatively, strip ANSI before measuring width for the purpose of splitting, then re-join with the original bytes.

## Footer actions to add when implemented

Once the above features exist, the footer (line 2) should read:

```
t mode   f follow   w wrap   / search   [ ] since   c container   pgup/pgdn page
```

The status line (line 1) should show active non-default state:

```
mode: previous   follow: off   wrap: off   since: 15m   match 3/7
```
