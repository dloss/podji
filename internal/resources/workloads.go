package resources

import (
	"os"
	"sort"
	"strings"
)

type Workloads struct {
	namespaceScope
	sortMode string
	sortDesc bool
	scenario string
}

func NewWorkloads() *Workloads {
	scenario := os.Getenv("PODJI_SCENARIO")
	switch scenario {
	case "empty", "forbidden", "partial", "offline":
		// valid
	default:
		scenario = "normal"
	}
	return &Workloads{namespaceScope: newNamespaceScope(), sortMode: "name", scenario: scenario}
}

func (w *Workloads) Name() string { return "workloads" }
func (w *Workloads) Key() rune    { return 'W' }

func (w *Workloads) Items() []ResourceItem {
	if w.scenario == "forbidden" {
		return nil
	}

	var items []ResourceItem
	if w.Namespace() == AllNamespaces {
		items = allNamespaceItems(workloadItemsForNamespace)
	} else {
		items = workloadItemsForNamespace(w.Namespace())
		switch w.scenario {
		case "partial":
			if len(items) > 4 {
				items = items[:4]
			}
		case "empty":
			items = nil
		}
		if w.scenario == "normal" {
			items = expandMockItems(items, 40)
		}
	}
	w.Sort(items)
	return items
}

func workloadItemsForNamespace(ns string) []ResourceItem {
	switch ns {
	case "production":
		return []ResourceItem{
			{Name: "api", Kind: "DEP", Ready: "3/3", Status: "Healthy", Restarts: "0", Age: "14d", Selector: map[string]string{"app": "api"}},
			{Name: "frontend", Kind: "DEP", Ready: "4/4", Status: "Healthy", Restarts: "0", Age: "7d", Selector: map[string]string{"app": "frontend"}},
			{Name: "worker", Kind: "DEP", Ready: "2/2", Status: "Healthy", Restarts: "0", Age: "12d", Selector: map[string]string{"app": "worker"}},
			{Name: "db", Kind: "STS", Ready: "3/3", Status: "Healthy", Restarts: "0", Age: "30d", Selector: map[string]string{"app": "db"}},
			{Name: "cache", Kind: "STS", Ready: "2/2", Status: "Healthy", Restarts: "0", Age: "30d", Selector: map[string]string{"app": "cache"}},
			{Name: "nightly-backup", Kind: "CJ", Ready: "Last: 6h", Status: "Healthy", Restarts: "—", Age: "90d"},
		}
	case "staging":
		return []ResourceItem{
			{Name: "api", Kind: "DEP", Ready: "1/1", Status: "Healthy", Restarts: "2", Age: "1d", Selector: map[string]string{"app": "api"}},
			{Name: "frontend", Kind: "DEP", Ready: "1/1", Status: "Healthy", Restarts: "0", Age: "3h", Selector: map[string]string{"app": "frontend"}},
			{Name: "worker", Kind: "DEP", Ready: "0/1", Status: "CrashLoop", Restarts: "47", Age: "6h", Selector: map[string]string{"app": "worker"}},
			{Name: "db", Kind: "STS", Ready: "1/1", Status: "Healthy", Restarts: "0", Age: "5d", Selector: map[string]string{"app": "db"}},
			{Name: "seed-data", Kind: "JOB", Ready: "1/1", Status: "Healthy", Restarts: "0", Age: "2d"},
		}
	case "monitoring":
		return []ResourceItem{
			{Name: "prometheus", Kind: "STS", Ready: "1/1", Status: "Healthy", Restarts: "0", Age: "30d", Selector: map[string]string{"app": "prometheus"}},
			{Name: "grafana", Kind: "DEP", Ready: "1/1", Status: "Healthy", Restarts: "0", Age: "15d", Selector: map[string]string{"app": "grafana"}},
			{Name: "alertmanager", Kind: "STS", Ready: "1/1", Status: "Healthy", Restarts: "0", Age: "30d", Selector: map[string]string{"app": "alertmanager"}},
			{Name: "node-exporter", Kind: "DS", Ready: "6/6", Status: "Healthy", Restarts: "0", Age: "30d"},
			{Name: "kube-state-metrics", Kind: "DEP", Ready: "1/1", Status: "Healthy", Restarts: "0", Age: "20d", Selector: map[string]string{"app": "kube-state-metrics"}},
		}
	case "kube-system":
		return []ResourceItem{
			{Name: "coredns", Kind: "DEP", Ready: "2/2", Status: "Healthy", Restarts: "0", Age: "180d", Selector: map[string]string{"app": "coredns"}},
			{Name: "etcd", Kind: "STS", Ready: "3/3", Status: "Healthy", Restarts: "0", Age: "180d", Selector: map[string]string{"app": "etcd"}},
			{Name: "kube-proxy", Kind: "DS", Ready: "6/6", Status: "Healthy", Restarts: "0", Age: "180d"},
			{Name: "kube-apiserver", Kind: "DEP", Ready: "2/2", Status: "Healthy", Restarts: "0", Age: "180d", Selector: map[string]string{"app": "kube-apiserver"}},
		}
	case "kube-public":
		return []ResourceItem{
			{Name: "cluster-info-publisher", Kind: "DEP", Ready: "1/1", Status: "Healthy", Restarts: "0", Age: "180d", Selector: map[string]string{"app": "cluster-info-publisher"}},
		}
	case "kube-node-lease":
		return []ResourceItem{
			{Name: "lease-heartbeat-sync", Kind: "DEP", Ready: "1/1", Status: "Healthy", Restarts: "0", Age: "180d", Selector: map[string]string{"app": "lease-heartbeat-sync"}},
		}
	case "ingress-nginx":
		return []ResourceItem{
			{Name: "ingress-nginx-controller", Kind: "DEP", Ready: "2/2", Status: "Healthy", Restarts: "0", Age: "60d", Selector: map[string]string{"app": "ingress-nginx-controller"}},
			{Name: "ingress-nginx-admission", Kind: "JOB", Ready: "1/1", Status: "Healthy", Restarts: "0", Age: "60d"},
		}
	case "cert-manager":
		return []ResourceItem{
			{Name: "cert-manager", Kind: "DEP", Ready: "1/1", Status: "Healthy", Restarts: "0", Age: "55d", Selector: map[string]string{"app": "cert-manager"}},
			{Name: "cert-manager-cainjector", Kind: "DEP", Ready: "1/1", Status: "Healthy", Restarts: "0", Age: "55d", Selector: map[string]string{"app": "cert-manager-cainjector"}},
			{Name: "cert-manager-webhook", Kind: "DEP", Ready: "1/1", Status: "Healthy", Restarts: "0", Age: "55d", Selector: map[string]string{"app": "cert-manager-webhook"}},
		}
	case "argocd":
		return []ResourceItem{
			{Name: "argocd-application-controller", Kind: "STS", Ready: "1/1", Status: "Healthy", Restarts: "0", Age: "45d", Selector: map[string]string{"app": "argocd-application-controller"}},
			{Name: "argocd-repo-server", Kind: "DEP", Ready: "1/1", Status: "Healthy", Restarts: "1", Age: "45d", Selector: map[string]string{"app": "argocd-repo-server"}},
			{Name: "argocd-server", Kind: "DEP", Ready: "1/1", Status: "Healthy", Restarts: "0", Age: "45d", Selector: map[string]string{"app": "argocd-server"}},
			{Name: "argocd-dex-server", Kind: "DEP", Ready: "1/1", Status: "Healthy", Restarts: "0", Age: "45d", Selector: map[string]string{"app": "argocd-dex-server"}},
		}
	case "dev":
		return []ResourceItem{
			{Name: "api", Kind: "DEP", Ready: "1/1", Status: "Healthy", Restarts: "0", Age: "3d", Selector: map[string]string{"app": "api"}},
			{Name: "frontend", Kind: "DEP", Ready: "1/1", Status: "Healthy", Restarts: "0", Age: "1d", Selector: map[string]string{"app": "frontend"}},
			{Name: "feature-flags-sync", Kind: "CJ", Ready: "Last: 14m", Status: "Healthy", Restarts: "—", Age: "12d"},
			{Name: "seed-demo-data", Kind: "JOB", Ready: "0/1", Status: "Failed", Restarts: "1", Age: "6m"},
		}
	case "sandbox":
		return []ResourceItem{
			{Name: "playground-api", Kind: "DEP", Ready: "0/1", Status: "Pending", Restarts: "0", Age: "5m", Selector: map[string]string{"app": "playground-api"}},
			{Name: "playground-ui", Kind: "DEP", Ready: "1/1", Status: "Healthy", Restarts: "0", Age: "2h", Selector: map[string]string{"app": "playground-ui"}},
		}
	default:
		return []ResourceItem{
			{Name: "api", Kind: "DEP", Ready: "2/3", Status: "Degraded", Restarts: "14", Age: "3d", Selector: map[string]string{"app": "api"}},
			{Name: "worker", Kind: "DEP", Ready: "1/1", Status: "Healthy", Restarts: "0", Age: "12d", Selector: map[string]string{"app": "worker"}},
			{Name: "db", Kind: "STS", Ready: "2/3", Status: "Progressing", Restarts: "0", Age: "6h", Selector: map[string]string{"app": "db"}},
			{Name: "node-exporter", Kind: "DS", Ready: "5/6", Status: "Degraded", Restarts: "0", Age: "30d"},
			{Name: "seed-users", Kind: "JOB", Ready: "0/1", Status: "Failed", Restarts: "3", Age: "18m"},
			{Name: "nightly-backup", Kind: "CJ", Ready: "Last: 6h", Status: "Healthy", Restarts: "—", Age: "90d"},
			{Name: "sync-reports", Kind: "CJ", Ready: "Last: —", Status: "Healthy", Restarts: "—", Age: "2d"},
			{Name: "cleanup-tmp", Kind: "CJ", Ready: "Last: 22m", Status: "Degraded", Restarts: "—", Age: "15d"},
			{Name: "old-data-prune", Kind: "CJ", Ready: "Last: 3d", Status: "Suspended", Restarts: "—", Age: "220d"},
		}
	}
}

