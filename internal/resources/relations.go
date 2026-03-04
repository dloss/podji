package resources

import (
	"context"
	"fmt"
	"strings"
)

type relatedResource struct {
	namespaceScope
	name        string
	key         rune
	items       []ResourceItem
	empty       string
	logPrefix   string
	description string
	exact       bool // when true, items are computed matches — skip expandMockItems
}

func (r *relatedResource) Name() string { return r.name }
func (r *relatedResource) Key() rune    { return r.key }
func (r *relatedResource) Items() []ResourceItem {
	items := make([]ResourceItem, len(r.items))
	copy(items, r.items)
	if !r.exact {
		items = expandMockItems(items, 18)
	}
	defaultSort(items)
	return items
}
func (r *relatedResource) Sort(items []ResourceItem) { defaultSort(items) }
func (r *relatedResource) Detail(item ResourceItem) DetailData {
	status := item.Status
	if status == "" {
		status = "Healthy"
	}

	events := []string{
		"1m ago   Normal   Related   Opened from related panel",
	}
	if r.description != "" {
		events = append(events, "5m ago   Normal   Note      "+r.description)
	}

	return DetailData{
		Summary: []SummaryField{
			{Key: "status", Label: "Status", Value: status},
			{Key: "relation", Label: "Relation", Value: r.name},
			{Key: "object", Label: "Object", Value: item.Name},
		},
		Events: events,
		Labels: []string{"relation=" + strings.ReplaceAll(r.name, " ", "-")},
	}
}
func (r *relatedResource) Logs(item ResourceItem) []string {
	prefix := r.logPrefix
	if prefix == "" {
		prefix = "Related view"
	}
	return expandMockLogs([]string{
		fmt.Sprintf("%s log stream for %s", prefix, item.Name),
		"mock line: connected",
		"mock line: healthy",
	}, 50)
}
func (r *relatedResource) Events(item ResourceItem) []string {
	return []string{
		"1m ago   Normal   Related   Opened from related panel",
	}
}
func (r *relatedResource) Describe(item ResourceItem) string {
	return "Name:         " + item.Name + "\n" +
		"Namespace:    " + r.Namespace() + "\n" +
		"Relation:     " + r.name + "\n" +
		"Status:       " + item.Status + "\n" +
		"Age:          " + item.Age
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
  namespace: ` + r.Namespace() + `
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
	namespaceScope
	workload ResourceItem
	registry *Registry
}

func NewWorkloadPods(workload ResourceItem, registry *Registry) *WorkloadPods {
	w := &WorkloadPods{namespaceScope: newNamespaceScope(), workload: workload, registry: registry}
	if registry != nil {
		w.SetNamespace(registry.Namespace())
	}
	return w
}

func (w *WorkloadPods) Name() string {
	if w.workload.Kind == "CJ" {
		job := w.NewestJobName()
		return "pods (CronJob: " + w.workload.Name + ", newest job: " + job + ")"
	}
	return "pods (" + w.workload.Name + ")"
}

func (w *WorkloadPods) Key() rune { return 'P' }

func (w *WorkloadPods) TableColumns() []TableColumn {
	return namespacedColumnsFor(w.Namespace(), []TableColumn{
		{ID: "name", Name: "NAME", Width: 48, Default: true},
		{ID: "status", Name: "STATUS", Width: 12, Default: true},
		{ID: "ready", Name: "READY", Width: 7, Default: true},
		{ID: "restarts", Name: "RESTARTS", Width: 14, Default: true},
		{ID: "age", Name: "AGE", Width: 6, Default: true},
	})
}

func (w *WorkloadPods) TableRow(item ResourceItem) map[string]string {
	return map[string]string{
		"namespace": item.Namespace,
		"name":      item.Name,
		"status":    item.Status,
		"ready":     item.Ready,
		"restarts":  item.Restarts,
		"age":       item.Age,
	}
}

