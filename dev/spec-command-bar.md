# Command Bar

A Vim-style `:` command bar for fast targeted navigation.

## Motivation

Three gaps exist in current navigation that hotkeys and `/` filter cannot fill:

**You always navigate through lists, even when you know the name.** In incident response, you arrive with a pod name from an alert or Slack message. The current path is: press `P` → scroll or filter the list → find the pod → drill to logs. Four steps of friction when the destination is already known.

**Label-based filtering is not expressible.** The `/` filter matches visible text. Labels are not visible. Kubernetes operators think in label selectors — `app=frontend`, `env=prod` — but there is no way to express this in the current UI. This is how `kubectl get pods -l app=frontend` works; Podji has no equivalent.

**No path to cross-resource health queries.** "What's broken right now?" requires visiting workloads, pods, and other resource lists separately. There is no single view that answers it.

The command bar addresses all three. It is a navigation tool, not an execution tool. The execute menu owns mutations; the command bar owns *getting to the right place*.

## Activation

`:` opens the command bar from any list view. The bar appears at the bottom of the screen, replacing the footer. The current list remains visible behind it. `Esc` dismisses without side effects.

The bar is managed by `app.go` as an overlay, consistent with the `overlaypicker` pattern from Phase 1.

## Command Vocabulary

### 1. Named resource navigation

```
:<kind> [name-fragment] [subview]
```

`<kind>` accepts kubectl shortnames and full names:

| Shortname | Full name | Hotkey equivalent |
|---|---|---|
| `po` | `pods` | `P` |
| `deploy` | `deployments` | `D` |
| `svc` | `services` | `S` |
| `cm` | `configmaps` | `C` |
| `secret`, `sec` | `secrets` | `K` |
| `node` | `nodes` | `O` |
| `ing` | `ingresses` | `I` |
| `pvc` | `pvcs` | `V` |
| `ev` | `events` | `E` |
| `ns` | `namespaces` | — |

`<subview>` tokens: `logs`, `yaml`, `events`, `describe`

A single match with no subview lands on the resource's detail view (structured panel with status, containers, conditions). `describe` navigates to the raw text describe view instead.

**Matching rules:**

- No name fragment → jump to that resource's list (same as pressing the hotkey). If the current view is already that list, the command bar closes and nothing changes (no-op).
- One match → jump directly to that resource's detail view; apply subview if given.
- Multiple matches → push a filtered list; any subview token is discarded. Navigate to the subview manually from the filtered list.
- No matches → show "no match" inline, leave current view unchanged.
- Empty command (`:` then Enter) → no-op; command bar closes.

Name matching is prefix-then-substring against the Name field, scoped to the active namespace.

**Token disambiguation:** `events` is both a kind shortname (full name for `ev`) and a subview token. Interpretation is position-dependent: the first token after `:` is always parsed as a kind; `events` in third position is always a subview token. No ambiguity arises.

K9s users can use `:po`, `:deploy`, `:svc` etc. exactly as they do today — the no-name-fragment case is compatible by design. The name fragment and subview suffix are extensions that existing muscle memory naturally grows into.

**Examples:**

```
:po                      → pods list (same as P)
:deploy                  → deployments list (same as D)
:po crashloop            → pods filtered to "crashloop"; single match jumps there
:po my-app logs          → logs for my-app pod (direct if single match)
:deploy api yaml         → YAML for api deployment
:po my-app events        → events for my-app pod
:cm app-config           → configmap named app-config
```

This is the primary use case. In incident response, a user arrives with a pod name and can reach logs in one command instead of four steps.

### 2. Computed health views

Two built-in cross-resource queries. No parameters.

**`:unhealthy`**

Shows all resources not in a healthy state, across all resource types in the active namespace. Includes:

- Pods: phase not Running or Succeeded (e.g. CrashLoopBackOff, Pending, Error, OOMKilled)
- Deployments/StatefulSets/DaemonSets: available replicas < desired replicas
- PVCs: phase not Bound

Sorted by severity (Failed before Degraded before Progressing), then by creation time descending (most recently created first). Displays resource kind alongside name since the list is heterogeneous. Shows namespace column when in all-namespaces mode, consistent with `:restarts`.

**`:restarts`**

Shows all pods with at least one restart, sorted by restart count descending. Scoped to active namespace; shows namespace column when in all-namespaces mode.

Both views push a synthetic list onto the navigation stack. `Enter` drills into the selected item through the normal stack model. `Backspace` returns to the previous view.

### 3. Label selector filtering