func (w *Workloads) Sort(items []ResourceItem) {
	switch w.sortMode {
	case "status":
		problemSort(items, w.sortDesc)
	case "age":
		ageSort(items, w.sortDesc)
	case "kind":
		kindSort(items, w.sortDesc)
	case "ready":
		readySort(items, w.sortDesc)
	case "restarts":
		restartsSort(items, w.sortDesc)
	case "pods":
		workloadPodsSort(items, w.sortDesc)
	case "images":
		workloadImagesSort(items, w.sortDesc)
	case "service-account":
		workloadServiceAccountSort(items, w.sortDesc)
	case "selector":
		workloadSelectorSort(items, w.sortDesc)
	default:
		nameSort(items, w.sortDesc)
	}
}

func (w *Workloads) TableColumns() []TableColumn {
	return namespacedColumnsFor(w.Namespace(), []TableColumn{
		{ID: "name", Name: "NAME", Width: 35, Default: true},
		{ID: "kind", Name: "KIND", Width: 4, Default: true},
		{ID: "ready", Name: "READY", Width: 11, Default: true},
		{ID: "pods", Name: "PODS", Width: 6, Default: true},
		{ID: "status", Name: "STATUS", Width: 12, Default: true},
		{ID: "restarts", Name: "RESTARTS", Width: 8, Default: true},
		{ID: "age", Name: "AGE", Width: 6, Default: true},
	})
}

