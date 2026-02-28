package resources

import "strings"

type Pods struct {
	sortMode string
	sortDesc bool
}

func NewPods() *Pods {
	return &Pods{sortMode: "name"}
}

func (p *Pods) Name() string {
	return "pods"
}

func (p *Pods) Key() rune {
	return 'P'
}

func (p *Pods) Items() []ResourceItem {
	var items []ResourceItem
	if ActiveNamespace == AllNamespaces {
		items = allNamespaceItems(podItemsForNamespace)
	} else {
		items = podItemsForNamespace(ActiveNamespace)
		items = expandMockItems(items, 36)
	}
	p.Sort(items)
	return items
}

func podItemsForNamespace(ns string) []ResourceItem {
	switch ns {
	case "production":
		return []ResourceItem{
			{Name: "api-7c6c8d5f7d-x8p2k", Status: "Running", Ready: "2/2", Restarts: "0", Age: "14d"},
			{Name: "api-7c6c8d5f7d-m3n9p", Status: "Running", Ready: "2/2", Restarts: "0", Age: "14d"},
			{Name: "api-7c6c8d5f7d-q5r2s", Status: "Running", Ready: "2/2", Restarts: "0", Age: "14d"},
			{Name: "frontend-8b4d9e2f-k1l3m", Status: "Running", Ready: "1/1", Restarts: "0", Age: "7d"},
			{Name: "frontend-8b4d9e2f-n4o6p", Status: "Running", Ready: "1/1", Restarts: "0", Age: "7d"},
			{Name: "worker-55c6c6f9f-9mlr", Status: "Running", Ready: "1/1", Restarts: "0", Age: "12d"},
			{Name: "db-0", Status: "Running", Ready: "1/1", Restarts: "0", Age: "30d"},
			{Name: "db-1", Status: "Running", Ready: "1/1", Restarts: "0", Age: "30d"},
			{Name: "db-2", Status: "Running", Ready: "1/1", Restarts: "0", Age: "30d"},
		}
	case "staging":
		return []ResourceItem{
			{Name: "api-6d4e2c1a-h7j9k", Status: "Running", Ready: "1/1", Restarts: "2", Age: "1d"},
			{Name: "frontend-3a5b7c9d-p2q4r", Status: "Running", Ready: "1/1", Restarts: "0", Age: "3h"},
			{Name: "worker-55c6c6f9f-t6u8v", Status: "CrashLoop", Ready: "0/1", Restarts: "47", Age: "6h"},
			{Name: "db-0", Status: "Running", Ready: "1/1", Restarts: "0", Age: "5d"},
		}
	case "monitoring":
		return []ResourceItem{
			{Name: "prometheus-0", Status: "Running", Ready: "2/2", Restarts: "0", Age: "30d"},
			{Name: "grafana-5c8d7e9f-w1x3y", Status: "Running", Ready: "1/1", Restarts: "0", Age: "15d"},
			{Name: "alertmanager-0", Status: "Running", Ready: "1/1", Restarts: "0", Age: "30d"},
		}
	default:
		return []ResourceItem{
			{Name: "api-7c6c8d5f7d-x8p2k", Status: "CrashLoop", Ready: "1/2", Restarts: "5 (10m)", Age: "2d"},
			{Name: "worker-55c6c6f9f-9mlr", Status: "Pending", Ready: "0/1", Restarts: "0", Age: "3m"},
			{Name: "web-6d9f9f7b7d-2r9kq", Status: "Running", Ready: "2/2", Restarts: "0", Age: "5d"},
			{Name: "web-6d9f9f7b7d-kp4mn", Status: "Running", Ready: "2/2", Restarts: "0", Age: "5d"},
			{Name: "db-0", Status: "Running", Ready: "1/1", Restarts: "0", Age: "12d"},
			{Name: "cache-redis-0", Status: "Running", Ready: "1/1", Restarts: "0", Age: "12d"},
		}
	}
}

func (p *Pods) TableColumns() []TableColumn {
	return namespacedColumns([]TableColumn{
		{Name: "NAME", Width: 48},
		{Name: "STATUS", Width: 12},
		{Name: "READY", Width: 7},
		{Name: "RESTARTS", Width: 14},
		{Name: "AGE", Width: 6},
	})
}

func (p *Pods) TableRow(item ResourceItem) []string {
	return namespacedRow(item.Namespace, []string{item.Name, item.Status, item.Ready, item.Restarts, item.Age})
}

func (p *Pods) Sort(items []ResourceItem) {
	switch p.sortMode {
	case "status":
		problemSort(items, p.sortDesc)
	case "age":
		ageSort(items, p.sortDesc)
	default:
		nameSort(items, p.sortDesc)
	}
}

func (p *Pods) SetSort(mode string, desc bool) { p.sortMode = mode; p.sortDesc = desc }
func (p *Pods) SortMode() string               { return p.sortMode }
func (p *Pods) SortDesc() bool                 { return p.sortDesc }
func (p *Pods) SortKeys() []SortKey {
	return sortKeysFor([]string{"name", "status", "age"})
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
	return expandMockLogs([]string{
		"2025-06-15T12:03:01Z  Starting envoy proxy...",
		"2025-06-15T12:03:01Z  Loading configuration from /etc/envoy/config.yaml",
		"2025-06-15T12:03:02Z  Listener 0.0.0.0:8080 created",
		"2025-06-15T12:03:02Z  Allocating buffer pool (128Mi limit)",
		"2025-06-15T12:03:03Z  ERROR: buffer allocation failed: OOM",
		"2025-06-15T12:03:03Z  Fatal: cannot start with current memory limits",
	}, 120)
}

