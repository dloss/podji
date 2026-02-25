package resources

import (
	"fmt"
	"strings"
)

type Ingresses struct {
	sortMode string
}

func NewIngresses() *Ingresses {
	return &Ingresses{sortMode: "name"}
}

func (g *Ingresses) Name() string { return "ingresses" }
func (g *Ingresses) Key() rune    { return 'I' }

func (g *Ingresses) TableColumns() []TableColumn {
	return []TableColumn{
		{Name: "NAME", Width: 24},
		{Name: "CLASS", Width: 10},
		{Name: "HOSTS", Width: 30},
		{Name: "ADDRESS", Width: 16},
		{Name: "PORTS", Width: 10},
		{Name: "AGE", Width: 6},
	}
}

func (g *Ingresses) TableRow(item ResourceItem) []string {
	// Kind holds the ingress class, Ready holds the hostname.
	class := item.Kind
	if class == "" {
		class = "nginx"
	}
	return []string{item.Name, class, item.Ready, ingressAddress(item.Status), ingressPorts(item.Name), item.Age}
}

func ingressAddress(status string) string {
	if status != "Healthy" {
		return "<pending>"
	}
	return "203.0.113.10"
}

func ingressPorts(name string) string {
	switch name {
	case "api-gateway", "frontend", "admin":
		return "80, 443"
	default:
		return "80, 443"
	}
}

func (g *Ingresses) Items() []ResourceItem {
	items := ingressItemsForNamespace(ActiveNamespace)
	g.Sort(items)
	return items
}

func ingressItemsForNamespace(ns string) []ResourceItem {
	switch ns {
	case "production":
		return []ResourceItem{
			{Name: "admin", Status: "Healthy", Kind: "nginx", Ready: "admin.example.com", Age: "30d"},
			{Name: "api-gateway", Status: "Healthy", Kind: "nginx", Ready: "api.example.com", Age: "14d"},
			{Name: "frontend", Status: "Healthy", Kind: "nginx", Ready: "app.example.com", Age: "7d"},
			{Name: "status-page", Status: "Healthy", Kind: "nginx", Ready: "status.example.com", Age: "60d"},
		}
	case "kube-system":
		return []ResourceItem{
			{Name: "grafana", Status: "Healthy", Kind: "nginx", Ready: "grafana.internal", Age: "30d"},
		}
	default:
		return []ResourceItem{
			{Name: "api-gateway", Status: "Healthy", Kind: "nginx", Ready: "api.example.com", Age: "14d"},
			{Name: "grafana", Status: "Healthy", Kind: "nginx", Ready: "grafana.example.com", Age: "30d"},
		}
	}
}

func (g *Ingresses) Sort(items []ResourceItem) {
	switch g.sortMode {
	case "status":
		problemSort(items)
	case "age":
		ageSort(items)
	default:
		defaultSort(items)
	}
}

func (g *Ingresses) ToggleSort() {
	g.sortMode = cycleSortMode(g.sortMode, []string{"name", "status", "age"})
}

func (g *Ingresses) SortMode() string {
	return g.sortMode
}

func (g *Ingresses) Detail(item ResourceItem) DetailData {
	statusLine := item.Status + "    host: " + item.Ready + "    class: " + item.Kind + "    address: " + ingressAddress(item.Status)
	return DetailData{
		StatusLine: statusLine,
		Events:     []string{"—   No recent events"},
		Labels:     []string{"app.kubernetes.io/managed-by=helm"},
	}
}

func (g *Ingresses) Logs(item ResourceItem) []string {
	return []string{"Logs are not available for ingresses."}
}

func (g *Ingresses) Events(item ResourceItem) []string {
	return []string{"—   No recent events"}
}

func (g *Ingresses) Describe(item ResourceItem) string {
	host := item.Ready
	class := item.Kind
	address := ingressAddress(item.Status)

	return "Name:             " + item.Name + "\n" +
		"Namespace:        " + ActiveNamespace + "\n" +
		"Address:          " + address + "\n" +
		"Ingress Class:    " + class + "\n" +
		"Default backend:  <default>\n" +
		"TLS:\n" +
		"  " + host + " terminates " + item.Name + "-tls\n" +
		"Rules:\n" +
		"  Host              Path  Backends\n" +
		"  ----              ----  --------\n" +
		"  " + fmt.Sprintf("%-18s", host) + "/     " + item.Name + ":80 (10.244.1.22:8080)\n" +
		"Annotations:\n" +
		"  kubernetes.io/ingress.class:                  " + class + "\n" +
		"  nginx.ingress.kubernetes.io/ssl-redirect:     true\n" +
		"  nginx.ingress.kubernetes.io/proxy-body-size:  8m\n" +
		"Events:  <none>"
}

func (g *Ingresses) YAML(item ResourceItem) string {
	host := item.Ready
	class := item.Kind

	return strings.TrimSpace(`apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: ` + item.Name + `
  namespace: ` + ActiveNamespace + `
  labels:
    app.kubernetes.io/managed-by: helm
  annotations:
    kubernetes.io/ingress.class: ` + class + `
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
    nginx.ingress.kubernetes.io/proxy-body-size: 8m
spec:
  ingressClassName: ` + class + `
  tls:
  - hosts:
    - ` + host + `
    secretName: ` + item.Name + `-tls
  rules:
  - host: ` + host + `
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: ` + item.Name + `
            port:
              number: 80
status:
  loadBalancer:
    ingress:
    - ip: 203.0.113.10`)
}