func (w *Workloads) TableRow(item ResourceItem) map[string]string {
	return map[string]string{
		"namespace": item.Namespace,
		"name":      item.Name,
		"kind":      item.Kind,
		"ready":     item.Ready,
		"pods":      workloadPodCount(item),
		"status":    item.Status,
		"restarts":  item.Restarts,
		"age":       item.Age,
	}
}

func (w *Workloads) TableColumnsWide() []TableColumn {
	return namespacedColumnsFor(w.Namespace(), []TableColumn{
		{ID: "name", Name: "NAME", Width: 35, Default: true},
		{ID: "kind", Name: "KIND", Width: 4, Default: true},
		{ID: "ready", Name: "READY", Width: 11, Default: true},
		{ID: "pods", Name: "PODS", Width: 6, Default: true},
		{ID: "status", Name: "STATUS", Width: 12, Default: true},
		{ID: "restarts", Name: "RESTARTS", Width: 8, Default: true},
		{ID: "age", Name: "AGE", Width: 6, Default: true},
		{ID: "selector", Name: "SELECTOR", Width: 20, Default: false},
		{ID: "images", Name: "IMAGES", Width: 28, Default: false},
		{ID: "service-account", Name: "SERVICEACCOUNT", Width: 20, Default: false},
	})
}

