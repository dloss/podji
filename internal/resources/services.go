package resources

import (
	"fmt"
	"sort"
	"strings"
)

type Services struct {
	namespaceScope
	sortMode string
	sortDesc bool
}

func (s *Services) TableColumns() []TableColumn {
	return namespacedColumnsFor(s.Namespace(), []TableColumn{
		{ID: "name", Name: "NAME", Width: 28, Default: true},
		{ID: "type", Name: "TYPE", Width: 12, Default: true},
		{ID: "cluster-ip", Name: "CLUSTER-IP", Width: 15, Default: true},
		{ID: "external-ip", Name: "EXTERNAL-IP", Width: 15, Default: true},
		{ID: "port", Name: "PORT(S)", Width: 20, Default: true},
		{ID: "endpoints", Name: "ENDPOINTS", Width: 12, Default: true},
		{ID: "session-affinity", Name: "SESSION-AFFINITY", Width: 18, Default: false},
		{ID: "age", Name: "AGE", Width: 6, Default: true},
	})
}

func (s *Services) TableRow(item ResourceItem) map[string]string {
	svcType := item.Kind
	if svcType == "" {
		svcType = "ClusterIP"
	}
	externalIP := item.Extra["external-ip"]
	if externalIP == "" {
		externalIP = "<none>"
	}
	return map[string]string{
		"namespace":   item.Namespace,
		"name":        item.Name,
		"type":        svcType,
		"cluster-ip":  serviceClusterIP(item.Name, svcType),
		"external-ip": externalIP,
		"port":        servicePort(item.Name, svcType),
		"endpoints":   item.Ready,
		"session-affinity": serviceSessionAffinity(item.Name),
		"age":         item.Age,
	}
}

func (s *Services) TableColumnsWide() []TableColumn {
	return namespacedColumnsFor(s.Namespace(), []TableColumn{
		{ID: "name", Name: "NAME", Width: 28, Default: true},
		{ID: "type", Name: "TYPE", Width: 12, Default: true},
		{ID: "cluster-ip", Name: "CLUSTER-IP", Width: 15, Default: true},
		{ID: "external-ip", Name: "EXTERNAL-IP", Width: 15, Default: true},
		{ID: "port", Name: "PORT(S)", Width: 20, Default: true},
		{ID: "endpoints", Name: "ENDPOINTS", Width: 12, Default: true},
		{ID: "session-affinity", Name: "SESSION-AFFINITY", Width: 18, Default: false},
		{ID: "age", Name: "AGE", Width: 6, Default: true},
		{ID: "selector", Name: "SELECTOR", Width: 24, Default: false},
		{ID: "target-port", Name: "TARGET-PORT", Width: 16, Default: false},
	})
}

func (s *Services) TableRowWide(item ResourceItem) map[string]string {
	row := s.TableRow(item)
	row["selector"] = item.Extra["selector"]
	row["target-port"] = serviceTargetPort(item.Name)
	return row
}

func servicePort(name, svcType string) string {
	switch name {
	case "postgres", "db":
		return "5432/TCP"
	case "redis-master", "redis", "redis-slave", "cache-redis":
		return "6379/TCP"
	case "prometheus":
		return "9090/TCP"
	case "grafana":
		return "3000/TCP"
	case "alertmanager":
		return "9093/TCP"
	case "kubernetes":
		return "443/TCP"
	case "ingress-external", "ingress-nginx":
		return "80:31080/TCP,443:31443/TCP"
	default:
		return "80/TCP"
	}
}