```
:<kind> <key>=<value>[,<key>=<value>...]
```

Filters a resource list by label selector, using AND semantics matching `kubectl -l`.

**Examples:**

```
:po app=frontend               → pods with label app=frontend
:po app=frontend,env=prod      → pods matching both labels
:deploy tier=backend           → deployments with label tier=backend
:svc component=cache           → services with that label
```

The command pushes a filtered list view. The breadcrumb reflects the selector, e.g. `pods: app=frontend`. From there, normal navigation applies.

Combining a label selector with a subview token (e.g. `:po app=frontend logs`) is not supported and produces an "unknown command" error inline.

Label filtering requires `Labels map[string]string` on `ResourceItem` — see Data Model below.

## UX Behaviour

### Rendering

The bar replaces the footer line. Prompt is `:`, consistent with Vim convention:

```
: po my-app log█
```

Inline suggestion appears greyed to the right of the cursor as the longest unambiguous completion. `Tab` accepts it.

When input is longer than the available width, the display scrolls left to keep the cursor visible. No wrapping.

`:` only activates the command bar from list views. In detail, logs, YAML, describe, and events views the key is not forwarded to the command bar.

### Autocomplete

Suggestions are offered at each position in the grammar:

| Position | Suggestion source |
|---|---|
| After `:` | Built-in command names + kind shortnames/fullnames matching typed prefix |
| After `:<kind> ` | Resource names in current scope, prefix match |
| After `:<kind> <name> ` | Subview tokens: `logs`, `yaml`, `events`, `describe` |

Suggestions are single-token completions, not full-line. `Tab` completes the current token. No popup list; inline only.

### History

`Up`/`Down` in the command bar navigate session command history. History is not persisted across sessions.

### Error display

Errors appear inline in the bar, not as a banner or modal:

```
: po nonexistent                                     no match
: :badcommand                                  unknown command
```

The current view is not disturbed. The user can edit the command or press `Esc`.

## Navigation Stack Behaviour

Commands push onto the stack in the same way as manual navigation. `:po my-app logs` produces a stack equivalent to navigating `pods → my-app → logs` by hand. `Backspace` unwinds it step by step.

Computed views (`:unhealthy`, `:restarts`) push a single synthetic list view. The breadcrumb is the command name.

Label-filtered views push a filtered resource list. The breadcrumb is `<kind>: <selector>`.

This means command-bar navigation is fully reversible and composable with manual navigation. There is no special "command bar mode" to exit; it is just navigation.

## Data Model

### Phase 1 (no data model changes)

Named navigation and computed views require no changes to `ResourceItem`. Named navigation filters by the existing `Name` field. Computed views need a cross-resource query function — analogous to `workloads.go` aggregating across kinds — but no new fields.

New function needed in `internal/resources/`:

```go
// UnhealthyItems returns all items across resource types that are not in a healthy state.
// Results are sorted by severity then age.
func UnhealthyItems() []ResourceItem

// PodsByRestarts returns all pods with restart count > 0, sorted by restart count descending.
func PodsByRestarts() []ResourceItem
```

### Phase 2 (label selector support)

Add to `ResourceItem`:

```go
Labels map[string]string
```

All resource implementations must populate `Labels`. Mock data needs representative label sets (at minimum `app`, `env`, `tier`, `component`).

New helper:

```go
// MatchesSelector reports whether item labels satisfy a comma-separated key=value selector.
func MatchesSelector(item ResourceItem, selector string) bool
```

Selector parsing: split on `,`, each part splits on first `=`, require exact match on both key and value. No operators (`!=`, `in`, `notin`) in v1.

## Out of Scope

**Kubectl passthrough.** `:delete`, `:restart`, `:apply` are not command bar commands. Mutations belong in the execute menu. The command bar does not shell out.

**CRD navigation.** `:certificates`, `:prometheusrules` etc. are deferred until CRD support lands. The `A` resource browser remains the path to custom resources.

**Regex in name fragments.** Prefix-then-substring matching is sufficient. Regex adds implementation complexity and is rarely needed in practice.

**Cross-namespace label queries.** Label filtering is scoped to the active namespace in v1. All-namespace label queries are a natural extension but are deferred.

**Bookmarks.** The `:save`/`:go` concept (noted in CONCEPT.md Future Extensions) could be delivered via the command bar but is a separate feature.

**Namespace/context switching.** `:ns kube-system` and `:ctx prod` are possible future additions, but the Phase 1 overlay pickers (`N`/`X`) already handle this well. Avoid duplication until there is a clear reason to prefer the command bar over the pickers.
