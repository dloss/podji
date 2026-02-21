package resources

import "strings"

type Nodes struct{}

func NewNodes() *Nodes {
	return &Nodes{}
}

func (n *Nodes) Name() string { return "nodes" }
func (n *Nodes) Key() rune   { return 'O' }

func (n *Nodes) Items() []ResourceItem {
	items := []ResourceItem{
		{Name: "worker-01", Status: "Ready", Ready: "48/110", Age: "90d"},
		{Name: "worker-02", Status: "Ready", Ready: "35/110", Age: "90d"},
		{Name: "worker-03", Status: "Ready", Ready: "52/110", Age: "60d"},
		{Name: "worker-04", Status: "NotReady", Ready: "0/110", Age: "30d"},
		{Name: "control-plane-01", Status: "Ready", Ready: "12/110", Age: "180d"},
		{Name: "control-plane-02", Status: "Ready", Ready: "11/110", Age: "180d"},
	}
	n.Sort(items)
	return items
}

func (n *Nodes) Sort(items []ResourceItem) {
	defaultSort(items)
}

func (n *Nodes) Detail(item ResourceItem) DetailData {
	conditions := []string{
		"Ready = True               kubelet is posting ready status",
		"MemoryPressure = False",
		"DiskPressure = False",
		"PIDPressure = False",
	}
	events := []string{
		"—   No recent events",
	}

	if item.Name == "worker-04" {
		conditions = []string{
			"Ready = False              kubelet stopped posting node status",
			"MemoryPressure = Unknown",
			"DiskPressure = Unknown",
		}
		events = []string{
			"5m ago    Warning  NodeNotReady        Node worker-04 status is now: NodeNotReady",
			"6m ago    Normal   NodeHasNoDiskPressure   Node worker-04 status is now: NodeHasNoDiskPressure",
		}
	}

	ip := "10.0.1.1"
	switch item.Name {
	case "worker-01":
		ip = "10.0.1.11"
	case "worker-02":
		ip = "10.0.1.12"
	case "worker-03":
		ip = "10.0.1.13"
	case "worker-04":
		ip = "10.0.1.14"
	case "control-plane-01":
		ip = "10.0.0.1"
	case "control-plane-02":
		ip = "10.0.0.2"
	}

	return DetailData{
		StatusLine: item.Status + "    pods: " + item.Ready + "    ip: " + ip + "    os: linux/amd64    kubelet: v1.29.2",
		Conditions: conditions,
		Events:     events,
		Labels: []string{
			"kubernetes.io/hostname=" + item.Name,
			"node.kubernetes.io/instance-type=m5.xlarge",
			"topology.kubernetes.io/zone=us-east-1a",
		},
	}
}

func (n *Nodes) Logs(item ResourceItem) []string {
	return []string{
		"Logs are not available for nodes directly. Check kubelet logs on the host.",
	}
}

func (n *Nodes) Events(item ResourceItem) []string {
	if item.Name == "worker-04" {
		return []string{
			"5m ago    Warning  NodeNotReady        Node worker-04 status is now: NodeNotReady",
			"6m ago    Normal   NodeHasNoDiskPressure   Node worker-04 status is now: NodeHasNoDiskPressure",
		}
	}
	return []string{"—   No recent events"}
}

func (n *Nodes) YAML(item ResourceItem) string {
	return strings.TrimSpace(`apiVersion: v1
kind: Node
metadata:
  name: ` + item.Name + `
status:
  conditions:
  - type: Ready
    status: "` + strings.Replace(item.Status, "Not", "False # ", 1) + `"`)
}
