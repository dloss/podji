package resources

import (
	"fmt"
	"strings"
)

type ConfigMaps struct{}

func (c *ConfigMaps) TableColumns() []TableColumn {
	return []TableColumn{
		{Name: "NAME", Width: 48},
		{Name: "DATA", Width: 8},
		{Name: "AGE", Width: 6},
	}
}

func (c *ConfigMaps) TableRow(item ResourceItem) []string {
	dataCount := 3
	switch item.Name {
	case "coredns":
		dataCount = 1
	case "kube-proxy":
		dataCount = 2
	case "feature-flags":
		dataCount = 8
	case "prometheus-rules":
		dataCount = 5
	case "nginx-config":
		dataCount = 2
	}
	return []string{item.Name, fmt.Sprintf("%d", dataCount), item.Age}
}

func NewConfigMaps() *ConfigMaps {
	return &ConfigMaps{}
}

func (c *ConfigMaps) Name() string { return "configmaps" }
func (c *ConfigMaps) Key() rune   { return 'C' }

func (c *ConfigMaps) Items() []ResourceItem {
	items := []ResourceItem{
		{Name: "api-gateway-config", Status: "Healthy", Age: "14d"},
		{Name: "auth-service-config", Status: "Healthy", Age: "21d"},
		{Name: "coredns", Status: "Healthy", Age: "180d"},
		{Name: "feature-flags", Status: "Healthy", Age: "1d"},
		{Name: "kube-proxy", Status: "Healthy", Age: "180d"},
		{Name: "nginx-config", Status: "Healthy", Age: "60d"},
		{Name: "prometheus-rules", Status: "Healthy", Age: "15d"},
	}
	c.Sort(items)
	return items
}

func (c *ConfigMaps) Sort(items []ResourceItem) {
	defaultSort(items)
}

func (c *ConfigMaps) Detail(item ResourceItem) DetailData {
	return DetailData{
		StatusLine: "Healthy    data-keys: 3    age: " + item.Age,
		Events: []string{
			"—   No recent events",
		},
		Labels: []string{
			"app.kubernetes.io/managed-by=helm",
		},
	}
}

func (c *ConfigMaps) Logs(item ResourceItem) []string {
	return []string{
		"Logs are not available for configmaps.",
	}
}

func (c *ConfigMaps) Events(item ResourceItem) []string {
	return []string{"—   No recent events"}
}

func (c *ConfigMaps) YAML(item ResourceItem) string {
	return strings.TrimSpace(`apiVersion: v1
kind: ConfigMap
metadata:
  name: ` + item.Name + `
  namespace: ` + ActiveNamespace + `
  labels:
    app.kubernetes.io/managed-by: helm
    app.kubernetes.io/part-of: ` + item.Name + `
  annotations:
    meta.helm.sh/release-name: ` + item.Name + `
data:
  config.yaml: |
    server:
      port: 8080
      readTimeout: 30s
      writeTimeout: 30s
    logging:
      level: info
      format: json
    metrics:
      enabled: true
      port: 9090
  database.yaml: |
    host: postgres.` + ActiveNamespace + `.svc.cluster.local
    port: 5432
    sslMode: require
    maxConnections: 25
  features.json: |
    {
      "enableNewUI": true,
      "enableBetaAPI": false,
      "maintenanceMode": false
    }`)
}
