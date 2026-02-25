package resources

import (
	"fmt"
	"strings"
)

// CRDMeta holds API metadata for a Custom Resource Definition.
type CRDMeta struct {
	Group      string
	Version    string
	Kind       string
	Namespaced bool
}

// CRDResource implements ResourceType for an arbitrary CRD, using stub data.
type CRDResource struct {
	meta CRDMeta
}

// NewCRDResource creates a CRDResource for the given CRD metadata.
func NewCRDResource(meta CRDMeta) *CRDResource {
	return &CRDResource{meta: meta}
}

// Name returns a qualified resource name (e.g. "certificates.cert-manager.io").
func (c *CRDResource) Name() string {
	kind := strings.ToLower(c.meta.Kind)
	if c.meta.Group == "" {
		return kind + "s"
	}
	return kind + "s." + c.meta.Group
}

// Key returns 0 — CRD resources have no single-letter hotkey.
func (c *CRDResource) Key() rune { return 0 }

func (c *CRDResource) Items() []ResourceItem {
	return stubCRDItems(c.meta)
}

func (c *CRDResource) Sort(items []ResourceItem) { defaultSort(items) }

func (c *CRDResource) Detail(item ResourceItem) DetailData {
	return DetailData{
		StatusLine: fmt.Sprintf("%s/%s  %s/%s",
			c.meta.Group, c.meta.Version,
			strings.ToLower(c.meta.Kind), item.Name),
	}
}

func (c *CRDResource) Logs(item ResourceItem) []string   { return nil }
func (c *CRDResource) Events(item ResourceItem) []string { return nil }

func (c *CRDResource) YAML(item ResourceItem) string {
	group := c.meta.Group
	if group == "" {
		group = "example.io"
	}
	ns := ""
	if c.meta.Namespaced {
		ns = "\n  namespace: " + ActiveNamespace
	}
	return strings.TrimSpace(fmt.Sprintf(`apiVersion: %s/%s
kind: %s
metadata:
  name: %s%s
spec: {}
status:
  conditions:
  - type: Ready
    status: "True"`, group, c.meta.Version, c.meta.Kind, item.Name, ns))
}

func (c *CRDResource) Describe(item ResourceItem) string {
	group := c.meta.Group
	if group == "" {
		group = "example.io"
	}
	nsLine := ""
	if c.meta.Namespaced {
		nsLine = "\nNamespace:  " + ActiveNamespace
	}
	return fmt.Sprintf("Name:       %s%s\nGroup:      %s\nVersion:    %s\nKind:       %s\nStatus:     %s",
		item.Name, nsLine, group, c.meta.Version, c.meta.Kind, item.Status)
}

// TableColumns implements TableResource for custom column headers.
func (c *CRDResource) TableColumns() []TableColumn {
	return []TableColumn{
		{Name: "NAME", Width: 40},
		{Name: "STATUS", Width: 10},
		{Name: "AGE", Width: 6},
	}
}

// TableRow implements TableResource.
func (c *CRDResource) TableRow(item ResourceItem) []string {
	return []string{item.Name, item.Status, item.Age}
}