func serviceTargetPort(name string) string {
	switch name {
	case "postgres", "db":
		return "5432/TCP"
	case "redis-master", "redis", "cache-redis":
		return "6379/TCP"
	case "prometheus":
		return "9090/TCP"
	case "grafana":
		return "3000/TCP"
	case "alertmanager":
		return "9093/TCP"
	case "kubernetes":
		return "6443/TCP"
	case "ingress-external", "ingress-nginx":
		return "80/TCP,443/TCP"
	default:
		return "8080/TCP"
	}
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

func serviceSessionAffinity(name string) string {
	switch name {
	case "api-gateway", "frontend":
		return "None"
	default:
		return "None"
	}
}

func NewServices() *Services {
	return &Services{namespaceScope: newNamespaceScope(), sortMode: "name"}
}

func (s *Services) Name() string { return "services" }
func (s *Services) Key() rune    { return 'S' }

func (s *Services) Items() []ResourceItem {
	var items []ResourceItem
	if s.Namespace() == AllNamespaces {
		items = allNamespaceItems(serviceItemsForNamespace)
	} else {
		items = serviceItemsForNamespace(s.Namespace())
		items = expandMockItems(items, 26)
	}
	s.Sort(items)
	return items
}

func serviceItemsForNamespace(ns string) []ResourceItem {
	switch ns {
	case "production":
		return []ResourceItem{
			{Name: "api-gateway", Kind: "ClusterIP", Status: "Healthy", Ready: "3 endpoints", Age: "14d", Selector: map[string]string{"app": "api"}, Extra: map[string]string{"external-ip": "<none>", "selector": "app=api"}},
			{Name: "frontend", Kind: "ClusterIP", Status: "Healthy", Ready: "4 endpoints", Age: "7d", Selector: map[string]string{"app": "frontend"}, Extra: map[string]string{"external-ip": "<none>", "selector": "app=frontend"}},
			{Name: "auth-service", Kind: "ClusterIP", Status: "Healthy", Ready: "2 endpoints", Age: "21d", Selector: map[string]string{"app": "auth-service"}, Extra: map[string]string{"external-ip": "<none>", "selector": "app=auth-service"}},
			{Name: "postgres", Kind: "ClusterIP", Status: "Healthy", Ready: "3 endpoints", Age: "30d", Selector: map[string]string{"app": "db"}, Extra: map[string]string{"external-ip": "<none>", "selector": "app=db"}},
			{Name: "redis-master", Kind: "ClusterIP", Status: "Healthy", Ready: "2 endpoints", Age: "30d", Selector: map[string]string{"app": "cache"}, Extra: map[string]string{"external-ip": "<none>", "selector": "app=cache"}},
			{Name: "ingress-external", Kind: "LoadBalancer", Status: "Healthy", Ready: "1 endpoint", Age: "60d", Extra: map[string]string{"external-ip": "203.0.113.10", "selector": "<none>"}},
		}
	case "staging":
		return []ResourceItem{
			{Name: "api-gateway", Kind: "ClusterIP", Status: "Healthy", Ready: "1 endpoint", Age: "5d", Selector: map[string]string{"app": "api"}, Extra: map[string]string{"external-ip": "<none>", "selector": "app=api"}},
			{Name: "frontend", Kind: "ClusterIP", Status: "Healthy", Ready: "1 endpoint", Age: "3h", Selector: map[string]string{"app": "frontend"}, Extra: map[string]string{"external-ip": "<none>", "selector": "app=frontend"}},
			{Name: "postgres", Kind: "ClusterIP", Status: "Healthy", Ready: "1 endpoint", Age: "5d", Selector: map[string]string{"app": "db"}, Extra: map[string]string{"external-ip": "<none>", "selector": "app=db"}},
		}
	case "monitoring":
		return []ResourceItem{
			{Name: "prometheus", Kind: "ClusterIP", Status: "Healthy", Ready: "1 endpoint", Age: "30d", Selector: map[string]string{"app": "prometheus"}, Extra: map[string]string{"external-ip": "<none>", "selector": "app=prometheus"}},
			{Name: "grafana", Kind: "ClusterIP", Status: "Healthy", Ready: "1 endpoint", Age: "15d", Selector: map[string]string{"app": "grafana"}, Extra: map[string]string{"external-ip": "<none>", "selector": "app=grafana"}},
			{Name: "alertmanager", Kind: "ClusterIP", Status: "Healthy", Ready: "1 endpoint", Age: "30d", Selector: map[string]string{"app": "alertmanager"}, Extra: map[string]string{"external-ip": "<none>", "selector": "app=alertmanager"}},
		}
	default:
		return []ResourceItem{
			{Name: "api-gateway", Kind: "ClusterIP", Status: "Healthy", Ready: "3 endpoints", Age: "14d", Selector: map[string]string{"app": "api"}, Extra: map[string]string{"external-ip": "<none>", "selector": "app=api"}},
			{Name: "frontend", Kind: "ClusterIP", Status: "Healthy", Ready: "2 endpoints", Age: "7d", Selector: map[string]string{"app": "frontend"}, Extra: map[string]string{"external-ip": "<none>", "selector": "app=frontend"}},
			{Name: "auth-service", Kind: "ClusterIP", Status: "Healthy", Ready: "2 endpoints", Age: "21d", Selector: map[string]string{"app": "auth-service"}, Extra: map[string]string{"external-ip": "<none>", "selector": "app=auth-service"}},
			{Name: "payment-service", Kind: "ClusterIP", Status: "Warning", Ready: "1 endpoint", Age: "5d", Selector: map[string]string{"app": "payment-service"}, Extra: map[string]string{"external-ip": "<none>", "selector": "app=payment-service"}},
			{Name: "postgres", Kind: "ClusterIP", Status: "Healthy", Ready: "1 endpoint", Age: "30d", Selector: map[string]string{"app": "db"}, Extra: map[string]string{"external-ip": "<none>", "selector": "app=db"}},
			{Name: "redis-master", Kind: "ClusterIP", Status: "Healthy", Ready: "1 endpoint", Age: "30d", Selector: map[string]string{"app": "cache-redis"}, Extra: map[string]string{"external-ip": "<none>", "selector": "app=cache-redis"}},
			{Name: "ingress-external", Kind: "LoadBalancer", Status: "Healthy", Ready: "1 endpoint", Age: "60d", Extra: map[string]string{"external-ip": "203.0.113.10", "selector": "<none>"}},
			{Name: "kubernetes", Kind: "ClusterIP", Status: "Healthy", Ready: "1 endpoint", Age: "180d", Extra: map[string]string{"external-ip": "<none>", "selector": "<none>"}},
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
	case "session-affinity":
		serviceSessionAffinitySort(items, s.sortDesc)
	default:
		nameSort(items, s.sortDesc)
	}
}

func (s *Services) SetSort(mode string, desc bool) { s.sortMode = mode; s.sortDesc = desc }
func (s *Services) SortMode() string               { return s.sortMode }
func (s *Services) SortDesc() bool                 { return s.sortDesc }
func (s *Services) SortKeys() []SortKey {
	return sortKeysFor([]string{"name", "status", "kind", "session-affinity", "age"})
}

func serviceSessionAffinitySort(items []ResourceItem, desc bool) {
	sort.SliceStable(items, func(i, j int) bool {
		ai := serviceSessionAffinity(items[i].Name)
		aj := serviceSessionAffinity(items[j].Name)
		if ai != aj {
			if desc {
				return ai > aj
			}
			return ai < aj
		}
		return items[i].Name < items[j].Name
	})
}

func (s *Services) Detail(item ResourceItem) DetailData {
	svcType := item.Kind
	if svcType == "" {
		svcType = "ClusterIP"
	}
	clusterIP := serviceClusterIP(item.Name, svcType)

	return DetailData{
		Summary: []SummaryField{
			{Key: "status", Label: "Status", Value: item.Status},
			{Key: "type", Label: "Type", Value: svcType},
			{Key: "cluster-ip", Label: "Cluster IP", Value: clusterIP},
			{Key: "ports", Label: "Ports", Value: "80/TCP"},
		},
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
		"Namespace:         " + s.Namespace() + "\n" +
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
  namespace: ` + s.Namespace() + `
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
