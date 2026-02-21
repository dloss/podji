package resources

import (
	"fmt"
	"strings"
)

type Services struct{}

func (s *Services) TableColumns() []TableColumn {
	return []TableColumn{
		{Name: "NAME", Width: 30},
		{Name: "TYPE", Width: 14},
		{Name: "CLUSTER-IP", Width: 16},
		{Name: "ENDPOINTS", Width: 14},
		{Name: "AGE", Width: 6},
	}
}

func (s *Services) TableRow(item ResourceItem) []string {
	svcType := item.Kind
	if svcType == "" {
		svcType = "ClusterIP"
	}
	return []string{item.Name, svcType, serviceClusterIP(item.Name, svcType), item.Ready, item.Age}
}

func serviceClusterIP(name string, svcType string) string {
	if svcType == "LoadBalancer" {
		return "10.96.0.10"
	}
	// Simple hash to generate a stable, realistic-looking cluster IP.
	var h byte
	for i := 0; i < len(name); i++ {
		h = h*31 + name[i]
	}
	return fmt.Sprintf("10.96.%d.%d", int(h)%256, int(h)%200+1)
}

func NewServices() *Services {
	return &Services{}
}

func (s *Services) Name() string { return "services" }
func (s *Services) Key() rune   { return 'S' }

func (s *Services) Items() []ResourceItem {
	items := serviceItemsForNamespace(ActiveNamespace)
	s.Sort(items)
	return items
}

func serviceItemsForNamespace(ns string) []ResourceItem {
	switch ns {
	case "production":
		return []ResourceItem{
			{Name: "api-gateway", Kind: "ClusterIP", Status: "Healthy", Ready: "3 endpoints", Age: "14d"},
			{Name: "frontend", Kind: "ClusterIP", Status: "Healthy", Ready: "4 endpoints", Age: "7d"},
			{Name: "auth-service", Kind: "ClusterIP", Status: "Healthy", Ready: "2 endpoints", Age: "21d"},
			{Name: "postgres", Kind: "ClusterIP", Status: "Healthy", Ready: "3 endpoints", Age: "30d"},
			{Name: "redis-master", Kind: "ClusterIP", Status: "Healthy", Ready: "2 endpoints", Age: "30d"},
			{Name: "ingress-external", Kind: "LoadBalancer", Status: "Healthy", Ready: "1 endpoint", Age: "60d"},
		}
	case "staging":
		return []ResourceItem{
			{Name: "api-gateway", Kind: "ClusterIP", Status: "Healthy", Ready: "1 endpoint", Age: "5d"},
			{Name: "frontend", Kind: "ClusterIP", Status: "Healthy", Ready: "1 endpoint", Age: "3h"},
			{Name: "postgres", Kind: "ClusterIP", Status: "Healthy", Ready: "1 endpoint", Age: "5d"},
		}
	case "monitoring":
		return []ResourceItem{
			{Name: "prometheus", Kind: "ClusterIP", Status: "Healthy", Ready: "1 endpoint", Age: "30d"},
			{Name: "grafana", Kind: "ClusterIP", Status: "Healthy", Ready: "1 endpoint", Age: "15d"},
			{Name: "alertmanager", Kind: "ClusterIP", Status: "Healthy", Ready: "1 endpoint", Age: "30d"},
		}
	default:
		return []ResourceItem{
			{Name: "api-gateway", Kind: "ClusterIP", Status: "Healthy", Ready: "3 endpoints", Age: "14d"},
			{Name: "frontend", Kind: "ClusterIP", Status: "Healthy", Ready: "2 endpoints", Age: "7d"},
			{Name: "auth-service", Kind: "ClusterIP", Status: "Healthy", Ready: "2 endpoints", Age: "21d"},
			{Name: "payment-service", Kind: "ClusterIP", Status: "Warning", Ready: "1 endpoint", Age: "5d"},
			{Name: "postgres", Kind: "ClusterIP", Status: "Healthy", Ready: "1 endpoint", Age: "30d"},
			{Name: "redis-master", Kind: "ClusterIP", Status: "Healthy", Ready: "1 endpoint", Age: "30d"},
			{Name: "ingress-external", Kind: "LoadBalancer", Status: "Healthy", Ready: "1 endpoint", Age: "60d"},
			{Name: "kubernetes", Kind: "ClusterIP", Status: "Healthy", Ready: "1 endpoint", Age: "180d"},
		}
	}
}

func (s *Services) Sort(items []ResourceItem) {
	defaultSort(items)
}

func (s *Services) Detail(item ResourceItem) DetailData {
	svcType := item.Kind
	if svcType == "" {
		svcType = "ClusterIP"
	}
	clusterIP := serviceClusterIP(item.Name, svcType)

	return DetailData{
		StatusLine: item.Status + "    type: " + svcType + "    clusterIP: " + clusterIP + "    ports: 80/TCP",
		Events: []string{
			"—   No recent events",
		},
		Labels: []string{
			"app=" + item.Name,
		},
	}
}

func (s *Services) Logs(item ResourceItem) []string {
	return []string{
		"Logs are not available for services.",
	}
}

func (s *Services) Events(item ResourceItem) []string {
	if item.Kind == "LoadBalancer" {
		return []string{
			"2d ago   Normal   EnsuredLoadBalancer   Load balancer provisioned successfully",
		}
	}
	return []string{"—   No recent events"}
}

func (s *Services) YAML(item ResourceItem) string {
	svcType := item.Kind
	if svcType == "" {
		svcType = "ClusterIP"
	}
	return strings.TrimSpace(`apiVersion: v1
kind: Service
metadata:
  name: ` + item.Name + `
spec:
  type: ` + svcType + `
  ports:
  - port: 80
    targetPort: 8080`)
}
