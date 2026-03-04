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
just build
./podji
```

Default startup is `kube` mode.

## Versioning

The binary embeds version metadata at build time:

```bash
./podji --version
./podji -v
```

Output includes:

- tag/describe version
- git commit short SHA
- UTC build timestamp

For consistent releases:

1. Create and push a semver tag (for example `v0.0.2`).
2. Build from that tagged commit via `just build`.
3. Verify `./podji --version` reports that exact tag before publishing artifacts.

## Runtime Modes

`PODJI_MOCK` enables deterministic mock mode. When unset, Podji runs in `kube` mode (client-go backed live cluster reads).

Examples:

```bash
PODJI_MOCK=1 ./podji
./podji -mock
```

## Mock Scenarios (for Testing and Demos)

Mock mode supports deterministic scenarios.

Use:

- `PODJI_MOCK_SCENARIO`

Supported values:

- `normal` (default)
- `empty`
- `forbidden`
- `partial`
- `offline`
- `stress` (enables synthetic list expansion)

Examples:

```bash
PODJI_MOCK=1 PODJI_MOCK_SCENARIO=forbidden ./podji
PODJI_MOCK=1 PODJI_STRESS=1 ./podji
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
| `o` | Context action (open logs or next view) |
| `r` | Related resources |
| `N` | Namespace picker |
| `X` | Context picker |
| `/` | Search current list |
| `&` | Filter current list |
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