func (w *WorkloadPods) Items() []ResourceItem {
	// In live kube mode, never synthesize mock pods for a workload. If we
	// cannot resolve owned pods from available data, return empty and let UI
	// show an empty state / related picker instead of mixed mock data.
	if strings.TrimSpace(w.workload.UID) != "" {
		var items []ResourceItem
		if w.registry != nil && len(w.workload.Selector) > 0 {
			pods := w.registry.ByName("pods")
			if pods != nil {
				for _, pod := range pods.Items() {
					if MatchesSelector(w.workload.Selector, pod.Labels) {
						items = append(items, pod)
					}
				}
			}
		}
		return items
	}

	var items []ResourceItem
	switch w.workload.Kind {
	case "CJ":
		switch w.workload.Name {
		case "sync-reports":
			return nil
		case "nightly-backup":
			items = []ResourceItem{
				{Name: "nightly-backup-289173-7m2kq", Status: "Running", Ready: "1/1", Restarts: "0", Age: "2m"},
			}
		default:
			items = []ResourceItem{
				{Name: w.workload.Name + "-job-99211-fx8qz", Status: "Completed", Ready: "0/1", Restarts: "0", Age: "8m"},
			}
		}
	case "JOB":
		items = []ResourceItem{
			{Name: w.workload.Name + "-6l4mh", Status: "Error", Ready: "0/1", Restarts: "3", Age: "17m"},
		}
	case "DS":
		items = []ResourceItem{
			{Name: w.workload.Name + "-node-a", Status: "Running", Ready: "1/1", Restarts: "0", Age: "3d"},
			{Name: w.workload.Name + "-node-b", Status: "Running", Ready: "1/1", Restarts: "0", Age: "3d"},
			{Name: w.workload.Name + "-node-c", Status: "Pending", Ready: "0/1", Restarts: "0", Age: "4m"},
		}
	default:
		// DEP/STS: use label-selector matching when possible.
		if w.registry != nil && len(w.workload.Selector) > 0 {
			pods := w.registry.ByName("pods")
			if pods != nil {
				for _, pod := range pods.Items() {
					if MatchesSelector(w.workload.Selector, pod.Labels) {
						items = append(items, pod)
					}
				}
			}
		}
		if len(items) == 0 {
			// Fallback: generic stub pods when no registry or no selector match.
			items = []ResourceItem{
				{Name: w.workload.Name + "-7d9c7c9d4f-qwz8p", Namespace: w.workload.Namespace, Status: "Running", Ready: "2/2", Restarts: "0", Age: "1h"},
				{Name: w.workload.Name + "-7d9c7c9d4f-r52lk", Namespace: w.workload.Namespace, Status: "CrashLoop", Ready: "1/2", Restarts: "7", Age: "44m"},
			}
		}
	}
	return expandMockItems(items, 22)
}

func (w *WorkloadPods) Sort(items []ResourceItem) { defaultSort(items) }
func (w *WorkloadPods) Detail(item ResourceItem) DetailData {
	containers := []ContainerRow{
		{Name: "app", Image: "ghcr.io/example/" + w.workload.Name + ":latest", State: "Running", Restarts: "0"},
		{Name: "sidecar", Image: "busybox:stable", State: "Running", Restarts: "0"},
	}
	rawContainers := strings.TrimSpace(item.Extra["containers"])
	if rawContainers != "" {
		names := strings.Split(rawContainers, ",")
		images := strings.Split(strings.TrimSpace(item.Extra["images"]), ",")
		containers = make([]ContainerRow, 0, len(names))
		state := strings.TrimSpace(item.Status)
		if state == "" {
			state = "Unknown"
		}
		restarts := strings.TrimSpace(item.Restarts)
		if restarts == "" {
			restarts = "0"
		}
		for i := range names {
			name := strings.TrimSpace(names[i])
			if name == "" {
				continue
			}
			image := "<unknown>"
			if i < len(images) && strings.TrimSpace(images[i]) != "" {
				image = strings.TrimSpace(images[i])
			}
			containers = append(containers, ContainerRow{
				Name:     name,
				Image:    image,
				State:    state,
				Restarts: restarts,
			})
		}
		if len(containers) == 0 {
			containers = []ContainerRow{{Name: "app", Image: "<unknown>", State: state, Restarts: restarts}}
		}
	}

	return DetailData{
		Summary: []SummaryField{
			{Key: "status", Label: "Status", Value: item.Status},
			{Key: "ready", Label: "Ready", Value: item.Ready},
			{Key: "workload", Label: "Workload", Value: w.workload.Name},
		},
		Containers: containers,
		Events:     []string{"2m ago   Normal   Pulled   Pulled container image"},
	}
}
func (w *WorkloadPods) Logs(item ResourceItem) []string {
	lines, _ := w.LogsWithOptions(context.Background(), item, LogOptions{Tail: 200})
	return lines
}

