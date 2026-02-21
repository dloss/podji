package resources

import "strings"

type Namespaces struct{}

func NewNamespaces() *Namespaces {
	return &Namespaces{}
}

func (n *Namespaces) Name() string { return "namespaces" }
func (n *Namespaces) Key() rune   { return 'N' }

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
	n.Sort(items)
	return items
}

func (n *Namespaces) Sort(items []ResourceItem) {
	defaultSort(items)
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
	return []string{
		"Logs are not available for namespaces.",
	}
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

func (n *Namespaces) YAML(item ResourceItem) string {
	return strings.TrimSpace(`apiVersion: v1
kind: Namespace
metadata:
  name: ` + item.Name + `
status:
  phase: ` + item.Status)
}
