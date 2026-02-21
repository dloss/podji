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

## Interactive UI Testing via tmux

The app uses stub data (no real Kubernetes cluster needed). To observe and interact with the running TUI:

```bash
# Launch in tmux
tmux new-session -d -s podji -x 120 -y 40 './podji'

# Send keystrokes
tmux send-keys -t podji Down
tmux send-keys -t podji Tab
tmux send-keys -t podji Enter
tmux send-keys -t podji q

# Capture current screen as plain text
tmux capture-pane -t podji -p

# Capture with ANSI color codes (for parsing styling)
tmux capture-pane -t podji -p -e

# Quit and clean up
tmux send-keys -t podji q; tmux kill-session -t podji
```

## Visual Screenshots via VHS

[VHS](https://github.com/charmbracelet/vhs) produces color screenshots as GIF/PNG. Output must use `.gif` extension. Dimensions are in pixels (minimum 120x120).

```bash
# Write a tape file
cat > /tmp/podji_screenshot.tape << 'EOF'
Set Width 1200
Set Height 600
Set FontSize 14
Hide
Type "./podji"
Enter
Sleep 2s
Show
Sleep 1s
EOF

# Run it (produces a GIF with full color rendering)
vhs -o /tmp/podji_screen.gif /tmp/podji_screenshot.tape
```

Use VHS when you need to verify colors, styling, or visual layout. Use tmux for fast interactive exploration.

## Architecture

- **Framework**: Bubbletea (charmbracelet/bubbletea)
- **Entry point**: `cmd/podji/main.go`
- **Main model**: `internal/app/app.go` - stack-based navigation with scope/lens abstraction
- **Views** implement `viewstate.View` interface (Init, Update, View, Breadcrumb, Footer, SetSize)
- **Resources**: `internal/resources/` - each resource type registered with a single-letter hotkey
- **Three lenses**: Apps, Network, Infrastructure (cycle with Tab)
- **Three scopes**: Context, Namespace, Lens (navigate with N/X or left arrow)
