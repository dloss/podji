package resources

import (
	"fmt"
	"strings"
)

type Services struct {
	sortMode string
	sortDesc bool
}

func (s *Services) TableColumns() []TableColumn {
	return namespacedColumns([]TableColumn{
		{Name: "NAME", Width: 30},
		{Name: "TYPE", Width: 14},
		{Name: "CLUSTER-IP", Width: 16},
		{Name: "ENDPOINTS", Width: 14},
		{Name: "AGE", Width: 6},
	})
}

func (s *Services) TableRow(item ResourceItem) []string {
	svcType := item.Kind
	if svcType == "" {
		svcType = "ClusterIP"
	}
	return namespacedRow(item.Namespace, []string{item.Name, svcType, serviceClusterIP(item.Name, svcType), item.Ready, item.Age})
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
	return &Services{sortMode: "name"}
}

func (s *Services) Name() string { return "services" }
func (s *Services) Key() rune    { return 'S' }

func (s *Services) Items() []ResourceItem {
	var items []ResourceItem
	if ActiveNamespace == AllNamespaces {
		items = allNamespaceItems(serviceItemsForNamespace)
	} else {
		items = serviceItemsForNamespace(ActiveNamespace)
		items = expandMockItems(items, 26)
	}
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
	switch s.sortMode {
	case "status":
		problemSort(items, s.sortDesc)
	case "age":
		ageSort(items, s.sortDesc)
	case "kind":
		kindSort(items, s.sortDesc)
	default:
		nameSort(items, s.sortDesc)
	}
}

func (s *Services) SetSort(mode string, desc bool) { s.sortMode = mode; s.sortDesc = desc }
func (s *Services) SortMode() string               { return s.sortMode }
func (s *Services) SortDesc() bool                 { return s.sortDesc }
func (s *Services) SortKeys() []SortKey {
	return sortKeysFor([]string{"name", "status", "kind", "age"})
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
	return expandMockLogs([]string{
		"Logs are not available for services.",
	}, 32)
}

func (s *Services) Events(item ResourceItem) []string {
	if item.Kind == "LoadBalancer" {
		return []string{
			"2d ago   Normal   EnsuredLoadBalancer   Load balancer provisioned successfully",
		}
	}
	return []string{"—   No recent events"}
}

func (s *Services) Describe(item ResourceItem) string {
	svcType := item.Kind
	if svcType == "" {
		svcType = "ClusterIP"
	}
	clusterIP := serviceClusterIP(item.Name, svcType)
	endpoints := item.Ready
	if endpoints == "" {
		endpoints = "0 endpoints"
	}
	return "Name:              " + item.Name + "\n" +
		"Namespace:         " + ActiveNamespace + "\n" +
		"Labels:            app=" + item.Name + "\n" +
		"                   app.kubernetes.io/managed-by=helm\n" +
		"Annotations:       <none>\n" +
		"Selector:          app=" + item.Name + "\n" +
		"Type:              " + svcType + "\n" +
		"IP Family Policy:  SingleStack\n" +
		"IP Families:       IPv4\n" +
		"IP:                " + clusterIP + "\n" +
		"Port:              http  80/TCP\n" +
		"TargetPort:        8080/TCP\n" +
		"Port:              metrics  9090/TCP\n" +
		"TargetPort:        9090/TCP\n" +
		"Endpoints:         " + endpoints + "\n" +
		"Session Affinity:  None\n" +
		"Events:            <none>"
}

func (s *Services) YAML(item ResourceItem) string {
	svcType := item.Kind
	if svcType == "" {
		svcType = "ClusterIP"
	}
	clusterIP := serviceClusterIP(item.Name, svcType)
	return strings.TrimSpace(`apiVersion: v1
kind: Service
metadata:
  name: ` + item.Name + `
  namespace: ` + ActiveNamespace + `
  labels:
    app: ` + item.Name + `
    app.kubernetes.io/managed-by: helm
spec:
  type: ` + svcType + `
  selector:
    app: ` + item.Name + `
  ports:
  - name: http
    port: 80
    targetPort: 8080
    protocol: TCP
  - name: metrics
    port: 9090
    targetPort: 9090
    protocol: TCP
  clusterIP: ` + clusterIP + `
  sessionAffinity: None
status:
  loadBalancer: {}`)
}
