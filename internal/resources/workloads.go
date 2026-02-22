package resources

import "strings"

type Workloads struct {
	sortMode string
	scenario string
}

func NewWorkloads() *Workloads {
	return &Workloads{sortMode: "name", scenario: "normal"}
}

func (w *Workloads) Name() string { return "workloads" }
func (w *Workloads) Key() rune    { return 'W' }

func (w *Workloads) Items() []ResourceItem {
	if w.scenario == "forbidden" {
		return nil
	}

	items := workloadItemsForNamespace(ActiveNamespace)
	switch w.scenario {
	case "partial":
		if len(items) > 4 {
			items = items[:4]
		}
	case "empty":
		items = nil
	}

	w.Sort(items)
	return items
}

func workloadItemsForNamespace(ns string) []ResourceItem {
	switch ns {
	case "production":
		return []ResourceItem{
			{Name: "api", Kind: "DEP", Ready: "3/3", Status: "Healthy", Restarts: "0", Age: "14d"},
			{Name: "frontend", Kind: "DEP", Ready: "4/4", Status: "Healthy", Restarts: "0", Age: "7d"},
			{Name: "worker", Kind: "DEP", Ready: "2/2", Status: "Healthy", Restarts: "0", Age: "12d"},
			{Name: "db", Kind: "STS", Ready: "3/3", Status: "Healthy", Restarts: "0", Age: "30d"},
			{Name: "cache", Kind: "STS", Ready: "2/2", Status: "Healthy", Restarts: "0", Age: "30d"},
			{Name: "nightly-backup", Kind: "CJ", Ready: "Last: 6h", Status: "Healthy", Restarts: "—", Age: "90d"},
		}
	case "staging":
		return []ResourceItem{
			{Name: "api", Kind: "DEP", Ready: "1/1", Status: "Healthy", Restarts: "2", Age: "1d"},
			{Name: "frontend", Kind: "DEP", Ready: "1/1", Status: "Healthy", Restarts: "0", Age: "3h"},
			{Name: "worker", Kind: "DEP", Ready: "0/1", Status: "CrashLoop", Restarts: "47", Age: "6h"},
			{Name: "db", Kind: "STS", Ready: "1/1", Status: "Healthy", Restarts: "0", Age: "5d"},
			{Name: "seed-data", Kind: "JOB", Ready: "1/1", Status: "Healthy", Restarts: "0", Age: "2d"},
		}
	case "monitoring":
		return []ResourceItem{
			{Name: "prometheus", Kind: "STS", Ready: "1/1", Status: "Healthy", Restarts: "0", Age: "30d"},
			{Name: "grafana", Kind: "DEP", Ready: "1/1", Status: "Healthy", Restarts: "0", Age: "15d"},
			{Name: "alertmanager", Kind: "STS", Ready: "1/1", Status: "Healthy", Restarts: "0", Age: "30d"},
			{Name: "node-exporter", Kind: "DS", Ready: "6/6", Status: "Healthy", Restarts: "0", Age: "30d"},
			{Name: "kube-state-metrics", Kind: "DEP", Ready: "1/1", Status: "Healthy", Restarts: "0", Age: "20d"},
		}
	case "kube-system":
		return []ResourceItem{
			{Name: "coredns", Kind: "DEP", Ready: "2/2", Status: "Healthy", Restarts: "0", Age: "180d"},
			{Name: "etcd", Kind: "STS", Ready: "3/3", Status: "Healthy", Restarts: "0", Age: "180d"},
			{Name: "kube-proxy", Kind: "DS", Ready: "6/6", Status: "Healthy", Restarts: "0", Age: "180d"},
			{Name: "kube-apiserver", Kind: "DEP", Ready: "2/2", Status: "Healthy", Restarts: "0", Age: "180d"},
		}
	default:
		return []ResourceItem{
			{Name: "api", Kind: "DEP", Ready: "2/3", Status: "Degraded", Restarts: "14", Age: "3d"},
			{Name: "worker", Kind: "DEP", Ready: "1/1", Status: "Healthy", Restarts: "0", Age: "12d"},
			{Name: "db", Kind: "STS", Ready: "2/3", Status: "Progressing", Restarts: "0", Age: "6h"},
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
	if w.sortMode == "problem" {
		problemSort(items)
		return
	}
	defaultSort(items)
}

func (w *Workloads) TableColumns() []TableColumn {
	return []TableColumn{
		{Name: "NAME", Width: 35},
		{Name: "KIND", Width: 4},
		{Name: "READY", Width: 11},
		{Name: "STATUS", Width: 12},
		{Name: "RESTARTS", Width: 8},
		{Name: "AGE", Width: 6},
	}
}

func (w *Workloads) TableRow(item ResourceItem) []string {
	return []string{
		item.Name,
		item.Kind,
		item.Ready,
		item.Status,
		item.Restarts,
		item.Age,
	}
}

func (w *Workloads) ToggleSort() {
	if w.sortMode == "name" {
		w.sortMode = "problem"
		return
	}
	w.sortMode = "name"
}

func (w *Workloads) SortMode() string {
	return w.sortMode
}

func (w *Workloads) CycleScenario() {
	switch w.scenario {
	case "normal":
		w.scenario = "empty"
	case "empty":
		w.scenario = "forbidden"
	case "forbidden":
		w.scenario = "partial"
	case "partial":
		w.scenario = "offline"
	default:
		w.scenario = "normal"
	}
}

func (w *Workloads) Scenario() string {
	return w.scenario
}

func (w *Workloads) Banner() string {
	switch w.scenario {
	case "forbidden":
		return "Access denied: cannot list workloads in namespace `default` (need get/list on deployments,statefulsets,daemonsets,jobs,cronjobs)."
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
		return "No workloads found in namespace `default`. Switch namespace or clear filter."
	default:
		return "No workloads found in namespace `default`."
	}
}

func (w *Workloads) Detail(item ResourceItem) DetailData {
	status := item.Status
	if status == "" {
		status = "Healthy"
	}
	return DetailData{
		StatusLine: status + " " + item.Ready + "    kind: " + item.Kind + "    age: " + item.Age,
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
	return []string{
		"Selecting logs for workload " + item.Name + "...",
		"Use related views for per-pod details.",
	}
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
		"Namespace:        " + ActiveNamespace + "\n" +
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
	spec := `  replicas: 2
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
  namespace: ` + ActiveNamespace + `
  labels:
    app: ` + item.Name + `
    team: platform
    app.kubernetes.io/managed-by: helm
spec:
` + spec)
}
