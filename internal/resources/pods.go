package resources

import "strings"

type Pods struct{}

func NewPods() *Pods {
	return &Pods{}
}

func (p *Pods) Name() string {
	return "pods"
}

func (p *Pods) Key() rune {
	return 'P'
}

func (p *Pods) Items() []ResourceItem {
	items := []ResourceItem{
		{Name: "api-7c6c8d5f7d-x8p2k", Status: "CrashLoop", Ready: "1/2", Restarts: "5 (10m)", Age: "2d"},
		{Name: "worker-55c6c6f9f-9mlr", Status: "Pending", Ready: "0/1", Restarts: "0", Age: "3m"},
		{Name: "web-6d9f9f7b7d-2r9kq", Status: "Running", Ready: "2/2", Restarts: "0", Age: "5d"},
		{Name: "web-6d9f9f7b7d-kp4mn", Status: "Running", Ready: "2/2", Restarts: "0", Age: "5d"},
		{Name: "db-0", Status: "Running", Ready: "1/1", Restarts: "0", Age: "12d"},
		{Name: "cache-redis-0", Status: "Running", Ready: "1/1", Restarts: "0", Age: "12d"},
	}
	p.Sort(items)
	return items
}

func (p *Pods) Sort(items []ResourceItem) {
	defaultSort(items)
}

func (p *Pods) Detail(item ResourceItem) DetailData {
	return DetailData{
		StatusLine: "Running " + item.Ready + "    node: worker-03    ip: 10.244.2.15    qos: Burstable",
		Containers: []ContainerRow{
			{Name: "api", Image: "myco/api:v2.3.1", State: "Running", Restarts: "0", Reason: ""},
			{Name: "sidecar", Image: "envoy:1.28", State: "CrashLoopBackOff", Restarts: "5", Reason: "OOMKilled (10m ago)"},
		},
		Conditions: []string{
			"Ready = False              containers with unready status: [sidecar]",
			"ContainersReady = False",
		},
		Events: []string{
			"10m ago   Warning  BackOff      Back-off restarting failed container sidecar",
			"12m ago   Normal   Pulled       Successfully pulled image \"envoy:1.28\"",
			"15m ago   Warning  OOMKilling   Memory capped at 128Mi",
		},
		Labels: []string{
			"app=api",
			"tier=backend",
			"env=prod",
		},
	}
}

func (p *Pods) Logs(item ResourceItem) []string {
	return []string{
		"2025-06-15T12:03:01Z  Starting envoy proxy...",
		"2025-06-15T12:03:01Z  Loading configuration from /etc/envoy/config.yaml",
		"2025-06-15T12:03:02Z  Listener 0.0.0.0:8080 created",
		"2025-06-15T12:03:02Z  Allocating buffer pool (128Mi limit)",
		"2025-06-15T12:03:03Z  ERROR: buffer allocation failed: OOM",
		"2025-06-15T12:03:03Z  Fatal: cannot start with current memory limits",
	}
}

func (p *Pods) Events(item ResourceItem) []string {
	return []string{
		"10m ago   Warning  BackOff      Back-off restarting failed container sidecar",
		"12m ago   Normal   Pulled       Successfully pulled image \"envoy:1.28\"",
		"15m ago   Warning  OOMKilling   Memory capped at 128Mi",
	}
}

func (p *Pods) YAML(item ResourceItem) string {
	return strings.TrimSpace(`apiVersion: v1
kind: Pod
metadata:
  name: api-7c6c8d5f7d-x8p2k
spec:
  containers:
  - name: api
    image: myco/api:v2.3.1
  - name: sidecar
    image: envoy:1.28
status:
  phase: Running`)
}
