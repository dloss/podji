package resources

import "strings"

type Deployments struct{}

func NewDeployments() *Deployments {
	return &Deployments{}
}

func (d *Deployments) Name() string { return "deployments" }
func (d *Deployments) Key() rune   { return 'D' }

func (d *Deployments) Items() []ResourceItem {
	items := []ResourceItem{
		{Name: "api-gateway", Status: "Healthy", Ready: "3/3", Age: "14d"},
		{Name: "frontend", Status: "Healthy", Ready: "2/2", Age: "7d"},
		{Name: "auth-service", Status: "Healthy", Ready: "2/2", Age: "21d"},
		{Name: "payment-service", Status: "Degraded", Ready: "1/2", Age: "5d"},
		{Name: "notification-worker", Status: "Healthy", Ready: "1/1", Age: "10d"},
		{Name: "search-indexer", Status: "Progressing", Ready: "2/3", Age: "45m"},
		{Name: "user-service", Status: "Healthy", Ready: "2/2", Age: "3d"},
	}
	d.Sort(items)
	return items
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
spec:
  replicas: 2
  selector:
    matchLabels:
      app: ` + item.Name)
}
