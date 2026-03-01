# Podji - Kubernetes TUI

## Build & Run

```bash
go build ./cmd/podji
./podji
```

## Test

```bash
go test ./...
```

## Interactive UI Testing

The app uses stub data (no real Kubernetes cluster needed). Use `dev/ui.sh` to interact with the running TUI:

```bash
dev/ui.sh start          # Build, launch in tmux (120x40), capture initial screen
dev/ui.sh key Down       # Send key(s), auto-capture after delay
dev/ui.sh key Tab Enter  # Multiple keys in one call
dev/ui.sh type hello     # Send literal text (for filter input etc)
dev/ui.sh cap            # Capture plain text screen
dev/ui.sh cap -e         # Capture with ANSI escape codes (for color parsing)
dev/ui.sh vhs            # Run built-in VHS tape, returns path to gif
dev/ui.sh vhs file.tape  # Run custom VHS tape
dev/ui.sh quit           # Send q, kill session
```

`key`, `type`, and `start` auto-capture after a short delay (default 300ms, override with `PODJI_DELAY`).

Use VHS when you need to verify colors, styling, or visual layout. Use tmux for fast interactive exploration.

## Architecture

- **Framework**: Bubbletea (charmbracelet/bubbletea)
- **Entry point**: `cmd/podji/main.go`
- **Main model**: `internal/app/app.go` - stack-based navigation; Enter/Right pushes, Backspace/Left pops
- **Views** implement `viewstate.View` interface (Init, Update, View, Breadcrumb, Footer, SetSize)
- **Resources**: `internal/resources/` - each resource type registered with a single-letter hotkey
- **Scope switching**: `N`/`X` open overlay pickers for namespace/context; selecting applies scope without changing stack depth
- **Related panel**: `r` toggles a persistent related side panel; `Tab` switches focus between main and side panels
