package resources

import (
	"fmt"
	"strings"
)

type ConfigMaps struct {
	sortMode string
	sortDesc bool
}

func (c *ConfigMaps) TableColumns() []TableColumn {
	return []TableColumn{
		{ID: "name", Name: "NAME", Width: 48, Default: true},
		{ID: "data", Name: "DATA", Width: 8, Default: true},
		{ID: "age", Name: "AGE", Width: 6, Default: true},
	}
}

func (c *ConfigMaps) TableRow(item ResourceItem) map[string]string {
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
	return map[string]string{
		"name": item.Name,
		"data": fmt.Sprintf("%d", dataCount),
		"age":  item.Age,
	}
}

func NewConfigMaps() *ConfigMaps {
	return &ConfigMaps{sortMode: "name"}
}

func (c *ConfigMaps) Name() string { return "configmaps" }
func (c *ConfigMaps) Key() rune    { return 'C' }

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
	items = expandMockItems(items, 24)
	c.Sort(items)
	return items
}

func (c *ConfigMaps) Sort(items []ResourceItem) {
	switch c.sortMode {
	case "status":
		problemSort(items, c.sortDesc)
	case "age":
		ageSort(items, c.sortDesc)
	default:
		nameSort(items, c.sortDesc)
	}
}

func (c *ConfigMaps) SetSort(mode string, desc bool) { c.sortMode = mode; c.sortDesc = desc }
func (c *ConfigMaps) SortMode() string               { return c.sortMode }
func (c *ConfigMaps) SortDesc() bool                 { return c.sortDesc }
func (c *ConfigMaps) SortKeys() []SortKey {
	return sortKeysFor([]string{"name", "status", "age"})
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
	return expandMockLogs([]string{
		"Logs are not available for configmaps.",
	}, 30)
}

func (c *ConfigMaps) Events(item ResourceItem) []string {
	return []string{"—   No recent events"}
}

func (c *ConfigMaps) Describe(item ResourceItem) string {
	return "Name:         " + item.Name + "\n" +
		"Namespace:    " + ActiveNamespace + "\n" +
		"Labels:       app.kubernetes.io/managed-by=helm\n" +
		"              app.kubernetes.io/part-of=" + item.Name + "\n" +
		"Annotations:  meta.helm.sh/release-name: " + item.Name + "\n" +
		"\n" +
		"Data\n" +
		"====\n" +
		"config.yaml:\n" +
		"----\n" +
		"server:\n" +
		"  port: 8080\n" +
		"  readTimeout: 30s\n" +
		"logging:\n" +
		"  level: info\n" +
		"\n" +
		"database.yaml:\n" +
		"----\n" +
		"host: postgres." + ActiveNamespace + ".svc.cluster.local\n" +
		"port: 5432\n" +
		"\n" +
		"features.json:\n" +
		"----\n" +
		"{\"enableNewUI\": true, \"enableBetaAPI\": false}\n" +
		"\n" +
		"Events:  <none>"
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
