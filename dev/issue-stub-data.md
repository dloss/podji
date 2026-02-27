# Issue: Stub Data Quality

Four separate data inconsistencies in the stub layer make the TUI misleading during development and demos.

## 1. Hardcoded `"default"` in empty/error state messages

**File:** `internal/resources/workloads.go:150,169,171`

```go
// line 150
return "Access denied: cannot list workloads in namespace `default` ..."

// line 169
return "No workloads found in namespace `default`. Switch namespace or clear filter."

// line 171
return "No workloads found in namespace `default`."
```

All three messages hardcode the string `"default"` regardless of the active namespace. After switching to `staging`, the forbidden/empty messages still say `default`.

### Fix

Replace the literal with `ActiveNamespace`:

```go
return "Access denied: cannot list workloads in namespace `" + ActiveNamespace + "` ..."
return "No workloads found in namespace `" + ActiveNamespace + "`. Switch namespace or clear filter."
return "No workloads found in namespace `" + ActiveNamespace + "`."
```

The same guard applies to the `Banner()` method (line 150), which also hardcodes `default` in the access-denied message.

---

## 2. Log stub doesn't vary by pod status

**File:** `internal/resources/relations.go` — `WorkloadPods.Logs()`

```go
func (w *WorkloadPods) Logs(item ResourceItem) []string {
    return []string{
        "2026-02-20T15:01:00Z  pod=" + item.Name + "  container=app  Booting",
        "2026-02-20T15:01:02Z  pod=" + item.Name + "  container=app  Ready",
        "2026-02-20T15:01:09Z  pod=" + item.Name + "  container=sidecar  Sync complete",
    }
}
```

A CrashLooping pod (`api-7d9c7c9d4f-r52lk`, 7 restarts) shows the same three "Booting / Ready / Sync complete" lines as a healthy pod. This makes it impossible to demo the crashloop debugging flow — a core use case of the tool.

### Fix

Branch on `item.Status` (and optionally `item.Restarts`):

```go
func (w *WorkloadPods) Logs(item ResourceItem) []string {
    switch item.Status {
    case "CrashLoop", "Error":
        return []string{
            "2026-02-20T15:03:11Z  pod=" + item.Name + "  container=app  Starting server",
            "2026-02-20T15:03:12Z  pod=" + item.Name + "  container=app  panic: runtime error: invalid memory address or nil pointer dereference",
            "2026-02-20T15:03:12Z  pod=" + item.Name + "  container=app  goroutine 1 [running]:",
            "2026-02-20T15:03:12Z  pod=" + item.Name + "  container=app  main.run(0xc0001a6000)",
            "2026-02-20T15:03:12Z  pod=" + item.Name + "  container=app  \t/app/main.go:42 +0x1c4",
            "2026-02-20T15:03:12Z  pod=" + item.Name + "  container=app  exit status 2",
        }
    case "Completed":
        return []string{
            "2026-02-20T15:01:00Z  pod=" + item.Name + "  container=app  Starting job",
            "2026-02-20T15:01:04Z  pod=" + item.Name + "  container=app  Processed 1420 records",
            "2026-02-20T15:01:05Z  pod=" + item.Name + "  container=app  Done. Exiting 0.",
        }
    default:
        return []string{
            "2026-02-20T15:01:00Z  pod=" + item.Name + "  container=app  Booting",
            "2026-02-20T15:01:02Z  pod=" + item.Name + "  container=app  Ready",
            "2026-02-20T15:01:09Z  pod=" + item.Name + "  container=sidecar  Sync complete",
        }
    }
}
```

---

## 3. Event stub too thin for failing pods

**File:** `internal/resources/relations.go` — `WorkloadPods.Events()`

```go
func (w *WorkloadPods) Events(item ResourceItem) []string {
    return []string{"2m ago   Normal   Scheduled   Assigned to node worker-01"}
}
```

`seed-users` (Failed, 3 restarts) and `api-...-r52lk` (CrashLoop, 7 restarts) both show a single bland "Scheduled" event. In a real cluster these would have BackOff, OOMKilled, or Error events that explain what's going wrong.

### Fix

Branch on status:

```go
func (w *WorkloadPods) Events(item ResourceItem) []string {
    base := "5m ago   Normal    Scheduled    Assigned to node worker-01"
    switch item.Status {
    case "CrashLoop":
        return []string{
            base,
            "5m ago   Normal    Pulled       Pulled container image successfully",
            "5m ago   Normal    Started      Started container app",
            "4m ago   Warning   BackOff      Back-off restarting failed container app in pod " + item.Name,
            "4m ago   Warning   BackOff      Back-off restarting failed container app in pod " + item.Name,
        }
    case "Error":
        return []string{
            base,
            "18m ago  Normal    Pulled       Pulled container image successfully",
            "18m ago  Normal    Started      Started container app",
            "17m ago  Warning   Failed       Error: failed to create containerd task: " + item.Name + ": exit status 1",
            "3m ago   Warning   BackOff      Back-off restarting failed container app in pod " + item.Name,
        }
    default:
        return []string{base}
    }
}
```

---

## 4. YAML shows `replicas: 2` but list shows `2/3`

**File:** `internal/resources/workloads.go:242`

The Deployment YAML template hardcodes `replicas: 2` for all deployments:

```go
spec := `  replicas: 2
  selector:
    matchLabels:
      app: ` + item.Name + `
```

But in the default namespace, `api` has `Ready: "2/3"` — implying desired replicas is 3. The YAML and the list are inconsistent.

### Fix

Derive replicas from `item.Ready`. The `Ready` field is formatted as `"ready/desired"` for deployments:

```go
desired := "2"
if parts := strings.SplitN(item.Ready, "/", 2); len(parts) == 2 {
    desired = strings.TrimSpace(parts[1])
}
spec := `  replicas: ` + desired + `
  selector:
    matchLabels:
      app: ` + item.Name
```

This makes `api`'s YAML show `replicas: 3`, consistent with the list view.
