package resources

import (
	"fmt"
	"strings"
)

type relatedResource struct {
	name        string
	key         rune
	items       []ResourceItem
	empty       string
	logPrefix   string
	statusLine  string
	description string
}

func (r *relatedResource) Name() string { return r.name }
func (r *relatedResource) Key() rune    { return r.key }
func (r *relatedResource) Items() []ResourceItem {
	items := make([]ResourceItem, len(r.items))
	copy(items, r.items)
	defaultSort(items)
	return items
}
func (r *relatedResource) Sort(items []ResourceItem) { defaultSort(items) }
func (r *relatedResource) Detail(item ResourceItem) DetailData {
	line := r.statusLine
	if line == "" {
		line = "Healthy    relation: " + r.name + "    object: " + item.Name
	}

	events := []string{
		"1m ago   Normal   Related   Opened from related panel",
	}
	if r.description != "" {
		events = append(events, "5m ago   Normal   Note      "+r.description)
	}

	return DetailData{
		StatusLine: line,
		Events:     events,
		Labels:     []string{"relation=" + strings.ReplaceAll(r.name, " ", "-")},
	}
}
func (r *relatedResource) Logs(item ResourceItem) []string {
	prefix := r.logPrefix
	if prefix == "" {
		prefix = "Related view"
	}
	return []string{
		fmt.Sprintf("%s log stream for %s", prefix, item.Name),
		"mock line: connected",
		"mock line: healthy",
	}
}
func (r *relatedResource) Events(item ResourceItem) []string {
	return []string{
		"1m ago   Normal   Related   Opened from related panel",
	}
}
func (r *relatedResource) YAML(item ResourceItem) string {
	kind := "Pod"
	apiVersion := "v1"
	extraSpec := ""

	switch {
	case strings.HasPrefix(r.name, "backends"):
		kind = "Pod"
		extraSpec = `
  nodeName: worker-01
  containers:
  - name: app
    image: ghcr.io/example/` + item.Name + `:latest
    ports:
    - containerPort: 8080
status:
  phase: ` + item.Status + `
  podIP: 10.244.1.22`
	case strings.HasPrefix(r.name, "consumers"):
		apiVersion = "apps/v1"
		kind = "Deployment"
		if item.Kind == "JOB" {
			apiVersion = "batch/v1"
			kind = "Job"
		}
		extraSpec = `
  replicas: 2
  selector:
    matchLabels:
      app: ` + item.Name
	case strings.HasPrefix(r.name, "services"):
		kind = "Service"
		extraSpec = `
  type: ClusterIP
  selector:
    app: ` + item.Name + `
  ports:
  - port: 80
    targetPort: 8080`
	case strings.HasPrefix(r.name, "config"):
		if strings.HasSuffix(item.Name, "-secret") {
			kind = "Secret"
			extraSpec = `
type: Opaque
data:
  username: <redacted>
  password: <redacted>`
		} else {
			kind = "ConfigMap"
			extraSpec = `
data:
  config.yaml: |
    server:
      port: 8080`
		}
	case strings.HasPrefix(r.name, "storage"):
		kind = "PersistentVolumeClaim"
		extraSpec = `
  accessModes:
  - ReadWriteOnce
  resources:
    requests:
      storage: 10Gi
  storageClassName: gp3
status:
  phase: Bound`
	case strings.HasPrefix(r.name, "jobs"):
		apiVersion = "batch/v1"
		kind = "Job"
		extraSpec = `
  completions: 1
  parallelism: 1
  template:
    spec:
      restartPolicy: OnFailure
      containers:
      - name: job
        image: ghcr.io/example/` + item.Name + `:latest`
	case strings.HasPrefix(r.name, "owner"):
		apiVersion = "apps/v1"
		kind = "Deployment"
		extraSpec = `
  replicas: 2
  selector:
    matchLabels:
      app: ` + item.Name
	case strings.HasPrefix(r.name, "mounted-by"):
		kind = "Pod"
		extraSpec = `
  containers:
  - name: app
    volumeMounts:
    - name: data
      mountPath: /var/lib/data`
	}

	return strings.TrimSpace(`apiVersion: ` + apiVersion + `
kind: ` + kind + `
metadata:
  name: ` + item.Name + `
  namespace: ` + ActiveNamespace + `
  labels:
    app: ` + item.Name + `
spec:` + extraSpec)
}
func (r *relatedResource) EmptyMessage(filtered bool, filter string) string {
	if filtered {
		return "No related items match `" + filter + "`."
	}
	if r.empty != "" {
		return r.empty
	}
	return "No related items found."
}

type WorkloadPods struct {
	workload ResourceItem
}

func NewWorkloadPods(workload ResourceItem) *WorkloadPods {
	return &WorkloadPods{workload: workload}
}

func (w *WorkloadPods) Name() string {
	return "pods (" + w.workload.Name + ")"
}

func (w *WorkloadPods) Key() rune { return 'P' }

