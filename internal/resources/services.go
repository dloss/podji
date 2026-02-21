package resources

import "strings"

type Services struct{}

func NewServices() *Services {
	return &Services{}
}

func (s *Services) Name() string { return "services" }
func (s *Services) Key() rune   { return 'S' }

func (s *Services) Items() []ResourceItem {
	items := []ResourceItem{
		{Name: "api-gateway", Kind: "ClusterIP", Status: "Healthy", Ready: "3 endpoints", Age: "14d"},
		{Name: "frontend", Kind: "ClusterIP", Status: "Healthy", Ready: "2 endpoints", Age: "7d"},
		{Name: "auth-service", Kind: "ClusterIP", Status: "Healthy", Ready: "2 endpoints", Age: "21d"},
		{Name: "payment-service", Kind: "ClusterIP", Status: "Warning", Ready: "1 endpoint", Age: "5d"},
		{Name: "postgres", Kind: "ClusterIP", Status: "Healthy", Ready: "1 endpoint", Age: "30d"},
		{Name: "redis-master", Kind: "ClusterIP", Status: "Healthy", Ready: "1 endpoint", Age: "30d"},
		{Name: "ingress-external", Kind: "LoadBalancer", Status: "Healthy", Ready: "1 endpoint", Age: "60d"},
		{Name: "kubernetes", Kind: "ClusterIP", Status: "Healthy", Ready: "1 endpoint", Age: "180d"},
	}
	s.Sort(items)
	return items
}

func (s *Services) Sort(items []ResourceItem) {
	defaultSort(items)
}

func (s *Services) Detail(item ResourceItem) DetailData {
	svcType := item.Kind
	if svcType == "" {
		svcType = "ClusterIP"
	}
	clusterIP := "10.96." + item.Name[:1] + ".1"

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
