package resources

import "strings"

// AllNamespaces is the sentinel value meaning "show resources from all namespaces".
const AllNamespaces = "(all)"

// DefaultNamespace is the default namespace when no explicit scope is set.
const DefaultNamespace = "default"

// NamespaceScoped is implemented by resources that are rendered within a namespace scope.
type NamespaceScoped interface {
	SetNamespace(namespace string)
	Namespace() string
}

type namespaceScope struct {
	namespace string
}

func newNamespaceScope() namespaceScope {
	return namespaceScope{namespace: DefaultNamespace}
}

func (s *namespaceScope) SetNamespace(namespace string) {
	s.namespace = normalizeNamespace(namespace)
}

func (s *namespaceScope) Namespace() string {
	return normalizeNamespace(s.namespace)
}

func normalizeNamespace(namespace string) string {
	ns := strings.TrimSpace(namespace)
	if ns == "" {
		return DefaultNamespace
	}
	return ns
}

// namespacedColumnsFor prepends a NAMESPACE column when in all-namespaces mode.
func namespacedColumnsFor(namespace string, cols []TableColumn) []TableColumn {
	if normalizeNamespace(namespace) != AllNamespaces {
		return cols
	}
	return append([]TableColumn{{ID: "namespace", Name: "NAMESPACE", Width: 16, Default: true}}, cols...)
}