func (w *Workloads) TableRowWide(item ResourceItem) map[string]string {
	row := w.TableRow(item)
	parts := make([]string, 0, len(item.Selector))
	for k, v := range item.Selector {
		parts = append(parts, k+"="+v)
	}
	sort.Strings(parts)
	row["selector"] = strings.Join(parts, ",")
	row["pods"] = workloadPodCount(item)
	row["images"] = workloadImages(item)
	row["service-account"] = workloadServiceAccount(item)
	return row
}

func workloadPodCount(item ResourceItem) string {
	if strings.HasPrefix(item.Ready, "Last:") {
		return "-"
	}
	parts := strings.SplitN(item.Ready, "/", 2)
	if len(parts) != 2 {
		return "-"
	}
	return parts[0]
}

func workloadImages(item ResourceItem) string {
	switch item.Name {
	case "api":
		return "myco/api:v2.3.1"
	case "frontend":
		return "myco/frontend:v1.8.0"
	case "worker":
		return "myco/worker:v2.0.1"
	case "db":
		return "postgres:16"
	case "cache":
		return "redis:7"
	case "prometheus":
		return "prom/prometheus:v2.51.0"
	case "grafana":
		return "grafana/grafana:10.4.2"
	case "alertmanager":
		return "prom/alertmanager:v0.27.0"
	case "node-exporter":
		return "quay.io/prometheus/node-exporter:v1.8.1"
	case "kube-state-metrics":
		return "registry.k8s.io/kube-state-metrics:v2.13.0"
	case "coredns":
		return "registry.k8s.io/coredns/coredns:v1.11.1"
	case "kube-proxy":
		return "registry.k8s.io/kube-proxy:v1.30.0"
	case "kube-apiserver":
		return "registry.k8s.io/kube-apiserver:v1.30.0"
	case "cert-manager":
		return "quay.io/jetstack/cert-manager-controller:v1.14.5"
	case "cert-manager-cainjector":
		return "quay.io/jetstack/cert-manager-cainjector:v1.14.5"
	case "cert-manager-webhook":
		return "quay.io/jetstack/cert-manager-webhook:v1.14.5"
	default:
		switch item.Kind {
		case "JOB", "CJ":
			return "busybox:1.36"
		default:
			return "ghcr.io/example/" + item.Name + ":v1.0.0"
		}
	}
}

func workloadServiceAccount(item ResourceItem) string {
	switch item.Kind {
	case "JOB", "CJ":
		return "batch-runner"
	case "DS":
		return "node-reader"
	default:
		return item.Name
	}
}

func (w *Workloads) SetSort(mode string, desc bool) {
	w.sortMode = mode
	w.sortDesc = desc
}

func (w *Workloads) SortMode() string { return w.sortMode }
func (w *Workloads) SortDesc() bool   { return w.sortDesc }
func (w *Workloads) SortKeys() []SortKey {
	return sortKeysFor([]string{"name", "kind", "ready", "pods", "status", "restarts", "age", "selector", "images", "service-account"})
}