func (p *Pods) Events(item ResourceItem) []string {
	return []string{
		"10m ago   Warning  BackOff      Back-off restarting failed container sidecar",
		"12m ago   Normal   Pulled       Successfully pulled image \"envoy:1.28\"",
		"15m ago   Warning  OOMKilling   Memory capped at 128Mi",
	}
}

func (p *Pods) Describe(item ResourceItem) string {
	return "Name:             " + item.Name + "\n" +
		"Namespace:        " + ActiveNamespace + "\n" +
		"Priority:         0\n" +
		"Service Account:  default\n" +
		"Node:             worker-03/10.0.1.13\n" +
		"Start Time:       Sat, 19 Feb 2026 08:12:00 +0000\n" +
		"Labels:           app=api\n" +
		"                  tier=backend\n" +
		"                  env=prod\n" +
		"                  pod-template-hash=7c6c8d5f7d\n" +
		"Status:           " + item.Status + "\n" +
		"IP:               10.244.2.15\n" +
		"Controlled By:    ReplicaSet/api-7c6c8d5f7d\n" +
		"Containers:\n" +
		"  api:\n" +
		"    Image:          myco/api:v2.3.1\n" +
		"    Port:           8080/TCP\n" +
		"    State:          Running\n" +
		"      Started:      Sat, 19 Feb 2026 08:12:10 +0000\n" +
		"    Ready:          True\n" +
		"    Restart Count:  0\n" +
		"    Limits:\n" +
		"      cpu:     1\n" +
		"      memory:  512Mi\n" +
		"    Requests:\n" +
		"      cpu:     250m\n" +
		"      memory:  256Mi\n" +
		"    Liveness:   http-get http://:8080/healthz delay=15s period=10s\n" +
		"    Readiness:  http-get http://:8080/readyz delay=5s period=5s\n" +
		"    Mounts:\n" +
		"      /etc/api from config (ro)\n" +
		"  sidecar:\n" +
		"    Image:          envoy:1.28\n" +
		"    Port:           9901/TCP\n" +
		"    State:          CrashLoopBackOff\n" +
		"      Reason:       OOMKilled\n" +
		"    Ready:          False\n" +
		"    Restart Count:  " + item.Restarts + "\n" +
		"    Limits:\n" +
		"      cpu:     200m\n" +
		"      memory:  128Mi\n" +
		"    Requests:\n" +
		"      cpu:     100m\n" +
		"      memory:  64Mi\n" +
		"Conditions:\n" +
		"  Type              Status\n" +
		"  Ready             False\n" +
		"  ContainersReady   False\n" +
		"  PodScheduled      True\n" +
		"QOS Class:        Burstable\n" +
		"Events:\n" +
		"  Type     Reason      Age   Message\n" +
		"  ----     ------      ---   -------\n" +
		"  Warning  BackOff     10m   Back-off restarting failed container sidecar\n" +
		"  Normal   Pulled      12m   Successfully pulled image \"envoy:1.28\"\n" +
		"  Warning  OOMKilling  15m   Memory capped at 128Mi"
}

func (p *Pods) YAML(item ResourceItem) string {
	return strings.TrimSpace(`apiVersion: v1
kind: Pod
metadata:
  name: ` + item.Name + `
  namespace: ` + ActiveNamespace + `
  labels:
    app: api
    tier: backend
    env: prod
    pod-template-hash: 7c6c8d5f7d
  ownerReferences:
  - apiVersion: apps/v1
    kind: ReplicaSet
    name: api-7c6c8d5f7d
    uid: 3a4b5c6d-7e8f-9a0b-1c2d-3e4f5a6b7c8d
spec:
  nodeName: worker-03
  serviceAccountName: default
  restartPolicy: Always
  terminationGracePeriodSeconds: 30
  dnsPolicy: ClusterFirst
  containers:
  - name: api
    image: myco/api:v2.3.1
    ports:
    - containerPort: 8080
      protocol: TCP
    resources:
      requests:
        cpu: 250m
        memory: 256Mi
      limits:
        cpu: "1"
        memory: 512Mi
    livenessProbe:
      httpGet:
        path: /healthz
        port: 8080
      initialDelaySeconds: 15
      periodSeconds: 10
    readinessProbe:
      httpGet:
        path: /readyz
        port: 8080
      initialDelaySeconds: 5
      periodSeconds: 5
    volumeMounts:
    - name: config
      mountPath: /etc/api
      readOnly: true
  - name: sidecar
    image: envoy:1.28
    ports:
    - containerPort: 9901
      protocol: TCP
    resources:
      requests:
        cpu: 100m
        memory: 64Mi
      limits:
        cpu: 200m
        memory: 128Mi
  volumes:
  - name: config
    configMap:
      name: api-gateway-config
status:
  phase: ` + item.Status + `
  podIP: 10.244.2.15
  hostIP: 10.0.1.13
  startTime: "2026-02-19T08:12:00Z"
  qosClass: Burstable
  conditions:
  - type: Ready
    status: "True"
    lastTransitionTime: "2026-02-19T08:12:30Z"
  - type: ContainersReady
    status: "True"
    lastTransitionTime: "2026-02-19T08:12:30Z"
  - type: PodScheduled
    status: "True"
    lastTransitionTime: "2026-02-19T08:12:00Z"
  containerStatuses:
  - name: api
    ready: true
    restartCount: 0
    state:
      running:
        startedAt: "2026-02-19T08:12:10Z"
    image: myco/api:v2.3.1
    imageID: docker-pullable://myco/api@sha256:a1b2c3d4e5f6
  - name: sidecar
    ready: true
    restartCount: ` + item.Restarts + `
    state:
      running:
        startedAt: "2026-02-19T08:12:12Z"
    image: envoy:1.28
    imageID: docker-pullable://envoy@sha256:f6e5d4c3b2a1`)
}
