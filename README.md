# Podji

A fast, opinionated Kubernetes navigation TUI focused on debugging workflows.

Same scope as k9s for day-to-day exploration, but with instant navigation, calmer visuals, and detail/log flows optimized for finding problems quickly.

## Vision

Podji helps engineers move through Kubernetes the way they debug it: fast navigation, predictable keybindings, short paths to logs and relationships.

It intentionally avoids modeling the full Kubernetes ontology and instead optimizes for practical troubleshooting.

**Non-goals:** exposing every resource as first-class UI, replacing `kubectl` for advanced operations, providing a full graph explorer. Podji is a read-focused navigation and debugging tool.

## UX Principles

1. One obvious forward path per object.
2. Graph relationships exist, but are optional and consistent.
3. Most debugging ends in logs — logs must be extremely fast.
4. Few concepts, repeated everywhere.
5. Arrow keys define structure and feel natural.
6. Color indicates status only (no decorative color).
7. Whitespace over borders; dim non-essential noise.

## Navigation

| Key | Action |
|-----|--------|
| `→` / `Enter` / `l` | Drill down |
| `←` / `Backspace` / `h` | Back |
| `o` | Open logs directly from any list |
| `r` | Related resource panel |
| `N` | Namespace picker |
| `X` | Context picker |
| `/` | Filter current list |

**Drill-down model:** every object has one default next step — Namespace → Workloads → Pods → Logs. Predictable and reversible.

## Success Criteria

Users can:

- reach crash logs in ≤ 4 steps
- diagnose "Service has no backends" in one view + one drill-down
- find ConfigMap consumers quickly
- switch namespace without losing their place in the navigation stack

## Build & Run

```bash
go build ./cmd/podji
./podji
```

The app uses stub data — no real Kubernetes cluster needed.

## Test

```bash
go test ./...
```

## Architecture

- **Framework**: [Bubbletea](https://github.com/charmbracelet/bubbletea)
- **Entry point**: `cmd/podji/main.go`
- **Main model**: `internal/app/app.go` — stack-based navigation; Enter/Right pushes, Backspace/Left pops
- **Views** implement `viewstate.View` (Init, Update, View, Breadcrumb, Footer, SetSize)
- **Resources**: `internal/resources/` — each resource type registered with a single-letter hotkey
- **Scope switching**: `N`/`X` open overlay pickers; selecting applies scope without changing stack depth
- **Related panel**: `r` toggles a persistent side panel; `Tab` switches focus

**Store**: thin in-memory cluster projection via client-go shared informers, with minimal eager indexes (owner → children, service → EndpointSlices). UI reads from Store only.

## v1 Scope

**In:** pods, deployments, statefulsets, daemonsets, jobs, cronjobs, services, configmaps, secrets, nodes, namespaces, events — full navigation, detail views, log viewer, context/namespace switching.

**Out:** mutating actions, metrics, CRDs, multi-cluster, config persistence, plugins.
