# Podji

A fast, opinionated Kubernetes navigation TUI for debugging workflows.

## Experimental Status

Podji is **early-stage experimental software**.

- APIs, behavior, and keybindings can still change quickly.
- Some flows are intentionally incomplete or simulated.
- Bugs and UX inconsistencies should be expected.
- Use it for development/testing, not as a production-critical dependency.

## What Podji Is

- Read-focused Kubernetes navigation and troubleshooting UI
- Fast drill-down from resources to details/logs/events
- Deterministic mock mode for demos, tests, and offline development
- Optional live cluster mode via `client-go`

## What Podji Is Not (Yet)

- Not a full `kubectl` replacement
- Not a complete Kubernetes object browser/graph explorer
- Not a stable/feature-complete product

## Build and Run

```bash
go build ./cmd/podji
./podji
```

Default startup is `mock` mode (no cluster required).

## Runtime Modes

`PODJI_MODE` controls which store backend is used at startup:

- `mock` (default): deterministic stub data
- `kube`: client-go backed live cluster reads (with mock fallback if unavailable)

Example:

```bash
PODJI_MODE=kube ./podji
```

CLI flag alternative:

```bash
./podji --mode kube
```

## Mock Scenarios (for Testing and Demos)

Mock mode supports deterministic scenarios.

Use:

- `PODJI_MOCK_SCENARIO` (preferred)
- `PODJI_SCENARIO` (legacy fallback)

Supported values:

- `normal` (default)
- `empty`
- `forbidden`
- `partial`
- `offline`
- `stress` (enables synthetic list expansion)

Examples:

```bash
PODJI_MODE=mock PODJI_MOCK_SCENARIO=forbidden ./podji
PODJI_MODE=mock PODJI_STRESS=1 ./podji
```

## Test

```bash
go test ./...
```

## Core Navigation

| Key | Action |
|---|---|
| `→` / `Enter` / `l` | Drill down |
| `←` / `Backspace` / `h` | Back |
| `o` | Open logs from list |
| `r` | Related resources |
| `N` | Namespace picker |
| `X` | Context picker |
| `/` | Filter current list |
| `:` | Command bar |

## Development Helpers

- Interactive UI helper: `dev/ui.sh`
- Kube fixture helpers: `dev/kube/tui-fixtures.sh`, `dev/kube/tui-cluster.sh`
- Startup timing debug:

```bash
PODJI_DEBUG_DATA=1 ./podji
```

Look for `podji:app startup_ms=...` in logs.

## Current Scope (Subject to Change)

Working areas include navigation for workloads, pods, services, configmaps, secrets, nodes, namespaces, events, plus detail/log/yaml/describe views.

Still evolving: mutation behavior, CRD command-bar routing, and broader end-to-end kube integration coverage.
