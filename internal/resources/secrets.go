package resources

import (
	"fmt"
	"strings"
)

type Secrets struct {
	sortMode string
	sortDesc bool
}

func (s *Secrets) TableColumns() []TableColumn {
	return []TableColumn{
		{ID: "name", Name: "NAME", Width: 35, Default: true},
		{ID: "type", Name: "TYPE", Width: 38, Default: true},
		{ID: "data", Name: "DATA", Width: 8, Default: true},
		{ID: "age", Name: "AGE", Width: 6, Default: true},
	}
}

func (s *Secrets) TableRow(item ResourceItem) map[string]string {
	kind := item.Kind
	if kind == "" {
		kind = "Opaque"
	}
	dataCount := 2
	switch item.Name {
	case "default-token-x7m2k":
		dataCount = 3
	case "docker-registry-creds":
		dataCount = 1
	case "api-gateway-tls":
		dataCount = 2
	}
	return map[string]string{
		"name": item.Name,
		"type": kind,
		"data": fmt.Sprintf("%d", dataCount),
		"age":  item.Age,
	}
}

func NewSecrets() *Secrets {
	return &Secrets{sortMode: "name"}
}

func (s *Secrets) Name() string { return "secrets" }
func (s *Secrets) Key() rune    { return 'K' }

func (s *Secrets) Items() []ResourceItem {
	items := []ResourceItem{
		{Name: "api-gateway-tls", Kind: "kubernetes.io/tls", Status: "Healthy", Age: "14d"},
		{Name: "auth-service-credentials", Kind: "Opaque", Status: "Healthy", Age: "21d"},
		{Name: "default-token-x7m2k", Kind: "kubernetes.io/service-account-token", Status: "Healthy", Age: "180d"},
		{Name: "docker-registry-creds", Kind: "kubernetes.io/dockerconfigjson", Status: "Healthy", Age: "90d"},
		{Name: "payment-stripe-key", Kind: "Opaque", Status: "Healthy", Age: "5d"},
		{Name: "postgres-credentials", Kind: "Opaque", Status: "Healthy", Age: "30d"},
		{Name: "redis-password", Kind: "Opaque", Status: "Healthy", Age: "30d"},
	}
	items = expandMockItems(items, 24)
	s.Sort(items)
	return items
}

func (s *Secrets) Sort(items []ResourceItem) {
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

func (s *Secrets) SetSort(mode string, desc bool) { s.sortMode = mode; s.sortDesc = desc }
func (s *Secrets) SortMode() string               { return s.sortMode }
func (s *Secrets) SortDesc() bool                 { return s.sortDesc }
func (s *Secrets) SortKeys() []SortKey {
	return sortKeysFor([]string{"name", "status", "kind", "age"})
}

func (s *Secrets) Detail(item ResourceItem) DetailData {
	kind := item.Kind
	if kind == "" {
		kind = "Opaque"
	}
	return DetailData{
		StatusLine: "Healthy    type: " + kind + "    data-keys: 2    age: " + item.Age,
		Events: []string{
			"—   No recent events",
		},
		Labels: []string{
			"app.kubernetes.io/managed-by=helm",
		},
	}
}

func (s *Secrets) Logs(item ResourceItem) []string {
	return expandMockLogs([]string{
		"Logs are not available for secrets.",
	}, 30)
}

func (s *Secrets) Events(item ResourceItem) []string {
	return []string{"—   No recent events"}
}

func (s *Secrets) Describe(item ResourceItem) string {
	kind := item.Kind
	if kind == "" {
		kind = "Opaque"
	}
	dataKeys := "  username:  8 bytes\n  password:  24 bytes"
	switch kind {
	case "kubernetes.io/tls":
		dataKeys = "  tls.crt:  1164 bytes\n  tls.key:  1704 bytes"
	case "kubernetes.io/service-account-token":
		dataKeys = "  ca.crt:     1066 bytes\n  namespace:  7 bytes\n  token:      832 bytes"
	case "kubernetes.io/dockerconfigjson":
		dataKeys = "  .dockerconfigjson:  186 bytes"
	}
	return "Name:         " + item.Name + "\n" +
		"Namespace:    " + ActiveNamespace + "\n" +
		"Labels:       app.kubernetes.io/managed-by=helm\n" +
		"Annotations:  meta.helm.sh/release-name: " + item.Name + "\n" +
		"\n" +
		"Type:  " + kind + "\n" +
		"\n" +
		"Data\n" +
		"====\n" +
		dataKeys + "\n" +
		"\n" +
		"Events:  <none>"
}

func (s *Secrets) YAML(item ResourceItem) string {
	kind := item.Kind
	if kind == "" {
		kind = "Opaque"
	}
	dataKeys := `  username: <redacted>
  password: <redacted>`
	switch kind {
	case "kubernetes.io/tls":
		dataKeys = `  tls.crt: <redacted>
  tls.key: <redacted>`
	case "kubernetes.io/service-account-token":
		dataKeys = `  ca.crt: <redacted>
  namespace: <redacted>
  token: <redacted>`
	case "kubernetes.io/dockerconfigjson":
		dataKeys = `  .dockerconfigjson: <redacted>`
	}
	return strings.TrimSpace(`apiVersion: v1
kind: Secret
metadata:
  name: ` + item.Name + `
  namespace: ` + ActiveNamespace + `
  labels:
    app.kubernetes.io/managed-by: helm
  annotations:
    meta.helm.sh/release-name: ` + item.Name + `
type: ` + kind + `
data:
` + dataKeys)
}
