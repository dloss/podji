# Kubira

A Kubernetes TUI for people who find k9s frustrating. Same scope — browse resources, read logs, understand your cluster — but with calm visuals, instant navigation, a log viewer worth using, and detail views that show you what matters instead of dumping YAML. Go, single binary, fast.

## Why not k9s

| Problem | Kubira's answer |
|---|---|
| Noisy UI — decorative colors, dense tables, box-drawing clutter | Color means status, nothing else. Whitespace over borders. |
| `:command` palette — typing `:pod`, `:deploy` is opaque and hard to discover | Uppercase hotkeys: `P` pods, `D` deployments, `S` services. Instant. |
| Bad log viewer — basic streaming, weak search, no multi-container | First-class viewer: `/` search, wrap, timestamps, follow, container picker |
| YAML walls — describe dumps hundreds of unformatted lines | Structured detail views: status, containers, conditions, events — formatted for humans |

## Design Principles

- **Color = status.** Red for crash/error, yellow for warning/pending, green for ready, dim for healthy/boring. No decorative color anywhere.
- **Whitespace over borders.** Sections separated by space, not box-drawing characters. Clean, scannable.
- **Dim the noise.** Pod hash suffixes, UIDs, internal labels — dimmed or truncated. The meaningful part (deployment name, user-set labels) is prominent.
- **Instant navigation.** Uppercase hotkeys jump to resource types. Left/right moves through a hierarchy. `/` filters. No command palette, no memorizing strings.
- **Lowercase is free.** Uppercase handles resource navigation. Every lowercase key is available for actions, filtering, and future features.
- **Fast.** Informer caches, instant startup, single binary. Ready before you finish thinking.

## Resources (v1)

Pods, Deployments, Services, ConfigMaps, Secrets, Nodes, Namespaces, Events.

The architecture should make adding resource types straightforward — each type defines its list columns, detail sections, and available actions.

## Navigation

Three mechanisms, each with a clear role:

### Uppercase hotkeys — jump to resource type

From any view, Shift+key jumps to that resource's list in the current namespace:

| Key | Resource    |
|-----|-------------|
| `P` | Pods        |
| `D` | Deployments |
| `S` | Services    |
| `C` | ConfigMaps  |
| `K` | Secrets     |
| `N` | Namespaces  |
| `O` | Nodes       |
| `E` | Events      |

No menus, no pickers, no typing. Memorize once.

### Left/right — drill down and back up

You're always somewhere in a hierarchy. Enter goes deeper, Backspace goes back. A breadcrumb in the header always shows where you are.

```
Resource list  ──Enter/l/→──>  Resource detail  ──Enter/l/→──>  Logs / Events / YAML
               <──Backspace/h/←──               <──Backspace/h/←──
```

### `/` — fuzzy filter

Opens an fzf-style filter over the current list. Type to narrow, Enter to select, Esc to cancel. Filters by name, status, or any visible column.

## Landing View

Pods in the default namespace of the current kubeconfig context. Unhealthy pods sorted to the top. You see problems first.

## List Views

Minimal columns — enough to scan, not a spreadsheet. Everything else lives in the detail view.

**Pods:**

| Column   | Notes |
|----------|-------|
| Name     | Hash suffix dimmed |
| Status   | Color-coded, one word |
| Ready    | e.g. `2/2` |
| Restarts | Count + recency if recent |
| Age      | |

**Sorting:** unhealthy first (CrashLoop, Error, Pending), then by name. This is a triage view — problems float to the top.

**Color:** red = CrashLoopBackOff/Error/Failed, yellow = Pending/Warning, default = Running/Succeeded.

Other resource types follow the same principle: 4-5 columns max, status color-coded, sorted so problems surface.

## Detail Views

Drill into a resource and see structured sections — not YAML. Each section shows only what's useful. Raw YAML is always available via `y` as an escape hatch.

**Pod detail sections:**

**Status line** — phase, node, IP, QoS class, all on one line.

**Containers** — table: name, image (short), state, restarts, reason for last restart.

**Conditions** — only non-True conditions shown (the interesting ones). If everything is fine, this section is empty.

**Recent events** — last 10, reverse-chronological, Warning events in color.

**Labels / Annotations** — collapsed by default, expand with a key.

## Log Viewer

Good enough to replace `kubectl logs` and `stern`. This is the view you'll spend the most time in — it needs to be a real tool, not an afterthought.

**Defaults:** follow mode on, `--since=5m`, line wrap on.

| Key       | Action |
|-----------|--------|
| `f`       | Toggle follow on/off |
| `space`   | Pause scrolling (buffer continues, catch up on unpause) |
| `/`       | Search — highlight matches, `n`/`N` jump between them |
| `w`       | Toggle line wrap |
| `t`       | Toggle timestamps |
| `c`       | Container picker (multi-container pods) |
| `[` / `]` | Cycle `--since` window: 1m, 5m, 15m, 1h, all |
| `Esc`     | Back to previous view |

## Header and Footer

**Header** (one line):
- Left: breadcrumb showing your location — `prod > default > pods > api-7c6...`
- Right: warning indicator if there are cluster warnings (e.g. `! 3 warnings`), otherwise empty

