package resources

import "strings"

type Deployments struct{}

func (d *Deployments) TableColumns() []TableColumn {
	return []TableColumn{
		{Name: "NAME", Width: 35},
		{Name: "STATUS", Width: 14},
		{Name: "READY", Width: 8},
		{Name: "AGE", Width: 6},
	}
}

func (d *Deployments) TableRow(item ResourceItem) []string {
	return []string{item.Name, item.Status, item.Ready, item.Age}
}

func NewDeployments() *Deployments {
	return &Deployments{}
}

func (d *Deployments) Name() string { return "deployments" }
func (d *Deployments) Key() rune   { return 'D' }

func (d *Deployments) Items() []ResourceItem {
	items := deploymentItemsForNamespace(ActiveNamespace)
	d.Sort(items)
	return items
}

func deploymentItemsForNamespace(ns string) []ResourceItem {
	switch ns {
	case "production":
		return []ResourceItem{
			{Name: "api-gateway", Status: "Healthy", Ready: "3/3", Age: "14d"},
			{Name: "frontend", Status: "Healthy", Ready: "4/4", Age: "7d"},
			{Name: "auth-service", Status: "Healthy", Ready: "2/2", Age: "21d"},
			{Name: "notification-worker", Status: "Healthy", Ready: "2/2", Age: "10d"},
			{Name: "user-service", Status: "Healthy", Ready: "3/3", Age: "3d"},
		}
	case "staging":
		return []ResourceItem{
			{Name: "api-gateway", Status: "Healthy", Ready: "1/1", Age: "5d"},
			{Name: "frontend", Status: "Healthy", Ready: "1/1", Age: "3h"},
			{Name: "search-indexer", Status: "Progressing", Ready: "0/1", Age: "10m"},
		}
	case "monitoring":
		return []ResourceItem{
			{Name: "grafana", Status: "Healthy", Ready: "1/1", Age: "15d"},
			{Name: "kube-state-metrics", Status: "Healthy", Ready: "1/1", Age: "20d"},
		}
	default:
		return []ResourceItem{
			{Name: "api-gateway", Status: "Healthy", Ready: "3/3", Age: "14d"},
			{Name: "frontend", Status: "Healthy", Ready: "2/2", Age: "7d"},
			{Name: "auth-service", Status: "Healthy", Ready: "2/2", Age: "21d"},
			{Name: "payment-service", Status: "Degraded", Ready: "1/2", Age: "5d"},
			{Name: "notification-worker", Status: "Healthy", Ready: "1/1", Age: "10d"},
			{Name: "search-indexer", Status: "Progressing", Ready: "2/3", Age: "45m"},
			{Name: "user-service", Status: "Healthy", Ready: "2/2", Age: "3d"},
		}
	}
}

func (d *Deployments) Sort(items []ResourceItem) {
	defaultSort(items)
}

func (d *Deployments) Detail(item ResourceItem) DetailData {
	return DetailData{
		StatusLine: item.Status + " " + item.Ready + "    strategy: RollingUpdate    revision: 12",
		Conditions: []string{
			"Available = True              Deployment has minimum availability",
			"Progressing = True            ReplicaSet has successfully progressed",
		},
		Events: []string{
			"5m ago   Normal   ScalingReplicaSet   Scaled up replica set " + item.Name + "-7d8f9c to 2",
		},
		Labels: []string{
			"app=" + item.Name,
			"app.kubernetes.io/managed-by=helm",
		},
	}
}

func (d *Deployments) Logs(item ResourceItem) []string {
	return []string{
		"Logs are not available for deployments. View pods instead.",
	}
}

func (d *Deployments) Events(item ResourceItem) []string {
	return []string{
		"5m ago   Normal   ScalingReplicaSet   Scaled up replica set " + item.Name + "-7d8f9c to 2",
		"3d ago   Normal   ScalingReplicaSet   Scaled down replica set " + item.Name + "-6c7e8b to 0",
	}
}

func (d *Deployments) YAML(item ResourceItem) string {
	return strings.TrimSpace(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: ` + item.Name + `
  namespace: ` + ActiveNamespace + `
  labels:
    app: ` + item.Name + `
    app.kubernetes.io/managed-by: helm
    app.kubernetes.io/version: v2.3.1
  annotations:
    deployment.kubernetes.io/revision: "12"
    meta.helm.sh/release-name: ` + item.Name + `
spec:
  replicas: 2
  revisionHistoryLimit: 10
  selector:
    matchLabels:
      app: ` + item.Name + `
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 25%
      maxUnavailable: 25%
  template:
    metadata:
      labels:
        app: ` + item.Name + `
        tier: backend
    spec:
      serviceAccountName: ` + item.Name + `
      terminationGracePeriodSeconds: 30
      containers:
      - name: ` + item.Name + `
        image: ghcr.io/example/` + item.Name + `:v2.3.1
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
        envFrom:
        - configMapRef:
            name: ` + item.Name + `-config
        - secretRef:
            name: ` + item.Name + `-credentials
status:
  observedGeneration: 12
  replicas: 2
  updatedReplicas: 2
  readyReplicas: 2
  availableReplicas: 2
  conditions:
  - type: Available
    status: "True"
    lastTransitionTime: "2026-02-10T14:00:00Z"
    reason: MinimumReplicasAvailable
    message: Deployment has minimum availability.
  - type: Progressing
    status: "True"
    lastTransitionTime: "2026-02-10T14:00:00Z"
    reason: NewReplicaSetAvailable
    message: ReplicaSet "` + item.Name + `-7d8f9c" has successfully progressed.`)
}