func (w *WorkloadPods) LogsWithOptions(ctx context.Context, item ResourceItem, opts LogOptions) ([]string, error) {
	if w.registry != nil {
		if podRes, ok := w.registry.ByName("pods").(LogOptionsReader); ok {
			lines, err := podRes.LogsWithOptions(ctx, item, opts)
			if err == nil && len(lines) > 0 {
				return lines, nil
			}
		}
	}
	switch item.Status {
	case "CrashLoop", "Error":
		return expandMockLogs([]string{
			"2026-02-20T15:03:11Z  pod=" + item.Name + "  container=app  Starting server",
			"2026-02-20T15:03:12Z  pod=" + item.Name + "  container=app  panic: runtime error: invalid memory address or nil pointer dereference",
			"2026-02-20T15:03:12Z  pod=" + item.Name + "  container=app  goroutine 1 [running]:",
			"2026-02-20T15:03:12Z  pod=" + item.Name + "  container=app  main.run(0xc0001a6000)",
			"2026-02-20T15:03:12Z  pod=" + item.Name + "  container=app  \t/app/main.go:42 +0x1c4",
			"2026-02-20T15:03:12Z  pod=" + item.Name + "  container=app  exit status 2",
		}, 120), nil
	case "Completed":
		return expandMockLogs([]string{
			"2026-02-20T15:01:00Z  pod=" + item.Name + "  container=app  Starting job",
			"2026-02-20T15:01:04Z  pod=" + item.Name + "  container=app  Processed 1420 records",
			"2026-02-20T15:01:05Z  pod=" + item.Name + "  container=app  Done. Exiting 0.",
		}, 120), nil
	default:
		return expandMockLogs([]string{
			"2026-02-20T15:01:00Z  pod=" + item.Name + "  container=app  Booting",
			"2026-02-20T15:01:02Z  pod=" + item.Name + "  container=app  Ready",
			"2026-02-20T15:01:09Z  pod=" + item.Name + "  container=sidecar  Sync complete",
		}, 120), nil
	}
}

