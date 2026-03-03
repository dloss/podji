package resources

import (
	"sort"
	"strings"
)

type Nodes struct {
	sortMode string
	sortDesc bool
}

func (n *Nodes) TableColumns() []TableColumn {
	return []TableColumn{
		{ID: "name", Name: "NAME", Width: 30, Default: true},
		{ID: "status", Name: "STATUS", Width: 12, Default: true},
		{ID: "roles", Name: "ROLES", Width: 16, Default: true},
		{ID: "version", Name: "VERSION", Width: 12, Default: true},
		{ID: "internal-ip", Name: "INTERNAL-IP", Width: 14, Default: false},
		{ID: "age", Name: "AGE", Width: 6, Default: true},
	}
}

func nodeRole(name string) string {
	if strings.HasPrefix(name, "control-plane") {
		return "control-plane"
	}
	return "worker"
}

func (n *Nodes) TableRow(item ResourceItem) map[string]string {
	return map[string]string{
		"name":        item.Name,
		"status":      item.Status,
		"roles":       nodeRole(item.Name),
		"version":     "v1.29.2",
		"internal-ip": item.Extra["internal-ip"],
		"age":         item.Age,
	}
}

func (n *Nodes) TableColumnsWide() []TableColumn {
	return []TableColumn{
		{ID: "name", Name: "NAME", Width: 30, Default: true},
		{ID: "status", Name: "STATUS", Width: 12, Default: true},
		{ID: "roles", Name: "ROLES", Width: 16, Default: true},
		{ID: "version", Name: "VERSION", Width: 12, Default: true},
		{ID: "age", Name: "AGE", Width: 6, Default: true},
		{ID: "internal-ip", Name: "INTERNAL-IP", Width: 14, Default: false},
		{ID: "instance-type", Name: "INSTANCE-TYPE", Width: 13, Default: false},
		{ID: "zone", Name: "ZONE", Width: 12, Default: false},
		{ID: "os", Name: "OS", Width: 7, Default: false},
		{ID: "arch", Name: "ARCH", Width: 7, Default: false},
		{ID: "kernel-version", Name: "KERNEL-VERSION", Width: 22, Default: false},
		{ID: "runtime", Name: "CONTAINER-RUNTIME", Width: 22, Default: false},
		{ID: "taints", Name: "TAINTS", Width: 6, Default: false},
	}
}

func (n *Nodes) TableRowWide(item ResourceItem) map[string]string {
	row := n.TableRow(item)
	row["os"] = item.Extra["os"]
	row["arch"] = item.Extra["arch"]
	row["kernel-version"] = item.Extra["kernel-version"]
	row["internal-ip"] = item.Extra["internal-ip"]
	row["instance-type"] = item.Extra["instance-type"]
	row["zone"] = item.Extra["zone"]
	row["runtime"] = item.Extra["runtime"]
	row["taints"] = item.Extra["taints"]
	return row
}

func NewNodes() *Nodes {
	return &Nodes{sortMode: "name"}
}

func (n *Nodes) Name() string { return "nodes" }
func (n *Nodes) Key() rune    { return 'O' }

func (n *Nodes) Items() []ResourceItem {
	items := []ResourceItem{
		{Name: "worker-01", Status: "Ready", Ready: "48/110", Age: "90d", Extra: map[string]string{"os": "linux", "arch": "amd64", "kernel-version": "5.15.0-76-generic", "internal-ip": "10.0.1.11", "instance-type": "m5.xlarge", "zone": "us-east-1a", "runtime": "containerd://1.7.11", "taints": "0"}},
		{Name: "worker-02", Status: "Ready", Ready: "35/110", Age: "90d", Extra: map[string]string{"os": "linux", "arch": "amd64", "kernel-version": "5.15.0-76-generic", "internal-ip": "10.0.1.12", "instance-type": "m5.xlarge", "zone": "us-east-1b", "runtime": "containerd://1.7.11", "taints": "0"}},
		{Name: "worker-03", Status: "Ready", Ready: "52/110", Age: "60d", Extra: map[string]string{"os": "linux", "arch": "amd64", "kernel-version": "5.15.0-91-generic", "internal-ip": "10.0.1.13", "instance-type": "m5.xlarge", "zone": "us-east-1c", "runtime": "containerd://1.7.11", "taints": "0"}},
		{Name: "worker-04", Status: "NotReady", Ready: "0/110", Age: "30d", Extra: map[string]string{"os": "linux", "arch": "amd64", "kernel-version": "5.15.0-91-generic", "internal-ip": "10.0.1.14", "instance-type": "m5.xlarge", "zone": "us-east-1a", "runtime": "containerd://1.7.11", "taints": "1"}},
		{Name: "control-plane-01", Status: "Ready", Ready: "12/110", Age: "180d", Extra: map[string]string{"os": "linux", "arch": "amd64", "kernel-version": "5.15.0-76-generic", "internal-ip": "10.0.0.1", "instance-type": "m5.large", "zone": "us-east-1a", "runtime": "containerd://1.7.11", "taints": "1"}},
		{Name: "control-plane-02", Status: "Ready", Ready: "11/110", Age: "180d", Extra: map[string]string{"os": "linux", "arch": "amd64", "kernel-version": "5.15.0-76-generic", "internal-ip": "10.0.0.2", "instance-type": "m5.large", "zone": "us-east-1b", "runtime": "containerd://1.7.11", "taints": "1"}},
	}
	items = expandMockItems(items, 20)
	n.Sort(items)
	return items
}

func (n *Nodes) Sort(items []ResourceItem) {
	switch n.sortMode {
	case "status":
		problemSort(items, n.sortDesc)
	case "age":
		ageSort(items, n.sortDesc)
	case "internal-ip":
		nodeInternalIPSort(items, n.sortDesc)
	default:
		nameSort(items, n.sortDesc)
	}
}

func (n *Nodes) SetSort(mode string, desc bool) { n.sortMode = mode; n.sortDesc = desc }
func (n *Nodes) SortMode() string               { return n.sortMode }
func (n *Nodes) SortDesc() bool                 { return n.sortDesc }
func (n *Nodes) SortKeys() []SortKey {
	return sortKeysFor([]string{"name", "status", "internal-ip", "age"})
}

func nodeInternalIPSort(items []ResourceItem, desc bool) {
	sort.SliceStable(items, func(i, j int) bool {
		ipi := items[i].Extra["internal-ip"]
		ipj := items[j].Extra["internal-ip"]
		if ipi != ipj {
			if desc {
				return ipi > ipj
			}
			return ipi < ipj
		}
		return items[i].Name < items[j].Name
	})
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
		Summary: []SummaryField{
			{Key: "status", Label: "Status", Value: item.Status},
			{Key: "pods", Label: "Pods", Value: item.Ready},
			{Key: "ip", Label: "IP", Value: ip},
			{Key: "os", Label: "OS", Value: "linux/amd64"},
			{Key: "kubelet", Label: "Kubelet", Value: "v1.29.2"},
		},
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
	return expandMockLogs([]string{
		"Logs are not available for nodes directly. Check kubelet logs on the host.",
	}, 40)
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
