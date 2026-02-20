package resources

import (
	"sort"
	"strings"
)

type Workloads struct {
	sortMode string
}

func NewWorkloads() *Workloads {
	return &Workloads{sortMode: "problem"}
}

func (w *Workloads) Name() string { return "workloads" }
func (w *Workloads) Key() rune    { return 'W' }

func (w *Workloads) Items() []ResourceItem {
	items := []ResourceItem{
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
	w.Sort(items)
	return items
}

func (w *Workloads) Sort(items []ResourceItem) {
	if w.sortMode == "name" {
		sort.SliceStable(items, func(i, j int) bool {
			return items[i].Name < items[j].Name
		})
		return
	}

	sort.SliceStable(items, func(i, j int) bool {
		wi := workloadStatusWeight(items[i].Status)
		wj := workloadStatusWeight(items[j].Status)
		if wi != wj {
			return wi < wj
		}
		return items[i].Name < items[j].Name
	})
}

func workloadStatusWeight(status string) int {
	switch status {
	case "Failed":
		return 0
	case "Degraded":
		return 1
	case "Progressing":
		return 2
	case "Healthy":
		return 3
	case "Suspended":
		return 4
	default:
		return 5
	}
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
	if w.sortMode == "problem" {
		w.sortMode = "name"
		return
	}
	w.sortMode = "problem"
}

func (w *Workloads) SortMode() string {
	return w.sortMode
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

func (w *Workloads) YAML(item ResourceItem) string {
	return strings.TrimSpace("kind: Workload\nmetadata:\n  name: " + item.Name + "\nspec:\n  kind: " + item.Kind)
}
