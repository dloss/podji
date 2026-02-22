package resources

import "strings"

type Nodes struct {
	sortMode string
}

func (n *Nodes) TableColumns() []TableColumn {
	return []TableColumn{
		{Name: "NAME", Width: 30},
		{Name: "STATUS", Width: 12},
		{Name: "ROLES", Width: 16},
		{Name: "VERSION", Width: 12},
		{Name: "AGE", Width: 6},
	}
}

func (n *Nodes) TableRow(item ResourceItem) []string {
	role := "worker"
	version := "v1.29.2"
	if strings.HasPrefix(item.Name, "control-plane") {
		role = "control-plane"
	}
	return []string{item.Name, item.Status, role, version, item.Age}
}

func NewNodes() *Nodes {
	return &Nodes{sortMode: "name"}
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
	if n.sortMode == "problem" {
		problemSort(items)
		return
	}
	defaultSort(items)
}

func (n *Nodes) ToggleSort() {
	if n.sortMode == "name" {
		n.sortMode = "problem"
		return
	}
	n.sortMode = "name"
}

func (n *Nodes) SortMode() string {
	return n.sortMode
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

func (n *Nodes) Describe(item ResourceItem) string {
	role := "worker"
	if strings.HasPrefix(item.Name, "control-plane") {
		role = "control-plane"
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
	readyStatus := "True"
	if item.Status == "NotReady" {
		readyStatus = "False"
	}
	return "Name:               " + item.Name + "\n" +
		"Roles:              " + role + "\n" +
		"Labels:             kubernetes.io/hostname=" + item.Name + "\n" +
		"                    node.kubernetes.io/instance-type=m5.xlarge\n" +
		"                    topology.kubernetes.io/zone=us-east-1a\n" +
		"                    node-role.kubernetes.io/" + role + "=\n" +
		"Annotations:        node.alpha.kubernetes.io/ttl: 0\n" +
		"CreationTimestamp:   <age: " + item.Age + ">\n" +
		"Addresses:\n" +
		"  InternalIP:  " + ip + "\n" +
		"  Hostname:    " + item.Name + "\n" +
		"Capacity:\n" +
		"  cpu:                4\n" +
		"  memory:             16384Mi\n" +
		"  pods:               110\n" +
		"  ephemeral-storage:  100Gi\n" +
		"Allocatable:\n" +
		"  cpu:                3920m\n" +
		"  memory:             15896Mi\n" +
		"  pods:               110\n" +
		"  ephemeral-storage:  95Gi\n" +
		"Conditions:\n" +
		"  Type             Status\n" +
		"  ----             ------\n" +
		"  Ready            " + readyStatus + "\n" +
		"  MemoryPressure   False\n" +
		"  DiskPressure     False\n" +
		"  PIDPressure      False\n" +
		"System Info:\n" +
		"  Kubelet Version:            v1.29.2\n" +
		"  Container Runtime Version:  containerd://1.7.11\n" +
		"  Kernel Version:             5.15.0-1051-aws\n" +
		"  OS Image:                   Ubuntu 22.04.3 LTS\n" +
		"  Operating System:           linux\n" +
		"  Architecture:               amd64\n" +
		"PodCIDR:            10.244.0.0/24\n" +
		"Pods:               " + item.Ready
}

func (n *Nodes) YAML(item ResourceItem) string {
	role := "worker"
	if strings.HasPrefix(item.Name, "control-plane") {
		role = "control-plane"
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
	readyStatus := "True"
	if item.Status == "NotReady" {
		readyStatus = "False"
	}
	return strings.TrimSpace(`apiVersion: v1
kind: Node
metadata:
  name: ` + item.Name + `
  labels:
    kubernetes.io/hostname: ` + item.Name + `
    kubernetes.io/os: linux
    kubernetes.io/arch: amd64
    node.kubernetes.io/instance-type: m5.xlarge
    topology.kubernetes.io/zone: us-east-1a
    topology.kubernetes.io/region: us-east-1
    node-role.kubernetes.io/` + role + `: ""
spec:
  podCIDR: 10.244.0.0/24
  providerID: aws:///us-east-1a/i-0abc123def456
status:
  capacity:
    cpu: "4"
    memory: 16384Mi
    pods: "110"
    ephemeral-storage: 100Gi
  allocatable:
    cpu: "3920m"
    memory: 15896Mi
    pods: "110"
    ephemeral-storage: 95Gi
  conditions:
  - type: Ready
    status: "` + readyStatus + `"
    lastHeartbeatTime: "2026-02-21T10:00:00Z"
    lastTransitionTime: "2025-11-23T08:00:00Z"
    reason: KubeletReady
    message: kubelet is posting ready status
  - type: MemoryPressure
    status: "False"
    lastHeartbeatTime: "2026-02-21T10:00:00Z"
  - type: DiskPressure
    status: "False"
    lastHeartbeatTime: "2026-02-21T10:00:00Z"
  - type: PIDPressure
    status: "False"
    lastHeartbeatTime: "2026-02-21T10:00:00Z"
  addresses:
  - type: InternalIP
    address: ` + ip + `
  - type: Hostname
    address: ` + item.Name + `
  nodeInfo:
    kubeletVersion: v1.29.2
    kubeProxyVersion: v1.29.2
    operatingSystem: linux
    architecture: amd64
    containerRuntimeVersion: containerd://1.7.11
    kernelVersion: 5.15.0-1051-aws
    osImage: Ubuntu 22.04.3 LTS`)
}
