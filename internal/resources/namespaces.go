package resources

import "strings"

type Namespaces struct {
	sortMode string
}

func (n *Namespaces) TableColumns() []TableColumn {
	return []TableColumn{
		{Name: "NAME", Width: 48},
		{Name: "STATUS", Width: 14},
		{Name: "AGE", Width: 6},
	}
}

func (n *Namespaces) TableRow(item ResourceItem) []string {
	return []string{item.Name, item.Status, item.Age}
}

func NewNamespaces() *Namespaces {
	return &Namespaces{sortMode: "name"}
}

func (n *Namespaces) Name() string { return "namespaces" }
func (n *Namespaces) Key() rune    { return 'N' }

func (n *Namespaces) Items() []ResourceItem {
	items := []ResourceItem{
		{Name: "default", Status: "Active", Age: "180d"},
		{Name: "kube-system", Status: "Active", Age: "180d"},
		{Name: "kube-public", Status: "Active", Age: "180d"},
		{Name: "kube-node-lease", Status: "Active", Age: "180d"},
		{Name: "production", Status: "Active", Age: "90d"},
		{Name: "staging", Status: "Active", Age: "85d"},
		{Name: "monitoring", Status: "Active", Age: "60d"},
		{Name: "ingress-nginx", Status: "Active", Age: "60d"},
		{Name: "cert-manager", Status: "Active", Age: "55d"},
		{Name: "argocd", Status: "Active", Age: "45d"},
		{Name: "dev", Status: "Active", Age: "30d"},
		{Name: "sandbox", Status: "Terminating", Age: "2d"},
	}
	items = expandMockItems(items, 24)
	n.Sort(items)
	return items
}

func (n *Namespaces) Sort(items []ResourceItem) {
	switch n.sortMode {
	case "status":
		problemSort(items)
	case "age":
		ageSort(items)
	default:
		defaultSort(items)
	}
}

func (n *Namespaces) ToggleSort() {
	n.sortMode = cycleSortMode(n.sortMode, []string{"name", "status", "age"})
}

func (n *Namespaces) SortMode() string {
	return n.sortMode
}

func (n *Namespaces) Detail(item ResourceItem) DetailData {
	labels := []string{"kubernetes.io/metadata.name=" + item.Name}
	events := []string{
		"—   No recent events",
	}

	switch item.Name {
	case "kube-system":
		labels = append(labels, "kubernetes.io/cluster-service=true")
	case "production":
		labels = append(labels,
			"env=production",
			"team=platform",
		)
		events = []string{
			"2h ago   Normal   ResourceQuotaUpdated   CPU limit adjusted to 32 cores",
		}
	case "staging":
		labels = append(labels, "env=staging", "team=platform")
	case "monitoring":
		labels = append(labels, "app.kubernetes.io/managed-by=helm")
	case "sandbox":
		labels = append(labels, "env=sandbox")
		events = []string{
			"1m ago   Warning  NamespaceDeletionDiscoveryFailure   Discovery failed for some groups",
			"2m ago   Normal   DeleteNamespace   Namespace sandbox is being terminated",
		}
	}

	return DetailData{
		StatusLine: item.Status + "    age: " + item.Age,
		Events:     events,
		Labels:     labels,
	}
}

func (n *Namespaces) Logs(item ResourceItem) []string {
	return expandMockLogs([]string{
		"Logs are not available for namespaces.",
	}, 30)
}

func (n *Namespaces) Events(item ResourceItem) []string {
	if item.Name == "sandbox" {
		return []string{
			"1m ago   Warning  NamespaceDeletionDiscoveryFailure   Discovery failed for some groups",
			"2m ago   Normal   DeleteNamespace   Namespace sandbox is being terminated",
		}
	}
	return []string{"—   No recent events"}
}

func (n *Namespaces) Describe(item ResourceItem) string {
	return "Name:          " + item.Name + "\n" +
		"Labels:        kubernetes.io/metadata.name=" + item.Name + "\n" +
		"Annotations:   <none>\n" +
		"Status:        " + item.Status + "\n" +
		"Age:           " + item.Age + "\n" +
		"Events:        <none>"
}

func (n *Namespaces) YAML(item ResourceItem) string {
	annotations := ""
	extraLabels := ""
	switch item.Name {
	case "production":
		extraLabels = "\n    env: production\n    team: platform"
		annotations = "\n  annotations:\n    scheduler.alpha.kubernetes.io/node-selector: env=production"
	case "staging":
		extraLabels = "\n    env: staging\n    team: platform"
	case "monitoring":
		extraLabels = "\n    app.kubernetes.io/managed-by: helm"
	case "kube-system":
		extraLabels = "\n    kubernetes.io/cluster-service: \"true\""
	}
	return strings.TrimSpace(`apiVersion: v1
kind: Namespace
metadata:
  name: ` + item.Name + `
  labels:
    kubernetes.io/metadata.name: ` + item.Name + extraLabels + annotations + `
  uid: a1b2c3d4-e5f6-7890-abcd-ef0123456789
spec:
  finalizers:
  - kubernetes
status:
  phase: ` + item.Status)
}
