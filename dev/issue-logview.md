# Issue: Log View Wrap Ignores ANSI Escape Codes

**File:** `internal/ui/logview/logview.go` — `wrapLine`

`wrapLine` counts runes including ANSI escape sequences, so colorised log lines wrap at the wrong column (escape codes inflate the measured width).

### Fix

Use `ansi.PrintableRuneWidth` (already in the module via `charmbracelet/x/ansi`) to measure visible width, and strip/re-apply colour spans around the wrap points. Alternatively, strip ANSI before measuring width for the purpose of splitting, then re-join with the original bytes.