func (w *WorkloadPods) Items() []ResourceItem {
	switch w.workload.Kind {
	case "CJ":
		switch w.workload.Name {
		case "sync-reports":
			return nil
		case "nightly-backup":
			return []ResourceItem{
				{Name: "nightly-backup-289173-7m2kq", Status: "Running", Ready: "1/1", Restarts: "0", Age: "2m"},
			}
		default:
			return []ResourceItem{
				{Name: w.workload.Name + "-job-99211-fx8qz", Status: "Completed", Ready: "0/1", Restarts: "0", Age: "8m"},
			}
		}
	case "JOB":
		return []ResourceItem{
			{Name: w.workload.Name + "-6l4mh", Status: "Error", Ready: "0/1", Restarts: "3", Age: "17m"},
		}
	case "DS":
		return []ResourceItem{
			{Name: w.workload.Name + "-node-a", Status: "Running", Ready: "1/1", Restarts: "0", Age: "3d"},
			{Name: w.workload.Name + "-node-b", Status: "Running", Ready: "1/1", Restarts: "0", Age: "3d"},
			{Name: w.workload.Name + "-node-c", Status: "Pending", Ready: "0/1", Restarts: "0", Age: "4m"},
		}
	default:
		return []ResourceItem{
			{Name: w.workload.Name + "-7d9c7c9d4f-qwz8p", Status: "Running", Ready: "2/2", Restarts: "0", Age: "1h"},
			{Name: w.workload.Name + "-7d9c7c9d4f-r52lk", Status: "CrashLoop", Ready: "1/2", Restarts: "7", Age: "44m"},
		}
	}
}

func (w *WorkloadPods) Sort(items []ResourceItem) { defaultSort(items) }
func (w *WorkloadPods) Detail(item ResourceItem) DetailData {
	return DetailData{
		StatusLine: item.Status + " " + item.Ready + "    workload: " + w.workload.Name,
		Containers: []ContainerRow{
			{Name: "app", Image: "ghcr.io/example/" + w.workload.Name + ":latest", State: "Running", Restarts: "0"},
			{Name: "sidecar", Image: "busybox:stable", State: "Running", Restarts: "0"},
		},
		Events: []string{"2m ago   Normal   Pulled   Pulled container image"},
	}
}
func (w *WorkloadPods) Logs(item ResourceItem) []string {
	return []string{
		"2026-02-20T15:01:00Z  pod=" + item.Name + "  container=app  Booting",
		"2026-02-20T15:01:02Z  pod=" + item.Name + "  container=app  Ready",
		"2026-02-20T15:01:09Z  pod=" + item.Name + "  container=sidecar  Sync complete",
	}
}
func (w *WorkloadPods) Events(item ResourceItem) []string {
	return []string{"2m ago   Normal   Scheduled   Assigned to node worker-01"}
}
func (w *WorkloadPods) YAML(item ResourceItem) string {
	return strings.TrimSpace(`apiVersion: v1
kind: Pod
metadata:
  name: ` + item.Name + `
  namespace: ` + ActiveNamespace + `
  labels:
    app: ` + w.workload.Name + `
    pod-template-hash: 7d9c7c9d4f
  ownerReferences:
  - apiVersion: apps/v1
    kind: ReplicaSet
    name: ` + w.workload.Name + `-7d9c7c9d4f
spec:
  nodeName: worker-01
  serviceAccountName: default
  containers:
  - name: app
    image: ghcr.io/example/` + w.workload.Name + `:latest
    ports:
    - containerPort: 8080
    resources:
      requests:
        cpu: 250m
        memory: 256Mi
      limits:
        cpu: "1"
        memory: 512Mi
  - name: sidecar
    image: busybox:stable
status:
  phase: ` + item.Status + `
  podIP: 10.244.1.35
  hostIP: 10.0.1.11
  conditions:
  - type: Ready
    status: "True"
  containerStatuses:
  - name: app
    ready: true
    restartCount: 0
    image: ghcr.io/example/` + w.workload.Name + `:latest
  - name: sidecar
    ready: true
    restartCount: 0
    image: busybox:stable`)
}
func (w *WorkloadPods) EmptyMessage(filtered bool, filter string) string {
	if filtered {
		return "No pods match `" + filter + "`."
	}

	if w.workload.Kind == "CJ" && w.workload.Name == "sync-reports" {
		return "No pods found for workload `sync-reports`. Hint: press r to view Related (Jobs, Events, Config, Network)."
	}
	return "No pods found for workload `" + w.workload.Name + "`."
}

func (w *WorkloadPods) NewestJobName() string {
	if w.workload.Kind != "CJ" {
		return ""
	}
	switch w.workload.Name {
	case "sync-reports":
		return "—"
	case "nightly-backup":
		return "nightly-backup-289173"
	default:
		return w.workload.Name + "-99211"
	}
}

func NewBackends(service string) ResourceType {
	return &relatedResource{
		name:        "backends (" + service + ")",
		items:       []ResourceItem{{Name: service + "-api-7d9c7", Status: "Healthy", Ready: "1/1", Restarts: "0", Age: "2h"}, {Name: service + "-api-7d9c7-k4mxp", Status: "Warning", Ready: "0/1", Restarts: "1", Age: "20m"}},
		description: "Pods not Ready or port mismatch",
		empty:       "No backends observed from EndpointSlices.",
	}
}

