# Note: Podji Development Workflow

## Recommended Workflow

Use two modes in parallel:

- `mock` mode for day-to-day UI and behavior development
- `kube` mode for quick integration sanity checks

This keeps iteration fast while still verifying real-cluster paths.

## Daily Fast Loop (Mock)

```bash
go test ./...
./podji --mode mock
```

For reproducible UI checks:

```bash
dev/ui.sh start
dev/ui.sh key Down Enter
dev/ui.sh cap
dev/ui.sh quit
```

## Lightweight Real-Cluster Loop (Kube)

Use the kube helper scripts for quick fixture setup/teardown:

```bash
dev/kube/tui-fixtures.sh up
./podji --mode kube
dev/kube/tui-fixtures.sh down
```

If kube mode cannot initialize, Podji exits with a clear startup error.

For local startup timing checks:

```bash
PODJI_DEBUG_DATA=1 ./podji
```

Look for `podji:app startup_ms=...` in logs.

## Resource Tips

- Prefer single-node local clusters.
- Avoid heavy add-ons unless needed for a specific test.
- Keep dataset small (few namespaces/workloads).
- Close Docker Desktop when not actively running kube-mode tests.

## Testing Split

- `mock` mode: primary coverage for navigation, UX, and regression tests.
- `kube` mode: targeted smoke checks (context/namespace discovery, pod logs/events).

Practical target:

- 80-90% work in `mock` mode
- 10-20% quick kube smoke checks