func workloadPodsSort(items []ResourceItem, desc bool) {
	sort.SliceStable(items, func(i, j int) bool {
		pi := workloadPodCount(items[i])
		pj := workloadPodCount(items[j])
		if pi != pj {
			if desc {
				return pi > pj
			}
			return pi < pj
		}
		return items[i].Name < items[j].Name
	})
}

func workloadImagesSort(items []ResourceItem, desc bool) {
	sort.SliceStable(items, func(i, j int) bool {
		ii := workloadImages(items[i])
		ij := workloadImages(items[j])
		if ii != ij {
			if desc {
				return ii > ij
			}
			return ii < ij
		}
		return items[i].Name < items[j].Name
	})
}

func workloadServiceAccountSort(items []ResourceItem, desc bool) {
	sort.SliceStable(items, func(i, j int) bool {
		si := workloadServiceAccount(items[i])
		sj := workloadServiceAccount(items[j])
		if si != sj {
			if desc {
				return si > sj
			}
			return si < sj
		}
		return items[i].Name < items[j].Name
	})
}

func workloadSelectorSort(items []ResourceItem, desc bool) {
	sort.SliceStable(items, func(i, j int) bool {
		li := selectorString(items[i].Selector)
		lj := selectorString(items[j].Selector)
		if li != lj {
			if desc {
				return li > lj
			}
			return li < lj
		}
		return items[i].Name < items[j].Name
	})
}

func selectorString(selector map[string]string) string {
	parts := make([]string, 0, len(selector))
	for k, v := range selector {
		parts = append(parts, k+"="+v)
	}
	sort.Strings(parts)
	return strings.Join(parts, ",")
}

func (w *Workloads) Banner() string {
	switch w.scenario {
	case "forbidden":
		return "Access denied: cannot list workloads in namespace `" + w.Namespace() + "` (need get/list on deployments,statefulsets,daemonsets,jobs,cronjobs)."
	case "partial":
		return "Partial access: Jobs and CronJobs are hidden by RBAC."
	case "offline":
		return "Cluster unreachable. Showing stale data from last sync: 2m ago."
	default:
		return ""
	}
}

func (w *Workloads) EmptyMessage(filtered bool, filter string) string {
	if filtered {
		return "No workloads match `" + filter + "`. Press esc to clear."
	}

	switch w.scenario {
	case "forbidden":
		return "No workloads visible due to RBAC restrictions."
	case "empty":
		return "No workloads found in namespace `" + w.Namespace() + "`. Switch namespace or clear filter."
	default:
		return "No workloads found in namespace `" + w.Namespace() + "`."
	}
}

func (w *Workloads) Detail(item ResourceItem) DetailData {
	status := item.Status
	if status == "" {
		status = "Healthy"
	}
	return DetailData{
		Summary: []SummaryField{
			{Key: "status", Label: "Status", Value: status},
			{Key: "ready", Label: "Ready", Value: item.Ready},
			{Key: "kind", Label: "Kind", Value: item.Kind},
			{Key: "age", Label: "Age", Value: item.Age},
		},
		Events: []string{
			"2m ago   Normal   Reconciled   Workload " + item.Name + " is up to date",
		},
		Labels: []string{
			"app=" + item.Name,
			"team=platform",
		},
	}
}

func (w *Workloads) Logs(item ResourceItem) []string {
	return expandMockLogs([]string{
		"Selecting logs for workload " + item.Name + "...",
		"Use related views for per-pod details.",
	}, 80)
}

func (w *Workloads) Events(item ResourceItem) []string {
	return []string{
		"2m ago   Normal   Reconciled   Workload " + item.Name + " is up to date",
		"15m ago  Normal   Scaling      Replica count evaluated",
	}
}