func NewConsumers(object string) ResourceType {
	return &relatedResource{
		name:  "consumers (" + object + ")",
		items: []ResourceItem{{Name: "api", Kind: "DEP", Status: "Healthy", Ready: "2/2", Restarts: "0", Age: "5d"}, {Name: "worker", Kind: "JOB", Status: "Progressing", Ready: "0/1", Restarts: "0", Age: "8m"}},
		empty: "No consumers reference this object.",
	}
}

func NewMountedBy(pvc string) ResourceType {
	return &relatedResource{
		name:  "mounted-by (" + pvc + ")",
		items: []ResourceItem{{Name: "db-0", Status: "Healthy", Ready: "1/1", Restarts: "0", Age: "12d"}},
		empty: "No pods mount this PVC.",
	}
}

func NewRelatedServices(workload string) ResourceType {
	return &relatedResource{
		name:  "services (" + workload + ")",
		items: []ResourceItem{{Name: workload, Status: "Healthy", Ready: "3 backends", Restarts: "-", Age: "20d"}},
		empty: "No related services.",
	}
}

func NewRelatedConfig(workload string) ResourceType {
	return &relatedResource{
		name:  "config (" + workload + ")",
		items: []ResourceItem{{Name: workload + "-config", Status: "Healthy", Ready: "configmap", Restarts: "-", Age: "30d"}, {Name: workload + "-secret", Status: "Healthy", Ready: "secret", Restarts: "-", Age: "30d"}},
		empty: "No related ConfigMaps or Secrets.",
	}
}

func NewRelatedStorage(workload string) ResourceType {
	return &relatedResource{
		name:  "storage (" + workload + ")",
		items: []ResourceItem{{Name: workload + "-data", Status: "Healthy", Ready: "PVC", Restarts: "-", Age: "90d"}},
		empty: "No related storage objects.",
	}
}

func NewJobsForCronJob(name string) ResourceType {
	return &relatedResource{
		name:  "jobs (" + name + ")",
		items: []ResourceItem{{Name: name + "-289173", Status: "Healthy", Ready: "1/1", Restarts: "-", Age: "6h"}, {Name: name + "-289172", Status: "Healthy", Ready: "1/1", Restarts: "-", Age: "30h"}},
		empty: "No jobs found for this CronJob.",
	}
}

func NewPodOwner(pod string) ResourceType {
	// Derive workload name by stripping the pod hash suffix.
	workload := pod
	if idx := strings.LastIndex(pod, "-"); idx > 0 {
		prefix := pod[:idx]
		if idx2 := strings.LastIndex(prefix, "-"); idx2 > 0 {
			workload = prefix[:idx2]
		} else {
			workload = prefix
		}
	}
	return &relatedResource{
		name:        "owner (" + pod + ")",
		items:       []ResourceItem{{Name: workload, Kind: "DEP", Status: "Available", Ready: "2/2", Restarts: "-", Age: "14d"}},
		description: "Owning workload for this pod",
		empty:       "No owner workload found (standalone pod).",
	}
}

func NewPodServices(pod string) ResourceType {
	workload := pod
	if idx := strings.LastIndex(pod, "-"); idx > 0 {
		prefix := pod[:idx]
		if idx2 := strings.LastIndex(prefix, "-"); idx2 > 0 {
			workload = prefix[:idx2]
		} else {
			workload = prefix
		}
	}
	return &relatedResource{
		name:        "services (" + pod + ")",
		items:       []ResourceItem{{Name: workload + "-svc", Status: "Healthy", Ready: "ClusterIP", Restarts: "-", Age: "14d"}},
		description: "Services selecting this pod via label match",
		empty:       "No services select this pod.",
	}
}

func NewPodConfig(pod string) ResourceType {
	workload := pod
	if idx := strings.LastIndex(pod, "-"); idx > 0 {
		prefix := pod[:idx]
		if idx2 := strings.LastIndex(prefix, "-"); idx2 > 0 {
			workload = prefix[:idx2]
		} else {
			workload = prefix
		}
	}
	return &relatedResource{
		name:  "config (" + pod + ")",
		items: []ResourceItem{{Name: workload + "-config", Status: "Healthy", Ready: "configmap", Restarts: "-", Age: "30d"}, {Name: workload + "-secret", Status: "Healthy", Ready: "secret", Restarts: "-", Age: "30d"}},
		empty: "No ConfigMaps or Secrets mounted by this pod.",
	}
}

func NewPodStorage(pod string) ResourceType {
	workload := pod
	if idx := strings.LastIndex(pod, "-"); idx > 0 {
		prefix := pod[:idx]
		if idx2 := strings.LastIndex(prefix, "-"); idx2 > 0 {
			workload = prefix[:idx2]
		} else {
			workload = prefix
		}
	}
	return &relatedResource{
		name:  "storage (" + pod + ")",
		items: []ResourceItem{{Name: workload + "-data", Status: "Bound", Ready: "PVC", Restarts: "-", Age: "90d"}},
		empty: "No PVCs mounted by this pod.",
	}
}