**Footer** (one line):
- Context-sensitive key hints for the current view — changes when you switch views
- Active filter shown if filtering

Nothing else. No mode label, no node counts, no timestamps.

## Keybindings

All navigation uses vim conventions. Uppercase = go somewhere. Lowercase = do something here.

**Always available:**

| Key | Action |
|-----|--------|
| `j` / `k` | Move down / up |
| `Enter` / `l` / `Right` | Drill in |
| `Backspace` / `h` / `Left` | Go back |
| `/` | Filter (list) or search (logs) |
| `Esc` | Cancel / close overlay |
| `?` | Help |
| `q` | Quit |
| `P` `D` `S` `C` `K` `N` `O` `E` | Jump to resource type |

**Detail view:**

| Key | Action |
|-----|--------|
| `l` | Logs |
| `e` | Events for this resource |
| `y` | Raw YAML |

**Log view:**

| Key | Action |
|-----|--------|
| `f` | Toggle follow |
| `w` | Toggle wrap |
| `t` | Toggle timestamps |
| `c` | Container picker |
| `space` | Pause / resume |
| `n` / `N` | Next / previous search match |
| `[` / `]` | Cycle since window |

## Mockups

### Pod list

```
prod > default > pods                                            ! 2 warnings

  NAME                          STATUS       READY  RESTARTS   AGE
  api-7c6c8d5f7d-x8p2k         CrashLoop    1/2    5 (10m)    2d
  worker-55c6c6f9f-9mlr        Pending      0/1    0          3m
  web-6d9f9f7b7d-2r9kq         Running      2/2    0          5d
  web-6d9f9f7b7d-kp4mn         Running      2/2    0          5d
  db-0                          Running      1/1    0          12d
  cache-redis-0                 Running      1/1    0          12d

 j/k navigate  enter detail  l logs  / filter  ? help  q quit
```

- CrashLoop is red, Pending is yellow, Running is default
- Hash suffixes dimmed in the actual TUI
- Unhealthy pods float to the top
- No grid lines, no box drawing

### Pod detail

```
prod > default > pods > api-7c6c8d5f7d-x8p2k

  Running 1/2    node: worker-03    ip: 10.244.2.15    qos: Burstable

  CONTAINERS
  NAME       IMAGE              STATE              RESTARTS  REASON
  api        myco/api:v2.3.1    Running            0
  sidecar    envoy:1.28         CrashLoopBackOff   5         OOMKilled (10m ago)

  CONDITIONS
  Ready = False              containers with unready status: [sidecar]
  ContainersReady = False

  RECENT EVENTS
  10m ago   Warning  BackOff      Back-off restarting failed container sidecar
  12m ago   Normal   Pulled       Successfully pulled image "envoy:1.28"
  15m ago   Warning  OOMKilling   Memory capped at 128Mi

 backspace back  l logs  e events  y yaml  ? help
```

### Log view

```
prod > default > pods > api-7c6c8d5f7d-x8p2k > logs [sidecar]   follow: on

  2025-06-15T12:03:01Z  Starting envoy proxy...
  2025-06-15T12:03:01Z  Loading configuration from /etc/envoy/config.yaml
  2025-06-15T12:03:02Z  Listener 0.0.0.0:8080 created
  2025-06-15T12:03:02Z  Allocating buffer pool (128Mi limit)
  2025-06-15T12:03:03Z  ERROR: buffer allocation failed: OOM
  2025-06-15T12:03:03Z  Fatal: cannot start with current memory limits

 f follow  w wrap  t timestamps  c container  / search  space pause  esc back
```

## Implementation Notes

- Go with `client-go` for Kubernetes API access
- Informer-based caches for all resource types (pods, nodes, deployments, etc.)
- Log streaming via REST client with cancelable context
- Single binary, non-root install, no runtime dependencies
- Resource types as a pluggable abstraction: each type provides list columns, detail sections, and formatting

## MVP Decisions (locked unless revised)

- **TUI framework:** Bubble Tea + Bubbles + Lip Gloss (clean layout control, fast input handling).
- **Navigation/state:** Explicit view stack (list → detail → logs/events/yaml) with a small state machine per view.
- **Resource abstraction:** `ResourceType` interface defines list columns, row render, sort order, detail sections, and available actions.
- **Filtering:** Fuzzy match over visible columns + status; no labels/annotations by default.
- **Sorting defaults:** Pods (as specified), Deployments/Services/ConfigMaps/Secrets/Nodes/Namespaces/Events = status/health first, then name.
- **Log buffer:** Ring buffer capped at 10,000 lines (drop oldest), pause does not stop ingest.
- **Namespace/context switching:** Overlay picker: `n` = namespace, `x` = context (simple list with `/` filter).
- **Events view:** Per-resource events only for MVP (global events view deferred).
- **YAML rendering:** Raw YAML (no syntax highlight) for MVP; keep it fast and simple.
- **Errors/offline:** Top-line banner for auth or connectivity errors; last good data stays visible.

## v1 Scope

**In:** core resources (pods, deployments, services, configmaps, secrets, nodes, namespaces, events), full navigation model, detail views, log viewer, context and namespace switching.

**Out:** mutating actions (delete, cordon, restart), metrics/resource usage, custom resource definitions, multi-cluster, config file persistence, plugins.