func (w *WorkloadPods) LogsStream(ctx context.Context, item ResourceItem, opts LogOptions, onLine func(string)) error {
	if w.registry != nil {
		if podRes, ok := w.registry.ByName("pods").(LogStreamReader); ok {
			return podRes.LogsStream(ctx, item, opts, onLine)
		}
	}
	lines, err := w.LogsWithOptions(ctx, item, opts)
	if err != nil {
		return err
	}
	for _, line := range lines {
		onLine(line)
	}
	return nil
}
func (w *WorkloadPods) Events(item ResourceItem) []string {
	if w.registry != nil {
		if podRes, ok := w.registry.ByName("pods").(*Pods); ok {
			return podRes.Events(item)
		}
	}
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
func (w *WorkloadPods) Describe(item ResourceItem) string {
	return "Name:             " + item.Name + "\n" +
		"Namespace:        " + w.Namespace() + "\n" +
		"Node:             worker-01/10.0.1.11\n" +
		"Status:           " + item.Status + "\n" +
		"IP:               10.244.1.35\n" +
		"Controlled By:    ReplicaSet/" + w.workload.Name + "-7d9c7c9d4f\n" +
		"Containers:\n" +
		"  app:\n" +
		"    Image:   ghcr.io/example/" + w.workload.Name + ":latest\n" +
		"    Port:    8080/TCP\n" +
		"    State:   Running\n" +
		"  sidecar:\n" +
		"    Image:   busybox:stable\n" +
		"    State:   Running\n" +
		"Events:\n" +
		"  Type    Reason   Age  Message\n" +
		"  ----    ------   ---  -------\n" +
		"  Normal  Pulled   2m   Pulled container image"
}
func (w *WorkloadPods) YAML(item ResourceItem) string {
	return strings.TrimSpace(`apiVersion: v1
kind: Pod
metadata:
  name: ` + item.Name + `
  namespace: ` + w.Namespace() + `
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
	if w.workload.Kind == "CJ" && w.NewestJobName() == "—" {
		return "No jobs have run for CronJob `" + w.workload.Name + "` yet."
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

func NewBackends(svc ResourceItem, registry *Registry) ResourceType {
	var pods []ResourceItem
	if registry != nil && len(svc.Selector) > 0 {
		podResource := registry.ByName("pods")
		if podResource != nil {
			for _, pod := range podResource.Items() {
				if MatchesSelector(svc.Selector, pod.Labels) {
					pods = append(pods, pod)
				}
			}
		}
	}
	return &relatedResource{
		namespaceScope: newNamespaceScope(),
		name:           "backends (" + svc.Name + ")",
		items:          pods,
		empty:          "No backends observed from EndpointSlices.",
		exact:          true,
	}
}

func NewConsumers(object string) ResourceType {
	return &relatedResource{
		namespaceScope: newNamespaceScope(),
		name:           "consumers (" + object + ")",
		items:          []ResourceItem{{Name: "api", Kind: "DEP", Status: "Healthy", Ready: "2/2", Restarts: "0", Age: "5d"}, {Name: "worker", Kind: "JOB", Status: "Progressing", Ready: "0/1", Restarts: "0", Age: "8m"}},
		empty:          "No consumers reference this object.",
	}
}

func NewMountedBy(pvc string) ResourceType {
	return &relatedResource{
		namespaceScope: newNamespaceScope(),
		name:           "mounted-by (" + pvc + ")",
		items:          []ResourceItem{{Name: "db-0", Status: "Healthy", Ready: "1/1", Restarts: "0", Age: "12d"}},
		empty:          "No pods mount this PVC.",
	}
}

func NewRelatedServices(workload ResourceItem, registry *Registry) ResourceType {
	var svcs []ResourceItem
	if registry != nil && len(workload.Selector) > 0 {
		svcResource := registry.ByName("services")
		if svcResource != nil {
			for _, svc := range svcResource.Items() {
				// A service is related if its selector matches the workload's pod labels
				// (i.e. the service's selector is a subset of the workload selector).
				if MatchesSelector(svc.Selector, workload.Selector) {
					svcs = append(svcs, svc)
				}
			}
		}
	}
	return &relatedResource{
		namespaceScope: newNamespaceScope(),
		name:           "services (" + workload.Name + ")",
		items:          svcs,
		empty:          "No related services.",
		exact:          true,
	}
}

func NewRelatedConfig(workload string) ResourceType {
	return &relatedResource{
		namespaceScope: newNamespaceScope(),
		name:           "config (" + workload + ")",
		items:          []ResourceItem{{Name: workload + "-config", Status: "Healthy", Ready: "configmap", Restarts: "-", Age: "30d"}, {Name: workload + "-secret", Status: "Healthy", Ready: "secret", Restarts: "-", Age: "30d"}},
		empty:          "No related ConfigMaps or Secrets.",
	}
}

func NewRelatedStorage(workload string) ResourceType {
	return &relatedResource{
		namespaceScope: newNamespaceScope(),
		name:           "storage (" + workload + ")",
		items:          []ResourceItem{{Name: workload + "-data", Status: "Healthy", Ready: "PVC", Restarts: "-", Age: "90d"}},
		empty:          "No related storage objects.",
	}
}

func NewJobsForCronJob(name string) ResourceType {
	return &relatedResource{
		namespaceScope: newNamespaceScope(),
		name:           "jobs (" + name + ")",
		items:          []ResourceItem{{Name: name + "-289173", Status: "Healthy", Ready: "1/1", Restarts: "-", Age: "6h"}, {Name: name + "-289172", Status: "Healthy", Ready: "1/1", Restarts: "-", Age: "30h"}},
		empty:          "No jobs found for this CronJob.",
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
		namespaceScope: newNamespaceScope(),
		name:           "owner (" + pod + ")",
		items:          []ResourceItem{{Name: workload, Kind: "DEP", Status: "Available", Ready: "2/2", Restarts: "-", Age: "14d"}},
		description:    "Owning workload for this pod",
		empty:          "No owner workload found (standalone pod).",
	}
}

func NewPodServices(pod ResourceItem, registry *Registry) ResourceType {
	var svcs []ResourceItem
	if registry != nil && len(pod.Labels) > 0 {
		svcResource := registry.ByName("services")
		if svcResource != nil {
			for _, svc := range svcResource.Items() {
				if MatchesSelector(svc.Selector, pod.Labels) {
					svcs = append(svcs, svc)
				}
			}
		}
	}
	return &relatedResource{
		namespaceScope: newNamespaceScope(),
		name:           "services (" + pod.Name + ")",
		items:          svcs,
		description:    "Services selecting this pod via label match",
		empty:          "No services select this pod.",
		exact:          true,
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
		namespaceScope: newNamespaceScope(),
		name:           "config (" + pod + ")",
		items:          []ResourceItem{{Name: workload + "-config", Status: "Healthy", Ready: "configmap", Restarts: "-", Age: "30d"}, {Name: workload + "-secret", Status: "Healthy", Ready: "secret", Restarts: "-", Age: "30d"}},
		empty:          "No ConfigMaps or Secrets mounted by this pod.",
	}
}

func NewIngressServices(ingress string) ResourceType {
	return &relatedResource{
		namespaceScope: newNamespaceScope(),
		name:           "services (" + ingress + ")",
		items:          []ResourceItem{{Name: ingress, Status: "Healthy", Ready: "ClusterIP", Restarts: "-", Age: "14d"}},
		description:    "Backend services this Ingress routes to",
		empty:          "No backend services found.",
	}
}

func NewRelatedIngresses(service string) ResourceType {
	return &relatedResource{
		namespaceScope: newNamespaceScope(),
		name:           "ingresses (" + service + ")",
		items:          []ResourceItem{{Name: service, Status: "Healthy", Ready: service + ".example.com", Restarts: "-", Age: "14d"}},
		description:    "Ingresses exposing this service",
		empty:          "No Ingresses route to this service.",
	}
}

type NodePods struct {
	namespaceScope
	node string
}

func NewNodePods(node string) *NodePods {
	return &NodePods{namespaceScope: newNamespaceScope(), node: node}
}

func (n *NodePods) Name() string { return "pods (node: " + n.node + ")" }
func (n *NodePods) Key() rune    { return 'P' }

func (n *NodePods) Items() []ResourceItem {
	switch n.node {
	case "worker-01":
		return []ResourceItem{
			{Name: "api-gateway-7d9c7c9d4f-qwz8p", Status: "Running", Ready: "2/2", Restarts: "0", Age: "1h"},
			{Name: "auth-service-6c7e8b-xk2lp", Status: "Running", Ready: "2/2", Restarts: "0", Age: "21d"},
			{Name: "payment-service-6b8d4f-r52lk", Status: "CrashLoop", Ready: "1/2", Restarts: "7", Age: "44m"},
			{Name: "prometheus-0", Status: "Running", Ready: "1/1", Restarts: "0", Age: "30d"},
		}
	case "worker-02":
		return []ResourceItem{
			{Name: "frontend-7d9c7c9d4f-m8nqp", Status: "Running", Ready: "2/2", Restarts: "0", Age: "7d"},
			{Name: "notification-worker-5c9f7b-lz3wx", Status: "Running", Ready: "1/1", Restarts: "0", Age: "10d"},
			{Name: "redis-0", Status: "Running", Ready: "1/1", Restarts: "0", Age: "45d"},
		}
	case "worker-03":
		return []ResourceItem{
			{Name: "search-indexer-5c9f7b-p7vhn", Status: "Progressing", Ready: "2/3", Restarts: "0", Age: "45m"},
			{Name: "user-service-7d9c7c9d4f-b4jks", Status: "Running", Ready: "2/2", Restarts: "0", Age: "3d"},
			{Name: "grafana-59d8f9b4c6-7xkpz", Status: "Running", Ready: "1/1", Restarts: "0", Age: "30d"},
		}
	case "worker-04":
		return []ResourceItem{
			{Name: "search-indexer-5c9f7b-m8nqp", Status: "Unknown", Ready: "0/1", Restarts: "3", Age: "30d"},
		}
	default:
		return []ResourceItem{
			{Name: n.node + "-system-pod", Status: "Running", Ready: "1/1", Restarts: "0", Age: "180d"},
		}
	}
}

func (n *NodePods) Sort(items []ResourceItem) { problemSort(items, false) }

func (n *NodePods) Detail(item ResourceItem) DetailData {
	return DetailData{
		Summary: []SummaryField{
			{Key: "status", Label: "Status", Value: item.Status},
			{Key: "ready", Label: "Ready", Value: item.Ready},
			{Key: "node", Label: "Node", Value: n.node},
		},
		Containers: []ContainerRow{
			{Name: "app", Image: "ghcr.io/example/" + item.Name + ":latest", State: "Running", Restarts: "0"},
		},
		Events: []string{"2m ago   Normal   Pulled   Pulled container image"},
	}
}

func (n *NodePods) Logs(item ResourceItem) []string {
	return []string{
		"2026-02-20T15:01:00Z  pod=" + item.Name + "  container=app  Booting",
		"2026-02-20T15:01:02Z  pod=" + item.Name + "  container=app  Ready",
	}
}

func (n *NodePods) Events(item ResourceItem) []string {
	return []string{"2m ago   Normal   Scheduled   Assigned to node " + n.node}
}

func (n *NodePods) Describe(item ResourceItem) string {
	return "Name:             " + item.Name + "\n" +
		"Namespace:        " + n.Namespace() + "\n" +
		"Node:             " + n.node + "\n" +
		"Status:           " + item.Status + "\n" +
		"Controlled By:    ReplicaSet/" + item.Name
}

func (n *NodePods) YAML(item ResourceItem) string {
	return strings.TrimSpace(`apiVersion: v1
kind: Pod
metadata:
  name: ` + item.Name + `
  namespace: ` + n.Namespace() + `
spec:
  nodeName: ` + n.node + `
  containers:
  - name: app
    image: ghcr.io/example/` + item.Name + `:latest
status:
  phase: ` + item.Status)
}

func (n *NodePods) EmptyMessage(filtered bool, filter string) string {
	if filtered {
		return "No pods match `" + filter + "`."
	}
	return "No pods found on node `" + n.node + "`."
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
		namespaceScope: newNamespaceScope(),
		name:           "storage (" + pod + ")",
		items:          []ResourceItem{{Name: workload + "-data", Status: "Bound", Ready: "PVC", Restarts: "-", Age: "90d"}},
		empty:          "No PVCs mounted by this pod.",
	}
}