func (w *Workloads) Describe(item ResourceItem) string {
	kind := "Deployment"
	extra := ""
	switch item.Kind {
	case "STS":
		kind = "StatefulSet"
	case "DS":
		kind = "DaemonSet"
	case "JOB":
		kind = "Job"
		extra = "Completions:      1\nParallelism:      1\nBackoff Limit:    3\n"
	case "CJ":
		kind = "CronJob"
		extra = "Schedule:             0 2 * * *\nConcurrency Policy:   Forbid\nSuspend:              False\n"
	}

	return "Name:             " + item.Name + "\n" +
		"Namespace:        " + w.Namespace() + "\n" +
		"Kind:             " + kind + "\n" +
		"Selector:         app=" + item.Name + "\n" +
		"Labels:           app=" + item.Name + "\n" +
		"                  team=platform\n" +
		extra +
		"Replicas:         " + item.Ready + "\n" +
		"Status:           " + item.Status + "\n" +
		"Age:              " + item.Age + "\n" +
		"Events:\n" +
		"  Type    Reason      Age  Message\n" +
		"  ----    ------      ---  -------\n" +
		"  Normal  Reconciled  2m   Workload " + item.Name + " is up to date\n" +
		"  Normal  Scaling     15m  Replica count evaluated"
}

func (w *Workloads) YAML(item ResourceItem) string {
	apiVersion := "apps/v1"
	kind := "Deployment"
	desired := "2"
	if parts := strings.SplitN(item.Ready, "/", 2); len(parts) == 2 {
		desired = strings.TrimSpace(parts[1])
	}
	spec := `  replicas: ` + desired + `
  selector:
    matchLabels:
      app: ` + item.Name + `
  strategy:
    type: RollingUpdate
  template:
    metadata:
      labels:
        app: ` + item.Name + `
    spec:
      containers:
      - name: ` + item.Name + `
        image: ghcr.io/example/` + item.Name + `:latest
        ports:
        - containerPort: 8080`

	switch item.Kind {
	case "STS":
		kind = "StatefulSet"
		spec = `  replicas: 3
  serviceName: ` + item.Name + `
  selector:
    matchLabels:
      app: ` + item.Name + `
  template:
    metadata:
      labels:
        app: ` + item.Name + `
    spec:
      containers:
      - name: ` + item.Name + `
        image: ghcr.io/example/` + item.Name + `:latest
        ports:
        - containerPort: 8080
        volumeMounts:
        - name: data
          mountPath: /var/lib/` + item.Name + `
  volumeClaimTemplates:
  - metadata:
      name: data
    spec:
      accessModes: ["ReadWriteOnce"]
      resources:
        requests:
          storage: 10Gi`
	case "DS":
		kind = "DaemonSet"
		spec = `  selector:
    matchLabels:
      app: ` + item.Name + `
  updateStrategy:
    type: RollingUpdate
  template:
    metadata:
      labels:
        app: ` + item.Name + `
    spec:
      tolerations:
      - operator: Exists
      containers:
      - name: ` + item.Name + `
        image: ghcr.io/example/` + item.Name + `:latest
        ports:
        - containerPort: 9100`
	case "JOB":
		apiVersion = "batch/v1"
		kind = "Job"
		spec = `  backoffLimit: 3
  completions: 1
  parallelism: 1
  template:
    spec:
      restartPolicy: OnFailure
      containers:
      - name: ` + item.Name + `
        image: ghcr.io/example/` + item.Name + `:latest
        command: ["./run-job"]`
	case "CJ":
		apiVersion = "batch/v1"
		kind = "CronJob"
		spec = `  schedule: "0 2 * * *"
  concurrencyPolicy: Forbid
  successfulJobsHistoryLimit: 3
  failedJobsHistoryLimit: 1
  jobTemplate:
    spec:
      template:
        spec:
          restartPolicy: OnFailure
          containers:
          - name: ` + item.Name + `
            image: ghcr.io/example/` + item.Name + `:latest
            command: ["./run-job"]`
	}

	return strings.TrimSpace(`apiVersion: ` + apiVersion + `
kind: ` + kind + `
metadata:
  name: ` + item.Name + `
  namespace: ` + w.Namespace() + `
  labels:
    app: ` + item.Name + `
    team: platform
    app.kubernetes.io/managed-by: helm
spec:
` + spec)
}
