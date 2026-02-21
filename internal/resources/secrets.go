package resources

import (
	"fmt"
	"strings"
)

type Secrets struct{}

func (s *Secrets) TableColumns() []TableColumn {
	return []TableColumn{
		{Name: "NAME", Width: 35},
		{Name: "TYPE", Width: 38},
		{Name: "DATA", Width: 8},
		{Name: "AGE", Width: 6},
	}
}

func (s *Secrets) TableRow(item ResourceItem) []string {
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
	return []string{item.Name, kind, fmt.Sprintf("%d", dataCount), item.Age}
}

func NewSecrets() *Secrets {
	return &Secrets{}
}

func (s *Secrets) Name() string { return "secrets" }
func (s *Secrets) Key() rune   { return 'K' }

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
	s.Sort(items)
	return items
}

func (s *Secrets) Sort(items []ResourceItem) {
	defaultSort(items)
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
	return []string{
		"Logs are not available for secrets.",
	}
}

func (s *Secrets) Events(item ResourceItem) []string {
	return []string{"—   No recent events"}
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