func stubCRDItems(meta CRDMeta) []ResourceItem {
	switch meta.Kind {
	case "Certificate":
		return []ResourceItem{
			{Name: "api-tls", Status: "Ready", Age: "30d"},
			{Name: "frontend-tls", Status: "Ready", Age: "14d"},
			{Name: "wildcard-cert", Status: "Pending", Age: "2m"},
		}
	case "ClusterIssuer":
		return []ResourceItem{
			{Name: "letsencrypt-prod", Status: "Ready", Age: "90d"},
			{Name: "letsencrypt-staging", Status: "Ready", Age: "90d"},
		}
	case "Issuer":
		return []ResourceItem{
			{Name: "selfsigned-issuer", Status: "Ready", Age: "30d"},
		}
	case "VirtualService":
		return []ResourceItem{
			{Name: "api-vs", Status: "Ready", Age: "20d"},
			{Name: "frontend-vs", Status: "Ready", Age: "20d"},
		}
	case "DestinationRule":
		return []ResourceItem{
			{Name: "api-dr", Status: "Ready", Age: "20d"},
		}
	case "Gateway":
		return []ResourceItem{
			{Name: "ingress-gateway", Status: "Ready", Age: "45d"},
		}
	case "Application":
		return []ResourceItem{
			{Name: "api", Status: "Healthy", Age: "14d"},
			{Name: "frontend", Status: "Degraded", Age: "14d"},
			{Name: "infra", Status: "Healthy", Age: "30d"},
		}
	case "AppProject":
		return []ResourceItem{
			{Name: "default", Status: "Ready", Age: "90d"},
			{Name: "production", Status: "Ready", Age: "45d"},
		}
	case "ExternalSecret":
		return []ResourceItem{
			{Name: "db-credentials", Status: "Ready", Age: "5d"},
			{Name: "api-keys", Status: "Ready", Age: "5d"},
		}
	case "ClusterSecretStore":
		return []ResourceItem{
			{Name: "vault-backend", Status: "Ready", Age: "60d"},
		}
	case "PrometheusRule":
		return []ResourceItem{
			{Name: "api-alerts", Status: "Ready", Age: "30d"},
			{Name: "infra-alerts", Status: "Ready", Age: "30d"},
		}
	case "ServiceMonitor":
		return []ResourceItem{
			{Name: "api-monitor", Status: "Ready", Age: "30d"},
			{Name: "worker-monitor", Status: "Ready", Age: "30d"},
		}
	case "ScaledObject":
		return []ResourceItem{
			{Name: "api-scaler", Status: "Ready", Age: "7d"},
		}
	case "HelmRepository":
		return []ResourceItem{
			{Name: "bitnami", Status: "Ready", Age: "90d"},
			{Name: "ingress-nginx", Status: "Ready", Age: "90d"},
		}
	case "HelmRelease":
		return []ResourceItem{
			{Name: "nginx-ingress", Status: "Ready", Age: "30d"},
			{Name: "cert-manager", Status: "Ready", Age: "30d"},
		}
	default:
		kind := strings.ToLower(meta.Kind)
		return []ResourceItem{
			{Name: kind + "-sample-1", Status: "Ready", Age: "7d"},
			{Name: kind + "-sample-2", Status: "Ready", Age: "3d"},
		}
	}
}

// StubCRDs returns a representative set of well-known CRDs for demo/stub mode.
func StubCRDs() []CRDMeta {
	return []CRDMeta{
		// cert-manager
		{Group: "cert-manager.io", Version: "v1", Kind: "Certificate", Namespaced: true},
		{Group: "cert-manager.io", Version: "v1", Kind: "ClusterIssuer", Namespaced: false},
		{Group: "cert-manager.io", Version: "v1", Kind: "Issuer", Namespaced: true},
		// Istio
		{Group: "networking.istio.io", Version: "v1beta1", Kind: "VirtualService", Namespaced: true},
		{Group: "networking.istio.io", Version: "v1beta1", Kind: "DestinationRule", Namespaced: true},
		{Group: "networking.istio.io", Version: "v1beta1", Kind: "Gateway", Namespaced: true},
		// ArgoCD
		{Group: "argoproj.io", Version: "v1alpha1", Kind: "Application", Namespaced: true},
		{Group: "argoproj.io", Version: "v1alpha1", Kind: "AppProject", Namespaced: false},
		// external-secrets
		{Group: "external-secrets.io", Version: "v1beta1", Kind: "ExternalSecret", Namespaced: true},
		{Group: "external-secrets.io", Version: "v1beta1", Kind: "ClusterSecretStore", Namespaced: false},
		// Prometheus Operator
		{Group: "monitoring.coreos.com", Version: "v1", Kind: "PrometheusRule", Namespaced: true},
		{Group: "monitoring.coreos.com", Version: "v1", Kind: "ServiceMonitor", Namespaced: true},
		// KEDA
		{Group: "keda.sh", Version: "v1alpha1", Kind: "ScaledObject", Namespaced: true},
		// Flux
		{Group: "source.toolkit.fluxcd.io", Version: "v1", Kind: "HelmRepository", Namespaced: true},
		{Group: "helm.toolkit.fluxcd.io", Version: "v2", Kind: "HelmRelease", Namespaced: true},
	}
}
